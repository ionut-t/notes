package ui

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
	"github.com/ionut-t/notes/internal/help"
	"github.com/ionut-t/notes/internal/keymap"
	"github.com/ionut-t/notes/note"
	"github.com/ionut-t/notes/styles"
)

type NoteModel struct {
	note           note.Note
	viewport       viewport.Model
	width, height  int
	help           help.Model
	successMessage string
	error          error
}

func NewNoteModel(note note.Note, width, height int) NoteModel {
	vp := viewport.New(width, height-1)

	content, err := glamour.Render(note.Content, "dark")
	if err != nil {
		content = "Error rendering content"
	}
	vp.SetContent(content)

	helpMenu := help.New()

	helpMenu.Keys.ShortHelpBindings = []key.Binding{
		keymap.Editor,
	}

	helpMenu.Keys.FullHelpBindings = []key.Binding{
		keymap.Up,
		keymap.Down,
		keymap.QuickEditor,
		keymap.Rename,
		keymap.Copy,
		keymap.CopyCodeBlock,
		keymap.Quit,
		keymap.Help,
	}

	helpMenu.SetSize(width, height)

	return NoteModel{
		note:     note,
		viewport: vp,
		width:    width,
		height:   height,
		help:     helpMenu,
	}
}

func (m NoteModel) Init() tea.Cmd {
	return nil
}

func (m NoteModel) View() string {
	content := lipgloss.JoinVertical(
		lipgloss.Top,
		m.viewport.View(),
		m.statusBarView(),
	)

	if m.help.FullView {
		return lipgloss.JoinVertical(
			lipgloss.Top,
			content,
			m.help.View(),
		)
	}

	return content
}

func (m NoteModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		cmd  tea.Cmd
		cmds []tea.Cmd
	)

	m.viewport, cmd = m.viewport.Update(msg)
	cmds = append(cmds, cmd)

	helpModel, cmd := m.help.Update(msg)
	m.help = helpModel.(help.Model)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m NoteModel) statusBarView() string {
	bg := styles.Surface0.GetBackground()

	if m.successMessage != "" {
		return styles.Success.Background(bg).Width(m.width).Padding(0, 1).Render(m.successMessage)
	}

	if m.error != nil {
		return styles.Error.Background(bg).Width(m.width).Padding(0, 1).Render(m.error.Error())
	}

	scroll := styles.Surface0.Padding(0, 1).Render(fmt.Sprintf("%3.f%%", m.viewport.ScrollPercent()*100))

	name := styles.Primary.Background(bg).Render(m.note.Name)

	modifiedDate := styles.Accent.Background(bg).Render("Last Modified " + m.note.CreatedAt.Format("02/01/2006 15:04"))

	noteInfo := styles.Surface0.Padding(0, 1).Render(
		name + styles.Surface0.Render(" | ") + modifiedDate,
	)

	lineNumbers := styles.Info.Background(bg).Render(strconv.Itoa(m.getLineNumbers()))

	helpText := styles.Info.Background(bg).Padding(0, 1).Render("? Help")
	displayedInfoWidth := m.viewport.Width - lipgloss.Width(noteInfo) - lipgloss.Width(scroll) - lipgloss.Width(lineNumbers) - lipgloss.Width(helpText)

	spaces := styles.Surface0.Render(strings.Repeat(" ", max(0, displayedInfoWidth)))

	return styles.Surface0.Width(m.width).Padding(0, 0).Render(
		lipgloss.JoinHorizontal(lipgloss.Right, noteInfo, spaces, lineNumbers, scroll, helpText),
	)
}

func (m NoteModel) getLineNumbers() int {
	return len(strings.Split(m.note.Content, "\n"))
}

func (m *NoteModel) setNote(note note.Note) {
	m.note = note
	content, err := glamour.Render(note.Content, "dark")
	if err != nil {
		content = "Error rendering content"
	}
	m.viewport.SetContent(content)
}

func (m *NoteModel) setHeight(height int) {
	m.height = height

	if m.help.FullView {
		m.viewport.Height = height - lipgloss.Height(m.statusBarView()) - lipgloss.Height(m.help.View())
	} else {
		m.viewport.Height = height - lipgloss.Height(m.statusBarView())
	}
}
