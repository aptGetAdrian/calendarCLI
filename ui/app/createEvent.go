package app

import (
	calendar "calendarCli/internal"
	"fmt"
	"time"

	"calendarCli/ui"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
)

type createEventModel struct {
	service *calendar.Service
	form    *huh.Form
	state   AppState

	// form field values
	calendarID  string
	title       string
	location    string
	description string
	startStr    string
	endStr      string
}

func newCreateEventModel(service *calendar.Service, state AppState, width, height int) *createEventModel {
	// build calendar options from state or fetch live
	calendars, err := service.GetAllCalendars()
	calendarOptions := []huh.Option[string]{}
	if err == nil {
		for _, cal := range calendars.Items {
			calendarOptions = append(calendarOptions, huh.NewOption(cal.Summary, cal.Id))
		}
	}
	// pre-select the already chosen calendar if there is one
	defaultCalendar := state.SelectedCalendar

	m := &createEventModel{
		service:    service,
		state:      state,
		calendarID: defaultCalendar,
	}

	m.form = huh.NewForm(
		// Group 1: which calendar
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Calendar").
				Description("Which calendar should this event be added to?").
				Options(calendarOptions...).
				Value(&m.calendarID),
		),

		// Group 2: event details
		huh.NewGroup(
			huh.NewInput().
				Title("Title").
				Placeholder("Team standup").
				Value(&m.title).
				Validate(func(s string) error {
					if s == "" {
						return fmt.Errorf("title is required")
					}
					return nil
				}),

			huh.NewInput().
				Title("Location").
				Placeholder("Conference room / Google Meet link").
				Value(&m.location),

			huh.NewText().
				Title("Description").
				Placeholder("What's this event about?").
				CharLimit(500).
				Value(&m.description),
		),

		// Group 3: start & end times
		huh.NewGroup(
			huh.NewInput().
				Title("Start").
				Placeholder("2026-03-15 14:00").
				Value(&m.startStr).
				Validate(func(s string) error {
					_, err := time.ParseInLocation("2006-01-02 15:04", s, time.Local)
					if err != nil {
						return fmt.Errorf("use format YYYY-MM-DD HH:MM")
					}
					return nil
				}),

			huh.NewInput().
				Title("End").
				Placeholder("2026-03-15 15:00").
				Value(&m.endStr).
				Validate(func(s string) error {
					_, err := time.ParseInLocation("2006-01-02 15:04", s, time.Local)
					if err != nil {
						return fmt.Errorf("use format YYYY-MM-DD HH:MM")
					}
					return nil
				}),
		),
	).WithWidth(width)

	return m
}

func (m *createEventModel) Init() tea.Cmd {
	return m.form.Init()
}

func (m *createEventModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case sizedMsg:
		m.form = m.form.WithWidth(msg.width)
		return m, nil

	case tea.KeyMsg:
		if msg.String() == "q" || msg.String() == "esc" {
			// go back without saving
			return m, func() tea.Msg { return NavigateTo{Screen: ui.MainMenuScreen} }
		}
	}

	// delegate to huh
	form, cmd := m.form.Update(msg)
	if f, ok := form.(*huh.Form); ok {
		m.form = f
	}

	// form completed — submit the event
	if m.form.State == huh.StateCompleted {
		return m, m.submitEvent()
	}

	return m, cmd
}

func (m *createEventModel) View() string {
	if m.form.State == huh.StateCompleted {
		return "Creating event..."
	}
	return m.form.View()
}

func (m *createEventModel) submitEvent() tea.Cmd {
	return func() tea.Msg {
		start, _ := time.ParseInLocation("2006-01-02 15:04", m.startStr, time.Local)
		end, _ := time.ParseInLocation("2006-01-02 15:04", m.endStr, time.Local)

		// Get the real IANA timezone name
		zone, offset := start.Zone()
		// If Local resolved to a real name, use it; otherwise fall back to a fixed offset zone
		if zone == "Local" || zone == "" {
			// construct a fixed offset like "Etc/GMT+1"
			offsetHours := -offset / 3600
			zone = fmt.Sprintf("Etc/GMT%+d", offsetHours)
		}

		_, err := m.service.CreateEvent(m.calendarID, m.title, m.location, m.description, start, end, zone)
		if err != nil {
			fmt.Printf("Error creating event: %s\n", err)
			return NavigateTo{Screen: ui.MainMenuScreen}
		}

		return eventCreatedMsg{}
	}
}
