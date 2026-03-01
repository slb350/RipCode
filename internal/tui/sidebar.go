package tui

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/stephenbrandon/ripcode/internal/agent"
	"github.com/stephenbrandon/ripcode/internal/tui/components"
	"github.com/stephenbrandon/ripcode/internal/tui/styles"
)

// sidebarSection keys for collapse state.
const (
	sectionContext  = "context"
	sectionMCP      = "mcp"
	sectionLSP      = "lsp"
	sectionTodo     = "todo"
	sectionModified = "modified"
	sectionTools    = "tools"
)

// sidebarSectionKeys maps 1-indexed number keys to section names.
var sidebarSectionKeys = []string{
	sectionContext,
	sectionMCP,
	sectionLSP,
	sectionTodo,
	sectionModified,
	sectionTools,
}

func (a App) handleSidebarOverlayKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch {
	case msg.Code == tea.KeyEscape:
		a.sidebarOverlay = false
		if !a.streaming {
			a.input.Focus()
		}
		return a, nil
	case msg.Mod == tea.ModCtrl && msg.Code == 'b':
		a.sidebarOverlay = false
		if !a.streaming {
			a.input.Focus()
		}
		return a, nil
	case msg.Code >= '1' && msg.Code <= '6' && msg.Mod == 0:
		idx := int(msg.Code - '1')
		if a.uiPrefs != nil && idx < len(sidebarSectionKeys) {
			a.uiPrefs.ToggleCollapsed(sidebarSectionKeys[idx])
			a.warnOnErr(a.uiPrefs.Save(), "preferences")
		}
		return a, nil
	case msg.Code == 'd' && msg.Mod == 0:
		if a.uiPrefs != nil {
			a.uiPrefs.DismissGettingStarted()
			a.warnOnErr(a.uiPrefs.Save(), "preferences")
		}
		return a, nil
	default:
		return a, nil
	}
}

func (a App) toggleSidebar() App {
	if a.width >= 120 {
		a.sidebarHidden = !a.sidebarHidden
		if a.sidebarHidden {
			a.sidebarOverlay = false
		}
		return a
	}

	if a.sidebarOverlayActive() {
		a.sidebarOverlay = false
		if !a.streaming {
			a.input.Focus()
		}
		return a
	}

	a.sidebarHidden = false
	a.sidebarOverlay = true
	a.commandPalette.open = false
	a.inline.open = false
	a.modelDialog.open = false
	a.input.Blur()
	return a
}

func renderSideBySide(main, side string, mainW, sideW int) string {
	mainLines := strings.Split(main, "\n")
	sideLines := strings.Split(side, "\n")
	n := max(len(mainLines), len(sideLines))

	mainCell := lipgloss.NewStyle().Width(mainW).MaxWidth(mainW)
	sideCell := lipgloss.NewStyle().Width(sideW).MaxWidth(sideW)

	var sb strings.Builder
	for i := 0; i < n; i++ {
		m := ""
		if i < len(mainLines) {
			m = mainLines[i]
		}
		s := ""
		if i < len(sideLines) {
			s = sideLines[i]
		}
		sb.WriteString(mainCell.Render(m))
		sb.WriteString(" ")
		sb.WriteString(sideCell.Render(s))
		if i < n-1 {
			sb.WriteByte('\n')
		}
	}
	return sb.String()
}

func (a App) renderSidebarSection(title string, sectionKey string, count int, content []string) []string {
	muted := styles.DefaultTheme.TextMutedStyle
	text := styles.DefaultTheme.TextStyle

	collapsed := a.uiPrefs != nil && a.uiPrefs.IsCollapsed(sectionKey)
	indicator := "▾"
	if collapsed {
		indicator = "▸"
	}

	header := fmt.Sprintf("%s %s (%d)", indicator, title, count)
	lines := []string{text.Render(header)}

	if !collapsed {
		lines = append(lines, content...)
	}
	lines = append(lines, muted.Render(repeatStr("─", 34)))
	lines = append(lines, "")
	return lines
}

func (a App) renderSidebar() string {
	if !a.sidebarVisible() {
		return ""
	}

	muted := styles.DefaultTheme.TextMutedStyle
	text := styles.DefaultTheme.TextStyle
	success := styles.DefaultTheme.SuccessStyle
	errorStyle := styles.DefaultTheme.ErrorStyle
	warning := styles.DefaultTheme.WarningStyle
	modeColor := lipgloss.NewStyle().Foreground(styles.DefaultTheme.ModeColor(a.agent.Name)).Bold(true)

	var lines []string

	// Session header
	headerTitle := "Session"
	if a.session != nil {
		headerTitle = shortSessionTitle(a.session.ID)
	}
	lines = append(lines, text.Render(headerTitle))
	if a.session != nil {
		age := time.Since(a.session.CreatedAt).Round(time.Second)
		if age < 0 {
			age = 0
		}
		lines = append(lines, muted.Render(fmt.Sprintf("started %s ago", age)))
	}
	lines = append(lines, muted.Render(repeatStr("─", 34)))
	lines = append(lines, "")

	// Getting Started card
	if a.showGettingStarted() {
		lines = append(lines, text.Render("Getting Started"))
		lines = append(lines, muted.Render("  Type a message to begin"))
		lines = append(lines, muted.Render("  / for commands"))
		lines = append(lines, muted.Render("  @ for file references"))
		lines = append(lines, muted.Render("  Tab to switch agents"))
		lines = append(lines, muted.Render("  Press d to dismiss"))
		lines = append(lines, muted.Render(repeatStr("─", 34)))
		lines = append(lines, "")
	}

	// Context section
	tokens := 0
	if a.session != nil {
		tokens = a.session.Tokens.Input + a.session.Tokens.Output
	}
	msgCount := 0
	if a.session != nil {
		msgCount = a.session.Len()
	}
	var contextContent []string
	contextContent = append(contextContent, muted.Render(fmt.Sprintf("%s tokens", components.FormatTokens(tokens))))
	const assumedContextLimit = 200_000
	percent := clamp((tokens*100)/assumedContextLimit, 0, 100)
	const barWidth = 20
	filled := clamp((percent*barWidth)/100, 0, barWidth)
	bar := "[" + repeatStr("■", filled) + repeatStr("·", barWidth-filled) + "]"
	contextContent = append(contextContent, muted.Render(bar)+" "+muted.Render(fmt.Sprintf("%d%%", percent)))
	modeName := strings.TrimSpace(a.agent.Name)
	if modeName == "" {
		modeName = agent.NameBuild
	}
	modeLabel := "Build"
	if len(modeName) > 0 {
		modeLabel = strings.ToUpper(modeName[:1]) + modeName[1:]
	}
	contextContent = append(contextContent, modeColor.Render(modeLabel)+" "+muted.Render("agent"))
	if a.model != "" {
		contextContent = append(contextContent, muted.Render(a.model))
	}
	contextContent = append(contextContent, muted.Render(fmt.Sprintf("%d messages", msgCount)))
	lines = append(lines, a.renderSidebarSection("Context", sectionContext, msgCount, contextContent)...)

	// MCP section
	if a.mcpConfig != nil && len(a.mcpConfig.Servers) > 0 {
		var mcpContent []string
		for _, srv := range a.mcpConfig.Servers {
			mcpContent = append(mcpContent, muted.Render(fmt.Sprintf("  %s %s", enabledIcon(srv.Enabled), srv.Name)))
		}
		lines = append(lines, a.renderSidebarSection("MCP", sectionMCP, len(a.mcpConfig.Servers), mcpContent)...)
	}

	// LSP section
	if a.lspConfig != nil && len(a.lspConfig.Clients) > 0 {
		var lspContent []string
		for _, cl := range a.lspConfig.Clients {
			lspContent = append(lspContent, muted.Render(fmt.Sprintf("  %s %s", enabledIcon(cl.Enabled), cl.Name)))
		}
		lines = append(lines, a.renderSidebarSection("LSP", sectionLSP, len(a.lspConfig.Clients), lspContent)...)
	}

	// Todo section
	if a.todoTool != nil {
		items := a.todoTool.Items()
		if len(items) > 0 {
			var todoContent []string
			for _, item := range items {
				marker := "[ ]"
				switch item.Status {
				case "completed":
					marker = "[x]"
				case "in_progress":
					marker = "[~]"
				}
				todoContent = append(todoContent, muted.Render(fmt.Sprintf("  %s %s", marker, item.Subject)))
			}
			lines = append(lines, a.renderSidebarSection("Todo", sectionTodo, len(items), todoContent)...)
		}
	}

	// Modified files section
	if len(a.modifiedFiles) > 0 {
		var modContent []string
		for _, f := range a.modifiedFiles {
			modContent = append(modContent, muted.Render("  "+filepath.Base(f)))
		}
		lines = append(lines, a.renderSidebarSection("Modified", sectionModified, len(a.modifiedFiles), modContent)...)
	}

	// Tools section
	events := a.toolpanel.Events()
	successCount := 0
	errorCount := 0
	pendingCount := 0
	for _, ev := range events {
		switch {
		case ev.Error != "":
			errorCount++
		case ev.Output != "":
			successCount++
		default:
			pendingCount++
		}
	}
	var toolsContent []string
	summary := fmt.Sprintf("✓ %d", successCount)
	if errorCount > 0 {
		summary += fmt.Sprintf("  ✗ %d", errorCount)
	}
	if pendingCount > 0 {
		summary += fmt.Sprintf("  ⋯ %d", pendingCount)
	}
	toolsContent = append(toolsContent, muted.Render(summary))
	if len(events) == 0 {
		toolsContent = append(toolsContent, muted.Render("No tool activity yet"))
	} else {
		start := max(0, len(events)-8)
		for i := start; i < len(events); i++ {
			ev := events[i]
			switch {
			case ev.Error != "":
				toolsContent = append(toolsContent, errorStyle.Render("✗ "+ev.Name))
			case ev.Output != "":
				toolsContent = append(toolsContent, success.Render("✓ "+ev.Name))
			default:
				toolsContent = append(toolsContent, warning.Render("⋯ "+ev.Name))
			}
		}
	}
	lines = append(lines, a.renderSidebarSection("Tools", sectionTools, len(events), toolsContent)...)

	return strings.Join(lines, "\n")
}

func (a App) sidebarOverlayPanel() string {
	body := "Sidebar overlay (Esc close)\n\n" + a.renderSidebar()
	panelW := min(a.width-2, a.sidebarWidth()+2)
	if panelW < 30 {
		panelW = max(20, a.width)
	}
	return lipgloss.NewStyle().
		Width(panelW).
		MaxWidth(panelW).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(styles.DefaultTheme.Palette.Border)).
		Padding(0, 1).
		Render(body)
}

func (a App) sidebarOverlayPanelRect() (x, y, w, h int) {
	return a.panelRectFromRendered(a.sidebarOverlayPanel())
}

func (a App) panelRectFromRendered(panel string) (x, y, w, h int) {
	lines := strings.Split(panel, "\n")
	maxW := 0
	for _, ln := range lines {
		maxW = max(maxW, lipgloss.Width(ln))
	}
	h = len(lines)
	w = maxW
	if w < 1 {
		w = 1
	}
	if h < 1 {
		h = 1
	}
	x = max(0, (a.width-w)/2)
	y = max(0, (a.height-h)/2)
	return
}

func (a App) renderSidebarOverlay(main string) string {
	dimmed := lipgloss.NewStyle().Faint(true).Render(main)
	mainLines := strings.Split(dimmed, "\n")
	panel := a.sidebarOverlayPanel()
	panelLines := strings.Split(panel, "\n")
	x, y, _, _ := a.panelRectFromRendered(panel)

	for i, ln := range panelLines {
		row := y + i
		if row < 0 || row >= len(mainLines) {
			continue
		}
		line := strings.Repeat(" ", x) + ln
		lineW := lipgloss.Width(line)
		if lineW < a.width {
			line += strings.Repeat(" ", a.width-lineW)
		}
		mainLines[row] = line
	}
	return strings.Join(mainLines, "\n")
}

func (a App) renderSessionView() string {
	var sb strings.Builder
	sb.WriteString(a.statusbar.View())
	sb.WriteByte('\n')
	toastView := a.toasts.View()
	if toastView != "" {
		sb.WriteString(toastView)
		sb.WriteByte('\n')
	}
	sb.WriteString(a.chat.View())
	sb.WriteByte('\n')
	if a.helpDialog.open {
		sb.WriteString(a.renderHelpDialog())
		sb.WriteByte('\n')
	}
	if a.statusDialog.open {
		sb.WriteString(a.renderStatusDialog())
		sb.WriteByte('\n')
	}
	if a.stashDialog.open {
		sb.WriteString(a.renderStashDialog())
		sb.WriteByte('\n')
	}
	if a.agentDialog.open {
		sb.WriteString(a.renderAgentDialog())
		sb.WriteByte('\n')
	}
	if a.connectDialog.open {
		sb.WriteString(a.renderConnectDialog())
		sb.WriteByte('\n')
	}
	if a.mcpDialog.open {
		sb.WriteString(a.renderMCPDialog())
		sb.WriteByte('\n')
	}
	if a.themesDialog.open {
		sb.WriteString(a.renderThemesDialog())
		sb.WriteByte('\n')
	}
	if a.forkDialog.open {
		sb.WriteString(a.renderForkDialog())
		sb.WriteByte('\n')
	}
	if a.timelineDialog.open {
		sb.WriteString(a.renderTimelineDialog())
		sb.WriteByte('\n')
	}
	if a.sessionsDialog.open {
		sb.WriteString(a.renderSessionsDialog())
		sb.WriteByte('\n')
	}
	if a.renameDialog.open {
		sb.WriteString(a.renderRenameDialog())
		sb.WriteByte('\n')
	}
	if a.exportDialog.open {
		sb.WriteString(a.renderExportDialog())
		sb.WriteByte('\n')
	}
	if a.modelDialog.open {
		sb.WriteString(a.renderModelDialog())
		sb.WriteByte('\n')
	}
	if a.commandPalette.open {
		sb.WriteString(a.renderCommandPalette())
		sb.WriteByte('\n')
	}
	if a.inline.open {
		sb.WriteString(a.renderInlineSuggestions())
		sb.WriteByte('\n')
	}
	sb.WriteString(a.input.View())
	sb.WriteByte('\n')
	sb.WriteString(a.footer.View())
	main := sb.String()
	if a.sidebarOverlayActive() {
		return a.renderSidebarOverlay(main)
	}
	if !a.sidebarWideVisible() {
		return main
	}
	return renderSideBySide(main, a.renderSidebar(), a.mainContentWidth(), a.sidebarWidth())
}
