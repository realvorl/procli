package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	cleenet "github.com/realvorl/procli/net"
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

		fmt.Println("Joining", address)

		return cleenet.JoinServer(address, joinSession, joinName)
	},
}

func init() {
	joinCmd.Flags().StringVar(&joinName, "name", "", "Display name for this session")
	joinCmd.Flags().StringVar(&joinSession, "session", "", "Session code to join")
	_ = joinCmd.MarkFlagRequired("session")

	rootCmd.AddCommand(joinCmd)
}
