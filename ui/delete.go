package ui

import (
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	"github.com/ionut-t/notes/internal/keymap"
	"github.com/ionut-t/notes/note"
	"github.com/ionut-t/notes/styles"
)

type deleteModel struct {
	store        *note.Store
	active       bool
	width        int
	confirmation *huh.Confirm
}

func newDelete(store *note.Store) deleteModel {
	confirmation := huh.NewConfirm().
		Affirmative("Yes").
		Negative("No")

	confirmation.WithKeyMap(&huh.KeyMap{
		Confirm: huh.NewDefaultKeyMap().Confirm,
	})

	confirmation.WithTheme(styles.ThemeCatppuccin())

	return deleteModel{
		store:        store,
		width:        40,
		confirmation: confirmation,
	}
}

func (m deleteModel) Init() tea.Cmd {
	return nil
}

func (m deleteModel) View() string {
	if !m.active {
		return ""
	}

	confirmation := lipgloss.NewStyle().
		Width(m.width).
		Align(lipgloss.Center).
		Padding(1, 2).
		Render(m.confirmation.View())

	content := lipgloss.JoinVertical(lipgloss.Center, confirmation)

	container := lipgloss.NewStyle().
		Width(m.width).
		Align(lipgloss.Center).
		Render(content)

	return lipgloss.PlaceHorizontal(m.width, lipgloss.Center, container)
}

func (m deleteModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if !m.active {
		return m, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, keymap.Accept):
			return m.executeNoteDeletion()

		case key.Matches(msg, keymap.Reject), key.Matches(msg, keymap.Quit):
			m.active = false
			return m, dispatch(cmdAbortMsg{})

		case key.Matches(msg, keymap.Save):
			if m.confirmation.GetValue().(bool) {
				return m.executeNoteDeletion()
			}
			m.active = false
			return m, dispatch(cmdAbortMsg{})
		}
	}

	var cmds []tea.Cmd
	if m.active {
		confirmation, cmd := m.confirmation.Update(msg)
		m.confirmation = confirmation.(*huh.Confirm)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m deleteModel) executeNoteDeletion() (deleteModel, tea.Cmd) {
	err := m.store.DeleteCurrentNote()

	if err != nil {
		return m, dispatch(cmdErrorMsg(err))
	}

	m.active = false

	return m, tea.Sequence(
		dispatch(cmdNoteDeletedMsg{}),
		dispatch(cmdSuccessMsg("Note successfully deleted")),
	)
}

func (m *deleteModel) setActive() {
	if note, ok := m.store.GetCurrentNote(); ok {
		m.active = true
		question := styles.Text.Render(
			"Are you sure you want to delete " +
				styles.Primary.Render(note.Name) + "?",
		)
		m.confirmation.Title(question)
	}
}
