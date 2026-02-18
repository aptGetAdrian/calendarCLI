package calendar

import (
	"context"
	"os"

	"golang.org/x/oauth2/google"
	gcalendar "google.golang.org/api/calendar/v3"
	"google.golang.org/api/option"
)

type Service struct {
	client *gcalendar.Service
}

func NewService() (*Service, error) {
	ctx := context.Background()

	b, err := os.ReadFile("credentials.json")
	if err != nil {
		return nil, err
	}

	config, err := google.ConfigFromJSON(b, gcalendar.CalendarScope)
	if err != nil {
		return nil, err
	}

	client, err := getClient(config)
	if err != nil {
		return nil, err
	}

	srv, err := gcalendar.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, err
	}

	return &Service{client: srv}, nil
}

func (s *Service) insert(event *gcalendar.Event) (*gcalendar.Event, error) {
	return s.client.Events.Insert("primary", event).Do()
}
