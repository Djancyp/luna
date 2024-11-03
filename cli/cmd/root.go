package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	tea "github.com/charmbracelet/bubbletea"
)

var (
	// Used for flags.
	cfgFile     string
	userLicense string

	rootCmd = &cobra.Command{
		Use:   "luna-cli",
		Short: "Luna is fullstack framework for building modern web applications",
		Long:  "Luna is fullstack framework for building modern web applications, Use Golang/Echo and React to create SSR web aplications",
	}
)
var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize the workspace",
	Long:  `This command initializes the workspace`,
	Run: func(cmd *cobra.Command, args []string) {
    // how to convert initialModel() to tea.Model
		p := tea.NewProgram(initialModel())
		if _, err := p.Run(); err != nil {
			fmt.Printf("Alas, there's been an error: %v", err)
			os.Exit(1)
		}
	},
}

// Execute executes the root command.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	cobra.OnInitialize()

	// rootCmd.AddCommand(addCmd)
	rootCmd.AddCommand(initCmd)
}
