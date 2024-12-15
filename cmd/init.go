package cmd

import (
	"fmt"
	"strings"

	bubbletea "github.com/charmbracelet/bubbletea"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

type wizardModel struct {
	step           int
	projectName    string
	requiredTools  []string
	envVars        []string
	tokens         []string
	versionControl string
	done           bool
}

// Messages for Bubble Tea
type tickMsg struct{}
type inputMsg string

func (m wizardModel) Init() bubbletea.Cmd {
	return nil
}

func (m wizardModel) Update(msg bubbletea.Msg) (bubbletea.Model, bubbletea.Cmd) {
	switch msg := msg.(type) {
	case bubbletea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, bubbletea.Quit
		}
	case inputMsg:
		switch m.step {
		case 0:
			m.projectName = string(msg)
		case 1:
			m.requiredTools = parseCommaSeparated(string(msg))
		case 2:
			m.envVars = parseCommaSeparated(string(msg))
		case 3:
			m.tokens = parseCommaSeparated(string(msg))
		case 4:
			m.versionControl = string(msg)
			m.done = true
		}
		m.step++
	}

	return m, nil
}

func (m wizardModel) View() string {
	if m.done {
		return fmt.Sprintf("Setup complete! Project: %s\nPress q to quit.", m.projectName)
	}

	switch m.step {
	case 0:
		return "Enter project name: "
	case 1:
		return "Enter required tools (comma-separated): "
	case 2:
		return "Enter environment variables (comma-separated): "
	case 3:
		return "Enter required tokens (comma-separated): "
	case 4:
		return "Enter version control system (e.g., git): "
	}

	return "Unexpected step. Press q to quit."
}

func parseCommaSeparated(input string) []string {
	var result []string
	for _, item := range strings.Split(input, ",") {
		result = append(result, strings.TrimSpace(item))
	}
	return result
}

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Create a new project configuration",
	Run: func(cmd *cobra.Command, args []string) {
		initialModel := wizardModel{}
		p := bubbletea.NewProgram(initialModel)
		if err := p.Start(); err != nil {
			fmt.Println(color.RedString("Error starting wizard: %s", err))
		}
	},
}

func init() {
	rootCmd.AddCommand(initCmd)
}
