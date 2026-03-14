package handlers

import (
	"net/http"
)

// FrostInfo is a shared data holder used across handlers.
type FrostInfo struct {
	City  string
	State string
	Last  string
	First string
}

// FormatMMDD converts "0415" → "April 15".
func FormatMMDD(mmdd string) string {
	if len(mmdd) != 4 {
		return mmdd
	}
	if mmdd == "0000" {
		return "No frost (tropical)"
	}
	months := []string{
		"", "January", "February", "March", "April", "May", "June",
		"July", "August", "September", "October", "November", "December",
	}
	m := int(mmdd[0]-'0')*10 + int(mmdd[1]-'0')
	d := int(mmdd[2]-'0')*10 + int(mmdd[3]-'0')
	if m < 1 || m > 12 || d < 1 || d > 31 {
		return mmdd
	}
	return months[m] + " " + itoa(d)
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	digits := []byte{}
	for n > 0 {
		digits = append([]byte{byte('0' + n%10)}, digits...)
		n /= 10
	}
	return string(digits)
}

type localeData struct {
	Flash      string
	Zip        string
	State      string
	City       string
	FrostState string
	LastFrost  string
	FirstFrost string
	Error      string
}

func (h *Handler) LocaleShow(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	data := localeData{Flash: r.URL.Query().Get("flash")}

	zip, _ := h.store.GetConfig(ctx, "zip")
	state, _ := h.store.GetConfig(ctx, "state")
	data.Zip = zip
	data.State = state

	if zip != "" {
		if fd, err := h.frostSvc.LookupByZip(zip); err == nil {
			data.City = fd.City
			data.FrostState = fd.State
			data.LastFrost = FormatMMDD(fd.LastFrostMMDD)
			data.FirstFrost = FormatMMDD(fd.FirstFrostMMDD)
		}
	} else if state != "" {
		if fd, err := h.frostSvc.LookupByState(state); err == nil {
			data.City = fd.City
			data.FrostState = fd.State
			data.LastFrost = FormatMMDD(fd.LastFrostMMDD)
			data.FirstFrost = FormatMMDD(fd.FirstFrostMMDD)
		}
	}

	h.render(w, "locale", data)
}

func (h *Handler) LocaleSet(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	zip := r.FormValue("zip")
	state := r.FormValue("state")

	if zip != "" {
		fd, err := h.frostSvc.LookupByZip(zip)
		if err != nil {
			data := localeData{Error: "Zip code not found in frost database."}
			h.render(w, "locale", data)
			return
		}
		_ = h.store.SetConfig(ctx, "zip", zip)
		_ = h.store.SetConfig(ctx, "state", "")
		_ = fd
	} else if state != "" {
		_, err := h.frostSvc.LookupByState(state)
		if err != nil {
			data := localeData{Error: "State not found in frost database."}
			h.render(w, "locale", data)
			return
		}
		_ = h.store.SetConfig(ctx, "state", state)
		_ = h.store.SetConfig(ctx, "zip", "")
	} else {
		data := localeData{Error: "Please enter a zip code or state."}
		h.render(w, "locale", data)
		return
	}

	http.Redirect(w, r, "/locale?flash=Locale+updated", http.StatusSeeOther)
}
