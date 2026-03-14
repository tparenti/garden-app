package planting

import (
	_ "embed"
	"encoding/csv"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"
)

//go:embed data/frost_dates.csv
var frostCSV []byte

// FrostDate holds average spring and fall frost dates for a location.
type FrostDate struct {
	ZipCode       string
	City          string
	State         string
	LastFrostMMDD string // e.g. "0415" = April 15
	FirstFrostMMDD string // e.g. "1015" = October 15
}

// FrostDateService resolves frost dates by zip code or state.
type FrostDateService struct {
	byZip    map[string]FrostDate
	byPrefix map[string]FrostDate // 3-digit zip prefix fallback
	byState  map[string][]FrostDate
}

// NewFrostDateService parses the embedded CSV and builds lookup maps.
func NewFrostDateService() (*FrostDateService, error) {
	svc := &FrostDateService{
		byZip:    make(map[string]FrostDate),
		byPrefix: make(map[string]FrostDate),
		byState:  make(map[string][]FrostDate),
	}

	r := csv.NewReader(strings.NewReader(string(frostCSV)))
	// skip header
	if _, err := r.Read(); err != nil {
		return nil, fmt.Errorf("read frost csv header: %w", err)
	}

	for {
		rec, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("parse frost csv: %w", err)
		}
		if len(rec) < 5 {
			continue
		}
		fd := FrostDate{
			ZipCode:        strings.TrimSpace(rec[0]),
			City:           strings.TrimSpace(rec[1]),
			State:          strings.TrimSpace(rec[2]),
			LastFrostMMDD:  strings.TrimSpace(rec[3]),
			FirstFrostMMDD: strings.TrimSpace(rec[4]),
		}
		svc.byZip[fd.ZipCode] = fd
		prefix := fd.ZipCode
		if len(prefix) >= 3 {
			prefix = prefix[:3]
		}
		// first record wins for prefix
		if _, exists := svc.byPrefix[prefix]; !exists {
			svc.byPrefix[prefix] = fd
		}
		svc.byState[strings.ToUpper(fd.State)] = append(svc.byState[strings.ToUpper(fd.State)], fd)
	}
	return svc, nil
}

// LookupByZip returns the frost date for a zip code, falling back to 3-digit prefix.
func (s *FrostDateService) LookupByZip(zip string) (*FrostDate, error) {
	zip = strings.TrimSpace(zip)
	if fd, ok := s.byZip[zip]; ok {
		return &fd, nil
	}
	if len(zip) >= 3 {
		if fd, ok := s.byPrefix[zip[:3]]; ok {
			return &fd, nil
		}
	}
	return nil, fmt.Errorf("no frost data found for zip %q", zip)
}

// LookupByState returns the median frost dates for a state abbreviation.
func (s *FrostDateService) LookupByState(state string) (*FrostDate, error) {
	entries, ok := s.byState[strings.ToUpper(state)]
	if !ok || len(entries) == 0 {
		return nil, fmt.Errorf("no frost data for state %q", state)
	}
	// Return the median entry as a rough representative
	median := entries[len(entries)/2]
	return &median, nil
}

// ParseDate converts an MMDD string and year into a time.Time.
func ParseDate(mmdd string, year int) (time.Time, error) {
	if len(mmdd) != 4 {
		return time.Time{}, fmt.Errorf("invalid date format %q (expected MMDD)", mmdd)
	}
	month, err := strconv.Atoi(mmdd[:2])
	if err != nil {
		return time.Time{}, err
	}
	day, err := strconv.Atoi(mmdd[2:])
	if err != nil {
		return time.Time{}, err
	}
	return time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.Local), nil
}
