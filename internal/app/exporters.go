package app

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// writeString writes to w and logs any error (helper for ICS generation)
func writeString(w io.Writer, s string) {
	if _, err := fmt.Fprint(w, s); err != nil {
		log.Printf("Error writing to response: %v", err)
	}
}

// GenerateICS generates an iCalendar (ICS) file with optional reminders
func GenerateICS(w http.ResponseWriter, r *http.Request, district string, year int, events []Event) {
	// Parse reminder settings
	reminder2Days := r.URL.Query().Get("reminder2Days") == "true"
	reminder1Day := r.URL.Query().Get("reminder1Day") == "true"
	reminderSameDay := r.URL.Query().Get("reminderSameDay") == "true"
	time2Days := r.URL.Query().Get("time2Days")
	time1Day := r.URL.Query().Get("time1Day")
	timeSameDay := r.URL.Query().Get("timeSameDay")

	w.Header().Set("Content-Type", "text/calendar; charset=utf-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=abfallkalender_%s_%d.ics", district, year))

	// ICS header
	fmt.Fprintln(w, "BEGIN:VCALENDAR")
	fmt.Fprintln(w, "VERSION:2.0")
	fmt.Fprintf(w, "PRODID:%s\n", ICSProductID)
	fmt.Fprintf(w, "X-WR-CALNAME:Abfallkalender %s %d\n", district, year)
	fmt.Fprintf(w, "X-WR-TIMEZONE:%s\n", ICSTimezone)
	fmt.Fprintln(w, "CALSCALE:GREGORIAN")

	// Generate events
	for _, event := range events {
		// Parse event date
		eventDate, err := time.Parse("2006-01-02", event.Date)
		if err != nil {
			continue
		}

		// Generate UID
		uid := fmt.Sprintf("%s-%s-%s@abfallkalender.winterberg.de", event.Date, event.Type, district)

		// Event - all-day event
		fmt.Fprintln(w, "BEGIN:VEVENT")
		fmt.Fprintf(w, "UID:%s\n", uid)
		fmt.Fprintf(w, "DTSTAMP:%s\n", time.Now().UTC().Format("20060102T150405Z"))
		fmt.Fprintf(w, "DTSTART;VALUE=DATE:%s\n", eventDate.Format("20060102"))
		fmt.Fprintf(w, "DTEND;VALUE=DATE:%s\n", eventDate.AddDate(0, 0, 1).Format("20060102"))
		fmt.Fprintf(w, "SUMMARY:%s\n", event.Description)
		fmt.Fprintf(w, "DESCRIPTION:Abfuhr %s in %s\n", event.Description, district)
		fmt.Fprintf(w, "LOCATION:%s\n", district)

		// Add reminders
		if reminder2Days && time2Days != "" {
			AddAlarm(w, eventDate, 2, time2Days, event.Description)
		}
		if reminder1Day && time1Day != "" {
			AddAlarm(w, eventDate, 1, time1Day, event.Description)
		}
		if reminderSameDay && timeSameDay != "" {
			AddAlarm(w, eventDate, 0, timeSameDay, event.Description)
		}

		fmt.Fprintln(w, "END:VEVENT")
	}

	fmt.Fprintln(w, "END:VCALENDAR")
}

// AddAlarm adds an alarm/reminder to an ICS event
func AddAlarm(w io.Writer, eventDate time.Time, daysBefore int, alarmTime string, description string) {
	// Parse alarm time (HH:MM format)
	parts := strings.Split(alarmTime, ":")
	if len(parts) != 2 {
		return
	}

	hour, err1 := strconv.Atoi(parts[0])
	minute, err2 := strconv.Atoi(parts[1])
	if err1 != nil || err2 != nil {
		return
	}

	// Calculate absolute alarm datetime
	// Event is at 00:00 on eventDate, alarm should be at alarmTime on (eventDate - daysBefore)
	alarmDate := eventDate.AddDate(0, 0, -daysBefore)
	alarmDateTime := time.Date(alarmDate.Year(), alarmDate.Month(), alarmDate.Day(), hour, minute, 0, 0, time.UTC)

	// For all-day events starting at midnight, we need to calculate trigger relative to event start
	eventStart := time.Date(eventDate.Year(), eventDate.Month(), eventDate.Day(), 0, 0, 0, 0, time.UTC)
	duration := alarmDateTime.Sub(eventStart)

	// Format as ISO 8601 duration
	// For triggers before the event, we need negative duration
	totalMinutes := int(duration.Minutes())
	isNegative := totalMinutes < 0
	if isNegative {
		totalMinutes = -totalMinutes
	}

	days := totalMinutes / (24 * 60)
	remainingMinutes := totalMinutes % (24 * 60)
	hours := remainingMinutes / 60
	minutes := remainingMinutes % 60

	var trigger string
	if isNegative {
		trigger = fmt.Sprintf("-P%dDT%dH%dM", days, hours, minutes)
	} else {
		trigger = fmt.Sprintf("P%dDT%dH%dM", days, hours, minutes)
	}

	fmt.Fprintln(w, "BEGIN:VALARM")
	fmt.Fprintln(w, "ACTION:DISPLAY")
	fmt.Fprintf(w, "DESCRIPTION:Erinnerung: %s\n", description)
	fmt.Fprintf(w, "TRIGGER:%s\n", trigger)
	fmt.Fprintln(w, "END:VALARM")
}

// GenerateCSV generates a CSV file with waste collection events
func GenerateCSV(w http.ResponseWriter, district string, year int, events []Event) {
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=abfallkalender_%s_%d.csv", district, year))

	// CSV header
	fmt.Fprintln(w, "Datum,Abfalltyp,Beschreibung")

	// CSV rows
	for _, event := range events {
		fmt.Fprintf(w, "%s,%s,%s\n", event.Date, event.Type, event.Description)
	}
}

// GenerateJSON generates a JSON file with waste collection events
func GenerateJSON(w http.ResponseWriter, district string, year int, events []Event) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=abfallkalender_%s_%d.json", district, year))

	data := map[string]interface{}{
		"district": district,
		"year":     year,
		"events":   events,
	}

	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Printf("Error encoding JSON export: %v", err)
		http.Error(w, ErrFailedToGenerateJSON, http.StatusInternalServerError)
	}
}

// GenerateSubscriptionICS generates an iCalendar (ICS) subscription feed
// Unlike GenerateICS, this is designed for calendar subscriptions:
// - No Content-Disposition attachment header (inline content)
// - No VALARM blocks (most calendar apps ignore them in subscriptions)
// - Includes METHOD:PUBLISH and refresh interval headers
func GenerateSubscriptionICS(w http.ResponseWriter, r *http.Request, district string, events []Event) {
	w.Header().Set("Content-Type", "text/calendar; charset=utf-8")
	// No Content-Disposition header - calendar apps need inline content for subscriptions

	// ICS header for subscription
	fmt.Fprintln(w, "BEGIN:VCALENDAR")
	fmt.Fprintln(w, "VERSION:2.0")
	fmt.Fprintf(w, "PRODID:%s\n", ICSProductID)
	fmt.Fprintln(w, "METHOD:PUBLISH") // Required for subscriptions
	fmt.Fprintf(w, "X-WR-CALNAME:Abfallkalender %s\n", district)
	fmt.Fprintf(w, "X-WR-TIMEZONE:%s\n", ICSTimezone)
	fmt.Fprintln(w, "CALSCALE:GREGORIAN")
	fmt.Fprintln(w, "X-PUBLISHED-TTL:PT1H") // Suggest refresh every 1 hour

	// Generate events
	for _, event := range events {
		// Parse event date
		eventDate, err := time.Parse("2006-01-02", event.Date)
		if err != nil {
			continue
		}

		// Generate UID - must be stable for proper calendar updates
		uid := fmt.Sprintf("%s-%s-%s@abfallkalender.winterberg.de", event.Date, event.Type, district)

		// Event - all-day event
		fmt.Fprintln(w, "BEGIN:VEVENT")
		fmt.Fprintf(w, "UID:%s\n", uid)
		fmt.Fprintf(w, "DTSTAMP:%s\n", time.Now().UTC().Format("20060102T150405Z"))
		fmt.Fprintf(w, "DTSTART;VALUE=DATE:%s\n", eventDate.Format("20060102"))
		fmt.Fprintf(w, "DTEND;VALUE=DATE:%s\n", eventDate.AddDate(0, 0, 1).Format("20060102"))
		fmt.Fprintf(w, "SUMMARY:%s\n", event.Description)
		fmt.Fprintf(w, "DESCRIPTION:Abfuhr %s in %s\n", event.Description, district)
		fmt.Fprintf(w, "LOCATION:%s\n", district)

		// Note: No VALARM blocks for subscriptions
		// Calendar apps typically ignore alarms in subscribed calendars
		// Users should set their own reminders in their calendar app

		fmt.Fprintln(w, "END:VEVENT")
	}

	fmt.Fprintln(w, "END:VCALENDAR")
}
