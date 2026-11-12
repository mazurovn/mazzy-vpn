// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright © 2026 Nik m (@mazurovn). All rights reserved.

package elevate

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"sync"
	"syscall"
)

// Proc is a started CLI command. Privileged connect stays running; the
// caller stops it with SIGINT, which is how `mazzy-vpn up` tears down.
type Proc struct {
	cmd *exec.Cmd
	buf bytes.Buffer
	mu  sync.Mutex
}

// Start runs plan. The process keeps running after Start returns.
func Start(ctx context.Context, plan Plan) (*Proc, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if len(plan.Argv) == 0 {
		return nil, exec.ErrNotFound
	}
	cmd := exec.Command(plan.Argv[0], plan.Argv[1:]...)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	p := &Proc{cmd: cmd}
	cmd.Stdout = p
	cmd.Stderr = p
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	return p, nil
}

// Wait blocks until the process exits.
func (p *Proc) Wait() error {
	if p == nil || p.cmd == nil {
		return nil
	}
	return p.cmd.Wait()
}

// Output is the combined output so far.
func (p *Proc) Output() string {
	if p == nil {
		return ""
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.buf.String()
}

// Stop signals the process group the way a terminal Ctrl-C would.
func (p *Proc) Stop() error {
	if p == nil || p.cmd == nil || p.cmd.Process == nil {
		return nil
	}
	if err := syscall.Kill(-p.cmd.Process.Pid, syscall.SIGINT); err != nil {
		return p.cmd.Process.Signal(os.Interrupt)
	}
	return nil
}

// Look finds mazzy-vpn, sudo, and pkexec on PATH or in the usual locations.
func Look(name string) string {
	if path, err := exec.LookPath(name); err == nil {
		return path
	}
	for _, dir := range []string{"/usr/bin", "/usr/local/bin", "/bin"} {
		candidate := dir + "/" + name
		if st, err := os.Stat(candidate); err == nil && !st.IsDir() {
			return candidate
		}
	}
	return ""
}

// Graphical reports whether this process has a display to ask PolicyKit.
func Graphical() bool {
	return os.Getenv("DISPLAY") != "" || os.Getenv("WAYLAND_DISPLAY") != ""
}

// Write implements io.Writer for the child, serialized with Output.
func (p *Proc) Write(b []byte) (int, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.buf.Write(b)
}
