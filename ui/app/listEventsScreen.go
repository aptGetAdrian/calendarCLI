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
	gcalendar "google.golang.org/api/calendar/v3"
)

// ─── field enum ───────────────────────────────────────────────────────────────

type leField int

const (
	leCalendar leField = iota
	leWeek
	leFieldCount
)

const leTitleBarH = 4

// ─── event data ───────────────────────────────────────────────────────────────

type leEvent struct {
	title    string
	startDay int // 0-6 offset from weekStart (Mon=0)
	startHr  int // 0-23
	endHr    int // exclusive end hour (1-24); event spans rows startHr..endHr-1
}

type fetchedEventsMsg struct {
	events []leEvent
	err    error
}

// buildLeEvents converts raw API events into grid-positioned leEvents.
func buildLeEvents(raw []*gcalendar.Event, weekStart time.Time) []leEvent {
	var result []leEvent
	for _, e := range raw {
		if e.Start == nil || e.Summary == "" {
			continue
		}

		var startT time.Time
		if e.Start.DateTime != "" {
			t, err := time.Parse(time.RFC3339, e.Start.DateTime)
			if err != nil {
				continue
			}
			startT = t.Local()
		} else if e.Start.Date != "" {
			// all-day: skip for now
			continue
		} else {
			continue
		}

		var endT time.Time
		if e.End != nil && e.End.DateTime != "" {
			t, err := time.Parse(time.RFC3339, e.End.DateTime)
			if err == nil {
				endT = t.Local()
			}
		}

		// day offset relative to weekStart
		ws := time.Date(weekStart.Year(), weekStart.Month(), weekStart.Day(), 0, 0, 0, 0, time.Local)
		sd := time.Date(startT.Year(), startT.Month(), startT.Day(), 0, 0, 0, 0, time.Local)
		dayOffset := int(sd.Sub(ws).Hours() / 24)
		if dayOffset < 0 || dayOffset > 6 {
			continue
		}

		startHr := startT.Hour()
		endHr := startHr + 1 // default: 1-hour block
		if !endT.IsZero() {
			endHr = endT.Hour()
			if endT.Minute() > 0 {
				endHr++
			}
		}
		if endHr > 24 {
			endHr = 24
		}
		if endHr <= startHr {
			endHr = startHr + 1
		}

		result = append(result, leEvent{
			title:    e.Summary,
			startDay: dayOffset,
			startHr:  startHr,
			endHr:    endHr,
		})
	}
	return result
}

// ─── model ────────────────────────────────────────────────────────────────────

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
	events     []leEvent
	loading    bool
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
		loading:    true,
	}
}

func (m *listEventsModel) Init() tea.Cmd {
	return m.fetchEvents()
}

func (m *listEventsModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case sizedMsg:
		m.width, m.height = msg.width, msg.height
		return m, nil
	case fetchedEventsMsg:
		m.loading = false
		if msg.err != nil {
			m.logger.Error("Failed to fetch events: %v", msg.err)
		} else {
			m.events = msg.events
		}
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
				m.loading = true
				return m, m.fetchEvents()
			}
		case leWeek:
			m.weekStart = m.weekStart.AddDate(0, 0, -7)
			m.loading = true
			return m, m.fetchEvents()
		}
	case "right":
		switch m.focused {
		case leCalendar:
			if m.calIdx < len(m.calOpts)-1 {
				m.calIdx++
				m.loading = true
				return m, m.fetchEvents()
			}
		case leWeek:
			m.weekStart = m.weekStart.AddDate(0, 0, 7)
			m.loading = true
			return m, m.fetchEvents()
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

// fetchEvents fires the API call in a background goroutine.
func (m *listEventsModel) fetchEvents() tea.Cmd {
	calID := ""
	if len(m.calOpts) > 0 {
		calID = m.calOpts[m.calIdx].id
	}
	weekStart := m.weekStart
	svc := m.service
	log := m.logger
	return func() tea.Msg {
		weekEnd := weekStart.AddDate(0, 0, 7)
		raw, err := svc.ListEventsForCalendarInRange(calID, weekStart, weekEnd)
		if err != nil {
			log.Error("fetchEvents: %v", err)
			return fetchedEventsMsg{err: err}
		}
		return fetchedEventsMsg{events: buildLeEvents(raw, weekStart)}
	}
}

// ─── event lookup helpers ─────────────────────────────────────────────────────

// eventsStartingAt returns events that start at (day, hour).
func (m *listEventsModel) eventsStartingAt(day, hour int) []leEvent {
	var out []leEvent
	for _, e := range m.events {
		if e.startDay == day && e.startHr == hour {
			out = append(out, e)
		}
	}
	return out
}

// continuesAt returns true if an event started before hour and still spans hour.
func (m *listEventsModel) continuesAt(day, hour int) bool {
	for _, e := range m.events {
		if e.startDay == day && e.startHr < hour && hour < e.endHr {
			return true
		}
	}
	return false
}

// ─── layout helpers ───────────────────────────────────────────────────────────

func (m *listEventsModel) gridRows() int {
	// blank + controls + blank + header + blank + help = 6 fixed lines
	r := m.height - leTitleBarH - 6
	if r < 1 {
		return 1
	}
	return r
}

func (m *listEventsModel) colWidth() int {
	avail := m.width - 6 // 6-char time column
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

// ─── controls row ─────────────────────────────────────────────────────────────

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
	if m.loading {
		calVal += lipgloss.NewStyle().Foreground(muted).Render("  loading…")
	}

	// Week slider
	weekLblSty := lipgloss.NewStyle().Foreground(dim)
	if weekFocused {
		weekLblSty = lipgloss.NewStyle().Foreground(accent).Bold(true)
	}
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

	return calLblSty.Render("Calendar: ") + calVal +
		"    " +
		weekLblSty.Render("Week: ") + weekVal
}

// ─── day header ───────────────────────────────────────────────────────────────

func (m *listEventsModel) renderDayHeader() string {
	accent := lipgloss.Color(styles.ColorSecondaryFg)
	dim := lipgloss.Color(styles.ColorBorder)
	sepSty := lipgloss.NewStyle().Foreground(lipgloss.Color("238"))
	today := time.Now()
	todayY, todayM, todayD := today.Year(), today.Month(), today.Day()
	colW := m.colWidth()

	row := strings.Repeat(" ", 6)
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

const (
	eventColor       = "#5f9fd7" // steel blue for event starts
	eventContColor   = "#3d6e8a" // dimmer blue for event continuation
	eventStartPrefix = "▶ "
	eventContPrefix  = "  │ "
)

func (m *listEventsModel) renderGrid() string {
	timeSty := lipgloss.NewStyle().Foreground(lipgloss.Color(styles.ColorBorder))
	sepSty := lipgloss.NewStyle().Foreground(lipgloss.Color("238"))
	eventSty := lipgloss.NewStyle().Foreground(lipgloss.Color(eventColor)).Bold(true)
	contSty := lipgloss.NewStyle().Foreground(lipgloss.Color(eventContColor))
	emptySty := lipgloss.NewStyle().Foreground(lipgloss.Color("236"))

	colW := m.colWidth()
	maxRows := m.gridRows()

	// inner content width: colW minus 1 for the separator
	innerW := colW - 1
	if innerW < 1 {
		innerW = 1
	}

	var rows []string
	for i := 0; i < maxRows; i++ {
		h := m.scrollHour + i
		if h >= 24 {
			rows = append(rows, "")
			continue
		}

		row := timeSty.Render(fmt.Sprintf("%02d:00 ", h))
		for d := 0; d < 7; d++ {
			var cell string
			evts := m.eventsStartingAt(d, h)
			if len(evts) > 0 {
				e := evts[0]
				label := eventStartPrefix + e.title
				if len(label) > innerW {
					label = label[:innerW]
				} else {
					label = fmt.Sprintf("%-*s", innerW, label)
				}
				cell = eventSty.Render(label)
				if len(evts) > 1 {
					// show count indicator in last char
					cell = eventSty.Render(fmt.Sprintf("%-*s+%d", innerW-2, eventStartPrefix+e.title, len(evts)-1))
				}
			} else if m.continuesAt(d, h) {
				bar := eventContPrefix
				rest := strings.Repeat("─", innerW-len([]rune(eventContPrefix)))
				if rest == "" {
					rest = ""
				}
				cell = contSty.Render(fmt.Sprintf("%-*s", innerW, bar+rest))
			} else {
				cell = emptySty.Render(strings.Repeat(" ", innerW))
			}

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
