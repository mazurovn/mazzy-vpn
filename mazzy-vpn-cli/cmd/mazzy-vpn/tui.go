// SPDX-License-Identifier: PolyForm-Noncommercial-1.0.0
// Copyright © 2026 Nik m (@mazurovn). All rights reserved.

package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/mazurovn/mazzy-vpn/core/livecheck"
	"github.com/mazurovn/mazzy-vpn/core/measure"
	"github.com/mazurovn/mazzy-vpn/core/netadapter"
	"github.com/mazurovn/mazzy-vpn/core/runstatus"
	"github.com/mazurovn/mazzy-vpn/core/settings"
)

// ---- styles (ADR-0006 D6: ASCII status glyphs, color-optional) ----

var (
	stProtected = lipgloss.NewStyle().Foreground(lipgloss.Color("42")).Bold(true)
	stWarn      = lipgloss.NewStyle().Foreground(lipgloss.Color("214")).Bold(true)
	stDown      = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	stBox       = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Padding(0, 1)
	stKey       = lipgloss.NewStyle().Foreground(lipgloss.Color("81")).Bold(true)
	stDim       = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	stLogTitle  = lipgloss.NewStyle().Foreground(lipgloss.Color("245")).Bold(true)
)

// screen identifies the active view.
type screen int

const (
	scrMain screen = iota
	scrZones
	scrProfiles
	scrImport
	scrRemoveConfirm
	scrDiagnostics
	scrSettings
	scrLanguage
	scrLog
	scrHelp
)

// tickMsg drives the live header refresh.
type tickMsg time.Time

// statusMsg carries a fresh status snapshot.
type statusMsg struct {
	iface  string
	snap   livecheck.Snapshot
	uplink string
}

// zonesMsg carries ranked zones for the picker.
type zonesMsg struct {
	results []measure.Result
	err     string
}

// logMsg appends a line to the activity log.
type logMsg string

// tuiModel is the bubbletea Model (ADR-0006).
type tuiModel struct {
	scr      screen
	width    int
	height   int
	set      settings.Settings
	setStore *settings.Store

	// live status
	iface   string
	snap    livecheck.Snapshot
	uplink  string
	loading bool

	// activity log (ring buffer, D3)
	logs    []string
	maxLogs int

	// zones/profile overlays
	zones         []measure.Result
	zonesErr      string
	profiles      []profileRow
	cursor        int
	input         string
	pendingDelete string

	// graphWindow selects 1m/5m/20m/session and graphCursor walks historical
	// samples (0 = newest), making the dashboard graph genuinely interactive.
	graphWindow int
	graphCursor int

	// spin advances every tick so the "measuring…" overlay visibly animates
	// instead of looking frozen while a large catalog is probed.
	spin     int
	quitting bool
}

func newTUIModel() tuiModel {
	store := settings.NewStore()
	return tuiModel{
		scr: scrMain, set: store.Load(), setStore: store,
		maxLogs: 200,
	}
}

func (m tuiModel) Init() tea.Cmd {
	return tea.Batch(tickCmd(), refreshStatusCmd())
}

// tickCmd schedules the next header refresh.
func tickCmd() tea.Cmd {
	return tea.Tick(2*time.Second, func(t time.Time) tea.Msg { return tickMsg(t) })
}

// fastTickCmd is a quicker cadence used while a probe wave is in flight so the
// "measuring…" spinner animates smoothly instead of updating every 2s.
func fastTickCmd() tea.Cmd {
	return tea.Tick(150*time.Millisecond, func(t time.Time) tea.Msg { return tickMsg(t) })
}

// refreshStatusCmd samples the live status without blocking input.
func refreshStatusCmd() tea.Cmd {
	return func() tea.Msg {
		iface := detectLiveInterface()
		var snap livecheck.Snapshot
		if iface != "" {
			ctx, cancel := context.WithTimeout(context.Background(), 6*time.Second)
			snap = livecheck.New().Check(ctx, iface)
			cancel()
		}
		uplink := ""
		if adapters, err := netadapter.List(); err == nil {
			if rec, _, ok := netadapter.Recommend(adapters); ok {
				uplink = rec.Name + " (" + rec.Kind() + ")"
			}
		}
		return statusMsg{iface: iface, snap: snap, uplink: uplink}
	}
}

// rankZonesCmd ranks zones asynchronously under a bounded deadline so the picker
// can never hang on a large/dead catalog (D5). The bound scales with the number
// of targets; progress is surfaced via the overlay's "Measuring…" state.
func rankZonesCmd() tea.Cmd {
	return func() tea.Msg {
		cat := newCatalog()
		targets, err := targetsFromCatalog(cat)
		if err != nil || len(targets) == 0 {
			return zonesMsg{err: "no profiles with endpoints; import first"}
		}
		ctx, cancel := context.WithTimeout(context.Background(), rankBudget(len(targets)))
		defer cancel()
		return zonesMsg{results: newMeasurer().RankBest(ctx, targets)}
	}
}

func (m *tuiModel) appendLog(line string) {
	stamp := time.Now().Format("15:04:05")
	m.logs = append(m.logs, stamp+"  "+line)
	if len(m.logs) > m.maxLogs {
		m.logs = m.logs[len(m.logs)-m.maxLogs:]
	}
}

func (m tuiModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		return m, nil

	case tickMsg:
		m.spin++
		// Animate faster while the zones overlay is probing so the spinner is
		// smooth; otherwise the slower status cadence is enough.
		if m.loading {
			return m, tea.Batch(fastTickCmd(), refreshStatusCmd())
		}
		return m, tea.Batch(tickCmd(), refreshStatusCmd())

	case statusMsg:
		m.iface, m.snap, m.uplink = msg.iface, msg.snap, msg.uplink
		return m, nil

	case zonesMsg:
		m.loading = false
		m.zones, m.zonesErr = msg.results, msg.err
		if msg.err != "" {
			m.appendLog("zones: " + msg.err)
		} else {
			m.appendLog(fmt.Sprintf("ranked %d zones", len(msg.results)))
		}
		return m, nil

	case logMsg:
		m.appendLog(string(msg))
		return m, nil

	case tuiActionDoneMsg:
		if msg.err != nil {
			m.appendLog(msg.label + " failed: " + msg.err.Error())
		} else {
			m.appendLog(msg.label + ": done")
		}
		m.set = m.setStore.Load()
		m.profiles = loadProfileRows()
		// Refresh status right away so the header reflects the new state.
		return m, refreshStatusCmd()

	case quitAfterStopMsg:
		m.quitting = true
		return m, tea.Quit

	case tea.KeyMsg:
		return m.handleKey(msg)
	}
	return m, nil
}

func (m tuiModel) handleKey(k tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Overlay input contexts (D4).
	switch m.scr {
	case scrZones:
		return m.keyZones(k)
	case scrProfiles:
		return m.keyProfiles(k)
	case scrImport:
		return m.keyImport(k)
	case scrRemoveConfirm:
		return m.keyRemoveConfirm(k)
	case scrDiagnostics:
		return m.keyDiagnostics(k)
	case scrSettings:
		return m.keySettings(k)
	case scrLanguage:
		return m.keyLanguage(k)
	case scrLog:
		return m.keyLog(k)
	case scrHelp:
		return m.keyHelp(k)
	}
	// Main screen global hotkeys.
	switch k.String() {
	case "q", "ctrl+c":
		if snap, ok := daemonRunning(); ok && !snap.Background {
			return m, stopSessionAndQuitCmd()
		}
		m.quitting = true
		return m, tea.Quit
	case "c":
		m.appendLog("quick connect requested (best zone)")
		return m, requestConnectCmd("--best")
	case "b":
		m.appendLog("background daemon requested")
		return m, requestBackgroundCmd()
	case "k":
		if _, ok := daemonRunning(); !ok {
			m.appendLog("no background daemon running")
			return m, nil
		}
		m.appendLog("stop background daemon requested")
		return m, requestStopCmd()
	case "l":
		m.scr = scrLog
		return m, nil
	case "z":
		m.scr = scrZones
		m.loading = true
		m.cursor = 0
		return m, rankZonesCmd()
	case "d":
		m.appendLog("disconnect requested")
		return m, requestDisconnectCmd()
	case "p":
		m.scr = scrProfiles
		m.profiles = loadProfileRows()
		m.cursor = 0
		return m, nil
	case "x":
		m.scr = scrDiagnostics
		return m, nil
	case "s":
		m.scr = scrSettings
		m.cursor = 0
		return m, nil
	case "t":
		m.scr = scrZones
		m.loading = true
		m.cursor = 0
		return m, rankZonesCmd()
	case "r":
		m.scr = scrRemoveConfirm
		m.pendingDelete = "__RECOVER__"
		return m, nil
	case "?":
		m.scr = scrHelp
		return m, nil
	case "g":
		m.graphWindow = (m.graphWindow + 1) % len(graphWindows)
		m.graphCursor = 0
		return m, nil
	case "left":
		m.graphCursor++
		return m, nil
	case "right":
		if m.graphCursor > 0 {
			m.graphCursor--
		}
		return m, nil
	}
	return m, nil
}

func (m tuiModel) keyZones(k tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch k.String() {
	case "esc", "q":
		m.scr = scrMain
		return m, nil
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
	case "down", "j":
		if m.cursor < len(m.zones)-1 {
			m.cursor++
		}
	case "enter":
		if m.cursor < len(m.zones) {
			z := m.zones[m.cursor]
			m.appendLog("connect requested: " + z.Name)
			m.scr = scrMain
			return m, requestConnectCmd(z.Name)
		}
	}
	return m, nil
}

func (m tuiModel) keyLog(k tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch k.String() {
	case "esc", "q", "l":
		m.scr = scrMain
	}
	return m, nil
}

func (m tuiModel) keySettings(k tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch k.String() {
	case "esc", "q":
		m.scr = scrMain
		return m, nil
	case "1":
		m.set.AutoConnect = !m.set.AutoConnect
	case "2":
		m.set.AutoDiagnostics = !m.set.AutoDiagnostics
	case "3":
		m.set.Notifications = !m.set.Notifications
	case "4":
		m.set.AutoReconnect = !m.set.AutoReconnect
	case "5":
		m.set.KillSwitch = !m.set.KillSwitch
	case "6":
		m.set.AutoMimic = !m.set.AutoMimic
	case "7":
		m.scr = scrLanguage
		m.cursor = languageCursor()
		return m, nil
	default:
		return m, nil
	}
	if err := m.setStore.Save(m.set); err != nil {
		m.appendLog("could not save settings: " + err.Error())
	}
	return m, nil
}

// tuiActionDoneMsg reports the outcome of a suspended privileged action.
type tuiActionDoneMsg struct {
	label string
	err   error
}

// privilegedTUICmd suspends the alt-screen (tea.ExecProcess), runs the
// privileged mazzy-vpn action so sudo/pkexec can prompt on the real terminal,
// then resumes the UI and reports the outcome. This makes the TUI a real
// control surface instead of a note-writer (fixes P0-1/P0-4).
//
// execProcess is indirected so tests can assert the built command without
// spawning anything.
var execProcess = tea.ExecProcess

func privilegedTUICmd(label, subcmd string, sargs ...string) tea.Cmd {
	c, err := buildPrivilegedCmd(subcmd, sargs...)
	if err != nil {
		return func() tea.Msg { return tuiActionDoneMsg{label: label, err: err} }
	}
	return execProcess(c, func(err error) tea.Msg {
		return tuiActionDoneMsg{label: label, err: err}
	})
}

func requestConnectCmd(zone string) tea.Cmd {
	// Resume/switch an existing daemon without spawning a competing process.
	if _, ok := daemonRunning(); ok {
		intentZone := zone
		if intentZone == "--best" {
			intentZone = ""
		}
		_ = writeDesired(intentZone, "up")
		return func() tea.Msg { return tuiActionDoneMsg{label: "connect " + zone} }
	}
	// Start a detached menu-scoped daemon so Bubble Tea immediately returns to
	// the live dashboard. The old `up` path was foreground and blocked the TUI
	// until disconnect.
	return privilegedTUICmd("connect "+zone, "daemon", zone, "--session")
}

func requestDisconnectCmd() tea.Cmd {
	_ = writeDesired("", "down")
	return privilegedTUICmd("disconnect", "disconnect")
}

// requestBackgroundCmd starts a detached background daemon (best zone) via the
// elevation path, then returns to the dashboard. The daemon self-detaches so it
// survives the TUI exiting or the terminal closing.
func requestBackgroundCmd() tea.Cmd {
	return privilegedTUICmd("background", "daemon", "--best", "--background")
}

// requestStopCmd stops a running daemon via the elevated `stop` subcommand, so a
// root-owned daemon is actually terminated instead of failing with EPERM.
func requestStopCmd() tea.Cmd {
	return privilegedTUICmd("stop", "stop")
}

func requestRecoverCmd() tea.Cmd {
	_ = writeDesired("", "down")
	return privilegedTUICmd("recover", "recover")
}

func (m tuiModel) View() string {
	if m.quitting {
		return "Bye.\n"
	}
	switch m.scr {
	case scrZones:
		return m.viewZones()
	case scrProfiles:
		return m.viewProfiles()
	case scrImport:
		return m.viewImport()
	case scrRemoveConfirm:
		return m.viewRemoveConfirm()
	case scrDiagnostics:
		return m.viewDiagnostics()
	case scrSettings:
		return m.viewSettings()
	case scrLanguage:
		return m.viewLanguage()
	case scrLog:
		return m.viewLog()
	case scrHelp:
		return m.viewHelp()
	default:
		return m.viewMain()
	}
}

func (m tuiModel) header() string {
	// If a background daemon is publishing a heartbeat, render the rich dashboard
	// (latency graph, error-rate, recent errors) instead of the one-line status.
	if h, ok := m.dashboardHeader(); ok {
		return h
	}
	var status string
	switch {
	case m.snap.Protected():
		status = stProtected.Render("● PROTECTED") +
			fmt.Sprintf("  %s · %s", safeDisplay(m.iface), safeDisplay(m.snap.EgressIP))
	case m.iface != "":
		status = stWarn.Render("▲ LINK UP") + "  " + safeDisplay(m.iface) + " (no egress)"
	default:
		status = stDown.Render("● DISCONNECTED") + "  plain uplink"
	}
	up := m.uplink
	if up == "" {
		up = "?"
	}
	title := "Mazzy VPN" + strings.Repeat(" ", max(1, 30-len("Mazzy VPN"))) + stDim.Render(up)
	return stBox.Render(title + "\n" + status)
}

// dashboardHeader renders the rich header from a background daemon's heartbeat.
// ok is false when no fresh heartbeat exists, so the caller falls back.
func (m tuiModel) dashboardHeader() (string, bool) {
	snap, ok := daemonRunning()
	if !ok {
		return "", false
	}
	t := translator()
	var status string
	switch snap.State {
	case runstatus.StateProtected:
		status = stProtected.Render("● PROTECTED") + "  " +
			safeDisplay(snap.Interface) + " · " + safeDisplay(snap.Egress)
	case runstatus.StateReconnect:
		status = stWarn.Render("⟳ RECONNECTING") + "  " + safeDisplay(snap.Zone)
	case runstatus.StateConnecting:
		status = stWarn.Render("… CONNECTING") + "  " + safeDisplay(snap.Zone)
	case runstatus.StateLinkUp:
		status = stWarn.Render("▲ LINK UP") + "  " + safeDisplay(snap.Interface) + " (no egress)"
	case runstatus.StatePaused:
		// A paused daemon is alive and deliberately down; without this case it
		// rendered as DISCONNECTED and looked like a dead daemon.
		status = stWarn.Render("⏸ PAUSED") + "  " + safeDisplay(snap.Zone) + stDim.Render("  (press c to resume)")
	default:
		status = stDown.Render("● DISCONNECTED")
	}
	// Surface a wedged/busy writer instead of silently rendering old numbers.
	if age := snap.HeartbeatAge(); age > 15*time.Second {
		status += stWarn.Render(fmt.Sprintf("  ⚠ status %s old", shortDur(age)))
	}
	mode := "fg"
	if snap.Background {
		mode = "bg"
	}
	up := ""
	if snap.StartedAt > 0 {
		up = shortDur(time.Since(time.Unix(snap.StartedAt, 0)))
	}
	proto := ""
	if snap.Protocol != "" {
		proto = " · " + safeDisplay(snap.Protocol)
	}
	title := "Mazzy VPN" + strings.Repeat(" ", 18) +
		stDim.Render(fmt.Sprintf("zone %s%s · %s %s · %s", trunc(safeDisplay(snap.Zone), 16), proto, t.T("cli.dash.uptime"), up, mode))

	series, windowLabel, selected := m.graphSeries(snap)
	spark := runstatus.Sparkline(series, 44)
	mn, avg, mx := runstatus.LatencyStats(series)
	graph := stDim.Render(t.T("cli.dash.graph")+" ["+windowLabel+"] ") + spark +
		stDim.Render(fmt.Sprintf("  %d/%d/%d ms%s", mn, avg, mx, selected))

	rate := snap.ErrorRatePerMin(10 * time.Minute)
	errLine := stDim.Render(fmt.Sprintf("%s %d · %.1f %s · reconnects %d",
		t.T("cli.dash.errors"), len(snap.Errors), rate, t.T("cli.dash.errrate"), snap.Reconnects))
	// The newest error inline: counts alone said "something is wrong" while
	// hiding WHAT — the user had to open the log to learn the reason.
	if recent := snap.RecentErrors(1); len(recent) > 0 {
		errLine += "\n" + stDim.Render("last: "+trunc(time.Unix(recent[0].TS, 0).Format("15:04:05")+" "+safeDisplay(recent[0].Reason), 72))
	}

	body := title + "\n" + status + "\n" + graph + "\n" + errLine
	return stBox.Render(body), true
}

func (m tuiModel) actionBar() string {
	line1 := fmt.Sprintf("%s Connect   %s Zones   %s Profiles   %s Diagnostics   %s Disconnect   %s Recover",
		stKey.Render("[c]"), stKey.Render("[z]"), stKey.Render("[p]"), stKey.Render("[x]"), stKey.Render("[d]"), stKey.Render("[r]"))
	line2 := fmt.Sprintf("%s Graph window   %s Log   %s Stop bg   %s Settings   %s Help   %s Quit",
		stKey.Render("[g]"), stKey.Render("[l]"), stKey.Render("[k]"), stKey.Render("[s]"), stKey.Render("[?]"), stKey.Render("[q]"))
	return line1 + "\n" + line2
}

// viewLog shows the persisted daemon activity log (background runs) plus the
// in-session ring, so the log is one keypress away and dismissible with esc.
func (m tuiModel) viewLog() string {
	var b strings.Builder
	b.WriteString(m.header() + "\n\n")
	b.WriteString(stLogTitle.Render("  Activity log (esc/l back)") + "\n")
	lines := tailLog(60)
	if len(lines) == 0 && len(m.logs) == 0 {
		b.WriteString(stDim.Render("  (no activity yet)") + "\n")
		return b.String()
	}
	for _, ln := range lines {
		b.WriteString("  " + safeDisplay(ln) + "\n")
	}
	for _, ln := range m.logs {
		b.WriteString("  " + ln + "\n")
	}
	return b.String()
}

func (m tuiModel) logPane() string {
	title := stLogTitle.Render("Activity")
	n := 8
	// Preserve the five-row status pane on common 80x24 terminals by shrinking
	// secondary activity history before clipping operational state.
	if m.height > 0 && m.height < 30 {
		n = 3
	}
	if m.height > 0 && m.height < 24 {
		n = 1
	}
	start := 0
	if len(m.logs) > n {
		start = len(m.logs) - n
	}
	body := strings.Join(m.logs[start:], "\n")
	if body == "" {
		body = stDim.Render("(no activity yet)")
	}
	return stBox.Render(title + "\n" + body)
}

func (m tuiModel) viewMain() string {
	return m.header() + "\n" + m.actionBar() + "\n\n" + m.statusPane() + "\n" + m.logPane() + "\n"
}

// spinFrames animates the "measuring" indicator so a long probe wave never
// looks frozen.
var spinFrames = []string{"⠹", "⠸", "⠴", "⠦", "⠧", "⠇", "⠏"}

func (m tuiModel) viewZones() string {
	if m.loading {
		frame := spinFrames[m.spin%len(spinFrames)]
		return m.header() + "\n\n  " + stKey.Render(frame) +
			" Measuring servers via the physical uplink…\n" +
			stDim.Render("  (ranking live servers; this is bounded — press esc to cancel)") + "\n"
	}
	if m.zonesErr != "" {
		return m.header() + "\n\n  " + m.zonesErr + "\n\n  [esc] back\n"
	}
	var b strings.Builder
	b.WriteString(m.header() + "\n\n")
	b.WriteString(stLogTitle.Render("  Zones (↑/↓ select, Enter connect, esc back)") + "\n")
	alive := 0
	for i, z := range m.zones {
		cur := "  "
		if i == m.cursor {
			cur = stKey.Render("> ")
		}
		st := stDown.Render("✖ down")
		lat := "-"
		bar := ""
		if z.ICMPAlive {
			st = stProtected.Render("● alive")
			lat = fmt.Sprintf("%d ms", z.LatencyMS)
			bar = latencyBar(z.LatencyMS)
			alive++
		} else if z.Reachable {
			st = stWarn.Render("▲ no icmp")
		}
		b.WriteString(fmt.Sprintf("%s%-26s %-9s %-10s %s\n", cur, safeDisplay(z.Name), lat, st, bar))
	}
	b.WriteString(stDim.Render(fmt.Sprintf("\n  %d/%d alive", alive, len(m.zones))) + "\n")
	return b.String()
}

// latencyBar renders a compact quality bar from a latency in ms: greener/fuller
// is faster. Purely visual, so the user can eyeball the best server at a glance.
func latencyBar(ms int64) string {
	switch {
	case ms <= 0:
		return ""
	case ms < 50:
		return stProtected.Render("█████ excellent")
	case ms < 100:
		return stProtected.Render("████░ great")
	case ms < 160:
		return stWarn.Render("███░░ good")
	case ms < 250:
		return stWarn.Render("██░░░ fair")
	default:
		return stDown.Render("█░░░░ slow")
	}
}

func (m tuiModel) viewSettings() string {
	onoff := func(b bool) string {
		if b {
			return stProtected.Render("✔ on")
		}
		return stDown.Render("✖ off")
	}
	var b strings.Builder
	b.WriteString(m.header() + "\n\n")
	b.WriteString(stLogTitle.Render("  Settings (press number to toggle, esc back)") + "\n")
	b.WriteString(fmt.Sprintf("  %s Auto-connect on start   %s\n", stKey.Render("[1]"), onoff(m.set.AutoConnect)))
	b.WriteString(fmt.Sprintf("  %s Auto-diagnostics        %s\n", stKey.Render("[2]"), onoff(m.set.AutoDiagnostics)))
	b.WriteString(fmt.Sprintf("  %s Notifications           %s\n", stKey.Render("[3]"), onoff(m.set.Notifications)))
	b.WriteString(fmt.Sprintf("  %s Auto-reconnect          %s\n", stKey.Render("[4]"), onoff(m.set.AutoReconnect)))
	b.WriteString(fmt.Sprintf("  %s Kill-switch             %s\n", stKey.Render("[5]"), onoff(m.set.KillSwitch)))
	b.WriteString(fmt.Sprintf("  %s Auto-mimic timezone     %s\n", stKey.Render("[6]"), onoff(m.set.AutoMimic)))
	b.WriteString(fmt.Sprintf("  %s Language               %s\n", stKey.Render("[7]"), safeDisplay(string(resolveLang()))))
	return b.String()
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// runTUI starts the full-screen bubbletea interface. Falls back handled by
// caller when not a TTY.
func runTUI() int {
	p := tea.NewProgram(newTUIModel(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Println("tui error:", err)
		return 1
	}
	return 0
}
