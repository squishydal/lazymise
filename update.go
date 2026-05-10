package main

import (
	"os/exec"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// ── Init ──────────────────────────────────────────────────────────────────────

func (m model) Init() tea.Cmd {
	// Kick off the spinner tick loop immediately.
	return tickCmd()
}

func tickCmd() tea.Cmd {
	return tea.Tick(120*time.Millisecond, func(time.Time) tea.Msg {
		return tickMsg{}
	})
}

func statusClearCmd() tea.Cmd {
	return tea.Tick(3*time.Second, func(time.Time) tea.Msg {
		return clearStatusMsg{}
	})
}

// ── Update ────────────────────────────────────────────────────────────────────

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tickMsg:
		// Advance spinner when installing or loading.
		if len(m.installingTools) > 0 || m.loadingVersions || m.loadingRegistry {
			m.spinFrame = (m.spinFrame + 1) % len(spinFrames)
		}
		return m, tickCmd()

	case clearStatusMsg:
		m.statusMsg = ""
		return m, nil

	case tea.KeyMsg:
		switch m.state {
		case stateList:
			return m.updateList(msg)
		case stateToolBrowser:
			return m.updateToolBrowser(msg)
		case stateAddVersion:
			return m.updateInput(msg)
		case stateVersionPicker:
			return m.updateVersionPicker(msg)
		case stateConfirmDownload:
			return m.updateConfirm(msg)
		case stateConfirmDelete:
			return m.updateConfirmDelete(msg)
		}

	case registryLoadedMsg:
		m.loadingRegistry = false
		if msg.err != nil {
			m.statusMsg = "Error loading registry: " + msg.err.Error()
			m.statusIsOK = false
			m.state = stateList
			return m, statusClearCmd()
		}
		m.registryTools = msg.tools
		m.filteredTools = msg.tools
		m.browserCursor = 0
		return m, nil

	case versionsLoadedMsg:
		if msg.err != nil {
			m.statusMsg = "Error loading versions: " + msg.err.Error()
			m.statusIsOK = false
			m.state = stateList
			m.loadingVersions = false
			return m, statusClearCmd()
		}
		m.versionList = msg.versions
		// Warm the cache.
		m.versionCache[msg.tool] = msg.versions
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
		return m, statusClearCmd()
	}

	return m, nil
}

// ── List state ────────────────────────────────────────────────────────────────

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
		return m.openVersionPicker(m.entries[m.cursor].name)

	// Edit: like Enter but scoped to local tools only.
	case "e":
		if len(m.entries) == 0 {
			break
		}
		entry := m.entries[m.cursor]
		if entry.source == sourceGlobal {
			m.statusMsg = "Cannot edit a global tool — edit ~/.mise.toml directly."
			m.statusIsOK = false
			return m, statusClearCmd()
		}
		return m.openVersionPicker(entry.name)

	case "a":
		// If we already have the registry cached, go straight to browser.
		if len(m.registryTools) > 0 {
			m.filteredTools = m.registryTools
			m.browserCursor = 0
			m.searchBuffer = ""
			m.state = stateToolBrowser
			m.statusMsg = ""
			break
		}
		m.state = stateToolBrowser
		m.loadingRegistry = true
		m.searchBuffer = ""
		m.browserCursor = 0
		m.statusMsg = ""
		return m, fetchRegistry()

	case "d":
		if len(m.entries) == 0 {
			break
		}
		entry := m.entries[m.cursor]
		if entry.source == sourceGlobal {
			m.statusMsg = "Cannot delete a global tool — edit ~/.mise.toml directly."
			m.statusIsOK = false
			return m, statusClearCmd()
		}
		// Move to confirm-delete instead of deleting immediately.
		m.pendingTool = entry.name
		m.state = stateConfirmDelete

	case "x":
		if len(m.localTools) == 0 {
			m.statusMsg = "No local tools to install."
			m.statusIsOK = false
			return m, statusClearCmd()
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

// openVersionPicker centralises the shared setup for Enter and 'e'.
func (m model) openVersionPicker(tool string) (model, tea.Cmd) {
	m.pendingTool = tool
	m.versionCursor = 0
	m.statusMsg = ""
	m.state = stateVersionPicker

	// Use cached versions if available — no subprocess needed.
	if cached, ok := m.versionCache[tool]; ok {
		m.versionList = cached
		m.loadingVersions = false
		return m, nil
	}

	m.versionList = nil
	m.loadingVersions = true
	return m, fetchVersions(tool)
}

// ── Tool browser state ────────────────────────────────────────────────────────

func (m model) updateToolBrowser(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		return m, tea.Quit

	case "esc", "q":
		if m.searchBuffer != "" {
			// First Esc clears the search, second exits.
			m.searchBuffer = ""
			m.filteredTools = m.registryTools
			m.browserCursor = 0
			return m, nil
		}
		m.state = stateList

	case "up", "k":
		if m.browserCursor > 0 {
			m.browserCursor--
		}
	case "down", "j":
		if !m.loadingRegistry && m.browserCursor < len(m.filteredTools)-1 {
			m.browserCursor++
		}
	case "enter":
		if m.loadingRegistry || len(m.filteredTools) == 0 {
			break
		}
		selected := m.filteredTools[m.browserCursor]
		m.newToolName = selected.name
		m.searchBuffer = ""
		m.state = stateAddVersion
		m.inputBuffer = ""

	case "backspace":
		if len(m.searchBuffer) > 0 {
			m.searchBuffer = m.searchBuffer[:len(m.searchBuffer)-1]
			m.applyFilter()
		}

	default:
		// Any printable char goes into the search buffer.
		if len(msg.String()) == 1 {
			m.searchBuffer += msg.String()
			m.applyFilter()
		}
	}
	return m, nil
}

// applyFilter rebuilds filteredTools based on searchBuffer.
func (m *model) applyFilter() {
	if m.searchBuffer == "" {
		m.filteredTools = m.registryTools
		m.browserCursor = 0
		return
	}
	q := strings.ToLower(m.searchBuffer)
	var out []registryTool
	for _, t := range m.registryTools {
		if strings.Contains(strings.ToLower(t.name), q) ||
			strings.Contains(strings.ToLower(t.desc), q) ||
			strings.Contains(strings.ToLower(t.full), q) {
			out = append(out, t)
		}
	}
	m.filteredTools = out
	m.browserCursor = 0
}

// fetchRegistry runs `mise registry` and parses the output.
func fetchRegistry() tea.Cmd {
	return func() tea.Msg {
		out, err := exec.Command("mise", "registry").Output()
		if err != nil {
			return registryLoadedMsg{err: err}
		}
		lines := strings.Split(strings.TrimSpace(string(out)), "\n")
		var tools []registryTool
		for _, line := range lines {
			if line == "" {
				continue
			}
			// Output format: short-name   full-name   description...
			// Columns are separated by 2+ spaces (or tabs).
			fields := strings.Fields(line)
			if len(fields) == 0 {
				continue
			}
			t := registryTool{name: fields[0]}
			if len(fields) >= 2 {
				t.full = fields[1]
			}
			if len(fields) >= 3 {
				t.desc = strings.Join(fields[2:], " ")
			}
			tools = append(tools, t)
		}
		return registryLoadedMsg{tools: tools}
	}
}

// ── Input state (version entry after tool browser) ────────────────────────────

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
		return m, statusClearCmd()
	case "enter":
		input := strings.TrimSpace(m.inputBuffer)
		// Only stateAddVersion remains; stateAddName is replaced by the tool browser.
		if m.state == stateAddVersion {
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
			if err := m.saveLocal(); err != nil {
				m.statusMsg = "✗ Could not save: " + err.Error()
				m.statusIsOK = false
				return m, statusClearCmd()
			}
			m.installingTools[m.newToolName] = true
			cmd := runMiseInstall(m.newToolName, version)
			m.statusMsg = "✓ Added & queued: " + m.newToolName + " " + version
			m.statusIsOK = true
			m.state = stateList
			m.inputBuffer = ""
			m.newToolName = ""
			return m, tea.Batch(cmd, statusClearCmd())
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

// ── Version picker state ──────────────────────────────────────────────────────

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

// ── Confirm-download state ────────────────────────────────────────────────────

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

// ── Confirm-delete state ──────────────────────────────────────────────────────

func (m model) updateConfirmDelete(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		return m, tea.Quit
	case "y", "Y":
		name := m.pendingTool
		delete(m.localTools, name)
		m.rebuildEntries()
		if m.cursor >= len(m.entries) && m.cursor > 0 {
			m.cursor--
		}
		m.pendingTool = ""
		m.state = stateList
		if err := m.saveLocal(); err != nil {
			m.statusMsg = "✗ Could not save: " + err.Error()
			m.statusIsOK = false
			return m, statusClearCmd()
		}
		m.statusMsg = "✓ Deleted: " + name
		m.statusIsOK = true
		return m, statusClearCmd()
	case "n", "N", "esc", "q":
		m.pendingTool = ""
		m.state = stateList
	}
	return m, nil
}

// ── Shared helpers ────────────────────────────────────────────────────────────

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
	if err := m.saveLocal(); err != nil {
		m.statusMsg = "✗ Could not save: " + err.Error()
		m.statusIsOK = false
		m.state = stateList
		m.versionList = nil
		m.pendingVersion = ""
		return m, statusClearCmd()
	}
	m.installingTools[tool] = true
	m.statusMsg = "✓ Queued install: " + tool + " " + version
	m.statusIsOK = true
	m.state = stateList
	m.versionList = nil
	m.pendingVersion = ""
	return m, tea.Batch(runMiseInstall(tool, version), statusClearCmd())
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
		// Reverse so newest is at the top.
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
