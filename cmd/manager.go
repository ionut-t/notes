package cmd

import (
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"
	"github.com/ionut-t/notes/note"
	"github.com/ionut-t/notes/ui"
)

func runManagerUI(store *note.Store) {
	m := ui.NewManager(store)
	p := tea.NewProgram(m)

	_, err := p.Run()
	_ = m.Close()

	if err != nil {
		fmt.Printf("Error running UI: %v\n", err)
		os.Exit(1)
	}
}
