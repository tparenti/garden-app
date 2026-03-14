package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/alexr/garden-app/internal/models"
)

func (s *SQLiteStore) AddPlantingEntry(ctx context.Context, e *models.PlantingEntry) (int64, error) {
	res, err := s.db.ExecContext(ctx, `
		INSERT INTO planting_entries
			(seed_id, plant_spec_id, plant_name, planting_type, planned_date,
			 actual_date, location, quantity_planted, notes)
		VALUES (?,?,?,?,?,?,?,?,?)`,
		e.SeedID, e.PlantSpecID, e.PlantName, e.PlantingType, e.PlannedDate,
		e.ActualDate, e.Location, e.QuantityPlanted, e.Notes,
	)
	if err != nil {
		return 0, fmt.Errorf("insert planting entry: %w", err)
	}
	return res.LastInsertId()
}

func (s *SQLiteStore) ListPlantingEntries(ctx context.Context, filter PlantingFilter) ([]models.PlantingEntry, error) {
	query := `
		SELECT id, seed_id, plant_spec_id, plant_name, planting_type, planned_date,
		       actual_date, location, quantity_planted, notes, created_at
		FROM planting_entries WHERE 1=1`

	var args []any
	if filter.FromDate != nil {
		query += " AND planned_date >= ?"
		args = append(args, filter.FromDate)
	}
	if filter.ToDate != nil {
		query += " AND planned_date <= ?"
		args = append(args, filter.ToDate)
	}
	if filter.PlantName != "" {
		query += " AND plant_name LIKE ?"
		args = append(args, "%"+filter.PlantName+"%")
	}
	if filter.Type != "" {
		query += " AND planting_type = ?"
		args = append(args, strings.ToLower(filter.Type))
	}
	query += " ORDER BY planned_date"

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanPlantingEntries(rows)
}

func (s *SQLiteStore) GetPlantingEntry(ctx context.Context, id int64) (*models.PlantingEntry, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, seed_id, plant_spec_id, plant_name, planting_type, planned_date,
		       actual_date, location, quantity_planted, notes, created_at
		FROM planting_entries WHERE id = ?`, id)

	e := &models.PlantingEntry{}
	err := row.Scan(&e.ID, &e.SeedID, &e.PlantSpecID, &e.PlantName, &e.PlantingType,
		&e.PlannedDate, &e.ActualDate, &e.Location, &e.QuantityPlanted, &e.Notes, &e.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("planting entry %d not found", id)
	}
	return e, err
}

func (s *SQLiteStore) UpdatePlantingEntry(ctx context.Context, e *models.PlantingEntry) error {
	res, err := s.db.ExecContext(ctx, `
		UPDATE planting_entries
		SET seed_id=?, plant_spec_id=?, plant_name=?, planting_type=?, planned_date=?,
		    actual_date=?, location=?, quantity_planted=?, notes=?
		WHERE id=?`,
		e.SeedID, e.PlantSpecID, e.PlantName, e.PlantingType, e.PlannedDate,
		e.ActualDate, e.Location, e.QuantityPlanted, e.Notes, e.ID,
	)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("planting entry %d not found", e.ID)
	}
	return nil
}

func (s *SQLiteStore) RemovePlantingEntry(ctx context.Context, id int64) error {
	res, err := s.db.ExecContext(ctx, "DELETE FROM planting_entries WHERE id = ?", id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("planting entry %d not found", id)
	}
	return nil
}

func scanPlantingEntries(rows *sql.Rows) ([]models.PlantingEntry, error) {
	var entries []models.PlantingEntry
	for rows.Next() {
		var e models.PlantingEntry
		if err := rows.Scan(&e.ID, &e.SeedID, &e.PlantSpecID, &e.PlantName, &e.PlantingType,
			&e.PlannedDate, &e.ActualDate, &e.Location, &e.QuantityPlanted, &e.Notes, &e.CreatedAt); err != nil {
			return nil, err
		}
		entries = append(entries, e)
	}
	return entries, rows.Err()
}
