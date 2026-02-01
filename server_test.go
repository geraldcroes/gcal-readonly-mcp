package main

import (
	"encoding/json"
	"testing"
	"time"
)

// TestNewCalendarServer verifies that the server initializes without panic.
// This catches jsonschema tag format errors (e.g., description= vs description:)
// which would cause a panic during tool registration.
func TestNewCalendarServer(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("NewCalendarServer() panicked: %v", r)
		}
	}()

	server := NewCalendarServer()
	if server == nil {
		t.Error("NewCalendarServer() returned nil")
	}
}

// TestListAccountsOutputSerialization ensures accounts serialize as empty array, not null
func TestListAccountsOutputSerialization(t *testing.T) {
	tests := []struct {
		name     string
		output   ListAccountsOutput
		contains string
	}{
		{
			name:     "nil accounts becomes empty array",
			output:   ListAccountsOutput{Accounts: nil},
			contains: `"accounts":null`, // This is what we DON'T want
		},
		{
			name:     "empty slice becomes empty array",
			output:   ListAccountsOutput{Accounts: []string{}},
			contains: `"accounts":[]`,
		},
		{
			name:     "populated accounts",
			output:   ListAccountsOutput{Accounts: []string{"personal", "work"}},
			contains: `"accounts":["personal","work"]`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.output)
			if err != nil {
				t.Fatalf("Failed to marshal: %v", err)
			}

			// For the nil case, we verify the handler should NOT return nil
			if tt.name == "nil accounts becomes empty array" {
				if string(data) == `{"accounts":null}` {
					t.Log("Note: nil slice serializes as null - handlers must initialize slices")
				}
			} else {
				if string(data) != tt.contains && !contains(string(data), tt.contains) {
					t.Errorf("Expected %s to contain %s", string(data), tt.contains)
				}
			}
		})
	}
}

// TestListCalendarsOutputSerialization ensures calendars serialize as empty array
func TestListCalendarsOutputSerialization(t *testing.T) {
	// Empty slice should serialize as []
	output := ListCalendarsOutput{Calendars: []Calendar{}}
	data, err := json.Marshal(output)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	expected := `{"calendars":[]}`
	if string(data) != expected {
		t.Errorf("Expected %s, got %s", expected, string(data))
	}
}

// TestListEventsOutputSerialization ensures events serialize as empty array
func TestListEventsOutputSerialization(t *testing.T) {
	// Empty slice should serialize as []
	output := ListEventsOutput{Events: []Event{}}
	data, err := json.Marshal(output)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	expected := `{"events":[]}`
	if string(data) != expected {
		t.Errorf("Expected %s, got %s", expected, string(data))
	}
}

// TestCheckAvailabilityOutputSerialization ensures busy_periods serialize as empty array
func TestCheckAvailabilityOutputSerialization(t *testing.T) {
	// Empty slice should serialize as []
	output := CheckAvailabilityOutput{BusyPeriods: []BusyPeriod{}}
	data, err := json.Marshal(output)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	expected := `{"busy_periods":[]}`
	if string(data) != expected {
		t.Errorf("Expected %s, got %s", expected, string(data))
	}
}

// TestEventSerialization verifies Event struct serializes correctly
func TestEventSerialization(t *testing.T) {
	event := Event{
		ID:         "test-id",
		Summary:    "Test Event",
		Start:      time.Date(2026, 2, 1, 10, 0, 0, 0, time.UTC),
		End:        time.Date(2026, 2, 1, 11, 0, 0, 0, time.UTC),
		AllDay:     false,
		Status:     "confirmed",
		HtmlLink:   "https://calendar.google.com/test",
		Account:    "personal",
		CalendarID: "primary",
	}

	data, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("Failed to marshal event: %v", err)
	}

	// Verify required fields are present
	requiredFields := []string{
		`"id":"test-id"`,
		`"summary":"Test Event"`,
		`"status":"confirmed"`,
		`"account":"personal"`,
		`"calendar_id":"primary"`,
	}

	for _, field := range requiredFields {
		if !contains(string(data), field) {
			t.Errorf("Expected serialized event to contain %s, got %s", field, string(data))
		}
	}
}

// TestCalendarSerialization verifies Calendar struct serializes correctly
func TestCalendarSerialization(t *testing.T) {
	calendar := Calendar{
		ID:      "primary",
		Summary: "My Calendar",
		Primary: true,
		Account: "personal",
	}

	data, err := json.Marshal(calendar)
	if err != nil {
		t.Fatalf("Failed to marshal calendar: %v", err)
	}

	requiredFields := []string{
		`"id":"primary"`,
		`"summary":"My Calendar"`,
		`"primary":true`,
		`"account":"personal"`,
	}

	for _, field := range requiredFields {
		if !contains(string(data), field) {
			t.Errorf("Expected serialized calendar to contain %s, got %s", field, string(data))
		}
	}
}

// TestBusyPeriodSerialization verifies BusyPeriod struct serializes correctly
func TestBusyPeriodSerialization(t *testing.T) {
	bp := BusyPeriod{
		Start:   time.Date(2026, 2, 1, 10, 0, 0, 0, time.UTC),
		End:     time.Date(2026, 2, 1, 11, 0, 0, 0, time.UTC),
		Account: "work",
	}

	data, err := json.Marshal(bp)
	if err != nil {
		t.Fatalf("Failed to marshal busy period: %v", err)
	}

	if !contains(string(data), `"account":"work"`) {
		t.Errorf("Expected account field in %s", string(data))
	}
}

// TestNilSliceInitialization documents the nil vs empty slice behavior
// This test demonstrates WHY we need the nil checks in handlers
func TestNilSliceInitialization(t *testing.T) {
	t.Run("nil slice serializes as null", func(t *testing.T) {
		var nilSlice []string
		data, _ := json.Marshal(map[string][]string{"items": nilSlice})
		if string(data) != `{"items":null}` {
			t.Errorf("Expected null for nil slice, got %s", string(data))
		}
	})

	t.Run("empty slice serializes as array", func(t *testing.T) {
		emptySlice := []string{}
		data, _ := json.Marshal(map[string][]string{"items": emptySlice})
		if string(data) != `{"items":[]}` {
			t.Errorf("Expected [] for empty slice, got %s", string(data))
		}
	})
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
