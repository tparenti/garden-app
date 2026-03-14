package store

import (
	"context"
	"time"

	"github.com/alexr/garden-app/internal/models"
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

	Close() error
}
