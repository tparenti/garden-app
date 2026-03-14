package handlers

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/alexr/garden-app/internal/models"
)

type seedsListData struct {
	Flash string
	Error string
	Seeds []models.Seed
	Specs []models.PlantSpec
}

type seedsNewData struct {
	Flash string
	Specs []models.PlantSpec
	Error string
}

func (h *Handler) SeedsList(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	seeds, err := h.store.ListSeeds(ctx)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	specs, _ := h.store.ListPlantSpecs(ctx)

	h.render(w, "seeds", seedsListData{
		Flash: r.URL.Query().Get("flash"),
		Seeds: seeds,
		Specs: specs,
	})
}

func (h *Handler) SeedsNew(w http.ResponseWriter, r *http.Request) {
	specs, _ := h.store.ListPlantSpecs(r.Context())
	h.render(w, "seeds_new", seedsNewData{Specs: specs})
}

func (h *Handler) SeedsCreate(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	qty, _ := strconv.Atoi(r.FormValue("quantity"))
	if qty == 0 {
		qty = 1
	}

	s := &models.Seed{
		Name:     r.FormValue("name"),
		Variety:  r.FormValue("variety"),
		Quantity: qty,
		Unit:     r.FormValue("unit"),
		Notes:    r.FormValue("notes"),
	}
	if s.Name == "" {
		specs, _ := h.store.ListPlantSpecs(r.Context())
		h.render(w, "seeds_new", seedsNewData{Error: "Name is required.", Specs: specs})
		return
	}
	now := time.Now()
	s.PurchasedAt = &now

	if specIDStr := r.FormValue("plant_spec_id"); specIDStr != "" {
		if id, err := strconv.ParseInt(specIDStr, 10, 64); err == nil && id > 0 {
			s.PlantSpecID = &id
		}
	}

	if _, err := h.store.AddSeed(r.Context(), s); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, fmt.Sprintf("/seeds?flash=%s+added", s.Name), http.StatusSeeOther)
}

func (h *Handler) SeedsDelete(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	if err := h.store.RemoveSeed(r.Context(), id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	// HTMX: return empty response — the row will be removed from the DOM
	w.WriteHeader(http.StatusOK)
}
