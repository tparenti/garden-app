package handlers

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/alexr/garden-app/internal/models"
	"github.com/alexr/garden-app/internal/planting"
	"github.com/alexr/garden-app/internal/store"
)

type scheduleListData struct {
	Flash   string
	Error   string
	Entries []models.PlantingEntry
}

type scheduleNewData struct {
	Flash     string
	Specs     []models.PlantSpec
	Seeds     []models.Seed
	Error     string
	PlantName string
}

type scheduleSuggestData struct {
	Flash   string
	Specs   []models.PlantSpec
	Window  *planting.PlantingWindow
	Error   string
	Plant   string
	Zip     string
	SpecID  int64
}

func (h *Handler) ScheduleList(w http.ResponseWriter, r *http.Request) {
	filter := store.PlantingFilter{
		PlantName: r.URL.Query().Get("plant"),
		Type:      r.URL.Query().Get("type"),
	}
	if from := r.URL.Query().Get("from"); from != "" {
		if t, err := time.ParseInLocation("2006-01-02", from, time.Local); err == nil {
			filter.FromDate = &t
		}
	}
	if to := r.URL.Query().Get("to"); to != "" {
		if t, err := time.ParseInLocation("2006-01-02", to, time.Local); err == nil {
			filter.ToDate = &t
		}
	}

	entries, err := h.store.ListPlantingEntries(r.Context(), filter)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	h.render(w, "schedule", scheduleListData{
		Flash:   r.URL.Query().Get("flash"),
		Entries: entries,
	})
}

func (h *Handler) ScheduleNew(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	specs, _ := h.store.ListPlantSpecs(ctx)
	seeds, _ := h.store.ListSeeds(ctx)
	h.render(w, "schedule_new", scheduleNewData{
		Specs:     specs,
		Seeds:     seeds,
		PlantName: r.URL.Query().Get("plant"),
	})
}

func (h *Handler) ScheduleCreate(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	plantName := r.FormValue("plant_name")
	plantingType := r.FormValue("planting_type")
	if plantName == "" || plantingType == "" {
		ctx := r.Context()
		specs, _ := h.store.ListPlantSpecs(ctx)
		seeds, _ := h.store.ListSeeds(ctx)
		h.render(w, "schedule_new", scheduleNewData{
			Error: "Plant name and type are required.",
			Specs: specs, Seeds: seeds,
		})
		return
	}

	planned := time.Now()
	if dateStr := r.FormValue("planned_date"); dateStr != "" {
		if t, err := time.ParseInLocation("2006-01-02", dateStr, time.Local); err == nil {
			planned = t
		}
	}

	qty, _ := strconv.Atoi(r.FormValue("quantity"))

	e := &models.PlantingEntry{
		PlantName:       plantName,
		PlantingType:    plantingType,
		PlannedDate:     planned,
		Location:        r.FormValue("location"),
		QuantityPlanted: qty,
		Notes:           r.FormValue("notes"),
	}
	if specIDStr := r.FormValue("plant_spec_id"); specIDStr != "" {
		if id, err := strconv.ParseInt(specIDStr, 10, 64); err == nil && id > 0 {
			e.PlantSpecID = &id
		}
	}
	if seedIDStr := r.FormValue("seed_id"); seedIDStr != "" {
		if id, err := strconv.ParseInt(seedIDStr, 10, 64); err == nil && id > 0 {
			e.SeedID = &id
		}
	}

	if _, err := h.store.AddPlantingEntry(r.Context(), e); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, fmt.Sprintf("/schedule?flash=%s+scheduled", plantName), http.StatusSeeOther)
}

func (h *Handler) ScheduleDone(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	e, err := h.store.GetPlantingEntry(r.Context(), id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	now := time.Now()
	e.ActualDate = &now
	if err := h.store.UpdatePlantingEntry(r.Context(), e); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	// HTMX partial: re-render just this row
	h.renderPartial(w, "schedule", "schedule-row", e)
}

func (h *Handler) ScheduleDelete(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	if err := h.store.RemovePlantingEntry(r.Context(), id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (h *Handler) ScheduleSuggestForm(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	specs, _ := h.store.ListPlantSpecs(ctx)
	zip, _ := h.store.GetConfig(ctx, "zip")

	var specID int64
	if s := r.URL.Query().Get("spec"); s != "" {
		if id, err := strconv.ParseInt(s, 10, 64); err == nil {
			specID = id
		}
	}

	h.render(w, "schedule_suggest", scheduleSuggestData{
		Specs:  specs,
		Zip:    zip,
		SpecID: specID,
	})
}

func (h *Handler) ScheduleSuggestResult(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	specs, _ := h.store.ListPlantSpecs(ctx)

	zip := r.FormValue("zip")
	if zip == "" {
		zip, _ = h.store.GetConfig(ctx, "zip")
	}
	state := ""
	if zip == "" {
		state, _ = h.store.GetConfig(ctx, "state")
	}

	year := time.Now().Year()
	if y, err := strconv.Atoi(r.FormValue("year")); err == nil && y > 2000 {
		year = y
	}

	data := scheduleSuggestData{Specs: specs, Zip: zip}

	var spec *models.PlantSpec
	if specIDStr := r.FormValue("spec_id"); specIDStr != "" {
		if id, err := strconv.ParseInt(specIDStr, 10, 64); err == nil && id > 0 {
			if s, err := h.store.GetPlantSpec(ctx, id); err == nil {
				spec = s
				data.SpecID = id
			}
		}
	} else if plantName := r.FormValue("plant"); plantName != "" {
		data.Plant = plantName
		if found, err := h.store.SearchPlantSpecs(ctx, plantName); err == nil && len(found) > 0 {
			spec = &found[0]
		} else {
			data.Error = fmt.Sprintf("No plant spec found for %q", plantName)
			h.render(w, "schedule_suggest", data)
			return
		}
	} else {
		data.Error = "Enter a plant name or select a spec."
		h.render(w, "schedule_suggest", data)
		return
	}

	if zip == "" && state == "" {
		data.Error = "No locale set. Go to the Locale page to set your zip code."
		h.render(w, "schedule_suggest", data)
		return
	}

	win, err := h.calc.CalculateWindow(spec, zip, state, year)
	if err != nil {
		data.Error = err.Error()
		h.render(w, "schedule_suggest", data)
		return
	}
	data.Window = win
	h.render(w, "schedule_suggest", data)
}
