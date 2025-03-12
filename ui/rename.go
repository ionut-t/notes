package ui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/ionut-t/notes/note"
	"github.com/ionut-t/notes/styles"
)

type renameModel struct {
	store  *note.Store
	input  *huh.Input
	active bool
	width  int
}

func newRenameModel(store *note.Store) renameModel {
	input := huh.NewInput().Placeholder("new name")
	input.WithTheme(styles.ThemeCatppuccin())
	input.Focus()

	return renameModel{
		store: store,
		input: input,
	}
}

func (r *renameModel) SetWidth(width int) {
	r.width = width
}

func (r renameModel) Init() tea.Cmd {
	return nil
}

func (r renameModel) View() string {
	return r.input.View()
}

func (r renameModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if !r.active {
		if keyMsg, ok := msg.(tea.KeyMsg); ok {
			if keyMsg.String() == "r" {
				if note, ok := r.store.GetCurrentNote(); ok {
					r.input.Prompt("Rename: ")
					value := note.Name
					r.input.Value(&value)
					r.active = true
					return r, dispatch(cmdInitMsg{})
				}
			}
		}

		return r, nil
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		r.input.WithWidth(msg.Width)

	case tea.KeyMsg:
		return r.handleRenameNote(msg)
	}

	return r, nil
}

func (r renameModel) handleRenameNote(msg tea.KeyMsg) (renameModel, tea.Cmd) {
	var cmds []tea.Cmd
	keyMsg := tea.KeyMsg(msg).String()

	inputModel, cmd := r.input.Update(msg)
	r.input = inputModel.(*huh.Input)
	cmds = append(cmds, cmd)

	switch keyMsg {
	case "enter":
		newName, err := validateNoteName(r.input)

		if err != nil {
			return r, dispatch(cmdErrorMsg(err))
		}

		note, err := r.store.RenameCurrentNote(newName)

		if err != nil {
			return r, dispatch(cmdErrorMsg(err))
		}

		r.active = false

		empty := ""
		r.input.Value(&empty)

		return r, tea.Sequence(
			dispatch(cmdNoteRenamedMsg{note}),
			dispatch(cmdSuccessMsg(fmt.Sprintf("Note renamed to \"%s\"", note.Name))),
		)

	case "esc":
		r.active = false
		empty := ""
		r.input.Value(&empty)
		return r, dispatch(cmdAbortMsg{})
	}

	return r, tea.Batch(cmds...)
}
