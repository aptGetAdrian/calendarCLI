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
)

type todoScreenModel struct {
	state       AppState
	todos       []calendar.Todo
	cursor      int
	mode        todoMode
	titleInput  textinput.Model
	descInput   textarea.Model
	activeField int // 0 = title, 1 = description
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

func (m *todoScreenModel) updateListMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "esc":
		return m, func() tea.Msg { return NavigateTo{Screen: ui.MainMenuScreen} }
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
	case "down", "j":
		if m.cursor < len(m.todos)-1 {
			m.cursor++
		}
	case "a":
		m.mode = todoAddMode
		m.activeField = 0
		m.titleInput.SetValue("")
		m.descInput.SetValue("")
		m.titleInput.Focus()
		m.descInput.Blur()
		return m, textinput.Blink
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
		todos, err := calendar.AddTodo(title, desc)
		if err != nil {
			m.logger.Error("Failed to save todo: %v", err)
		} else {
			m.todos = todos
			m.cursor = len(m.todos) - 1
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

	titleH := lipgloss.Height(styles.SecondaryMenuTtitle().Render("To-Do List")) + 1

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
	if m.mode == todoListMode {
		text = "↑↓/jk: navigate  a: add  d: delete  esc: back"
	} else {
		text = "tab: switch field  ctrl+s: save  esc: cancel"
	}
	return lipgloss.NewStyle().
		Width(m.width).
		PaddingLeft(1).
		Render(styles.InfoText.Render(text))
}

func (m *todoScreenModel) renderLeftPanel(width int) string {
	title := styles.SecondaryMenuTtitle().Render("To-Do List")

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

		for i, todo := range m.todos {
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

func (m *todoScreenModel) renderRightPanel(width int) string {
	if m.mode == todoAddMode {
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

	desc := m.todos[m.cursor].Description
	if desc == "" {
		desc = mutedStyle.Render("(no description)")
	} else {
		desc = mutedStyle.Width(width).Render(desc)
	}

	return lipgloss.JoinVertical(lipgloss.Left,
		labelStyle.Render("Description"),
		"",
		desc,
	)
}

func (m *todoScreenModel) renderAddForm() string {
	formTitle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#f0d080")).
		Render("New To-Do")

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
