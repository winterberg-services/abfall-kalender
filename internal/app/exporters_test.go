package app

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestGenerateICS(t *testing.T) {
	// Setup test data
	events := []Event{
		{Date: "2025-01-15", Type: "restmuell", Description: "Restmüll"},
		{Date: "2025-01-20", Type: "biotonne", Description: "Biotonne"},
	}

	// Create test request with reminders
	req := httptest.NewRequest("GET", "/api/download?reminder2Days=true&time2Days=18:00&reminder1Day=true&time1Day=19:00&reminderSameDay=true&timeSameDay=07:00", nil)
	w := httptest.NewRecorder()

	// Call GenerateICS
	GenerateICS(w, req, "Winterberg", 2025, events)

	// Get response
	resp := w.Result()
	body := w.Body.String()

	// Assertions
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	// Check content type
	contentType := resp.Header.Get("Content-Type")
	if !strings.Contains(contentType, "text/calendar") {
		t.Errorf("Expected Content-Type text/calendar, got %s", contentType)
	}

	// Check for required ICS structure
	requiredFields := []string{
		"BEGIN:VCALENDAR",
		"VERSION:2.0",
		"PRODID:-//Winterberg//Abfallkalender//DE",
		"BEGIN:VEVENT",
		"END:VEVENT",
		"END:VCALENDAR",
	}

	for _, field := range requiredFields {
		if !strings.Contains(body, field) {
			t.Errorf("ICS output missing required field: %s", field)
		}
	}

	// Check for all-day event format
	if !strings.Contains(body, "DTSTART;VALUE=DATE:20250115") {
		t.Error("Event should be all-day (DTSTART;VALUE=DATE)")
	}
	if !strings.Contains(body, "DTEND;VALUE=DATE:20250116") {
		t.Error("All-day event should end on next day")
	}

	// Check for event descriptions
	if !strings.Contains(body, "SUMMARY:Restmüll") {
		t.Error("Missing event summary for Restmüll")
	}
	if !strings.Contains(body, "SUMMARY:Biotonne") {
		t.Error("Missing event summary for Biotonne")
	}

	// Check for alarms
	alarmCount := strings.Count(body, "BEGIN:VALARM")
	// Each event should have 3 alarms (2 days, 1 day, same day)
	expectedAlarms := 6 // 2 events × 3 reminders
	if alarmCount != expectedAlarms {
		t.Errorf("Expected %d alarms, got %d", expectedAlarms, alarmCount)
	}

	// Verify alarm structure
	if !strings.Contains(body, "ACTION:DISPLAY") {
		t.Error("Alarm missing ACTION:DISPLAY")
	}
	if !strings.Contains(body, "TRIGGER:-P") {
		t.Error("Alarm missing TRIGGER with negative duration")
	}
}

func TestAddAlarm(t *testing.T) {
	tests := []struct {
		name        string
		eventDate   time.Time
		daysBefore  int
		alarmTime   string
		description string
		wantTrigger string
	}{
		{
			name:        "2 days before at 18:00",
			eventDate:   time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC),
			daysBefore:  2,
			alarmTime:   "18:00",
			description: "Restmüll",
			wantTrigger: "-P1DT6H0M", // 1 day + 6 hours before (event is at 00:00, alarm at 18:00 2 days before)
		},
		{
			name:        "1 day before at 19:00",
			eventDate:   time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC),
			daysBefore:  1,
			alarmTime:   "19:00",
			description: "Biotonne",
			wantTrigger: "-P0DT5H0M", // 5 hours before (event at 00:00, alarm at 19:00 day before)
		},
		{
			name:        "Same day at 07:00",
			eventDate:   time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC),
			daysBefore:  0,
			alarmTime:   "07:00",
			description: "Papiertonne",
			wantTrigger: "P0DT7H0M", // 7 hours after (event at 00:00, alarm at 07:00 same day)
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			AddAlarm(&buf, tt.eventDate, tt.daysBefore, tt.alarmTime, tt.description)

			output := buf.String()

			// Check for alarm structure
			if !strings.Contains(output, "BEGIN:VALARM") {
				t.Error("Missing BEGIN:VALARM")
			}
			if !strings.Contains(output, "END:VALARM") {
				t.Error("Missing END:VALARM")
			}
			if !strings.Contains(output, "ACTION:DISPLAY") {
				t.Error("Missing ACTION:DISPLAY")
			}
			if !strings.Contains(output, "TRIGGER:"+tt.wantTrigger) {
				t.Errorf("Expected TRIGGER:%s, got output:\n%s", tt.wantTrigger, output)
			}
			if !strings.Contains(output, tt.description) {
				t.Errorf("Missing description: %s", tt.description)
			}
		})
	}
}

func TestGenerateCSV(t *testing.T) {
	events := []Event{
		{Date: "2025-01-15", Type: "restmuell", Description: "Restmüll"},
		{Date: "2025-01-20", Type: "biotonne", Description: "Biotonne"},
	}

	w := httptest.NewRecorder()
	GenerateCSV(w, "Winterberg", 2025, events)

	resp := w.Result()
	body := w.Body.String()

	// Check content type
	contentType := resp.Header.Get("Content-Type")
	if !strings.Contains(contentType, "text/csv") {
		t.Errorf("Expected Content-Type text/csv, got %s", contentType)
	}

	// Check CSV header
	if !strings.Contains(body, "Datum,Abfalltyp,Beschreibung") {
		t.Error("Missing CSV header")
	}

	// Check CSV rows
	if !strings.Contains(body, "2025-01-15,restmuell,Restmüll") {
		t.Error("Missing first event in CSV")
	}
	if !strings.Contains(body, "2025-01-20,biotonne,Biotonne") {
		t.Error("Missing second event in CSV")
	}
}

func TestGenerateJSON(t *testing.T) {
	events := []Event{
		{Date: "2025-01-15", Type: "restmuell", Description: "Restmüll"},
	}

	w := httptest.NewRecorder()
	GenerateJSON(w, "Winterberg", 2025, events)

	resp := w.Result()
	body := w.Body.String()

	// Check content type
	contentType := resp.Header.Get("Content-Type")
	if !strings.Contains(contentType, "application/json") {
		t.Errorf("Expected Content-Type application/json, got %s", contentType)
	}

	// Check JSON structure
	if !strings.Contains(body, `"district":"Winterberg"`) {
		t.Error("Missing district in JSON")
	}
	if !strings.Contains(body, `"year":2025`) {
		t.Error("Missing year in JSON")
	}
	if !strings.Contains(body, `"events"`) {
		t.Error("Missing events in JSON")
	}
}
