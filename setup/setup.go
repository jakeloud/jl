package setup

import (
	"fmt"
	"os"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
)

var dry bool = false

type model struct {
	spinner   spinner.Model
	choice    int
	confirmed bool
	quitting  bool
}

func initialModel() model {
	s := spinner.New()
	s.Spinner = spinner.Dot
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

type finishMsg struct{}

func installDocker() tea.Cmd {
	return func() tea.Msg {
		out, err := execWrapped(dry, "apt-get install -y docker.io")
		if err != nil {
			fmt.Println(out)
			fmt.Print(err)
		}

		return finishMsg{}
	}
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter", "y", "Y":
			m.confirmed = true
			return m, installDocker()
		case "n", "esc", "ctrl+c":
			m.quitting = true
			return m, tea.Quit
		}
	case finishMsg:
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
			"%s Installing docker...\n",
			m.spinner.View(),
		)
	}
	if m.quitting {
		return "NOT installing docker.\n"
	}
	return fmt.Sprintf("Install docker? [Y/n]")
}

func Start(d bool) {
	dry = d
	p := tea.NewProgram(initialModel())
	start := time.Now()

	fmt.Printf("Installing required packages\n")
	install(dry)
	err := setupService(dry)
	if err != nil {
		fmt.Fprint(os.Stderr, err)
		os.Exit(1)
	}
  cmd := "ssh-keyscan -H github.com >> ~/.ssh/known_hosts"
  _, err = execWrapped(dry, cmd)
  if err != nil {
    fmt.Fprint(os.Stderr, err)
    os.Exit(1)
  }

	if _, err = p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Alas, there's been an error: %v", err)
		os.Exit(1)
	}

	l, err := link()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Alas, there's been an error: %v", err)
		os.Exit(1)
	}
	fmt.Printf("JakeLoud successfully installed! 🎊 took %s\n", time.Since(start))
	fmt.Printf("go to %s to finish installation\n", l)
}
