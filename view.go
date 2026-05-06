package main

import "fmt"

func (m model) View() string {
	switch m.state {
	case stateAddName:
		return renderPrompt(
			"Add Tool — Step 1/2",
			"Tool name (e.g. python, node, go)",
			m.inputBuffer,
			"",
		)
	case stateAddVersion:
		return renderPrompt(
			"Add Tool — Step 2/2",
			fmt.Sprintf("Version for %s (e.g. 3.11, latest)", m.newToolName),
			m.inputBuffer,
			fmt.Sprintf("Tool: %s\n", m.newToolName),
		)
	}
	return m.renderList()
}

func (m model) renderList() string {
	// Header
	localLabel := m.localPath
	if localLabel == "" {
		localLabel = "(none — will be created on first add)"
	}
	s := "LazyMise\n\n"
	s += fmt.Sprintf("  local  : %s\n", localLabel)
	s += fmt.Sprintf("  global : %s\n\n", m.globalPath)

	if len(m.entries) == 0 {
		s += "  (no tools configured — press 'a' to add one)\n"
	} else {
		lastSource := toolSource(-1) // sentinel

		for i, e := range m.entries {
			// Section header when source changes
			if e.source != lastSource {
				if e.source == sourceLocal {
					s += "  ── local ──────────────────────────\n"
				} else {
					s += "  ── global (read-only) ──────────────\n"
				}
				lastSource = e.source
			}

			cursor := "   "
			if i == m.cursor {
				cursor = " ▶ "
			}

			hint := ""
			if e.source == sourceGlobal {
				hint = "  [global]"
			}

			s += fmt.Sprintf("%s%-20s %s%s\n", cursor, e.name, e.version, hint)
		}
	}

	if m.statusMsg != "" {
		s += "\n» " + m.statusMsg
	}

	s += "\n\n[↑/k ↓/j] navigate  [a] add  [d] delete local  [q] quit"
	return s
}

func renderPrompt(title, label, input, extra string) string {
	s := title + "\n\n"
	if extra != "" {
		s += extra + "\n"
	}
	s += label + ": " + input + "█\n\n"
	s += "[Enter] confirm  [Esc] cancel"
	return s
}
