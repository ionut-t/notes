package cmd

import (
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"
	"github.com/ionut-t/notes/note"
	"github.com/ionut-t/notes/ui"
	"github.com/spf13/cobra"
)

func newAddCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add",
		Short: "Add a new note",
		Long:  `Add a new note to your collection.`,
		Run: func(cmd *cobra.Command, args []string) {
			store := note.NewStore()
			runAddUI(store)
		},
	}

	return cmd
}

func runAddUI(store *note.Store) {
	if _, err := store.LoadNotes(); err != nil {
		fmt.Printf("Error loading notes: %v\n", err)
		os.Exit(1)
	}

	p := tea.NewProgram(ui.NewAddModel(store))
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error running UI: %v\n", err)
		os.Exit(1)
	}
}
