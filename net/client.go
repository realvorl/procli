package net

import (
	"bufio"
	"encoding/json"
	"fmt"
	stdnet "net"
	"os"
)

func JoinServer(address string, session string, name string) error {

	conn, err := stdnet.Dial("tcp", address)
	if err != nil {
		return err
	}

	fmt.Println("Connected to", address)

	join := Join{
		Type:    "join",
		Session: session,
		Name:    name,
	}

	if err := writeJSONLine(conn, join); err != nil {
		return err
	}

	reader := bufio.NewReader(conn)

	for {
		line, err := reader.ReadBytes('\n')
		if err != nil {
			fmt.Println("connection closed")
			os.Exit(0)
		}

		var msg Message
		if err := json.Unmarshal(line, &msg); err != nil {
			fmt.Println("invalid message:", err)
			continue
		}

		switch msg.Type {

		case "welcome":
			var w Welcome
			json.Unmarshal(line, &w)
			fmt.Println("Welcome! ClientID:", w.ClientID)

		case "state":
			var s State
			json.Unmarshal(line, &s)

			fmt.Println("Session:", s.Session)

			if s.StoryTitle != "" {
				fmt.Println("Story:", s.StoryTitle)
			}
			if s.StoryURL != "" {
				fmt.Println("URL:", s.StoryURL)
			}

			fmt.Println("Current clients:")

			for _, c := range s.Clients {
				fmt.Printf(" - %s\n", c.Name)
			}

		case "error":
			var e ErrorMessage
			json.Unmarshal(line, &e)
			fmt.Printf("Server error: %s (%s)\n", e.Message, e.Code)
			return nil

		default:
			fmt.Println("unknown message:", string(line))
		}
	}
}
