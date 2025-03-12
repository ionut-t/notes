package ui

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/ionut-t/notes/note"
)

type editorClosedMsg struct{}

type clearMsg struct{}

type cmdInitMsg struct{}

type cmdSuccessMsg string

type cmdErrorMsg error

type cmdAbortMsg struct{}

type cmdNoteRenamedMsg struct {
	note note.Note
}

type cmdNoteDeletedMsg struct{}

func dispatch(msg tea.Msg) tea.Cmd {
	return func() tea.Msg {
		return msg
	}
}

func dispatchClearMsg() tea.Cmd {
	return tea.Tick(time.Second*2, func(t time.Time) tea.Msg {
		return clearMsg{}
	})
}
