package main

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// ── Colour palette ────────────────────────────────────────────────────────────

var (
	colGreen  = lipgloss.Color("#A8CC8C")
	colYellow = lipgloss.Color("#DBAB79")
	colBlue   = lipgloss.Color("#71BEF2")
	colPurple = lipgloss.Color("#D290E4")
	colRed    = lipgloss.Color("#E88388")
	colGray   = lipgloss.Color("#6E7681")
	colWhite  = lipgloss.Color("#CDD9E5")
	colBg     = lipgloss.Color("#22272E")
	colBorder = lipgloss.Color("#444C56")
	colSel    = lipgloss.Color("#2D333B")
)

// ── Base styles ───────────────────────────────────────────────────────────────

var (
	styleTitle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colGreen)

	styleSubtitle = lipgloss.NewStyle().
			Foreground(colGray)

	styleSelected = lipgloss.NewStyle().
			Background(colSel).
			Foreground(colWhite).
			Bold(true)

	styleGlobal = lipgloss.NewStyle().
			Foreground(colGray).
			Italic(true)

	styleInstalling = lipgloss.NewStyle().
			Foreground(colYellow)

	styleVersion = lipgloss.NewStyle().
			Foreground(colBlue)

	styleSectionHeader = lipgloss.NewStyle().
				Foreground(colPurple).
				Bold(true)

	styleStatusOK = lipgloss.NewStyle().
			Foreground(colGreen).
			Bold(true)

	styleStatusErr = lipgloss.NewStyle().
			Foreground(colRed).
			Bold(true)

	styleKey = lipgloss.NewStyle().
			Foreground(colYellow).
			Bold(true)

	styleKeyDesc = lipgloss.NewStyle().
			Foreground(colGray)

	styleInput = lipgloss.NewStyle().
			Foreground(colGreen).
			Bold(true)

	styleCursor = lipgloss.NewStyle().
			Foreground(colGreen).
			Bold(true)

	styleConfirmY = lipgloss.NewStyle().
			Foreground(colGreen).
			Bold(true)

	styleConfirmN = lipgloss.NewStyle().
			Foreground(colRed).
			Bold(true)
)

// panelBorder returns a styled box around content.
func panelBorder(title, content string, width int) string {
	style := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colBorder).
		Padding(0, 1).
		Width(width - 2)

	if title != "" {
		style = style.BorderTop(true)
	}

	box := style.Render(content)
	if title != "" {
		// Overlay title into top border
		titleStr := " " + styleTitle.Render(title) + " "
		box = strings.Replace(box, "╭──", "╭"+titleStr, 1)
	}
	return box
}

// ── Main dispatcher ───────────────────────────────────────────────────────────

func (m model) View() string {
	switch m.state {
	case stateAddName:
		return m.renderInputScreen(
			"Add Tool  [1/2]",
			"Tool name",
			"e.g. node, python, go, rust",
			"",
		)
	case stateAddVersion:
		return m.renderInputScreen(
			"Add Tool  [2/2]",
			"Version",
			"e.g. 20.11.0, latest — press Enter for latest",
			styleSubtitle.Render("tool  ")+styleTitle.Render(m.newToolName),
		)
	case stateVersionPicker:
		return m.renderVersionPicker()
	case stateConfirmDownload:
		return m.renderConfirm()
	}
	return m.renderDashboard()
}

// ── Dashboard (main list + detail sidebar) ────────────────────────────────────

func (m model) renderDashboard() string {
	totalW := m.width
	if totalW < 40 {
		totalW = 40
	}

	leftW := totalW * 55 / 100
	rightW := totalW - leftW - 1

	left := m.renderToolList(leftW)
	right := m.renderDetailPanel(rightW)

	body := lipgloss.JoinHorizontal(lipgloss.Top, left, " ", right)

	header := m.renderHeader(totalW)
	footer := m.renderFooter(totalW)
	status := m.renderStatus(totalW)

	return header + "\n" + body + "\n" + status + "\n" + footer
}

func (m model) renderHeader(width int) string {
	logo := styleTitle.Render("⚡ LazyMise")

	localLabel := m.localPath
	if localLabel == "" {
		localLabel = "(none)"
	}
	info := styleSubtitle.Render("local: "+localLabel+"   global: "+m.globalPath)

	gap := width - lipgloss.Width(logo) - lipgloss.Width(info) - 2
	if gap < 1 {
		gap = 1
	}
	return logo + strings.Repeat(" ", gap) + info
}

func (m model) renderToolList(width int) string {
	visibleRows := m.height - 7
	if visibleRows < 3 {
		visibleRows = 3
	}

	var lines []string
	lastSource := toolSource(-1)

	// Windowed scroll
	start := m.cursor - visibleRows/2
	if start < 0 {
		start = 0
	}
	end := start + visibleRows
	if end > len(m.entries) {
		end = len(m.entries)
		start = end - visibleRows
		if start < 0 {
			start = 0
		}
	}

	if len(m.entries) == 0 {
		lines = append(lines, styleGlobal.Render("  no tools configured — press 'a' to add one"))
	}

	for i := start; i < end; i++ {
		e := m.entries[i]

		// Section divider
		if e.source != lastSource {
			label := "LOCAL"
			if e.source == sourceGlobal {
				label = "GLOBAL"
			}
			divider := styleSectionHeader.Render(" "+label+" ") +
				styleSubtitle.Render(strings.Repeat("─", width-len(label)-4))
			lines = append(lines, divider)
			lastSource = e.source
		}

		selected := i == m.cursor
		arrow := "  "
		if selected {
			arrow = styleCursor.Render("▶ ")
		}

		nameStyle := lipgloss.NewStyle().Foreground(colWhite).Width(18)
		if e.source == sourceGlobal {
			nameStyle = nameStyle.Foreground(colGray)
		}
		if selected {
			nameStyle = nameStyle.Foreground(colGreen).Bold(true)
		}

		ver := styleVersion.Render(e.version)

		spin := ""
		if m.installingTools[e.name] {
			spin = "  " + styleInstalling.Render("⟳ installing…")
		}

		row := arrow + nameStyle.Render(e.name) + "  " + ver + spin

		if selected {
			row = lipgloss.NewStyle().
				Background(colSel).
				Width(width - 4).
				Render(row)
		}
		lines = append(lines, "  "+row)
	}

	content := strings.Join(lines, "\n")
	return panelBorder("Tools", content, width)
}

func (m model) renderDetailPanel(width int) string {
	if len(m.entries) == 0 || m.cursor >= len(m.entries) {
		return panelBorder("Detail", styleGlobal.Render("  nothing selected"), width)
	}

	e := m.entries[m.cursor]

	sourceLabel := "local  (editable)"
	sourceStyle := styleStatusOK
	if e.source == sourceGlobal {
		sourceLabel = "global (read-only)"
		sourceStyle = styleGlobal
	}

	installing := ""
	if m.installingTools[e.name] {
		installing = "\n\n" + styleInstalling.Render("  ⟳  mise install running in background…")
	}

	content := fmt.Sprintf(
		"%s\n%s\n\n%s\n%s\n\n%s\n%s\n%s",
		styleSubtitle.Render("name"),
		"  "+styleTitle.Render(e.name),
		styleSubtitle.Render("version"),
		"  "+styleVersion.Render(e.version),
		styleSubtitle.Render("scope"),
		"  "+sourceStyle.Render(sourceLabel),
		installing,
	)

	return panelBorder("Detail", content, width)
}

func (m model) renderStatus(width int) string {
	if m.statusMsg == "" {
		return styleSubtitle.Render("  ready")
	}
	style := styleStatusErr
	if m.statusIsOK {
		style = styleStatusOK
	}
	return "  " + style.Render(m.statusMsg)
}

func (m model) renderFooter(width int) string {
	bindings := []struct{ key, desc string }{
		{"↑/k ↓/j", "navigate"},
		{"Enter", "versions"},
		{"a", "add"},
		{"d", "delete"},
		{"x", "install all"},
		{"q", "quit"},
	}
	var parts []string
	for _, b := range bindings {
		parts = append(parts, styleKey.Render(b.key)+" "+styleKeyDesc.Render(b.desc))
	}
	bar := "  " + strings.Join(parts, styleSubtitle.Render("  ·  "))
	return lipgloss.NewStyle().
		Foreground(colGray).
		Width(width).
		Render(bar)
}

// ── Input screen ──────────────────────────────────────────────────────────────

func (m model) renderInputScreen(title, label, hint, extra string) string {
	cursor := styleCursor.Render("█")
	inputLine := styleInput.Render(m.inputBuffer) + cursor

	content := ""
	if extra != "" {
		content += extra + "\n\n"
	}
	content += styleSubtitle.Render(label+":") + "\n"
	content += "  " + inputLine + "\n\n"
	content += styleGlobal.Render("  "+hint) + "\n\n"
	content += styleKey.Render("Enter") + styleKeyDesc.Render(" confirm") +
		"   " + styleKey.Render("Esc") + styleKeyDesc.Render(" cancel")

	box := panelBorder(title, content, 60)

	// Centre vertically
	topPad := (m.height - strings.Count(box, "\n") - 4) / 2
	if topPad < 0 {
		topPad = 0
	}
	return strings.Repeat("\n", topPad) + box
}

// ── Version picker ────────────────────────────────────────────────────────────

func (m model) renderVersionPicker() string {
	totalW := m.width
	if totalW < 40 {
		totalW = 40
	}
	listW := totalW * 55 / 100
	infoW := totalW - listW - 1

	// Left: version list
	visibleRows := m.height - 8
	if visibleRows < 3 {
		visibleRows = 3
	}

	var listLines []string

	if m.loadingVersions {
		listLines = append(listLines, "  "+styleInstalling.Render("⟳  Fetching versions…"))
	} else if len(m.versionList) == 0 {
		listLines = append(listLines, "  "+styleGlobal.Render("No versions found."))
	} else {
		start := m.versionCursor - visibleRows/2
		if start < 0 {
			start = 0
		}
		end := start + visibleRows
		if end > len(m.versionList) {
			end = len(m.versionList)
			start = end - visibleRows
			if start < 0 {
				start = 0
			}
		}
		for i := start; i < end; i++ {
			v := m.versionList[i]
			if i == m.versionCursor {
				row := styleCursor.Render("▶ ") +
					lipgloss.NewStyle().
						Background(colSel).
						Foreground(colGreen).
						Bold(true).
						Width(listW - 6).
						Render(v)
				listLines = append(listLines, "  "+row)
			} else {
				listLines = append(listLines, "  "+styleVersion.Render(v))
			}
		}
		listLines = append(listLines, "")
		progress := fmt.Sprintf("  %d / %d", m.versionCursor+1, len(m.versionList))
		listLines = append(listLines, styleSubtitle.Render(progress))
	}

	leftPanel := panelBorder(
		"Versions — "+m.pendingTool,
		strings.Join(listLines, "\n"),
		listW,
	)

	// Right: info panel
	infoContent := styleSubtitle.Render("tool") + "\n" +
		"  " + styleTitle.Render(m.pendingTool) + "\n\n"

	if !m.loadingVersions && len(m.versionList) > 0 {
		infoContent += styleSubtitle.Render("selected") + "\n" +
			"  " + styleVersion.Render(m.versionList[m.versionCursor]) + "\n\n"
	}

	infoContent += styleSubtitle.Render("tip") + "\n" +
		styleGlobal.Render("  Enter     confirm + prompt\n  S+Enter   force install\n  Esc/q     go back")

	rightPanel := panelBorder("Info", infoContent, infoW)

	body := lipgloss.JoinHorizontal(lipgloss.Top, leftPanel, " ", rightPanel)
	footer := m.renderFooterPicker(totalW)
	return body + "\n" + footer
}

func (m model) renderFooterPicker(width int) string {
	bindings := []struct{ key, desc string }{
		{"↑/k ↓/j", "navigate"},
		{"Enter", "select"},
		{"Shift+Enter", "force install"},
		{"Esc/q", "back"},
	}
	var parts []string
	for _, b := range bindings {
		parts = append(parts, styleKey.Render(b.key)+" "+styleKeyDesc.Render(b.desc))
	}
	return "  " + strings.Join(parts, styleSubtitle.Render("  ·  "))
}

// ── Confirm pop-up ────────────────────────────────────────────────────────────

func (m model) renderConfirm() string {
	content := fmt.Sprintf(
		"%s\n  %s\n\n%s\n  %s\n\n%s\n  %s\n\n%s\n  %s                %s",
		styleSubtitle.Render("tool"),
		styleTitle.Render(m.pendingTool),
		styleSubtitle.Render("version"),
		styleVersion.Render(m.pendingVersion),
		styleSubtitle.Render("action"),
		styleGlobal.Render("write to TOML immediately, download in background"),
		styleSubtitle.Render("confirm?"),
		styleConfirmY.Render("[y] Yes, install"),
		styleConfirmN.Render("[n] Cancel"),
	)

	box := panelBorder("Confirm Install", content, 54)

	topPad := (m.height - strings.Count(box, "\n") - 4) / 2
	if topPad < 0 {
		topPad = 0
	}
	return strings.Repeat("\n", topPad) + box
}
