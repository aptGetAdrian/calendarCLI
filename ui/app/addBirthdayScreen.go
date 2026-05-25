package app

import (
	calendar "calendarCli/internal"
	"calendarCli/internal/logger"
	"calendarCli/ui"
	"calendarCli/ui/styles"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type addBirthdayScreenModel struct {
	service     *calendar.Service
	state       AppState
	firstInput  textinput.Model
	lastInput   textinput.Model
	dayInput    textinput.Model
	monthInput  textinput.Model
	activeField int // 0=first, 1=last, 2=day, 3=month
	errMsg      string
	width       int
	height      int
	logger      *logger.Logger
}

func newAddBirthdayScreenModel(service *calendar.Service, state AppState, width, height int, logger *logger.Logger) *addBirthdayScreenModel {
	newInput := func(placeholder string, maxChars int) textinput.Model {
		ti := textinput.New()
		ti.Placeholder = placeholder
		ti.Prompt = ""
		ti.CharLimit = maxChars
		return ti
	}

	first := newInput("e.g. John", 50)
	last := newInput("e.g. Doe", 50)
	day := newInput("1–31", 2)
	month := newInput("1–12", 2)

	first.Focus()

	return &addBirthdayScreenModel{
		service:    service,
		state:      state,
		firstInput: first,
		lastInput:  last,
		dayInput:   day,
		monthInput: month,
		width:      width,
		height:     height,
		logger:     logger,
	}
}

func (m *addBirthdayScreenModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m *addBirthdayScreenModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case sizedMsg:
		m.width = msg.width
		m.height = msg.height
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			return m, func() tea.Msg { return NavigateTo{Screen: ui.MainMenuScreen} }

		case "tab", "enter":
			return m.advanceField()

		case "shift+tab":
			return m.retreatField()

		case "ctrl+s":
			return m.submit()
		}
	}

	var cmd tea.Cmd
	switch m.activeField {
	case 0:
		m.firstInput, cmd = m.firstInput.Update(msg)
	case 1:
		m.lastInput, cmd = m.lastInput.Update(msg)
	case 2:
		m.dayInput, cmd = m.dayInput.Update(msg)
	case 3:
		m.monthInput, cmd = m.monthInput.Update(msg)
	}
	return m, cmd
}

func (m *addBirthdayScreenModel) advanceField() (tea.Model, tea.Cmd) {
	if m.activeField == 3 {
		return m.submit()
	}
	m.blurAll()
	m.activeField++
	return m, m.focusCurrent()
}

func (m *addBirthdayScreenModel) retreatField() (tea.Model, tea.Cmd) {
	if m.activeField == 0 {
		return m, nil
	}
	m.blurAll()
	m.activeField--
	return m, m.focusCurrent()
}

func (m *addBirthdayScreenModel) blurAll() {
	m.firstInput.Blur()
	m.lastInput.Blur()
	m.dayInput.Blur()
	m.monthInput.Blur()
}

func (m *addBirthdayScreenModel) focusCurrent() tea.Cmd {
	switch m.activeField {
	case 0:
		return m.firstInput.Focus()
	case 1:
		return m.lastInput.Focus()
	case 2:
		return m.dayInput.Focus()
	case 3:
		return m.monthInput.Focus()
	}
	return nil
}

func (m *addBirthdayScreenModel) submit() (tea.Model, tea.Cmd) {
	firstName := strings.TrimSpace(m.firstInput.Value())
	lastName := strings.TrimSpace(m.lastInput.Value())
	dayStr := strings.TrimSpace(m.dayInput.Value())
	monthStr := strings.TrimSpace(m.monthInput.Value())

	if firstName == "" {
		m.errMsg = "First name is required."
		return m, nil
	}

	day, err := strconv.Atoi(dayStr)
	if err != nil || day < 1 || day > 31 {
		m.errMsg = "Day must be a number between 1 and 31."
		return m, nil
	}

	month, err := strconv.Atoi(monthStr)
	if err != nil || month < 1 || month > 12 {
		m.errMsg = "Month must be a number between 1 and 12."
		return m, nil
	}

	// validate the date is real (e.g. not Feb 30)
	if _, err := time.Parse("2006-01-02", fmt.Sprintf("2000-%02d-%02d", month, day)); err != nil {
		m.errMsg = fmt.Sprintf("Invalid date: %d/%d.", day, month)
		return m, nil
	}

	if err := m.service.CreateBirthdayEvent(firstName, lastName, day, month); err != nil {
		m.logger.Error("CreateBirthdayEvent: %v", err)
		m.errMsg = "Failed to save birthday. Check the logs."
		return m, nil
	}

	return m, func() tea.Msg { return NavigateTo{Screen: ui.MainMenuScreen} }
}

var monthNames = [13]string{"", "January", "February", "March", "April", "May", "June",
	"July", "August", "September", "October", "November", "December"}

func (m *addBirthdayScreenModel) View() string {
	formW := m.width / 2
	if formW < 30 {
		formW = 30
	}

	inputW := formW - 4
	for _, inp := range []*textinput.Model{&m.firstInput, &m.lastInput, &m.dayInput, &m.monthInput} {
		inp.Width = inputW
	}

	amber := lipgloss.NewStyle().Foreground(lipgloss.Color("#f0d080")).Bold(true)
	muted := lipgloss.NewStyle().Foreground(lipgloss.Color(styles.ColorBorder))
	errStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(styles.ColorError))

	title := styles.AddBirthdayTtitle().Render("Add Birthday")

	monthHint := ""
	if n, err := strconv.Atoi(strings.TrimSpace(m.monthInput.Value())); err == nil && n >= 1 && n <= 12 {
		monthHint = muted.Render("  → " + monthNames[n])
	}

	errLine := ""
	if m.errMsg != "" {
		errLine = errStyle.Render("✗ " + m.errMsg)
	}

	hint := styles.InfoText.Render("tab/enter: next  shift+tab: prev  ctrl+s: save  esc: back")

	form := lipgloss.JoinVertical(lipgloss.Left,
		title,
		"",
		amber.Render("First name *"),
		m.firstInput.View(),
		"",
		amber.Render("Last name"),
		m.lastInput.View(),
		"",
		amber.Render("Day *"),
		m.dayInput.View(),
		"",
		amber.Render("Month *"),
		m.monthInput.View(),
		monthHint,
		"",
		errLine,
	)

	body := lipgloss.NewStyle().
		Width(formW).
		PaddingLeft(2).
		Render(form)

	hintBar := lipgloss.NewStyle().
		Width(m.width).
		PaddingLeft(1).
		Render(hint)

	content := lipgloss.NewStyle().
		Height(m.height - lipgloss.Height(hintBar)).
		Render(body)

	return lipgloss.JoinVertical(lipgloss.Left, content, hintBar)
}
