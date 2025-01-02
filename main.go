package main

import (
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"golang.org/x/crypto/ssh"
	"gopkg.in/yaml.v2"
)

type Server struct {
	Name     string `yaml:"name"`
	Host     string `yaml:"host"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
}

type Config struct {
	Servers []Server `yaml:"servers"`
}

type TestResult struct {
	Server    string
	Target    string
	Success   bool
	Output    string
	Timestamp time.Time
}

type Model struct {
	servers    []Server
	results    []TestResult
	spinner    spinner.Model
	table      table.Model
	testing    bool
	err        error
	resultChan chan TestResult
	textInput  textinput.Model
	inputting  bool
	targetIP   string
}

var style = lipgloss.NewStyle().
	BorderStyle(lipgloss.NormalBorder()).
	BorderForeground(lipgloss.Color("240"))

func initialModel() Model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	columns := []table.Column{
		{Title: "Server", Width: 15},
		{Title: "Target", Width: 20},
		{Title: "Status", Width: 10},
		{Title: "Time", Width: 20},
	}

	t := table.New(
		table.WithColumns(columns),
		table.WithHeight(10),
		table.WithWidth(72),
	)

	//text input initialization
	ti := textinput.New()
	ti.Placeholder = "Enter target IP(e.g., 8.8.8.8)"
	ti.Focus()
	ti.CharLimit = 15
	ti.Width = 30

	//get servers from yaml file
	servers, err := getServerDetails("servers.yaml")
	if err != nil {
		log.Fatalf("Error reading server details: %v", err)
	}

	return Model{
		spinner:    s,
		table:      t,
		servers:    servers,
		resultChan: make(chan TestResult, 10),
		textInput:  ti,
		inputting:  false,
	}

}

// get server details from yaml file
func getServerDetails(filename string) ([]Server, error) {
	//var servers []Server
	config := Config{}
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		return nil, err
	}
	return config.Servers, nil
}

func (m Model) Init() tea.Cmd {
	return m.spinner.Tick
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			close(m.resultChan)
			return m, tea.Quit
		case "t":
			if !m.testing && !m.inputting {
				m.inputting = true
				m.textInput.Focus()
				return m, textinput.Blink
			}
		case "enter":
			if m.inputting {
				if m.textInput.Value() != "" {
					m.targetIP = m.textInput.Value()
					m.inputting = false
					m.testing = true
					m.results = []TestResult{}
					m.resultChan = make(chan TestResult, len(m.servers))
					return m, runTests(m.servers, m.resultChan, m.targetIP)
				}
			}
		case "esc":
			if m.inputting {
				m.inputting = false
				m.textInput.Blur()
				return m, nil
			}
		}

	case TestResult:
		m.results = append(m.results, msg)
		rows := []table.Row{}
		for _, r := range m.results {
			status := "❌"
			if r.Success {
				status = "✅"
			}
			rows = append(rows, table.Row{
				r.Server,
				r.Target,
				status,
				r.Timestamp.Format(time.RFC822),
			})
		}
		m.table.SetRows(rows)

		// If we're still testing, check for more results
		if m.testing && len(m.results) < len(m.servers) {
			return m, tea.Batch(
				m.spinner.Tick,
				CheckMoreResults(m.resultChan),
			)
		}

		// If all results are in, stop testing
		if len(m.results) == len(m.servers) {
			m.testing = false
		}

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}
	if m.inputting {
		var cmd tea.Cmd
		m.textInput, cmd = m.textInput.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m Model) View() string {
	if m.err != nil {
		return fmt.Sprintf("Error: %v\n", m.err)
	}

	var s strings.Builder
	s.WriteString("\n  🌐 Network Connectivity Tester\n\n")

	if m.inputting {
		s.WriteString("Enter target IP for ping tests:\n")
		s.WriteString(m.textInput.View() + "\n")
		s.WriteString("\n(Press Enter to start, Esc to cancel)\n")
	} else if m.testing {
		s.WriteString(fmt.Sprintf("Target IP: %s\n", m.targetIP))
		s.WriteString(m.spinner.View() + " Running tests...\n\n")
	} else if len(m.results) > 0 {
		s.WriteString(fmt.Sprintf("Target IP: %s\n", m.targetIP))
		s.WriteString("✨ Tests complete!\n")
		s.WriteString("Press 't' to run tests again, or 'q' to quit\n\n")
	} else {
		s.WriteString("Press 't' to start tests, 'q' to quit\n\n")
	}

	s.WriteString(style.Render(m.table.View()) + "\n")

	if len(m.results) > 0 && !m.testing {
		successCount := 0
		for _, r := range m.results {
			if r.Success {
				successCount++
			}
		}
		s.WriteString(fmt.Sprintf("\nSummary: %d/%d tests passed\n",
			successCount, len(m.results)))
	}

	return s.String()
}

func runTests(servers []Server, resultChan chan TestResult, targetIP string) tea.Cmd {
	return func() tea.Msg {
		var wg sync.WaitGroup

		for _, server := range servers {
			wg.Add(1)
			go func(srv Server) {
				defer wg.Done()
				result := testServer(srv, targetIP)
				resultChan <- result
			}(server)
		}

		// Close results channel after all tests complete
		go func() {
			wg.Wait()
		}()

		if result, ok := <-resultChan; ok {
			return result
		}
		return nil

	}
}

func CheckMoreResults(resultChan chan TestResult) tea.Cmd {
	return func() tea.Msg {
		if result, ok := <-resultChan; ok {
			return result
		}
		return nil
	}
}

func (m Model) Exit() {
	close(m.resultChan)
}

func testServer(server Server, targetIP string) TestResult {

	config := &ssh.ClientConfig{
		User: server.User,
		Auth: []ssh.AuthMethod{
			ssh.Password(server.Password),
			ssh.KeyboardInteractive(func(user, instruction string, questions []string, echos []bool) ([]string, error) {
				answers := make([]string, len(questions))
				for i := range questions {
					answers[i] = server.Password
				}
				return answers, nil
			}),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         15 * time.Second,
	}

	client, err := ssh.Dial("tcp", server.Host+":22", config)
	if err != nil {
		return TestResult{
			Server:    server.Name,
			Target:    "SSH Connection",
			Success:   false,
			Output:    err.Error(),
			Timestamp: time.Now(),
		}
	}
	defer client.Close()

	// Run ping test
	session, err := client.NewSession()
	if err != nil {
		return TestResult{
			Server:    server.Name,
			Target:    "Session Creation",
			Success:   false,
			Output:    err.Error(),
			Timestamp: time.Now(),
		}
	}
	defer session.Close()

	//var outBuf bytes.Buffer
	//session.Stdout = &outBuf

	// Example ping test (adjust target as needed)
	//target := "10.1.0.1" // Example target
	//err = session.Run(fmt.Sprintf("ping %s", targetIP))
	output, err := session.CombinedOutput("ping -c 3 " + targetIP)

	success := err == nil && !strings.Contains(string(output), "100% packet loss")

	// Save results to file
	logResult(TestResult{
		Server:    server.Name,
		Target:    targetIP,
		Success:   success,
		Output:    string(output),
		Timestamp: time.Now(),
	})

	return TestResult{
		Server:    server.Name,
		Target:    targetIP,
		Success:   success,
		Output:    string(output),
		Timestamp: time.Now(),
	}
}

func logResult(result TestResult) {
	f, err := os.OpenFile("network_tests.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Printf("Error opening log file: %v", err)
		return
	}
	defer f.Close()

	logEntry := fmt.Sprintf("[%s] Server: %s, Target: %s, Success: %v\nOutput:\n%s\n---\n",
		result.Timestamp.Format(time.RFC3339),
		result.Server,
		result.Target,
		result.Success,
		result.Output)

	if _, err := f.WriteString(logEntry); err != nil {
		log.Printf("Error writing to log file: %v", err)
	}
}

func main() {
	p := tea.NewProgram(initialModel())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error running program: %v", err)
		os.Exit(1)
	}
}
