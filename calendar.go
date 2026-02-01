package main

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/api/calendar/v3"
)

// GetCalendars returns all calendars for the specified account (or all accounts if empty)
func GetCalendars(ctx context.Context, accountName string) ([]Calendar, error) {
	accounts, err := getTargetAccounts(accountName)
	if err != nil {
		return nil, err
	}

	var calendars []Calendar
	for _, acc := range accounts {
		srv, err := GetCalendarService(ctx, acc)
		if err != nil {
			return nil, fmt.Errorf("failed to get service for account '%s': %w", acc, err)
		}

		list, err := srv.CalendarList.List().Do()
		if err != nil {
			return nil, fmt.Errorf("failed to list calendars for account '%s': %w", acc, err)
		}

		for _, item := range list.Items {
			calendars = append(calendars, Calendar{
				ID:          item.Id,
				Summary:     item.Summary,
				Description: item.Description,
				Primary:     item.Primary,
				Account:     acc,
			})
		}
	}

	return calendars, nil
}

// GetEvents returns events matching the specified criteria
func GetEvents(ctx context.Context, input ListEventsInput) ([]Event, error) {
	accounts, err := getTargetAccounts(input.Account)
	if err != nil {
		return nil, err
	}

	// Parse time range
	timeMin := time.Now()
	if input.TimeMin != "" {
		t, err := time.Parse(time.RFC3339, input.TimeMin)
		if err != nil {
			return nil, fmt.Errorf("invalid time_min format: %w", err)
		}
		timeMin = t
	}

	timeMax := timeMin.AddDate(0, 0, 7) // Default: 7 days
	if input.TimeMax != "" {
		t, err := time.Parse(time.RFC3339, input.TimeMax)
		if err != nil {
			return nil, fmt.Errorf("invalid time_max format: %w", err)
		}
		timeMax = t
	}

	maxResults := input.MaxResults
	if maxResults <= 0 {
		maxResults = 50
	}
	if maxResults > 250 {
		maxResults = 250
	}

	var events []Event
	for _, acc := range accounts {
		srv, err := GetCalendarService(ctx, acc)
		if err != nil {
			return nil, fmt.Errorf("failed to get service for account '%s': %w", acc, err)
		}

		calendarID := input.CalendarID
		if calendarID == "" {
			calendarID = "primary"
		}

		call := srv.Events.List(calendarID).
			TimeMin(timeMin.Format(time.RFC3339)).
			TimeMax(timeMax.Format(time.RFC3339)).
			MaxResults(int64(maxResults)).
			SingleEvents(true).
			OrderBy("startTime")

		if input.Query != "" {
			call = call.Q(input.Query)
		}

		result, err := call.Do()
		if err != nil {
			return nil, fmt.Errorf("failed to list events for account '%s': %w", acc, err)
		}

		for _, item := range result.Items {
			events = append(events, parseEvent(item, acc, calendarID))
		}
	}

	return events, nil
}

// GetEvent returns details for a specific event
func GetEvent(ctx context.Context, accountName, calendarID, eventID string) (*Event, error) {
	srv, err := GetCalendarService(ctx, accountName)
	if err != nil {
		return nil, err
	}

	item, err := srv.Events.Get(calendarID, eventID).Do()
	if err != nil {
		return nil, fmt.Errorf("failed to get event: %w", err)
	}

	event := parseEvent(item, accountName, calendarID)
	return &event, nil
}

// CheckAvailability returns busy periods for the specified calendars
func CheckAvailability(ctx context.Context, input CheckAvailabilityInput) ([]BusyPeriod, error) {
	accounts, err := getTargetAccounts(input.Account)
	if err != nil {
		return nil, err
	}

	timeMin, err := time.Parse(time.RFC3339, input.TimeMin)
	if err != nil {
		return nil, fmt.Errorf("invalid time_min format: %w", err)
	}

	timeMax, err := time.Parse(time.RFC3339, input.TimeMax)
	if err != nil {
		return nil, fmt.Errorf("invalid time_max format: %w", err)
	}

	var busyPeriods []BusyPeriod
	for _, acc := range accounts {
		srv, err := GetCalendarService(ctx, acc)
		if err != nil {
			return nil, fmt.Errorf("failed to get service for account '%s': %w", acc, err)
		}

		calendars := input.Calendars
		if len(calendars) == 0 {
			calendars = []string{"primary"}
		}

		var items []*calendar.FreeBusyRequestItem
		for _, calID := range calendars {
			items = append(items, &calendar.FreeBusyRequestItem{Id: calID})
		}

		req := &calendar.FreeBusyRequest{
			TimeMin: timeMin.Format(time.RFC3339),
			TimeMax: timeMax.Format(time.RFC3339),
			Items:   items,
		}

		result, err := srv.Freebusy.Query(req).Do()
		if err != nil {
			return nil, fmt.Errorf("failed to query free/busy for account '%s': %w", acc, err)
		}

		for _, cal := range result.Calendars {
			for _, busy := range cal.Busy {
				start, _ := time.Parse(time.RFC3339, busy.Start)
				end, _ := time.Parse(time.RFC3339, busy.End)
				busyPeriods = append(busyPeriods, BusyPeriod{
					Start:   start,
					End:     end,
					Account: acc,
				})
			}
		}
	}

	return busyPeriods, nil
}

// Helper functions

func getTargetAccounts(accountName string) ([]string, error) {
	if accountName != "" {
		return []string{accountName}, nil
	}
	return ListConfiguredAccounts()
}

func parseEvent(item *calendar.Event, account, calendarID string) Event {
	var start, end time.Time
	allDay := false

	if item.Start != nil {
		if item.Start.DateTime != "" {
			start, _ = time.Parse(time.RFC3339, item.Start.DateTime)
		} else if item.Start.Date != "" {
			start, _ = time.Parse("2006-01-02", item.Start.Date)
			allDay = true
		}
	}

	if item.End != nil {
		if item.End.DateTime != "" {
			end, _ = time.Parse(time.RFC3339, item.End.DateTime)
		} else if item.End.Date != "" {
			end, _ = time.Parse("2006-01-02", item.End.Date)
		}
	}

	var attendees []string
	for _, att := range item.Attendees {
		attendees = append(attendees, att.Email)
	}

	var organizer string
	if item.Organizer != nil {
		organizer = item.Organizer.Email
	}

	return Event{
		ID:          item.Id,
		Summary:     item.Summary,
		Description: item.Description,
		Location:    item.Location,
		Start:       start,
		End:         end,
		AllDay:      allDay,
		Attendees:   attendees,
		Organizer:   organizer,
		Status:      item.Status,
		HtmlLink:    item.HtmlLink,
		Account:     account,
		CalendarID:  calendarID,
	}
}
