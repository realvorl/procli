package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/realvorl/procli/core"
	cleenet "github.com/realvorl/procli/net"
)

var hostStory string
var hostStoryURL string

var hostVoteCmd = &cobra.Command{
	Use:   "host-vote",
	Short: "Start a Scrum poker host session",
	RunE: func(cmd *cobra.Command, args []string) error {
		session := core.GenerateSessionCode(6)
		server := cleenet.NewServer(session, hostStory, hostStoryURL)

		fmt.Println("Starting vote host...")

		return server.Listen()
	},
}

func init() {
	hostVoteCmd.Flags().StringVar(&hostStory, "story", "", "Story title for this session")
	hostVoteCmd.Flags().StringVar(&hostStoryURL, "url", "", "Optional story URL")
	rootCmd.AddCommand(hostVoteCmd)
}
