package handlers

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/alexramsey92/garden-app/internal/models"
)

type bedsListData struct {
	Flash string
	Error string
	Beds  []models.RaisedBed
}

type bedsNewData struct {
	Flash string
	Error string
	Sizes []string
}

var bedPresetSizes = []string{"3x4", "3x10", "4x4", "4x8", "4x12", "6x8", "6x12"}

type bedsViewData struct {
	Flash string
	Error string
	Bed   *models.RaisedBed
	Seeds []models.Seed
}

type bedBulkResultData struct {
	BedID int64
	Cells []models.BedCell
}

type bedCellFormData struct {
	BedID   int64
	BedName string
	Cell    models.BedCell
	Seeds   []models.Seed
}

type bedCellPartialData struct {
	BedID int64
	Cell  models.BedCell
}

type timelinePageData struct {
	Flash  string
	Error  string
	Year   int
	Months []monthMarker
	Rows   []timelineRow
}

func (h *Handler) BedsList(w http.ResponseWriter, r *http.Request) {
	beds, err := h.store.ListBeds(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	h.render(w, "beds", bedsListData{
		Flash: r.URL.Query().Get("flash"),
		Beds:  beds,
	})
}

func (h *Handler) BedsNew(w http.ResponseWriter, r *http.Request) {
	h.render(w, "beds_new", bedsNewData{Sizes: bedPresetSizes})
}

func (h *Handler) BedsCreate(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	name := r.FormValue("name")
	if name == "" {
		h.render(w, "beds_new", bedsNewData{Error: "Name is required.", Sizes: bedPresetSizes})
		return
	}

	rows, cols := parseGridSize(r.FormValue("size"), r.FormValue("rows"), r.FormValue("cols"))
	if rows <= 0 || cols <= 0 {
		h.render(w, "beds_new", bedsNewData{Error: "Invalid grid size.", Sizes: bedPresetSizes})
		return
	}

	b := &models.RaisedBed{Name: name, Rows: rows, Cols: cols}
	id, err := h.store.AddBed(r.Context(), b)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, fmt.Sprintf("/beds/%d", id), http.StatusSeeOther)
}

func (h *Handler) BedsDelete(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	if err := h.store.RemoveBed(r.Context(), id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (h *Handler) BedsView(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	bed, err := h.store.GetBed(r.Context(), id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if bed == nil {
		http.NotFound(w, r)
		return
	}
	seeds, _ := h.store.ListSeeds(r.Context())
	h.render(w, "beds_view", bedsViewData{
		Flash: r.URL.Query().Get("flash"),
		Bed:   bed,
		Seeds: seeds,
	})
}

func (h *Handler) BedsCellEditForm(w http.ResponseWriter, r *http.Request) {
	bedID, cellID, ok := parseBedCell(w, r)
	if !ok {
		return
	}
	ctx := r.Context()
	cell, err := h.store.GetBedCell(ctx, cellID)
	if err != nil || cell == nil {
		http.Error(w, "cell not found", http.StatusNotFound)
		return
	}
	bed, err := h.store.GetBed(ctx, bedID)
	if err != nil || bed == nil {
		http.Error(w, "bed not found", http.StatusNotFound)
		return
	}
	seeds, _ := h.store.ListSeeds(ctx)
	h.renderPartial(w, "beds_view", "bed-cell-form", bedCellFormData{
		BedID:   bedID,
		BedName: bed.Name,
		Cell:    *cell,
		Seeds:   seeds,
	})
}

func (h *Handler) BedsCellSave(w http.ResponseWriter, r *http.Request) {
	bedID, cellID, ok := parseBedCell(w, r)
	if !ok {
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	ctx := r.Context()
	cell, err := h.store.GetBedCell(ctx, cellID)
	if err != nil || cell == nil {
		http.Error(w, "cell not found", http.StatusNotFound)
		return
	}

	cell.Label = r.FormValue("label")
	cell.Status = r.FormValue("status")
	cell.Notes = r.FormValue("notes")

	// Link seed if selected
	cell.SeedID = nil
	if seedStr := r.FormValue("seed_id"); seedStr != "" {
		if sid, err := strconv.ParseInt(seedStr, 10, 64); err == nil && sid > 0 {
			cell.SeedID = &sid
			if cell.Label == "" {
				if s, err := h.store.GetSeed(ctx, sid); err == nil && s != nil {
					cell.Label = s.Name
					if s.Variety != "" {
						cell.Label += " " + s.Variety
					}
				}
			}
		}
	}

	now := time.Now()
	switch cell.Status {
	case "planted":
		if cell.PlantedAt == nil {
			cell.PlantedAt = &now
		}
	case "growing":
		if cell.PlantedAt == nil {
			cell.PlantedAt = &now
		}
	case "harvested":
		if cell.PlantedAt == nil {
			cell.PlantedAt = &now
		}
		if cell.HarvestedAt == nil {
			cell.HarvestedAt = &now
		}
	case "failed":
		if cell.FailedAt == nil {
			cell.FailedAt = &now
		}
	case "empty":
		cell.PlantedAt = nil
		cell.HarvestedAt = nil
		cell.FailedAt = nil
		cell.SeedID = nil
		cell.Label = ""
		cell.SourceTrayCellID = nil
	}

	if err := h.store.SetBedCell(ctx, cell); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	h.renderPartial(w, "beds_view", "bed-cell-oob", bedCellPartialData{
		BedID: bedID,
		Cell:  *cell,
	})
}

func (h *Handler) BedsCellsBulkSet(w http.ResponseWriter, r *http.Request) {
	bedID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Error(w, "invalid bed id", http.StatusBadRequest)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	ctx := r.Context()

	cellIDStrs := r.Form["cell_ids"]
	if len(cellIDStrs) == 0 {
		w.WriteHeader(http.StatusOK)
		return
	}
	cellIDs := make([]int64, 0, len(cellIDStrs))
	for _, s := range cellIDStrs {
		if id, err := strconv.ParseInt(s, 10, 64); err == nil {
			cellIDs = append(cellIDs, id)
		}
	}

	var seedID *int64
	label := r.FormValue("label")
	if seedStr := r.FormValue("seed_id"); seedStr != "" {
		if sid, err := strconv.ParseInt(seedStr, 10, 64); err == nil && sid > 0 {
			seedID = &sid
			if label == "" {
				if s, err := h.store.GetSeed(ctx, sid); err == nil && s != nil {
					label = s.Name
					if s.Variety != "" {
						label += " " + s.Variety
					}
				}
			}
		}
	}
	status := r.FormValue("status")
	if status == "" {
		status = "planted"
	}

	cells, err := h.store.BulkSetBedCells(ctx, cellIDs, seedID, label, status)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	h.renderPartial(w, "beds_view", "bed-bulk-result", bedBulkResultData{
		BedID: bedID,
		Cells: cells,
	})
}

func (h *Handler) BedsCellClear(w http.ResponseWriter, r *http.Request) {
	bedID, cellID, ok := parseBedCell(w, r)
	if !ok {
		return
	}
	if err := h.store.ClearBedCell(r.Context(), cellID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	h.renderPartial(w, "beds_view", "bed-cell-oob", bedCellPartialData{
		BedID: bedID,
		Cell:  models.BedCell{ID: cellID, Status: "empty"},
	})
}

func (h *Handler) Timeline(w http.ResponseWriter, r *http.Request) {
	items, err := h.store.ListTimeline(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	year := time.Now().Year()
	h.render(w, "timeline", timelinePageData{
		Year:   year,
		Months: yearMonthMarkers(year),
		Rows:   buildTimelineRows(items, year),
	})
}

// parseBedCell extracts bed id and cell id from the request path.
func parseBedCell(w http.ResponseWriter, r *http.Request) (bedID, cellID int64, ok bool) {
	bedID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Error(w, "invalid bed id", http.StatusBadRequest)
		return 0, 0, false
	}
	cellID, err = strconv.ParseInt(r.PathValue("cid"), 10, 64)
	if err != nil {
		http.Error(w, "invalid cell id", http.StatusBadRequest)
		return 0, 0, false
	}
	return bedID, cellID, true
}
