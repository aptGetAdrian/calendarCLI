package app

import (
	calendar "calendarCli/internal"
	"calendarCli/internal/logger"
	"calendarCli/ui"
	"calendarCli/ui/styles"
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type leField int

const (
	leCalendar leField = iota
	leWeek
	leFieldCount
)

const leTitleBarH = 4

type listEventsModel struct {
	service    *calendar.Service
	state      AppState
	width      int
	height     int
	logger     *logger.Logger
	focused    leField
	calOpts    []calOpt
	calIdx     int
	weekStart  time.Time
	scrollHour int
}

func mondayOf(t time.Time) time.Time {
	wd := int(t.Weekday())
	if wd == 0 {
		wd = 7
	}
	return time.Date(t.Year(), t.Month(), t.Day()-(wd-1), 0, 0, 0, 0, t.Location())
}

func newListEventsModel(service *calendar.Service, state AppState, width, height int, log *logger.Logger) *listEventsModel {
	calOpts := []calOpt{}
	if cals, err := service.GetAllCalendars(); err == nil {
		for _, c := range cals.Items {
			calOpts = append(calOpts, calOpt{c.Summary, c.Id})
		}
	}

	calIdx := 0
	for i, o := range calOpts {
		if o.id == state.SelectedCalendar || o.name == state.SelectedCalendar {
			calIdx = i
			break
		}
	}

	return &listEventsModel{
		service:    service,
		state:      state,
		width:      width,
		height:     height,
		logger:     log,
		focused:    leCalendar,
		calOpts:    calOpts,
		calIdx:     calIdx,
		weekStart:  mondayOf(time.Now()),
		scrollHour: 8,
	}
}

func (m *listEventsModel) Init() tea.Cmd { return nil }

func (m *listEventsModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case sizedMsg:
		m.width, m.height = msg.width, msg.height
		return m, nil
	case tea.KeyMsg:
		return m.handleKey(msg)
	}
	return m, nil
}

func (m *listEventsModel) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		return m, func() tea.Msg { return NavigateTo{Screen: ui.MainMenuScreen} }
	case "tab":
		m.focused = leField((int(m.focused) + 1) % int(leFieldCount))
	case "shift+tab":
		m.focused = leField((int(m.focused) + int(leFieldCount) - 1) % int(leFieldCount))
	case "left":
		switch m.focused {
		case leCalendar:
			if m.calIdx > 0 {
				m.calIdx--
			}
		case leWeek:
			m.weekStart = m.weekStart.AddDate(0, 0, -7)
		}
	case "right":
		switch m.focused {
		case leCalendar:
			if m.calIdx < len(m.calOpts)-1 {
				m.calIdx++
			}
		case leWeek:
			m.weekStart = m.weekStart.AddDate(0, 0, 7)
		}
	case "up":
		if m.scrollHour > 0 {
			m.scrollHour--
		}
	case "down":
		if m.scrollHour < 23 {
			m.scrollHour++
		}
	}
	return m, nil
}

// gridRows returns the number of hour rows that fit in the available height.
func (m *listEventsModel) gridRows() int {
	// fixed lines below title bar: blank + controls + blank + header + blank + help
	const fixedBody = 6
	r := m.height - leTitleBarH - fixedBody
	if r < 1 {
		return 1
	}
	return r
}

// colWidth returns the width of each day column.
func (m *listEventsModel) colWidth() int {
	const timeColW = 6
	avail := m.width - timeColW
	if avail < 7 {
		return 1
	}
	return avail / 7
}

// ─── View ─────────────────────────────────────────────────────────────────────

func (m *listEventsModel) View() string {
	titleBar := lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(styles.ColorSecondaryBorder)).
		Padding(0, 2).Bold(true).
		Foreground(lipgloss.Color(styles.ColorSecondaryFg)).
		Background(lipgloss.Color(styles.ColorSecondaryBg)).
		Render("List Events")

	body := m.buildBody()
	lines := strings.Split(body, "\n")
	avail := m.height - leTitleBarH
	if avail < 1 {
		avail = 1
	}
	return titleBar + "\n" + clipAndPad(lines, 0, avail)
}

func (m *listEventsModel) buildBody() string {
	return strings.Join([]string{
		"",
		m.renderControls(),
		"",
		m.renderDayHeader(),
		m.renderGrid(),
		"",
		m.renderHelp(),
	}, "\n")
}

// ─── controls row (calendar + week sliders) ───────────────────────────────────

func (m *listEventsModel) renderControls() string {
	accent := lipgloss.Color(styles.ColorSecondaryFg)
	dim := lipgloss.Color(styles.ColorBorder)
	muted := lipgloss.Color("244")

	calFocused := m.focused == leCalendar
	weekFocused := m.focused == leWeek

	// Calendar slider
	calLblSty := lipgloss.NewStyle().Foreground(dim)
	if calFocused {
		calLblSty = lipgloss.NewStyle().Foreground(accent).Bold(true)
	}
	calLbl := calLblSty.Render("Calendar: ")

	var calVal string
	if len(m.calOpts) == 0 {
		calVal = lipgloss.NewStyle().Foreground(muted).Render("(no calendars)")
	} else {
		name := m.calOpts[m.calIdx].name
		lA := lipgloss.NewStyle().Foreground(muted).Render("◀")
		rA := lipgloss.NewStyle().Foreground(muted).Render("▶")
		if calFocused {
			if m.calIdx > 0 {
				lA = lipgloss.NewStyle().Foreground(accent).Render("◀")
			}
			if m.calIdx < len(m.calOpts)-1 {
				rA = lipgloss.NewStyle().Foreground(accent).Render("▶")
			}
		}
		nameSty := lipgloss.NewStyle().Foreground(accent)
		if calFocused {
			nameSty = nameSty.Bold(true)
		}
		calVal = lA + " " + nameSty.Render(name) + " " + rA
	}

	// Week slider
	weekLblSty := lipgloss.NewStyle().Foreground(dim)
	if weekFocused {
		weekLblSty = lipgloss.NewStyle().Foreground(accent).Bold(true)
	}
	weekLbl := weekLblSty.Render("Week: ")

	weekEnd := m.weekStart.AddDate(0, 0, 6)
	var weekStr string
	if m.weekStart.Month() == weekEnd.Month() {
		weekStr = fmt.Sprintf("%s %d–%d, %d",
			m.weekStart.Format("Jan"), m.weekStart.Day(), weekEnd.Day(), weekEnd.Year())
	} else {
		weekStr = fmt.Sprintf("%s %d – %s %d, %d",
			m.weekStart.Format("Jan"), m.weekStart.Day(),
			weekEnd.Format("Jan"), weekEnd.Day(), weekEnd.Year())
	}
	wlA := lipgloss.NewStyle().Foreground(muted).Render("◀")
	wrA := lipgloss.NewStyle().Foreground(muted).Render("▶")
	if weekFocused {
		wlA = lipgloss.NewStyle().Foreground(accent).Render("◀")
		wrA = lipgloss.NewStyle().Foreground(accent).Render("▶")
	}
	weekValSty := lipgloss.NewStyle().Foreground(accent)
	if weekFocused {
		weekValSty = weekValSty.Bold(true)
	}
	weekVal := wlA + " " + weekValSty.Render(weekStr) + " " + wrA

	return calLbl + calVal + "    " + weekLbl + weekVal
}

// ─── day header row ───────────────────────────────────────────────────────────

func (m *listEventsModel) renderDayHeader() string {
	accent := lipgloss.Color(styles.ColorSecondaryFg)
	dim := lipgloss.Color(styles.ColorBorder)
	sepSty := lipgloss.NewStyle().Foreground(lipgloss.Color("238"))
	today := time.Now()
	todayY, todayM, todayD := today.Year(), today.Month(), today.Day()
	colW := m.colWidth()

	row := strings.Repeat(" ", 6) // time column placeholder
	for i := 0; i < 7; i++ {
		day := m.weekStart.AddDate(0, 0, i)
		label := fmt.Sprintf("%s %d", day.Format("Mon"), day.Day())
		isToday := day.Year() == todayY && day.Month() == todayM && day.Day() == todayD

		padded := centerIn(label, colW-1)
		var cell string
		if isToday {
			cell = lipgloss.NewStyle().Foreground(accent).Bold(true).Render(padded)
		} else {
			cell = lipgloss.NewStyle().Foreground(dim).Render(padded)
		}
		if i < 6 {
			cell += sepSty.Render("│")
		}
		row += cell
	}
	return row
}

// ─── schedule grid ────────────────────────────────────────────────────────────

func (m *listEventsModel) renderGrid() string {
	timeSty := lipgloss.NewStyle().Foreground(lipgloss.Color(styles.ColorBorder))
	sepSty := lipgloss.NewStyle().Foreground(lipgloss.Color("238"))
	colW := m.colWidth()
	maxRows := m.gridRows()

	var rows []string
	for i := 0; i < maxRows; i++ {
		h := m.scrollHour + i
		if h >= 24 {
			rows = append(rows, "")
			continue
		}
		row := timeSty.Render(fmt.Sprintf("%02d:00 ", h))
		for d := 0; d < 7; d++ {
			cell := strings.Repeat(" ", colW-1)
			if d < 6 {
				cell += sepSty.Render("│")
			}
			row += cell
		}
		rows = append(rows, row)
	}
	return strings.Join(rows, "\n")
}

// ─── help line ────────────────────────────────────────────────────────────────

func (m *listEventsModel) renderHelp() string {
	key := lipgloss.NewStyle().Foreground(lipgloss.Color(styles.ColorSecondaryBorder))
	dim := lipgloss.NewStyle().Foreground(lipgloss.Color("243"))

	base := key.Render("tab") + dim.Render("/") + key.Render("shift+tab") + dim.Render(" switch · ") +
		key.Render("←→") + dim.Render(" navigate · ") +
		key.Render("↑↓") + dim.Render(" scroll hours · ") +
		key.Render("esc") + dim.Render(" back")

	if m.scrollHour > 0 {
		base += dim.Render(fmt.Sprintf("  [↑ %02d:00]", m.scrollHour))
	}
	if m.scrollHour+m.gridRows() < 24 {
		base += dim.Render(fmt.Sprintf("  [↓ %02d:00]", m.scrollHour+m.gridRows()))
	}
	return base
}
