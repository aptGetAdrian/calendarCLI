package app

import (
	calendar "calendarCli/internal"
	"calendarCli/ui"
	"calendarCli/ui/styles"
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
)

type createEventModel struct {
	service *calendar.Service
	form    *huh.Form
	state   AppState
	width   int
	height  int

	submitted bool

	calendarID  string
	title       string
	location    string
	description string
	startStr    string
	endStr      string
}

var (
	createEventTitleStyle = lipgloss.NewStyle().
				BorderStyle(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color(styles.ColorSecondaryBorder)).
				Padding(0, 2).
				Bold(true).
				Foreground(lipgloss.Color(styles.ColorSecondaryFg)).
				Background(lipgloss.Color(styles.ColorSecondaryBg))

	createEventTheme        = huh.ThemeBase()
	createEventWrapperStyle = lipgloss.NewStyle().PaddingTop(1).PaddingLeft(2)
)

func init() {
	// Define theme first, before using it in the form
	t := huh.ThemeBase()

	fg := lipgloss.Color(styles.ColorWhite)
	fgMuted := lipgloss.Color(styles.ColorBorder)
	accent := lipgloss.Color(styles.ColorSecondaryFg)
	accentBorder := lipgloss.Color(styles.ColorSecondaryBorder)

	noBase := lipgloss.NewStyle()

	t.Focused.Base = noBase.BorderLeft(true).
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(accentBorder).
		PaddingLeft(1)
	t.Focused.Title = lipgloss.NewStyle().Foreground(accent).Bold(true)
	t.Focused.Description = lipgloss.NewStyle().Foreground(fgMuted)
	t.Focused.ErrorIndicator = lipgloss.NewStyle().Foreground(lipgloss.Color(styles.ColorError))
	t.Focused.ErrorMessage = lipgloss.NewStyle().Foreground(lipgloss.Color(styles.ColorError))
	t.Focused.SelectSelector = lipgloss.NewStyle().Foreground(accent)
	t.Focused.Option = lipgloss.NewStyle().Foreground(fg)
	t.Focused.SelectedOption = lipgloss.NewStyle().Foreground(accent).Bold(true)

	// Focused text input styles
	t.Focused.TextInput.Text = lipgloss.NewStyle().Foreground(fg)
	t.Focused.TextInput.Placeholder = lipgloss.NewStyle().
		Foreground(fgMuted).
		Italic(true) // Optional: adds visual distinction
	t.Focused.TextInput.Cursor = lipgloss.NewStyle().Foreground(accent)
	t.Focused.TextInput.Prompt = lipgloss.NewStyle().Foreground(accentBorder)

	t.Blurred.Base = noBase.BorderLeft(true).
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("0")).
		PaddingLeft(1)
	t.Blurred.Title = lipgloss.NewStyle().Foreground(fgMuted)
	t.Blurred.Description = lipgloss.NewStyle().Foreground(fgMuted)

	// Blurred text input styles - making placeholders less opaque than the text
	t.Blurred.TextInput.Text = lipgloss.NewStyle().Foreground(fgMuted) // Regular text
	t.Blurred.TextInput.Placeholder = lipgloss.NewStyle().
		Foreground(lipgloss.Color("237")).
		Faint(true) // Makes it more transparent/less opaque
	t.Blurred.TextInput.Prompt = lipgloss.NewStyle().Foreground(fgMuted)

	t.Blurred.SelectSelector = lipgloss.NewStyle().Foreground(fgMuted)
	t.Blurred.Option = lipgloss.NewStyle().Foreground(fgMuted)
	t.Blurred.SelectedOption = lipgloss.NewStyle().Foreground(fgMuted)
}

func newCreateEventModel(service *calendar.Service, state AppState, width, height int) *createEventModel {
	calendars, err := service.GetAllCalendars()
	calendarOptions := []huh.Option[string]{}
	if err == nil {
		for _, cal := range calendars.Items {
			calendarOptions = append(calendarOptions, huh.NewOption(cal.Summary, cal.Id))
		}
	}

	m := &createEventModel{
		service:    service,
		state:      state,
		calendarID: state.SelectedCalendar,
		width:      width,
		height:     height,
	}

	// single group = all fields visible at once
	m.form = huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Description("Which calendar to add this event to?").
				Options(calendarOptions...).
				Value(&m.calendarID),

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
	).
		WithWidth(width).
		WithHeight(m.formHeight(height)).
		WithTheme(createEventTheme)

	return m
}

func (m *createEventModel) Init() tea.Cmd {
	return m.form.Init()
}

func (m *createEventModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case sizedMsg:
		m.width = msg.width
		m.height = msg.height
		m.form = m.form.WithWidth(msg.width).WithHeight(m.formHeight(msg.height))
		return m, nil

	case tea.KeyMsg:
		if msg.String() == "esc" {
			return m, func() tea.Msg { return NavigateTo{Screen: ui.MainMenuScreen} }
		}
	}

	form, cmd := m.form.Update(msg)
	if f, ok := form.(*huh.Form); ok {
		m.form = f
	}

	if m.form.State == huh.StateCompleted && !m.submitted {
		m.submitted = true
		m.form = nil
		return m, m.submitEvent()
	}

	return m, cmd
}

func (m *createEventModel) View() string {
	if m.submitted {
		return "Creating event..."
	}

	title := createEventTitleStyle.Render("Create Event")

	return createEventWrapperStyle.Render(
		lipgloss.JoinVertical(
			lipgloss.Left,
			title,
			m.form.View(),
		),
	)
}

// formHeight reserves space for the title so the form doesn't overflow into the status bar
func (m *createEventModel) formHeight(totalHeight int) int {
	titleHeight := lipgloss.Height(createEventTitleStyle.Render("Create Event"))
	_, wrapperV := createEventWrapperStyle.GetFrameSize()
	return totalHeight - titleHeight - wrapperV
}

func (m *createEventModel) submitEvent() tea.Cmd {
	return func() tea.Msg {
		start, _ := time.ParseInLocation("2006-01-02 15:04", m.startStr, time.Local)
		end, _ := time.ParseInLocation("2006-01-02 15:04", m.endStr, time.Local)

		_, err := m.service.CreateEvent(m.calendarID, m.title, m.location, m.description, start, end)
		if err != nil {
			fmt.Printf("Error creating event: %s\n", err)
			return NavigateTo{Screen: ui.MainMenuScreen}
		}

		return eventCreatedMsg{}
	}
}
