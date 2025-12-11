package app

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// ServeIndex serves the download interface HTML
func ServeIndex(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if _, err := w.Write(IndexHTML); err != nil {
		log.Printf("Error writing index HTML: %v", err)
	}
}

// ServeEdit serves the editor interface HTML
func ServeEdit(w http.ResponseWriter, r *http.Request) {
	if !RequireEditMode(w) {
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if _, err := w.Write(EditHTML); err != nil {
		log.Printf("Error writing edit HTML: %v", err)
	}
}

// GetConfig returns the application configuration
func GetConfig(w http.ResponseWriter, r *http.Request) {
	currentYear := GetCurrentYear()
	availableYears := GetAvailableYears()

	config := map[string]interface{}{
		"districts":      Districts,
		"wasteTypes":     WasteTypes,
		"currentYear":    currentYear,
		"availableYears": availableYears,
		"editMode":       EditMode,
		"holidays":       GetNRWHolidays(currentYear),
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(config); err != nil {
		log.Printf("Error encoding config: %v", err)
		http.Error(w, ErrInternalServer, http.StatusInternalServerError)
	}
}

// HandleCalendar returns calendar data for a specific year
// Query param: year (optional, defaults to current year)
func HandleCalendar(w http.ResponseWriter, r *http.Request) {
	yearStr := r.URL.Query().Get("year")
	year := GetCurrentYear()

	if yearStr != "" {
		var err error
		year, err = strconv.Atoi(yearStr)
		if err != nil {
			http.Error(w, ErrInvalidYear, http.StatusBadRequest)
			return
		}
	}

	yearData, ok := GetYear(year)
	if !ok {
		http.Error(w, ErrYearNotFound, http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(yearData); err != nil {
		log.Printf("Error encoding calendar: %v", err)
		http.Error(w, ErrInternalServer, http.StatusInternalServerError)
	}
}

// HandleCalendarCommit commits temporary changes
func HandleCalendarCommit(w http.ResponseWriter, r *http.Request) {
	if !RequireMethod(w, r, http.MethodPost) || !RequireEditMode(w) {
		return
	}

	if err := CommitAllYears(); err != nil {
		log.Printf("Error committing calendar: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]string{"status": "ok"}); err != nil {
		log.Printf("Error encoding response: %v", err)
	}
}

// HandleCalendarRevert reverts temporary changes
func HandleCalendarRevert(w http.ResponseWriter, r *http.Request) {
	if !RequireMethod(w, r, http.MethodPost) || !RequireEditMode(w) {
		return
	}

	if err := RevertAllYears(); err != nil {
		log.Printf("Error reverting calendar: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]string{"status": "ok"}); err != nil {
		log.Printf("Error encoding response: %v", err)
	}
}

// HandleCalendarStatus returns whether there are unsaved changes
func HandleCalendarStatus(w http.ResponseWriter, r *http.Request) {
	if !RequireEditMode(w) {
		return
	}

	w.Header().Set("Content-Type", "application/json")
	status := map[string]bool{
		"has_changes": HasTmpChanges(),
	}
	if err := json.NewEncoder(w).Encode(status); err != nil {
		log.Printf("Error encoding response: %v", err)
	}
}

// HandleDistrictCalendar returns calendar data for a specific district
// URL: /api/calendar/{district}?year=2025
func HandleDistrictCalendar(w http.ResponseWriter, r *http.Request) {
	district := r.URL.Path[len("/api/calendar/"):]
	yearStr := r.URL.Query().Get("year")
	year := GetCurrentYear()

	if yearStr != "" {
		var err error
		year, err = strconv.Atoi(yearStr)
		if err != nil {
			http.Error(w, ErrInvalidYear, http.StatusBadRequest)
			return
		}
	}

	yearData, ok := GetYear(year)
	if !ok {
		http.Error(w, ErrYearNotFound, http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if dist, ok := yearData.Districts[district]; ok {
		if err := json.NewEncoder(w).Encode(dist); err != nil {
			log.Printf("Error encoding district calendar: %v", err)
			http.Error(w, ErrInternalServer, http.StatusInternalServerError)
		}
	} else {
		if err := json.NewEncoder(w).Encode(&District{Events: []Event{}}); err != nil {
			log.Printf("Error encoding empty district: %v", err)
			http.Error(w, ErrInternalServer, http.StatusInternalServerError)
		}
	}
}

// AddEvent adds a new event to the calendar (edit mode only)
func AddEvent(w http.ResponseWriter, r *http.Request) {
	if !RequireMethod(w, r, http.MethodPost) || !RequireEditMode(w) {
		return
	}

	var req struct {
		District  string `json:"district"`
		Date      string `json:"date"`
		WasteType string `json:"waste_type"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Validate date format and extract year
	eventDate, err := time.Parse("2006-01-02", req.Date)
	if err != nil {
		http.Error(w, ErrInvalidDateFormat, http.StatusBadRequest)
		return
	}
	year := eventDate.Year()

	CalendarMutex.Lock()
	defer CalendarMutex.Unlock()

	// Get or create year data
	yearData, ok := Store.Years[year]
	if !ok {
		// Create new year
		yearData = &YearData{
			Year:      year,
			Districts: make(map[string]*District),
		}
		Store.Years[year] = yearData
		Store.YearsList = append(Store.YearsList, year)
	}

	// Initialize district if needed
	if yearData.Districts[req.District] == nil {
		yearData.Districts[req.District] = &District{Events: []Event{}}
	}

	// Check if event already exists
	events := yearData.Districts[req.District].Events
	for _, e := range events {
		if e.Date == req.Date && e.Type == req.WasteType {
			w.Header().Set("Content-Type", "application/json")
			if err := json.NewEncoder(w).Encode(map[string]string{"status": "exists"}); err != nil {
				log.Printf("Error encoding response: %v", err)
			}
			return
		}
	}

	// Add event
	event := Event{
		Date:        req.Date,
		Type:        req.WasteType,
		Description: WasteTypes[req.WasteType],
	}
	yearData.Districts[req.District].Events = append(
		yearData.Districts[req.District].Events,
		event,
	)

	// Sort by date
	SortEventsByDate(yearData.Districts[req.District].Events)

	// Auto-save to tmp file
	if err := saveTmpYear(year); err != nil {
		log.Printf("Error saving tmp calendar: %v", err)
		http.Error(w, ErrFailedToSave, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]string{"status": "ok"}); err != nil {
		log.Printf("Error encoding response: %v", err)
	}
}

// DeleteEvent deletes an event from the calendar (edit mode only)
func DeleteEvent(w http.ResponseWriter, r *http.Request) {
	if !RequireMethod(w, r, http.MethodPost) || !RequireEditMode(w) {
		return
	}

	var req struct {
		District string `json:"district"`
		Date     string `json:"date"`
		Type     string `json:"type"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Extract year from date
	eventDate, err := time.Parse("2006-01-02", req.Date)
	if err != nil {
		http.Error(w, ErrInvalidDateFormat, http.StatusBadRequest)
		return
	}
	year := eventDate.Year()

	CalendarMutex.Lock()
	defer CalendarMutex.Unlock()

	yearData, ok := Store.Years[year]
	if !ok {
		http.Error(w, ErrYearNotFound, http.StatusNotFound)
		return
	}

	if dist, ok := yearData.Districts[req.District]; ok {
		newEvents := []Event{}
		for _, e := range dist.Events {
			if !(e.Date == req.Date && e.Type == req.Type) {
				newEvents = append(newEvents, e)
			}
		}
		dist.Events = newEvents

		if err := saveTmpYear(year); err != nil {
			log.Printf("Error saving tmp calendar: %v", err)
			http.Error(w, ErrFailedToSave, http.StatusInternalServerError)
			return
		}
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]string{"status": "ok"}); err != nil {
		log.Printf("Error encoding response: %v", err)
	}
}

// MoveEvent moves an event to a different date (edit mode only)
func MoveEvent(w http.ResponseWriter, r *http.Request) {
	if !RequireMethod(w, r, http.MethodPost) || !RequireEditMode(w) {
		return
	}

	var req struct {
		District string `json:"district"`
		OldDate  string `json:"old_date"`
		NewDate  string `json:"new_date"`
		Type     string `json:"type"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Extract years
	oldDate, err := time.Parse("2006-01-02", req.OldDate)
	if err != nil {
		http.Error(w, ErrInvalidDateFormat, http.StatusBadRequest)
		return
	}
	newDate, err := time.Parse("2006-01-02", req.NewDate)
	if err != nil {
		http.Error(w, ErrInvalidDateFormat, http.StatusBadRequest)
		return
	}

	oldYear := oldDate.Year()
	newYear := newDate.Year()

	CalendarMutex.Lock()
	defer CalendarMutex.Unlock()

	// If same year, just update date
	if oldYear == newYear {
		yearData, ok := Store.Years[oldYear]
		if !ok {
			http.Error(w, ErrYearNotFound, http.StatusNotFound)
			return
		}

		if dist, ok := yearData.Districts[req.District]; ok {
			for i := range dist.Events {
				if dist.Events[i].Date == req.OldDate && dist.Events[i].Type == req.Type {
					dist.Events[i].Date = req.NewDate
					break
				}
			}
			SortEventsByDate(dist.Events)

			if err := saveTmpYear(oldYear); err != nil {
				log.Printf("Error saving tmp calendar: %v", err)
				http.Error(w, ErrFailedToSave, http.StatusInternalServerError)
				return
			}
		}
	} else {
		// Moving between years - delete from old, add to new
		// Delete from old year
		oldYearData, ok := Store.Years[oldYear]
		if ok {
			if dist, ok := oldYearData.Districts[req.District]; ok {
				newEvents := []Event{}
				var movedEvent Event
				for _, e := range dist.Events {
					if e.Date == req.OldDate && e.Type == req.Type {
						movedEvent = e
					} else {
						newEvents = append(newEvents, e)
					}
				}
				dist.Events = newEvents

				// Add to new year
				newYearData, ok := Store.Years[newYear]
				if !ok {
					newYearData = &YearData{
						Year:      newYear,
						Districts: make(map[string]*District),
					}
					Store.Years[newYear] = newYearData
					Store.YearsList = append(Store.YearsList, newYear)
				}

				if newYearData.Districts[req.District] == nil {
					newYearData.Districts[req.District] = &District{Events: []Event{}}
				}

				movedEvent.Date = req.NewDate
				newYearData.Districts[req.District].Events = append(
					newYearData.Districts[req.District].Events,
					movedEvent,
				)
				SortEventsByDate(newYearData.Districts[req.District].Events)

				// Save both years
				if err := saveTmpYear(oldYear); err != nil {
					log.Printf("Error saving tmp calendar: %v", err)
				}
				if err := saveTmpYear(newYear); err != nil {
					log.Printf("Error saving tmp calendar: %v", err)
					http.Error(w, ErrFailedToSave, http.StatusInternalServerError)
					return
				}
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]string{"status": "ok"}); err != nil {
		log.Printf("Error encoding response: %v", err)
	}
}

// HandleDownload handles export downloads in ICS, CSV or JSON format
func HandleDownload(w http.ResponseWriter, r *http.Request) {
	district := r.URL.Query().Get("district")
	year := r.URL.Query().Get("year")
	format := r.URL.Query().Get("format")
	wasteTypesFilter := r.URL.Query().Get("wasteTypes")

	// Parse year
	yearInt, err := strconv.Atoi(year)
	if err != nil {
		http.Error(w, ErrInvalidYear, http.StatusBadRequest)
		return
	}

	// Get events for district and year
	yearData, ok := GetYear(yearInt)
	if !ok {
		http.Error(w, ErrYearNotFound, http.StatusNotFound)
		return
	}

	var events []Event
	if dist, ok := yearData.Districts[district]; ok {
		events = make([]Event, len(dist.Events))
		copy(events, dist.Events)
	}

	// Filter by waste types if specified
	if wasteTypesFilter != "" {
		types := strings.Split(wasteTypesFilter, ",")
		typeMap := make(map[string]bool)
		for _, t := range types {
			typeMap[t] = true
		}

		var filtered []Event
		for _, e := range events {
			if typeMap[e.Type] {
				filtered = append(filtered, e)
			}
		}
		events = filtered
	}

	switch format {
	case "ics":
		GenerateICS(w, r, district, yearInt, events)
	case "csv":
		GenerateCSV(w, district, yearInt, events)
	case "json":
		GenerateJSON(w, district, yearInt, events)
	default:
		http.Error(w, ErrInvalidFormat, http.StatusBadRequest)
	}
}

// HandleSubscribe handles calendar subscription requests
// Returns an ICS feed with events from (current year - 1) onwards
func HandleSubscribe(w http.ResponseWriter, r *http.Request) {
	district := r.URL.Path[len("/api/subscribe/"):]
	wasteTypesFilter := r.URL.Query().Get("wasteTypes")

	// Get all events for this district across all years
	events := GetAllEvents(district)

	// Calculate minimum year (current year - 1)
	currentYear := time.Now().Year()
	minYear := currentYear - 1

	// Filter by waste types if specified
	if wasteTypesFilter != "" {
		types := strings.Split(wasteTypesFilter, ",")
		typeMap := make(map[string]bool)
		for _, t := range types {
			typeMap[t] = true
		}

		var filtered []Event
		for _, e := range events {
			if typeMap[e.Type] {
				filtered = append(filtered, e)
			}
		}
		events = filtered
	}

	// Filter by year: include events from (current year - 1) onwards
	var filteredEvents []Event
	for _, e := range events {
		if len(e.Date) >= 4 {
			eventYear, err := strconv.Atoi(e.Date[:4])
			if err == nil && eventYear >= minYear {
				filteredEvents = append(filteredEvents, e)
			}
		}
	}

	GenerateSubscriptionICS(w, r, district, filteredEvents)
}
