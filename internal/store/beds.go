package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/alexramsey92/garden-app/internal/models"
)

func (s *SQLiteStore) AddBed(ctx context.Context, b *models.RaisedBed) (int64, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	res, err := tx.ExecContext(ctx,
		`INSERT INTO beds (name, rows, cols) VALUES (?, ?, ?)`,
		b.Name, b.Rows, b.Cols,
	)
	if err != nil {
		return 0, fmt.Errorf("insert bed: %w", err)
	}
	id, _ := res.LastInsertId()

	for r := 0; r < b.Rows; r++ {
		for c := 0; c < b.Cols; c++ {
			if _, err := tx.ExecContext(ctx,
				`INSERT INTO bed_cells (bed_id, row, col) VALUES (?, ?, ?)`,
				id, r, c,
			); err != nil {
				return 0, fmt.Errorf("insert bed cell (%d,%d): %w", r, c, err)
			}
		}
	}
	return id, tx.Commit()
}

func (s *SQLiteStore) ListBeds(ctx context.Context) ([]models.RaisedBed, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, name, rows, cols, created_at FROM beds ORDER BY created_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("query beds: %w", err)
	}
	defer rows.Close()

	var beds []models.RaisedBed
	for rows.Next() {
		var b models.RaisedBed
		if err := rows.Scan(&b.ID, &b.Name, &b.Rows, &b.Cols, &b.CreatedAt); err != nil {
			return nil, err
		}
		beds = append(beds, b)
	}
	return beds, rows.Err()
}

func (s *SQLiteStore) GetBed(ctx context.Context, id int64) (*models.RaisedBed, error) {
	var b models.RaisedBed
	err := s.db.QueryRowContext(ctx,
		`SELECT id, name, rows, cols, created_at FROM beds WHERE id = ?`, id,
	).Scan(&b.ID, &b.Name, &b.Rows, &b.Cols, &b.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get bed: %w", err)
	}

	cellRows, err := s.db.QueryContext(ctx,
		`SELECT id, bed_id, row, col, seed_id, label, status,
		        planted_at, harvested_at, failed_at, source_tray_cell_id, notes
		 FROM bed_cells WHERE bed_id = ? ORDER BY row, col`, id)
	if err != nil {
		return nil, fmt.Errorf("query bed cells: %w", err)
	}
	defer cellRows.Close()

	b.Cells = make([][]models.BedCell, b.Rows)
	for i := range b.Cells {
		b.Cells[i] = make([]models.BedCell, b.Cols)
	}
	for cellRows.Next() {
		var c models.BedCell
		if err := cellRows.Scan(
			&c.ID, &c.BedID, &c.Row, &c.Col,
			&c.SeedID, &c.Label, &c.Status,
			&c.PlantedAt, &c.HarvestedAt, &c.FailedAt,
			&c.SourceTrayCellID, &c.Notes,
		); err != nil {
			return nil, err
		}
		if c.Row >= 0 && c.Row < b.Rows && c.Col >= 0 && c.Col < b.Cols {
			b.Cells[c.Row][c.Col] = c
		}
	}
	return &b, cellRows.Err()
}

func (s *SQLiteStore) RemoveBed(ctx context.Context, id int64) error {
	res, err := s.db.ExecContext(ctx, `DELETE FROM beds WHERE id = ?`, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("bed %d not found", id)
	}
	return nil
}

func (s *SQLiteStore) GetBedCell(ctx context.Context, id int64) (*models.BedCell, error) {
	var c models.BedCell
	err := s.db.QueryRowContext(ctx,
		`SELECT id, bed_id, row, col, seed_id, label, status,
		        planted_at, harvested_at, failed_at, source_tray_cell_id, notes
		 FROM bed_cells WHERE id = ?`, id,
	).Scan(
		&c.ID, &c.BedID, &c.Row, &c.Col,
		&c.SeedID, &c.Label, &c.Status,
		&c.PlantedAt, &c.HarvestedAt, &c.FailedAt,
		&c.SourceTrayCellID, &c.Notes,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get bed cell %d: %w", id, err)
	}
	return &c, nil
}

func (s *SQLiteStore) SetBedCell(ctx context.Context, c *models.BedCell) error {
	res, err := s.db.ExecContext(ctx, `
		UPDATE bed_cells
		SET seed_id=?, label=?, status=?, planted_at=?, harvested_at=?, failed_at=?, notes=?
		WHERE id=?`,
		c.SeedID, c.Label, c.Status,
		c.PlantedAt, c.HarvestedAt, c.FailedAt,
		c.Notes, c.ID,
	)
	if err != nil {
		return fmt.Errorf("set bed cell: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("bed cell %d not found", c.ID)
	}
	return nil
}

func (s *SQLiteStore) ClearBedCell(ctx context.Context, id int64) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE bed_cells
		SET seed_id=NULL, label='', status='empty',
		    planted_at=NULL, harvested_at=NULL, failed_at=NULL,
		    source_tray_cell_id=NULL, notes=''
		WHERE id=?`, id)
	return err
}

// TransplantCell copies a germinated tray cell into a bed cell and marks it transplanted.
func (s *SQLiteStore) TransplantCell(ctx context.Context, trayCellID, bedID int64, row, col int) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	var label string
	var seedID *int64
	if err := tx.QueryRowContext(ctx,
		`SELECT label, seed_id FROM tray_cells WHERE id = ?`, trayCellID,
	).Scan(&label, &seedID); err != nil {
		return fmt.Errorf("get tray cell: %w", err)
	}

	now := time.Now()
	res, err := tx.ExecContext(ctx, `
		UPDATE bed_cells
		SET label=?, seed_id=?, status='planted', planted_at=?, source_tray_cell_id=?
		WHERE bed_id=? AND row=? AND col=?`,
		label, seedID, now, trayCellID, bedID, row, col,
	)
	if err != nil {
		return fmt.Errorf("update bed cell: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("bed cell (%d,%d) not found in bed %d", row, col, bedID)
	}

	if _, err := tx.ExecContext(ctx,
		`UPDATE tray_cells SET status='transplanted' WHERE id=?`, trayCellID,
	); err != nil {
		return fmt.Errorf("mark tray cell transplanted: %w", err)
	}
	return tx.Commit()
}

func (s *SQLiteStore) BulkSetBedCells(ctx context.Context, cellIDs []int64, seedID *int64, label, status string) ([]models.BedCell, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	now := time.Now()
	for _, id := range cellIDs {
		var plantedAt, harvestedAt, failedAt *time.Time
		switch status {
		case "planted", "growing":
			plantedAt = &now
		case "harvested":
			harvestedAt = &now
		case "failed":
			failedAt = &now
		}
		if _, err := tx.ExecContext(ctx, `
			UPDATE bed_cells
			SET seed_id=?, label=?, status=?, planted_at=?, harvested_at=?, failed_at=?, notes=''
			WHERE id=?`,
			seedID, label, status, plantedAt, harvestedAt, failedAt, id,
		); err != nil {
			return nil, fmt.Errorf("bulk update bed cell %d: %w", id, err)
		}
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit: %w", err)
	}

	cells := make([]models.BedCell, 0, len(cellIDs))
	for _, id := range cellIDs {
		c, err := s.GetBedCell(ctx, id)
		if err != nil || c == nil {
			continue
		}
		cells = append(cells, *c)
	}
	return cells, nil
}

func (s *SQLiteStore) ListTimeline(ctx context.Context) ([]models.TimelineItem, error) {
	const q = `
	SELECT
		COALESCE(
			NULLIF(tc.label, ''),
			CASE WHEN s.id IS NOT NULL
			     THEN s.name || CASE WHEN s.variety != '' THEN ' ' || s.variety ELSE '' END
			     ELSE NULL END,
			'?'),
		tc.seed_id,
		t.name,
		tc.sown_at, tc.germinated_at, tc.failed_at,
		COALESCE(b.name, ''),
		bc.planted_at, bc.harvested_at, bc.failed_at
	FROM tray_cells tc
	JOIN trays t ON t.id = tc.tray_id
	LEFT JOIN seeds s ON s.id = tc.seed_id
	LEFT JOIN bed_cells bc ON bc.source_tray_cell_id = tc.id
	LEFT JOIN beds b ON b.id = bc.bed_id
	WHERE tc.label != '' OR tc.seed_id IS NOT NULL

	UNION ALL

	SELECT
		COALESCE(
			NULLIF(bc.label, ''),
			CASE WHEN s.id IS NOT NULL
			     THEN s.name || CASE WHEN s.variety != '' THEN ' ' || s.variety ELSE '' END
			     ELSE NULL END,
			'?'),
		bc.seed_id,
		'',
		NULL, NULL, NULL,
		b.name,
		bc.planted_at, bc.harvested_at, bc.failed_at
	FROM bed_cells bc
	JOIN beds b ON b.id = bc.bed_id
	LEFT JOIN seeds s ON s.id = bc.seed_id
	WHERE bc.source_tray_cell_id IS NULL AND (bc.label != '' OR bc.seed_id IS NOT NULL)
	`

	rows, err := s.db.QueryContext(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("query timeline: %w", err)
	}
	defer rows.Close()

	var items []models.TimelineItem
	for rows.Next() {
		var it models.TimelineItem
		if err := rows.Scan(
			&it.Label, &it.SeedID,
			&it.TrayName,
			&it.SownAt, &it.GerminatedAt, &it.TrayFailedAt,
			&it.BedName,
			&it.PlantedAt, &it.HarvestedAt, &it.BedFailedAt,
		); err != nil {
			return nil, err
		}
		items = append(items, it)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Sort chronologically by first date
	sort.Slice(items, func(i, j int) bool {
		ti := firstTimelineDate(items[i])
		tj := firstTimelineDate(items[j])
		if ti == nil {
			return false
		}
		if tj == nil {
			return true
		}
		return ti.Before(*tj)
	})
	return items, nil
}

func firstTimelineDate(it models.TimelineItem) *time.Time {
	if it.SownAt != nil {
		return it.SownAt
	}
	return it.PlantedAt
}
