package ui

import (
	"errors"
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/ionut-t/notes/internal/utils"
	"github.com/ionut-t/notes/note"
	"github.com/ionut-t/notes/styles"
)

type cmdInputModel struct {
	store  *note.Store
	input  *huh.Input
	active bool
	width  int
}

func newCmdInputModel(store *note.Store) cmdInputModel {
	cmdInput := huh.NewInput().Prompt(": ")
	cmdInput.WithTheme(styles.ThemeCatppuccin())

	return cmdInputModel{
		store: store,
		input: cmdInput,
	}
}

func (c *cmdInputModel) SetWidth(width int) {
	c.width = width
}

func (c cmdInputModel) Init() tea.Cmd {
	return nil
}

func (c cmdInputModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if !c.active {
		if keyMsg, ok := msg.(tea.KeyMsg); ok {
			if keyMsg.String() == ":" {
				if _, ok := c.store.GetCurrentNote(); ok {
					c.active = true
					return c, dispatch(cmdInitMsg{})
				}
			}
		}

		return c, nil
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		c.input.WithWidth(msg.Width)

	case tea.KeyMsg:
		return c.handleCmdRunner(msg)
	}

	return c, nil
}

func (c cmdInputModel) View() string {
	return c.input.View()
}

func (c cmdInputModel) handleCmdRunner(msg tea.KeyMsg) (cmdInputModel, tea.Cmd) {
	c.input.Focus()

	keyMsg := tea.KeyMsg(msg).String()
	switch keyMsg {
	case "esc":
		c.active = false
		empty := ""
		c.input.Value(&empty)
		return c, dispatch(cmdAbortMsg{})

	case "enter":
		cmdValue := c.input.GetValue().(string)
		cmdValue = strings.TrimSpace(cmdValue)

		if cmdValue == "" {
			return c, nil
		}

		if cmdValue == "q" {
			return c, tea.Quit
		}

		if strings.HasPrefix(cmdValue, "set-editor") {
			return c.handleEditorSetCmd(cmdValue)
		}

		if strings.HasPrefix(cmdValue, "set-v_line") {
			return c.handleVLineCmd(cmdValue)
		}

		return c.handleCopyCmd(cmdValue)
	}

	cmdModel, cmd := c.input.Update(msg)
	c.input = cmdModel.(*huh.Input)

	return c, cmd
}

func (c cmdInputModel) handleCopyCmd(cmdValue string) (cmdInputModel, tea.Cmd) {
	start, end, err := note.ParseCopyLinesCommand(cmdValue)

	if err != nil {
		return c, dispatch(cmdErrorMsg(err))
	}

	if note, ok := c.store.GetCurrentNote(); ok {
		copiedLines, err := c.store.CopyLines(note.Content, start, end)

		if err != nil {
			return c, dispatch(cmdErrorMsg(err))
		}

		successMessage := fmt.Sprintf(
			"Copied %d %s to clipboard from %s",
			copiedLines,
			utils.Ternary(copiedLines == 1, "line", "lines"),
			note.Name,
		)

		c.active = false
		empty := ""
		c.input.Value(&empty)

		return c, dispatch(cmdSuccessMsg(successMessage))
	}

	return c, nil
}

func (c cmdInputModel) handleEditorSetCmd(cmdValue string) (cmdInputModel, tea.Cmd) {
	editor := strings.TrimSpace(strings.TrimPrefix(cmdValue, "set-editor"))

	if editor == "" {
		return c, dispatch(cmdErrorMsg(errors.New("no editor specified")))
	}

	err := c.store.SetEditor(editor)

	if err != nil {
		return c, dispatch(cmdErrorMsg(err))
	}

	successMessage := fmt.Sprintf("Editor set to %s", editor)

	empty := ""
	c.input.Value(&empty)
	c.active = false

	return c, dispatch(cmdSuccessMsg(successMessage))
}

func (c cmdInputModel) handleVLineCmd(cmdValue string) (cmdInputModel, tea.Cmd) {
	value := strings.TrimSpace(strings.TrimPrefix(cmdValue, "set-v_line"))

	var enabled bool

	if value == "" || value == "true" {
		enabled = true
	} else if value == "false" {
		enabled = false
	} else {
		return c, dispatch(cmdErrorMsg(errors.New("invalid value for v_line")))
	}

	if err := c.store.SetDefaultVLineStatus(enabled); err != nil {
		return c, dispatch(cmdErrorMsg(err))
	}

	successMessage := fmt.Sprint(
		utils.Ternary(enabled,
			"Show line numbers in markdown by default",
			"Don't show line numbers in markdown by default",
		),
	)

	empty := ""
	c.input.Value(&empty)
	c.active = false

	return c, tea.Batch(
		dispatch(cmdSuccessMsg(successMessage)),
		dispatch(cmdSetVLineMsg(enabled)),
	)
}
