package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
)

type model struct {
	counter     int
	text        string
	fileContent string
}

func check(e error) {
	if e != nil {
		panic(e)
	}
}

func readFiles() string {
	path := "/home/squishydal/Projects/lazymise/mise.toml"
	dat, err := os.ReadFile(path)
	check(err)
	return string(dat)
}

func initialModel() model {
	content := readFiles()
	return model{
		counter:     0,
		text:        "",
		fileContent: content,
	}
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "up", "k":
			m.counter++
		case "down", "j":
			m.counter--
		}
	}
	return m, nil
}

// 4. VIEW: Render the UI as a string
func (m model) View() string {
	s := fmt.Sprintf("Counter: %d\n\n", m.counter)
	s += fmt.Sprintf("File contect: \n%s\n\n", m.fileContent)
	s += "Press up/down to change, q to quit."
	return s
}

func main() {
	p := tea.NewProgram(initialModel())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error: %v", err)
		os.Exit(1)
	}
}
