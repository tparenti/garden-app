package planting

import (
	"fmt"
	"time"

	"github.com/alexr/garden-app/internal/models"
)

// PlantingWindow holds the calculated planting dates for a plant.
type PlantingWindow struct {
	PlantName      string
	LastFrost      time.Time
	FirstFrost     time.Time
	IndoorStart    *time.Time // nil if not applicable
	Transplant     *time.Time // nil if not applicable
	DirectSow      *time.Time // nil if not applicable
	GrowingDays    int        // days between transplant/direct sow and first frost
	WarningMessage string     // set if growing season may be too short
}

// Calculator computes planting windows from frost dates and plant specs.
type Calculator struct {
	FrostSvc *FrostDateService
}

// NewCalculator creates a Calculator backed by the given FrostDateService.
func NewCalculator(svc *FrostDateService) *Calculator {
	return &Calculator{FrostSvc: svc}
}

// CalculateWindow returns planting dates for a spec given a zip code and year.
// If zip is empty, it tries the state fallback via the state param.
func (c *Calculator) CalculateWindow(spec *models.PlantSpec, zip, state string, year int) (*PlantingWindow, error) {
	var fd *FrostDate
	var err error

	if zip != "" {
		fd, err = c.FrostSvc.LookupByZip(zip)
	} else if state != "" {
		fd, err = c.FrostSvc.LookupByState(state)
	} else {
		return nil, fmt.Errorf("zip or state is required")
	}
	if err != nil {
		return nil, err
	}

	lastFrost, err := ParseDate(fd.LastFrostMMDD, year)
	if err != nil {
		return nil, fmt.Errorf("parse last frost: %w", err)
	}
	firstFrost, err := ParseDate(fd.FirstFrostMMDD, year)
	if err != nil {
		return nil, fmt.Errorf("parse first frost: %w", err)
	}
	// Handle southern climates where first frost is in the next calendar year
	if firstFrost.Before(lastFrost) {
		firstFrost = firstFrost.AddDate(1, 0, 0)
	}

	win := &PlantingWindow{
		PlantName:  spec.Name,
		LastFrost:  lastFrost,
		FirstFrost: firstFrost,
	}

	if spec.StartIndoors && spec.WeeksBeforeFrost > 0 {
		t := lastFrost.AddDate(0, 0, -spec.WeeksBeforeFrost*7)
		win.IndoorStart = &t
		transplant := lastFrost.AddDate(0, 0, spec.WeeksAfterFrost*7)
		win.Transplant = &transplant
	}

	if spec.DirectSow {
		t := lastFrost.AddDate(0, 0, spec.WeeksAfterFrost*7)
		win.DirectSow = &t
	}

	// Calculate growing days from the earliest outdoor date to first frost
	var outdoorDate *time.Time
	if win.Transplant != nil {
		outdoorDate = win.Transplant
	} else if win.DirectSow != nil {
		outdoorDate = win.DirectSow
	}
	if outdoorDate != nil {
		win.GrowingDays = int(firstFrost.Sub(*outdoorDate).Hours() / 24)
		if win.GrowingDays < spec.DaysToMaturity {
			win.WarningMessage = fmt.Sprintf(
				"WARNING: Only %d growing days before first frost, but %s needs %d days to maturity.",
				win.GrowingDays, spec.Name, spec.DaysToMaturity,
			)
		}
	}

	return win, nil
}

// FormatWindow returns a human-readable summary of the planting window.
func FormatWindow(w *PlantingWindow) string {
	const layout = "January 2, 2006"
	out := fmt.Sprintf("Planting Window for: %s\n", w.PlantName)
	out += fmt.Sprintf("  Last Spring Frost:  %s\n", w.LastFrost.Format(layout))
	out += fmt.Sprintf("  First Fall Frost:   %s\n", w.FirstFrost.Format(layout))
	if w.IndoorStart != nil {
		out += fmt.Sprintf("  Start Indoors:      %s\n", w.IndoorStart.Format(layout))
	}
	if w.Transplant != nil {
		out += fmt.Sprintf("  Transplant Outside: %s\n", w.Transplant.Format(layout))
	}
	if w.DirectSow != nil {
		out += fmt.Sprintf("  Direct Sow:         %s\n", w.DirectSow.Format(layout))
	}
	out += fmt.Sprintf("  Growing Days:       %d\n", w.GrowingDays)
	if w.WarningMessage != "" {
		out += fmt.Sprintf("\n  %s\n", w.WarningMessage)
	}
	return out
}
