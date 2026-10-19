package ui

import (
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/huh/v2"
	"charm.land/lipgloss/v2"
	"github.com/ionut-t/coffee/styles"
	editor "github.com/ionut-t/goeditor"
	"github.com/ionut-t/notes/internal/help"
	"github.com/ionut-t/notes/internal/keymap"
	"github.com/ionut-t/notes/internal/utils"
	"github.com/ionut-t/notes/note"
)

type NoteModel struct {
	store            *note.Store
	width, height    int
	help             help.Model
	successMessage   string
	error            error
	fullScreen       bool
	editor           editor.Model
	confirmation     *huh.Confirm
	showConfirmation bool

	currentNoteName string
}

func NewNoteModel(store *note.Store, width, height int) NoteModel {
	note, _ := store.GetCurrentNote()

	textEditor := editor.New(80, 20)
	textEditor.SetCursorMode(editor.CursorBlink)
	textEditor.WithTheme(styles.EditorTheme(_styles))
	textEditor.SetLanguage("markdown", styles.EditorLanguageTheme(true))
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

	confirmation.WithTheme(styles.HuhThemeCatppuccin{Styles: _styles})

	return NoteModel{
		store:           store,
		width:           width,
		height:          height,
		help:            helpMenu,
		editor:          textEditor,
		confirmation:    confirmation,
		currentNoteName: note.Name,
	}
}

func (m *NoteModel) Init() tea.Cmd {
	return nil
}

func (m *NoteModel) View() string {
	view := m.editor.View()

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

func (m *NoteModel) Update(msg tea.Msg) (NoteModel, tea.Cmd) {
	var cmds []tea.Cmd

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

					return *m, dispatch(changesDiscardedMsg{})
				}

				m.confirm(false)

				m.editor.Focus()
			}
		}
	}

	if m.fullScreen {
		helpModel, cmd := m.help.Update(msg)
		m.help = helpModel
		m.help.SetSize(m.width, m.height)
		cmds = append(cmds, cmd)
	}

	if m.showConfirmation {
		confirmation, cmd := m.confirmation.Update(msg)
		m.confirmation = confirmation.(*huh.Confirm)
		cmds = append(cmds, cmd)
	} else {
		editorModel, cmd := m.editor.Update(msg)
		m.editor = editorModel
		cmds = append(cmds, cmd)
	}

	return *m, tea.Batch(cmds...)
}

func (m *NoteModel) setSize(width, height int) {
	m.width = width
	m.height = height

	helpHeight := utils.Ternary(m.help.FullView, lipgloss.Height(m.help.View()), 0)

	if m.showConfirmation {
		m.editor.SetSize(width, max(height-helpHeight-lipgloss.Height(m.confirmation.View()), 0))
	} else {
		m.editor.SetSize(width, max(height-helpHeight, 0))
	}
}

func (m *NoteModel) updateContent() {
	if note, ok := m.store.GetCurrentNote(); ok {
		if m.currentNoteName != note.Name {
			m.editor.SetContent(note.Content)
			m.currentNoteName = note.Name

			if err := m.editor.SetCursorPosition(0, 0); err != nil {
				m.error = fmt.Errorf("failed to set cursor position: %w", err)
			}
		}
	}

	m.setSize(m.width, m.height)

	editorModel, _ := m.editor.Update(nil)
	m.editor = editorModel
}

// reloadContent refreshes the editor with the note's content from disk while
// keeping the cursor in place. It returns false and leaves the editor untouched
// when the content changed on disk but the editor has unsaved changes.
func (m *NoteModel) reloadContent() bool {
	note, ok := m.store.GetCurrentNote()
	if !ok || note.Name != m.currentNoteName {
		m.updateContent()
		return true
	}

	if m.editor.GetCurrentContent() == note.Content {
		return true
	}

	if m.hasChanges() {
		return false
	}

	position := m.editor.GetCursorPosition()
	m.editor.SetContent(note.Content)

	lines := strings.Split(note.Content, "\n")
	row := min(position.Row, len(lines)-1)
	col := min(position.Col, utf8.RuneCountInString(lines[row]))
	_ = m.editor.SetCursorPosition(row, col)

	m.updateContent()
	return true
}

func (m *NoteModel) isEditing() bool {
	return m.editor.IsInsertMode()
}

func (m *NoteModel) hasChanges() bool {
	return m.editor.HasChanges()
}

func (m *NoteModel) focus() tea.Cmd {
	m.editor.Focus()
	return m.editor.CursorBlink()
}

func (m *NoteModel) blur() {
	m.editor.Blur()

	if m.editor.IsCommandMode() {
		m.editor.SetNormalMode()
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

func (m *NoteModel) dispatchEditorError(err error) tea.Cmd {
	return m.editor.DispatchError(err, 2*time.Second)
}
