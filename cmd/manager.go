package cmd

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/ionut-t/notes/note"
	"github.com/ionut-t/notes/ui"
)

func runManagerUI(store *note.Store) {
	m := ui.NewManager(store)
	p := tea.NewProgram(m, tea.WithAltScreen(), tea.WithMouseAllMotion())

	if _, err := p.Run(); err != nil {
		fmt.Printf("Error running UI: %v\n", err)
		os.Exit(1)
	}
}
