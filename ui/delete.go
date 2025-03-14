package ui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/ionut-t/notes/note"
	"github.com/ionut-t/notes/styles"
)

type deleteModel struct {
	store  *note.Store
	active bool
	width  int
}

func newDelete(store *note.Store) deleteModel {
	return deleteModel{
		store: store,
		width: 40,
	}
}

func (m deleteModel) Init() tea.Cmd {
	return nil
}

func (m deleteModel) View() string {
	if !m.active {
		return ""
	}

	bg := styles.Crust.GetBackground()

	question := styles.Text.
		Background(bg).
		Width(m.width).
		Align(lipgloss.Center).
		Padding(1, 2).
		Render(fmt.Sprintf(
			"Are you sure you want to delete %s",
			styles.Accent.Background(bg).Render(m.getCurrentNoteName())+styles.Crust.Render("?"),
		))

	options := styles.Text.
		Background(bg).
		Width(m.width).
		Align(lipgloss.Center).
		Padding(0, 2, 1).
		Render(
			styles.Error.Background(bg).Render("[Y]es") + styles.Crust.Render(" | ") + styles.Crust.Render("[N]o"),
		)

	content := lipgloss.JoinVertical(lipgloss.Center, question, options)

	container := styles.Crust.
		Width(m.width).
		Align(lipgloss.Center).
		Render(content)

	return lipgloss.PlaceHorizontal(m.width, lipgloss.Center, container)
}

func (m deleteModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		if !m.active {
			if keyMsg.String() == "ctrl+d" {
				if _, ok := m.store.GetCurrentNote(); ok {
					m.active = true
					return m, dispatch(cmdInitMsg{})
				}
			}

			return m, nil
		}

		switch keyMsg.String() {
		case "y", "Y":
			return m.executeNoteDeletion()
		case "n", "N", "esc", "q":
			m.active = false
			return m, dispatch(cmdAbortMsg{})
		default:
			return m, nil
		}
	}

	return m, nil
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

func (m deleteModel) getCurrentNoteName() string {
	if note, ok := m.store.GetCurrentNote(); ok {
		return note.Name
	}

	return ""
}
