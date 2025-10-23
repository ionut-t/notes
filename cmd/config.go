package cmd

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/ionut-t/notes/internal/config"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func configCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Manage configuration",
		Long:  `Manage the configuration of the notes tool.`,
		Run: func(cmd *cobra.Command, args []string) {
			configPath := config.GetConfigFilePath()

			// Check if flags were provided
			editorFlag, _ := cmd.Flags().GetString("editor")
			storageFlag, _ := cmd.Flags().GetString("storage")

			// Handle flag updates
			flagsSet := false

			// Update string flags

			if editorFlag != "" {
				viper.Set("editor", editorFlag)
				flagsSet = true
				fmt.Println("Editor set to:", editorFlag)
			}

			if storageFlag != "" {
				viper.Set("storage", storageFlag)
				flagsSet = true
				fmt.Println("Storage set to:", storageFlag)
			}

			// Write config if any flags were set
			if flagsSet {
				if err := viper.WriteConfig(); err != nil {
					fmt.Println("Error writing config:", err)
					os.Exit(1)
				}
			}

			// Only open editor if no flags were set
			if !flagsSet {
				openInEditor(configPath)
			}
		},
	}

	cmd.Flags().StringP("editor", "e", "", "Set the editor to use for notes")
	cmd.Flags().StringP("storage", "s", "", "Set the storage path for notes")

	return cmd
}

func openInEditor(configPath string) {
	editor := config.GetEditor()

	cmd := exec.Command(editor, configPath)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		fmt.Println("Error opening editor:", err)
		os.Exit(1)
	}
}
