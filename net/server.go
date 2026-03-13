package net

import (
	"bufio"
	"encoding/json"
	"fmt"
	stdnet "net"
	"sync"
)

const DefaultPort = 32896

type Client struct {
	ID   string
	Name string
	Conn stdnet.Conn
}

type Server struct {
	Session   string
	Clients   map[string]*Client
	mu        sync.Mutex
	nextID    int
	nextGuest int
}

func NewServer(session string) *Server {
	return &Server{
		Session: session,
		Clients: make(map[string]*Client),
		nextID:  1,
	}
}

func (s *Server) Listen() error {
	addr := fmt.Sprintf(":%d", DefaultPort)
	ln, err := stdnet.Listen("tcp", addr)
	if err != nil {
		return err
	}
	defer ln.Close()

	fmt.Printf("Starting session %s on %s\n", s.Session, addr)

	for {
		conn, err := ln.Accept()
		if err != nil {
			fmt.Println("accept error:", err)
			continue
		}

		go s.handleConn(conn)
	}
}
func (s *Server) handleConn(conn stdnet.Conn) {
	reader := bufio.NewReader(conn)

	line, err := reader.ReadBytes('\n')
	if err != nil {
		fmt.Println("read error:", err)
		_ = conn.Close()
		return
	}

	var join Join
	if err := json.Unmarshal(line, &join); err != nil {
		fmt.Println("invalid join message:", err)
		_ = conn.Close()
		return
	}

	if join.Type != "join" {
		fmt.Println("unexpected message type:", join.Type)
		_ = conn.Close()
		return
	}

	if join.Session != s.Session {
		_ = writeJSONLine(conn, ErrorMessage{
			Type:    "error",
			Session: s.Session,
			Code:    "invalid_session",
			Message: "session code does not match",
		})
		_ = conn.Close()
		return
	}

	client := s.registerClient(join.Name, conn)

	fmt.Printf("Client joined: %s\n", client.Name)

	if err := s.sendWelcome(client); err != nil {
		fmt.Println("welcome error:", err)
		return
	}

	if err := s.broadcastState(); err != nil {
		fmt.Println("broadcast state error:", err)
		return
	}

	// --- keep connection open ---
	for {
		_, err := reader.ReadBytes('\n')
		if err != nil {
			fmt.Printf("Client disconnected: %s\n", client.Name)
			s.removeClient(client.ID)
			_ = conn.Close()
			return
		}
	}
}

func (s *Server) registerClient(name string, conn stdnet.Conn) *Client {
	s.mu.Lock()
	defer s.mu.Unlock()

	if name == "" {
		name = fmt.Sprintf("Guest_%d", s.nextGuest)
		s.nextGuest++
	}

	id := fmt.Sprintf("c%d", s.nextID)
	s.nextID++

	client := &Client{
		ID:   id,
		Name: name,
		Conn: conn,
	}

	s.Clients[id] = client
	return client
}

func (s *Server) removeClient(id string) {

	s.mu.Lock()
	delete(s.Clients, id)
	s.mu.Unlock()

	s.broadcastState()
}

func (s *Server) sendWelcome(client *Client) error {
	msg := Welcome{
		Type:     "welcome",
		Session:  s.Session,
		ClientID: client.ID,
		Role:     "client",
	}

	return writeJSONLine(client.Conn, msg)
}

func (s *Server) broadcastState() error {
	s.mu.Lock()
	clients := make([]ClientInfo, 0, len(s.Clients))
	conns := make([]stdnet.Conn, 0, len(s.Clients))

	for _, c := range s.Clients {
		clients = append(clients, ClientInfo{
			ID:    c.ID,
			Name:  c.Name,
			Voted: false,
		})
		conns = append(conns, c.Conn)
	}
	s.mu.Unlock()

	state := State{
		Type:    "state",
		Session: s.Session,
		Phase:   "lobby",
		Clients: clients,
	}

	for _, conn := range conns {
		if err := writeJSONLine(conn, state); err != nil {
			return err
		}
	}

	return nil
}

func writeJSONLine(conn stdnet.Conn, v any) error {
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}

	data = append(data, '\n')
	_, err = conn.Write(data)
	return err
}
