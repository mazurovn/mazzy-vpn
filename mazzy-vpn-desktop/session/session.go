// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright © 2026 Nik m (@mazurovn). All rights reserved.

// Package session binds the desktop window to mazzy-core and to the CLI.
//
// Connect and disconnect do not require this process to be root. They run
// the same privileged CLI commands the line menu prints (`sudo mazzy-vpn up`
// and the rest). In a graphical session those commands go through pkexec,
// which is how the existing Linux desktop elevates them. The window only
// stops a tunnel process it started.
package session

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/mazurovn/mazzy-vpn/core"
	"github.com/mazurovn/mazzy-vpn/core/catalog"
	"github.com/mazurovn/mazzy-vpn/core/connect"
	"github.com/mazurovn/mazzy-vpn/core/livecheck"
	"github.com/mazurovn/mazzy-vpn/core/pathtrace"
	"github.com/mazurovn/mazzy-vpn/core/profile"
	"github.com/mazurovn/mazzy-vpn/core/settings"
	"github.com/mazurovn/mazzy-vpn/core/state"
	"github.com/mazurovn/mazzy-vpn/desktop/elevate"
	"github.com/mazurovn/mazzy-vpn/desktop/menu"
	"github.com/mazurovn/mazzy-vpn/desktop/monitor"
	"github.com/mazurovn/mazzy-vpn/desktop/panel"
)

// Config points the session at the same directories the CLI uses.
type Config struct {
	CatalogDir string
	StateDir   string
	RunDir     string
}

// DefaultConfig honors MAZZY_CONFIG_DIR, MAZZY_STATE_DIR and MAZZY_RUN_DIR.
func DefaultConfig() Config {
	stateDir := "/var/lib/mazzy-vpn"
	if d := os.Getenv("MAZZY_STATE_DIR"); d != "" {
		stateDir = d
	}
	runDir := "/run/mazzy-vpn"
	if d := os.Getenv("MAZZY_RUN_DIR"); d != "" {
		runDir = d
	}
	return Config{CatalogDir: catalog.DefaultDir(), StateDir: stateDir, RunDir: runDir}
}

// Session implements panel.Control.
type Session struct {
	cat   *catalog.Catalog
	store *state.Store
	run   string

	mu         sync.Mutex
	conn       *connect.Conn
	server     string
	proto      core.Protocol
	path       string
	detail     string
	phase      panel.Phase
	expectDown bool
	hold       *elevate.Proc
	logText    string
	board      monitor.View
	samples    []monitor.Point
	launch     func(context.Context, []string) (*elevate.Proc, error)
	onChange   func()
}

// New binds a session to cfg.
func New(cfg Config) *Session {
	s := &Session{
		cat:   catalog.New(cfg.CatalogDir),
		store: &state.Store{Dir: cfg.StateDir},
		run:   cfg.RunDir,
		phase: panel.PhaseDisconnected,
		board: monitor.FromLive(monitor.Live{}),
	}
	s.launch = s.launchCLI
	return s
}

// SetFire registers a callback for log and tunnel changes. It must not
// call back into Session while holding a lock the caller owns.
func (s *Session) SetFire(fn func()) {
	s.mu.Lock()
	s.onChange = fn
	s.mu.Unlock()
}

// CatalogDir is the profile directory the CLI list command reads.
func (s *Session) CatalogDir() string { return s.cat.Dir }

// Board is the latest traffic-light view. Missing measurements stay unknown.
func (s *Session) Board() monitor.View {
	s.mu.Lock()
	defer s.mu.Unlock()
	v := s.board
	v.Points = append([]monitor.Point(nil), s.board.Points...)
	return v
}

// Log is the CLI transcript.
func (s *Session) Log() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.logText
}

// Note appends a line that is not a CLI command, such as a disabled reason.
func (s *Session) Note(text string) {
	s.appendLog(text)
	s.fire()
}

// Load lists profiles and reports a tunnel this process owns, or one left up
// by another Mazzy process.
func (s *Session) Load(ctx context.Context) (panel.Snapshot, error) {
	s.observe(ctx)
	return s.snapshot(), nil
}

// Connect starts `mazzy-vpn up NAME` and returns while that process stays
// up. The CLI holds the tunnel until SIGINT; the window does not become root.
func (s *Session) Connect(ctx context.Context, name string) (panel.Snapshot, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return s.snapshot(), fmt.Errorf("select a server")
	}
	if _, err := s.cat.Get(name); err != nil {
		return s.snapshot(), err
	}
	if err := s.startHold(ctx, []string{"up", name}); err != nil {
		return s.snapshot(), err
	}
	s.mu.Lock()
	s.server = name
	s.phase = panel.PhaseConnecting
	s.expectDown = false
	s.detail = "mazzy-vpn up " + name
	s.mu.Unlock()
	return s.snapshot(), nil
}

// Close stops a tunnel process this window started. A tunnel owned by
// another CLI process is left running.
func (s *Session) Close(ctx context.Context) {
	s.mu.Lock()
	hold := s.hold
	s.hold = nil
	conn := s.conn
	s.conn = nil
	s.phase = panel.PhaseDisconnected
	s.path = ""
	s.detail = ""
	s.mu.Unlock()
	if hold != nil {
		_ = hold.Stop()
	}
	if conn == nil {
		return
	}
	_ = conn.Down(ctx)
	if st, err := s.store.Read(); err == nil {
		st.Desired = core.DesiredDown
		_ = s.store.Write(*st)
	}
}

// Disconnect signals the CLI process this window started. A tunnel started
// outside the window is stopped with `mazzy-vpn disconnect`.
func (s *Session) Disconnect(ctx context.Context) (panel.Snapshot, error) {
	s.mu.Lock()
	hold := s.hold
	s.hold = nil
	conn := s.conn
	s.conn = nil
	s.expectDown = true
	s.phase = panel.PhaseDisconnected
	s.detail = "отключение"
	s.mu.Unlock()

	if hold != nil {
		err := hold.Stop()
		s.appendLog(hold.Output())
		return s.snapshot(), err
	}
	if conn != nil {
		return s.snapshot(), conn.Down(ctx)
	}
	out, err := s.runWait(ctx, []string{"disconnect"})
	s.appendLog(out)
	return s.snapshot(), err
}

// Do runs one menu action. Disabled calls are recorded and not executed.
func (s *Session) Do(ctx context.Context, call menu.Call) {
	if call.Disabled != "" {
		s.appendLog(call.Label + " — " + call.Disabled)
		s.fire()
		return
	}
	if call.Settings {
		s.appendLog(s.settingsText())
		s.fire()
		return
	}
	if call.TraceZone != "" {
		s.runTrace(ctx, call.TraceZone)
	}
	var b strings.Builder
	for _, args := range call.Argv {
		if len(args) == 0 {
			continue
		}
		fmt.Fprintf(&b, "$ mazzy-vpn %s\n", strings.Join(args, " "))
		switch args[0] {
		case "up", "connect", "auto":
			if err := s.startHold(ctx, args); err != nil {
				fmt.Fprintf(&b, "%s\n", err)
				continue
			}
			s.mu.Lock()
			s.phase = panel.PhaseConnecting
			s.expectDown = false
			s.detail = "mazzy-vpn " + strings.Join(args, " ")
			if args[0] == "up" && len(args) > 1 && !strings.HasPrefix(args[1], "-") {
				s.server = args[1]
			}
			s.mu.Unlock()
			b.WriteString("запущено\n")
		case "disconnect":
			if _, err := s.Disconnect(ctx); err != nil {
				fmt.Fprintf(&b, "%s\n", err)
			}
		default:
			out, err := s.runWait(ctx, args)
			b.WriteString(out)
			if out == "" || !strings.HasSuffix(out, "\n") {
				b.WriteByte('\n')
			}
			if err != nil {
				fmt.Fprintf(&b, "%s\n", err)
			}
		}
	}
	if text := b.String(); text != "" {
		s.appendLog(text)
	}
	s.fire()
}

func (s *Session) snapshot() panel.Snapshot {
	profiles, _ := s.profiles()
	s.mu.Lock()
	defer s.mu.Unlock()
	phase := s.phase
	server, proto, path, detail := s.server, s.proto.Title(), s.path, s.detail
	if s.expectDown {
		phase = panel.PhaseDisconnected
	} else if s.conn == nil && phase != panel.PhaseError {
		if iface := liveInterface(); iface != "" {
			phase = panel.PhaseConnected
			if server == "" {
				if st, err := s.store.Read(); err == nil {
					server = st.Profile
					proto = st.Protocol.Title()
				}
			}
			if detail == "" {
				detail = "tunnel " + iface + " is up"
			}
		} else if phase == "" {
			phase = panel.PhaseDisconnected
		}
	}
	if proto == "" && phase == panel.PhaseDisconnected {
		proto = ""
	}
	return panel.Snapshot{
		Phase:     phase,
		Server:    server,
		Protocol:  proto,
		Transport: path,
		Detail:    detail,
		Profiles:  profiles,
	}
}

func (s *Session) profiles() ([]panel.Profile, error) {
	entries, err := s.cat.List()
	if err != nil {
		return nil, err
	}
	out := make([]panel.Profile, 0, len(entries))
	for _, e := range entries {
		item := panel.Profile{Name: e.Name, Protocol: e.Protocol.Title()}
		if data, err := os.ReadFile(e.File); err == nil {
			if cfg, err := profile.Parse(string(data)); err == nil {
				item.Endpoint = cfg.Endpoint()
			}
		}
		out = append(out, item)
	}
	return out, nil
}

func (s *Session) launchCLI(ctx context.Context, args []string) (*elevate.Proc, error) {
	plan, err := elevate.Command(os.Geteuid(), elevate.Graphical(), elevate.Look, args)
	if err != nil {
		return nil, err
	}
	plan = elevate.ForwardEnv(plan, []string{
		"MAZZY_CONFIG_DIR=" + s.cat.Dir,
		"MAZZY_STATE_DIR=" + s.store.Dir,
		"MAZZY_RUN_DIR=" + s.run,
	}, elevate.Look)
	return elevate.Start(ctx, plan)
}

func (s *Session) start(ctx context.Context, args []string) (*elevate.Proc, error) {
	launch := s.launch
	if launch == nil {
		launch = s.launchCLI
	}
	return launch(ctx, args)
}

func (s *Session) startHold(ctx context.Context, args []string) error {
	proc, err := s.start(ctx, args)
	if err != nil {
		return err
	}
	s.mu.Lock()
	old := s.hold
	s.hold = proc
	s.mu.Unlock()
	if old != nil {
		_ = old.Stop()
	}
	go func() {
		err := proc.Wait()
		s.appendLog(proc.Output())
		s.mu.Lock()
		if s.hold == proc {
			s.hold = nil
			if liveInterface() == "" && !s.expectDown && err != nil {
				s.phase = panel.PhaseError
				s.detail = err.Error()
			}
		}
		s.mu.Unlock()
		s.fire()
	}()
	return nil
}

func (s *Session) runWait(ctx context.Context, args []string) (string, error) {
	cctx, cancel := context.WithTimeout(ctx, 90*time.Second)
	defer cancel()
	proc, err := s.start(cctx, args)
	if err != nil {
		return "", err
	}
	done := make(chan error, 1)
	go func() { done <- proc.Wait() }()
	select {
	case err := <-done:
		return proc.Output(), err
	case <-cctx.Done():
		_ = proc.Stop()
		return proc.Output(), cctx.Err()
	}
}

func (s *Session) observe(ctx context.Context) {
	iface := liveInterface()
	view := monitor.FromLive(monitor.Live{Checked: true})
	var point *monitor.Point
	if iface == "" {
		if st, err := s.store.Read(); err == nil && st.Desired == core.DesiredUp {
			view.Connection = monitor.Yellow
			view.ConnectionText = "запрошено включение, интерфейса нет"
		}
	} else {
		start := time.Now()
		snap := livecheck.New().Check(ctx, iface)
		ms := time.Since(start).Milliseconds()
		view = monitor.FromLive(monitor.Live{
			Checked: true,
			Link:    snap.LinkUp,
			Egress:  snap.EgressOK,
			IP:      snap.EgressIP,
			Reason:  snap.Reason,
			ProbeMS: ms,
			Timed:   true,
		})
		if len(view.Points) == 1 {
			p := view.Points[0]
			point = &p
		}
		if snap.EgressIP != "" {
			s.mu.Lock()
			if s.detail == "" || s.detail == "mazzy-vpn up "+s.server {
				s.detail = snap.EgressIP
			}
			s.mu.Unlock()
		}
	}
	s.mu.Lock()
	if s.board.Path != "" {
		view.Path = s.board.Path
		view.PathText = s.board.PathText
	}
	if point != nil {
		s.samples = append(s.samples, *point)
		if len(s.samples) > 32 {
			s.samples = s.samples[len(s.samples)-32:]
		}
	}
	view.Points = append([]monitor.Point(nil), s.samples...)
	s.board = view
	if iface != "" && !s.expectDown && s.phase != panel.PhaseError {
		s.phase = panel.PhaseConnected
		if s.server == "" {
			if st, err := s.store.Read(); err == nil {
				s.server = st.Profile
				s.proto = st.Protocol
			}
		}
	}
	s.mu.Unlock()
}

func (s *Session) runTrace(ctx context.Context, name string) {
	entry, err := s.cat.Get(name)
	if err != nil {
		s.appendLog("Путь — " + err.Error())
		return
	}
	data, err := os.ReadFile(entry.File)
	if err != nil {
		s.appendLog("Путь — " + err.Error())
		return
	}
	cfg, err := profile.Parse(string(data))
	if err != nil {
		s.appendLog("Путь — " + err.Error())
		return
	}
	tr := (&pathtrace.Tracer{}).Run(ctx, cfg.Endpoint(), liveInterface())
	light, text := monitor.FromTrace(tr.Steps)
	var extra []monitor.Point
	for _, step := range tr.Steps {
		if step.Duration > 0 && step.Status == pathtrace.OK {
			extra = append(extra, monitor.Point{MS: step.Duration, HasMS: true})
		}
	}
	s.mu.Lock()
	s.board.Path = light
	s.board.PathText = text
	s.samples = append(s.samples, extra...)
	if len(s.samples) > 32 {
		s.samples = s.samples[len(s.samples)-32:]
	}
	s.board.Points = append([]monitor.Point(nil), s.samples...)
	s.mu.Unlock()
	var b strings.Builder
	for _, step := range tr.Steps {
		fmt.Fprintf(&b, "%s  %s  %s\n", step.Status, step.Name, step.Detail)
	}
	s.appendLog(b.String())
}

func (s *Session) settingsText() string {
	set := settings.NewStore().Load()
	data, err := json.MarshalIndent(set, "", "  ")
	if err != nil {
		return err.Error()
	}
	return "settings.json\n" + string(data)
}

func (s *Session) appendLog(text string) {
	text = strings.TrimSpace(text)
	if text == "" {
		return
	}
	s.mu.Lock()
	if s.logText != "" {
		s.logText += "\n"
	}
	s.logText += text
	if len(s.logText) > 12000 {
		s.logText = s.logText[len(s.logText)-12000:]
	}
	s.mu.Unlock()
}

func (s *Session) fire() {
	s.mu.Lock()
	fn := s.onChange
	s.mu.Unlock()
	if fn != nil {
		fn()
	}
}

func liveInterface() string {
	for _, iface := range core.ManagedInterfaces() {
		if iface == "" {
			continue
		}
		if _, err := net.InterfaceByName(iface); err == nil {
			return iface
		}
	}
	return ""
}
