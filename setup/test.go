package main

import (
	"fmt"
	"os"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	green  = lipgloss.NewStyle().Foreground(lipgloss.Color("46"))
	yellow = lipgloss.NewStyle().Foreground(lipgloss.Color("220"))
	red    = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
)

type model struct {
	spinner   spinner.Model
	choice    int
	confirmed bool
	quitting  bool
}

func initialModel() model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))
	return model{
		spinner:   s,
		choice:    0,
		confirmed: false,
		quitting:  false,
	}
}

func (m model) Init() tea.Cmd {
	return m.spinner.Tick
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "k", "up":
			if m.choice > 0 {
				m.choice--
			}
		case "j", "down":
			if m.choice < 1 {
				m.choice++
			}
		case "enter", " ":
			if m.choice == 0 {
				m.confirmed = true
				return m, tea.Tick(time.Second, func(time.Time) tea.Msg {
					return nil
				})
			} else {
				m.quitting = true
				return m, tea.Quit
			}
		case "q", "esc", "ctrl+c":
			m.quitting = true
			return m, tea.Quit
		}
	case nil:
		return m, tea.Quit
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}
	return m, nil
}

func (m model) View() string {
	if m.confirmed {
		return fmt.Sprintf(
			"\n%s Installing docker...\n\n",
			m.spinner.View(),
		)
	}
	if m.quitting {
		return red.Render("\nCancelled.\n\n")
	}
	question := yellow.Render("Install docker? [Y/n]")
	choices := fmt.Sprintf(
		"%s\n%s\n",
		highlightChoice(m.choice == 0, green.Render("Yes")),
		highlightChoice(m.choice == 1, red.Render("No")),
	)
	return fmt.Sprintf(
		"%s\n\n%s\npress q or esc to quit\n",
		question,
		choices,
	)
}

func highlightChoice(selected bool, choice string) string {
	if selected {
		return fmt.Sprintf("> %s", choice)
	}
	return fmt.Sprintf("  %s", choice)
}

func main() {
	p := tea.NewProgram(initialModel())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Alas, there's been an error: %v", err)
		os.Exit(1)
	}
}
