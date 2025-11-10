package app

import (
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGenerateSubscriptionICS(t *testing.T) {
	// Setup test data
	events := []Event{
		{Date: "2025-01-15", Type: "restmuell", Description: "Restmüll"},
		{Date: "2025-01-20", Type: "biotonne", Description: "Biotonne"},
	}

	// Create test request (subscriptions don't use reminder params)
	req := httptest.NewRequest("GET", "/api/subscribe/Winterberg", nil)
	w := httptest.NewRecorder()

	// Call GenerateSubscriptionICS
	GenerateSubscriptionICS(w, req, "Winterberg", events)

	// Get response
	resp := w.Result()
	body := w.Body.String()

	// Check status code
	if resp.StatusCode != 200 {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	// Check content type (should be text/calendar for subscription)
	contentType := resp.Header.Get("Content-Type")
	if !strings.Contains(contentType, "text/calendar") {
		t.Errorf("Expected Content-Type text/calendar, got %s", contentType)
	}

	// IMPORTANT: Subscription should NOT have Content-Disposition attachment header
	contentDisposition := resp.Header.Get("Content-Disposition")
	if contentDisposition != "" {
		t.Errorf("Subscription should not have Content-Disposition header, got: %s", contentDisposition)
	}

	// Check for required ICS structure
	requiredFields := []string{
		"BEGIN:VCALENDAR",
		"VERSION:2.0",
		"PRODID:-//Winterberg//Abfallkalender//DE",
		"METHOD:PUBLISH",
		"X-PUBLISHED-TTL:PT1H", // Refresh every hour
		"BEGIN:VEVENT",
		"END:VEVENT",
		"END:VCALENDAR",
	}

	for _, field := range requiredFields {
		if !strings.Contains(body, field) {
			t.Errorf("ICS subscription output missing required field: %s", field)
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

	// IMPORTANT: Subscriptions should NOT contain VALARM blocks
	// Most calendar apps ignore alarms in subscriptions for security reasons
	alarmCount := strings.Count(body, "BEGIN:VALARM")
	if alarmCount != 0 {
		t.Errorf("Subscription should not contain alarms (found %d VALARM blocks)", alarmCount)
	}

	// Verify UID format for proper updates
	if !strings.Contains(body, "UID:2025-01-15-restmuell-Winterberg@abfallkalender.winterberg.de") {
		t.Error("Missing or incorrect UID format")
	}
}

func TestGenerateSubscriptionICS_EmptyEvents(t *testing.T) {
	// Test with no events
	events := []Event{}

	req := httptest.NewRequest("GET", "/api/subscribe/Winterberg", nil)
	w := httptest.NewRecorder()

	GenerateSubscriptionICS(w, req, "Winterberg", events)

	resp := w.Result()
	body := w.Body.String()

	// Should still generate valid ICS
	if resp.StatusCode != 200 {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	// Should have calendar structure even with no events
	if !strings.Contains(body, "BEGIN:VCALENDAR") {
		t.Error("Missing BEGIN:VCALENDAR")
	}
	if !strings.Contains(body, "END:VCALENDAR") {
		t.Error("Missing END:VCALENDAR")
	}

	// Should not have any events
	eventCount := strings.Count(body, "BEGIN:VEVENT")
	if eventCount != 0 {
		t.Errorf("Expected 0 events, got %d", eventCount)
	}
}

func TestGenerateSubscriptionICS_MultipleEventsOnSameDay(t *testing.T) {
	// Test with multiple waste types on the same day
	events := []Event{
		{Date: "2025-01-15", Type: "restmuell", Description: "Restmüll"},
		{Date: "2025-01-15", Type: "biotonne", Description: "Biotonne"},
		{Date: "2025-01-15", Type: "gelber_sack", Description: "Gelber Sack"},
	}

	req := httptest.NewRequest("GET", "/api/subscribe/Winterberg", nil)
	w := httptest.NewRecorder()

	GenerateSubscriptionICS(w, req, "Winterberg", events)

	body := w.Body.String()

	// Should have 3 separate events
	eventCount := strings.Count(body, "BEGIN:VEVENT")
	if eventCount != 3 {
		t.Errorf("Expected 3 events, got %d", eventCount)
	}

	// Each should have unique UIDs
	if !strings.Contains(body, "UID:2025-01-15-restmuell-Winterberg@abfallkalender.winterberg.de") {
		t.Error("Missing UID for Restmüll")
	}
	if !strings.Contains(body, "UID:2025-01-15-biotonne-Winterberg@abfallkalender.winterberg.de") {
		t.Error("Missing UID for Biotonne")
	}
	if !strings.Contains(body, "UID:2025-01-15-gelber_sack-Winterberg@abfallkalender.winterberg.de") {
		t.Error("Missing UID for Gelber Sack")
	}
}

func TestGenerateSubscriptionICS_Headers(t *testing.T) {
	// Test that subscription-specific headers are set correctly
	events := []Event{
		{Date: "2025-01-15", Type: "restmuell", Description: "Restmüll"},
	}

	req := httptest.NewRequest("GET", "/api/subscribe/Winterberg", nil)
	w := httptest.NewRecorder()

	GenerateSubscriptionICS(w, req, "Winterberg", events)

	resp := w.Result()
	body := w.Body.String()

	// Check for METHOD:PUBLISH (required for subscriptions)
	if !strings.Contains(body, "METHOD:PUBLISH") {
		t.Error("Subscription should contain METHOD:PUBLISH")
	}

	// Check for refresh interval
	if !strings.Contains(body, "X-PUBLISHED-TTL:PT1H") {
		t.Error("Subscription should contain X-PUBLISHED-TTL")
	}

	// Check for calendar name (no year in subscription)
	if !strings.Contains(body, "X-WR-CALNAME:Abfallkalender Winterberg") {
		t.Error("Missing calendar name")
	}

	// Verify Content-Type header
	contentType := resp.Header.Get("Content-Type")
	if !strings.Contains(contentType, "text/calendar") {
		t.Errorf("Expected text/calendar content type, got %s", contentType)
	}

	// Verify charset
	if !strings.Contains(contentType, "charset=utf-8") {
		t.Error("Content-Type should include charset=utf-8")
	}
}

func TestGenerateSubscriptionICS_InvalidDate(t *testing.T) {
	// Test with invalid date format (should be skipped)
	events := []Event{
		{Date: "invalid-date", Type: "restmuell", Description: "Restmüll"},
		{Date: "2025-01-15", Type: "biotonne", Description: "Biotonne"},
	}

	req := httptest.NewRequest("GET", "/api/subscribe/Winterberg", nil)
	w := httptest.NewRecorder()

	GenerateSubscriptionICS(w, req, "Winterberg", events)

	body := w.Body.String()

	// Should only have 1 valid event (invalid one should be skipped)
	eventCount := strings.Count(body, "BEGIN:VEVENT")
	if eventCount != 1 {
		t.Errorf("Expected 1 valid event, got %d", eventCount)
	}

	// Should have the valid event
	if !strings.Contains(body, "SUMMARY:Biotonne") {
		t.Error("Missing valid event")
	}

	// Should not have the invalid event
	if strings.Contains(body, "SUMMARY:Restmüll") {
		t.Error("Invalid event should be skipped")
	}
}
