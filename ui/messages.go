package ui

type editorFinishedMsg struct{}

type clearMsg struct{}

type deleteNoteMsg struct {
	confirmed bool
}
