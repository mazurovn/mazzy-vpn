// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright © 2026 Nik m (@mazurovn). All rights reserved.

// Package monitor turns observations the core already produced into
// traffic lights and a chart. A missing measurement stays unknown.
// Numbers are never filled in.
package monitor

import "github.com/mazurovn/mazzy-vpn/core/pathtrace"

// Light is green, yellow, red, or unknown.
type Light string

const (
	Unknown Light = "unknown"
	Green   Light = "green"
	Yellow  Light = "yellow"
	Red     Light = "red"
)

// Point is one real probe. HasMS is false when the probe was observed to
// fail; MS is then not a latency.
type Point struct {
	MS    int64
	HasMS bool
}

// View is the dashboard.
type View struct {
	Connection     Light
	ConnectionText string
	Handshake      Light
	HandshakeText  string
	Path           Light
	PathText       string
	Egress         string
	EgressLight    Light
	EgressText     string
	Clean          Light
	CleanText      string
	Service        Light
	ServiceText    string
	Points         []Point
}

// Live is one livecheck observation. Checked is false before the first look.
// ProbeMS is set only when the check was timed.
type Live struct {
	Checked bool
	Link    bool
	Egress  bool
	IP      string
	Reason  string
	ProbeMS int64
	Timed   bool
	// Handshake is non-nil only when a handshake age was actually read.
	// livecheck.Check does not fill HandshakeAge, so callers pass nil.
	Handshake *int64
}

// FromLive maps a livecheck snapshot. Handshake stays unknown unless age
// was supplied by a real read.
func FromLive(s Live) View {
	v := View{
		Connection:    Unknown,
		Handshake:     Unknown,
		Path:          Unknown,
		HandshakeText: "возраст рукопожатия не сообщается",
		PathText:      "трассировка ещё не запускалась",
		Egress:        s.IP,
		EgressLight:   Unknown,
		EgressText:    "ещё не проверяли",
		Clean:         Unknown,
		CleanText:     "ещё не проверяли",
		Service:       Unknown,
		ServiceText:   "Antigravity ещё не проверяли",
	}
	if !s.Checked {
		v.ConnectionText = "ещё не проверяли"
	} else if s.Link && s.Egress {
		v.Connection = Green
		v.ConnectionText = "трафик идёт"
		if s.IP != "" {
			v.ConnectionText = s.IP
		}
	} else if s.Link {
		v.Connection = Yellow
		v.ConnectionText = s.Reason
		if v.ConnectionText == "" {
			v.ConnectionText = "интерфейс есть, выхода нет"
		}
	} else {
		v.Connection = Red
		v.ConnectionText = "туннель не поднят"
	}
	if s.Handshake != nil {
		age := *s.Handshake
		v.HandshakeText = formatAge(age)
		switch {
		case age < 180:
			v.Handshake = Green
		case age < 600:
			v.Handshake = Yellow
		default:
			v.Handshake = Red
		}
	}
	if s.Timed {
		v.Points = []Point{{MS: s.ProbeMS, HasMS: s.Egress && s.ProbeMS >= 0}}
		if !s.Egress {
			v.Points[0].HasMS = false
		}
	}
	return v
}

// FromTrace maps pathtrace steps. No steps leaves the path unknown.
func FromTrace(steps []pathtrace.Step) (Light, string) {
	if len(steps) == 0 {
		return Unknown, "трассировка ещё не запускалась"
	}
	worst := Green
	text := steps[len(steps)-1].Detail
	for _, s := range steps {
		switch s.Status {
		case pathtrace.Fail:
			return Red, s.Name + ": " + s.Detail
		case pathtrace.Warn:
			worst = Yellow
			text = s.Name + ": " + s.Detail
		}
	}
	if worst == Green {
		text = steps[len(steps)-1].Name
	}
	return worst, text
}

func formatAge(sec int64) string {
	if sec < 0 {
		sec = 0
	}
	return "рукопожатие " + itoa(sec) + " с назад"
}

func itoa(n int64) string {
	if n == 0 {
		return "0"
	}
	var b [20]byte
	i := len(b)
	neg := n < 0
	if neg {
		n = -n
	}
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		b[i] = '-'
	}
	return string(b[i:])
}
