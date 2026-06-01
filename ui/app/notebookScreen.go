package app

import (
	calendar "calendarCli/internal"
	"calendarCli/internal/logger"
	"calendarCli/ui"
	"calendarCli/ui/styles"
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/glamour"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type notebookMode int

const (
	notebookBrowseMode notebookMode = iota
	notebookNewMode
	notebookEditMode
)

const notebookTopicBarH = 2 // topic bar line + blank separator

type notebookScreenModel struct {
	state         AppState
	topicIdx      int
	notes         []calendar.Note
	cursor        int
	activePanel   int // 0 = left list, 1 = right content
	listScroll    int
	contentScroll int
	renderedLines []string
	rawContent    string
	mode          notebookMode
	titleInput    textinput.Model
	bodyInput     textarea.Model
	editFilename  string
	width         int
	height        int
	logger        *logger.Logger
}

func newNotebookScreenModel(state AppState, width, height int, logger *logger.Logger) *notebookScreenModel {
	ti := textinput.New()
	ti.Placeholder = "Note title"
	ti.Prompt = "  "
	ti.Focus()

	ta := textarea.New()
	ta.Placeholder = "Write your note in Markdown..."
	ta.ShowLineNumbers = false
	ta.Prompt = "  "
	ta.FocusedStyle.CursorLine = lipgloss.NewStyle().Background(lipgloss.Color("#1e2018"))

	m := &notebookScreenModel{
		state:      state,
		topicIdx:   0,
		cursor:     0,
		mode:       notebookBrowseMode,
		titleInput: ti,
		bodyInput:  ta,
		width:      width,
		height:     height,
		logger:     logger,
	}
	m.resizeInputs()
	m.reloadNotes()
	return m
}

func (m *notebookScreenModel) currentTopic() string {
	return calendar.Topics[m.topicIdx]
}

func (m *notebookScreenModel) reloadNotes() {
	notes, err := calendar.ListNotes(m.currentTopic())
	if err != nil {
		m.logger.Error("Failed to list notes: %v", err)
		notes = []calendar.Note{}
	}
	m.notes = notes
	if m.cursor >= len(m.notes) {
		m.cursor = max(0, len(m.notes)-1)
	}
	m.listScroll = 0
	m.contentScroll = 0
	m.loadCurrentNote()
}

func (m *notebookScreenModel) loadCurrentNote() {
	if len(m.notes) == 0 {
		m.rawContent = ""
		m.renderedLines = nil
		return
	}
	note := m.notes[m.cursor]
	content, err := calendar.LoadNote(note.Topic, note.Filename)
	if err != nil {
		m.logger.Error("Failed to load note: %v", err)
		m.renderedLines = []string{"(error loading note)"}
		return
	}
	m.rawContent = content
	m.renderedLines = m.renderMarkdown(content)
	m.contentScroll = 0
}

func (m *notebookScreenModel) renderMarkdown(content string) []string {
	rightW := m.width - m.width/2 - 4
	if rightW < 20 {
		rightW = 20
	}
	r, err := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(rightW),
	)
	if err != nil {
		return strings.Split(content, "\n")
	}
	rendered, err := r.Render(content)
	if err != nil {
		return strings.Split(content, "\n")
	}
	return strings.Split(strings.TrimRight(rendered, "\n"), "\n")
}

func (m *notebookScreenModel) resizeInputs() {
	rightW := m.width - m.width/2 - 4
	if rightW < 10 {
		rightW = 10
	}
	m.titleInput.Width = rightW
	m.bodyInput.SetWidth(rightW)
	bodyH := m.height - notebookTopicBarH - 8
	if bodyH < 3 {
		bodyH = 3
	}
	m.bodyInput.SetHeight(bodyH)
}

func (m *notebookScreenModel) panelH() int {
	return m.height - notebookTopicBarH - 1 // -1 for hint bar
}

func (m *notebookScreenModel) titleAreaH() int {
	return lipgloss.Height(styles.NotebookTtitle().Render("Notebook")) + 1
}

func (m *notebookScreenModel) visibleListCount() int {
	v := m.panelH() - m.titleAreaH()
	if v < 1 {
		return 1
	}
	return v
}

func (m *notebookScreenModel) visibleContentCount() int {
	v := m.panelH() - m.titleAreaH()
	if v < 1 {
		return 1
	}
	return v
}

func (m *notebookScreenModel) clampListScroll() {
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

func (m *notebookScreenModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m *notebookScreenModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case sizedMsg:
		m.width = msg.width
		m.height = msg.height
		m.resizeInputs()
		if len(m.notes) > 0 {
			m.renderedLines = m.renderMarkdown(m.rawContent)
		}
		return m, nil

	case tea.KeyMsg:
		if m.mode == notebookBrowseMode {
			return m.updateBrowseMode(msg)
		}
		return m.updateEditMode(msg)
	}
	return m, nil
}

func (m *notebookScreenModel) updateBrowseMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "esc":
		return m, func() tea.Msg { return NavigateTo{Screen: ui.MainMenuScreen} }

	case "tab":
		if m.activePanel == 0 {
			m.activePanel = 1
		} else {
			m.activePanel = 0
		}

	case "left", "h":
		if m.topicIdx > 0 {
			m.topicIdx--
			m.reloadNotes()
		}

	case "right", "l":
		if m.topicIdx < len(calendar.Topics)-1 {
			m.topicIdx++
			m.reloadNotes()
		}

	case "up", "k":
		if m.activePanel == 0 {
			if m.cursor > 0 {
				m.cursor--
				m.loadCurrentNote()
			}
			m.clampListScroll()
		} else {
			if m.contentScroll > 0 {
				m.contentScroll--
			}
		}

	case "down", "j":
		if m.activePanel == 0 {
			if m.cursor < len(m.notes)-1 {
				m.cursor++
				m.loadCurrentNote()
			}
			m.clampListScroll()
		} else {
			vis := m.visibleContentCount()
			if m.contentScroll+vis < len(m.renderedLines) {
				m.contentScroll++
			}
		}

	case "n":
		m.mode = notebookNewMode
		m.activeField(0)
		m.titleInput.SetValue("")
		m.bodyInput.SetValue("")
		m.titleInput.Focus()
		m.bodyInput.Blur()
		return m, textinput.Blink

	case "e":
		if len(m.notes) > 0 {
			m.mode = notebookEditMode
			m.editFilename = m.notes[m.cursor].Filename
			m.bodyInput.SetValue(m.rawContent)
			m.bodyInput.Focus()
			m.titleInput.Blur()
			return m, textarea.Blink
		}

	case "d":
		if len(m.notes) > 0 {
			err := calendar.DeleteNote(m.notes[m.cursor].Topic, m.notes[m.cursor].Filename)
			if err != nil {
				m.logger.Error("Failed to delete note: %v", err)
			} else {
				m.reloadNotes()
			}
		}
	}
	return m, nil
}

func (m *notebookScreenModel) activeField(n int) {
	_ = n // only used to set title=0 body=1 logically
}

func (m *notebookScreenModel) updateEditMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.mode = notebookBrowseMode
		return m, nil

	case "ctrl+s":
		if m.mode == notebookNewMode {
			title := strings.TrimSpace(m.titleInput.Value())
			if title == "" {
				return m, nil
			}
			content := m.bodyInput.Value()
			note, err := calendar.CreateNote(m.currentTopic(), title, content)
			if err != nil {
				m.logger.Error("Failed to create note: %v", err)
			} else {
				m.reloadNotes()
				// position cursor on the new note
				for i, n := range m.notes {
					if n.Filename == note.Filename {
						m.cursor = i
						break
					}
				}
				m.clampListScroll()
				m.loadCurrentNote()
			}
		} else {
			content := m.bodyInput.Value()
			err := calendar.SaveNote(m.currentTopic(), m.editFilename, content)
			if err != nil {
				m.logger.Error("Failed to save note: %v", err)
			} else {
				m.reloadNotes()
				m.loadCurrentNote()
			}
		}
		m.mode = notebookBrowseMode
		return m, nil

	case "tab":
		if m.mode == notebookNewMode {
			// toggle between title and body inputs
			if m.titleInput.Focused() {
				m.titleInput.Blur()
				return m, m.bodyInput.Focus()
			}
			m.bodyInput.Blur()
			m.titleInput.Focus()
			return m, textinput.Blink
		}
	}

	var cmd tea.Cmd
	if m.mode == notebookNewMode && m.titleInput.Focused() {
		m.titleInput, cmd = m.titleInput.Update(msg)
	} else {
		m.bodyInput, cmd = m.bodyInput.Update(msg)
	}
	return m, cmd
}

// ─── View ─────────────────────────────────────────────────────────────────────

func (m *notebookScreenModel) View() string {
	topicBar := m.renderTopicBar()
	hint := m.renderHint()
	panelH := m.panelH()

	leftW := m.width / 2
	rightW := m.width - leftW - 1

	titleH := m.titleAreaH()

	leftPanel := lipgloss.NewStyle().
		Width(leftW).
		Height(panelH).
		Render(m.renderLeftPanel(leftW - 1))

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
		Render(m.renderRightPanel())

	panels := lipgloss.JoinHorizontal(lipgloss.Top, leftPanel, divider, rightPanel)
	return lipgloss.JoinVertical(lipgloss.Left, topicBar, panels, hint)
}

func (m *notebookScreenModel) renderTopicBar() string {
	activeStyle := lipgloss.NewStyle().
		Background(lipgloss.Color(styles.ColorNotebookActiveTab)).
		Foreground(lipgloss.Color(styles.ColorNotebookActiveTabFg)).
		Padding(0, 1).
		Bold(true)

	inactiveStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(styles.ColorNotebookInactiveTab)).
		Padding(0, 1)

	var parts []string
	for i, topic := range calendar.Topics {
		label := calendar.TopicLabels[topic]
		if i == m.topicIdx {
			parts = append(parts, activeStyle.Render(label))
		} else {
			parts = append(parts, inactiveStyle.Render(label))
		}
	}

	bar := lipgloss.NewStyle().
		Width(m.width).
		PaddingLeft(1).
		Render(strings.Join(parts, "  "))

	return bar + "\n"
}

func (m *notebookScreenModel) renderHint() string {
	var text string
	switch m.mode {
	case notebookBrowseMode:
		if m.activePanel == 1 {
			text = "↑↓: scroll  tab: focus list  ←→: topics  n: new  e: edit  d: delete  esc: back"
		} else {
			text = "↑↓: navigate  tab: focus note  ←→: topics  n: new  e: edit  d: delete  esc: back"
		}
	case notebookNewMode:
		text = "tab: title↔body  ctrl+s: save  esc: cancel"
	case notebookEditMode:
		text = "ctrl+s: save  esc: cancel"
	}
	return lipgloss.NewStyle().
		Width(m.width).
		PaddingLeft(1).
		Render(styles.InfoText.Render(text))
}

func (m *notebookScreenModel) renderLeftPanel(width int) string {
	title := styles.NotebookTtitle().Render("Notebook")

	var listLines strings.Builder
	if len(m.notes) == 0 {
		listLines.WriteString(lipgloss.NewStyle().
			Foreground(lipgloss.Color(styles.ColorBorder)).
			PaddingLeft(2).
			Render("No notes yet. Press 'n' to add one."))
	} else {
		selectedStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color(styles.ColorNotebookFg)).
			Background(lipgloss.Color(styles.ColorNotebookBg)).
			Bold(true).
			Width(width)

		normalStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color(styles.ColorBorder)).
			Width(width)

		vis := m.visibleListCount()
		start := m.listScroll
		end := start + vis
		if end > len(m.notes) {
			end = len(m.notes)
		}

		for i := start; i < end; i++ {
			note := m.notes[i]
			if i == m.cursor {
				listLines.WriteString(selectedStyle.Render("▸ "+note.Title) + "\n")
			} else {
				listLines.WriteString(normalStyle.Render("  "+note.Title) + "\n")
			}
		}
	}

	return lipgloss.JoinVertical(lipgloss.Left, title, "", listLines.String())
}

func (m *notebookScreenModel) renderRightPanel() string {
	switch m.mode {
	case notebookNewMode:
		return m.renderNewForm()
	case notebookEditMode:
		return m.renderEditForm()
	}

	// Browse mode: show rendered markdown
	headerStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color(styles.ColorNotebookFg))

	if len(m.notes) == 0 {
		return lipgloss.JoinVertical(lipgloss.Left,
			headerStyle.Render("Note"),
			"",
			lipgloss.NewStyle().Foreground(lipgloss.Color(styles.ColorBorder)).
				Render("Select a topic and press 'n' to create your first note."),
		)
	}

	if len(m.renderedLines) == 0 {
		return lipgloss.JoinVertical(lipgloss.Left,
			headerStyle.Render(m.notes[m.cursor].Title),
			"",
			lipgloss.NewStyle().Foreground(lipgloss.Color(styles.ColorBorder)).Render("(empty note)"),
		)
	}

	vis := m.visibleContentCount()
	end := m.contentScroll + vis
	if end > len(m.renderedLines) {
		end = len(m.renderedLines)
	}
	visible := strings.Join(m.renderedLines[m.contentScroll:end], "\n")
	return visible
}

func (m *notebookScreenModel) renderNewForm() string {
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(styles.ColorNotebookFg))
	labelStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(styles.ColorBorder))

	return lipgloss.JoinVertical(lipgloss.Left,
		titleStyle.Render("New Note"),
		"",
		labelStyle.Render("Title"),
		m.titleInput.View(),
		"",
		labelStyle.Render("Content (Markdown)"),
		m.bodyInput.View(),
	)
}

func (m *notebookScreenModel) renderEditForm() string {
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(styles.ColorNotebookFg))
	labelStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(styles.ColorBorder))

	name := ""
	if len(m.notes) > 0 {
		name = m.notes[m.cursor].Title
	}

	return lipgloss.JoinVertical(lipgloss.Left,
		titleStyle.Render("Editing: "+name),
		"",
		labelStyle.Render("Content (Markdown)"),
		m.bodyInput.View(),
	)
}
