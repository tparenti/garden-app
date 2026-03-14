package handlers

import (
	"net/http"
	"time"

	"github.com/alexr/garden-app/internal/store"
)

type dashboardData struct {
	Flash          string
	Error          string
	SeedCount      int
	SpecCount      int
	UpcomingCount  int
	Upcoming       []upcomingEntry
	Zip            string
	State          string
	LastFrost      string
	FirstFrost     string
}

type upcomingEntry struct {
	ID          int64
	PlantName   string
	PlantingType string
	PlannedDate string
	DaysAway    int
	Location    string
}

func (h *Handler) Dashboard(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	seeds, _ := h.store.ListSeeds(ctx)
	specs, _ := h.store.ListPlantSpecs(ctx)

	now := time.Now()
	end := now.AddDate(0, 1, 0)
	entries, _ := h.store.ListPlantingEntries(ctx, store.PlantingFilter{
		FromDate: &now,
		ToDate:   &end,
	})

	var upcoming []upcomingEntry
	for _, e := range entries {
		if e.ActualDate != nil {
			continue
		}
		days := int(e.PlannedDate.Sub(now).Hours() / 24)
		upcoming = append(upcoming, upcomingEntry{
			ID:           e.ID,
			PlantName:    e.PlantName,
			PlantingType: e.PlantingType,
			PlannedDate:  e.PlannedDate.Format("Jan 2"),
			DaysAway:     days,
			Location:     e.Location,
		})
	}

	zip, _ := h.store.GetConfig(ctx, "zip")
	state, _ := h.store.GetConfig(ctx, "state")
	lastFrost, firstFrost := "", ""

	var fd *FrostInfo
	if zip != "" {
		if result, err := h.frostSvc.LookupByZip(zip); err == nil {
			fd = &FrostInfo{City: result.City, State: result.State, Last: result.LastFrostMMDD, First: result.FirstFrostMMDD}
		}
	} else if state != "" {
		if result, err := h.frostSvc.LookupByState(state); err == nil {
			fd = &FrostInfo{City: result.City, State: result.State, Last: result.LastFrostMMDD, First: result.FirstFrostMMDD}
		}
	}
	if fd != nil {
		lastFrost = FormatMMDD(fd.Last)
		firstFrost = FormatMMDD(fd.First)
	}

	h.render(w, "dashboard", dashboardData{
		SeedCount:     len(seeds),
		SpecCount:     len(specs),
		UpcomingCount: len(upcoming),
		Upcoming:      upcoming,
		Zip:           zip,
		State:         state,
		LastFrost:     lastFrost,
		FirstFrost:    firstFrost,
	})
}
