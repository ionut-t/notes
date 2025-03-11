package cmd

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
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
	p := tea.NewProgram(ui.NewAddModel(store) /* , tea.WithAltScreen() */)
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error running UI: %v\n", err)
		os.Exit(1)
	}
}
