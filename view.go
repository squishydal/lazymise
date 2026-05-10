package main

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// ── Catppuccin Mocha palette ──────────────────────────────────────────────────

var (
	colSurface = lipgloss.Color("#313244")
	colOverlay = lipgloss.Color("#6c7086")
	colText    = lipgloss.Color("#cdd6f4")
	colSubtext = lipgloss.Color("#a6adc8")

	colPink    = lipgloss.Color("#f5c2e7")
	colMauve   = lipgloss.Color("#cba6f7")
	colRed     = lipgloss.Color("#f38ba8")
	colPeach   = lipgloss.Color("#fab387")
	colGreen   = lipgloss.Color("#a6e3a1")
	colTeal    = lipgloss.Color("#94e2d5")
	colSky     = lipgloss.Color("#89dceb")
	colBlue    = lipgloss.Color("#89b4fa") //nolint:unused
	colLavndr  = lipgloss.Color("#b4befe")
	colSapphir = lipgloss.Color("#74c7ec")

	colBorder = lipgloss.Color("#45475a")
	colBase   = lipgloss.Color("#1e1e2e")
)

// ── Styles ────────────────────────────────────────────────────────────────────

var (
	styleLogo = lipgloss.NewStyle().Bold(true).Foreground(colPink)
	styleTitle = lipgloss.NewStyle().Bold(true).Foreground(colMauve)
	styleLabel = lipgloss.NewStyle().Foreground(colSubtext).Italic(true)
	styleGlobal = lipgloss.NewStyle().Foreground(colOverlay).Italic(true)
	styleSubtitle = lipgloss.NewStyle().Foreground(colOverlay)
	styleLavender = lipgloss.NewStyle().Foreground(colLavndr)
	styleVersion = lipgloss.NewStyle().Foreground(colSapphir)

	styleSectionLocal  = lipgloss.NewStyle().Bold(true).Foreground(colMauve)
	styleSectionGlobal = lipgloss.NewStyle().Bold(true).Foreground(colOverlay)

	styleInstalling = lipgloss.NewStyle().Foreground(colPeach)
	styleStatusOK   = lipgloss.NewStyle().Bold(true).Foreground(colGreen)
	styleStatusErr  = lipgloss.NewStyle().Bold(true).Foreground(colRed)
	styleDanger     = lipgloss.NewStyle().Bold(true).Foreground(colRed)

	styleInput  = lipgloss.NewStyle().Bold(true).Foreground(colTeal)
	styleCursor = lipgloss.NewStyle().Bold(true).Foreground(colPink)

	// Pill badges — coloured background, dark text.
	stylePillLocal     = lipgloss.NewStyle().Padding(0, 1).Background(colMauve).Foreground(colBase).Bold(true)
	stylePillGlobal    = lipgloss.NewStyle().Padding(0, 1).Background(colOverlay).Foreground(colBase)
	stylePillInstalled = lipgloss.NewStyle().Padding(0, 1).Background(colGreen).Foreground(colBase).Bold(true)
	stylePillCached    = lipgloss.NewStyle().Padding(0, 1).Background(colSky).Foreground(colBase)

	// Confirm buttons.
	styleConfirmY = lipgloss.NewStyle().Padding(0, 2).Background(colGreen).Foreground(colBase).Bold(true)
	styleConfirmN = lipgloss.NewStyle().Padding(0, 2).Background(colRed).Foreground(colBase).Bold(true)

	// Footer keys.
	styleKey     = lipgloss.NewStyle().Padding(0, 1).Background(colMauve).Foreground(colBase).Bold(true)
	styleKeyDesc = lipgloss.NewStyle().Foreground(colSubtext)
)

// ── Helpers ───────────────────────────────────────────────────────────────────

func panelBorder(title, content string, width int) string {
	s := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colBorder).
		Padding(0, 1).
		Width(width - 2)
	box := s.Render(content)
	if title != "" {
		badge := " " + styleTitle.Render(title) + " "
		box = strings.Replace(box, "╭──", "╭"+badge, 1)
	}
	return box
}

func keyBind(k, desc string) string {
	return styleKey.Render(k) + " " + styleKeyDesc.Render(desc)
}

func footerBar(bindings []struct{ key, desc string }) string {
	var parts []string
	for _, b := range bindings {
		parts = append(parts, keyBind(b.key, b.desc))
	}
	return "  " + strings.Join(parts, "  ")
}

// ── View dispatcher ───────────────────────────────────────────────────────────

func (m model) View() string {
	switch m.state {
	case stateToolBrowser:
		return m.renderToolBrowser()
	case stateAddVersion:
		return m.renderInputScreen(
			"🫧 pick a version",
			"version",
			"e.g. 20.11.0  ·  latest  ·  Enter for latest",
			styleLabel.Render("tool ❯ ")+styleTitle.Render(m.newToolName),
		)
	case stateVersionPicker:
		return m.renderVersionPicker()
	case stateConfirmDownload:
		return m.renderConfirm()
	case stateConfirmDelete:
		return m.renderConfirmDelete()
	}
	return m.renderDashboard()
}

// ── Dashboard ─────────────────────────────────────────────────────────────────

func (m model) renderDashboard() string {
	totalW := m.width
	if totalW < 40 {
		totalW = 40
	}
	leftW := totalW * 56 / 100
	rightW := totalW - leftW - 1

	body := lipgloss.JoinHorizontal(lipgloss.Top,
		m.renderToolList(leftW), " ", m.renderDetailPanel(rightW))

	return m.renderHeader(totalW) + "\n" +
		body + "\n" +
		m.renderStatus(totalW) + "\n" +
		m.renderFooter(totalW)
}

func (m model) renderHeader(width int) string {
	logo := styleLogo.Render("🫧 lazymise")

	localLabel := m.localPath
	if localLabel == "" {
		localLabel = "none"
	}
	info := styleSubtitle.Render("local: ") + styleLabel.Render(localLabel) +
		styleSubtitle.Render("   global: ") + styleLabel.Render(m.globalPath)

	gap := width - lipgloss.Width(logo) - lipgloss.Width(info) - 2
	if gap < 1 {
		gap = 1
	}
	header := logo + strings.Repeat(" ", gap) + info
	if m.initErr != "" {
		header += "\n" + styleStatusErr.Render("  ⚠  "+m.initErr)
	}
	return header
}

func (m model) renderToolList(width int) string {
	visibleRows := m.height - 7
	if visibleRows < 3 {
		visibleRows = 3
	}

	var lines []string
	lastSource := toolSource(-1)

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
		lines = append(lines, "\n  "+styleGlobal.Render("nothing here yet ✧ press a to add"))
	}

	for i := start; i < end; i++ {
		e := m.entries[i]

		if e.source != lastSource {
			lastSource = e.source
			var hdr, rule string
			if e.source == sourceLocal {
				hdr = styleSectionLocal.Render("  local ")
				rule = styleSubtitle.Render(strings.Repeat("╌", width-11))
			} else {
				hdr = styleSectionGlobal.Render("  global ")
				rule = styleSubtitle.Render(strings.Repeat("╌", width-12))
			}
			lines = append(lines, hdr+rule)
		}

		selected := i == m.cursor
		cur := "   "
		if selected {
			cur = styleCursor.Render(" ❯ ")
		}

		nameStyle := lipgloss.NewStyle().Width(18).Foreground(colText)
		if e.source == sourceGlobal {
			nameStyle = nameStyle.Foreground(colOverlay)
		}
		if selected {
			nameStyle = nameStyle.Foreground(colPink).Bold(true)
		}

		spin := ""
		if m.installingTools[e.name] {
			spin = "  " + styleInstalling.Render(spinFrames[m.spinFrame]+" installing")
		}

		row := cur + nameStyle.Render(e.name) + " " + styleVersion.Render(e.version) + spin
		if selected {
			row = lipgloss.NewStyle().Background(colSurface).Width(width - 2).Render(row)
		}
		lines = append(lines, row)
	}

	return panelBorder("✦ tools", strings.Join(lines, "\n"), width)
}

func (m model) renderDetailPanel(width int) string {
	if len(m.entries) == 0 || m.cursor >= len(m.entries) {
		return panelBorder("detail", "\n  "+styleGlobal.Render("select a tool ✧"), width)
	}

	e := m.entries[m.cursor]

	scopePill := stylePillLocal.Render("local")
	if e.source == sourceGlobal {
		scopePill = stylePillGlobal.Render("global")
	}

	installing := ""
	if m.installingTools[e.name] {
		installing = "\n\n  " + styleInstalling.Render(spinFrames[m.spinFrame]+"  running mise install…")
	}

	content := fmt.Sprintf(
		"\n  %s\n  %s\n\n  %s\n  %s\n\n  %s  %s\n%s",
		styleLabel.Render("name"),
		styleTitle.Render(e.name),
		styleLabel.Render("version"),
		styleVersion.Render(e.version),
		styleLabel.Render("scope"),
		scopePill,
		installing,
	)
	return panelBorder("detail", content, width)
}

func (m model) renderStatus(width int) string {
	if m.statusMsg == "" {
		return styleSubtitle.Render("  ✧ ready")
	}
	if m.statusIsOK {
		return "  " + styleStatusOK.Render(m.statusMsg)
	}
	return "  " + styleStatusErr.Render(m.statusMsg)
}

func (m model) renderFooter(_ int) string {
	return footerBar([]struct{ key, desc string }{
		{"↑↓/kj", "move"},
		{"Enter", "versions"},
		{"e", "edit"},
		{"a", "add"},
		{"d", "delete"},
		{"x", "install all"},
		{"q", "quit"},
	})
}

// ── Input screen ──────────────────────────────────────────────────────────────

func (m model) renderInputScreen(title, label, hint, extra string) string {
	caret := styleCursor.Render("▌")
	content := "\n"
	if extra != "" {
		content += "  " + extra + "\n\n"
	}
	content += "  " + styleLabel.Render(label+" ❯") + "\n"
	content += "    " + styleInput.Render(m.inputBuffer) + caret + "\n\n"
	content += "  " + styleGlobal.Render(hint) + "\n\n"
	content += "  " + keyBind("Enter", "confirm") + "   " + keyBind("Esc", "back")

	box := panelBorder(title, content, 62)
	topPad := (m.height - strings.Count(box, "\n") - 4) / 2
	if topPad < 0 {
		topPad = 0
	}
	return strings.Repeat("\n", topPad) + box
}

// ── Tool browser ──────────────────────────────────────────────────────────────

func (m model) renderToolBrowser() string {
	totalW := m.width
	if totalW < 40 {
		totalW = 40
	}
	listW := totalW * 52 / 100
	detailW := totalW - listW - 1

	visibleRows := m.height - 10
	if visibleRows < 3 {
		visibleRows = 3
	}

	// Left: list
	var ll []string

	countStr := ""
	if !m.loadingRegistry && len(m.registryTools) > 0 {
		countStr = "  " + styleSubtitle.Render(fmt.Sprintf("%d / %d", len(m.filteredTools), len(m.registryTools)))
	}
	ll = append(ll, " "+styleLabel.Render("🔍 ")+styleInput.Render(m.searchBuffer)+styleCursor.Render("▌")+countStr)
	ll = append(ll, "  "+styleSubtitle.Render(strings.Repeat("╌", listW-6)))

	if m.loadingRegistry {
		ll = append(ll, "\n  "+styleInstalling.Render(spinFrames[m.spinFrame]+"  fetching registry…"))
	} else if len(m.filteredTools) == 0 {
		ll = append(ll, "\n  "+styleGlobal.Render("nothing matches ✧"))
	} else {
		start := m.browserCursor - visibleRows/2
		if start < 0 {
			start = 0
		}
		end := start + visibleRows
		if end > len(m.filteredTools) {
			end = len(m.filteredTools)
			start = end - visibleRows
			if start < 0 {
				start = 0
			}
		}
		for i := start; i < end; i++ {
			t := m.filteredTools[i]
			selected := i == m.browserCursor

			cur := "   "
			if selected {
				cur = styleCursor.Render(" ❯ ")
			}

			nameStyle := lipgloss.NewStyle().Width(18).Foreground(colText)
			if selected {
				nameStyle = nameStyle.Foreground(colPink).Bold(true)
			}

			shortDesc := t.desc
			maxInline := listW - 18 - 10
			if maxInline > 3 && len(shortDesc) > maxInline {
				shortDesc = shortDesc[:maxInline-1] + "…"
			}

			row := cur + nameStyle.Render(t.name) + styleGlobal.Render(shortDesc)
			if selected {
				row = lipgloss.NewStyle().Background(colSurface).Width(listW - 2).Render(row)
			}
			ll = append(ll, row)
		}
		ll = append(ll, "")
		ll = append(ll, "  "+styleSubtitle.Render(fmt.Sprintf("%d of %d", m.browserCursor+1, len(m.filteredTools))))
	}

	leftPanel := panelBorder("🫧 add tool", strings.Join(ll, "\n"), listW)

	// Right: detail
	var detail string
	if m.loadingRegistry {
		detail = "\n  " + styleGlobal.Render("loading…")
	} else if len(m.filteredTools) == 0 {
		detail = "\n  " + styleGlobal.Render("no tool selected ✧")
	} else {
		t := m.filteredTools[m.browserCursor]

		badge := ""
		if _, ok := m.localTools[t.name]; ok {
			badge = "\n  " + stylePillInstalled.Render("✓ local")
		} else if _, ok := m.globalTools[t.name]; ok {
			badge = "\n  " + stylePillGlobal.Render("● global")
		}

		desc := styleGlobal.Render("no description available")
		if t.desc != "" {
			desc = styleSubtitle.Render(wrapText(t.desc, detailW-6))
		}

		detail = fmt.Sprintf(
			"\n  %s\n  %s%s\n\n  %s\n  %s\n\n  %s\n  %s\n\n  %s",
			styleLabel.Render("name"),
			styleTitle.Render(t.name),
			badge,
			styleLabel.Render("registry"),
			styleLavender.Render(t.full),
			styleLabel.Render("about"),
			desc,
			styleGlobal.Render("Enter ❯ pick a version"),
		)
	}

	rightPanel := panelBorder("detail", detail, detailW)
	body := lipgloss.JoinHorizontal(lipgloss.Top, leftPanel, " ", rightPanel)
	return body + "\n" + footerBar([]struct{ key, desc string }{
		{"↑↓/kj", "move"},
		{"type", "search"},
		{"Enter", "select"},
		{"Esc", "clear · back"},
	})
}

// ── Version picker ────────────────────────────────────────────────────────────

func (m model) renderVersionPicker() string {
	totalW := m.width
	if totalW < 40 {
		totalW = 40
	}
	listW := totalW * 55 / 100
	infoW := totalW - listW - 1
	visibleRows := m.height - 8
	if visibleRows < 3 {
		visibleRows = 3
	}

	var ll []string
	if m.loadingVersions {
		ll = append(ll, "\n  "+styleInstalling.Render(spinFrames[m.spinFrame]+"  fetching versions…"))
	} else if len(m.versionList) == 0 {
		ll = append(ll, "\n  "+styleGlobal.Render("no versions found ✧"))
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
				row := styleCursor.Render(" ❯ ") +
					lipgloss.NewStyle().Background(colSurface).Foreground(colPink).Bold(true).Width(listW - 6).Render(v)
				ll = append(ll, row)
			} else {
				ll = append(ll, "   "+styleVersion.Render(v))
			}
		}
		ll = append(ll, "")
		ll = append(ll, "  "+styleSubtitle.Render(fmt.Sprintf("%d of %d", m.versionCursor+1, len(m.versionList))))
	}

	leftPanel := panelBorder("🏷️  versions · "+m.pendingTool, strings.Join(ll, "\n"), listW)

	info := "\n  " + styleLabel.Render("tool") + "\n  " + styleTitle.Render(m.pendingTool) + "\n"
	if !m.loadingVersions && len(m.versionList) > 0 {
		info += "\n  " + styleLabel.Render("selected") + "\n  " + styleVersion.Render(m.versionList[m.versionCursor]) + "\n"
	}
	if _, ok := m.versionCache[m.pendingTool]; ok && !m.loadingVersions {
		info += "\n  " + stylePillCached.Render("cached") + "\n"
	}
	info += "\n  " + styleLabel.Render("keys") + "\n" +
		styleGlobal.Render("  Enter    confirm\n  S+Enter  force\n  Esc / q  back")

	rightPanel := panelBorder("info", info, infoW)
	body := lipgloss.JoinHorizontal(lipgloss.Top, leftPanel, " ", rightPanel)
	return body + "\n" + footerBar([]struct{ key, desc string }{
		{"↑↓/kj", "move"},
		{"Enter", "select"},
		{"S+Enter", "force install"},
		{"Esc/q", "back"},
	})
}

// ── Confirm install ───────────────────────────────────────────────────────────

func (m model) renderConfirm() string {
	content := fmt.Sprintf(
		"\n  %s\n  %s\n\n  %s\n  %s\n\n  %s\n  %s\n\n  %s\n  %s   %s\n",
		styleLabel.Render("tool"),
		styleTitle.Render(m.pendingTool),
		styleLabel.Render("version"),
		styleVersion.Render(m.pendingVersion),
		styleLabel.Render("what happens"),
		styleGlobal.Render("saved to TOML · installed in background"),
		styleLabel.Render("confirm?"),
		styleConfirmY.Render("y  yes!"),
		styleConfirmN.Render("n  cancel"),
	)
	box := panelBorder("🫧 confirm install", content, 52)
	topPad := (m.height - strings.Count(box, "\n") - 4) / 2
	if topPad < 0 {
		topPad = 0
	}
	return strings.Repeat("\n", topPad) + box
}

// ── Confirm delete ────────────────────────────────────────────────────────────

func (m model) renderConfirmDelete() string {
	content := fmt.Sprintf(
		"\n  %s\n  %s\n\n  %s\n  %s\n\n  %s\n  %s   %s\n",
		styleLabel.Render("tool"),
		styleDanger.Render("⚠  "+m.pendingTool),
		styleLabel.Render("what happens"),
		styleGlobal.Render("removed from local mise.toml"),
		styleLabel.Render("sure?"),
		styleConfirmY.Render("y  delete"),
		styleConfirmN.Render("n  keep it"),
	)
	box := panelBorder("🗑  confirm delete", content, 52)
	topPad := (m.height - strings.Count(box, "\n") - 4) / 2
	if topPad < 0 {
		topPad = 0
	}
	return strings.Repeat("\n", topPad) + box
}

// ── wrapText ──────────────────────────────────────────────────────────────────

func wrapText(s string, width int) string {
	if width <= 0 {
		return s
	}
	words := strings.Fields(s)
	var lines []string
	cur := ""
	for _, w := range words {
		if cur == "" {
			cur = w
		} else if len(cur)+1+len(w) <= width {
			cur += " " + w
		} else {
			lines = append(lines, cur)
			cur = w
		}
	}
	if cur != "" {
		lines = append(lines, cur)
	}
	return strings.Join(lines, "\n  ")
}
