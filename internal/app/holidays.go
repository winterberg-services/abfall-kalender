package app

import (
	"time"
)

// GetNRWHolidays returns all public holidays in NRW for the given year
func GetNRWHolidays(year int) map[string]string {
	holidays := make(map[string]string)

	// Fixed holidays
	holidays[formatDate(year, 1, 1)] = "Neujahr"
	holidays[formatDate(year, 5, 1)] = "Tag der Arbeit"
	holidays[formatDate(year, 10, 3)] = "Tag der Deutschen Einheit"
	holidays[formatDate(year, 11, 1)] = "Allerheiligen"
	holidays[formatDate(year, 12, 25)] = "1. Weihnachtstag"
	holidays[formatDate(year, 12, 26)] = "2. Weihnachtstag"

	// Easter-based holidays (movable)
	easter := calculateEaster(year)

	// Karfreitag (Good Friday): Easter - 2 days
	holidays[formatDateFromTime(easter.AddDate(0, 0, -2))] = "Karfreitag"

	// Ostermontag (Easter Monday): Easter + 1 day
	holidays[formatDateFromTime(easter.AddDate(0, 0, 1))] = "Ostermontag"

	// Christi Himmelfahrt (Ascension Day): Easter + 39 days
	holidays[formatDateFromTime(easter.AddDate(0, 0, 39))] = "Christi Himmelfahrt"

	// Pfingstmontag (Whit Monday): Easter + 50 days
	holidays[formatDateFromTime(easter.AddDate(0, 0, 50))] = "Pfingstmontag"

	// Fronleichnam (Corpus Christi): Easter + 60 days
	holidays[formatDateFromTime(easter.AddDate(0, 0, 60))] = "Fronleichnam"

	return holidays
}

// calculateEaster calculates Easter Sunday using the Meeus/Jones/Butcher algorithm
func calculateEaster(year int) time.Time {
	a := year % 19
	b := year / 100
	c := year % 100
	d := b / 4
	e := b % 4
	f := (b + 8) / 25
	g := (b - f + 1) / 3
	h := (19*a + b - d - g + 15) % 30
	i := c / 4
	k := c % 4
	l := (32 + 2*e + 2*i - h - k) % 7
	m := (a + 11*h + 22*l) / 451
	month := (h + l - 7*m + 114) / 31
	day := ((h + l - 7*m + 114) % 31) + 1

	// Use noon to avoid timezone issues when formatting to YYYY-MM-DD
	return time.Date(year, time.Month(month), day, 12, 0, 0, 0, time.UTC)
}

// formatDate formats a date as YYYY-MM-DD
func formatDate(year, month, day int) string {
	// Use noon to avoid timezone issues when formatting to YYYY-MM-DD
	return time.Date(year, time.Month(month), day, 12, 0, 0, 0, time.UTC).Format("2006-01-02")
}

// formatDateFromTime formats a time.Time as YYYY-MM-DD
func formatDateFromTime(t time.Time) string {
	return t.Format("2006-01-02")
}
