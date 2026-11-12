// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright © 2026 Nik m (@mazurovn). All rights reserved.

// Package session binds the desktop window to mazzy-core and to the CLI.
//
// Connect, reconnect and disconnect talk to /run/mazzy-vpn/api-v1.sock as the
// current user. The window never starts pkexec or sudo. Other commands run
// the CLI directly, and only when they do not need root.
package session

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
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
	"github.com/mazurovn/mazzy-vpn/desktop/api"
	"github.com/mazurovn/mazzy-vpn/desktop/checks"
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
	notice     string
	expectDown bool
	hold       *elevate.Proc
	logText    string
	holdLog    int
	checking   bool
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
	s.followHold()
	s.observe(ctx)
	line := ""
	if liveInterface() == "" {
		line = s.gate()
	}
	s.mu.Lock()
	if s.phase != panel.PhaseError {
		s.notice = line
	}
	s.mu.Unlock()
	return s.snapshot(), nil
}

// PollChecks runs `mazzy-vpn egress --json` when a tunnel interface exists.
// Without one, the egress lights stay unknown and no address is invented.
func (s *Session) PollChecks(ctx context.Context) {
	s.mu.Lock()
	if s.checking {
		s.mu.Unlock()
		return
	}
	s.checking = true
	s.mu.Unlock()
	defer func() {
		s.mu.Lock()
		s.checking = false
		s.mu.Unlock()
	}()

	if liveInterface() == "" {
		s.mu.Lock()
		s.board.EgressLight = monitor.Unknown
		s.board.EgressText = "no tunnel interface"
		s.board.Clean = monitor.Unknown
		s.board.CleanText = "no tunnel interface"
		s.board.Service = monitor.Unknown
		s.board.ServiceText = "no tunnel interface"
		s.board.Egress = ""
		s.mu.Unlock()
		s.fire()
		return
	}
	out, err := s.runWait(ctx, []string{"egress", "--json"})
	if err != nil && out == "" {
		s.noteChecks(err.Error())
		return
	}
	doc, perr := checks.Parse(out)
	if perr != nil {
		s.noteChecks(perr.Error())
		return
	}
	s.mu.Lock()
	before := len(s.board.Points)
	s.board = checks.Apply(s.board, doc)
	if extra := len(s.board.Points) - before; extra > 0 {
		s.samples = append(s.samples, s.board.Points[before:]...)
		if len(s.samples) > 32 {
			s.samples = s.samples[len(s.samples)-32:]
		}
		s.board.Points = append([]monitor.Point(nil), s.samples...)
	}
	s.mu.Unlock()
	s.appendLog(out)
	s.fire()
}

func (s *Session) noteChecks(reason string) {
	s.mu.Lock()
	s.board.EgressLight = monitor.Unknown
	s.board.EgressText = reason
	s.board.Clean = monitor.Unknown
	s.board.CleanText = reason
	s.board.Service = monitor.Unknown
	s.board.ServiceText = reason
	s.mu.Unlock()
	s.fire()
}

func (s *Session) followHold() {
	s.mu.Lock()
	hold := s.hold
	seen := s.holdLog
	s.mu.Unlock()
	if hold == nil {
		return
	}
	out := hold.Output()
	if len(out) <= seen {
		return
	}
	s.appendLog(out[seen:])
	s.mu.Lock()
	s.holdLog = len(out)
	s.mu.Unlock()
}

// Connect sends lifecycle.connect for name. A missing socket or group leaves
// the phase disconnected and sets one Russian notice. It does not return an
// error for that, because the panel would show that error as «Ошибка».
func (s *Session) Connect(ctx context.Context, name string) (panel.Snapshot, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		s.failClosed("выберите сервер в списке")
		return s.snapshot(), nil
	}
	id, err := s.profileID(name)
	if err != nil {
		s.failClosed("выберите сервер в списке")
		return s.snapshot(), nil
	}
	if line := s.gate(); line != "" {
		s.failClosed(line)
		s.appendLog(line)
		s.fire()
		return s.snapshot(), nil
	}
	s.mu.Lock()
	s.server = name
	s.phase = panel.PhaseConnecting
	s.notice = ""
	s.expectDown = false
	s.mu.Unlock()
	s.applyLifecycle(s.socket().Connect(ctx, id), true)
	return s.snapshot(), nil
}

// Reconnect sends lifecycle.reconnect and does not tear the tunnel down first.
func (s *Session) Reconnect(ctx context.Context) (panel.Snapshot, error) {
	if line := s.gate(); line != "" {
		s.mu.Lock()
		s.notice = line
		s.mu.Unlock()
		s.appendLog(line)
		s.fire()
		return s.snapshot(), nil
	}
	s.mu.Lock()
	s.phase = panel.PhaseConnecting
	s.notice = ""
	s.expectDown = false
	s.mu.Unlock()
	s.applyLifecycle(s.socket().Reconnect(ctx), true)
	return s.snapshot(), nil
}

// Close leaves a tunnel owned by the service running. The window does not
// send disconnect and does not signal a root process.
func (s *Session) Close(context.Context) {}

// Disconnect sends lifecycle.disconnect. Without a socket the phase stays
// whatever the interface says, plus one Russian notice.
func (s *Session) Disconnect(ctx context.Context) (panel.Snapshot, error) {
	if line := s.gate(); line != "" {
		s.mu.Lock()
		s.notice = line
		s.mu.Unlock()
		s.appendLog(line)
		s.fire()
		return s.snapshot(), nil
	}
	s.applyLifecycle(s.socket().Disconnect(ctx), false)
	return s.snapshot(), nil
}

// Do runs one menu action. Disabled calls are recorded and not executed.
// Lifecycle actions use the socket. Nothing in this method elevates.
func (s *Session) Do(ctx context.Context, call menu.Call) {
	if call.Disabled != "" {
		s.appendLog(call.Label + " — " + call.Disabled)
		s.fire()
		return
	}
	if call.Lifecycle != "" {
		s.doLifecycle(ctx, call)
		return
	}
	if call.Update {
		s.doUpdate(ctx)
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
		if elevate.Privileged(args) {
			fmt.Fprintf(&b, "%s — окно не повышает права\n", call.Label)
			continue
		}
		fmt.Fprintf(&b, "$ mazzy-vpn %s\n", strings.Join(args, " "))
		out, err := s.runWait(ctx, args)
		b.WriteString(out)
		if out == "" || !strings.HasSuffix(out, "\n") {
			b.WriteByte('\n')
		}
		if err != nil {
			fmt.Fprintf(&b, "%s\n", err)
		}
	}
	if text := b.String(); text != "" {
		s.appendLog(text)
	}
	s.fire()
}

func (s *Session) doLifecycle(ctx context.Context, call menu.Call) {
	switch call.Lifecycle {
	case "connect":
		name := ""
		if len(call.Argv) > 0 && len(call.Argv[0]) > 0 {
			name = call.Argv[0][0]
		}
		if _, err := s.Connect(ctx, name); err != nil {
			s.appendLog(err.Error())
		}
	case "best":
		s.connectBest(ctx)
	case "reconnect":
		if _, err := s.Reconnect(ctx); err != nil {
			s.appendLog(err.Error())
		}
	case "disconnect":
		if _, err := s.Disconnect(ctx); err != nil {
			s.appendLog(err.Error())
		}
	}
	s.fire()
}

func (s *Session) connectBest(ctx context.Context) {
	out, err := s.runWait(ctx, []string{"best", "--json"})
	if err != nil && strings.TrimSpace(out) == "" {
		s.failClosed("не удалось выбрать сервер")
		s.appendLog("не удалось выбрать сервер")
		s.fire()
		return
	}
	var best struct {
		Name string `json:"name"`
	}
	if json.Unmarshal([]byte(out), &best) != nil || strings.TrimSpace(best.Name) == "" {
		s.failClosed("нет доступного сервера")
		s.appendLog("нет доступного сервера")
		s.fire()
		return
	}
	_, _ = s.Connect(ctx, best.Name)
}

func (s *Session) doUpdate(ctx context.Context) {
	ver, _ := s.runWait(ctx, []string{"version"})
	out, err := s.runWait(ctx, []string{"update"})
	line := oneLine(ver)
	if rest := oneLine(out); rest != "" {
		if line != "" {
			line += " — "
		}
		line += rest
	}
	if err != nil && !strings.Contains(line, err.Error()) {
		if line != "" {
			line += " — "
		}
		line += oneLine(err.Error())
	}
	if line == "" {
		line = "проверка обновления не дала ответа"
	}
	s.appendLog(line)
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
	notice := s.notice
	errText := ""
	if phase == panel.PhaseError {
		errText = detail
		if errText == "" {
			errText = "Служба приняла команду, туннель не поднялся."
		}
	}
	return panel.Snapshot{
		Phase:     phase,
		Server:    server,
		Protocol:  proto,
		Transport: path,
		Detail:    detail,
		Err:       errText,
		Notice:    notice,
		Profiles:  profiles,
	}
}

func (s *Session) socket() api.Client {
	return api.Client{Path: filepath.Join(s.run, "api-v1.sock")}
}

func (s *Session) gate() string {
	path := s.socket().Path
	conn, err := net.DialTimeout("unix", path, 300*time.Millisecond)
	if err == nil {
		_ = conn.Close()
		return ""
	}
	msg := err.Error()
	if os.IsNotExist(err) || strings.Contains(msg, "no such file") || strings.Contains(msg, "connection refused") {
		return "Служба не запущена. Нужен сокет /run/mazzy-vpn/api-v1.sock."
	}
	if os.IsPermission(err) || strings.Contains(msg, "permission denied") {
		return "Нет группы mazzy-vpn. Подключение без пароля только у этой группы. После добавления нужен новый вход в сеанс."
	}
	return "Нет ответа службы. Повтор через пароль не делается."
}

func (s *Session) failClosed(line string) {
	s.mu.Lock()
	s.phase = panel.PhaseDisconnected
	s.notice = line
	s.expectDown = false
	s.mu.Unlock()
}

func (s *Session) applyLifecycle(out api.Outcome, wantUp bool) {
	line := api.Scrub(out.Line)
	switch out.Kind {
	case api.KindAccepted:
		s.mu.Lock()
		s.notice = ""
		s.expectDown = !wantUp
		if wantUp && liveInterface() == "" {
			s.phase = panel.PhaseError
			s.detail = "Служба приняла команду, туннель не поднялся."
		} else if wantUp {
			s.phase = panel.PhaseConnected
			s.detail = ""
		} else {
			s.phase = panel.PhaseDisconnected
			s.detail = ""
		}
		s.mu.Unlock()
	case api.KindFailed:
		s.mu.Lock()
		s.phase = panel.PhaseError
		s.notice = ""
		s.detail = line
		s.mu.Unlock()
		s.appendLog(line)
	default:
		s.mu.Lock()
		if liveInterface() == "" {
			s.phase = panel.PhaseDisconnected
		}
		s.notice = line
		s.mu.Unlock()
		s.appendLog(line)
	}
	s.fire()
}

func (s *Session) profileID(name string) (string, error) {
	entry, err := s.cat.Get(name)
	if err != nil {
		return "", err
	}
	s.mu.Lock()
	s.server = entry.Name
	s.proto = entry.Protocol
	s.mu.Unlock()
	return api.ProfileID(string(entry.Protocol), filepath.Base(entry.File)), nil
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
	if elevate.Privileged(args) {
		return nil, fmt.Errorf("окно не повышает права")
	}
	cli := elevate.Look("mazzy-vpn")
	if cli == "" {
		return nil, fmt.Errorf("mazzy-vpn CLI is not installed")
	}
	return elevate.Start(ctx, elevate.Plan{Argv: append([]string{cli}, args...), Via: "direct"})
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
	s.holdLog = 0
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
	if s.board.EgressLight != "" {
		view.EgressLight = s.board.EgressLight
		view.EgressText = s.board.EgressText
		if s.board.Egress != "" {
			view.Egress = s.board.Egress
		}
	}
	if s.board.Clean != "" {
		view.Clean = s.board.Clean
		view.CleanText = s.board.CleanText
	}
	if s.board.Service != "" {
		view.Service = s.board.Service
		view.ServiceText = s.board.ServiceText
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

func oneLine(text string) string {
	text = strings.TrimSpace(text)
	if text == "" {
		return ""
	}
	if i := strings.IndexAny(text, "\r\n"); i >= 0 {
		text = strings.TrimSpace(text[:i])
	}
	return text
}

func (s *Session) appendLog(text string) {
	text = api.Scrub(strings.TrimSpace(text))
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
