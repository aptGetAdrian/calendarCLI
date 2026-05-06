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
	ti.Focus()

	ta := textarea.New()
	ta.Placeholder = "Description (optional)"
	ta.ShowLineNumbers = false

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
	rightW := m.width - leftW

	leftStyle := lipgloss.NewStyle().
		Width(leftW).
		BorderRight(true).
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color(styles.ColorBorder))

	rightStyle := lipgloss.NewStyle().
		Width(rightW).
		PaddingLeft(2)

	return lipgloss.JoinHorizontal(lipgloss.Top,
		leftStyle.Render(m.renderLeftPanel(leftW-2)),
		rightStyle.Render(m.renderRightPanel(rightW-4)),
	)
}

func (m *todoScreenModel) renderLeftPanel(width int) string {
	title := styles.SecondaryMenuTtitle().Render("To-Do List")

	var listLines strings.Builder
	if len(m.todos) == 0 {
		listLines.WriteString(styles.InfoText.Render("  No to-dos yet."))
	} else {
		selectedStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color(styles.ColorAccent)).
			Bold(true).
			BorderLeft(true).
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color(styles.ColorAccent)).
			PaddingLeft(1).
			Width(width - 2)

		normalStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color(styles.ColorInfo)).
			PaddingLeft(3).
			Width(width - 2)

		for i, todo := range m.todos {
			if i == m.cursor {
				listLines.WriteString(selectedStyle.Render(todo.Title) + "\n")
			} else {
				listLines.WriteString(normalStyle.Render(todo.Title) + "\n")
			}
		}
	}

	var hint string
	if m.mode == todoListMode {
		hint = styles.InfoText.Render("↑↓/jk: navigate  a: add  d: delete  esc: back")
	} else {
		hint = styles.InfoText.Render("tab: switch field  ctrl+s: save  esc: cancel")
	}

	return lipgloss.JoinVertical(lipgloss.Left,
		title,
		"",
		listLines.String(),
		"",
		hint,
	)
}

func (m *todoScreenModel) renderRightPanel(width int) string {
	if m.mode == todoAddMode {
		return m.renderAddForm()
	}

	panelTitle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color(styles.ColorAccent)).
		Render("Description")

	if len(m.todos) == 0 {
		return lipgloss.JoinVertical(lipgloss.Left,
			panelTitle,
			"",
			styles.InfoText.Render("Press 'a' to add your first to-do."),
		)
	}

	desc := m.todos[m.cursor].Description
	if desc == "" {
		desc = styles.InfoText.Render("(no description)")
	} else {
		desc = lipgloss.NewStyle().
			Foreground(lipgloss.Color(styles.ColorBorder)).
			Width(width).
			Render(desc)
	}

	return lipgloss.JoinVertical(lipgloss.Left,
		panelTitle,
		"",
		desc,
	)
}

func (m *todoScreenModel) renderAddForm() string {
	formTitle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color(styles.ColorAccent)).
		Render("New To-Do")

	titleLabel := lipgloss.NewStyle().
		Foreground(lipgloss.Color(styles.ColorInfo)).
		Render("Title *")

	descLabel := lipgloss.NewStyle().
		Foreground(lipgloss.Color(styles.ColorInfo)).
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
