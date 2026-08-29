// SPDX-License-Identifier: PolyForm-Noncommercial-1.0.0
// Copyright © 2026 Nik m (@mazurovn). All rights reserved.

package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"syscall"
)

// sudoersDropin is the file `trust` manages. A drop-in keeps the grant fully
// reversible (remove one file) and out of the main sudoers.
const sudoersDropin = "/etc/sudoers.d/mazzy-vpn"

// trustedSubcommands are the actions granted passwordless elevation — the set
// the menu/TUI elevate for. `connect` is deliberately EXCLUDED: it takes a raw
// filesystem path, and under a NOPASSWD wildcard a low-privilege user could
// probe root-only files (`sudo mazzy-vpn connect /etc/shadow` echoes the first
// line in the parse error — an arbitrary-file read oracle, security gate
// finding #3). The catalog-only verbs (`up NAME`, `daemon NAME`) resolve names
// through an exact manifest lookup and cover every UI flow.
// `doctor` is included for its --heal mode so unattended automation (agents,
// cron) can rescue the connection without a password; plain doctor is
// read-only anyway, and --heal wields only the already-trusted verbs' powers.
var trustedSubcommands = []string{"daemon", "stop", "disconnect", "recover", "disarm", "up", "auto", "mimic", "doctor", "probe", "reconnect"}

// cmdTrust installs (or removes, with --revoke) a sudoers drop-in that lets ONE
// user run the privileged mazzy-vpn subcommands without a password prompt, so
// the menu/TUI can control the daemon frictionlessly.
//
// Usage: sudo mazzy-vpn trust [--revoke] [--user NAME]
//
// Safety: the rule pins the ABSOLUTE binary path, and installation refuses a
// binary the target user could overwrite (that would be a free root shell).
// The generated file is syntax-checked with `visudo -c` before it is installed.
func cmdTrust(ctx context.Context, args []string) int {
	if !requireRoot("trust") {
		return 1
	}
	if hasFlag(args, "--revoke") {
		if err := os.Remove(sudoersDropin); err != nil {
			if os.IsNotExist(err) {
				fmt.Println("nothing to revoke:", sudoersDropin, "not present")
				return 0
			}
			fmt.Fprintln(os.Stderr, "revoke:", err)
			return 1
		}
		fmt.Println("✔ revoked passwordless control (" + sudoersDropin + " removed)")
		return 0
	}

	user := flagValue(args, "--user")
	if user == "" {
		user = os.Getenv("SUDO_USER") // the human who invoked sudo
	}
	if user == "" || user == "root" {
		fmt.Fprintln(os.Stderr, "cannot determine the user to trust; run via sudo or pass --user NAME")
		return 2
	}
	if !validSudoersName(user) {
		fmt.Fprintf(os.Stderr, "refusing suspicious user name %q\n", safeDisplay(user))
		return 2
	}

	bin, err := canonicalTrustBinary()
	if err != nil {
		fmt.Fprintln(os.Stderr, "trust:", err)
		return 1
	}
	if err := binarySafeForNopasswd(bin); err != nil {
		fmt.Fprintln(os.Stderr, "refusing to install a passwordless rule:", err)
		fmt.Fprintln(os.Stderr, "  Install the binary to a root-owned location first, e.g.:")
		fmt.Fprintln(os.Stderr, "    sudo install "+bin+" /usr/local/bin/mazzy-vpn && sudo mazzy-vpn trust")
		return 1
	}

	content := sudoersContent(user, bin)

	// Validate the candidate with visudo BEFORE it can break sudo for the host.
	tmp, err := os.CreateTemp("", "mazzy-sudoers-*")
	if err != nil {
		fmt.Fprintln(os.Stderr, "trust:", err)
		return 1
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.WriteString(content); err != nil {
		fmt.Fprintln(os.Stderr, "trust:", err)
		return 1
	}
	_ = tmp.Close()
	if visudo, lerr := exec.LookPath("visudo"); lerr == nil {
		if out, verr := exec.CommandContext(ctx, visudo, "-c", "-f", tmp.Name()).CombinedOutput(); verr != nil {
			fmt.Fprintf(os.Stderr, "generated rule failed visudo check; not installing:\n%s\n", strings.TrimSpace(string(out)))
			return 1
		}
	} else {
		fmt.Fprintln(os.Stderr, "trust: visudo not found; refusing to install unvalidated sudoers content")
		return 1
	}

	if err := os.WriteFile(sudoersDropin, []byte(content), 0o440); err != nil {
		fmt.Fprintln(os.Stderr, "trust:", err)
		return 1
	}
	fmt.Printf("✔ %s may now run mazzy-vpn %s without a password\n", safeDisplay(user), strings.Join(trustedSubcommands, "/"))
	fmt.Println("  Rule:", sudoersDropin, "(revoke with: sudo mazzy-vpn trust --revoke)")
	return 0
}

// canonicalTrustBinary picks the binary path to pin in the sudoers rule: the
// system install location when present, else the running executable.
func canonicalTrustBinary() (string, error) {
	for _, p := range []string{"/usr/local/bin/mazzy-vpn", "/usr/bin/mazzy-vpn"} {
		if fi, err := os.Stat(p); err == nil && fi.Mode().IsRegular() {
			return p, nil
		}
	}
	self, err := os.Executable()
	if err != nil || self == "" {
		return "", fmt.Errorf("cannot locate the mazzy-vpn binary")
	}
	return filepath.EvalSymlinks(self)
}

// binarySafeForNopasswd rejects binaries an unprivileged user could replace: a
// NOPASSWD rule on a user-writable path is an instant root escalation.
func binarySafeForNopasswd(bin string) error {
	fi, err := os.Stat(bin)
	if err != nil {
		return err
	}
	st := fi.Mode()
	if !st.IsRegular() {
		return fmt.Errorf("%s is not a regular file", bin)
	}
	if st.Perm()&0o022 != 0 {
		return fmt.Errorf("%s is group/world-writable (%o)", bin, st.Perm())
	}
	if sys, ok := sysOwner(fi); ok && sys != 0 {
		return fmt.Errorf("%s is not owned by root (uid %d)", bin, sys)
	}
	// The containing directory must not be writable by others either.
	dir := filepath.Dir(bin)
	if dfi, err := os.Stat(dir); err == nil {
		if dfi.Mode().Perm()&0o022 != 0 && dfi.Mode()&os.ModeSticky == 0 {
			return fmt.Errorf("directory %s is group/world-writable", dir)
		}
		if sys, ok := sysOwner(dfi); ok && sys != 0 {
			return fmt.Errorf("directory %s is not owned by root (uid %d)", dir, sys)
		}
	}
	return nil
}

// sudoersContent renders the drop-in. Each subcommand is listed both bare and
// with a trailing wildcard because sudo's `cmd *` does not match zero args.
func sudoersContent(user, bin string) string {
	var rules []string
	for _, sc := range trustedSubcommands {
		if sc == "doctor" {
			// Scope doctor to exactly the capability that justifies it: --heal.
			// Plain doctor is unprivileged anyway, and future doctor flags should
			// not inherit passwordless root by default.
			rules = append(rules, bin+" doctor --heal", bin+" doctor --heal *")
			continue
		}
		rules = append(rules, bin+" "+sc, bin+" "+sc+" *")
	}
	// SECURITY INVARIANT: this rule must NEVER carry SETENV or an env_keep for
	// MAZZY_RUN_DIR / MAZZY_CONFIG_HOME / MAZZY_STATE_DIR. sudo's default
	// env_reset strips them, which is what keeps the env-var-driven paths
	// (heartbeat, settings, state) unspoofable under the trusted verbs. Adding
	// env passthrough here would make those directories attacker-steerable.
	return "# Managed by `mazzy-vpn trust` — passwordless daemon control for the menu/TUI.\n" +
		"# Do NOT add SETENV/env_keep here (see sudoersContent security invariant).\n" +
		"# Remove with: sudo mazzy-vpn trust --revoke\n" +
		user + " ALL=(root) NOPASSWD: " + strings.Join(rules, ", ") + "\n"
}

// sysOwner returns the owning uid of a file when the platform exposes it.
func sysOwner(fi os.FileInfo) (uint32, bool) {
	if st, ok := fi.Sys().(*syscall.Stat_t); ok {
		return st.Uid, true
	}
	return 0, false
}

// validSudoersName accepts conventional Unix user names only, so a crafted
// SUDO_USER can never inject sudoers syntax.
var sudoersNameRe = regexp.MustCompile(`^[a-z_][a-z0-9_-]{0,31}\$?$`)

func validSudoersName(u string) bool { return sudoersNameRe.MatchString(u) }
