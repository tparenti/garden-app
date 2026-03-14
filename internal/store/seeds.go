package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/alexr/garden-app/internal/models"
)

func (s *SQLiteStore) AddSeed(ctx context.Context, seed *models.Seed) (int64, error) {
	res, err := s.db.ExecContext(ctx, `
		INSERT INTO seeds (name, variety, quantity, unit, purchased_at, notes, plant_spec_id)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		seed.Name, seed.Variety, seed.Quantity, seed.Unit,
		seed.PurchasedAt, seed.Notes, seed.PlantSpecID,
	)
	if err != nil {
		return 0, fmt.Errorf("insert seed: %w", err)
	}
	return res.LastInsertId()
}

func (s *SQLiteStore) ListSeeds(ctx context.Context) ([]models.Seed, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, name, variety, quantity, unit, purchased_at, notes, plant_spec_id
		FROM seeds ORDER BY name, variety`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanSeeds(rows)
}

func (s *SQLiteStore) GetSeed(ctx context.Context, id int64) (*models.Seed, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, name, variety, quantity, unit, purchased_at, notes, plant_spec_id
		FROM seeds WHERE id = ?`, id)
	seed := &models.Seed{}
	err := row.Scan(&seed.ID, &seed.Name, &seed.Variety, &seed.Quantity, &seed.Unit,
		&seed.PurchasedAt, &seed.Notes, &seed.PlantSpecID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("seed %d not found", id)
	}
	return seed, err
}

func (s *SQLiteStore) RemoveSeed(ctx context.Context, id int64) error {
	res, err := s.db.ExecContext(ctx, "DELETE FROM seeds WHERE id = ?", id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("seed %d not found", id)
	}
	return nil
}

func (s *SQLiteStore) UpdateSeed(ctx context.Context, seed *models.Seed) error {
	res, err := s.db.ExecContext(ctx, `
		UPDATE seeds SET name=?, variety=?, quantity=?, unit=?, purchased_at=?, notes=?, plant_spec_id=?
		WHERE id=?`,
		seed.Name, seed.Variety, seed.Quantity, seed.Unit,
		seed.PurchasedAt, seed.Notes, seed.PlantSpecID, seed.ID,
	)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("seed %d not found", seed.ID)
	}
	return nil
}

func scanSeeds(rows *sql.Rows) ([]models.Seed, error) {
	var seeds []models.Seed
	for rows.Next() {
		var s models.Seed
		if err := rows.Scan(&s.ID, &s.Name, &s.Variety, &s.Quantity, &s.Unit,
			&s.PurchasedAt, &s.Notes, &s.PlantSpecID); err != nil {
			return nil, err
		}
		seeds = append(seeds, s)
	}
	return seeds, rows.Err()
}
