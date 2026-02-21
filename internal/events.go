package calendar

import (
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

func (s *Service) CreateEvent(
	title, location, description string,
	start, end time.Time,
) (*gcalendar.Event, error) {

	event := &gcalendar.Event{
		Summary:     title,
		Location:    location,
		Description: description,
		Start: &gcalendar.EventDateTime{
			DateTime: start.Format(time.RFC3339),
			TimeZone: start.Location().String(),
		},
		End: &gcalendar.EventDateTime{
			DateTime: end.Format(time.RFC3339),
			TimeZone: end.Location().String(),
		},
	}

	return s.Insert(event)
}
