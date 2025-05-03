package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

const notesDir = ".notes"

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

	dir := filepath.Join(home, notesDir)
	if err := os.MkdirAll(dir, 0755); err != nil {
		fmt.Println("Error creating directory:", err)
		os.Exit(1)
	}

	return dir
}

func GetVLineEnabledByDefault() bool {
	return viper.GetBool("v_line")
}

func SetEditor(editor string) error {
	if _, err := InitialiseConfigFile(); err != nil {
		return err
	}

	if editor == GetEditor() {
		return nil
	}

	viper.Set("editor", editor)

	return viper.WriteConfig()
}

func SetDefaultVLineStatus(enabled bool) error {
	if _, err := InitialiseConfigFile(); err != nil {
		return err
	}

	if enabled == GetVLineEnabledByDefault() {
		return nil
	}

	viper.Set("v_line", enabled)
	return viper.WriteConfig()
}

func InitialiseConfigFile() (string, error) {
	configPath := viper.ConfigFileUsed()

	if configPath == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}

		dir := filepath.Join(home, notesDir)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return "", err
		}

		configPath = filepath.Join(dir, ".config.toml")
		viper.SetConfigFile(configPath)

		if _, err := os.Stat(configPath); os.IsNotExist(err) {
			viper.SetDefault("editor", GetEditor())
			viper.SetDefault("storage", dir)
			viper.SetDefault("v_line", false)

			if err := viper.WriteConfig(); err != nil {
				return "", err
			}

			fmt.Println("Created config at", configPath)
		} else {
			// File exists but Viper couldn't find it, so explicitly set it
			viper.SetConfigFile(configPath)
			_ = viper.ReadInConfig()
		}
	}

	return configPath, nil
}

func GetConfigFilePath() string {
	return viper.ConfigFileUsed()
}
