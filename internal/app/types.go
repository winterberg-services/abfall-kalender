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

// YearData represents calendar data for a single year
type YearData struct {
	Year      int                  `json:"year"`
	Districts map[string]*District `json:"districts"`
}

// CalendarStore holds all loaded calendar years
type CalendarStore struct {
	Years     map[int]*YearData // year -> data
	YearsList []int             // sorted list of available years
}
