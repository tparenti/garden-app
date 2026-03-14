package store

import (
	"context"
	"time"

	"github.com/alexramsey92/garden-app/internal/models"
)

// PlantingFilter filters planting entries in list queries.
type PlantingFilter struct {
	FromDate  *time.Time
	ToDate    *time.Time
	PlantName string
	Type      string
}

// Store defines the persistence interface for the garden app.
type Store interface {
	// Seeds
	AddSeed(ctx context.Context, s *models.Seed) (int64, error)
	ListSeeds(ctx context.Context) ([]models.Seed, error)
	GetSeed(ctx context.Context, id int64) (*models.Seed, error)
	RemoveSeed(ctx context.Context, id int64) error
	UpdateSeed(ctx context.Context, s *models.Seed) error

	// PlantSpecs
	ListPlantSpecs(ctx context.Context) ([]models.PlantSpec, error)
	GetPlantSpec(ctx context.Context, id int64) (*models.PlantSpec, error)
	SearchPlantSpecs(ctx context.Context, query string) ([]models.PlantSpec, error)
	AddPlantSpec(ctx context.Context, s *models.PlantSpec) (int64, error)

	// PlantingEntries
	AddPlantingEntry(ctx context.Context, e *models.PlantingEntry) (int64, error)
	ListPlantingEntries(ctx context.Context, filter PlantingFilter) ([]models.PlantingEntry, error)
	GetPlantingEntry(ctx context.Context, id int64) (*models.PlantingEntry, error)
	UpdatePlantingEntry(ctx context.Context, e *models.PlantingEntry) error
	RemovePlantingEntry(ctx context.Context, id int64) error

	// Config
	GetConfig(ctx context.Context, key string) (string, error)
	SetConfig(ctx context.Context, key, value string) error

	// Trays
	AddTray(ctx context.Context, t *models.Tray) (int64, error)
	ListTrays(ctx context.Context) ([]models.Tray, error)
	GetTray(ctx context.Context, id int64) (*models.Tray, error)
	RemoveTray(ctx context.Context, id int64) error
	GetTrayCell(ctx context.Context, id int64) (*models.TrayCell, error)
	SetTrayCell(ctx context.Context, c *models.TrayCell) error
	ClearTrayCell(ctx context.Context, id int64) error
	BulkSetTrayCells(ctx context.Context, cellIDs []int64, seedID *int64, label, status string, sownAt *time.Time) ([]models.TrayCell, error)

	// Beds
	AddBed(ctx context.Context, b *models.RaisedBed) (int64, error)
	ListBeds(ctx context.Context) ([]models.RaisedBed, error)
	GetBed(ctx context.Context, id int64) (*models.RaisedBed, error)
	RemoveBed(ctx context.Context, id int64) error
	GetBedCell(ctx context.Context, id int64) (*models.BedCell, error)
	SetBedCell(ctx context.Context, c *models.BedCell) error
	ClearBedCell(ctx context.Context, id int64) error
	BulkSetBedCells(ctx context.Context, cellIDs []int64, seedID *int64, label, status string) ([]models.BedCell, error)
	TransplantCell(ctx context.Context, trayCellID, bedID int64, row, col int) error

	// Timeline
	ListTimeline(ctx context.Context) ([]models.TimelineItem, error)

	Close() error
}
