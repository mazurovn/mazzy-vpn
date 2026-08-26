// SPDX-License-Identifier: PolyForm-Noncommercial-1.0.0
// Copyright © 2026 Nik m (@mazurovn). All rights reserved.

package main

import (
	"net"
	"os"
)

// sdNotify sends a state message to the systemd notify socket (sd_notify(3))
// when the daemon runs under a Type=notify unit. Outside systemd (no
// NOTIFY_SOCKET) it is a no-op, so the same binary works standalone.
//
// The daemon sends READY=1 once its loop is up, WATCHDOG=1 from the heartbeat
// pulse goroutine, and STOPPING=1 on shutdown. Combined with WatchdogSec in
// the unit, systemd restarts a daemon whose process wedges hard enough to stop
// even the pulse — the last-resort 99.9% guard the in-process self-healing
// cannot provide for its own crash/hang.
func sdNotify(msg string) {
	sock := os.Getenv("NOTIFY_SOCKET")
	if sock == "" || msg == "" {
		return
	}
	if sock[0] == '@' { // abstract namespace socket
		sock = "\x00" + sock[1:]
	}
	conn, err := net.DialUnix("unixgram", nil, &net.UnixAddr{Name: sock, Net: "unixgram"})
	if err != nil {
		return
	}
	defer conn.Close()
	_, _ = conn.Write([]byte(msg))
}
