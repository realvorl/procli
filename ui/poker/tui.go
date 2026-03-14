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
	titleStyle    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("39"))
	panelStyle    = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Padding(0, 1)
	labelStyle    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("214"))
	mutedStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	errorStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Bold(true)
	clientStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("118"))
	selectedStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("229"))
	revealedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("51"))
	inputStyle    = lipgloss.NewStyle().Border(lipgloss.NormalBorder()).Padding(0, 1)
)

type hostEventMsg cleenet.ServerEvent
type hostErrMsg struct{ err error }

type hostModel struct {
	server          *cleenet.Server
	events          <-chan cleenet.ServerEvent
	errs            <-chan error
	session         string
	phase           string
	currentRoundID  int
	story           string
	url             string
	finalEstimate   string
	clients         []cleenet.ClientInfo
	history         []cleenet.HistoryEntry
	selectedHistory int
	logs            []string
	lastErr         error
	editMode        bool
	editTitle       string
	editURL         string
	editFieldIsURL  bool
	finalEditMode   bool
	finalInput      string
	deleteConfirm   bool
	deleteTargetID  int
	deleteTarget    string
}

func RunHost(server *cleenet.Server) error {
	events := make(chan cleenet.ServerEvent, 128)
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
		server:    server,
		events:    events,
		errs:      errs,
		session:   server.Session,
		phase:     "voting",
		editTitle: server.StoryTitle,
		editURL:   server.StoryURL,
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
		if m.editMode {
			return m.updateEditMode(msg)
		}
		if m.finalEditMode {
			return m.updateFinalEditMode(msg)
		}
		if m.deleteConfirm {
			return m.updateDeleteConfirmMode(msg)
		}
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "r":
			m.server.RevealVotes()
		case "c":
			m.server.ClearVotes()
		case "n":
			m.server.StartNextStory("", "")
		case "e":
			m.editMode = true
			if strings.TrimSpace(m.story) != "" {
				m.editTitle = m.story
			}
			m.editURL = m.url
		case "f":
			m.finalEditMode = true
			m.finalInput = m.finalEstimate
		case "d":
			if len(m.history) > 0 && m.selectedHistory >= 0 && m.selectedHistory < len(m.history) {
				m.deleteConfirm = true
				m.deleteTargetID = m.history[m.selectedHistory].ID
				m.deleteTarget = m.history[m.selectedHistory].Title
			}
		case "up", "k":
			if m.selectedHistory > 0 {
				m.selectedHistory--
			}
		case "down", "j":
			if m.selectedHistory < len(m.history)-1 {
				m.selectedHistory++
			}
		case "enter":
			if len(m.history) > 0 && m.selectedHistory >= 0 && m.selectedHistory < len(m.history) {
				m.server.ReopenStory(m.history[m.selectedHistory].ID)
			}
		}
	case hostEventMsg:
		ev := cleenet.ServerEvent(msg)
		if ev.State.Type == "state" {
			m.phase = ev.State.Phase
			m.story = ev.State.CurrentRound.Title
			m.url = ev.State.CurrentRound.URL
			m.finalEstimate = ev.State.CurrentRound.FinalEstimate
			m.clients = ev.State.Clients
			m.history = ev.State.History
			m.currentRoundID = ev.State.CurrentRound.ID
			if ev.State.Session != "" {
				m.session = ev.State.Session
			}
			if len(m.history) > 0 {
				active := 0
				for i := range m.history {
					if m.history[i].ID == m.currentRoundID {
						active = i
						break
					}
				}
				if m.selectedHistory >= len(m.history) {
					m.selectedHistory = len(m.history) - 1
				}
				if m.selectedHistory < 0 {
					m.selectedHistory = active
				}
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

func (m hostModel) updateDeleteConfirmMode(key tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch key.String() {
	case "y":
		m.server.DeleteStory(m.deleteTargetID)
		m.deleteConfirm = false
		return m, nil
	case "n", "esc":
		m.deleteConfirm = false
		return m, nil
	default:
		return m, nil
	}
}

func (m hostModel) updateFinalEditMode(key tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch key.String() {
	case "esc":
		m.finalEditMode = false
		return m, nil
	case "enter":
		m.server.SetFinalEstimate(m.finalInput)
		m.finalEditMode = false
		return m, nil
	case "backspace":
		if len(m.finalInput) > 0 {
			m.finalInput = m.finalInput[:len(m.finalInput)-1]
		}
		return m, nil
	default:
		if len(key.Runes) == 0 {
			return m, nil
		}
		m.finalInput += string(key.Runes)
		return m, nil
	}
}

func (m hostModel) updateEditMode(key tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch key.String() {
	case "esc":
		m.editMode = false
		m.editFieldIsURL = false
		return m, nil
	case "tab":
		m.editFieldIsURL = !m.editFieldIsURL
		return m, nil
	case "enter":
		m.server.StartNextStory(m.editTitle, m.editURL)
		m.editMode = false
		m.editFieldIsURL = false
		return m, nil
	case "backspace":
		if m.editFieldIsURL {
			if len(m.editURL) > 0 {
				m.editURL = m.editURL[:len(m.editURL)-1]
			}
		} else {
			if len(m.editTitle) > 0 {
				m.editTitle = m.editTitle[:len(m.editTitle)-1]
			}
		}
		return m, nil
	default:
		if len(key.Runes) == 0 {
			return m, nil
		}
		if m.editFieldIsURL {
			m.editURL += string(key.Runes)
		} else {
			m.editTitle += string(key.Runes)
		}
		return m, nil
	}
}

func (m hostModel) View() string {
	header := titleStyle.Render("proCLI Scrum Poker Host")
	meta := fmt.Sprintf("%s %s\n%s #%d\n%s %s\n%s %s",
		labelStyle.Render("Session:"), m.session,
		labelStyle.Render("Round:"), m.currentRoundID,
		labelStyle.Render("Phase:"), nonEmpty(m.phase, "-"),
		labelStyle.Render("Story:"), nonEmpty(m.story, "-"),
	)
	meta += "\n" + labelStyle.Render("Final:") + " " + nonEmpty(m.finalEstimate, "-")
	if m.url != "" {
		meta += "\n" + labelStyle.Render("URL:") + " " + m.url
	}

	var clientRows []string
	if len(m.clients) == 0 {
		clientRows = []string{mutedStyle.Render("No clients connected yet")}
	} else {
		for _, c := range m.clients {
			status := "waiting"
			if c.Voted {
				status = "voted"
			}
			row := fmt.Sprintf("• %s [%s]", c.Name, status)
			if c.Vote != "" {
				row += " = " + c.Vote
			}
			clientRows = append(clientRows, clientStyle.Render(row))
		}
	}

	logText := mutedStyle.Render("No events yet")
	if len(m.logs) > 0 {
		logText = strings.Join(m.logs, "\n")
	}

	left := lipgloss.JoinVertical(
		lipgloss.Left,
		panelStyle.Render(meta),
		panelStyle.Render(labelStyle.Render("Participants")+"\n"+strings.Join(clientRows, "\n")),
		panelStyle.Render(labelStyle.Render("Events")+"\n"+logText),
	)

	right := panelStyle.Render(renderHistory(m.history, m.selectedHistory, m.currentRoundID, true))
	body := lipgloss.JoinHorizontal(lipgloss.Top, left, "  ", right)

	help := mutedStyle.Render("Host keys: n=new auto story, e=edit+start story, f=edit final, d=delete story, r=reveal, c=clear, ↑/↓ select history, enter=reopen, q=quit")
	if m.editMode {
		title := m.editTitle
		url := m.editURL
		if !m.editFieldIsURL {
			title = selectedStyle.Render(title + "_")
		} else {
			url = selectedStyle.Render(url + "_")
		}
		help = panelStyle.Render(
			labelStyle.Render("New Story Editor") + "\n" +
				"Title: " + title + "\n" +
				"URL:   " + url + "\n" +
				mutedStyle.Render("Type text, tab switch field, enter apply, esc cancel"),
		)
	}
	if m.finalEditMode {
		help = panelStyle.Render(
			labelStyle.Render("Final Estimate Editor") + "\n" +
				"Final: " + selectedStyle.Render(m.finalInput+"_") + "\n" +
				mutedStyle.Render("Type value, enter apply, esc cancel"),
		)
	}
	if m.deleteConfirm {
		help = panelStyle.Render(
			errorStyle.Render("Delete Story?") + "\n" +
				fmt.Sprintf("Story #%d: %s\n", m.deleteTargetID, nonEmpty(m.deleteTarget, "-")) +
				mutedStyle.Render("Press y to confirm, n or esc to cancel"),
		)
	}

	view := header + "\n\n" + body + "\n\n" + help
	if m.lastErr != nil {
		view += "\n" + errorStyle.Render("Server stopped: "+m.lastErr.Error())
	}
	return view + "\n"
}

type clientEventMsg cleenet.ClientEvent
type clientDoneMsg struct{ err error }

type clientModel struct {
	events        <-chan cleenet.ClientEvent
	done          <-chan error
	outbound      chan cleenet.VoteMessage
	address       string
	session       string
	name          string
	clientID      string
	phase         string
	story         string
	url           string
	roundID       int
	revealed      bool
	finalEstimate string
	clients       []cleenet.ClientInfo
	history       []cleenet.HistoryEntry
	logs          []string
	lastErr       error
	status        string
	voteInput     string
	selectedVote  string
	lastSentRound int
}

func RunClient(address string, session string, name string) error {
	events := make(chan cleenet.ClientEvent, 128)
	done := make(chan error, 1)
	outbound := make(chan cleenet.VoteMessage, 16)

	go func() {
		done <- cleenet.JoinServerWithHandler(address, session, name, func(event cleenet.ClientEvent) {
			select {
			case events <- event:
			default:
			}
		}, outbound)
	}()

	p := tea.NewProgram(clientModel{
		events:   events,
		done:     done,
		outbound: outbound,
		address:  address,
		session:  session,
		name:     name,
		status:   "connecting",
	}, tea.WithAltScreen())

	_, err := p.Run()
	close(outbound)
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
		case "enter":
			value := strings.TrimSpace(m.voteInput)
			if value != "" {
				m.sendVote(value)
			}
		case "backspace":
			if len(m.voteInput) > 0 {
				m.voteInput = m.voteInput[:len(m.voteInput)-1]
			}
		default:
			if len(msg.Runes) > 0 {
				r := msg.Runes[0]
				if (r >= '0' && r <= '9') || r == '.' {
					m.voteInput += string(r)
				}
			}
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
			m.story = ev.State.CurrentRound.Title
			m.url = ev.State.CurrentRound.URL
			m.roundID = ev.State.CurrentRound.ID
			m.revealed = ev.State.CurrentRound.Reveal
			m.finalEstimate = ev.State.CurrentRound.FinalEstimate
			m.clients = ev.State.Clients
			m.history = ev.State.History
			if ev.State.Session != "" {
				m.session = ev.State.Session
			}
			if m.lastSentRound != m.roundID {
				m.selectedVote = ""
				m.voteInput = ""
			}
		case "vote_sent":
			m.status = "voted"
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

func (m *clientModel) sendVote(v string) {
	if m.roundID == 0 {
		return
	}
	m.selectedVote = v
	m.lastSentRound = m.roundID
	select {
	case m.outbound <- cleenet.VoteMessage{Vote: v, Round: m.roundID}:
	default:
	}
}

func (m clientModel) View() string {
	header := titleStyle.Render("proCLI Scrum Poker Client")
	meta := fmt.Sprintf("%s %s\n%s %s\n%s %s\n%s %s\n%s #%d\n%s %s",
		labelStyle.Render("Address:"), m.address,
		labelStyle.Render("Session:"), nonEmpty(m.session, "-"),
		labelStyle.Render("Name:"), nonEmpty(m.name, "-"),
		labelStyle.Render("ClientID:"), nonEmpty(m.clientID, "-"),
		labelStyle.Render("Round:"), m.roundID,
		labelStyle.Render("Status:"), m.status,
	)
	meta += "\n" + labelStyle.Render("Phase:") + " " + nonEmpty(m.phase, "-")
	meta += "\n" + labelStyle.Render("Story:") + " " + nonEmpty(m.story, "-")
	if m.url != "" {
		meta += "\n" + labelStyle.Render("URL:") + " " + m.url
	}

	var clientRows []string
	if len(m.clients) == 0 {
		clientRows = []string{mutedStyle.Render("No clients in state")}
	} else {
		for _, c := range m.clients {
			row := fmt.Sprintf("• %s", c.Name)
			if c.Voted {
				row += " [voted]"
			}
			if c.Vote != "" {
				row += " = " + c.Vote
			}
			clientRows = append(clientRows, clientStyle.Render(row))
		}
	}

	votePanel := labelStyle.Render("Vote")
	votePanel += "\n" + inputStyle.Render(m.voteInput+"_")
	votePanel += "\n" + mutedStyle.Render("Type a number, press enter to submit")
	if m.selectedVote != "" {
		votePanel += "\nYour vote: " + selectedStyle.Render(m.selectedVote)
	}
	if m.revealed {
		votePanel += "\n" + revealedStyle.Render("Revealed final: "+nonEmpty(m.finalEstimate, "-"))
	}

	left := lipgloss.JoinVertical(
		lipgloss.Left,
		panelStyle.Render(meta),
		panelStyle.Render(votePanel),
		panelStyle.Render(labelStyle.Render("Participants")+"\n"+strings.Join(clientRows, "\n")),
	)
	right := panelStyle.Render(renderHistory(m.history, -1, m.roundID, false))
	body := lipgloss.JoinHorizontal(lipgloss.Top, left, "  ", right)

	logText := mutedStyle.Render("No events yet")
	if len(m.logs) > 0 {
		logText = strings.Join(m.logs, "\n")
	}
	footer := panelStyle.Render(labelStyle.Render("Events") + "\n" + logText)
	footer += "\n" + mutedStyle.Render("Press q to quit")

	view := header + "\n\n" + body + "\n\n" + footer
	if m.lastErr != nil {
		view += "\n" + errorStyle.Render("Connection error: "+m.lastErr.Error())
	}
	return view + "\n"
}

func renderHistory(history []cleenet.HistoryEntry, selected int, activeRoundID int, host bool) string {
	rows := []string{labelStyle.Render("History"), mutedStyle.Render("#   Story                      Final   Phase")}
	if len(history) == 0 {
		rows = append(rows, mutedStyle.Render("No rounds yet"))
		return strings.Join(rows, "\n")
	}

	for i, h := range history {
		title := h.Title
		if len(title) > 24 {
			title = title[:21] + "..."
		}
		line := fmt.Sprintf("%-3d %-26s %-7s %s", h.ID, title, nonEmpty(h.FinalEstimate, "-"), h.Phase)
		if h.ID == activeRoundID {
			line = "* " + line
		} else {
			line = "  " + line
		}
		if host && i == selected {
			line = selectedStyle.Render(line)
		}
		rows = append(rows, line)
	}

	if host {
		rows = append(rows, "", mutedStyle.Render("Use ↑/↓ and enter to reopen"))
	}
	return strings.Join(rows, "\n")
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
