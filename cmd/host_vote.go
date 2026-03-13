package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	cleenet "github.com/realvorl/procli/net"
)

var hostVoteCmd = &cobra.Command{
	Use:   "host-vote",
	Short: "Start a Scrum poker host session",
	RunE: func(cmd *cobra.Command, args []string) error {

		server := cleenet.NewServer("ABCD12")

		fmt.Println("Starting vote host...")

		return server.Listen()
	},
}

func init() {
	rootCmd.AddCommand(hostVoteCmd)
}
