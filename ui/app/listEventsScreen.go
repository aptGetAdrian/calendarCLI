package app

import (
	calendar "calendarCli/internal"
	"calendarCli/internal/logger"
	"calendarCli/ui"
	"calendarCli/ui/styles"
	"fmt"
	"sort"
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

// ─── color palette ────────────────────────────────────────────────────────────

const eventFg = "#d4cfc8"

var eventPalette = []string{
	"#3b5a4a", // faded sage
	"#4a4a2a", // dusty olive
	"#4a3a28", // warm umber
	"#2a3f4a", // muted slate
	"#4a3048", // faded mauve
	"#2e4a3a", // washed moss
	"#4a4030", // aged amber
	"#333a4a", // dim denim
}

// ─── event data ───────────────────────────────────────────────────────────────

type leEvent struct {
	title    string
	startDay int    // 0-6 offset from weekStart (Mon=0)
	startHr  int    // 0-23
	endHr    int    // exclusive end hour (1-24); event spans rows startHr..endHr-1
	color    string // background color from palette
}

type fetchedEventsMsg struct {
	events []leEvent
	err    error
}

// buildLeEvents converts raw API events into grid-positioned leEvents.
// Multi-day events are split into one leEvent per day they cover within the week.
func buildLeEvents(raw []*gcalendar.Event, weekStart time.Time) []leEvent {
	weekDay0 := time.Date(weekStart.Year(), weekStart.Month(), weekStart.Day(), 0, 0, 0, 0, time.Local)
	weekDay7 := weekDay0.AddDate(0, 0, 7)

	var result []leEvent
	colorIdx := 0 // increments per raw event so all segments share one color

	for _, e := range raw {
		if e.Start == nil {
			continue
		}
		title := e.Summary
		if title == "" {
			title = "(No title)"
		}

		// Parse start time (timed or all-day).
		var startT time.Time
		if e.Start.DateTime != "" {
			t, err := time.Parse(time.RFC3339, e.Start.DateTime)
			if err != nil {
				continue
			}
			startT = t.Local()
		} else if e.Start.Date != "" {
			t, err := time.Parse("2006-01-02", e.Start.Date)
			if err != nil {
				continue
			}
			startT = time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.Local)
		} else {
			continue
		}

		// Parse end time (timed or all-day; Google's all-day End.Date is exclusive).
		var endT time.Time
		if e.End != nil {
			if e.End.DateTime != "" {
				t, err := time.Parse(time.RFC3339, e.End.DateTime)
				if err == nil {
					endT = t.Local()
				}
			} else if e.End.Date != "" {
				t, err := time.Parse("2006-01-02", e.End.Date)
				if err == nil {
					endT = time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.Local)
				}
			}
		}
		if endT.IsZero() {
			endT = startT.Add(time.Hour)
		}

		// Skip events entirely outside this week.
		if !endT.After(weekDay0) || !startT.Before(weekDay7) {
			continue
		}

		color := eventPalette[colorIdx%len(eventPalette)]
		colorIdx++

		// Emit one leEvent segment for each day of the week this event overlaps.
		for d := 0; d < 7; d++ {
			dayStart := weekDay0.AddDate(0, 0, d)
			dayEnd := dayStart.AddDate(0, 0, 1)

			// Skip days the event doesn't touch.
			if !startT.Before(dayEnd) || !endT.After(dayStart) {
				continue
			}

			// Clamp start/end to this day's bounds.
			var startHr, endHr int
			if !startT.Before(dayStart) {
				startHr = startT.Hour()
			} else {
				startHr = 0
			}

			if endT.Before(dayEnd) || endT.Equal(dayEnd) {
				endHr = endT.Hour()
				if endT.Minute() > 0 {
					endHr++
				}
			} else {
				endHr = 24
			}
			if endHr > 24 {
				endHr = 24
			}
			if endHr <= startHr {
				endHr = startHr + 1
			}

			result = append(result, leEvent{
				title:    title,
				startDay: d,
				startHr:  startHr,
				endHr:    endHr,
				color:    color,
			})
		}
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
		scrollHour: 0,
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

// ─── event lookup ─────────────────────────────────────────────────────────────

// eventsAt returns every event covering (day, hour), sorted stably so relative
// left-to-right positions stay consistent across rows.
func (m *listEventsModel) eventsAt(day, hour int) []leEvent {
	var out []leEvent
	for _, e := range m.events {
		if e.startDay == day && e.startHr <= hour && hour < e.endHr {
			out = append(out, e)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].startHr != out[j].startHr {
			return out[i].startHr < out[j].startHr
		}
		return out[i].title < out[j].title
	})
	return out
}

// ─── layout helpers ───────────────────────────────────────────────────────────

// visibleHours returns how many full hour slots fit in the grid area.
// Each hour slot takes 2 terminal rows (content + separator line).
func (m *listEventsModel) visibleHours() int {
	// blank + controls + blank + header + blank + help = 6 fixed lines
	total := m.height - leTitleBarH - 6
	if total < 2 {
		return 1
	}
	return total / 2
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

// truncRunes truncates s to at most n runes.
func truncRunes(s string, n int) string {
	r := []rune(s)
	if len(r) > n {
		return string(r[:n])
	}
	return s
}

// renderEventCell builds the colored sub-columns for a single (day, hour) cell.
// showTitle controls whether to render the event title (true for the first
// visible row of the event block, false for all continuation rows).
func (m *listEventsModel) renderEventCell(evts []leEvent, innerW, h int, showTitle bool) string {
	n := len(evts)
	subW := innerW / n
	if subW < 1 {
		subW = 1
	}
	var sb strings.Builder
	for idx, e := range evts {
		w := subW
		if idx == n-1 {
			w = innerW - subW*(n-1) // last slot absorbs rounding remainder
		}
		sty := lipgloss.NewStyle().
			Background(lipgloss.Color(e.color)).
			Foreground(lipgloss.Color(eventFg)).
			Width(w)
		isFirstVisible := e.startHr == h || (e.startHr < m.scrollHour && h == m.scrollHour)
		if showTitle && isFirstVisible {
			sb.WriteString(sty.Bold(true).Render(truncRunes(e.title, w)))
		} else {
			sb.WriteString(sty.Render(""))
		}
	}
	return sb.String()
}

func (m *listEventsModel) renderGrid() string {
	timeSty := lipgloss.NewStyle().Foreground(lipgloss.Color(styles.ColorBorder))
	timeEmptySty := lipgloss.NewStyle().Foreground(lipgloss.Color("236"))
	sepSty := lipgloss.NewStyle().Foreground(lipgloss.Color("238"))
	hourLineSty := lipgloss.NewStyle().Foreground(lipgloss.Color("236"))

	colW := m.colWidth()
	innerW := colW - 1
	if innerW < 1 {
		innerW = 1
	}

	visHours := m.visibleHours()
	var rows []string

	for i := 0; i < visHours; i++ {
		h := m.scrollHour + i
		if h >= 24 {
			// Pad remaining rows with blank lines to keep layout stable.
			rows = append(rows, "", "")
			continue
		}

		// ── content row ──────────────────────────────────────────────────────
		contentRow := timeSty.Render(fmt.Sprintf("%02d:00 ", h))
		for d := 0; d < 7; d++ {
			evts := m.eventsAt(d, h)
			var cell string
			if len(evts) == 0 {
				cell = strings.Repeat(" ", innerW)
			} else {
				cell = m.renderEventCell(evts, innerW, h, true)
			}
			if d < 6 {
				cell += sepSty.Render("│")
			}
			contentRow += cell
		}
		rows = append(rows, contentRow)

		// ── separator row (dim horizontal lines between hours) ────────────────
		sepRow := timeEmptySty.Render(strings.Repeat(" ", 6))
		for d := 0; d < 7; d++ {
			evts := m.eventsAt(d, h)
			var cell string
			if len(evts) == 0 {
				cell = hourLineSty.Render(strings.Repeat("─", innerW))
			} else {
				// Keep event color block continuous through the separator row.
				cell = m.renderEventCell(evts, innerW, h, false)
			}
			if d < 6 {
				cell += sepSty.Render("│")
			}
			sepRow += cell
		}
		rows = append(rows, sepRow)
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
	if m.scrollHour+m.visibleHours() < 24 {
		base += dim.Render(fmt.Sprintf("  [↓ %02d:00]", m.scrollHour+m.visibleHours()))
	}
	return base
}
