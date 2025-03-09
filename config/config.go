package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

func getDefaultEditor() string {
	if editor := os.Getenv("EDITOR"); editor != "" {
		return editor
	}

	if os.Getenv("WINDIR") != "" {
		return "notepad"
	}

	return "vim"
}

func GetEditor() string {
	editor := viper.GetString("editor")

	if editor == "" {
		return getDefaultEditor()
	}

	return editor
}

func GetStorage() string {
	storage := viper.GetString("storage")

	if storage != "" {
		return storage
	}

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

	return notesDir
}

func GetVLineEnabledByDefault() bool {
	return viper.GetBool("v_line")
}

func SetEditor(editor string) error {
	if err := checkConfigFile(); err != nil {
		return err
	}

	if editor == GetEditor() {
		return nil
	}

	viper.Set("editor", editor)

	return viper.WriteConfig()
}

func SetDefaultVLineStatus(enabled bool) error {
	if err := checkConfigFile(); err != nil {
		return err
	}

	if enabled == GetVLineEnabledByDefault() {
		return nil
	}

	viper.Set("v_line", enabled)
	return viper.WriteConfig()
}

func checkConfigFile() error {
	configPath := viper.ConfigFileUsed()

	if configPath == "" {
		return fmt.Errorf("config file not found; close program and run `notes config`")
	}

	return nil
}
