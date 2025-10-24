package ui

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	"github.com/ionut-t/coffee/markdown"
	editor "github.com/ionut-t/goeditor/adapter-bubbletea"
	"github.com/ionut-t/goeditor/core"
	"github.com/ionut-t/notes/internal/help"
	"github.com/ionut-t/notes/internal/keymap"
	"github.com/ionut-t/notes/internal/utils"
	"github.com/ionut-t/notes/note"
	"github.com/ionut-t/notes/styles"
)

type NoteModel struct {
	store            *note.Store
	viewport         viewport.Model
	width, height    int
	help             help.Model
	successMessage   string
	error            error
	markdown         markdown.Model
	fullScreen       bool
	showEditor       bool
	editor           editor.Model
	confirmation     *huh.Confirm
	showConfirmation bool

	previousCursorPosition core.Position
	currentNoteName        string
}

func NewNoteModel(store *note.Store, width, height int) NoteModel {
	note, _ := store.GetCurrentNote()

	vp := viewport.New(width, height)

	md := markdown.New()

	md.Render(note.Content)

	textEditor := editor.New(80, 20)
	textEditor.SetCursorMode(editor.CursorBlink)
	textEditor.WithTheme(styles.EditorTheme())
	textEditor.SetLanguage("markdown", styles.EditorLanguageTheme())
	textEditor.SetExtraHighlightedContextLines(1000)

	textEditor.SetContent(note.Content)

	helpMenu := help.New()

	helpMenu.Keys.ShortHelpBindings = []key.Binding{
		keymap.ExternalEditor,
	}

	helpMenu.Keys.FullHelpBindings = []key.Binding{
		keymap.Up,
		keymap.Down,
		keymap.ExternalEditor,
		keymap.New,
		keymap.Quit,
		keymap.Help,
	}

	helpMenu.SetSize(width, height)

	confirmation := huh.NewConfirm().
		Title("You have unsaved changes. Are you sure you want to discard them?").
		Affirmative("Yes").
		Negative("No")

	confirmation.WithKeyMap(&huh.KeyMap{
		Confirm: huh.NewDefaultKeyMap().Confirm,
	})

	confirmation.WithTheme(styles.ThemeCatppuccin())

	return NoteModel{
		store:           store,
		viewport:        vp,
		width:           width,
		height:          height,
		help:            helpMenu,
		markdown:        md,
		editor:          textEditor,
		confirmation:    confirmation,
		showEditor:      true,
		currentNoteName: note.Name,
	}
}

func (m NoteModel) Init() tea.Cmd {
	return nil
}

func (m NoteModel) View() string {
	view := utils.Ternary(m.showEditor, m.editor.View(), m.viewport.View())

	if m.showConfirmation {
		view = lipgloss.JoinVertical(
			lipgloss.Left,
			view,
			m.confirmation.View(),
		)
	}

	if !m.fullScreen {
		return view
	}

	content := lipgloss.JoinVertical(
		lipgloss.Left,
		view,
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
	case editor.DeleteFileMsg:
		return m.executeNoteDeletion()

	case editor.RenameMsg:
		return m.renameNote(msg.FileName)

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, keymap.Save):
			if m.showConfirmation {
				confirmed := m.confirmation.GetValue().(bool)

				if confirmed {
					m.confirm(false)

					if note, ok := m.store.GetCurrentNote(); ok {
						m.editor.SetContent(note.Content)
					}

					m.editor.Blur()

					return m, dispatch(changesDiscardedMsg{})
				}

				m.confirm(false)

				m.editor.Focus()
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

	if m.showConfirmation {
		confirmation, cmd := m.confirmation.Update(msg)
		m.confirmation = confirmation.(*huh.Confirm)
		cmds = append(cmds, cmd)
	} else {
		editorModel, cmd := m.editor.Update(msg)
		m.editor = editorModel.(editor.Model)
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

	modifiedDate := styles.Accent.Background(bg).Render("Last Modified " + note.UpdatedAt.Format("02/01/2006 15:04"))

	noteInfo := styles.Surface0.Padding(0, 1).Render(
		name + separator + modifiedDate,
	)

	lineNumbers := styles.Info.Background(bg).Render(strconv.Itoa(m.getLineNumbers()))

	scroll := styles.Surface0.Render(fmt.Sprintf("%3.f%%", m.viewport.ScrollPercent()*100))

	helpText := styles.Info.Background(bg).PaddingRight(1).Render("? Help")

	displayedInfoWidth := m.viewport.Width -
		lipgloss.Width(noteInfo) -
		lipgloss.Width(scroll) -
		lipgloss.Width(lineNumbers) -
		lipgloss.Width(helpText) -
		2*lipgloss.Width(separator)

	spaces := styles.Surface0.Render(strings.Repeat(" ", max(0, displayedInfoWidth)))

	return styles.Surface0.Width(m.width).Padding(0, 0).Render(
		lipgloss.JoinHorizontal(
			lipgloss.Right,
			noteInfo,
			spaces,
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

	if m.showConfirmation {
		m.editor.SetSize(width, max(height-helpHeight-lipgloss.Height(m.confirmation.View()), 0))
	} else {
		m.editor.SetSize(width, max(height-helpHeight-statusBarViewHeight, 0))
	}
}

func (m *NoteModel) updateContent() {
	m.previousCursorPosition = m.editor.GetCursorPosition()

	m.viewport.Height = m.height
	m.viewport.Width = m.width
	m.viewport.SetYOffset(0)
	m.editor.SetSize(m.width, m.height)
	m.render()

	texteditor, _ := m.editor.Update(nil)
	m.editor = texteditor.(editor.Model)
}

func (m *NoteModel) render() {
	if note, ok := m.store.GetCurrentNote(); ok {
		if out, err := m.markdown.Render(note.Content); err != nil {
			m.error = fmt.Errorf("failed to render note content: %w", err)
		} else {
			m.viewport.SetContent(out)
			m.viewport.YOffset = 0
		}

		m.editor.SetContent(note.Content)

		var cursorPosition core.Position
		if m.currentNoteName == note.Name {
			cursorPosition = m.previousCursorPosition
		} else {
			cursorPosition = core.Position{Row: 0, Col: 0}
		}

		if err := m.editor.SetCursorPosition(cursorPosition.Row, cursorPosition.Col); err != nil {
			m.error = fmt.Errorf("failed to set cursor position: %w", err)
		}

		m.currentNoteName = note.Name

	} else {
		m.viewport.SetContent(` Press "ctrl+n" to create a new note`)
	}
}

func (m *NoteModel) isEditing() bool {
	return m.editor.IsInsertMode()
}

func (m *NoteModel) hasChanges() bool {
	return m.editor.HasChanges()
}

func (m *NoteModel) toggleEdit() {
	m.showEditor = !m.showEditor
	m.updateContent()

	if m.showEditor {
		m.editor.Focus()
		texteditor, _ := m.editor.Update(nil)
		m.editor = texteditor.(editor.Model)
	} else {
		m.editor.Blur()
	}

	m.setSize(m.width, m.height)
}

func (m *NoteModel) focus() tea.Cmd {
	if m.showEditor {
		m.editor.Focus()
		return m.editor.CursorBlink()
	}

	return nil
}

func (m *NoteModel) blur() {
	if m.showEditor {
		m.editor.Blur()

		if m.editor.IsCommandMode() {
			m.editor.SetNormalMode()
		}
	}
}

func (m *NoteModel) confirm(show bool) {
	m.showConfirmation = show
	m.setSize(m.width, m.height)
}

func (m NoteModel) executeNoteDeletion() (NoteModel, tea.Cmd) {
	err := m.store.DeleteCurrentNote()

	if err != nil {
		return m, dispatch(cmdErrorMsg(err))
	}

	return m, tea.Sequence(
		dispatch(cmdNoteDeletedMsg{}),
		dispatch(cmdSuccessMsg("Note successfully deleted")),
	)
}

func (m NoteModel) renameNote(name string) (NoteModel, tea.Cmd) {
	note, err := m.store.RenameCurrentNote(name)

	if err != nil {
		return m, dispatch(cmdErrorMsg(err))
	}

	return m, tea.Sequence(
		dispatch(cmdNoteRenamedMsg{note}),
		dispatch(cmdSuccessMsg(fmt.Sprintf("Note renamed to \"%s\"", note.Name))),
	)
}
