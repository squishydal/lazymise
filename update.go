package main

import (
	"os/exec"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		switch m.state {
		case stateList:
			return m.updateList(msg)
		case stateAddName, stateAddVersion:
			return m.updateInput(msg)
		case stateVersionPicker:
			return m.updateVersionPicker(msg)
		case stateConfirmDownload:
			return m.updateConfirm(msg)
		}

	case versionsLoadedMsg:
		if msg.err != nil {
			m.statusMsg = "Error loading versions: " + msg.err.Error()
			m.statusIsOK = false
			m.state = stateList
			m.loadingVersions = false
			return m, nil
		}
		m.versionList = msg.versions
		m.loadingVersions = false
		return m, nil

	case installDoneMsg:
		delete(m.installingTools, msg.tool)
		if msg.err != nil {
			m.statusMsg = "✗ Install failed — " + msg.tool + ": " + msg.err.Error()
			m.statusIsOK = false
		} else {
			m.statusMsg = "✓ Installed: " + msg.tool + " " + msg.version
			m.statusIsOK = true
		}
		return m, nil
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
	case "enter":
		if len(m.entries) == 0 {
			break
		}
		entry := m.entries[m.cursor]
		m.pendingTool = entry.name
		m.versionList = nil
		m.versionCursor = 0
		m.loadingVersions = true
		m.statusMsg = ""
		m.state = stateVersionPicker
		return m, fetchVersions(entry.name)
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
			m.statusMsg = "Cannot delete a global tool — edit ~/.mise.toml directly."
			m.statusIsOK = false
			break
		}
		delete(m.localTools, entry.name)
		m.rebuildEntries()
		if m.cursor >= len(m.entries) && m.cursor > 0 {
			m.cursor--
		}
		m.saveLocal()
		m.statusMsg = "✓ Deleted: " + entry.name
		m.statusIsOK = true
	case "x":
		if len(m.localTools) == 0 {
			m.statusMsg = "No local tools to install."
			m.statusIsOK = false
			break
		}
		var cmds []tea.Cmd
		for name, version := range m.localTools {
			if !m.installingTools[name] {
				m.installingTools[name] = true
				cmds = append(cmds, runMiseInstall(name, version))
			}
		}
		m.statusMsg = "Running mise install for all local tools…"
		m.statusIsOK = true
		return m, tea.Batch(cmds...)
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
		m.statusIsOK = false
	case "enter":
		input := strings.TrimSpace(m.inputBuffer)
		switch m.state {
		case stateAddName:
			if input == "" {
				return m, nil
			}
			m.newToolName = input
			m.inputBuffer = ""
			m.state = stateAddVersion
		case stateAddVersion:
			// Empty → default to "latest"
			if input == "" {
				input = "latest"
			}
			version := input
			m.localTools[m.newToolName] = version
			m.rebuildEntries()
			for i, e := range m.entries {
				if e.name == m.newToolName && e.source == sourceLocal {
					m.cursor = i
					break
				}
			}
			m.saveLocal()
			m.installingTools[m.newToolName] = true
			cmd := runMiseInstall(m.newToolName, version)
			m.statusMsg = "✓ Added & queued: " + m.newToolName + " " + version
			m.statusIsOK = true
			m.state = stateList
			m.inputBuffer = ""
			m.newToolName = ""
			return m, cmd
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

func (m model) updateVersionPicker(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		return m, tea.Quit
	case "esc", "q":
		m.state = stateList
		m.versionList = nil
		m.loadingVersions = false
	case "up", "k":
		if m.versionCursor > 0 {
			m.versionCursor--
		}
	case "down", "j":
		if !m.loadingVersions && m.versionCursor < len(m.versionList)-1 {
			m.versionCursor++
		}
	case "enter":
		if m.loadingVersions || len(m.versionList) == 0 {
			break
		}
		m.pendingVersion = m.versionList[m.versionCursor]
		m.state = stateConfirmDownload
	case "shift+enter":
		if m.loadingVersions || len(m.versionList) == 0 {
			break
		}
		return m.commitInstall(m.versionList[m.versionCursor])
	}
	return m, nil
}

func (m model) updateConfirm(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		return m, tea.Quit
	case "y", "Y":
		return m.commitInstall(m.pendingVersion)
	case "n", "N", "esc", "q":
		m.pendingVersion = ""
		m.state = stateVersionPicker
	}
	return m, nil
}

func (m model) commitInstall(version string) (model, tea.Cmd) {
	tool := m.pendingTool
	m.localTools[tool] = version
	m.rebuildEntries()
	for i, e := range m.entries {
		if e.name == tool && e.source == sourceLocal {
			m.cursor = i
			break
		}
	}
	m.saveLocal()
	m.installingTools[tool] = true
	m.statusMsg = "✓ Queued install: " + tool + " " + version
	m.statusIsOK = true
	m.state = stateList
	m.versionList = nil
	m.pendingVersion = ""
	return m, runMiseInstall(tool, version)
}

func fetchVersions(tool string) tea.Cmd {
	return func() tea.Msg {
		out, err := exec.Command("mise", "ls-remote", tool).Output()
		if err != nil {
			return versionsLoadedMsg{tool: tool, err: err}
		}
		raw := strings.TrimSpace(string(out))
		if raw == "" {
			return versionsLoadedMsg{tool: tool}
		}
		lines := strings.Split(raw, "\n")
		for i, j := 0, len(lines)-1; i < j; i, j = i+1, j-1 {
			lines[i], lines[j] = lines[j], lines[i]
		}
		return versionsLoadedMsg{tool: tool, versions: lines}
	}
}

func runMiseInstall(tool, version string) tea.Cmd {
	return func() tea.Msg {
		err := exec.Command("mise", "install", tool+"@"+version).Run()
		return installDoneMsg{tool: tool, version: version, err: err}
	}
}
