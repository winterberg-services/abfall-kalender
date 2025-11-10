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
	CalendarMutex.RLock()
	currentYear := Calendar.Year
	CalendarMutex.RUnlock()

	if currentYear == 0 {
		currentYear = time.Now().Year()
	}

	config := map[string]interface{}{
		"districts":   Districts,
		"wasteTypes":  WasteTypes,
		"currentYear": currentYear,
		"editMode":    EditMode,
		"holidays":    GetNRWHolidays(currentYear),
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(config); err != nil {
		log.Printf("Error encoding config: %v", err)
		http.Error(w, ErrInternalServer, http.StatusInternalServerError)
	}
}

// HandleCalendar returns the complete calendar data
func HandleCalendar(w http.ResponseWriter, r *http.Request) {
	CalendarMutex.RLock()
	defer CalendarMutex.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(Calendar); err != nil {
		log.Printf("Error encoding calendar: %v", err)
		http.Error(w, ErrInternalServer, http.StatusInternalServerError)
	}
}

// HandleCalendarCommit commits temporary changes (creates backup and makes tmp the new main)
func HandleCalendarCommit(w http.ResponseWriter, r *http.Request) {
	if !RequireMethod(w, r, http.MethodPost) || !RequireEditMode(w) {
		return
	}

	if err := CommitCalendar(); err != nil {
		log.Printf("Error committing calendar: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]string{"status": "ok"}); err != nil {
		log.Printf("Error encoding response: %v", err)
	}
}

// HandleCalendarRevert reverts temporary changes and reloads from main file
func HandleCalendarRevert(w http.ResponseWriter, r *http.Request) {
	if !RequireMethod(w, r, http.MethodPost) || !RequireEditMode(w) {
		return
	}

	if err := RevertCalendar(); err != nil {
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
		"has_changes": HasTmpCalendar(),
	}
	if err := json.NewEncoder(w).Encode(status); err != nil {
		log.Printf("Error encoding response: %v", err)
	}
}

// HandleDistrictCalendar returns calendar data for a specific district
func HandleDistrictCalendar(w http.ResponseWriter, r *http.Request) {
	district := r.URL.Path[len("/api/calendar/"):]

	CalendarMutex.RLock()
	defer CalendarMutex.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	if dist, ok := Calendar.Districts[district]; ok {
		if err := json.NewEncoder(w).Encode(dist); err != nil {
			log.Printf("Error encoding district calendar: %v", err)
			http.Error(w, ErrInternalServer, http.StatusInternalServerError)
		}
	} else {
		if err := json.NewEncoder(w).Encode(&District{Events: []Event{}}); err != nil {
			log.Printf("Error encoding empty district: %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
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

	// Validate date format
	if _, err := time.Parse("2006-01-02", req.Date); err != nil {
		http.Error(w, ErrInvalidDateFormat, http.StatusBadRequest)
		return
	}

	CalendarMutex.Lock()
	defer CalendarMutex.Unlock()

	// Initialize district if needed
	if Calendar.Districts[req.District] == nil {
		Calendar.Districts[req.District] = &District{Events: []Event{}}
	}

	// Check if event already exists
	events := Calendar.Districts[req.District].Events
	for _, e := range events {
		if e.Date == req.Date && e.Type == req.WasteType {
			// Already exists
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
	Calendar.Districts[req.District].Events = append(
		Calendar.Districts[req.District].Events,
		event,
	)

	// Sort by date
	SortEventsByDate(Calendar.Districts[req.District].Events)

	// Auto-save to tmp file
	if err := saveTmpCalendar(); err != nil {
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

	CalendarMutex.Lock()
	defer CalendarMutex.Unlock()

	if dist, ok := Calendar.Districts[req.District]; ok {
		newEvents := []Event{}
		for _, e := range dist.Events {
			if !(e.Date == req.Date && e.Type == req.Type) {
				newEvents = append(newEvents, e)
			}
		}
		dist.Events = newEvents
		// Auto-save to tmp file
		if err := saveTmpCalendar(); err != nil {
			log.Printf("Error saving tmp calendar: %v", err)
			http.Error(w, "Failed to save calendar", http.StatusInternalServerError)
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

	CalendarMutex.Lock()
	defer CalendarMutex.Unlock()

	if dist, ok := Calendar.Districts[req.District]; ok {
		for i := range dist.Events {
			if dist.Events[i].Date == req.OldDate && dist.Events[i].Type == req.Type {
				dist.Events[i].Date = req.NewDate
				break
			}
		}

		// Sort by date
		SortEventsByDate(dist.Events)

		// Auto-save to tmp file
		if err := saveTmpCalendar(); err != nil {
			log.Printf("Error saving tmp calendar: %v", err)
			http.Error(w, "Failed to save calendar", http.StatusInternalServerError)
			return
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

	// Get events for district
	CalendarMutex.RLock()
	var events []Event
	if dist, ok := Calendar.Districts[district]; ok {
		// Copy events to avoid holding lock during export generation
		events = make([]Event, len(dist.Events))
		copy(events, dist.Events)
	}
	CalendarMutex.RUnlock()

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

	// Filter by year
	var yearEvents []Event
	for _, e := range events {
		if strings.HasPrefix(e.Date, year+"-") {
			yearEvents = append(yearEvents, e)
		}
	}

	switch format {
	case "ics":
		GenerateICS(w, r, district, yearInt, yearEvents)
	case "csv":
		GenerateCSV(w, district, yearInt, yearEvents)
	case "json":
		GenerateJSON(w, district, yearInt, yearEvents)
	default:
		http.Error(w, ErrInvalidFormat, http.StatusBadRequest)
	}
}

// HandleSubscribe handles calendar subscription requests
// Returns an ICS feed that calendar apps can subscribe to for automatic updates
// Returns events from (current year - 1) onwards (e.g., in 2025: returns 2024, 2025, 2026, ...)
// URL format: /api/subscribe/{district}?wasteTypes=restmuell,biotonne
func HandleSubscribe(w http.ResponseWriter, r *http.Request) {
	// Extract district from path
	district := r.URL.Path[len("/api/subscribe/"):]

	// Get query parameters
	wasteTypesFilter := r.URL.Query().Get("wasteTypes")

	// Calculate minimum year (current year - 1)
	currentYear := time.Now().Year()
	minYear := currentYear - 1

	// Get events for district
	CalendarMutex.RLock()
	var events []Event
	if dist, ok := Calendar.Districts[district]; ok {
		// Copy events to avoid holding lock during export generation
		events = make([]Event, len(dist.Events))
		copy(events, dist.Events)
	}
	CalendarMutex.RUnlock()

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
		// Parse year from date (format: YYYY-MM-DD)
		if len(e.Date) >= 4 {
			eventYear, err := strconv.Atoi(e.Date[:4])
			if err == nil && eventYear >= minYear {
				filteredEvents = append(filteredEvents, e)
			}
		}
	}

	// Generate subscription ICS
	GenerateSubscriptionICS(w, r, district, filteredEvents)
}
