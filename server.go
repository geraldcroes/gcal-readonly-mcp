package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Tool input/output structures

type ListAccountsInput struct{}

type ListAccountsOutput struct {
	Accounts []string `json:"accounts"`
}

type ListCalendarsInput struct {
	Account string `json:"account,omitempty" jsonschema:"description:Account name (optional - if empty lists from all accounts)"`
}

type Calendar struct {
	ID          string `json:"id"`
	Summary     string `json:"summary"`
	Description string `json:"description,omitempty"`
	Primary     bool   `json:"primary,omitempty"`
	Account     string `json:"account"`
}

type ListCalendarsOutput struct {
	Calendars []Calendar `json:"calendars"`
}

type ListEventsInput struct {
	Account    string `json:"account,omitempty" jsonschema:"description:Account name (optional - if empty queries all accounts)"`
	CalendarID string `json:"calendar_id,omitempty" jsonschema:"description:Calendar ID (optional - if empty uses primary calendar)"`
	TimeMin    string `json:"time_min,omitempty" jsonschema:"description:Start of time range (RFC3339 format). Defaults to now."`
	TimeMax    string `json:"time_max,omitempty" jsonschema:"description:End of time range (RFC3339 format). Defaults to 7 days from now."`
	MaxResults int    `json:"max_results,omitempty" jsonschema:"description:Maximum number of events to return (default 50 max 250)"`
	Query      string `json:"query,omitempty" jsonschema:"description:Free text search query"`
}

type Event struct {
	ID          string    `json:"id"`
	Summary     string    `json:"summary"`
	Description string    `json:"description,omitempty"`
	Location    string    `json:"location,omitempty"`
	Start       time.Time `json:"start"`
	End         time.Time `json:"end"`
	AllDay      bool      `json:"all_day"`
	Attendees   []string  `json:"attendees,omitempty"`
	Organizer   string    `json:"organizer,omitempty"`
	Status      string    `json:"status"`
	HtmlLink    string    `json:"html_link"`
	Account     string    `json:"account"`
	CalendarID  string    `json:"calendar_id"`
}

type ListEventsOutput struct {
	Events []Event `json:"events"`
}

type GetEventInput struct {
	Account    string `json:"account" jsonschema:"description:Account name,required"`
	CalendarID string `json:"calendar_id" jsonschema:"description:Calendar ID,required"`
	EventID    string `json:"event_id" jsonschema:"description:Event ID,required"`
}

type GetEventOutput struct {
	Event Event `json:"event"`
}

type CheckAvailabilityInput struct {
	Account    string   `json:"account,omitempty" jsonschema:"description:Account name (optional - if empty checks all accounts)"`
	Calendars  []string `json:"calendars,omitempty" jsonschema:"description:List of calendar IDs to check (optional - if empty uses primary)"`
	TimeMin    string   `json:"time_min" jsonschema:"description:Start of time range (RFC3339 format),required"`
	TimeMax    string   `json:"time_max" jsonschema:"description:End of time range (RFC3339 format),required"`
}

type BusyPeriod struct {
	Start   time.Time `json:"start"`
	End     time.Time `json:"end"`
	Account string    `json:"account"`
}

type CheckAvailabilityOutput struct {
	BusyPeriods []BusyPeriod `json:"busy_periods"`
}

// NewCalendarServer creates and configures the MCP server with all tools
func NewCalendarServer() *mcp.Server {
	server := mcp.NewServer(&mcp.Implementation{
		Name:    ServerName,
		Version: ServerVersion,
	}, nil)

	// Register tools
	mcp.AddTool(server, &mcp.Tool{
		Name:        "list_accounts",
		Description: "List all configured Google accounts",
	}, handleListAccounts)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "list_calendars",
		Description: "List all calendars accessible by the configured accounts",
	}, handleListCalendars)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "list_events",
		Description: "List calendar events within a time range. Can filter by account, calendar, and search query.",
	}, handleListEvents)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_event",
		Description: "Get detailed information about a specific event",
	}, handleGetEvent)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "check_availability",
		Description: "Check free/busy status for specified calendars within a time range",
	}, handleCheckAvailability)

	return server
}

// Tool handlers

func handleListAccounts(ctx context.Context, req *mcp.CallToolRequest, input ListAccountsInput) (*mcp.CallToolResult, ListAccountsOutput, error) {
	accounts, err := ListConfiguredAccounts()
	if err != nil {
		return nil, ListAccountsOutput{}, fmt.Errorf("failed to list accounts: %w", err)
	}

	// Ensure we return an empty array, not null
	if accounts == nil {
		accounts = []string{}
	}

	output := ListAccountsOutput{Accounts: accounts}
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: fmt.Sprintf("Found %d configured account(s): %s", len(accounts), strings.Join(accounts, ", "))},
		},
	}, output, nil
}

func handleListCalendars(ctx context.Context, req *mcp.CallToolRequest, input ListCalendarsInput) (*mcp.CallToolResult, ListCalendarsOutput, error) {
	calendars, err := GetCalendars(ctx, input.Account)
	if err != nil {
		return nil, ListCalendarsOutput{}, fmt.Errorf("failed to list calendars: %w", err)
	}

	// Ensure we return an empty array, not null
	if calendars == nil {
		calendars = []Calendar{}
	}

	output := ListCalendarsOutput{Calendars: calendars}

	var lines []string
	for _, cal := range calendars {
		primary := ""
		if cal.Primary {
			primary = " (primary)"
		}
		lines = append(lines, fmt.Sprintf("- [%s] %s%s", cal.Account, cal.Summary, primary))
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: fmt.Sprintf("Found %d calendar(s):\n%s", len(calendars), strings.Join(lines, "\n"))},
		},
	}, output, nil
}

func handleListEvents(ctx context.Context, req *mcp.CallToolRequest, input ListEventsInput) (*mcp.CallToolResult, ListEventsOutput, error) {
	events, err := GetEvents(ctx, input)
	if err != nil {
		return nil, ListEventsOutput{}, fmt.Errorf("failed to list events: %w", err)
	}

	// Ensure we return an empty array, not null
	if events == nil {
		events = []Event{}
	}

	output := ListEventsOutput{Events: events}

	var lines []string
	for _, ev := range events {
		timeStr := ev.Start.Format("2006-01-02 15:04")
		if ev.AllDay {
			timeStr = ev.Start.Format("2006-01-02") + " (all day)"
		}
		lines = append(lines, fmt.Sprintf("- [%s] %s: %s", ev.Account, timeStr, ev.Summary))
	}

	text := fmt.Sprintf("Found %d event(s)", len(events))
	if len(lines) > 0 {
		text += ":\n" + strings.Join(lines, "\n")
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: text},
		},
	}, output, nil
}

func handleGetEvent(ctx context.Context, req *mcp.CallToolRequest, input GetEventInput) (*mcp.CallToolResult, GetEventOutput, error) {
	event, err := GetEvent(ctx, input.Account, input.CalendarID, input.EventID)
	if err != nil {
		return nil, GetEventOutput{}, fmt.Errorf("failed to get event: %w", err)
	}

	output := GetEventOutput{Event: *event}

	text := fmt.Sprintf("Event: %s\nWhen: %s - %s\nWhere: %s\nDescription: %s",
		event.Summary,
		event.Start.Format("2006-01-02 15:04"),
		event.End.Format("2006-01-02 15:04"),
		event.Location,
		event.Description,
	)

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: text},
		},
	}, output, nil
}

func handleCheckAvailability(ctx context.Context, req *mcp.CallToolRequest, input CheckAvailabilityInput) (*mcp.CallToolResult, CheckAvailabilityOutput, error) {
	busyPeriods, err := CheckAvailability(ctx, input)
	if err != nil {
		return nil, CheckAvailabilityOutput{}, fmt.Errorf("failed to check availability: %w", err)
	}

	// Ensure we return an empty array, not null
	if busyPeriods == nil {
		busyPeriods = []BusyPeriod{}
	}

	output := CheckAvailabilityOutput{BusyPeriods: busyPeriods}

	var lines []string
	for _, bp := range busyPeriods {
		lines = append(lines, fmt.Sprintf("- [%s] %s - %s",
			bp.Account,
			bp.Start.Format("2006-01-02 15:04"),
			bp.End.Format("2006-01-02 15:04"),
		))
	}

	text := fmt.Sprintf("Found %d busy period(s)", len(busyPeriods))
	if len(lines) > 0 {
		text += ":\n" + strings.Join(lines, "\n")
	} else {
		text += " - you're free during this time range!"
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: text},
		},
	}, output, nil
}
