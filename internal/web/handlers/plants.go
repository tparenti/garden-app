package handlers

import (
	"net/http"
	"strconv"

	"github.com/alexr/garden-app/internal/models"
)

type plantsListData struct {
	Flash string
	Error string
	Specs []models.PlantSpec
}

type plantsDetailData struct {
	Flash string
	Error string
	Spec  *models.PlantSpec
}

func (h *Handler) PlantsList(w http.ResponseWriter, r *http.Request) {
	specs, err := h.store.ListPlantSpecs(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	h.render(w, "plants", plantsListData{Specs: specs})
}

func (h *Handler) PlantsSearch(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	var specs []models.PlantSpec
	var err error
	if q == "" {
		specs, err = h.store.ListPlantSpecs(r.Context())
	} else {
		specs, err = h.store.SearchPlantSpecs(r.Context(), q)
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	// HTMX partial: render only the table rows
	h.renderPartial(w, "plants", "plant-rows", plantsListData{Specs: specs})
}

func (h *Handler) PlantsDetail(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	spec, err := h.store.GetPlantSpec(r.Context(), id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	h.render(w, "plants_detail", plantsDetailData{Spec: spec})
}
