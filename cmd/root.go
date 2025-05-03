package cmd

import (
	"fmt"
	"os"
	"runtime/debug"
	"strings"
	"time"

	"github.com/ionut-t/notes/internal/config"
	"github.com/ionut-t/notes/note"
	"github.com/spf13/cobra"
)

// Version information - these will be set during the build process
var (
	version = "dev"
	commit  = "unknown"
	date    = "unknown"
)

const logo = "  \033[31m" + `
   _   _    ___    _____   _____   ____  
  | \ | |  / _ \  |_   _| | ____| / ___| 
  |  \| | | | | |   | |   |  _|   \___ \ 
  | |\  | | |_| |   | |   | |___   ___) |
  |_| \_|  \___/    |_|   |_____| |____/                  	 
` + "\033[0m"

var rootCmd = &cobra.Command{
	Use:     "notes",
	Short:   "A simple notes manager",
	Long:    `A simple CLI tool for managing notes`,
	Version: version,
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

// init function to set version information from build info when available
func init() {
	// Try to get version info from Go module information
	if info, ok := debug.ReadBuildInfo(); ok {
		// Look for version in main module
		if info.Main.Version != "(devel)" && info.Main.Version != "" {
			version = strings.TrimPrefix(info.Main.Version, "v")
		}

		// Look for VCS info in settings
		for _, setting := range info.Settings {
			switch setting.Key {
			case "vcs.revision":
				if len(setting.Value) > 7 {
					commit = setting.Value[:7]
				} else if setting.Value != "" {
					commit = setting.Value
				}
			case "vcs.time":
				if t, err := time.Parse(time.RFC3339, setting.Value); err == nil {
					date = t.Format("02/01/2006")
				}
			}
		}
	}
}

func init() {
	versionTemplate := logo + `
  Version                    %s
  Commit                     %s
  Release date	             %s
`
	versionTemplate = fmt.Sprintf(versionTemplate, version, commit, date)

	rootCmd.SetVersionTemplate(versionTemplate)
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
