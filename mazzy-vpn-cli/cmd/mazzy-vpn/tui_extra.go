// SPDX-License-Identifier: PolyForm-Noncommercial-1.0.0
// Copyright © 2026 Nik m (@mazurovn). All rights reserved.

package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/mazurovn/mazzy-vpn/core/i18n"
	"github.com/mazurovn/mazzy-vpn/core/runstatus"
)

type profileRow struct {
	Name, Protocol, Country string
	Favorite                bool
}

func loadProfileRows() []profileRow {
	entries, err := newCatalog().List()
	if err != nil {
		return nil
	}
	rows := make([]profileRow, 0, len(entries))
	for _, entry := range entries {
		rows = append(rows, profileRow{
			Name: entry.Name, Protocol: string(entry.Protocol), Country: entry.Country,
			Favorite: entry.Favorite,
		})
	}
	return rows
}

// selfTUICmd runs a non-privileged CLI command while Bubble Tea temporarily
// leaves the alt-screen. Reusing the CLI handler keeps menu and terminal
// behavior identical instead of duplicating import/verify/diagnostic logic.
func selfTUICmd(label string, args ...string) tea.Cmd {
	exe, err := os.Executable()
	if err != nil {
		return func() tea.Msg { return tuiActionDoneMsg{label: label, err: err} }
	}
	cmd := exec.Command(exe, args...)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	return execProcess(cmd, func(err error) tea.Msg {
		return tuiActionDoneMsg{label: label, err: err}
	})
}

type quitAfterStopMsg struct{}

func stopSessionAndQuitCmd() tea.Cmd {
	cmd, err := buildPrivilegedCmd("stop")
	if err != nil {
		return func() tea.Msg { return quitAfterStopMsg{} }
	}
	return execProcess(cmd, func(error) tea.Msg { return quitAfterStopMsg{} })
}

func removeProfileCmd(name string) tea.Cmd {
	return func() tea.Msg {
		err := newCatalog().Remove(name)
		return tuiActionDoneMsg{label: "remove " + name, err: err}
	}
}

func (m tuiModel) keyProfiles(k tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch k.String() {
	case "esc", "q":
		m.scr = scrMain
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
	case "down", "j":
		if m.cursor < len(m.profiles)-1 {
			m.cursor++
		}
	case "i":
		m.scr, m.input = scrImport, ""
	case "v":
		return m, selfTUICmd("verify profiles", "verify")
	case "d", "delete", "backspace":
		if len(m.profiles) > 0 && m.cursor < len(m.profiles) {
			m.pendingDelete = m.profiles[m.cursor].Name
			m.scr = scrRemoveConfirm
		}
	case "f":
		if len(m.profiles) > 0 && m.cursor < len(m.profiles) {
			row := m.profiles[m.cursor]
			args := []string{"favorite", row.Name}
			if row.Favorite {
				args = append(args, "--off")
			}
			return m, selfTUICmd("favorite "+row.Name, args...)
		}
	case "enter":
		if len(m.profiles) > 0 && m.cursor < len(m.profiles) {
			row := m.profiles[m.cursor]
			// The embedded engine cannot connect OpenVPN: forwarding the zone to
			// the daemon would only spin it through connect-fail/backoff cycles.
			if strings.EqualFold(row.Protocol, "openvpn") {
				m.appendLog(row.Name + ": OpenVPN is not connectable by the embedded engine — pick a WireGuard/AmneziaWG zone")
				return m, nil
			}
			return m, requestConnectCmd(row.Name)
		}
	}
	return m, nil
}

func (m tuiModel) keyImport(k tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch k.String() {
	case "esc":
		m.scr = scrProfiles
		return m, nil
	case "enter":
		path := strings.TrimSpace(m.input)
		if path == "" {
			return m, nil
		}
		m.scr = scrProfiles
		return m, selfTUICmd("import "+path, "import", path)
	case "backspace":
		runes := []rune(m.input)
		if len(runes) > 0 {
			m.input = string(runes[:len(runes)-1])
		}
		return m, nil
	case "ctrl+u":
		m.input = ""
		return m, nil
	}
	if len(k.Runes) > 0 {
		m.input += string(k.Runes)
	}
	return m, nil
}

func (m tuiModel) keyRemoveConfirm(k tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch strings.ToLower(k.String()) {
	case "y":
		name := m.pendingDelete
		m.pendingDelete = ""
		m.scr = scrProfiles
		switch name {
		case "__RECOVER__":
			m.scr = scrMain
			m.appendLog("recover requested")
			return m, requestRecoverCmd()
		case "__DISARM__":
			m.scr = scrMain
			m.appendLog("HARD RESET (disarm) requested")
			return m, privilegedTUICmd("disarm", "disarm")
		case "__PROBE__":
			m.scr = scrMain
			m.appendLog("DEEP PROBE requested — stopping daemon, testing every zone")
			return m, privilegedTUICmd("probe", "probe", "--all")
		}
		return m, removeProfileCmd(name)
	case "n", "esc", "q":
		switch m.pendingDelete {
		case "__RECOVER__", "__DISARM__", "__PROBE__":
			m.scr = scrMain
		default:
			m.scr = scrProfiles
		}
		m.pendingDelete = ""
	}
	return m, nil
}

func (m tuiModel) keyDiagnostics(k tea.KeyMsg) (tea.Model, tea.Cmd) {
	if k.String() == "esc" || k.String() == "q" {
		m.scr = scrMain
		return m, nil
	}
	commands := map[string][]string{
		"1": {"doctor"}, "2": {"verify"}, "3": {"test"},
		"4": {"adapters"}, "5": {"netdiag"}, "6": {"diagnose"},
		"7": {"trace"}, "8": {"stealth"}, "9": {"dns-check"},
		"0": {"providers"}, "u": {"update"},
	}
	if args, ok := commands[k.String()]; ok {
		return m, selfTUICmd(strings.Join(args, " "), args...)
	}
	return m, nil
}

func languageCursor() int {
	cur := resolveLang()
	for i, lang := range i18n.Supported {
		if lang == cur {
			return i
		}
	}
	return 0
}

func (m tuiModel) keyLanguage(k tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch k.String() {
	case "esc", "q":
		m.scr = scrSettings
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
	case "down", "j":
		if m.cursor < len(i18n.Supported)-1 {
			m.cursor++
		}
	case "enter":
		if m.cursor >= 0 && m.cursor < len(i18n.Supported) {
			lang := string(i18n.Supported[m.cursor])
			m.scr = scrSettings
			return m, selfTUICmd("language "+lang, "language", lang)
		}
	}
	return m, nil
}

func (m tuiModel) keyHelp(k tea.KeyMsg) (tea.Model, tea.Cmd) {
	if k.String() == "esc" || k.String() == "q" || k.String() == "?" {
		m.scr = scrMain
	}
	return m, nil
}

func (m tuiModel) viewProfiles() string {
	var b strings.Builder
	b.WriteString(m.header() + "\n\n")
	b.WriteString(stLogTitle.Render("  Profiles (↑/↓, Enter connect, i import file/folder, f favorite, d remove, v verify, esc back)") + "\n")
	if len(m.profiles) == 0 {
		b.WriteString(stDim.Render("  No profiles. Press i to import a config or provider folder.") + "\n")
		return b.String()
	}
	for i, row := range m.profiles {
		cur := "  "
		if i == m.cursor {
			cur = stKey.Render("> ")
		}
		fav := " "
		if row.Favorite {
			fav = "★"
		}
		b.WriteString(fmt.Sprintf("%s%s %-28s %-11s %-4s\n", cur, fav, safeDisplay(row.Name), safeDisplay(row.Protocol), safeDisplay(row.Country)))
	}
	return b.String()
}

func (m tuiModel) viewImport() string {
	return m.header() + "\n\n" + stLogTitle.Render("  Import config or provider folder") +
		"\n  Enter a file or directory path. Directories are scanned recursively.\n\n  Path: " +
		stKey.Render(m.input+"█") + "\n\n  [enter] import   [ctrl+u] clear   [esc] cancel\n"
}

func (m tuiModel) viewRemoveConfirm() string {
	switch m.pendingDelete {
	case "__RECOVER__":
		return m.header() + "\n\n" + stWarn.Render("  Recover will force-clean all Mazzy VPN tunnels and guards.") +
			"\n  Continue? [y/N]\n"
	case "__DISARM__":
		return m.header() + "\n\n" + stWarn.Render("  ⛔ HARD RESET: kill the daemon and remove ALL rules/kill-switch, restore DNS.") +
			"\n  This briefly drops all networking to return you to a clean state. Continue? [y/N]\n"
	case "__PROBE__":
		return m.header() + "\n\n" + stWarn.Render("  🧪 DEEP PROBE connects every zone for real to find which servers route.") +
			"\n  This STOPS the VPN for a few minutes. Continue? [y/N]\n"
	}
	return m.header() + "\n\n" + stWarn.Render("  Remove profile: "+safeDisplay(m.pendingDelete)+"?") +
		"\n  This removes the managed catalog copy. Continue? [y/N]\n"
}

func (m tuiModel) viewDiagnostics() string {
	return m.header() + "\n\n" + stLogTitle.Render("  Diagnostics (command output opens temporarily; esc back)") + `
  [1] Doctor / host health          [2] Verify all configs
  [3] Test and rank servers         [4] Network adapters
  [5] Analyze network               [6] Diagnose problems
  [7] Trace packet path             [8] Stealth / leak check
  [9] DNS privacy                   [0] AI providers
  [u] Check for updates
`
}

func (m tuiModel) viewLanguage() string {
	var b strings.Builder
	b.WriteString(m.header() + "\n\n")
	b.WriteString(stLogTitle.Render("  Language (↑/↓ select, Enter apply, esc back)") + "\n")
	for i, lang := range i18n.Supported {
		cur := "  "
		if i == m.cursor {
			cur = stKey.Render("> ")
		}
		b.WriteString(fmt.Sprintf("%s%-4s %s\n", cur, lang, lang.NativeName()))
	}
	return b.String()
}

func (m tuiModel) viewHelp() string {
	return m.header() + "\n\n" + stLogTitle.Render("  Help / command equivalents (? or esc back)") + `
  c  connect best       mazzy-vpn daemon --best --session
  z  choose zone        mazzy-vpn test / up NAME
  p  profiles           mazzy-vpn profiles / import / remove / verify
  x  diagnostics        mazzy-vpn doctor / netdiag / diagnose / trace
  d  disconnect         mazzy-vpn disconnect
  r  recover            mazzy-vpn recover (confirmation required)
  g  graph window       left/right inspect historical samples
  s  settings           language and connection preferences
  l  activity log       k stop daemon       q quit

  Terminal help: mazzy-vpn help COMMAND or mazzy-vpn COMMAND --help
`
}

type graphWindow struct {
	label string
	span  time.Duration
}

var graphWindows = []graphWindow{
	{label: "1m", span: time.Minute},
	{label: "5m", span: 5 * time.Minute},
	{label: "20m", span: 20 * time.Minute},
	{label: "session", span: 0},
}

func (m tuiModel) graphSeries(snap runstatus.Snapshot) ([]int, string, string) {
	idx := m.graphWindow
	if idx < 0 || idx >= len(graphWindows) {
		idx = 0
	}
	window := graphWindows[idx]
	cutoff := int64(0)
	if window.span > 0 {
		cutoff = time.Now().Add(-window.span).Unix()
	}
	samples := make([]runstatus.Sample, 0, len(snap.Samples))
	for _, sample := range snap.Samples {
		if cutoff == 0 || sample.TS >= cutoff {
			samples = append(samples, sample)
		}
	}
	series := make([]int, len(samples))
	for i, sample := range samples {
		series[i] = sample.LatencyMS
	}
	selected := ""
	if len(samples) > 0 {
		cursor := m.graphCursor
		if cursor < 0 {
			cursor = 0
		}
		if cursor >= len(samples) {
			cursor = len(samples) - 1
		}
		sample := samples[len(samples)-1-cursor]
		value := "drop"
		if sample.OK && sample.LatencyMS > 0 {
			value = fmt.Sprintf("%dms", sample.LatencyMS)
		}
		selected = fmt.Sprintf(" · %s %s", time.Unix(sample.TS, 0).Format("15:04:05"), value)
	}
	return series, window.label, selected
}

// statusPane is a stable five-row operational panel. It combines connection
// state and the newest structured error events without layout jumping.
func (m tuiModel) statusPane() string {
	rows := []string{"State: disconnected", "Egress: —", "Health: no samples", "Probe: p50/p95/jitter —", "Event: no recent errors"}
	if snap, ok := daemonRunning(); ok {
		rows[0] = fmt.Sprintf("State: %s · zone %s · protocol %s", snap.State, safeDisplay(snap.Zone), safeDisplay(snap.Protocol))
		rows[1] = fmt.Sprintf("Egress: %s · interface %s · heartbeat %s ago", dashValue(safeDisplay(snap.Egress)), dashValue(safeDisplay(snap.Interface)), shortDur(snap.HeartbeatAge()))
		rows[2] = fmt.Sprintf("Health: %d checks · %d failed · %.1f%% loss · %d reconnects", snap.Checks, snap.Fails, snap.LossPercent(), snap.Reconnects)
		rows[3] = fmt.Sprintf("Egress probe: p50 %dms · p95 %dms · jitter %dms", snap.LatencyPercentile(50), snap.LatencyPercentile(95), snap.JitterMS())
		if recent := snap.RecentErrors(1); len(recent) > 0 {
			rows[4] = fmt.Sprintf("Event: %s %s", time.Unix(recent[0].TS, 0).Format("15:04:05"), safeDisplay(recent[0].Reason))
		}
	} else if m.iface != "" {
		rows[0] = "State: link up · interface " + safeDisplay(m.iface)
		rows[1] = "Egress: " + dashValue(safeDisplay(m.snap.EgressIP))
		rows[2] = "Health: " + dashValue(safeDisplay(m.snap.Reason))
	}
	for i := range rows {
		rows[i] = trunc(rows[i], 76)
	}
	return stBox.Render(stLogTitle.Render("Status & errors — 5 rows") + "\n" + strings.Join(rows, "\n"))
}

func dashValue(value string) string {
	if value == "" {
		return "—"
	}
	return value
}
