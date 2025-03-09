package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/ionut-t/notes/config"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func configCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Manage configuration",
		Long:  `Manage the configuration of the notes tool.`,
		Run: func(cmd *cobra.Command, args []string) {
			configPath := ensureConfigExists()

			// Check if flags were provided
			editorFlag, _ := cmd.Flags().GetString("editor")
			storageFlag, _ := cmd.Flags().GetString("storage")
			vLineFlag, _ := cmd.Flags().GetBool("v-line")

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

			if cmd.Flags().Changed("v-line") {
				viper.Set("v_line", vLineFlag)
				flagsSet = true
				fmt.Println("Show line numbers in markdown by default:", vLineFlag)
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
	cmd.Flags().Bool("v-line", false, "Show line numbers in markdown by default")

	return cmd
}

func ensureConfigExists() string {
	configPath := viper.ConfigFileUsed()

	if configPath == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			fmt.Println("Error getting home directory:", err)
			os.Exit(1)
		}

		notesDir := filepath.Join(home, ".notes")
		if err := os.MkdirAll(notesDir, 0755); err != nil {
			fmt.Println("Error creating directory:", err)
			os.Exit(1)
		}

		configPath = filepath.Join(notesDir, ".config.toml")
		viper.SetConfigFile(configPath)

		if _, err := os.Stat(configPath); os.IsNotExist(err) {
			viper.SetDefault("editor", config.GetEditor())
			viper.SetDefault("storage", notesDir)
			viper.SetDefault("v_line", false)

			if err := viper.WriteConfig(); err != nil {
				fmt.Println("Error writing config:", err)
				os.Exit(1)
			}

			fmt.Println("Created config at", configPath)
		} else {
			// File exists but Viper couldn't find it, so explicitly set it
			viper.SetConfigFile(configPath)
			_ = viper.ReadInConfig()
		}
	}

	return configPath
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
