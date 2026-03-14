package net

import (
	"bufio"
	"encoding/json"
	"fmt"
	stdnet "net"
	"time"
)

func JoinServer(address string, session string, name string) error {
	return JoinServerWithHandler(address, session, name, nil)
}

type ClientEvent struct {
	Kind    string
	Time    time.Time
	Message string
	Welcome Welcome
	State   State
	Error   ErrorMessage
}

func JoinServerWithHandler(address string, session string, name string, handler func(ClientEvent)) error {
	conn, err := stdnet.Dial("tcp", address)
	if err != nil {
		emitClientEvent(handler, ClientEvent{
			Kind:    "error",
			Time:    time.Now(),
			Message: err.Error(),
		})
		return err
	}
	defer conn.Close()

	if handler == nil {
		fmt.Println("Connected to", address)
	}
	emitClientEvent(handler, ClientEvent{
		Kind:    "connected",
		Time:    time.Now(),
		Message: fmt.Sprintf("connected to %s", address),
	})

	join := Join{
		Type:    "join",
		Session: session,
		Name:    name,
	}

	if err := writeJSONLine(conn, join); err != nil {
		emitClientEvent(handler, ClientEvent{
			Kind:    "error",
			Time:    time.Now(),
			Message: fmt.Sprintf("join send failed: %v", err),
		})
		return err
	}
	emitClientEvent(handler, ClientEvent{
		Kind:    "join_sent",
		Time:    time.Now(),
		Message: "join request sent",
	})

	reader := bufio.NewReader(conn)

	for {
		line, err := reader.ReadBytes('\n')
		if err != nil {
			if handler == nil {
				fmt.Println("connection closed")
			}
			emitClientEvent(handler, ClientEvent{
				Kind:    "disconnected",
				Time:    time.Now(),
				Message: "connection closed",
			})
			return nil
		}

		var msg Message
		if err := json.Unmarshal(line, &msg); err != nil {
			if handler == nil {
				fmt.Println("invalid message:", err)
			}
			emitClientEvent(handler, ClientEvent{
				Kind:    "error",
				Time:    time.Now(),
				Message: fmt.Sprintf("invalid message: %v", err),
			})
			continue
		}

		switch msg.Type {

		case "welcome":
			var w Welcome
			json.Unmarshal(line, &w)
			if handler == nil {
				fmt.Println("Welcome! ClientID:", w.ClientID)
			}
			emitClientEvent(handler, ClientEvent{
				Kind:    "welcome",
				Time:    time.Now(),
				Message: fmt.Sprintf("welcome as %s", w.ClientID),
				Welcome: w,
			})

		case "state":
			var s State
			json.Unmarshal(line, &s)

			if handler == nil {
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
			}
			emitClientEvent(handler, ClientEvent{
				Kind:    "state",
				Time:    time.Now(),
				Message: fmt.Sprintf("received state: %d client(s)", len(s.Clients)),
				State:   s,
			})

		case "error":
			var e ErrorMessage
			json.Unmarshal(line, &e)
			if handler == nil {
				fmt.Printf("Server error: %s (%s)\n", e.Message, e.Code)
			}
			emitClientEvent(handler, ClientEvent{
				Kind:    "error",
				Time:    time.Now(),
				Message: fmt.Sprintf("server error: %s (%s)", e.Message, e.Code),
				Error:   e,
			})
			return nil

		default:
			if handler == nil {
				fmt.Println("unknown message:", string(line))
			}
			emitClientEvent(handler, ClientEvent{
				Kind:    "unknown",
				Time:    time.Now(),
				Message: fmt.Sprintf("unknown message: %s", string(line)),
			})
		}
	}
}

func emitClientEvent(handler func(ClientEvent), event ClientEvent) {
	if handler != nil {
		handler(event)
	}
}
