package app

import (
	calendar "calendarCli/internal"
	"calendarCli/internal/logger"
	"calendarCli/ui"
	"calendarCli/ui/styles"
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type todoMode int

const (
	todoListMode todoMode = iota
	todoAddMode
	todoEditMode
)

type todoScreenModel struct {
	state       AppState
	todos       []calendar.Todo
	cursor      int
	mode        todoMode
	titleInput  textinput.Model
	descInput   textarea.Model
	activeField int // 0 = title, 1 = description (add/edit form)
	activePanel int // 0 = left list, 1 = right description
	listScroll  int
	descScroll  int
	width       int
	height      int
	logger      *logger.Logger
}

func newTodoScreenModel(state AppState, width, height int, logger *logger.Logger) *todoScreenModel {
	ti := textinput.New()
	ti.Placeholder = "Title (required)"
	ti.Prompt = "  "
	ti.Focus()

	ta := textarea.New()
	ta.Placeholder = "Description (optional)"
	ta.ShowLineNumbers = false
	ta.Prompt = "  "
	ta.FocusedStyle.CursorLine = lipgloss.NewStyle().Background(lipgloss.Color("#1e2018"))

	todos, err := calendar.LoadTodos()
	if err != nil {
		logger.Error("Failed to load todos: %v", err)
		todos = []calendar.Todo{}
	}

	m := &todoScreenModel{
		state:      state,
		todos:      todos,
		cursor:     0,
		mode:       todoListMode,
		titleInput: ti,
		descInput:  ta,
		width:      width,
		height:     height,
		logger:     logger,
	}
	m.resizeInputs()
	return m
}

func (m *todoScreenModel) resizeInputs() {
	rightW := m.width - m.width/2 - 4
	if rightW < 10 {
		rightW = 10
	}
	m.titleInput.Width = rightW
	m.descInput.SetWidth(rightW)
	descH := m.height / 3
	if descH < 3 {
		descH = 3
	}
	m.descInput.SetHeight(descH)
}

// panelH is the height available for the two panels (total height minus the hint bar).
func (m *todoScreenModel) panelH() int {
	return m.height - 1
}

func (m *todoScreenModel) titleAreaH() int {
	return lipgloss.Height(styles.TodoTtitle().Render("To-Do List")) + 1
}

// visibleListCount returns how many list rows fit in the left panel.
func (m *todoScreenModel) visibleListCount() int {
	v := m.panelH() - m.titleAreaH()
	if v < 1 {
		return 1
	}
	return v
}

// visibleDescCount returns how many description lines fit in the right panel.
func (m *todoScreenModel) visibleDescCount() int {
	// right panel has PaddingTop(titleAreaH) + "Description" label + blank line
	v := m.panelH() - m.titleAreaH() - 2
	if v < 1 {
		return 1
	}
	return v
}

func (m *todoScreenModel) clampListScroll() {
	vis := m.visibleListCount()
	if m.cursor < m.listScroll {
		m.listScroll = m.cursor
	}
	if m.cursor >= m.listScroll+vis {
		m.listScroll = m.cursor - vis + 1
	}
	if m.listScroll < 0 {
		m.listScroll = 0
	}
}

// wrappedDescLines word-wraps the current todo's description and returns its lines.
func (m *todoScreenModel) wrappedDescLines() []string {
	if len(m.todos) == 0 {
		return nil
	}
	desc := m.todos[m.cursor].Description
	if desc == "" {
		return nil
	}
	rightW := m.width - m.width/2 - 4
	if rightW < 10 {
		rightW = 10
	}
	rendered := lipgloss.NewStyle().Width(rightW).Render(desc)
	return strings.Split(rendered, "\n")
}

func (m *todoScreenModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m *todoScreenModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case sizedMsg:
		m.width = msg.width
		m.height = msg.height
		m.resizeInputs()
		return m, nil

	case tea.KeyMsg:
		if m.mode == todoListMode {
			return m.updateListMode(msg)
		}
		return m.updateAddMode(msg)
	}
	return m, nil
}

func (m *todoScreenModel) enterEditMode() tea.Cmd {
	if len(m.todos) == 0 {
		return nil
	}
	todo := m.todos[m.cursor]
	m.mode = todoEditMode
	m.activeField = 0
	m.titleInput.SetValue(todo.Title)
	m.descInput.SetValue(todo.Description)
	m.titleInput.Focus()
	m.descInput.Blur()
	return textinput.Blink
}

func (m *todoScreenModel) updateListMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "esc":
		return m, func() tea.Msg { return NavigateTo{Screen: ui.MainMenuScreen} }

	case "tab":
		if m.activePanel == 0 {
			m.activePanel = 1
		} else {
			m.activePanel = 0
		}

	case "up", "k":
		if m.activePanel == 0 {
			if m.cursor > 0 {
				m.cursor--
				m.descScroll = 0
			}
			m.clampListScroll()
		} else {
			if m.descScroll > 0 {
				m.descScroll--
			}
		}

	case "down", "j":
		if m.activePanel == 0 {
			if m.cursor < len(m.todos)-1 {
				m.cursor++
				m.descScroll = 0
			}
			m.clampListScroll()
		} else {
			lines := m.wrappedDescLines()
			vis := m.visibleDescCount()
			if m.descScroll+vis < len(lines) {
				m.descScroll++
			}
		}

	case "a":
		m.mode = todoAddMode
		m.activeField = 0
		m.titleInput.SetValue("")
		m.descInput.SetValue("")
		m.titleInput.Focus()
		m.descInput.Blur()
		return m, textinput.Blink

	case "e":
		return m, m.enterEditMode()

	case "d":
		if len(m.todos) > 0 {
			todos, err := calendar.DeleteTodo(m.todos[m.cursor].ID)
			if err != nil {
				m.logger.Error("Failed to delete todo: %v", err)
			} else {
				m.todos = todos
				if m.cursor >= len(m.todos) && m.cursor > 0 {
					m.cursor--
				}
				m.descScroll = 0
				m.clampListScroll()
			}
		}
	}
	return m, nil
}

func (m *todoScreenModel) updateAddMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.mode = todoListMode
		return m, nil

	case "tab":
		if m.activeField == 0 {
			m.activeField = 1
			m.titleInput.Blur()
			return m, m.descInput.Focus()
		}
		m.activeField = 0
		m.descInput.Blur()
		m.titleInput.Focus()
		return m, textinput.Blink

	case "enter":
		if m.activeField == 0 {
			m.activeField = 1
			m.titleInput.Blur()
			return m, m.descInput.Focus()
		}

	case "ctrl+s":
		title := strings.TrimSpace(m.titleInput.Value())
		if title == "" {
			return m, nil
		}
		desc := strings.TrimSpace(m.descInput.Value())
		if m.mode == todoEditMode {
			todos, err := calendar.UpdateTodo(m.todos[m.cursor].ID, title, desc)
			if err != nil {
				m.logger.Error("Failed to update todo: %v", err)
			} else {
				m.todos = todos
				m.descScroll = 0
			}
		} else {
			todos, err := calendar.AddTodo(title, desc)
			if err != nil {
				m.logger.Error("Failed to save todo: %v", err)
			} else {
				m.todos = todos
				m.cursor = len(m.todos) - 1
				m.descScroll = 0
				m.clampListScroll()
			}
		}
		m.mode = todoListMode
		return m, nil
	}

	var cmd tea.Cmd
	if m.activeField == 0 {
		m.titleInput, cmd = m.titleInput.Update(msg)
	} else {
		m.descInput, cmd = m.descInput.Update(msg)
	}
	return m, cmd
}

func (m *todoScreenModel) View() string {
	leftW := m.width / 2
	rightW := m.width - leftW - 1 // -1 for divider char

	hint := m.renderHint()
	hintH := lipgloss.Height(hint)
	panelH := m.height - hintH

	titleH := m.titleAreaH()

	leftPanel := lipgloss.NewStyle().
		Width(leftW).
		Height(panelH).
		Render(m.renderLeftPanel(leftW - 1))

	// divider: blank for the title area, then │ from one line above the first item downward
	divStart := titleH - 1
	if divStart < 0 {
		divStart = 0
	}
	var divBuilder strings.Builder
	for i := range panelH {
		if i < divStart {
			divBuilder.WriteString(" \n")
		} else {
			divBuilder.WriteString("│\n")
		}
	}
	divider := lipgloss.NewStyle().
		Foreground(lipgloss.Color(styles.ColorBorder)).
		Render(strings.TrimRight(divBuilder.String(), "\n"))

	rightPanel := lipgloss.NewStyle().
		Width(rightW).
		Height(panelH).
		PaddingLeft(2).
		PaddingTop(titleH).
		Render(m.renderRightPanel(rightW - 2))

	panels := lipgloss.JoinHorizontal(lipgloss.Top, leftPanel, divider, rightPanel)
	return lipgloss.JoinVertical(lipgloss.Left, panels, hint)
}

func (m *todoScreenModel) renderHint() string {
	var text string
	switch m.mode {
	case todoListMode:
		if m.activePanel == 1 {
			text = "↑↓: scroll desc  tab: focus list  a: add  e: edit  d: delete  esc: back"
		} else {
			text = "↑↓: navigate  tab: focus desc  a: add  e: edit  d: delete  esc: back"
		}
	default:
		text = "tab: switch field  ctrl+s: save  esc: cancel"
	}
	return lipgloss.NewStyle().
		Width(m.width).
		PaddingLeft(1).
		Render(styles.InfoText.Render(text))
}

func (m *todoScreenModel) renderLeftPanel(width int) string {
	title := styles.TodoTtitle().Render("To-Do List")

	var listLines strings.Builder
	if len(m.todos) == 0 {
		listLines.WriteString(lipgloss.NewStyle().
			Foreground(lipgloss.Color(styles.ColorBorder)).
			PaddingLeft(2).
			Render("No to-dos yet."))
	} else {
		selectedStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#f0d080")).
			Background(lipgloss.Color("#1e2018")).
			Bold(true).
			Width(width)

		normalStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color(styles.ColorBorder)).
			Width(width)

		vis := m.visibleListCount()
		start := m.listScroll
		end := start + vis
		if end > len(m.todos) {
			end = len(m.todos)
		}

		for i := start; i < end; i++ {
			todo := m.todos[i]
			if i == m.cursor {
				listLines.WriteString(selectedStyle.Render("▸ "+todo.Title) + "\n")
			} else {
				listLines.WriteString(normalStyle.Render("  "+todo.Title) + "\n")
			}
		}
	}

	return lipgloss.JoinVertical(lipgloss.Left,
		title,
		"",
		listLines.String(),
	)
}

func (m *todoScreenModel) renderRightPanel(_ int) string {
	if m.mode == todoAddMode || m.mode == todoEditMode {
		return m.renderAddForm()
	}

	labelStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#f0d080"))

	mutedStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(styles.ColorBorder))

	if len(m.todos) == 0 {
		return lipgloss.JoinVertical(lipgloss.Left,
			labelStyle.Render("Description"),
			"",
			mutedStyle.Render("Press 'a' to add your first to-do."),
		)
	}

	lines := m.wrappedDescLines()
	var desc string
	if len(lines) == 0 {
		desc = mutedStyle.Render("(no description)")
	} else {
		vis := m.visibleDescCount()
		end := m.descScroll + vis
		if end > len(lines) {
			end = len(lines)
		}
		desc = mutedStyle.Render(strings.Join(lines[m.descScroll:end], "\n"))
	}

	return lipgloss.JoinVertical(lipgloss.Left,
		labelStyle.Render("Description"),
		"",
		desc,
	)
}

func (m *todoScreenModel) renderAddForm() string {
	formTitleText := "New To-Do"
	if m.mode == todoEditMode {
		formTitleText = "Edit To-Do"
	}
	formTitle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#f0d080")).
		Render(formTitleText)

	titleLabel := lipgloss.NewStyle().
		Foreground(lipgloss.Color(styles.ColorBorder)).
		Render("Title *")

	descLabel := lipgloss.NewStyle().
		Foreground(lipgloss.Color(styles.ColorBorder)).
		Render("Description")

	return lipgloss.JoinVertical(lipgloss.Left,
		formTitle,
		"",
		titleLabel,
		m.titleInput.View(),
		"",
		descLabel,
		m.descInput.View(),
	)
}
