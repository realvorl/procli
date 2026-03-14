package cmd

import (
	"github.com/spf13/cobra"

	pokerui "github.com/realvorl/procli/ui/poker"
)

var joinName string
var joinSession string

var joinCmd = &cobra.Command{
	Use:   "join [host]",
	Short: "Join a Scrum poker session",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		host := args[0]
		address := host + ":32896"
		return pokerui.RunClient(address, joinSession, joinName)
	},
}

func init() {
	joinCmd.Flags().StringVar(&joinName, "name", "", "Display name for this session")
	joinCmd.Flags().StringVar(&joinSession, "session", "", "Session code to join")
	_ = joinCmd.MarkFlagRequired("session")

	rootCmd.AddCommand(joinCmd)
}
