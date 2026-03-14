package handlers

import (
	"embed"
	"fmt"
	"html/template"
	"net/http"
	"time"

	"github.com/alexr/garden-app/internal/planting"
	"github.com/alexr/garden-app/internal/store"
)

//go:embed templates
var templateFS embed.FS

// Handler holds shared dependencies for all HTTP handlers.
type Handler struct {
	store    store.Store
	frostSvc *planting.FrostDateService
	calc     *planting.Calculator
	tmpls    map[string]*template.Template
}

// New creates a Handler and parses all templates.
func New(st store.Store, frostSvc *planting.FrostDateService, calc *planting.Calculator) *Handler {
	h := &Handler{
		store:    st,
		frostSvc: frostSvc,
		calc:     calc,
	}
	h.tmpls = h.parseTemplates()
	return h
}

// RegisterRoutes wires all HTTP routes onto mux.
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /", h.Dashboard)

	mux.HandleFunc("GET /seeds", h.SeedsList)
	mux.HandleFunc("GET /seeds/new", h.SeedsNew)
	mux.HandleFunc("POST /seeds", h.SeedsCreate)
	mux.HandleFunc("DELETE /seeds/{id}", h.SeedsDelete)

	mux.HandleFunc("GET /plants", h.PlantsList)
	mux.HandleFunc("GET /plants/search", h.PlantsSearch)
	mux.HandleFunc("GET /plants/{id}", h.PlantsDetail)

	mux.HandleFunc("GET /schedule", h.ScheduleList)
	mux.HandleFunc("GET /schedule/new", h.ScheduleNew)
	mux.HandleFunc("POST /schedule", h.ScheduleCreate)
	mux.HandleFunc("POST /schedule/{id}/done", h.ScheduleDone)
	mux.HandleFunc("DELETE /schedule/{id}", h.ScheduleDelete)
	mux.HandleFunc("GET /schedule/suggest", h.ScheduleSuggestForm)
	mux.HandleFunc("POST /schedule/suggest", h.ScheduleSuggestResult)

	mux.HandleFunc("GET /locale", h.LocaleShow)
	mux.HandleFunc("POST /locale", h.LocaleSet)
}

// parseTemplates builds a map of page name → *template.Template,
// each containing the layout + that page's content template.
func (h *Handler) parseTemplates() map[string]*template.Template {
	pages := []struct {
		name string
		file string
	}{
		{"dashboard", "templates/dashboard.html"},
		{"seeds", "templates/seeds.html"},
		{"seeds_new", "templates/seeds_new.html"},
		{"plants", "templates/plants.html"},
		{"plants_detail", "templates/plants_detail.html"},
		{"schedule", "templates/schedule.html"},
		{"schedule_new", "templates/schedule_new.html"},
		{"schedule_suggest", "templates/schedule_suggest.html"},
		{"locale", "templates/locale.html"},
	}

	tmpls := make(map[string]*template.Template, len(pages))
	for _, p := range pages {
		t := template.Must(template.New("").Funcs(templateFuncs()).ParseFS(
			templateFS,
			"templates/layout.html",
			p.file,
		))
		tmpls[p.name] = t
	}
	return tmpls
}

// render executes the full page layout for the given page name.
func (h *Handler) render(w http.ResponseWriter, name string, data any) {
	t, ok := h.tmpls[name]
	if !ok {
		http.Error(w, fmt.Sprintf("template %q not found", name), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := t.ExecuteTemplate(w, "layout", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// renderPartial executes a named partial template (for HTMX responses).
func (h *Handler) renderPartial(w http.ResponseWriter, page, partial string, data any) {
	t, ok := h.tmpls[page]
	if !ok {
		http.Error(w, fmt.Sprintf("template %q not found", page), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := t.ExecuteTemplate(w, partial, data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// isHTMX returns true if the request came from HTMX.
func isHTMX(r *http.Request) bool {
	return r.Header.Get("HX-Request") == "true"
}

// templateFuncs returns custom template functions.
func templateFuncs() template.FuncMap {
	return template.FuncMap{
		"formatMMDD": FormatMMDD,
		"deref":      derefInt64,
		"typeClass":  plantingTypeClass,
		"formatDate":    func(t time.Time) string { return t.Format("2006-01-02") },
		"formatDatePtr": func(t *time.Time) string { if t == nil { return "" }; return t.Format("2006-01-02") },
		"neg":        func(n int) int { return -n },
		"lt0":        func(n int) bool { return n < 0 },
	}
}

func derefInt64(p *int64) int64 {
	if p == nil {
		return 0
	}
	return *p
}

func plantingTypeClass(t string) string {
	switch t {
	case "indoor_start":
		return "bg-blue-100 text-blue-700"
	case "transplant":
		return "bg-purple-100 text-purple-700"
	case "direct_sow":
		return "bg-green-100 text-green-700"
	default:
		return "bg-gray-100 text-gray-600"
	}
}
