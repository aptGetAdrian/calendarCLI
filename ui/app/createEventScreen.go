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
//
// One stop per *concept*, not per digit: a date+time is a single stop whose
// segments are walked with ←→, so creating an event is 8 tab presses at most
// and usually 3 (title → duration → submit).

type ceField int

const (
	ceTitle ceField = iota
	ceCalendar
	ceLocation
	ceDesc
	ceStart
	ceEnd
	ceDuration
	ceSubmit
	ceFieldCount
)

// dtSeg is the cursor position inside a date+time value.
type dtSeg int

const (
	segDay dtSeg = iota
	segMonth
	segYear
	segHour
	segMin
	segCount
)

// ─── layout ───────────────────────────────────────────────────────────────────
//
// The form is a single stack of one-line rows. The month grid, when it fits,
// is joined onto the right of those same rows — so it costs *zero* extra
// height and switching between the narrow and wide layout never changes where
// anything sits vertically. That is what keeps the view from jumping around.
//
//	  ┌ formW ────────────────────┐ gap ┌ gridW ─┐
//	  Title       ┃ Team standup         ◀ Start · Aug 2026 ▶
//	  Calendar    ┃ ‹ Personal › 1/4     Mo Tu We Th Fr Sa Su
//	  ...

const (
	ceTitleBarH   = 3  // height of the rounded title bar
	ceIndent      = 2  // left gutter before the label
	ceLabelW      = 12 // label column
	ceBarW        = 2  // "┃ "
	ceRowOverhead = ceIndent + ceLabelW + ceBarW
	ceGridW       = 21 // 7 day columns × 3 chars
	ceGridGap     = 3
	ceMinValueW   = 20 // below this the grid is dropped
	ceValueMax    = 44 // the value column stops growing here
	ceMaxContentW = 96 // hard cap on total width
)

var ceDurations = []struct {
	label string
	d     time.Duration
}{
	{"15m", 15 * time.Minute},
	{"30m", 30 * time.Minute},
	{"1h", time.Hour},
	{"2h", 2 * time.Hour},
	{"4h", 4 * time.Hour},
	{"1d", 24 * time.Hour},
}

// ─── palette ──────────────────────────────────────────────────────────────────
// Matches the violet "Create Event" title bar instead of the old green.

var (
	ceAccent = lipgloss.Color(styles.ColorCreateEventBorder)
	ceText   = lipgloss.Color("252")
	ceDim    = lipgloss.Color("245")
	ceFaint  = lipgloss.Color("240")
	ceToday  = lipgloss.Color(styles.ColorWarning)
	ceErr    = lipgloss.Color(styles.ColorError)
	ceInk    = lipgloss.Color(styles.ColorBlack)
)

func ceSelected() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(ceInk).Background(ceAccent).Bold(true)
}

// ─── model ────────────────────────────────────────────────────────────────────

type calOpt struct{ name, id string }

type createEventModel struct {
	service *calendar.Service
	state   AppState
	width   int
	height  int
	logger  *logger.Logger

	focused      ceField
	submitted    bool
	errMsg       string
	scrollOffset int

	titleInput    textinput.Model
	locationInput textinput.Model
	descInput     textinput.Model

	calOpts []calOpt
	calIdx  int

	start    time.Time
	end      time.Time
	startSeg dtSeg
	endSeg   dtSeg
}

func newCreateEventModel(service *calendar.Service, state AppState, width, height int, log *logger.Logger) *createEventModel {
	start := time.Now().Add(time.Hour).Truncate(15 * time.Minute)

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

	m := &createEventModel{
		service: service,
		state:   state,
		width:   width,
		height:  height,
		logger:  log,
		focused: ceTitle,
		calOpts: calOpts,
		calIdx:  calIdx,
		start:   start,
		end:     start.Add(time.Hour),
	}

	mkInput := func(placeholder string) textinput.Model {
		ti := textinput.New()
		ti.Placeholder = placeholder
		ti.Prompt = ""
		ti.Width = m.valueW() - 1
		ti.TextStyle = lipgloss.NewStyle().Foreground(ceText)
		ti.PlaceholderStyle = lipgloss.NewStyle().Foreground(ceFaint)
		ti.Cursor.Style = lipgloss.NewStyle().Foreground(ceAccent)
		ti.Blur()
		return ti
	}

	m.titleInput = mkInput("Team standup")
	m.titleInput.Focus()
	m.locationInput = mkInput("Room / meet link")
	m.descInput = mkInput("What's this about?")

	return m
}

// ─── width math ───────────────────────────────────────────────────────────────

func (m *createEventModel) layoutW() int {
	if m.width > ceMaxContentW {
		return ceMaxContentW
	}
	if m.width < 1 {
		return 1
	}
	return m.width
}

// showGrid reports whether the month preview fits beside the form without
// squeezing the value column below a usable width.
func (m *createEventModel) showGrid() bool {
	return m.layoutW()-ceGridW-ceGridGap-ceRowOverhead >= ceMinValueW
}

// valueW is capped so the form keeps a readable shape on a very wide terminal
// instead of stretching a text input across the whole screen.
func (m *createEventModel) valueW() int {
	w := m.layoutW() - ceRowOverhead
	if m.showGrid() {
		w -= ceGridW + ceGridGap
	}
	if w > ceValueMax {
		w = ceValueMax
	}
	if w < 8 {
		w = 8
	}
	return w
}

func (m *createEventModel) formW() int { return ceRowOverhead + m.valueW() }

func (m *createEventModel) syncInputWidths() {
	w := m.valueW() - 1
	m.titleInput.Width = w
	m.locationInput.Width = w
	m.descInput.Width = w
}

// ─── date arithmetic ──────────────────────────────────────────────────────────

func calMonthDays(y int, mo time.Month) int {
	return time.Date(y, mo+1, 0, 0, 0, 0, 0, time.Local).Day()
}

// shiftMonths moves by whole months, clamping the day so Jan 31 → Feb 28
// instead of Go's default overflow into March.
func shiftMonths(t time.Time, n int) time.Time {
	y, mo := t.Year(), int(t.Month())-1+n
	y += mo / 12
	mo %= 12
	if mo < 0 {
		mo += 12
		y--
	}
	day := t.Day()
	if mx := calMonthDays(y, time.Month(mo+1)); day > mx {
		day = mx
	}
	return time.Date(y, time.Month(mo+1), day, t.Hour(), t.Minute(), 0, 0, t.Location())
}

// adjustDT bumps a single segment. Hours and minutes wrap inside the day so
// nudging the clock never silently moves the date out from under you.
func adjustDT(t time.Time, seg dtSeg, delta int) time.Time {
	switch seg {
	case segDay:
		return t.AddDate(0, 0, delta)
	case segMonth:
		return shiftMonths(t, delta)
	case segYear:
		return shiftMonths(t, delta*12)
	case segHour:
		h := (t.Hour() + delta + 24) % 24
		return time.Date(t.Year(), t.Month(), t.Day(), h, t.Minute(), 0, 0, t.Location())
	case segMin:
		mi := (t.Minute() + delta*5 + 60) % 60
		return time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), mi, 0, 0, t.Location())
	}
	return t
}

func humanDur(d time.Duration) string {
	if d < 0 {
		return "−" + humanDur(-d) // kept short so it never overflows the row
	}
	total := int(d.Minutes())
	days, rem := total/1440, total%1440
	h, mi := rem/60, rem%60
	switch {
	case days > 0 && h > 0:
		return fmt.Sprintf("%dd %dh", days, h)
	case days > 0:
		return fmt.Sprintf("%dd", days)
	case h > 0 && mi > 0:
		return fmt.Sprintf("%dh %dm", h, mi)
	case h > 0:
		return fmt.Sprintf("%dh", h)
	default:
		return fmt.Sprintf("%dm", mi)
	}
}

// durIdx returns the preset matching the current span, or -1 when custom.
func (m *createEventModel) durIdx() int {
	d := m.end.Sub(m.start)
	for i, p := range ceDurations {
		if p.d == d {
			return i
		}
	}
	return -1
}

// ─── Bubble Tea ───────────────────────────────────────────────────────────────

func (m *createEventModel) Init() tea.Cmd { return textinput.Blink }

func (m *createEventModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case sizedMsg:
		m.width, m.height = msg.width, msg.height
		m.syncInputWidths()
		m.updateScroll()
		return m, nil
	case tea.KeyMsg:
		return m.handleKey(msg)
	}
	return m, nil
}

func (m *createEventModel) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		return m, func() tea.Msg { return NavigateTo{Screen: ui.MainMenuScreen} }
	case "ctrl+s":
		return m, m.submit()
	case "tab":
		m.moveFocus(1)
		return m, textinput.Blink
	case "shift+tab":
		m.moveFocus(-1)
		return m, textinput.Blink
	case "enter":
		if m.focused == ceSubmit {
			return m, m.submit()
		}
		m.moveFocus(1)
		return m, textinput.Blink
	}

	// Text fields swallow everything else while focused.
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

	switch msg.String() {
	case "left", "h":
		m.horizontal(-1)
	case "right", "l":
		m.horizontal(1)
	case "up", "k":
		m.vertical(1)
	case "down", "j":
		m.vertical(-1)
	case "[", "pgup":
		m.shiftFocusedDate(func(t time.Time) time.Time { return shiftMonths(t, -1) })
	case "]", "pgdown":
		m.shiftFocusedDate(func(t time.Time) time.Time { return shiftMonths(t, 1) })
	case "t":
		m.jumpToToday()
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
	case ceStart:
		m.startSeg = segDay
	case ceEnd:
		m.endSeg = segDay
	}
	m.updateScroll()
}

// horizontal: ←→ walks date segments, cycles calendars, and picks presets.
func (m *createEventModel) horizontal(dir int) {
	switch m.focused {
	case ceCalendar:
		m.cycleCalendar(dir)
	case ceStart:
		m.startSeg = clampSeg(int(m.startSeg) + dir)
	case ceEnd:
		m.endSeg = clampSeg(int(m.endSeg) + dir)
	case ceDuration:
		m.cycleDuration(dir)
	}
}

// vertical: ↑↓ changes the value under the cursor. dir is +1 for "up".
func (m *createEventModel) vertical(dir int) {
	switch m.focused {
	case ceCalendar:
		m.cycleCalendar(-dir)
	case ceStart:
		m.setStartKeepingDuration(adjustDT(m.start, m.startSeg, dir))
	case ceEnd:
		m.end = adjustDT(m.end, m.endSeg, dir)
	case ceDuration:
		m.cycleDuration(dir)
	}
}

func clampSeg(i int) dtSeg {
	if i < 0 {
		return 0
	}
	if i >= int(segCount) {
		return segCount - 1
	}
	return dtSeg(i)
}

func (m *createEventModel) cycleCalendar(dir int) {
	if len(m.calOpts) == 0 {
		return
	}
	m.calIdx = (m.calIdx + dir + len(m.calOpts)) % len(m.calOpts)
}

// cycleDuration snaps the end time to the next/previous preset. From a custom
// span it lands on the nearest preset in the direction of travel.
func (m *createEventModel) cycleDuration(dir int) {
	idx := m.durIdx()
	if idx == -1 {
		cur := m.end.Sub(m.start)
		if dir > 0 {
			idx = len(ceDurations) - 1
			for i, p := range ceDurations {
				if p.d > cur {
					idx = i
					break
				}
			}
			m.end = m.start.Add(ceDurations[idx].d)
			return
		}
		idx = 0
		for i := len(ceDurations) - 1; i >= 0; i-- {
			if ceDurations[i].d < cur {
				idx = i
				break
			}
		}
		m.end = m.start.Add(ceDurations[idx].d)
		return
	}
	idx += dir
	if idx < 0 {
		idx = 0
	}
	if idx >= len(ceDurations) {
		idx = len(ceDurations) - 1
	}
	m.end = m.start.Add(ceDurations[idx].d)
}

// setStartKeepingDuration is the reason the "end is before start" error almost
// never fires any more: moving the start drags the end along with it.
func (m *createEventModel) setStartKeepingDuration(t time.Time) {
	d := m.end.Sub(m.start)
	m.start = t
	if d > 0 {
		m.end = m.start.Add(d)
	}
}

func (m *createEventModel) shiftFocusedDate(f func(time.Time) time.Time) {
	switch m.focused {
	case ceStart:
		m.setStartKeepingDuration(f(m.start))
	case ceEnd:
		m.end = f(m.end)
	}
}

func (m *createEventModel) jumpToToday() {
	now := time.Now()
	at := func(t time.Time) time.Time {
		return time.Date(now.Year(), now.Month(), now.Day(), t.Hour(), t.Minute(), 0, 0, t.Location())
	}
	switch m.focused {
	case ceStart:
		m.setStartKeepingDuration(at(m.start))
	case ceEnd:
		m.end = at(m.end)
	}
}

func (m *createEventModel) submit() tea.Cmd {
	if strings.TrimSpace(m.titleInput.Value()) == "" {
		m.errMsg = "Title is required"
		m.focusField(ceTitle)
		return textinput.Blink
	}
	if !m.end.After(m.start) {
		m.errMsg = "End must be after start"
		m.focusField(ceEnd)
		return nil
	}

	calID := ""
	if len(m.calOpts) > 0 {
		calID = m.calOpts[m.calIdx].id
	}
	title := m.titleInput.Value()
	location := m.locationInput.Value()
	desc := m.descInput.Value()
	start, end := m.start, m.end
	m.submitted = true

	return func() tea.Msg {
		if _, err := m.service.CreateEvent(calID, title, location, desc, start, end); err != nil {
			m.logger.Error("Error creating event: %s", err)
			return NavigateTo{Screen: ui.MainMenuScreen}
		}
		return eventCreatedMsg{}
	}
}

func (m *createEventModel) focusField(f ceField) {
	m.titleInput.Blur()
	m.locationInput.Blur()
	m.descInput.Blur()
	m.focused = f
	switch f {
	case ceTitle:
		m.titleInput.Focus()
	case ceLocation:
		m.locationInput.Focus()
	case ceDesc:
		m.descInput.Focus()
	}
	m.updateScroll()
}

// ─── rows ─────────────────────────────────────────────────────────────────────
//
// The body is built as a flat list of rows, each tagged with the field it
// belongs to. Scroll positions are then *measured* from this list rather than
// hard-coded, so a rendering change can never desync the viewport again.

type ceRow struct {
	line  string
	field ceField // ceFieldCount = not focusable
}

func (m *createEventModel) rows() []ceRow {
	form := []ceRow{
		{m.fieldRow("Title", m.titleInput.View(), m.focused == ceTitle), ceTitle},
		{m.fieldRow("Calendar", m.calendarValue(), m.focused == ceCalendar), ceCalendar},
		{m.fieldRow("Location", m.locationInput.View(), m.focused == ceLocation), ceLocation},
		{m.fieldRow("Description", m.descInput.View(), m.focused == ceDesc), ceDesc},
		{"", ceFieldCount},
		{m.fieldRow("Start", m.startValue(), m.focused == ceStart), ceStart},
		{m.fieldRow("End", m.endValue(), m.focused == ceEnd), ceEnd},
		{m.fieldRow("Duration", m.durationValue(), m.focused == ceDuration), ceDuration},
	}

	if m.showGrid() {
		grid := m.renderGridPanel()
		pad := lipgloss.NewStyle().Width(m.formW())
		for i := range form {
			g := ""
			if i < len(grid) {
				g = grid[i]
			}
			form[i].line = pad.Render(form[i].line) + g
		}
	}

	out := make([]ceRow, 0, len(form)+8)
	out = append(out, ceRow{"", ceFieldCount})
	out = append(out, form...)
	out = append(out, ceRow{"", ceFieldCount})
	if m.errMsg != "" {
		out = append(out,
			ceRow{lipgloss.NewStyle().Foreground(ceErr).Render("  ⚠  " + m.errMsg), ceFieldCount},
			ceRow{"", ceFieldCount})
	}
	out = append(out,
		ceRow{m.renderSubmit(), ceSubmit},
		ceRow{"", ceFieldCount},
		ceRow{m.renderHelp(), ceFieldCount})
	return out
}

// lineOf returns the row index of a field, or 0 when it isn't rendered.
func lineOf(rows []ceRow, f ceField) int {
	for i, r := range rows {
		if r.field == f {
			return i
		}
	}
	return 0
}

// ─── sub-renderers ────────────────────────────────────────────────────────────

func (m *createEventModel) fieldRow(label, value string, focused bool) string {
	labelSty := lipgloss.NewStyle().Foreground(ceDim).Width(ceLabelW)
	bar := lipgloss.NewStyle().Foreground(ceFaint).Render("│")
	if focused {
		labelSty = labelSty.Foreground(ceAccent).Bold(true)
		bar = lipgloss.NewStyle().Foreground(ceAccent).Render("┃")
	}
	line := strings.Repeat(" ", ceIndent) + labelSty.Render(label) + bar + " " +
		lipgloss.NewStyle().MaxWidth(m.valueW()).Render(value)
	return lipgloss.NewStyle().MaxWidth(m.formW()).Render(line)
}

func (m *createEventModel) calendarValue() string {
	if len(m.calOpts) == 0 {
		return lipgloss.NewStyle().Foreground(ceFaint).Render("(no calendars)")
	}
	name := m.calOpts[m.calIdx].name
	counter := lipgloss.NewStyle().Foreground(ceFaint).
		Render(fmt.Sprintf("  %d/%d", m.calIdx+1, len(m.calOpts)))

	if m.focused != ceCalendar {
		return lipgloss.NewStyle().Foreground(ceText).Render(name) + counter
	}
	arrow := lipgloss.NewStyle().Foreground(ceAccent)
	return arrow.Render("‹ ") +
		lipgloss.NewStyle().Foreground(ceAccent).Bold(true).Render(name) +
		arrow.Render(" ›") + counter
}

// renderDateTime prints "Mon 25 Aug 2026  14:30", highlighting the active
// segment when the field owns the focus.
func renderDateTime(t time.Time, seg dtSeg, focused bool) string {
	parts := [segCount]string{
		fmt.Sprintf("%02d", t.Day()),
		t.Month().String()[:3],
		fmt.Sprintf("%d", t.Year()),
		fmt.Sprintf("%02d", t.Hour()),
		fmt.Sprintf("%02d", t.Minute()),
	}

	base := lipgloss.NewStyle().Foreground(ceText)
	if !focused {
		base = base.Foreground(ceDim)
	}
	sel := ceSelected()

	seg2s := func(i dtSeg) string {
		if focused && i == seg {
			return sel.Render(parts[i])
		}
		return base.Render(parts[i])
	}

	wd := lipgloss.NewStyle().Foreground(ceFaint).Render(t.Weekday().String()[:3])
	colon := base.Render(":")
	return wd + " " + seg2s(segDay) + " " + seg2s(segMonth) + " " + seg2s(segYear) +
		"  " + seg2s(segHour) + colon + seg2s(segMin)
}

func (m *createEventModel) startValue() string {
	return renderDateTime(m.start, m.startSeg, m.focused == ceStart)
}

func (m *createEventModel) endValue() string {
	d := m.end.Sub(m.start)
	tagSty := lipgloss.NewStyle().Foreground(ceFaint)
	if d <= 0 {
		tagSty = lipgloss.NewStyle().Foreground(ceErr).Bold(true)
	}
	return renderDateTime(m.end, m.endSeg, m.focused == ceEnd) +
		tagSty.Render("  · "+humanDur(d))
}

func (m *createEventModel) durationValue() string {
	cur := m.durIdx()
	sel := ceSelected()
	off := lipgloss.NewStyle().Foreground(ceDim)
	if m.focused != ceDuration {
		off = lipgloss.NewStyle().Foreground(ceFaint)
	}

	chips := make([]string, len(ceDurations))
	for i, p := range ceDurations {
		if i == cur {
			chips[i] = sel.Render(" " + p.label + " ")
		} else {
			chips[i] = off.Render(" " + p.label + " ")
		}
	}
	return strings.Join(chips, "")
}

// renderGridPanel returns exactly 8 lines of ceGridW width: a header, the
// weekday row, and six week rows. It is a read-only preview of whichever date
// field is in focus — the arrows always edit the field, never the grid, so
// there is no ambiguity about what a keypress does.
func (m *createEventModel) renderGridPanel() []string {
	t, label, live := m.start, "Start", m.focused == ceStart
	if m.focused == ceEnd {
		t, label, live = m.end, "End", true
	}

	now := time.Now()
	todayY, todayM, todayD := now.Year(), now.Month(), now.Day()

	headSty := lipgloss.NewStyle().Foreground(ceDim)
	daySty := lipgloss.NewStyle().Foreground(ceDim)
	selSty := lipgloss.NewStyle().Foreground(ceText).Underline(true)
	if live {
		headSty = lipgloss.NewStyle().Foreground(ceAccent).Bold(true)
		daySty = lipgloss.NewStyle().Foreground(ceText)
		selSty = ceSelected()
	}
	dowSty := lipgloss.NewStyle().Foreground(ceFaint)
	todaySty := lipgloss.NewStyle().Foreground(ceToday).Bold(true)

	title := fmt.Sprintf("%s · %s %d", label, t.Month().String()[:3], t.Year())
	lines := []string{
		headSty.Render(centerIn(title, ceGridW)),
		dowSty.Render("Mo Tu We Th Fr Sa Su "),
	}

	wd := int(time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, time.Local).Weekday())
	if wd == 0 {
		wd = 7
	}
	wd--

	total := calMonthDays(t.Year(), t.Month())
	day := 1
	for row := 0; row < 6; row++ { // always 6 rows → stable height
		var b strings.Builder
		for col := 0; col < 7; col++ {
			if (row == 0 && col < wd) || day > total {
				b.WriteString("   ")
				continue
			}
			cell := fmt.Sprintf("%2d ", day)
			switch {
			case day == t.Day():
				b.WriteString(selSty.Render(cell))
			case day == todayD && t.Month() == todayM && t.Year() == todayY:
				b.WriteString(todaySty.Render(cell))
			default:
				b.WriteString(daySty.Render(cell))
			}
			day++
		}
		lines = append(lines, b.String())
	}
	return lines
}

func (m *createEventModel) renderSubmit() string {
	label := " Create event "
	var btn string
	if m.focused == ceSubmit {
		btn = ceSelected().Render("▸" + label)
	} else {
		btn = lipgloss.NewStyle().Foreground(ceDim).
			BorderLeft(true).BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(ceFaint).Render(label)
	}
	hint := lipgloss.NewStyle().Foreground(ceFaint).Render("   ctrl+s from anywhere")
	return lipgloss.NewStyle().MaxWidth(m.layoutW()).
		Render(strings.Repeat(" ", ceIndent) + btn + hint)
}

func (m *createEventModel) renderHelp() string {
	key := lipgloss.NewStyle().Foreground(ceAccent)
	dim := lipgloss.NewStyle().Foreground(ceFaint)
	k := func(s, desc string) string { return key.Render(s) + dim.Render(" "+desc) }

	// Ordered by usefulness: whatever the focused field does comes first, so a
	// narrow terminal drops the generic hints rather than the relevant ones.
	var parts []string
	switch m.focused {
	case ceCalendar:
		parts = []string{k("←→", "calendar")}
	case ceStart, ceEnd:
		parts = []string{k("←→", "part"), k("↑↓", "adjust")}
	case ceDuration:
		parts = []string{k("←→", "preset")}
	case ceSubmit:
		parts = []string{k("enter", "create")}
	}
	parts = append(parts, k("tab", "next"), k("esc", "back"))
	switch m.focused {
	case ceStart, ceEnd:
		parts = append(parts, k("[]", "month"), k("t", "today"))
	default:
		parts = append(parts, k("⇧tab", "prev"))
	}

	sep := dim.Render(" · ")
	line := strings.Repeat(" ", ceIndent)
	for i, p := range parts {
		next := p
		if i > 0 {
			next = sep + p
		}
		if lipgloss.Width(line)+lipgloss.Width(next) > m.layoutW() {
			break
		}
		line += next
	}
	return line
}

// ─── scroll ───────────────────────────────────────────────────────────────────
//
// Scrolling is a safety net for terminals shorter than the form, not the
// normal mode of operation: at the 80×26 minimum the whole form fits.

func (m *createEventModel) availLines() int {
	if a := m.height - ceTitleBarH; a > 0 {
		return a
	}
	return 1
}

func (m *createEventModel) updateScroll() {
	m.reveal(m.rows(), m.availLines())
}

// reveal scrolls the minimum amount needed to bring the focused row into view.
func (m *createEventModel) reveal(rows []ceRow, avail int) {
	target := lineOf(rows, m.focused)
	if target < m.scrollOffset {
		m.scrollOffset = target
	} else if target >= m.scrollOffset+avail {
		m.scrollOffset = target - avail + 1
	}
	m.clampScroll(len(rows), avail)
}

func (m *createEventModel) clampScroll(total, avail int) {
	if max := total - avail; m.scrollOffset > max {
		m.scrollOffset = max
	}
	if m.scrollOffset < 0 {
		m.scrollOffset = 0
	}
}

// ─── View ─────────────────────────────────────────────────────────────────────

func (m *createEventModel) View() string {
	if m.submitted {
		lines := make([]string, m.height)
		if m.height > 2 {
			lines[2] = lipgloss.NewStyle().PaddingLeft(ceIndent).
				Foreground(ceAccent).Render("Creating event…")
		}
		return strings.Join(lines, "\n")
	}

	rows := m.rows()
	avail := m.availLines()
	m.reveal(rows, avail) // authoritative: the focused row is always on screen

	lines := make([]string, len(rows))
	for i, r := range rows {
		lines[i] = r.line
	}

	return styles.CreateEventTtitle().Render("Create Event") + "\n" +
		clipAndPad(lines, m.scrollOffset, avail)
}
