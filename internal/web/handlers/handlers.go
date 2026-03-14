package handlers

import (
	"embed"
	"fmt"
	"html/template"
	"net/http"
	"time"

	"github.com/alexramsey92/garden-app/internal/models"
	"github.com/alexramsey92/garden-app/internal/planting"
	"github.com/alexramsey92/garden-app/internal/store"
)

// monthMarker is one month label with its percentage position in the year.
type monthMarker struct {
	Name string
	Pct  float64
}

// timelineBar is a pre-computed bar segment for the Gantt chart.
type timelineBar struct {
	Left  float64
	Width float64
	Color string
	Title string
}

// timelineRow is one row in the Gantt chart.
type timelineRow struct {
	Label    string
	TrayName string
	BedName  string
	Bars     []timelineBar
}

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

	// Trays
	mux.HandleFunc("GET /trays", h.TraysList)
	mux.HandleFunc("GET /trays/new", h.TraysNew)
	mux.HandleFunc("POST /trays", h.TraysCreate)
	mux.HandleFunc("DELETE /trays/{id}", h.TraysDelete)
	mux.HandleFunc("GET /trays/{id}", h.TraysView)
	mux.HandleFunc("GET /trays/{id}/cells/{cid}/edit", h.TraysCellEditForm)
	mux.HandleFunc("POST /trays/{id}/cells/{cid}", h.TraysCellSave)
	mux.HandleFunc("DELETE /trays/{id}/cells/{cid}", h.TraysCellClear)
	mux.HandleFunc("POST /trays/{id}/cells/bulk", h.TraysCellsBulkSow)
	mux.HandleFunc("GET /trays/{id}/cells/{cid}/transplant", h.TraysCellTransplantForm)
	mux.HandleFunc("POST /trays/{id}/cells/{cid}/transplant", h.TraysCellTransplant)

	// Beds
	mux.HandleFunc("GET /beds", h.BedsList)
	mux.HandleFunc("GET /beds/new", h.BedsNew)
	mux.HandleFunc("POST /beds", h.BedsCreate)
	mux.HandleFunc("DELETE /beds/{id}", h.BedsDelete)
	mux.HandleFunc("GET /beds/{id}", h.BedsView)
	mux.HandleFunc("GET /beds/{id}/cells/{cid}/edit", h.BedsCellEditForm)
	mux.HandleFunc("POST /beds/{id}/cells/{cid}", h.BedsCellSave)
	mux.HandleFunc("DELETE /beds/{id}/cells/{cid}", h.BedsCellClear)
	mux.HandleFunc("POST /beds/{id}/cells/bulk", h.BedsCellsBulkSet)

	// Timeline
	mux.HandleFunc("GET /timeline", h.Timeline)
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
		{"trays", "templates/trays.html"},
		{"trays_new", "templates/trays_new.html"},
		{"trays_view", "templates/trays_view.html"},
		{"beds", "templates/beds.html"},
		{"beds_new", "templates/beds_new.html"},
		{"beds_view", "templates/beds_view.html"},
		{"timeline", "templates/timeline.html"},
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
		"formatDate": func(t time.Time) string { return t.Format("2006-01-02") },
		"formatDatePtr": func(t *time.Time) string {
			if t == nil {
				return ""
			}
			return t.Format("2006-01-02")
		},
		"neg":       func(n int) int { return -n },
		"lt0":       func(n int) bool { return n < 0 },
		"fmtPct":    func(f float64) string { return fmt.Sprintf("%.2f", f) },
		"todayPct":  func(year int) float64 { n := time.Now(); return datePctFloat(&n, year) },
		"cellBg":    cellBg,
		"cellText":  cellTextColor,
		"statusDot": statusDot,
		"abbrev":    abbrev,
		"imul":      func(a, b int) int { return a * b },
		"seq": func(n int) []int {
			s := make([]int, n)
			for i := range s {
				s[i] = i
			}
			return s
		},
		"trayCellData": func(trayID int64, cell models.TrayCell) trayCellPartialData {
			return trayCellPartialData{TrayID: trayID, Cell: cell}
		},
		"bedCellData": func(bedID int64, cell models.BedCell) bedCellPartialData {
			return bedCellPartialData{BedID: bedID, Cell: cell}
		},
	}
}

// cellBg returns a Tailwind bg color class based on the plant label and status.
func cellBg(label, status string) string {
	switch status {
	case "empty":
		return "bg-gray-50 hover:bg-gray-100"
	case "failed":
		return "bg-red-100 hover:bg-red-200"
	case "transplanted":
		return "bg-purple-100 hover:bg-purple-200"
	}
	if label == "" {
		return "bg-gray-50 hover:bg-gray-100"
	}
	colors := []string{
		"bg-green-200 hover:bg-green-300",
		"bg-blue-200 hover:bg-blue-300",
		"bg-amber-200 hover:bg-amber-300",
		"bg-rose-200 hover:bg-rose-300",
		"bg-teal-200 hover:bg-teal-300",
		"bg-orange-200 hover:bg-orange-300",
		"bg-cyan-200 hover:bg-cyan-300",
		"bg-lime-200 hover:bg-lime-300",
		"bg-pink-200 hover:bg-pink-300",
		"bg-indigo-200 hover:bg-indigo-300",
		"bg-yellow-200 hover:bg-yellow-300",
		"bg-violet-200 hover:bg-violet-300",
	}
	h := labelHash(label)
	return colors[h%len(colors)]
}

// cellTextColor returns a Tailwind text color class for cell content.
func cellTextColor(label, status string) string {
	if status == "failed" {
		return "text-red-700"
	}
	if status == "transplanted" {
		return "text-purple-700"
	}
	if label == "" {
		return "text-gray-400"
	}
	colors := []string{
		"text-green-800", "text-blue-800", "text-amber-800", "text-rose-800",
		"text-teal-800", "text-orange-800", "text-cyan-800", "text-lime-800",
		"text-pink-800", "text-indigo-800", "text-yellow-800", "text-violet-800",
	}
	h := labelHash(label)
	return colors[h%len(colors)]
}

// statusDot returns a Tailwind bg color for the status indicator dot.
func statusDot(status string) string {
	switch status {
	case "sown":
		return "bg-blue-400"
	case "germinated":
		return "bg-green-500"
	case "transplanted":
		return "bg-purple-500"
	case "failed":
		return "bg-red-400"
	case "planted":
		return "bg-lime-500"
	case "growing":
		return "bg-green-600"
	case "harvested":
		return "bg-amber-500"
	default:
		return "bg-transparent"
	}
}

func abbrev(s string, n int) string {
	runes := []rune(s)
	if len(runes) <= n {
		return s
	}
	return string(runes[:n])
}

func labelHash(s string) int {
	h := 0
	for _, c := range s {
		h = h*31 + int(c)
	}
	if h < 0 {
		h = -h
	}
	return h
}

// buildTimelineRows converts TimelineItems to pre-computed chart rows.
func buildTimelineRows(items []models.TimelineItem, year int) []timelineRow {
	rows := make([]timelineRow, 0, len(items))
	today := time.Now()
	for _, it := range items {
		row := timelineRow{
			Label:    it.Label,
			TrayName: it.TrayName,
			BedName:  it.BedName,
		}

		// Tray phase bar
		if it.SownAt != nil {
			trayEnd := &today
			color := "bg-blue-300"
			title := "In tray"
			if it.TrayFailedAt != nil && it.PlantedAt == nil {
				trayEnd = it.TrayFailedAt
				color = "bg-red-300"
				title = "Failed in tray"
			} else if it.PlantedAt != nil {
				trayEnd = it.PlantedAt
				color = "bg-purple-300"
				title = "Tray → transplanted"
			} else if it.GerminatedAt != nil {
				color = "bg-green-300"
				title = "Germinated"
			}
			left := datePctFloat(it.SownAt, year)
			right := datePctFloat(trayEnd, year)
			if right > left {
				row.Bars = append(row.Bars, timelineBar{Left: left, Width: right - left, Color: color, Title: title})
			}
		}

		// Bed phase bar
		if it.PlantedAt != nil {
			bedEnd := &today
			color := "bg-green-400"
			title := "In bed"
			if it.BedFailedAt != nil {
				bedEnd = it.BedFailedAt
				color = "bg-red-300"
				title = "Failed in bed"
			} else if it.HarvestedAt != nil {
				bedEnd = it.HarvestedAt
				color = "bg-amber-300"
				title = "Harvested"
			}
			left := datePctFloat(it.PlantedAt, year)
			right := datePctFloat(bedEnd, year)
			if right > left {
				row.Bars = append(row.Bars, timelineBar{Left: left, Width: right - left, Color: color, Title: title})
			}
		}

		rows = append(rows, row)
	}
	return rows
}

func datePctFloat(t *time.Time, year int) float64 {
	if t == nil {
		return 0
	}
	start := time.Date(year, 1, 1, 0, 0, 0, 0, time.Local)
	end := time.Date(year+1, 1, 1, 0, 0, 0, 0, time.Local)
	since := t.Sub(start)
	if since < 0 {
		return 0
	}
	pct := float64(since) / float64(end.Sub(start)) * 100
	if pct > 100 {
		return 100
	}
	return pct
}

func yearMonthMarkers(year int) []monthMarker {
	months := make([]monthMarker, 12)
	for m := range 12 {
		d := time.Date(year, time.Month(m+1), 1, 0, 0, 0, 0, time.Local)
		months[m] = monthMarker{
			Name: d.Format("Jan"),
			Pct:  datePctFloat(&d, year),
		}
	}
	return months
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
