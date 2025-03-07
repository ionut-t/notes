package cmd

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/ionut-t/notes/note"
	"github.com/ionut-t/notes/ui"
	"github.com/spf13/cobra"
)

func listCmd(store *note.NotesStore) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all notes",
		Long:  `List all notes in your collection.`,
		Run: func(cmd *cobra.Command, args []string) {
			runListUI(store)
		},
	}

	return cmd
}

func runListUI(store *note.NotesStore) {
	m := ui.NewListModel(store)
	p := tea.NewProgram(m, tea.WithAltScreen(), tea.WithMouseAllMotion())

	if _, err := p.Run(); err != nil {
		fmt.Printf("Error running UI: %v\n", err)
		os.Exit(1)
	}
}
