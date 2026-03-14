package poker

import (
	"errors"
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	cleenet "github.com/realvorl/procli/net"
)

var (
	titleStyle  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("39"))
	panelStyle  = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Padding(0, 1)
	labelStyle  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("214"))
	mutedStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	errorStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Bold(true)
	clientStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("118"))
)

type hostEventMsg cleenet.ServerEvent
type hostErrMsg struct{ err error }

type hostModel struct {
	events  <-chan cleenet.ServerEvent
	errs    <-chan error
	session string
	story   string
	url     string
	phase   string
	clients []cleenet.ClientInfo
	logs    []string
	lastErr error
}

func RunHost(server *cleenet.Server) error {
	events := make(chan cleenet.ServerEvent, 64)
	errs := make(chan error, 1)

	server.SetEventHandler(func(event cleenet.ServerEvent) {
		select {
		case events <- event:
		default:
		}
	})

	go func() {
		errs <- server.Listen()
	}()

	p := tea.NewProgram(hostModel{
		events:  events,
		errs:    errs,
		session: server.Session,
		story:   server.StoryTitle,
		url:     server.StoryURL,
		phase:   "lobby",
	}, tea.WithAltScreen())

	_, err := p.Run()
	return err
}

func (m hostModel) Init() tea.Cmd {
	return tea.Batch(waitHostEvent(m.events), waitHostErr(m.errs))
}

func (m hostModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		}
	case hostEventMsg:
		ev := cleenet.ServerEvent(msg)
		if ev.State.Type == "state" {
			m.phase = ev.State.Phase
			m.story = ev.State.StoryTitle
			m.url = ev.State.StoryURL
			m.clients = ev.State.Clients
			if ev.State.Session != "" {
				m.session = ev.State.Session
			}
		}
		if ev.Message != "" {
			m.logs = appendLog(m.logs, ev.Message)
		}
		return m, waitHostEvent(m.events)
	case hostErrMsg:
		m.lastErr = msg.err
		return m, nil
	}

	return m, nil
}

func (m hostModel) View() string {
	header := titleStyle.Render("proCLI Scrum Poker Host")
	meta := fmt.Sprintf("%s %s\n%s %s\n%s %s",
		labelStyle.Render("Session:"), m.session,
		labelStyle.Render("Phase:"), m.phase,
		labelStyle.Render("Story:"), nonEmpty(m.story, "-"),
	)
	if m.url != "" {
		meta += "\n" + labelStyle.Render("URL:") + " " + m.url
	}

	var clientRows []string
	if len(m.clients) == 0 {
		clientRows = []string{mutedStyle.Render("No clients connected yet")}
	} else {
		clientRows = make([]string, 0, len(m.clients))
		for _, c := range m.clients {
			clientRows = append(clientRows, clientStyle.Render("• "+c.Name))
		}
	}

	logText := mutedStyle.Render("No events yet")
	if len(m.logs) > 0 {
		logText = strings.Join(m.logs, "\n")
	}

	body := lipgloss.JoinVertical(
		lipgloss.Left,
		panelStyle.Render(meta),
		panelStyle.Render(labelStyle.Render("Participants")+"\n"+strings.Join(clientRows, "\n")),
		panelStyle.Render(labelStyle.Render("Events")+"\n"+logText),
		mutedStyle.Render("Press q to quit"),
	)

	if m.lastErr != nil {
		body += "\n" + errorStyle.Render("Server stopped: "+m.lastErr.Error())
	}

	return header + "\n\n" + body + "\n"
}

type clientEventMsg cleenet.ClientEvent
type clientDoneMsg struct{ err error }

type clientModel struct {
	events   <-chan cleenet.ClientEvent
	done     <-chan error
	address  string
	session  string
	name     string
	clientID string
	phase    string
	story    string
	url      string
	clients  []cleenet.ClientInfo
	logs     []string
	lastErr  error
	status   string
}

func RunClient(address string, session string, name string) error {
	events := make(chan cleenet.ClientEvent, 64)
	done := make(chan error, 1)

	go func() {
		done <- cleenet.JoinServerWithHandler(address, session, name, func(event cleenet.ClientEvent) {
			select {
			case events <- event:
			default:
			}
		})
	}()

	p := tea.NewProgram(clientModel{
		events:  events,
		done:    done,
		address: address,
		session: session,
		name:    name,
		status:  "connecting",
	}, tea.WithAltScreen())

	_, err := p.Run()
	return err
}

func (m clientModel) Init() tea.Cmd {
	return tea.Batch(waitClientEvent(m.events), waitClientDone(m.done))
}

func (m clientModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		}
	case clientEventMsg:
		ev := cleenet.ClientEvent(msg)
		switch ev.Kind {
		case "connected":
			m.status = "connected"
		case "welcome":
			m.clientID = ev.Welcome.ClientID
			m.status = "joined"
			if ev.Welcome.Session != "" {
				m.session = ev.Welcome.Session
			}
		case "state":
			m.phase = ev.State.Phase
			m.story = ev.State.StoryTitle
			m.url = ev.State.StoryURL
			m.clients = ev.State.Clients
			if ev.State.Session != "" {
				m.session = ev.State.Session
			}
		case "error":
			m.status = "error"
			if ev.Error.Message != "" {
				m.lastErr = fmt.Errorf("%s (%s)", ev.Error.Message, ev.Error.Code)
			} else {
				m.lastErr = errors.New(ev.Message)
			}
		case "disconnected":
			m.status = "disconnected"
		}
		if ev.Message != "" {
			m.logs = appendLog(m.logs, ev.Message)
		}
		return m, waitClientEvent(m.events)
	case clientDoneMsg:
		if msg.err != nil {
			m.lastErr = msg.err
			m.status = "error"
		} else if m.status != "error" {
			m.status = "disconnected"
		}
		return m, nil
	}

	return m, nil
}

func (m clientModel) View() string {
	header := titleStyle.Render("proCLI Scrum Poker Client")
	meta := fmt.Sprintf("%s %s\n%s %s\n%s %s\n%s %s\n%s %s",
		labelStyle.Render("Address:"), m.address,
		labelStyle.Render("Session:"), nonEmpty(m.session, "-"),
		labelStyle.Render("Name:"), nonEmpty(m.name, "-"),
		labelStyle.Render("ClientID:"), nonEmpty(m.clientID, "-"),
		labelStyle.Render("Status:"), m.status,
	)

	if m.story != "" || m.url != "" || m.phase != "" {
		meta += fmt.Sprintf("\n%s %s", labelStyle.Render("Phase:"), nonEmpty(m.phase, "-"))
	}

	var storyRows []string
	if m.story != "" {
		storyRows = append(storyRows, labelStyle.Render("Story:")+" "+m.story)
	}
	if m.url != "" {
		storyRows = append(storyRows, labelStyle.Render("URL:")+" "+m.url)
	}
	if len(storyRows) == 0 {
		storyRows = []string{mutedStyle.Render("No story data yet")}
	}

	var clientRows []string
	if len(m.clients) == 0 {
		clientRows = []string{mutedStyle.Render("No clients in state")}
	} else {
		clientRows = make([]string, 0, len(m.clients))
		for _, c := range m.clients {
			clientRows = append(clientRows, clientStyle.Render("• "+c.Name))
		}
	}

	logText := mutedStyle.Render("No events yet")
	if len(m.logs) > 0 {
		logText = strings.Join(m.logs, "\n")
	}

	body := lipgloss.JoinVertical(
		lipgloss.Left,
		panelStyle.Render(meta),
		panelStyle.Render(strings.Join(storyRows, "\n")),
		panelStyle.Render(labelStyle.Render("Participants")+"\n"+strings.Join(clientRows, "\n")),
		panelStyle.Render(labelStyle.Render("Events")+"\n"+logText),
		mutedStyle.Render("Press q to quit"),
	)

	if m.lastErr != nil {
		body += "\n" + errorStyle.Render("Connection error: "+m.lastErr.Error())
	}

	return header + "\n\n" + body + "\n"
}

func waitHostEvent(events <-chan cleenet.ServerEvent) tea.Cmd {
	return func() tea.Msg {
		return hostEventMsg(<-events)
	}
}

func waitHostErr(errs <-chan error) tea.Cmd {
	return func() tea.Msg {
		return hostErrMsg{err: <-errs}
	}
}

func waitClientEvent(events <-chan cleenet.ClientEvent) tea.Cmd {
	return func() tea.Msg {
		return clientEventMsg(<-events)
	}
}

func waitClientDone(done <-chan error) tea.Cmd {
	return func() tea.Msg {
		return clientDoneMsg{err: <-done}
	}
}

func appendLog(logs []string, line string) []string {
	const maxLogs = 12
	logs = append(logs, line)
	if len(logs) > maxLogs {
		return logs[len(logs)-maxLogs:]
	}
	return logs
}

func nonEmpty(v string, fallback string) string {
	if strings.TrimSpace(v) == "" {
		return fallback
	}
	return v
}
