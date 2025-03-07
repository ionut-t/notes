package cmd

import (
	"fmt"
	"os"

	"github.com/ionut-t/notes/note"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "notes",
	Short: "A simple notes manager",
	Long:  `A simple CLI tool to manage your notes with add, list, view, edit, and delete functionality.`,
	Run: func(cmd *cobra.Command, args []string) {
		store, err := note.NewNotesStore("")
		if err != nil {
			fmt.Printf("Error initializing notes store: %v\n", err)
			os.Exit(1)
		}

		runListUI(store)
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	store, err := note.NewNotesStore("")
	if err != nil {
		fmt.Printf("Error initializing notes store: %v\n", err)
		os.Exit(1)
	}

	rootCmd.AddCommand(newAddCmd(store))
	rootCmd.AddCommand(listCmd(store))

	err = rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	// rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.notes.yaml)")

	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	// rootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
