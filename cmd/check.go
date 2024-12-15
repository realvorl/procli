package cmd

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/fatih/color"
	"github.com/realvorl/procli/pkg"
	"github.com/spf13/cobra"
)

var checkCmd = &cobra.Command{
	Use:   "check [project]",
	Short: "Check project prerequisites",
	Run: func(cmd *cobra.Command, args []string) {
		config := pkg.LoadConfig()
		projectName := config.DefaultProject
		if len(args) < 1 {
			if config.DefaultProject == "" {
				fmt.Println(color.RedString("No project specified and no default project set."))
				return
			}
		} else {
			projectName = args[0]
		}
		// Load configuration
		project, exists := config.Projects[projectName]
		if !exists {
			fmt.Printf(color.RedString("Project '%s' not found in the configuration.\n"), projectName)
			return
		}

		// Perform checks
		fmt.Printf(color.CyanString("Checking prerequisites for project: %s\n"), projectName)

		checkTools(project.RequiredTools)
		checkEnvVars(project.EnvironmentVars)
		checkTokens(project.RequiredTokens)
		checkVersionControl(project.VersionControl)

		fmt.Println(color.GreenString("\nCheck complete!"))
	},
}

// Check tools
func checkTools(tools []string) {
	if len(tools) == 0 {
		return
	}
	fmt.Println(color.YellowString("\nChecking required tools:"))
	for _, tool := range tools {
		status := true
		if _, err := exec.LookPath(tool); err != nil {
			status = false
		}
		pkg.PrintCheckResult(tool, status)
	}
}

// Check environment variables
func checkEnvVars(vars []string) {
	if len(vars) == 0 {
		return
	}
	fmt.Println(color.YellowString("\nChecking environment variables:"))
	for _, envVar := range vars {
		status := os.Getenv(envVar) != ""
		pkg.PrintCheckResult(envVar, status)
	}
}

// Check tokens
func checkTokens(tokens []string) {
	if len(tokens) == 0 {
		return
	}
	fmt.Println(color.YellowString("\nChecking required tokens(%s):", len(tokens)))
	for _, token := range tokens {
		status := os.Getenv(token) != ""
		pkg.PrintCheckResult(token, status)
	}
}

// Check version control
func checkVersionControl(vcs string) {
	fmt.Println(color.YellowString("\nChecking version control system:"))
	status := true
	if _, err := exec.LookPath(vcs); err != nil {
		status = false
	}
	pkg.PrintCheckResult(vcs, status)
}

func init() {
	rootCmd.AddCommand(checkCmd)
}
