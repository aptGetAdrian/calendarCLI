package calendar

import (
	"calendarCli/internal/logger"
	"context"
	"os"

	"golang.org/x/oauth2/google"
	gcalendar "google.golang.org/api/calendar/v3"
	"google.golang.org/api/option"
)

type Service struct {
	client *gcalendar.Service
}

func NewService(logger *logger.Logger) (*Service, error) {
	ctx := context.Background()

	b, err := os.ReadFile("credentials.json")
	if err != nil {
		return nil, err
	}

	config, err := google.ConfigFromJSON(b, gcalendar.CalendarScope)
	if err != nil {
		return nil, err
	}

	client, err := getClient(config, logger)
	if err != nil {
		return nil, err
	}

	srv, err := gcalendar.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, err
	}

	return &Service{client: srv}, nil
}

func (s *Service) Insert(calendarId string, event *gcalendar.Event) (*gcalendar.Event, error) {
	return s.client.Events.Insert(calendarId, event).Do()
}

func (s *Service) GetNumCalendars() (int, error) {
	// we check for the user calendars
	calendarList, err := s.client.CalendarList.List().Do()
	if err != nil {
		return 0, err
	}
	count := len(calendarList.Items)

	// we check if "birthdays" exist (they do for most users)
	birthdayEvents, err := s.client.Events.List("primary").
		EventTypes("birthday").
		MaxResults(1).
		Do()

	if err != nil {
		return count, err
	}

	if len(birthdayEvents.Items) > 0 {
		count++
	}

	// TODO
	// the "Tasks" "calendar "is a whole separate API
	// would need to check google Tasks API

	return count, nil
}

func (s *Service) GetPrimaryCalendar() (string, error) {
	calendar, err := s.client.CalendarList.Get("primary").Do()

	if err != nil {
		return "", err
	}

	// jsonData, err := json.MarshalIndent(calendar, "", "  ")
	// if err != nil {
	// 	fmt.Println("Error marshaling JSON:", err)
	// } else {
	// 	fmt.Println(string(jsonData))
	// }

	return calendar.Summary, nil
}

func (s *Service) SelectCalendar(name string) (string, error) {
	calendar, err := s.client.CalendarList.Get(name).Do()

	if err != nil {
		return "", err
	}

	return calendar.Summary, nil
}

func (s *Service) GetAllCalendars() (*gcalendar.CalendarList, error) {
	calendars, err := s.client.CalendarList.List().Do()

	if err != nil {
		return nil, err
	}

	return calendars, nil
}

func (s *Service) ExistBirthday() (bool, error) {
	birthdayEvents, err := s.client.Events.List("primary").
		EventTypes("birthday").
		MaxResults(1).
		Do()

	if err != nil {
		return false, err
	}

	if len(birthdayEvents.Items) > 0 {
		return true, nil
	}

	return false, nil

}
