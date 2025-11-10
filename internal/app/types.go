package app

// Event represents a single waste collection event
type Event struct {
	Date        string `json:"date"`
	Type        string `json:"type"`
	Description string `json:"description"`
}

// District represents a district with its waste collection events
type District struct {
	Events []Event `json:"events"`
}

// CalendarData represents the complete calendar data structure
type CalendarData struct {
	Year      int                  `json:"year"`
	Districts map[string]*District `json:"districts"`
	Metadata  map[string]string    `json:"metadata"`
}
