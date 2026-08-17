// SPDX-License-Identifier: AGPL-3.0-or-later
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
	scrSettings
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

	// zones overlay
	zones    []measure.Result
	zonesErr string
	cursor   int

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

// rankZonesCmd ranks zones asynchronously (D5: bounded by measure timeouts).
func rankZonesCmd() tea.Cmd {
	return func() tea.Msg {
		cat := newCatalog()
		targets, err := targetsFromCatalog(cat)
		if err != nil || len(targets) == 0 {
			return zonesMsg{err: "no profiles with endpoints; import first"}
		}
		ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
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
	case scrSettings:
		return m.keySettings(k)
	}
	// Main screen global hotkeys.
	switch k.String() {
	case "q", "ctrl+c":
		m.quitting = true
		return m, tea.Quit
	case "c":
		m.appendLog("quick connect requested (best zone)")
		return m, requestConnectCmd("--best")
	case "z":
		m.scr = scrZones
		m.loading = true
		m.cursor = 0
		return m, rankZonesCmd()
	case "d":
		m.appendLog("disconnect requested")
		return m, requestDisconnectCmd()
	case "s":
		m.scr = scrSettings
		m.cursor = 0
		return m, nil
	case "t":
		m.loading = true
		return m, rankZonesCmd()
	case "r":
		m.appendLog("recover requested (needs: sudo mazzy-vpn recover)")
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
	}
	_ = m.setStore.Save(m.set)
	return m, nil
}

// requestConnectCmd/requestDisconnectCmd write desired intent for the daemon
// (ADR-0006 D2: TUI is unprivileged; the daemon applies changes).
func requestConnectCmd(zone string) tea.Cmd {
	return func() tea.Msg {
		if err := writeDesired(zone, "up"); err != nil {
			return logMsg("could not write intent: " + err.Error())
		}
		if zone == "--best" {
			return logMsg("intent saved: connect best zone (daemon will apply; or run: sudo mazzy-vpn up --best)")
		}
		return logMsg("intent saved: connect " + zone + " (or run: sudo mazzy-vpn up " + zone + ")")
	}
}

func requestDisconnectCmd() tea.Cmd {
	return func() tea.Msg {
		if err := writeDesired("", "down"); err != nil {
			return logMsg("could not write intent: " + err.Error())
		}
		return logMsg("intent saved: disconnect (or run: sudo mazzy-vpn disconnect)")
	}
}

func (m tuiModel) View() string {
	if m.quitting {
		return "Bye.\n"
	}
	switch m.scr {
	case scrZones:
		return m.viewZones()
	case scrSettings:
		return m.viewSettings()
	default:
		return m.viewMain()
	}
}

func (m tuiModel) header() string {
	var status string
	switch {
	case m.snap.Protected():
		status = stProtected.Render("● PROTECTED") +
			fmt.Sprintf("  %s · %s", m.iface, m.snap.EgressIP)
	case m.iface != "":
		status = stWarn.Render("▲ LINK UP") + "  " + m.iface + " (no egress)"
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

func (m tuiModel) actionBar() string {
	line1 := fmt.Sprintf("%s Connect best   %s Zones   %s Disconnect   %s Recover",
		stKey.Render("[c]"), stKey.Render("[z]"), stKey.Render("[d]"), stKey.Render("[r]"))
	line2 := fmt.Sprintf("%s Test/rank   %s Settings   %s Quit",
		stKey.Render("[t]"), stKey.Render("[s]"), stKey.Render("[q]"))
	return line1 + "\n" + line2
}

func (m tuiModel) logPane() string {
	title := stLogTitle.Render("Activity")
	n := 8
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
	return m.header() + "\n" + m.actionBar() + "\n\n" + m.logPane() + "\n"
}

func (m tuiModel) viewZones() string {
	if m.loading {
		return m.header() + "\n\n  Measuring servers (ping via uplink)...\n"
	}
	if m.zonesErr != "" {
		return m.header() + "\n\n  " + m.zonesErr + "\n\n  [esc] back\n"
	}
	var b strings.Builder
	b.WriteString(m.header() + "\n\n")
	b.WriteString(stLogTitle.Render("  Zones (↑/↓ select, Enter connect, esc back)") + "\n")
	for i, z := range m.zones {
		cur := "  "
		if i == m.cursor {
			cur = stKey.Render("> ")
		}
		st := stDown.Render("✖ down")
		lat := "-"
		if z.ICMPAlive {
			st = stProtected.Render("● alive")
			lat = fmt.Sprintf("%d ms", z.LatencyMS)
		} else if z.Reachable {
			st = stWarn.Render("▲ no icmp")
		}
		b.WriteString(fmt.Sprintf("%s%-26s %-9s %s\n", cur, z.Name, lat, st))
	}
	return b.String()
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
