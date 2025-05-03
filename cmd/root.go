package cmd

import (
	"fmt"
	"os"

	"github.com/ionut-t/notes/internal/config"
	"github.com/ionut-t/notes/note"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "notes",
	Short: "A simple notes manager",
	Long:  `A simple CLI tool for managing notes`,
	Run: func(cmd *cobra.Command, args []string) {
		store := note.NewStore()
		runManagerUI(store)
	},
}

func Execute() {
	rootCmd.AddCommand(configCmd())
	rootCmd.AddCommand(newAddCmd())

	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	var cfgFile string
	rootCmd.PersistentFlags().StringVar(&cfgFile, "set-config", "", "config file (default is $HOME/.notes/.config.toml)")

}

func initConfig() {
	if _, err := config.InitialiseConfigFile(); err != nil {
		fmt.Printf("Error initializing config: %v\n", err)
	}
}
