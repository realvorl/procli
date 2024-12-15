package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/realvorl/procli/pkg"
	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Add a new project to the configuration",
	Run: func(cmd *cobra.Command, args []string) {
		reader := bufio.NewReader(os.Stdin)

		// Load existing configuration
		config := pkg.LoadConfig()

		// Prompt for project name
		fmt.Print("Enter project name: ")
		projectName, _ := reader.ReadString('\n')
		projectName = strings.TrimSpace(projectName)

		// Check if the project already exists
		if _, exists := config.Projects[projectName]; exists {
			fmt.Println("Project already exists in the configuration.")
			return
		}

		fmt.Print("Do you want to set this as the default project? (y/n): ")
		setDefault, _ := reader.ReadString('\n')
		if strings.ToLower(strings.TrimSpace(setDefault)) == "y" {
			config.DefaultProject = projectName
			fmt.Printf("Default project set to '%s'.\n", projectName)
		}

		// Create new project configuration
		projectConfig := pkg.ProjectConfig{}

		// Ask if tools are needed
		fmt.Print("Does your project use tools? (y/n): ")
		usesTools, _ := reader.ReadString('\n')
		if strings.ToLower(strings.TrimSpace(usesTools)) == "y" {
			fmt.Print("Enter required tools (comma-separated): ")
			tools, _ := reader.ReadString('\n')
			projectConfig.RequiredTools = pkg.ParseCommaSeparated(tools)
		}

		// Ask if environment variables are needed
		fmt.Print("Does your project use environment variables? (y/n): ")
		usesEnvVars, _ := reader.ReadString('\n')
		if strings.ToLower(strings.TrimSpace(usesEnvVars)) == "y" {
			fmt.Print("Enter environment variables (comma-separated): ")
			envVars, _ := reader.ReadString('\n')
			projectConfig.EnvironmentVars = pkg.ParseCommaSeparated(envVars)
		}

		// Ask if tokens are needed
		fmt.Print("Does your project use tokens? (y/n): ")
		usesTokens, _ := reader.ReadString('\n')
		if strings.ToLower(strings.TrimSpace(usesTokens)) == "y" {
			fmt.Print("Enter required tokens (comma-separated): ")
			tokens, _ := reader.ReadString('\n')
			projectConfig.RequiredTokens = pkg.ParseCommaSeparated(tokens)
		}

		// Ask if version control is needed
		fmt.Print("Does your project use version control? (y/n): ")
		usesVCS, _ := reader.ReadString('\n')
		if strings.ToLower(strings.TrimSpace(usesVCS)) == "y" {
			fmt.Print("Enter version control system (e.g., git): ")
			vcs, _ := reader.ReadString('\n')
			projectConfig.VersionControl = strings.TrimSpace(vcs)
		}

		// Add to projects
		config.Projects[projectName] = projectConfig

		// Save configuration
		err := pkg.SaveConfig(config)
		if err != nil {
			fmt.Println("Error saving configuration:", err)
			return
		}

		fmt.Println("Project configuration saved!")
	},
}

func init() {
	rootCmd.AddCommand(initCmd)
}
