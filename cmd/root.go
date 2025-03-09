package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/ionut-t/notes/note"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var cfgFile string

var rootCmd = &cobra.Command{
	Use:   "notes",
	Short: "A simple notes manager",
	Long:  `A simple CLI tool to manage your notes with add, list, view, edit, and delete functionality.`,
	Run: func(cmd *cobra.Command, args []string) {
		store := note.NewNotesStore()
		runListUI(store)
	},
}

func Execute() {
	rootCmd.AddCommand(configCmd())
	rootCmd.AddCommand(newAddCmd())
	rootCmd.AddCommand(listCmd())

	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVar(&cfgFile, "set-config", "", "config file (default is $HOME/.notes/.config.toml)")

}

func initConfig() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory
		home, err := os.UserHomeDir()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		dir := filepath.Join(home, ".notes")
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			os.Mkdir(dir, 0755)
		}

		viper.AddConfigPath(dir)
		viper.SetConfigType("toml")
		viper.SetConfigName(".config")
	}

	// Read in environment variables that match
	viper.AutomaticEnv()

	// If a config file is found, read it in
	// Silently continue if it doesn't exist
	_ = viper.ReadInConfig()
}
