package main

import (
	"bufio"
	"errors"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type participant struct {
	Name    string
	Enabled bool
}

type screen int

const (
	screenList screen = iota
	screenEdit
	screenWinner
)

type editMode int

const (
	editAdd editMode = iota
	editRename
)

type model struct {
	// data
	filePath     string
	participants []participant
	cursor       int

	// ui state
	screen   screen
	editMode editMode
	input    textinput.Model
	err      error

	// draw state
	drawing     bool
	drawIndex   int
	drawSteps   int
	drawStepsTo int
	winnerName  string
	targetIndex int // the chosen winner index (in participants slice)
	cyclesLeft  int // how many full wrap-arounds to do before we’re allowed to stop
}

type tickMsg struct{}

var (
	titleStyle  = lipgloss.NewStyle().Bold(true)
	helpStyle   = lipgloss.NewStyle().Faint(true)
	errorStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Bold(true)
	winnerStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("10"))
)

func main() {
	fp := defaultParticipantsPath()

	m := newModel(fp)
	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Println("fatal:", err)
		os.Exit(1)
	}
}

func newModel(filePath string) model {
	items, err := loadParticipants(filePath)
	if err != nil {
		// start empty; show error in UI
		items = []participant{}
	}

	ti := textinput.New()
	ti.Placeholder = "Name"
	ti.CharLimit = 60
	ti.Width = 30

	return model{
		filePath:     filePath,
		participants: items,
		cursor:       0,
		screen:       screenList,
		input:        ti,
		err:          err,
	}
}

func (m model) Init() tea.Cmd { return nil }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch m.screen {
	case screenList:
		return m.updateList(msg)
	case screenEdit:
		return m.updateEdit(msg)
	case screenWinner:
		return m.updateWinner(msg)
	default:
		return m, nil
	}
}

func (m model) View() string {
	switch m.screen {
	case screenList:
		return m.viewList()
	case screenEdit:
		return m.viewEdit()
	case screenWinner:
		return m.viewWinner()
	default:
		return "unknown screen"
	}
}

/* -------------------- LIST SCREEN -------------------- */

func (m model) updateList(msg tea.Msg) (tea.Model, tea.Cmd) {
	// drawing tick
	if m.drawing {
		switch msg.(type) {
		case tickMsg:
			return m.stepDraw()
		}
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit

		case "up", "k":
			if len(m.participants) > 0 && m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if len(m.participants) > 0 && m.cursor < len(m.participants)-1 {
				m.cursor++
			}

		case " ":
			// toggle enabled
			if len(m.participants) > 0 && m.cursor >= 0 && m.cursor < len(m.participants) {
				m.participants[m.cursor].Enabled = !m.participants[m.cursor].Enabled
				m.err = saveParticipants(m.filePath, m.participants)
			}

		case "a":
			// add
			m.screen = screenEdit
			m.editMode = editAdd
			m.input.SetValue("")
			m.input.Focus()
			return m, nil

		case "e":
			// edit/rename
			if len(m.participants) == 0 {
				break
			}
			m.screen = screenEdit
			m.editMode = editRename
			m.input.SetValue(m.participants[m.cursor].Name)
			m.input.CursorEnd()
			m.input.Focus()
			return m, nil

		case "d", "backspace":
			// delete
			if len(m.participants) == 0 {
				break
			}
			m.participants = append(m.participants[:m.cursor], m.participants[m.cursor+1:]...)
			if m.cursor >= len(m.participants) && m.cursor > 0 {
				m.cursor--
			}
			m.err = saveParticipants(m.filePath, m.participants)

		case "enter":

			// start draw (winner picked first, animation is just theater)
			enabledIdx := enabledIndices(m.participants)
			if len(enabledIdx) < 1 {
				m.err = errors.New("no enabled participants to draw from (toggle with Space)")
				break
			}

			rand.Seed(time.Now().UnixNano())

			// pick winner upfront from enabled participants
			m.targetIndex = enabledIdx[rand.Intn(len(enabledIdx))]

			// animation setup
			m.drawing = true
			m.drawSteps = 0
			m.drawStepsTo = 0               // no longer used for stopping (can remove later)
			m.cyclesLeft = 2 + rand.Intn(3) // 2..4 full loops for suspense

			// start from current cursor (or 0)
			m.drawIndex = clamp(m.cursor, 0, max(0, len(m.participants)-1))

			// clear previous winner
			m.winnerName = ""

			return m, nextTick()
		}
	}
	return m, nil
}

func (m model) stepDraw() (tea.Model, tea.Cmd) {
	if len(m.participants) == 0 {
		m.drawing = false
		return m, nil
	}

	enabledIdx := enabledIndices(m.participants)
	if len(enabledIdx) == 0 {
		m.drawing = false
		return m, nil
	}

	// advance to next enabled participant
	prev := m.drawIndex
	m.drawIndex = nextEnabledIndex(m.participants, m.drawIndex)

	// if we wrapped around, count one completed cycle
	if m.drawIndex < prev {
		if m.cyclesLeft > 0 {
			m.cyclesLeft--
		}
	}

	// show animation by moving cursor highlight
	m.cursor = m.drawIndex

	// stop condition: only after cycles are done AND we landed on target
	if m.cyclesLeft == 0 && m.drawIndex == m.targetIndex {
		m.drawing = false
		m.winnerName = m.participants[m.drawIndex].Name
		m.screen = screenWinner
		return m, nil
	}

	// slow down near the end (once we're allowed to stop)
	delay := 25 * time.Millisecond
	if m.cyclesLeft == 0 {
		// as we get close to target, slow down progressively
		// simple heuristic: if within ~5 hops, slow down more
		// (we don't compute exact distance; this is “good enough” theater)
		delay = 80 * time.Millisecond
		if m.drawIndex == m.targetIndex {
			delay = 160 * time.Millisecond
		}
	}

	return m, tea.Tick(delay, func(time.Time) tea.Msg { return tickMsg{} })
}

func (m model) viewList() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("clee — random name chooser"))
	b.WriteString("\n")
	b.WriteString(helpStyle.Render(fmt.Sprintf("file: %s", m.filePath)))
	b.WriteString("\n\n")

	if m.err != nil {
		b.WriteString(errorStyle.Render("⚠ " + m.err.Error()))
		b.WriteString("\n\n")
	}

	if len(m.participants) == 0 {
		b.WriteString("No participants yet.\n")
		b.WriteString("Press 'a' to add.\n\n")
	} else {
		for i, p := range m.participants {
			cursor := " "
			if i == m.cursor {
				cursor = "➤"
			}
			box := "[ ]"
			if p.Enabled {
				box = "[x]"
			}
			line := fmt.Sprintf("%s %s %s", cursor, box, p.Name)
			b.WriteString(line)
			b.WriteString("\n")
		}
		b.WriteString("\n")
	}

	if m.drawing {
		b.WriteString(helpStyle.Render("Selecting…"))
		b.WriteString("\n")
	}

	b.WriteString(helpStyle.Render("Keys: ↑/↓ move | Space toggle | a add | e edit | d delete | Enter start | q quit"))
	b.WriteString("\n")

	return b.String()
}

/* -------------------- EDIT SCREEN -------------------- */

func (m model) updateEdit(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			m.screen = screenList
			m.input.Blur()
			return m, nil

		case "enter":
			name := strings.TrimSpace(m.input.Value())
			if name == "" {
				m.err = errors.New("name cannot be empty")
				return m, nil
			}

			switch m.editMode {
			case editAdd:
				m.participants = append(m.participants, participant{Name: name, Enabled: true})
				m.cursor = len(m.participants) - 1
			case editRename:
				if len(m.participants) > 0 && m.cursor >= 0 && m.cursor < len(m.participants) {
					m.participants[m.cursor].Name = name
				}
			}

			m.err = saveParticipants(m.filePath, m.participants)
			m.screen = screenList
			m.input.Blur()
			return m, nil
		}
	}

	return m, cmd
}

func (m model) viewEdit() string {
	var b strings.Builder
	title := "Add participant"
	if m.editMode == editRename {
		title = "Edit participant"
	}
	b.WriteString(titleStyle.Render(title))
	b.WriteString("\n\n")
	b.WriteString("Name:\n")
	b.WriteString(m.input.View())
	b.WriteString("\n\n")
	b.WriteString(helpStyle.Render("Enter save | Esc cancel"))
	b.WriteString("\n")
	return b.String()
}

/* -------------------- WINNER SCREEN -------------------- */

func (m model) updateWinner(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter", "esc", "q":
			// back to list; keep cursor on winner
			m.screen = screenList
			m.err = nil
			return m, nil
		case "ctrl+c":
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m model) viewWinner() string {
	var b strings.Builder
	b.WriteString("\n")
	b.WriteString(winnerStyle.Render("winner 🎉 " + m.winnerName))
	b.WriteString("\n\n")
	b.WriteString(helpStyle.Render("Press Enter to go back"))
	b.WriteString("\n")
	return b.String()
}

/* -------------------- STORAGE -------------------- */

// Format (simple + editable):
// each line: "<enabled>\t<name>"
// enabled: 1 or 0
func loadParticipants(path string) ([]participant, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		// if missing file, treat as empty (not an error)
		if os.IsNotExist(err) {
			return []participant{}, nil
		}
		return nil, err
	}
	lines := strings.Split(string(data), "\n")
	out := make([]participant, 0, len(lines))
	for _, ln := range lines {
		ln = strings.TrimSpace(ln)
		if ln == "" || strings.HasPrefix(ln, "#") {
			continue
		}
		parts := strings.SplitN(ln, "\t", 2)
		if len(parts) != 2 {
			continue
		}
		en := strings.TrimSpace(parts[0])
		name := strings.TrimSpace(parts[1])
		if name == "" {
			continue
		}
		enabled := en == "1" || strings.EqualFold(en, "true") || en == "x"
		out = append(out, participant{Name: name, Enabled: enabled})
	}
	return out, nil
}

func saveParticipants(path string, ps []participant) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	tmp := path + ".tmp"

	f, err := os.Create(tmp)
	if err != nil {
		return err
	}
	defer f.Close()

	w := bufio.NewWriter(f)
	_, _ = w.WriteString("# clee participants: <enabled>\\t<name>\n")
	for _, p := range ps {
		en := "0"
		if p.Enabled {
			en = "1"
		}
		_, _ = w.WriteString(en + "\t" + p.Name + "\n")
	}
	if err := w.Flush(); err != nil {
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func defaultParticipantsPath() string {
	// Cross-platform enough for now:
	// Linux/macOS: ~/.config/clee/participants.tsv
	// Windows: still works if HOME is set; we’ll refine once you tell me platform.
	home := os.Getenv("HOME")
	if home == "" {
		home = "."
	}
	return filepath.Join(home, ".config", "clee", "participants.tsv")
}

/* -------------------- HELPERS -------------------- */

func enabledIndices(ps []participant) []int {
	out := make([]int, 0, len(ps))
	for i, p := range ps {
		if p.Enabled {
			out = append(out, i)
		}
	}
	return out
}

func nextEnabledIndex(ps []participant, start int) int {
	if len(ps) == 0 {
		return 0
	}
	// walk forward circularly until we hit enabled
	for i := 1; i <= len(ps); i++ {
		idx := (start + i) % len(ps)
		if ps[idx].Enabled {
			return idx
		}
	}
	// none enabled (should be checked by caller)
	return start
}

func nextTick() tea.Cmd {
	return tea.Tick(40*time.Millisecond, func(time.Time) tea.Msg { return tickMsg{} })
}

func clamp(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// unused but handy later (kept tiny)
func atoi(s string) int {
	n, _ := strconv.Atoi(s)
	return n
}
