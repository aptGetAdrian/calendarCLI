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
	ceStartCal // month grid; arrows navigate days
	ceStartHr  // up/down ±1 hour
	ceStartMin // up/down ±5 min
	ceEndCal
	ceEndHr
	ceEndMin
	ceSubmit
	ceFieldCount
)

// Layout constants.
// ceTitleBarH = 3 lines (rounded border) + 1 blank separator = 4.
// ceRightW    = 23 chars (calendar grid 21 + 2 padding).
// ceWideTh    = min width to use 2-column layout.
const (
	ceTitleBarH    = 4
	ceRightW       = 23
	ceWideTh       = 60
	ceMaxContentW  = 100 // cap for very wide terminals
)

// layoutWidth returns the effective content width, capped so the layout
// doesn't stretch absurdly on ultra-wide terminals (e.g. i3 docked pane).
func layoutWidth(w int) int {
	if w > ceMaxContentW {
		return ceMaxContentW
	}
	return w
}

// ─── body line positions ──────────────────────────────────────────────────────
//
// The "body" is everything rendered below the title bar.  Both layouts share
// the same scroll machinery; only the line numbers differ.
//
// 2-column body:
//   0         ""
//   1-27      panels (JoinHorizontal, always 27 lines)
//              left col  0-10  : Title(0-1) gap(2) Cal(3-4) gap(5) Loc(6-7) gap(8) Desc(9-10)
//              right col 0-26  : Start(0-11) gap(12) div(13) gap(14) End(15-26)
//   28        ""
//   29-31     submit   [+2 if errMsg]
//   32        ""       [+2 if errMsg]
//   33        help     [+2 if errMsg]
//
// 1-column body:
//   0         ""
//   1-11      left fields
//   12        ""
//   13-24     dates side-by-side (12 lines, always)
//   25        ""
//   26-28     submit   [+2 if errMsg]
//   29        ""       [+2 if errMsg]
//   30        help     [+2 if errMsg]

func (m *createEventModel) fieldScrollLine(f ceField) int {
	err := m.errMsg != ""
	if m.isTwoColumn() {
		switch f {
		case ceTitle:
			return 1
		case ceCalendar:
			return 4
		case ceLocation:
			return 7
		case ceDesc:
			return 10
		case ceStartCal, ceStartHr, ceStartMin:
			return 1 // top of right panel
		case ceEndCal, ceEndHr, ceEndMin:
			return 16 // end section inside right panel
		case ceSubmit:
			if err {
				return 31
			}
			return 29
		}
	} else {
		switch f {
		case ceTitle:
			return 1
		case ceCalendar:
			return 4
		case ceLocation:
			return 7
		case ceDesc:
			return 10
		case ceStartCal, ceStartHr, ceStartMin, ceEndCal, ceEndHr, ceEndMin:
			return 13
		case ceSubmit:
			if err {
				return 28
			}
			return 26
		}
	}
	return 0
}

func fieldScrollHeight(f ceField) int {
	switch f {
	case ceStartCal, ceStartHr, ceStartMin, ceEndCal, ceEndHr, ceEndMin:
		return 12
	case ceSubmit:
		return 3
	default:
		return 2
	}
}

// clipAndPad slices lines[offset:offset+avail] and pads with blank lines so
// the result is always exactly avail lines — this keeps the status bar pinned.
func clipAndPad(lines []string, offset, avail int) string {
	end := offset + avail
	if end > len(lines) {
		end = len(lines)
	}
	if offset > end {
		offset = end
	}
	out := make([]string, avail) // zero-value = ""
	copy(out, lines[offset:end])
	return strings.Join(out, "\n")
}

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
	if mx := calMonthDays(c.year, c.month); c.day > mx {
		c.day = mx
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
	l := (width - n) / 2
	return strings.Repeat(" ", l) + s + strings.Repeat(" ", width-n-l)
}

// renderCalGrid always emits exactly 6 week rows → stable 8-line height.
func renderCalGrid(c calPicker, focused bool) string {
	now := time.Now()
	todayY, todayM, todayD := now.Year(), now.Month(), now.Day()

	accent := lipgloss.Color(styles.ColorSecondaryFg)
	border := lipgloss.Color(styles.ColorSecondaryBorder)
	dim := lipgloss.Color(styles.ColorBorder)
	dimmer := lipgloss.Color("246")
	warn := lipgloss.Color(styles.ColorWarning)

	var arrowSty, monthSty, dowSty, daySty, todaySty, selSty lipgloss.Style
	if focused {
		arrowSty = lipgloss.NewStyle().Foreground(accent)
		monthSty = lipgloss.NewStyle().Foreground(accent).Bold(true)
		dowSty   = lipgloss.NewStyle().Foreground(dim)
		daySty   = lipgloss.NewStyle().Foreground(lipgloss.Color("250"))
		todaySty = lipgloss.NewStyle().Foreground(warn).Bold(true)
		selSty   = lipgloss.NewStyle().Foreground(lipgloss.Color(styles.ColorBlack)).Background(border).Bold(true)
	} else {
		arrowSty = lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
		monthSty = lipgloss.NewStyle().Foreground(dimmer)
		dowSty   = lipgloss.NewStyle().Foreground(lipgloss.Color("243"))
		daySty   = lipgloss.NewStyle().Foreground(dimmer)
		todaySty = lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
		selSty   = lipgloss.NewStyle().Foreground(lipgloss.Color("252")).Underline(true)
	}

	mStr := fmt.Sprintf("%s %d", c.month.String()[:3], c.year)
	header := arrowSty.Render("◀") + monthSty.Render(centerIn(mStr, 19)) + arrowSty.Render("▶")

	dows := []string{"Mo", "Tu", "We", "Th", "Fr", "Sa", "Su"}
	var dowRow string
	for _, d := range dows {
		dowRow += dowSty.Render(fmt.Sprintf("%-3s", d))
	}

	wd := int(time.Date(c.year, c.month, 1, 0, 0, 0, 0, time.Local).Weekday())
	if wd == 0 {
		wd = 7
	}
	wd--

	total := calMonthDays(c.year, c.month)
	blank := daySty.Render("   ")

	var gridRows []string
	day := 1
	for row := 0; row < 6; row++ { // always 6 rows → stable height
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
			switch {
			case day == c.day:
				rowStr += selSty.Render(cell)
			case day == todayD && c.month == todayM && c.year == todayY:
				rowStr += todaySty.Render(cell)
			default:
				rowStr += daySty.Render(cell)
			}
			day++
		}
		gridRows = append(gridRows, rowStr)
	}

	return strings.Join(append([]string{header, dowRow}, gridRows...), "\n")
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

	startPicker calPicker
	startHr     int
	startMin    int
	endPicker   calPicker
	endHr       int
	endMin      int
}

func (m *createEventModel) isTwoColumn() bool {
	return layoutWidth(m.width) >= ceWideTh && m.height >= ceTitleBarH+27
}

func ceLeftW(totalW int) int {
	if w := totalW - ceRightW - 3; w >= 20 {
		return w
	}
	return 20
}

func inputWidth(w int, twoCol bool) int {
	var iw int
	if twoCol {
		iw = ceLeftW(w) - 4
	} else {
		iw = w - 4
	}
	if iw < 10 {
		return 10
	}
	return iw
}

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

	lw := layoutWidth(width)
	twoCol := lw >= ceWideTh
	iw := inputWidth(lw, twoCol)

	mkInput := func(placeholder string) textinput.Model {
		ti := textinput.New()
		ti.Placeholder = placeholder
		ti.Width = iw
		ti.PromptStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(styles.ColorSecondaryBorder))
		ti.TextStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(styles.ColorWhite))
		ti.PlaceholderStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("243"))
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

// ─── scroll ───────────────────────────────────────────────────────────────────

func (m *createEventModel) updateScroll() {
	target := m.fieldScrollLine(m.focused)
	h := fieldScrollHeight(m.focused)
	avail := m.height - ceTitleBarH
	if avail < 1 {
		avail = 1
	}
	if target < m.scrollOffset {
		m.scrollOffset = target
	} else if target+h > m.scrollOffset+avail {
		m.scrollOffset = target + h - avail
	}
	if m.scrollOffset < 0 {
		m.scrollOffset = 0
	}
}

// ─── Bubble Tea ───────────────────────────────────────────────────────────────

func (m *createEventModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m *createEventModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case sizedMsg:
		m.width, m.height = msg.width, msg.height
		iw := inputWidth(layoutWidth(m.width), m.isTwoColumn())
		m.titleInput.Width = iw
		m.locationInput.Width = iw
		m.descInput.Width = iw
		m.updateScroll()
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

	// Text inputs consume remaining keys when focused
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
	m.updateScroll()
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
		m.updateScroll()
		return textinput.Blink
	}

	start := time.Date(m.startPicker.year, m.startPicker.month, m.startPicker.day,
		m.startHr, m.startMin, 0, 0, time.Local)
	end := time.Date(m.endPicker.year, m.endPicker.month, m.endPicker.day,
		m.endHr, m.endMin, 0, 0, time.Local)

	if !end.After(start) {
		m.errMsg = "End must be after start — fix the end date"
		m.titleInput.Blur()
		m.locationInput.Blur()
		m.descInput.Blur()
		m.focused = ceEndCal
		m.updateScroll()
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
//
// The title bar is always rendered at the top outside the scroll viewport.
// The body is clipped to exactly (m.height - ceTitleBarH) lines and padded
// with blank lines when the content is shorter — this keeps the root model's
// status bar pinned to the bottom of the terminal regardless of content height.

func (m *createEventModel) View() string {
	if m.submitted {
		// Fill the full height so the status bar stays at the bottom.
		lines := make([]string, m.height)
		if m.height > 2 {
			lines[2] = lipgloss.NewStyle().
				PaddingLeft(4).
				Foreground(lipgloss.Color(styles.ColorCreateEventFg)).
				Render("Creating event...")
		}
		return strings.Join(lines, "\n")
	}

	titleBar := styles.CreateEventTtitle().Render("Create Event")

	body := m.buildBody()
	lines := strings.Split(body, "\n")

	avail := m.height - ceTitleBarH
	if avail < 1 {
		avail = 1
	}

	// Clamp scroll offset so it stays within valid range.
	if maxOff := len(lines) - avail; m.scrollOffset > maxOff {
		if maxOff < 0 {
			m.scrollOffset = 0
		} else {
			m.scrollOffset = maxOff
		}
	}
	if m.scrollOffset < 0 {
		m.scrollOffset = 0
	}

	return titleBar + "\n" + clipAndPad(lines, m.scrollOffset, avail)
}

func (m *createEventModel) buildBody() string {
	if m.isTwoColumn() {
		return m.buildTwoColBody()
	}
	return m.buildOneColBody()
}

func (m *createEventModel) buildTwoColBody() string {
	leftW := ceLeftW(layoutWidth(m.width))
	leftBox := lipgloss.NewStyle().Width(leftW).PaddingRight(3)
	panels := lipgloss.JoinHorizontal(lipgloss.Top,
		leftBox.Render(m.renderLeft(leftW)),
		m.renderRight(),
	)

	parts := []string{"", panels, ""}
	if m.errMsg != "" {
		parts = append(parts,
			lipgloss.NewStyle().Foreground(lipgloss.Color(styles.ColorError)).Render("  ⚠  "+m.errMsg),
			"")
	}
	parts = append(parts, m.renderSubmit(), "", m.renderHelp())
	return strings.Join(parts, "\n")
}

func (m *createEventModel) buildOneColBody() string {
	parts := []string{"", m.renderLeft(layoutWidth(m.width)), "", m.renderDatesSideBySide(), ""}
	if m.errMsg != "" {
		parts = append(parts,
			lipgloss.NewStyle().Foreground(lipgloss.Color(styles.ColorError)).Render("  ⚠  "+m.errMsg),
			"")
	}
	parts = append(parts, m.renderSubmit(), "", m.renderHelp())
	return strings.Join(parts, "\n")
}

// ─── sub-renderers ────────────────────────────────────────────────────────────

func (m *createEventModel) renderLeft(w int) string {
	return strings.Join([]string{
		m.renderInputField("Title", m.titleInput.View(), m.focused == ceTitle, w),
		"",
		m.renderCalSelect(m.focused == ceCalendar, w),
		"",
		m.renderInputField("Location", m.locationInput.View(), m.focused == ceLocation, w),
		"",
		m.renderInputField("Description", m.descInput.View(), m.focused == ceDesc, w),
	}, "\n")
}

func (m *createEventModel) renderInputField(label, content string, focused bool, w int) string {
	accent := lipgloss.Color(styles.ColorSecondaryFg)
	bord := lipgloss.Color(styles.ColorSecondaryBorder)
	dim := lipgloss.Color(styles.ColorBorder)

	labelSty := lipgloss.NewStyle().Foreground(dim)
	borderColor := lipgloss.Color("240")
	if focused {
		labelSty = lipgloss.NewStyle().Foreground(accent).Bold(true)
		borderColor = bord
	}

	box := lipgloss.NewStyle().
		BorderLeft(true).BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(borderColor).PaddingLeft(1).Width(w - 2)

	return labelSty.Render(label) + "\n" + box.Render(content)
}

func (m *createEventModel) renderCalSelect(focused bool, w int) string {
	accent := lipgloss.Color(styles.ColorSecondaryFg)
	bord := lipgloss.Color(styles.ColorSecondaryBorder)
	dim := lipgloss.Color(styles.ColorBorder)

	labelSty := lipgloss.NewStyle().Foreground(dim)
	borderColor := lipgloss.Color("240")
	if focused {
		labelSty = lipgloss.NewStyle().Foreground(accent).Bold(true)
		borderColor = bord
	}

	muted := lipgloss.Color("244")
	var content string
	if len(m.calOpts) == 0 {
		content = lipgloss.NewStyle().Foreground(muted).Render("(no calendars)")
	} else {
		name := m.calOpts[m.calIdx].name
		if focused {
			upSty := lipgloss.NewStyle().Foreground(muted)
			downSty := lipgloss.NewStyle().Foreground(muted)
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
		BorderLeft(true).BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(borderColor).PaddingLeft(1).Width(w - 2)

	return labelSty.Render("Calendar") + "\n" + box.Render(content)
}

// renderRight: start then end pickers stacked — 2-column mode right panel (27 lines).
func (m *createEventModel) renderRight() string {
	dim := lipgloss.Color(styles.ColorBorder)
	accent := lipgloss.Color(styles.ColorSecondaryFg)

	startFocused := m.focused == ceStartCal || m.focused == ceStartHr || m.focused == ceStartMin
	endFocused := m.focused == ceEndCal || m.focused == ceEndHr || m.focused == ceEndMin

	startLbl := lipgloss.NewStyle().Foreground(dim)
	if startFocused {
		startLbl = lipgloss.NewStyle().Foreground(accent).Bold(true)
	}
	endLbl := lipgloss.NewStyle().Foreground(dim)
	if endFocused {
		endLbl = lipgloss.NewStyle().Foreground(accent).Bold(true)
	}

	divider := lipgloss.NewStyle().Foreground(lipgloss.Color("243")).Render(strings.Repeat("─", ceRightW))

	startSec := strings.Join([]string{
		startLbl.Render("Start"),
		renderCalGrid(m.startPicker, m.focused == ceStartCal),
		m.renderTimePicker(m.startHr, m.startMin, m.focused == ceStartHr, m.focused == ceStartMin),
	}, "\n")

	endSec := strings.Join([]string{
		endLbl.Render("End"),
		renderCalGrid(m.endPicker, m.focused == ceEndCal),
		m.renderTimePicker(m.endHr, m.endMin, m.focused == ceEndHr, m.focused == ceEndMin),
	}, "\n")

	return strings.Join([]string{startSec, "", divider, "", endSec}, "\n")
}

// renderDatesSideBySide: start and end pickers side by side — 1-column mode (12 lines).
func (m *createEventModel) renderDatesSideBySide() string {
	dim := lipgloss.Color(styles.ColorBorder)
	accent := lipgloss.Color(styles.ColorSecondaryFg)

	startFocused := m.focused == ceStartCal || m.focused == ceStartHr || m.focused == ceStartMin
	endFocused := m.focused == ceEndCal || m.focused == ceEndHr || m.focused == ceEndMin

	startLbl := lipgloss.NewStyle().Foreground(dim)
	if startFocused {
		startLbl = lipgloss.NewStyle().Foreground(accent).Bold(true)
	}
	endLbl := lipgloss.NewStyle().Foreground(dim)
	if endFocused {
		endLbl = lipgloss.NewStyle().Foreground(accent).Bold(true)
	}

	startSec := strings.Join([]string{
		startLbl.Render("Start"),
		renderCalGrid(m.startPicker, m.focused == ceStartCal),
		m.renderTimePicker(m.startHr, m.startMin, m.focused == ceStartHr, m.focused == ceStartMin),
	}, "\n")

	endSec := strings.Join([]string{
		endLbl.Render("End"),
		renderCalGrid(m.endPicker, m.focused == ceEndCal),
		m.renderTimePicker(m.endHr, m.endMin, m.focused == ceEndHr, m.focused == ceEndMin),
	}, "\n")

	startBox := lipgloss.NewStyle().Width(ceRightW).PaddingRight(4)
	return lipgloss.JoinHorizontal(lipgloss.Top, startBox.Render(startSec), endSec)
}

func (m *createEventModel) renderTimePicker(hr, min int, hrFocused, minFocused bool) string {
	accent := lipgloss.Color(styles.ColorSecondaryFg)
	bord := lipgloss.Color(styles.ColorSecondaryBorder)
	dim := lipgloss.Color(styles.ColorBorder)

	hrSty := lipgloss.NewStyle().Foreground(dim)
	minSty := lipgloss.NewStyle().Foreground(dim)
	upSty := lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
	downSty := lipgloss.NewStyle().Foreground(lipgloss.Color("244"))

	if hrFocused {
		hrSty = lipgloss.NewStyle().Foreground(lipgloss.Color(styles.ColorBlack)).Background(bord).Bold(true)
		upSty = lipgloss.NewStyle().Foreground(accent)
		downSty = lipgloss.NewStyle().Foreground(accent)
	} else if minFocused {
		minSty = lipgloss.NewStyle().Foreground(lipgloss.Color(styles.ColorBlack)).Background(bord).Bold(true)
		upSty = lipgloss.NewStyle().Foreground(accent)
		downSty = lipgloss.NewStyle().Foreground(accent)
	}

	sep := lipgloss.NewStyle().Foreground(dim)
	line1 := fmt.Sprintf("  %s     %s", upSty.Render("▲"), upSty.Render("▲"))
	line2 := fmt.Sprintf("  %s %s %s", hrSty.Render(fmt.Sprintf("%02d", hr)), sep.Render(":"), minSty.Render(fmt.Sprintf("%02d", min)))
	line3 := fmt.Sprintf("  %s     %s", downSty.Render("▼"), downSty.Render("▼"))
	return strings.Join([]string{line1, line2, line3}, "\n")
}

func (m *createEventModel) renderSubmit() string {
	if m.focused == ceSubmit {
		return lipgloss.NewStyle().
			Foreground(lipgloss.Color(styles.ColorBlack)).
			Background(lipgloss.Color(styles.ColorSecondaryFg)).
			Bold(true).Padding(0, 2).
			Render("  Create Event  ")
	}
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color(styles.ColorBorder)).
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("243")).
		Padding(0, 2).
		Render("  Create Event  ")
}

func (m *createEventModel) renderHelp() string {
	key := lipgloss.NewStyle().Foreground(lipgloss.Color(styles.ColorSecondaryBorder))
	dim := lipgloss.NewStyle().Foreground(lipgloss.Color("243"))

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
