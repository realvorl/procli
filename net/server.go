package net

import (
	"bufio"
	"encoding/json"
	"fmt"
	stdnet "net"
	"sort"
	"strings"
	"sync"
	"time"
)

const DefaultPort = 32896

type Client struct {
	ID   string
	Name string
	Conn stdnet.Conn
}

type roundState struct {
	ID            int
	Title         string
	URL           string
	Phase         string
	Reveal        bool
	FinalEstimate string
	Votes         map[string]string
	UpdatedAt     time.Time
}

type Server struct {
	Session    string
	StoryTitle string
	StoryURL   string
	Clients    map[string]*Client
	mu         sync.Mutex
	nextID     int
	nextGuest  int
	nextRound  int
	onEvent    func(ServerEvent)
	rounds     []roundState
	currentIdx int
}

type ServerEvent struct {
	Kind    string
	Time    time.Time
	Message string
	State   State
}

func NewServer(session string, storyTitle string, storyURL string) *Server {
	title := strings.TrimSpace(storyTitle)
	if title == "" {
		title = "Untitled story"
	}
	initial := roundState{
		ID:        1,
		Title:     title,
		URL:       storyURL,
		Phase:     "voting",
		Reveal:    false,
		Votes:     make(map[string]string),
		UpdatedAt: time.Now(),
	}

	return &Server{
		Session:    session,
		StoryTitle: storyTitle,
		StoryURL:   storyURL,
		Clients:    make(map[string]*Client),
		nextID:     1,
		nextGuest:  1,
		nextRound:  2,
		rounds:     []roundState{initial},
		currentIdx: 0,
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
		s.emitEvent(ServerEvent{Kind: "error", Time: time.Now(), Message: fmt.Sprintf("listen error: %v", err)})
		return err
	}
	defer ln.Close()

	s.printf("Starting session %s on %s\n", s.Session, addr)
	s.emitEvent(ServerEvent{Kind: "started", Time: time.Now(), Message: fmt.Sprintf("session %s started on %s", s.Session, addr)})

	if err := s.broadcastState(); err != nil {
		s.emitEvent(ServerEvent{Kind: "error", Time: time.Now(), Message: fmt.Sprintf("initial state error: %v", err)})
	}

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
		s.emitEvent(ServerEvent{Kind: "error", Time: time.Now(), Message: fmt.Sprintf("read error: %v", err)})
		_ = conn.Close()
		return
	}

	var join Join
	if err := json.Unmarshal(line, &join); err != nil {
		s.println("invalid join message:", err)
		s.emitEvent(ServerEvent{Kind: "error", Time: time.Now(), Message: fmt.Sprintf("invalid join message: %v", err)})
		_ = conn.Close()
		return
	}

	if join.Type != "join" {
		s.println("unexpected message type:", join.Type)
		s.emitEvent(ServerEvent{Kind: "error", Time: time.Now(), Message: fmt.Sprintf("unexpected message type: %s", join.Type)})
		_ = conn.Close()
		return
	}

	if join.Session != s.Session {
		_ = writeJSONLine(conn, ErrorMessage{Type: "error", Session: s.Session, Code: "invalid_session", Message: "session code does not match"})
		s.emitEvent(ServerEvent{Kind: "rejected", Time: time.Now(), Message: fmt.Sprintf("rejected client %q: invalid session", join.Name)})
		_ = conn.Close()
		return
	}

	client := s.registerClient(join.Name, conn)
	s.printf("Client joined: %s\n", client.Name)
	s.emitEvent(ServerEvent{Kind: "client_joined", Time: time.Now(), Message: fmt.Sprintf("client joined: %s", client.Name)})

	if err := s.sendWelcome(client); err != nil {
		s.println("welcome error:", err)
		s.emitEvent(ServerEvent{Kind: "error", Time: time.Now(), Message: fmt.Sprintf("welcome error: %v", err)})
		return
	}

	if err := s.broadcastState(); err != nil {
		s.println("broadcast state error:", err)
		s.emitEvent(ServerEvent{Kind: "error", Time: time.Now(), Message: fmt.Sprintf("broadcast state error: %v", err)})
		return
	}

	for {
		line, err := reader.ReadBytes('\n')
		if err != nil {
			s.printf("Client disconnected: %s\n", client.Name)
			s.emitEvent(ServerEvent{Kind: "client_disconnected", Time: time.Now(), Message: fmt.Sprintf("client disconnected: %s", client.Name)})
			s.removeClient(client.ID)
			_ = conn.Close()
			return
		}
		s.handleClientMessage(client, line)
	}
}

func (s *Server) handleClientMessage(client *Client, line []byte) {
	var msg Message
	if err := json.Unmarshal(line, &msg); err != nil {
		s.emitEvent(ServerEvent{Kind: "error", Time: time.Now(), Message: fmt.Sprintf("invalid client message: %v", err)})
		return
	}

	switch msg.Type {
	case "vote":
		var vote VoteMessage
		if err := json.Unmarshal(line, &vote); err != nil {
			s.emitEvent(ServerEvent{Kind: "error", Time: time.Now(), Message: fmt.Sprintf("invalid vote message: %v", err)})
			return
		}
		s.recordVote(client.ID, vote.Vote, vote.Round)
	default:
		s.emitEvent(ServerEvent{Kind: "unknown", Time: time.Now(), Message: fmt.Sprintf("unknown client message type: %s", msg.Type)})
	}
}

func (s *Server) recordVote(clientID string, value string, roundID int) {
	value = strings.TrimSpace(value)
	if value == "" {
		return
	}

	s.mu.Lock()
	if s.currentIdx < 0 || s.currentIdx >= len(s.rounds) {
		s.mu.Unlock()
		return
	}
	round := &s.rounds[s.currentIdx]
	if roundID != 0 && roundID != round.ID {
		s.mu.Unlock()
		return
	}
	if round.Votes == nil {
		round.Votes = make(map[string]string)
	}
	round.Votes[clientID] = value
	round.UpdatedAt = time.Now()
	s.mu.Unlock()

	s.emitEvent(ServerEvent{Kind: "vote", Time: time.Now(), Message: fmt.Sprintf("vote received from %s", clientID)})
	_ = s.broadcastState()
}

func (s *Server) StartNextStory(title string, url string) {
	title = strings.TrimSpace(title)
	if title == "" {
		title = fmt.Sprintf("Story %d", s.nextRound)
	}

	s.mu.Lock()
	newRound := roundState{
		ID:        s.nextRound,
		Title:     title,
		URL:       strings.TrimSpace(url),
		Phase:     "voting",
		Reveal:    false,
		Votes:     make(map[string]string),
		UpdatedAt: time.Now(),
	}
	s.nextRound++
	s.rounds = append(s.rounds, newRound)
	s.currentIdx = len(s.rounds) - 1
	s.mu.Unlock()

	s.emitEvent(ServerEvent{Kind: "story_changed", Time: time.Now(), Message: fmt.Sprintf("active story changed to #%d", newRound.ID)})
	_ = s.broadcastState()
}

func (s *Server) ReopenStory(roundID int) {
	s.mu.Lock()
	idx := -1
	for i := range s.rounds {
		if s.rounds[i].ID == roundID {
			idx = i
			break
		}
	}
	if idx == -1 {
		s.mu.Unlock()
		return
	}
	s.currentIdx = idx
	s.rounds[idx].Phase = "voting"
	s.rounds[idx].Reveal = false
	s.rounds[idx].FinalEstimate = ""
	s.rounds[idx].Votes = make(map[string]string)
	s.rounds[idx].UpdatedAt = time.Now()
	s.mu.Unlock()

	s.emitEvent(ServerEvent{Kind: "story_reopened", Time: time.Now(), Message: fmt.Sprintf("reopened story #%d", roundID)})
	_ = s.broadcastState()
}

func (s *Server) RevealVotes() {
	s.mu.Lock()
	if s.currentIdx < 0 || s.currentIdx >= len(s.rounds) {
		s.mu.Unlock()
		return
	}
	round := &s.rounds[s.currentIdx]
	round.Reveal = true
	round.Phase = "revealed"
	round.FinalEstimate = chooseFinalEstimate(round.Votes)
	round.UpdatedAt = time.Now()
	s.mu.Unlock()

	s.emitEvent(ServerEvent{Kind: "reveal", Time: time.Now(), Message: "votes revealed"})
	_ = s.broadcastState()
}

func (s *Server) ClearVotes() {
	s.mu.Lock()
	if s.currentIdx < 0 || s.currentIdx >= len(s.rounds) {
		s.mu.Unlock()
		return
	}
	round := &s.rounds[s.currentIdx]
	round.Votes = make(map[string]string)
	round.Reveal = false
	round.Phase = "voting"
	round.FinalEstimate = ""
	round.UpdatedAt = time.Now()
	s.mu.Unlock()

	s.emitEvent(ServerEvent{Kind: "clear", Time: time.Now(), Message: "votes cleared"})
	_ = s.broadcastState()
}

func (s *Server) SetFinalEstimate(value string) {
	value = strings.TrimSpace(value)

	s.mu.Lock()
	if s.currentIdx < 0 || s.currentIdx >= len(s.rounds) {
		s.mu.Unlock()
		return
	}
	round := &s.rounds[s.currentIdx]
	round.FinalEstimate = value
	if value != "" {
		round.Reveal = true
		round.Phase = "revealed"
	}
	round.UpdatedAt = time.Now()
	s.mu.Unlock()

	s.emitEvent(ServerEvent{Kind: "final_updated", Time: time.Now(), Message: fmt.Sprintf("final estimate set to %q", value)})
	_ = s.broadcastState()
}

func (s *Server) DeleteStory(roundID int) bool {
	s.mu.Lock()
	idx := -1
	for i := range s.rounds {
		if s.rounds[i].ID == roundID {
			idx = i
			break
		}
	}
	if idx == -1 {
		s.mu.Unlock()
		return false
	}

	deletedID := s.rounds[idx].ID
	s.rounds = append(s.rounds[:idx], s.rounds[idx+1:]...)

	if len(s.rounds) == 0 {
		newRound := roundState{
			ID:        s.nextRound,
			Title:     "Untitled story",
			Phase:     "voting",
			Reveal:    false,
			Votes:     make(map[string]string),
			UpdatedAt: time.Now(),
		}
		s.nextRound++
		s.rounds = append(s.rounds, newRound)
		s.currentIdx = 0
	} else {
		if s.currentIdx == idx {
			if idx >= len(s.rounds) {
				s.currentIdx = len(s.rounds) - 1
			} else {
				s.currentIdx = idx
			}
		} else if idx < s.currentIdx {
			s.currentIdx--
		}
	}
	s.mu.Unlock()

	s.emitEvent(ServerEvent{Kind: "story_deleted", Time: time.Now(), Message: fmt.Sprintf("deleted story #%d", deletedID)})
	_ = s.broadcastState()
	return true
}

func (s *Server) CurrentRoundID() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.currentIdx < 0 || s.currentIdx >= len(s.rounds) {
		return 0
	}
	return s.rounds[s.currentIdx].ID
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

	client := &Client{ID: id, Name: name, Conn: conn}
	s.Clients[id] = client
	return client
}

func (s *Server) removeClient(id string) {
	s.mu.Lock()
	delete(s.Clients, id)
	if s.currentIdx >= 0 && s.currentIdx < len(s.rounds) {
		delete(s.rounds[s.currentIdx].Votes, id)
		s.rounds[s.currentIdx].UpdatedAt = time.Now()
	}
	s.mu.Unlock()
	_ = s.broadcastState()
}

func (s *Server) sendWelcome(client *Client) error {
	msg := Welcome{Type: "welcome", Session: s.Session, ClientID: client.ID, Role: "client"}
	return writeJSONLine(client.Conn, msg)
}

func (s *Server) broadcastState() error {
	s.mu.Lock()
	clients := make([]ClientInfo, 0, len(s.Clients))
	conns := make([]stdnet.Conn, 0, len(s.Clients))

	var current roundState
	if s.currentIdx >= 0 && s.currentIdx < len(s.rounds) {
		current = s.rounds[s.currentIdx]
	}

	for _, c := range s.Clients {
		info := ClientInfo{ID: c.ID, Name: c.Name, Voted: false}
		if v, ok := current.Votes[c.ID]; ok {
			info.Voted = true
			if current.Reveal {
				info.Vote = v
			}
		}
		clients = append(clients, info)
		conns = append(conns, c.Conn)
	}
	sort.Slice(clients, func(i, j int) bool { return clients[i].Name < clients[j].Name })

	history := make([]HistoryEntry, 0, len(s.rounds))
	for _, r := range s.rounds {
		history = append(history, HistoryEntry{
			ID:            r.ID,
			Title:         r.Title,
			URL:           r.URL,
			Phase:         r.Phase,
			FinalEstimate: r.FinalEstimate,
			UpdatedAt:     r.UpdatedAt.Format(time.RFC3339),
		})
	}

	state := State{
		Type:       "state",
		Session:    s.Session,
		Phase:      current.Phase,
		StoryTitle: current.Title,
		StoryURL:   current.URL,
		Clients:    clients,
		CurrentRound: CurrentRound{
			ID:            current.ID,
			Title:         current.Title,
			URL:           current.URL,
			Reveal:        current.Reveal,
			FinalEstimate: current.FinalEstimate,
		},
		History: history,
	}
	s.mu.Unlock()

	s.emitEvent(ServerEvent{Kind: "state", Time: time.Now(), Message: fmt.Sprintf("state broadcast: %d client(s)", len(clients)), State: state})

	for _, conn := range conns {
		if err := writeJSONLine(conn, state); err != nil {
			return err
		}
	}
	return nil
}

func chooseFinalEstimate(votes map[string]string) string {
	if len(votes) == 0 {
		return ""
	}
	count := map[string]int{}
	best := ""
	bestCount := 0
	for _, v := range votes {
		count[v]++
		if count[v] > bestCount {
			best = v
			bestCount = count[v]
		}
	}
	if best == "" {
		return ""
	}
	return best
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
