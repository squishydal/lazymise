package main

import (
	"fmt"
	"os"

	"github.com/BurntSushi/toml"
	tea "github.com/charmbracelet/bubbletea"
)

type MiseConfig struct {
	Tools map[string]string `toml:"tools"`
}

type model struct {
	counter int
	tools   map[string]string
}

func check(e error) {
	if e != nil {
		panic(e)
	}
}

func readFiles() []byte {
	path := "/home/squishydal/Projects/lazymise/mise.toml"
	dat, err := os.ReadFile(path)
	check(err)
	return dat
}

func initialModel() model {
	rawFile := readFiles()
	var config MiseConfig
	err := toml.Unmarshal(rawFile, &config)
	check(err)
	return model{
		counter: 0,
		tools:   config.Tools,
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

func (m model) View() string {
	s := fmt.Sprintf("Counter: %d\n\n", m.counter)
	s += "Your Tools:\n\n"
	for toolName, version := range m.tools {
		s += fmt.Sprintf("  -> %s: %s\n", toolName, version)
	}
	s += "\nPress up/down to change, q to quit."
	return s
}

func main() {
	p := tea.NewProgram(initialModel())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error: %v", err)
		os.Exit(1)
	}
}
