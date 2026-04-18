package app

import (
	calendar "calendarCli/internal"
	"calendarCli/internal/logger"
	"calendarCli/ui"
	"calendarCli/ui/styles"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ─── field enum (tab order) ───────────────────────────────────────────────────

type ceField int

const (
	ceTitle    ceField = iota
	ceCalendar         // up/down cycles calendars
	ceLocation
	ceDesc
	ceStartCal // month grid, arrows navigate days
	ceStartHr  // up/down ±1 hour
	ceStartMin // up/down ±5 min
	ceEndCal
	ceEndHr
	ceEndMin
	ceSubmit
	ceFieldCount
)

// ─── calendar grid widget ─────────────────────────────────────────────────────

type calPicker struct {
	year  int
	month time.Month
	day   int
}

func newCalPicker(t time.Time) calPicker {
	return calPicker{year: t.Year(), month: t.Month(), day: t.Day()}
}

func (c *calPicker) shiftDay(d int) {
	t := time.Date(c.year, c.month, c.day, 0, 0, 0, 0, time.Local).AddDate(0, 0, d)
	c.year, c.month, c.day = t.Year(), t.Month(), t.Day()
}

func (c *calPicker) shiftMonth(d int) {
	t := time.Date(c.year, c.month, 1, 0, 0, 0, 0, time.Local).AddDate(0, d, 0)
	c.year, c.month = t.Year(), t.Month()
	if max := calMonthDays(c.year, c.month); c.day > max {
		c.day = max
	}
}

func calMonthDays(y int, m time.Month) int {
	return time.Date(y, m+1, 0, 0, 0, 0, 0, time.Local).Day()
}

func centerIn(s string, width int) string {
	n := len(s)
	if n >= width {
		return s
	}
	left := (width - n) / 2
	right := width - n - left
	return strings.Repeat(" ", left) + s + strings.Repeat(" ", right)
}

// renderCalGrid renders a month calendar grid. Each cell is 3 chars wide.
func renderCalGrid(c calPicker, focused bool) string {
	now := time.Now()
	todayY, todayM, todayD := now.Year(), now.Month(), now.Day()

	accent := lipgloss.Color(styles.ColorSecondaryFg)
	border := lipgloss.Color(styles.ColorSecondaryBorder)
	dim := lipgloss.Color(styles.ColorBorder)
	dimmer := lipgloss.Color("238")
	warn := lipgloss.Color(styles.ColorWarning)

	var (
		arrowSty lipgloss.Style
		monthSty lipgloss.Style
		dowSty   lipgloss.Style
		daySty   lipgloss.Style
		todaySty lipgloss.Style
		selSty   lipgloss.Style
	)
	if focused {
		arrowSty = lipgloss.NewStyle().Foreground(accent)
		monthSty = lipgloss.NewStyle().Foreground(accent).Bold(true)
		dowSty   = lipgloss.NewStyle().Foreground(dim)
		daySty   = lipgloss.NewStyle().Foreground(dim)
		todaySty = lipgloss.NewStyle().Foreground(warn).Bold(true)
		selSty   = lipgloss.NewStyle().Foreground(lipgloss.Color(styles.ColorBlack)).Background(border).Bold(true)
	} else {
		arrowSty = lipgloss.NewStyle().Foreground(dimmer)
		monthSty = lipgloss.NewStyle().Foreground(dimmer)
		dowSty   = lipgloss.NewStyle().Foreground(dimmer)
		daySty   = lipgloss.NewStyle().Foreground(dimmer)
		todaySty = lipgloss.NewStyle().Foreground(dimmer)
		selSty   = lipgloss.NewStyle().Foreground(dim).Underline(true)
	}

	// Header: "◀  Apr 2026  ▶" centered in 21 chars
	mStr := fmt.Sprintf("%s %d", c.month.String()[:3], c.year)
	header := arrowSty.Render("◀") + monthSty.Render(centerIn(mStr, 19)) + arrowSty.Render("▶")

	// Day-of-week row (3 chars each)
	dows := []string{"Mo", "Tu", "We", "Th", "Fr", "Sa", "Su"}
	var dowRow string
	for _, d := range dows {
		dowRow += dowSty.Render(fmt.Sprintf("%-3s", d))
	}

	// Compute first weekday (Mon=0)
	wd := int(time.Date(c.year, c.month, 1, 0, 0, 0, 0, time.Local).Weekday())
	if wd == 0 {
		wd = 7
	}
	wd--

	total := calMonthDays(c.year, c.month)
	blank := daySty.Render("   ") // 3 spaces

	var gridRows []string
	day := 1
	for row := 0; row < 6 && day <= total; row++ {
		var rowStr string
		for col := 0; col < 7; col++ {
			if row == 0 && col < wd {
				rowStr += blank
				continue
			}
			if day > total {
				rowStr += blank
				continue
			}
			cell := fmt.Sprintf("%2d ", day)
			isSel := day == c.day
			isToday := day == todayD && c.month == todayM && c.year == todayY
			switch {
			case isSel:
				rowStr += selSty.Render(cell)
			case isToday:
				rowStr += todaySty.Render(cell)
			default:
				rowStr += daySty.Render(cell)
			}
			day++
		}
		gridRows = append(gridRows, rowStr)
	}

	lines := append([]string{header, dowRow}, gridRows...)
	return strings.Join(lines, "\n")
}

// ─── model ────────────────────────────────────────────────────────────────────

type calOpt struct{ name, id string }

type createEventModel struct {
	service *calendar.Service
	state   AppState
	width   int
	height  int
	logger  *logger.Logger

	focused   ceField
	submitted bool
	errMsg    string

	titleInput    textinput.Model
	locationInput textinput.Model
	descInput     textinput.Model

	calOpts []calOpt
	calIdx  int

	startPicker calPicker
	startHr     int
	startMin    int

	endPicker calPicker
	endHr     int
	endMin    int
}

const ceRightW = 23 // fixed width of the right panel (calendar grid = 21 + 2 padding)

func newCreateEventModel(service *calendar.Service, state AppState, width, height int, log *logger.Logger) *createEventModel {
	now := time.Now()
	startBase := now.Add(time.Hour).Round(15 * time.Minute)
	endBase := startBase.Add(time.Hour)

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

	inputW := ceLeftW(width) - 4
	if inputW < 10 {
		inputW = 10
	}

	mkInput := func(placeholder string) textinput.Model {
		ti := textinput.New()
		ti.Placeholder = placeholder
		ti.Width = inputW
		ti.PromptStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(styles.ColorSecondaryBorder))
		ti.TextStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(styles.ColorWhite))
		ti.PlaceholderStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("238")).Faint(true)
		ti.Cursor.Style = lipgloss.NewStyle().Foreground(lipgloss.Color(styles.ColorSecondaryFg))
		ti.Blur()
		return ti
	}

	titleIn := mkInput("Team standup")
	titleIn.Focus()

	return &createEventModel{
		service:       service,
		state:         state,
		width:         width,
		height:        height,
		logger:        log,
		focused:       ceTitle,
		titleInput:    titleIn,
		locationInput: mkInput("Conference room / Meet link"),
		descInput:     mkInput("What's this event about?"),
		calOpts:       calOpts,
		calIdx:        calIdx,
		startPicker:   newCalPicker(startBase),
		startHr:       startBase.Hour(),
		startMin:      startBase.Minute(),
		endPicker:     newCalPicker(endBase),
		endHr:         endBase.Hour(),
		endMin:        endBase.Minute(),
	}
}

func ceLeftW(totalW int) int {
	w := totalW - ceRightW - 3 // 3 = gap between panels
	if w < 20 {
		return 20
	}
	return w
}

// ─── Bubble Tea interface ─────────────────────────────────────────────────────

func (m *createEventModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m *createEventModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case sizedMsg:
		m.width, m.height = msg.width, msg.height
		inputW := ceLeftW(m.width) - 4
		if inputW < 10 {
			inputW = 10
		}
		m.titleInput.Width = inputW
		m.locationInput.Width = inputW
		m.descInput.Width = inputW
		return m, nil
	case tea.KeyMsg:
		return m.handleKey(msg)
	}
	return m, nil
}

func (m *createEventModel) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	switch key {
	case "esc":
		return m, func() tea.Msg { return NavigateTo{Screen: ui.MainMenuScreen} }
	case "tab", "enter":
		if m.focused == ceSubmit {
			return m, m.submit()
		}
		m.moveFocus(1)
		return m, textinput.Blink
	case "shift+tab":
		m.moveFocus(-1)
		return m, textinput.Blink
	}

	// Text inputs consume all other keys when focused
	var cmd tea.Cmd
	switch m.focused {
	case ceTitle:
		m.titleInput, cmd = m.titleInput.Update(msg)
		return m, cmd
	case ceLocation:
		m.locationInput, cmd = m.locationInput.Update(msg)
		return m, cmd
	case ceDesc:
		m.descInput, cmd = m.descInput.Update(msg)
		return m, cmd
	}

	// Directional keys for non-text fields
	switch key {
	case "up":
		m.arrowUp()
	case "down":
		m.arrowDown()
	case "left":
		m.arrowLeft()
	case "right":
		m.arrowRight()
	case "[", "pgup":
		switch m.focused {
		case ceStartCal:
			m.startPicker.shiftMonth(-1)
		case ceEndCal:
			m.endPicker.shiftMonth(-1)
		}
	case "]", "pgdown":
		switch m.focused {
		case ceStartCal:
			m.startPicker.shiftMonth(1)
		case ceEndCal:
			m.endPicker.shiftMonth(1)
		}
	}

	return m, nil
}

func (m *createEventModel) moveFocus(dir int) {
	m.titleInput.Blur()
	m.locationInput.Blur()
	m.descInput.Blur()
	m.errMsg = ""

	m.focused = ceField((int(m.focused) + int(ceFieldCount) + dir) % int(ceFieldCount))

	switch m.focused {
	case ceTitle:
		m.titleInput.Focus()
	case ceLocation:
		m.locationInput.Focus()
	case ceDesc:
		m.descInput.Focus()
	}
}

func (m *createEventModel) arrowUp() {
	switch m.focused {
	case ceCalendar:
		if m.calIdx > 0 {
			m.calIdx--
		}
	case ceStartCal:
		m.startPicker.shiftDay(-7)
	case ceStartHr:
		m.startHr = (m.startHr + 23) % 24
	case ceStartMin:
		m.startMin = (m.startMin + 55) % 60
	case ceEndCal:
		m.endPicker.shiftDay(-7)
	case ceEndHr:
		m.endHr = (m.endHr + 23) % 24
	case ceEndMin:
		m.endMin = (m.endMin + 55) % 60
	}
}

func (m *createEventModel) arrowDown() {
	switch m.focused {
	case ceCalendar:
		if m.calIdx < len(m.calOpts)-1 {
			m.calIdx++
		}
	case ceStartCal:
		m.startPicker.shiftDay(7)
	case ceStartHr:
		m.startHr = (m.startHr + 1) % 24
	case ceStartMin:
		m.startMin = (m.startMin + 5) % 60
	case ceEndCal:
		m.endPicker.shiftDay(7)
	case ceEndHr:
		m.endHr = (m.endHr + 1) % 24
	case ceEndMin:
		m.endMin = (m.endMin + 5) % 60
	}
}

func (m *createEventModel) arrowLeft() {
	switch m.focused {
	case ceStartCal:
		m.startPicker.shiftDay(-1)
	case ceEndCal:
		m.endPicker.shiftDay(-1)
	}
}

func (m *createEventModel) arrowRight() {
	switch m.focused {
	case ceStartCal:
		m.startPicker.shiftDay(1)
	case ceEndCal:
		m.endPicker.shiftDay(1)
	}
}

func (m *createEventModel) submit() tea.Cmd {
	if strings.TrimSpace(m.titleInput.Value()) == "" {
		m.errMsg = "Title is required"
		m.titleInput.Blur()
		m.locationInput.Blur()
		m.descInput.Blur()
		m.focused = ceTitle
		m.titleInput.Focus()
		return textinput.Blink
	}

	start := time.Date(m.startPicker.year, m.startPicker.month, m.startPicker.day,
		m.startHr, m.startMin, 0, 0, time.Local)
	end := time.Date(m.endPicker.year, m.endPicker.month, m.endPicker.day,
		m.endHr, m.endMin, 0, 0, time.Local)

	if !end.After(start) {
		m.errMsg = "End must be after start"
		return nil
	}

	calID := ""
	if len(m.calOpts) > 0 {
		calID = m.calOpts[m.calIdx].id
	}

	title := m.titleInput.Value()
	location := m.locationInput.Value()
	desc := m.descInput.Value()
	m.submitted = true

	return func() tea.Msg {
		_, err := m.service.CreateEvent(calID, title, location, desc, start, end)
		if err != nil {
			m.logger.Error("Error creating event: %s", err)
			return NavigateTo{Screen: ui.MainMenuScreen}
		}
		return eventCreatedMsg{}
	}
}

// ─── View ─────────────────────────────────────────────────────────────────────

func (m *createEventModel) View() string {
	if m.submitted {
		return lipgloss.NewStyle().
			PaddingTop(2).PaddingLeft(4).
			Foreground(lipgloss.Color(styles.ColorSecondaryFg)).
			Render("Creating event...")
	}

	titleBar := lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(styles.ColorSecondaryBorder)).
		Padding(0, 2).Bold(true).
		Foreground(lipgloss.Color(styles.ColorSecondaryFg)).
		Background(lipgloss.Color(styles.ColorSecondaryBg)).
		Render("Create Event")

	leftW := ceLeftW(m.width)
	left := m.renderLeft(leftW)
	right := m.renderRight()

	leftBox := lipgloss.NewStyle().Width(leftW).PaddingRight(3)
	panels := lipgloss.JoinHorizontal(lipgloss.Top, leftBox.Render(left), right)

	submit := m.renderSubmit()
	help := m.renderHelp()

	var parts []string
	parts = append(parts, titleBar, "", panels, "")
	if m.errMsg != "" {
		parts = append(parts, lipgloss.NewStyle().
			Foreground(lipgloss.Color(styles.ColorError)).
			Render("  ⚠  "+m.errMsg), "")
	}
	parts = append(parts, submit, "", help)

	return strings.Join(parts, "\n")
}

// ─── sub-renderers ────────────────────────────────────────────────────────────

func (m *createEventModel) renderLeft(w int) string {
	rows := []string{
		m.renderInputField("Title", m.titleInput.View(), m.focused == ceTitle, w),
		"",
		m.renderCalSelect(m.focused == ceCalendar, w),
		"",
		m.renderInputField("Location", m.locationInput.View(), m.focused == ceLocation, w),
		"",
		m.renderInputField("Description", m.descInput.View(), m.focused == ceDesc, w),
	}
	return strings.Join(rows, "\n")
}

func (m *createEventModel) renderInputField(label, content string, focused bool, w int) string {
	accent := lipgloss.Color(styles.ColorSecondaryFg)
	border := lipgloss.Color(styles.ColorSecondaryBorder)
	dim := lipgloss.Color(styles.ColorBorder)
	dimmer := lipgloss.Color("238")

	labelSty := lipgloss.NewStyle().Foreground(dim)
	borderColor := dimmer
	if focused {
		labelSty = lipgloss.NewStyle().Foreground(accent).Bold(true)
		borderColor = border
	}

	box := lipgloss.NewStyle().
		BorderLeft(true).
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(borderColor).
		PaddingLeft(1).
		Width(w - 2)

	return labelSty.Render(label) + "\n" + box.Render(content)
}

func (m *createEventModel) renderCalSelect(focused bool, w int) string {
	accent := lipgloss.Color(styles.ColorSecondaryFg)
	border := lipgloss.Color(styles.ColorSecondaryBorder)
	dim := lipgloss.Color(styles.ColorBorder)
	dimmer := lipgloss.Color("238")

	labelSty := lipgloss.NewStyle().Foreground(dim)
	borderColor := dimmer
	if focused {
		labelSty = lipgloss.NewStyle().Foreground(accent).Bold(true)
		borderColor = border
	}

	var content string
	if len(m.calOpts) == 0 {
		content = lipgloss.NewStyle().Foreground(dimmer).Render("(no calendars)")
	} else {
		name := m.calOpts[m.calIdx].name
		if focused {
			upSty := lipgloss.NewStyle().Foreground(dimmer)
			downSty := lipgloss.NewStyle().Foreground(dimmer)
			if m.calIdx > 0 {
				upSty = lipgloss.NewStyle().Foreground(accent)
			}
			if m.calIdx < len(m.calOpts)-1 {
				downSty = lipgloss.NewStyle().Foreground(accent)
			}
			content = upSty.Render("▲") + " " +
				lipgloss.NewStyle().Foreground(accent).Bold(true).Render(name) +
				" " + downSty.Render("▼")
		} else {
			content = lipgloss.NewStyle().Foreground(dim).Render(name)
		}
	}

	box := lipgloss.NewStyle().
		BorderLeft(true).
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(borderColor).
		PaddingLeft(1).
		Width(w - 2)

	return labelSty.Render("Calendar") + "\n" + box.Render(content)
}

func (m *createEventModel) renderRight() string {
	dim := lipgloss.Color(styles.ColorBorder)
	accent := lipgloss.Color(styles.ColorSecondaryFg)

	startFocused := m.focused == ceStartCal || m.focused == ceStartHr || m.focused == ceStartMin
	endFocused := m.focused == ceEndCal || m.focused == ceEndHr || m.focused == ceEndMin

	startLabelSty := lipgloss.NewStyle().Foreground(dim)
	if startFocused {
		startLabelSty = lipgloss.NewStyle().Foreground(accent).Bold(true)
	}
	endLabelSty := lipgloss.NewStyle().Foreground(dim)
	if endFocused {
		endLabelSty = lipgloss.NewStyle().Foreground(accent).Bold(true)
	}

	divider := lipgloss.NewStyle().Foreground(lipgloss.Color("238")).
		Render(strings.Repeat("─", ceRightW))

	startSection := strings.Join([]string{
		startLabelSty.Render("Start"),
		renderCalGrid(m.startPicker, m.focused == ceStartCal),
		m.renderTimePicker(m.startHr, m.startMin, m.focused == ceStartHr, m.focused == ceStartMin),
	}, "\n")

	endSection := strings.Join([]string{
		endLabelSty.Render("End"),
		renderCalGrid(m.endPicker, m.focused == ceEndCal),
		m.renderTimePicker(m.endHr, m.endMin, m.focused == ceEndHr, m.focused == ceEndMin),
	}, "\n")

	return strings.Join([]string{startSection, "", divider, "", endSection}, "\n")
}

func (m *createEventModel) renderTimePicker(hr, min int, hrFocused, minFocused bool) string {
	accent := lipgloss.Color(styles.ColorSecondaryFg)
	border := lipgloss.Color(styles.ColorSecondaryBorder)
	dim := lipgloss.Color(styles.ColorBorder)

	hrSty := lipgloss.NewStyle().Foreground(dim)
	minSty := lipgloss.NewStyle().Foreground(dim)
	upSty := lipgloss.NewStyle().Foreground(lipgloss.Color("238"))
	downSty := lipgloss.NewStyle().Foreground(lipgloss.Color("238"))

	if hrFocused {
		hrSty = lipgloss.NewStyle().
			Foreground(lipgloss.Color(styles.ColorBlack)).
			Background(border).Bold(true)
		upSty = lipgloss.NewStyle().Foreground(accent)
		downSty = lipgloss.NewStyle().Foreground(accent)
	} else if minFocused {
		minSty = lipgloss.NewStyle().
			Foreground(lipgloss.Color(styles.ColorBlack)).
			Background(border).Bold(true)
		upSty = lipgloss.NewStyle().Foreground(accent)
		downSty = lipgloss.NewStyle().Foreground(accent)
	}

	sepSty := lipgloss.NewStyle().Foreground(dim)

	// "  ▲     ▲  "
	// "  HH  :  MM  "
	// "  ▼     ▼  "
	hrStr := fmt.Sprintf("%02d", hr)
	minStr := fmt.Sprintf("%02d", min)

	line1 := fmt.Sprintf("  %s     %s", upSty.Render("▲"), upSty.Render("▲"))
	line2 := fmt.Sprintf("  %s %s %s", hrSty.Render(hrStr), sepSty.Render(":"), minSty.Render(minStr))
	line3 := fmt.Sprintf("  %s     %s", downSty.Render("▼"), downSty.Render("▼"))

	return strings.Join([]string{line1, line2, line3}, "\n")
}

func (m *createEventModel) renderSubmit() string {
	focused := m.focused == ceSubmit
	if focused {
		return lipgloss.NewStyle().
			Foreground(lipgloss.Color(styles.ColorBlack)).
			Background(lipgloss.Color(styles.ColorSecondaryFg)).
			Bold(true).
			Padding(0, 2).
			Render("  Create Event  ")
	}
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color(styles.ColorBorder)).
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("238")).
		Padding(0, 2).
		Render("  Create Event  ")
}

func (m *createEventModel) renderHelp() string {
	key := lipgloss.NewStyle().Foreground(lipgloss.Color(styles.ColorSecondaryBorder))
	dim := lipgloss.NewStyle().Foreground(lipgloss.Color("238"))

	base := key.Render("tab") + dim.Render("/") + key.Render("enter") + dim.Render(" next · ") +
		key.Render("shift+tab") + dim.Render(" prev · ") +
		key.Render("esc") + dim.Render(" cancel")

	var extra string
	switch m.focused {
	case ceStartCal, ceEndCal:
		extra = dim.Render("  ·  ") +
			key.Render("←→") + dim.Render(" day · ") +
			key.Render("↑↓") + dim.Render(" week · ") +
			key.Render("[]") + dim.Render(" month")
	case ceStartHr, ceEndHr, ceStartMin, ceEndMin:
		extra = dim.Render("  ·  ") + key.Render("↑↓") + dim.Render(" change value")
	case ceCalendar:
		extra = dim.Render("  ·  ") + key.Render("↑↓") + dim.Render(" cycle calendars")
	}

	return base + extra
}
