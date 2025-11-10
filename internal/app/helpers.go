package app

import (
	"net/http"
	"sort"
)

// RequireMethod validates that the request uses the specified HTTP method
func RequireMethod(w http.ResponseWriter, r *http.Request, method string) bool {
	if r.Method != method {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return false
	}
	return true
}

// RequireEditMode validates that edit mode is enabled
func RequireEditMode(w http.ResponseWriter) bool {
	if !EditMode {
		http.Error(w, ErrEditModeDisabled, http.StatusForbidden)
		return false
	}
	return true
}

// SortEventsByDate sorts events by date in ascending order
func SortEventsByDate(events []Event) {
	sort.Slice(events, func(i, j int) bool {
		return events[i].Date < events[j].Date
	})
}
