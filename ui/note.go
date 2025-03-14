package ui

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/ionut-t/notes/internal/help"
	"github.com/ionut-t/notes/internal/keymap"
	"github.com/ionut-t/notes/internal/utils"
	"github.com/ionut-t/notes/markdown"
	"github.com/ionut-t/notes/note"
	"github.com/ionut-t/notes/styles"
)

var markdownPadding = lipgloss.NewStyle().Padding(0, 4).Render

type NoteModel struct {
	store          *note.Store
	viewport       viewport.Model
	width, height  int
	help           help.Model
	successMessage string
	error          error
	markdown       markdown.Model
	vLine          bool
	fullScreen     bool
}

func NewNoteModel(store *note.Store, width, height int) NoteModel {
	note, _ := store.GetCurrentNote()

	vLine := store.GetVLineEnabledByDefault()

	vp := viewport.New(width, height)

	md := markdown.New(note.Content, width)
	md.SetLineNumbers(vLine)

	content := utils.Ternary(vLine, md.Render(), markdownPadding(md.Render()))
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
		keymap.VLine,
		keymap.Copy,
		keymap.CopyLines,
		keymap.New,
		keymap.Quit,
		keymap.Help,
	}

	helpMenu.SetSize(width, height)

	return NoteModel{
		store:    store,
		viewport: vp,
		width:    width,
		height:   height,
		help:     helpMenu,
		markdown: md,
		vLine:    vLine,
	}
}

func (m NoteModel) Init() tea.Cmd {
	return nil
}

func (m NoteModel) View() string {
	if !m.fullScreen {
		return m.viewport.View()
	}

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

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
	case tea.KeyMsg:
		keyMsg := tea.KeyMsg(msg).String()

		switch keyMsg {
		case "V":
			if note, ok := m.store.GetCurrentNote(); ok {
				m.vLine = !m.vLine
				m.markdown = markdown.New(note.Content, m.width)
				m.markdown.SetLineNumbers(m.vLine)
				content := utils.Ternary(m.vLine, m.markdown.Render(), markdownPadding(m.markdown.Render()))
				m.viewport.SetContent(content)
			}
		}
	}

	m.viewport, cmd = m.viewport.Update(msg)
	cmds = append(cmds, cmd)

	if m.fullScreen {
		helpModel, cmd := m.help.Update(msg)
		m.help = helpModel.(help.Model)
		m.help.SetSize(m.width, m.height)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m NoteModel) statusBarView() string {
	if !m.fullScreen {
		return ""
	}

	bg := styles.Surface0.GetBackground()

	if m.successMessage != "" {
		return styles.Success.Background(bg).Width(m.width).Padding(0, 1).Render(m.successMessage)
	}

	if m.error != nil {
		return styles.Error.Background(bg).Width(m.width).Padding(0, 1).Render(m.error.Error())
	}

	separator := styles.Surface0.Render(" | ")

	note, _ := m.store.GetCurrentNote()

	name := styles.Primary.Background(bg).Render(note.Name)

	modifiedDate := styles.Accent.Background(bg).Render("Last Modified " + note.CreatedAt.Format("02/01/2006 15:04"))

	noteInfo := styles.Surface0.Padding(0, 1).Render(
		name + separator + modifiedDate,
	)

	lineNumbers := styles.Info.Background(bg).Render(strconv.Itoa(m.getLineNumbers()))

	vLineStyles := utils.Ternary(m.vLine, styles.Accent, styles.Overlay0)

	vLineText := vLineStyles.Background(bg).Render("V-Line")

	scroll := styles.Surface0.Render(fmt.Sprintf("%3.f%%", m.viewport.ScrollPercent()*100))

	helpText := styles.Info.Background(bg).PaddingRight(1).Render("? Help")

	displayedInfoWidth := m.viewport.Width -
		lipgloss.Width(noteInfo) -
		lipgloss.Width(scroll) -
		lipgloss.Width(vLineText) -
		lipgloss.Width(lineNumbers) -
		lipgloss.Width(helpText) -
		3*lipgloss.Width(separator)

	spaces := styles.Surface0.Render(strings.Repeat(" ", max(0, displayedInfoWidth)))

	return styles.Surface0.Width(m.width).Padding(0, 0).Render(
		lipgloss.JoinHorizontal(
			lipgloss.Right,
			noteInfo,
			spaces,
			vLineText,
			separator,
			lineNumbers,
			separator,
			scroll,
			separator,
			helpText,
		),
	)
}

func (m NoteModel) getLineNumbers() int {
	if note, ok := m.store.GetCurrentNote(); ok {
		return len(strings.Split(note.Content, "\n"))
	}

	return 0
}

func (m *NoteModel) setSize(width, height int) {
	m.width = width
	m.height = height

	statusBarViewHeight := utils.Ternary(m.fullScreen, lipgloss.Height(m.statusBarView()), 0)
	helpHeight := utils.Ternary(m.help.FullView, lipgloss.Height(m.help.View()), 0)

	m.viewport.Height = height - helpHeight - statusBarViewHeight
	m.viewport.Width = width
}

func (m *NoteModel) updateContent() {
	if note, ok := m.store.GetCurrentNote(); ok {
		md := markdown.New(note.Content, max(m.width, 40))
		md.SetLineNumbers(m.vLine)
		content := utils.Ternary(m.vLine, md.Render(), markdownPadding(md.Render()))
		m.viewport.SetContent(content)
	} else {
		m.viewport.SetContent(` Press "n" to create a new note`)
	}

	m.viewport.Height = m.height
	m.viewport.Width = m.width
	m.viewport.SetYOffset(0)
}
