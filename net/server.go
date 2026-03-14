package net

import (
	"bufio"
	"encoding/json"
	"fmt"
	stdnet "net"
	"sync"
	"time"
)

const DefaultPort = 32896

type Client struct {
	ID   string
	Name string
	Conn stdnet.Conn
}

type Server struct {
	Session    string
	StoryTitle string
	StoryURL   string
	Clients    map[string]*Client
	mu         sync.Mutex
	nextID     int
	nextGuest  int
	onEvent    func(ServerEvent)
}

type ServerEvent struct {
	Kind    string
	Time    time.Time
	Message string
	State   State
}

func NewServer(session string, storyTitle string, storyURL string) *Server {
	return &Server{
		Session:    session,
		StoryTitle: storyTitle,
		StoryURL:   storyURL,
		Clients:    make(map[string]*Client),
		nextID:     1,
		nextGuest:  1,
	}
}

func (s *Server) SetEventHandler(handler func(ServerEvent)) {
	s.mu.Lock()
	s.onEvent = handler
	s.mu.Unlock()
}

func (s *Server) Listen() error {
	addr := fmt.Sprintf(":%d", DefaultPort)
	ln, err := stdnet.Listen("tcp", addr)
	if err != nil {
		s.emitEvent(ServerEvent{
			Kind:    "error",
			Time:    time.Now(),
			Message: fmt.Sprintf("listen error: %v", err),
		})
		return err
	}
	defer ln.Close()

	s.printf("Starting session %s on %s\n", s.Session, addr)
	s.emitEvent(ServerEvent{
		Kind:    "started",
		Time:    time.Now(),
		Message: fmt.Sprintf("session %s started on %s", s.Session, addr),
	})

	for {
		conn, err := ln.Accept()
		if err != nil {
			s.println("accept error:", err)
			continue
		}

		go s.handleConn(conn)
	}
}
func (s *Server) handleConn(conn stdnet.Conn) {
	reader := bufio.NewReader(conn)

	line, err := reader.ReadBytes('\n')
	if err != nil {
		s.println("read error:", err)
		s.emitEvent(ServerEvent{
			Kind:    "error",
			Time:    time.Now(),
			Message: fmt.Sprintf("read error: %v", err),
		})
		_ = conn.Close()
		return
	}

	var join Join
	if err := json.Unmarshal(line, &join); err != nil {
		s.println("invalid join message:", err)
		s.emitEvent(ServerEvent{
			Kind:    "error",
			Time:    time.Now(),
			Message: fmt.Sprintf("invalid join message: %v", err),
		})
		_ = conn.Close()
		return
	}

	if join.Type != "join" {
		s.println("unexpected message type:", join.Type)
		s.emitEvent(ServerEvent{
			Kind:    "error",
			Time:    time.Now(),
			Message: fmt.Sprintf("unexpected message type: %s", join.Type),
		})
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
		s.emitEvent(ServerEvent{
			Kind:    "rejected",
			Time:    time.Now(),
			Message: fmt.Sprintf("rejected client %q: invalid session", join.Name),
		})
		_ = conn.Close()
		return
	}

	client := s.registerClient(join.Name, conn)

	s.printf("Client joined: %s\n", client.Name)
	s.emitEvent(ServerEvent{
		Kind:    "client_joined",
		Time:    time.Now(),
		Message: fmt.Sprintf("client joined: %s", client.Name),
	})

	if err := s.sendWelcome(client); err != nil {
		s.println("welcome error:", err)
		s.emitEvent(ServerEvent{
			Kind:    "error",
			Time:    time.Now(),
			Message: fmt.Sprintf("welcome error: %v", err),
		})
		return
	}

	if err := s.broadcastState(); err != nil {
		s.println("broadcast state error:", err)
		s.emitEvent(ServerEvent{
			Kind:    "error",
			Time:    time.Now(),
			Message: fmt.Sprintf("broadcast state error: %v", err),
		})
		return
	}

	// --- keep connection open ---
	for {
		_, err := reader.ReadBytes('\n')
		if err != nil {
			s.printf("Client disconnected: %s\n", client.Name)
			s.emitEvent(ServerEvent{
				Kind:    "client_disconnected",
				Time:    time.Now(),
				Message: fmt.Sprintf("client disconnected: %s", client.Name),
			})
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

	state := State{
		Type:       "state",
		Session:    s.Session,
		Phase:      "lobby",
		StoryTitle: s.StoryTitle,
		StoryURL:   s.StoryURL,
		Clients:    clients,
	}
	s.mu.Unlock()
	s.emitEvent(ServerEvent{
		Kind:    "state",
		Time:    time.Now(),
		Message: fmt.Sprintf("state broadcast: %d client(s)", len(clients)),
		State:   state,
	})

	for _, conn := range conns {
		if err := writeJSONLine(conn, state); err != nil {
			return err
		}
	}

	return nil
}

func (s *Server) emitEvent(event ServerEvent) {
	s.mu.Lock()
	handler := s.onEvent
	s.mu.Unlock()

	if handler != nil {
		handler(event)
	}
}

func (s *Server) println(a ...any) {
	s.mu.Lock()
	hasHandler := s.onEvent != nil
	s.mu.Unlock()
	if !hasHandler {
		fmt.Println(a...)
	}
}

func (s *Server) printf(format string, a ...any) {
	s.mu.Lock()
	hasHandler := s.onEvent != nil
	s.mu.Unlock()
	if !hasHandler {
		fmt.Printf(format, a...)
	}
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
