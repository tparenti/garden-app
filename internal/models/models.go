package models

import "time"

// Seed represents a packet or supply of seeds owned by the user.
type Seed struct {
	ID          int64
	Name        string
	Variety     string
	Quantity    int
	Unit        string // "packets" | "grams" | "seeds"
	PurchasedAt *time.Time
	Notes       string
	PlantSpecID *int64
}

// PlantSpec holds horticultural data for a species/variety.
type PlantSpec struct {
	ID                int64
	Name              string
	Variety           string
	DaysToGermination int
	DaysToMaturity    int
	SpacingInches     float64
	DepthInches       float64
	SunRequirement    string  // "full" | "partial" | "shade"
	WaterRequirement  string  // "low" | "medium" | "high"
	WeeksBeforeFrost  int     // weeks before last frost to start indoors (positive = before)
	WeeksAfterFrost   int     // weeks after last frost to direct sow/transplant (positive = after)
	StartIndoors      bool
	DirectSow         bool
	HardinessZoneMin  string
	HardinessZoneMax  string
	Notes             string
}

// PlantingEntry is a scheduled or recorded planting event.
type PlantingEntry struct {
	ID              int64
	SeedID          *int64
	PlantSpecID     *int64
	PlantName       string
	PlantingType    string // "indoor_start" | "transplant" | "direct_sow"
	PlannedDate     time.Time
	ActualDate      *time.Time
	Location        string
	QuantityPlanted int
	Notes           string
	CreatedAt       time.Time
}

// Config stores user preferences.
type Config struct {
	Key   string
	Value string
}
