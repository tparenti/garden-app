package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	_ "embed"

	_ "modernc.org/sqlite"

	"github.com/alexr/garden-app/internal/models"
)

//go:embed data/plant_specs.json
var plantSpecsJSON []byte

// SQLiteStore is the SQLite-backed implementation of Store.
type SQLiteStore struct {
	db *sql.DB
}

// NewSQLiteStore opens (or creates) the SQLite database at the given path.
func NewSQLiteStore(dbPath string) (*SQLiteStore, error) {
	if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
		return nil, fmt.Errorf("create db directory: %w", err)
	}
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}
	db.SetMaxOpenConns(1)

	s := &SQLiteStore{db: db}
	if err := s.migrate(); err != nil {
		return nil, fmt.Errorf("migrate: %w", err)
	}
	if err := s.seedPlantSpecs(context.Background()); err != nil {
		return nil, fmt.Errorf("seed plant specs: %w", err)
	}
	return s, nil
}

func (s *SQLiteStore) Close() error {
	return s.db.Close()
}

func (s *SQLiteStore) migrate() error {
	schema := `
	PRAGMA journal_mode=WAL;
	PRAGMA foreign_keys=ON;

	CREATE TABLE IF NOT EXISTS plant_specs (
		id                   INTEGER PRIMARY KEY AUTOINCREMENT,
		name                 TEXT NOT NULL,
		variety              TEXT DEFAULT '',
		days_to_germination  INTEGER DEFAULT 0,
		days_to_maturity     INTEGER DEFAULT 0,
		spacing_inches       REAL DEFAULT 0,
		depth_inches         REAL DEFAULT 0,
		sun_requirement      TEXT DEFAULT 'full',
		water_requirement    TEXT DEFAULT 'medium',
		weeks_before_frost   INTEGER DEFAULT 0,
		weeks_after_frost    INTEGER DEFAULT 0,
		start_indoors        INTEGER DEFAULT 0,
		direct_sow           INTEGER DEFAULT 0,
		hardiness_zone_min   TEXT DEFAULT '',
		hardiness_zone_max   TEXT DEFAULT '',
		notes                TEXT DEFAULT ''
	);

	CREATE TABLE IF NOT EXISTS seeds (
		id              INTEGER PRIMARY KEY AUTOINCREMENT,
		name            TEXT NOT NULL,
		variety         TEXT DEFAULT '',
		quantity        INTEGER DEFAULT 0,
		unit            TEXT DEFAULT 'packets',
		purchased_at    DATETIME,
		notes           TEXT DEFAULT '',
		plant_spec_id   INTEGER REFERENCES plant_specs(id) ON DELETE SET NULL
	);

	CREATE TABLE IF NOT EXISTS planting_entries (
		id               INTEGER PRIMARY KEY AUTOINCREMENT,
		seed_id          INTEGER REFERENCES seeds(id) ON DELETE SET NULL,
		plant_spec_id    INTEGER REFERENCES plant_specs(id) ON DELETE SET NULL,
		plant_name       TEXT NOT NULL,
		planting_type    TEXT NOT NULL CHECK(planting_type IN ('indoor_start','transplant','direct_sow')),
		planned_date     DATETIME NOT NULL,
		actual_date      DATETIME,
		location         TEXT DEFAULT '',
		quantity_planted INTEGER DEFAULT 0,
		notes            TEXT DEFAULT '',
		created_at       DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS config (
		key     TEXT PRIMARY KEY,
		value   TEXT NOT NULL
	);
	`
	_, err := s.db.Exec(schema)
	return err
}

// seedPlantSpecs inserts the built-in plant specs on first run.
func (s *SQLiteStore) seedPlantSpecs(ctx context.Context) error {
	var count int
	if err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM plant_specs").Scan(&count); err != nil {
		return err
	}
	if count > 0 {
		return nil
	}

	var specs []models.PlantSpec
	if err := json.Unmarshal(plantSpecsJSON, &specs); err != nil {
		return fmt.Errorf("parse plant specs: %w", err)
	}

	for i := range specs {
		if _, err := s.AddPlantSpec(ctx, &specs[i]); err != nil {
			return err
		}
	}
	return nil
}
