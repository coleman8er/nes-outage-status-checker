package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const apiURL = "https://utilisocial.io/datacapable/v2/p/NES/map/events"

type OutageEvent struct {
	ID              int     `json:"id"`
	StartTime       int64   `json:"startTime"`
	LastUpdatedTime int64   `json:"lastUpdatedTime"`
	Title           string  `json:"title"`
	NumPeople       int     `json:"numPeople"`
	Status          string  `json:"status"`
	Cause           string  `json:"cause"`
	Identifier      string  `json:"identifier"`
	Latitude        float64 `json:"latitude"`
	Longitude       float64 `json:"longitude"`
}

type model struct {
	eventID      int
	event        *OutageEvent
	spinner      spinner.Model
	loading      bool
	err          error
	lastChecked  time.Time
	blinkOn      bool
	statusBlink  bool
}

type tickMsg time.Time
type blinkMsg time.Time
type fetchResultMsg struct {
	event *OutageEvent
	err   error
}

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("229")).
			Background(lipgloss.Color("57")).
			Padding(0, 1)

	labelStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("252"))

	valueStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("252"))

	statusUnassigned = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("196"))

	statusAssigned = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("46"))

	statusAssignedDim = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("22"))

	boxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("62")).
			Padding(1, 2)

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("196"))

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241"))

	timeStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("245")).
			Italic(true)
)

func initialModel(eventID int) model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))
	return model{
		eventID:     eventID,
		spinner:     s,
		loading:     true,
		blinkOn:     true,
		statusBlink: false,
	}
}

func (m model) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		fetchEvent(m.eventID),
		tickCmd(),
		blinkCmd(),
	)
}

func tickCmd() tea.Cmd {
	return tea.Tick(30*time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func blinkCmd() tea.Cmd {
	return tea.Tick(500*time.Millisecond, func(t time.Time) tea.Msg {
		return blinkMsg(t)
	})
}

func fetchEvent(eventID int) tea.Cmd {
	return func() tea.Msg {
		client := &http.Client{Timeout: 10 * time.Second}
		resp, err := client.Get(apiURL)
		if err != nil {
			return fetchResultMsg{nil, fmt.Errorf("failed to fetch: %w", err)}
		}
		defer resp.Body.Close()

		var events []OutageEvent
		if err := json.NewDecoder(resp.Body).Decode(&events); err != nil {
			return fetchResultMsg{nil, fmt.Errorf("failed to parse JSON: %w", err)}
		}

		for _, e := range events {
			if e.ID == eventID {
				return fetchResultMsg{&e, nil}
			}
		}

		return fetchResultMsg{nil, fmt.Errorf("event ID %d not found", eventID)}
	}
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c", "esc":
			return m, tea.Quit
		case "r":
			m.loading = true
			return m, fetchEvent(m.eventID)
		}

	case tickMsg:
		m.loading = true
		return m, tea.Batch(fetchEvent(m.eventID), tickCmd())

	case blinkMsg:
		m.blinkOn = !m.blinkOn
		return m, blinkCmd()

	case fetchResultMsg:
		m.loading = false
		m.lastChecked = time.Now()
		if msg.err != nil {
			m.err = msg.err
			m.event = nil
		} else {
			m.err = nil
			m.event = msg.event
			m.statusBlink = (m.event.Status != "Unassigned")
		}
		return m, nil

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m model) View() string {
	var s string

	s += "\n"
	s += titleStyle.Render("NES Outage Status Checker") + "\n\n"

	if m.loading && m.event == nil {
		s += m.spinner.View() + " Fetching outage data...\n"
	} else if m.err != nil {
		s += errorStyle.Render("Error: "+m.err.Error()) + "\n"
	} else if m.event != nil {
		content := ""

		content += labelStyle.Render("Event ID: ") + valueStyle.Render(fmt.Sprintf("%d", m.event.ID)) + "\n"
		content += labelStyle.Render("Identifier: ") + valueStyle.Render(m.event.Identifier) + "\n"
		content += labelStyle.Render("Title: ") + valueStyle.Render(m.event.Title) + "\n"
		content += labelStyle.Render("Affected: ") + valueStyle.Render(fmt.Sprintf("%d people", m.event.NumPeople)) + "\n"

		if m.event.Cause != "" {
			content += labelStyle.Render("Cause: ") + valueStyle.Render(m.event.Cause) + "\n"
		}

		startTime := time.UnixMilli(m.event.StartTime)
		content += labelStyle.Render("Started: ") + valueStyle.Render(startTime.Format("Mon Jan 2, 3:04 PM")) + "\n"

		lastUpdated := time.UnixMilli(m.event.LastUpdatedTime)
		content += labelStyle.Render("Last Updated: ") + valueStyle.Render(lastUpdated.Format("Mon Jan 2, 3:04 PM")) + "\n"

		content += "\n"

		var statusDisplay string
		if m.event.Status == "Unassigned" {
			statusDisplay = statusUnassigned.Render("STATUS: UNASSIGNED")
			content += statusDisplay + "\n"
			content += valueStyle.Render("No technician assigned yet") + "\n"
		} else {
			if m.blinkOn {
				statusDisplay = statusAssigned.Render("STATUS: " + m.event.Status)
			} else {
				statusDisplay = statusAssignedDim.Render("STATUS: " + m.event.Status)
			}
			content += statusDisplay + "\n"
			content += valueStyle.Render("A technician has been assigned!") + "\n"
		}

		s += boxStyle.Render(content) + "\n"

		if m.loading {
			s += "\n" + m.spinner.View() + " Refreshing..."
		}
	}

	s += "\n"
	if !m.lastChecked.IsZero() {
		s += timeStyle.Render(fmt.Sprintf("Last checked: %s", m.lastChecked.Format("3:04:05 PM"))) + "\n"
	}
	s += helpStyle.Render("Press 'r' to refresh • 'q' to quit • Auto-refreshes every 30s") + "\n"

	return s
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: nes-outage-status-checker <event-id>")
		fmt.Println("Example: nes-outage-status-checker 1971637")
		os.Exit(1)
	}

	eventID, err := strconv.Atoi(os.Args[1])
	if err != nil {
		fmt.Printf("Invalid event ID: %s\n", os.Args[1])
		os.Exit(1)
	}

	p := tea.NewProgram(initialModel(eventID))
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
}
