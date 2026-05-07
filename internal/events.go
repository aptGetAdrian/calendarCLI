package calendar

import (
	"fmt"
	"strings"
	"time"

	gcalendar "google.golang.org/api/calendar/v3"
)

func (s *Service) ListUpcoming(max int) ([]*gcalendar.Event, error) {
	t := time.Now().Format(time.RFC3339)

	events, err := s.client.Events.List("primary").
		ShowDeleted(false).
		SingleEvents(true).
		TimeMin(t).
		MaxResults(int64(max)).
		OrderBy("startTime").
		Do()

	if err != nil {
		return nil, err
	}

	return events.Items, nil
}

func (s *Service) ListEventsForCalendarInRange(calID string, start, end time.Time) ([]*gcalendar.Event, error) {
	events, err := s.client.Events.List(calID).
		ShowDeleted(false).
		SingleEvents(true).
		TimeMin(start.Format(time.RFC3339)).
		TimeMax(end.Format(time.RFC3339)).
		OrderBy("startTime").
		Do()
	if err != nil {
		return nil, err
	}
	return events.Items, nil
}

const birthdayTag = "calendarCLI:birthday"

func (s *Service) ListBirthdayEventsInRange(start, end time.Time) ([]*gcalendar.Event, error) {
	tMin := start.Format(time.RFC3339)
	tMax := end.Format(time.RFC3339)

	// source 1: Google Contacts birthdays
	contactResp, err := s.client.Events.List("primary").
		ShowDeleted(false).
		SingleEvents(true).
		EventTypes("birthday").
		TimeMin(tMin).
		TimeMax(tMax).
		OrderBy("startTime").
		Do()
	if err != nil {
		return nil, err
	}

	// source 2: manually added birthdays (tagged in description)
	allResp, err := s.client.Events.List("primary").
		ShowDeleted(false).
		SingleEvents(true).
		TimeMin(tMin).
		TimeMax(tMax).
		OrderBy("startTime").
		Do()
	if err != nil {
		return nil, err
	}

	merged := contactResp.Items
	for _, e := range allResp.Items {
		if strings.Contains(e.Description, birthdayTag) {
			merged = append(merged, e)
		}
	}
	return merged, nil
}

func (s *Service) CreateBirthdayEvent(firstName, lastName string, day, month int) error {
	now := time.Now()
	year := now.Year()

	// if the birthday has already passed this year, start next year
	target := time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.Local)
	if !target.After(now) {
		year++
	}

	name := firstName
	if lastName != "" {
		name = firstName + " " + lastName
	}

	start := time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC)
	end := start.AddDate(0, 0, 1)

	event := &gcalendar.Event{
		Summary:     name + "'s Birthday",
		Description: birthdayTag,
		Start:       &gcalendar.EventDateTime{Date: fmt.Sprintf("%04d-%02d-%02d", start.Year(), start.Month(), start.Day())},
		End:         &gcalendar.EventDateTime{Date: fmt.Sprintf("%04d-%02d-%02d", end.Year(), end.Month(), end.Day())},
		Recurrence:  []string{"RRULE:FREQ=YEARLY;COUNT=15"},
	}

	_, err := s.Insert("primary", event)
	return err
}

func (s *Service) CreateEvent(
	calendarID, title, location, description string,
	start, end time.Time) (*gcalendar.Event, error) {
	event := &gcalendar.Event{
		Summary:     title,
		Location:    location,
		Description: description,
		Start: &gcalendar.EventDateTime{
			DateTime: start.Format(time.RFC3339),
		},
		End: &gcalendar.EventDateTime{
			DateTime: end.Format(time.RFC3339),
		},
	}
	return s.Insert(calendarID, event)
}
