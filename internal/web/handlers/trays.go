package handlers

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/alexramsey92/garden-app/internal/models"
)

type traysListData struct {
	Flash string
	Error string
	Trays []models.Tray
}

type traysNewData struct {
	Flash string
	Error string
	Sizes []string
}

var trayPresetSizes = []string{"2x4", "2x8", "4x6", "6x6", "6x9", "6x12", "6x18", "12x6"}

type traysViewData struct {
	Flash string
	Error string
	Tray  *models.Tray
	Seeds []models.Seed
}

type trayBulkResultData struct {
	TrayID int64
	Cells  []models.TrayCell
}

type trayCellFormData struct {
	TrayID   int64
	TrayName string
	Cell     models.TrayCell
	Seeds    []models.Seed
	Beds     []models.RaisedBed
}

type trayCellPartialData struct {
	TrayID int64
	Cell   models.TrayCell
}

func (h *Handler) TraysList(w http.ResponseWriter, r *http.Request) {
	trays, err := h.store.ListTrays(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	h.render(w, "trays", traysListData{
		Flash: r.URL.Query().Get("flash"),
		Trays: trays,
	})
}

func (h *Handler) TraysNew(w http.ResponseWriter, r *http.Request) {
	h.render(w, "trays_new", traysNewData{Sizes: trayPresetSizes})
}

func (h *Handler) TraysCreate(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	name := r.FormValue("name")
	if name == "" {
		h.render(w, "trays_new", traysNewData{Error: "Name is required.", Sizes: trayPresetSizes})
		return
	}

	rows, cols := parseGridSize(r.FormValue("size"), r.FormValue("rows"), r.FormValue("cols"))
	if rows <= 0 || cols <= 0 {
		h.render(w, "trays_new", traysNewData{Error: "Invalid grid size.", Sizes: trayPresetSizes})
		return
	}

	t := &models.Tray{Name: name, Rows: rows, Cols: cols}
	id, err := h.store.AddTray(r.Context(), t)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, fmt.Sprintf("/trays/%d", id), http.StatusSeeOther)
}

func (h *Handler) TraysDelete(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	if err := h.store.RemoveTray(r.Context(), id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (h *Handler) TraysView(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	tray, err := h.store.GetTray(r.Context(), id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if tray == nil {
		http.NotFound(w, r)
		return
	}
	seeds, _ := h.store.ListSeeds(r.Context())
	h.render(w, "trays_view", traysViewData{
		Flash: r.URL.Query().Get("flash"),
		Tray:  tray,
		Seeds: seeds,
	})
}

func (h *Handler) TraysCellEditForm(w http.ResponseWriter, r *http.Request) {
	trayID, cellID, ok := parseTrayCell(w, r)
	if !ok {
		return
	}
	ctx := r.Context()
	cell, err := h.store.GetTrayCell(ctx, cellID)
	if err != nil || cell == nil {
		http.Error(w, "cell not found", http.StatusNotFound)
		return
	}
	tray, err := h.store.GetTray(ctx, trayID)
	if err != nil || tray == nil {
		http.Error(w, "tray not found", http.StatusNotFound)
		return
	}
	seeds, _ := h.store.ListSeeds(ctx)
	h.renderPartial(w, "trays_view", "tray-cell-form", trayCellFormData{
		TrayID:   trayID,
		TrayName: tray.Name,
		Cell:     *cell,
		Seeds:    seeds,
	})
}

func (h *Handler) TraysCellSave(w http.ResponseWriter, r *http.Request) {
	_, cellID, ok := parseTrayCell(w, r)
	if !ok {
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	ctx := r.Context()
	cell, err := h.store.GetTrayCell(ctx, cellID)
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
			// Use seed name as label if label is empty
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

	// Auto-stamp timestamps on status transitions
	now := time.Now()
	switch cell.Status {
	case "sown":
		if cell.SownAt == nil {
			if dateStr := r.FormValue("sown_at"); dateStr != "" {
				if t, err := time.Parse("2006-01-02", dateStr); err == nil {
					cell.SownAt = &t
				}
			}
			if cell.SownAt == nil {
				cell.SownAt = &now
			}
		}
	case "germinated":
		if cell.SownAt == nil {
			cell.SownAt = &now
		}
		if cell.GerminatedAt == nil {
			cell.GerminatedAt = &now
		}
	case "failed":
		if cell.FailedAt == nil {
			cell.FailedAt = &now
		}
	case "empty":
		cell.SownAt = nil
		cell.GerminatedAt = nil
		cell.FailedAt = nil
		cell.SeedID = nil
		cell.Label = ""
	}

	if err := h.store.SetTrayCell(ctx, cell); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	trayID, _ := strconv.ParseInt(r.PathValue("id"), 10, 64)
	h.renderPartial(w, "trays_view", "tray-cell-oob", trayCellPartialData{
		TrayID: trayID,
		Cell:   *cell,
	})
}

func (h *Handler) TraysCellClear(w http.ResponseWriter, r *http.Request) {
	trayID, cellID, ok := parseTrayCell(w, r)
	if !ok {
		return
	}
	if err := h.store.ClearTrayCell(r.Context(), cellID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	h.renderPartial(w, "trays_view", "tray-cell-oob", trayCellPartialData{
		TrayID: trayID,
		Cell:   models.TrayCell{ID: cellID, Status: "empty"},
	})
}

func (h *Handler) TraysCellsBulkSow(w http.ResponseWriter, r *http.Request) {
	trayID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Error(w, "invalid tray id", http.StatusBadRequest)
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
		status = "sown"
	}

	var sownAt *time.Time
	if dateStr := r.FormValue("sown_at"); dateStr != "" {
		if t, err := time.Parse("2006-01-02", dateStr); err == nil {
			sownAt = &t
		}
	}

	cells, err := h.store.BulkSetTrayCells(ctx, cellIDs, seedID, label, status, sownAt)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	h.renderPartial(w, "trays_view", "tray-bulk-result", trayBulkResultData{
		TrayID: trayID,
		Cells:  cells,
	})
}

func (h *Handler) TraysCellTransplantForm(w http.ResponseWriter, r *http.Request) {
	trayID, cellID, ok := parseTrayCell(w, r)
	if !ok {
		return
	}
	ctx := r.Context()
	cell, err := h.store.GetTrayCell(ctx, cellID)
	if err != nil || cell == nil {
		http.Error(w, "cell not found", http.StatusNotFound)
		return
	}
	beds, _ := h.store.ListBeds(ctx)
	h.renderPartial(w, "trays_view", "tray-cell-transplant-form", trayCellFormData{
		TrayID: trayID,
		Cell:   *cell,
		Beds:   beds,
	})
}

func (h *Handler) TraysCellTransplant(w http.ResponseWriter, r *http.Request) {
	trayID, cellID, ok := parseTrayCell(w, r)
	if !ok {
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	bedID, err := strconv.ParseInt(r.FormValue("bed_id"), 10, 64)
	if err != nil || bedID == 0 {
		http.Error(w, "invalid bed", http.StatusBadRequest)
		return
	}
	row, _ := strconv.Atoi(r.FormValue("row"))
	col, _ := strconv.Atoi(r.FormValue("col"))

	ctx := r.Context()
	if err := h.store.TransplantCell(ctx, cellID, bedID, row, col); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Return updated tray cell OOB + success message in panel
	cell, _ := h.store.GetTrayCell(ctx, cellID)
	bed, _ := h.store.GetBed(ctx, bedID)
	bedName := ""
	if bed != nil {
		bedName = bed.Name
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprintf(w, `<div class="bg-white border-t border-gray-200 shadow-2xl p-4 max-w-md mx-auto rounded-t-xl">
  <p class="text-green-700 font-medium text-sm">Transplanted to <strong>%s</strong>!</p>
  <a href="/beds/%d" class="text-green-600 hover:underline text-sm">View bed →</a>
  <button onclick="document.getElementById('edit-panel').innerHTML=''"
          class="ml-4 text-gray-400 hover:text-gray-600 text-sm">Close</button>
</div>`, bedName, bedID)

	if cell != nil {
		h.renderPartial(w, "trays_view", "tray-cell-oob", trayCellPartialData{
			TrayID: trayID,
			Cell:   *cell,
		})
	}
}

// parseTrayCell extracts tray id and cell id from the request path.
func parseTrayCell(w http.ResponseWriter, r *http.Request) (trayID, cellID int64, ok bool) {
	trayID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Error(w, "invalid tray id", http.StatusBadRequest)
		return 0, 0, false
	}
	cellID, err = strconv.ParseInt(r.PathValue("cid"), 10, 64)
	if err != nil {
		http.Error(w, "invalid cell id", http.StatusBadRequest)
		return 0, 0, false
	}
	return trayID, cellID, true
}

// parseGridSize returns rows, cols from a preset size string or custom inputs.
func parseGridSize(preset, rowStr, colStr string) (rows, cols int) {
	// Try parsing "RxC" preset format directly (e.g. "4x8", "3x10")
	if parts := strings.SplitN(preset, "x", 2); len(parts) == 2 {
		if r, err := strconv.Atoi(parts[0]); err == nil {
			if c, err := strconv.Atoi(parts[1]); err == nil {
				return r, c
			}
		}
	}
	rows, _ = strconv.Atoi(rowStr)
	cols, _ = strconv.Atoi(colStr)
	return rows, cols
}
