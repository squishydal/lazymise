package main

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch m.state {
		case stateList:
			return m.updateList(msg)
		case stateAddName, stateAddVersion:
			return m.updateInput(msg)
		}
	}
	return m, nil
}

func (m model) updateList(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "q":
		return m, tea.Quit

	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}

	case "down", "j":
		if m.cursor < len(m.entries)-1 {
			m.cursor++
		}

	case "a":
		m.state = stateAddName
		m.inputBuffer = ""
		m.newToolName = ""
		m.statusMsg = ""

	case "d":
		if len(m.entries) == 0 {
			break
		}
		entry := m.entries[m.cursor]
		if entry.source == sourceGlobal {
			m.statusMsg = "Cannot delete a global tool here — edit ~/.mise.toml directly."
			break
		}
		delete(m.localTools, entry.name)
		m.rebuildEntries()
		if m.cursor >= len(m.entries) && m.cursor > 0 {
			m.cursor--
		}
		m.saveLocal()
		m.statusMsg = "Deleted: " + entry.name
	}
	return m, nil
}

func (m model) updateInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		return m, tea.Quit

	case "esc":
		m.state = stateList
		m.inputBuffer = ""
		m.newToolName = ""
		m.statusMsg = "Cancelled."

	case "enter":
		input := strings.TrimSpace(m.inputBuffer)
		if input == "" {
			return m, nil
		}
		switch m.state {
		case stateAddName:
			m.newToolName = input
			m.inputBuffer = ""
			m.state = stateAddVersion

		case stateAddVersion:
			m.localTools[m.newToolName] = input
			m.rebuildEntries()
			// Move cursor to the newly added entry
			for i, e := range m.entries {
				if e.name == m.newToolName && e.source == sourceLocal {
					m.cursor = i
					break
				}
			}
			m.saveLocal()
			m.statusMsg = "Added: " + m.newToolName + " " + input
			m.state = stateList
			m.inputBuffer = ""
			m.newToolName = ""
		}

	case "backspace":
		if len(m.inputBuffer) > 0 {
			m.inputBuffer = m.inputBuffer[:len(m.inputBuffer)-1]
		}

	default:
		if len(msg.String()) == 1 {
			m.inputBuffer += msg.String()
		}
	}
	return m, nil
}
