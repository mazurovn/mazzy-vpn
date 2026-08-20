// SPDX-License-Identifier: PolyForm-Noncommercial-1.0.0
// Copyright © 2026 Nik m (@mazurovn). All rights reserved.

// Package mlog provides structured, leveled logging for mazzy-core with a
// human-readable console format and an optional JSON file sink. It records
// connection lifecycle, health checks, recovery, and packet-path traces so the
// user can debug what is happening and follow the packet flow end to end.
package mlog

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Level is a log severity.
type Level int

const (
	DEBUG Level = iota
	INFO
	WARN
	ERROR
)

func (l Level) String() string {
	switch l {
	case DEBUG:
		return "DEBUG"
	case INFO:
		return "INFO"
	case WARN:
		return "WARN"
	case ERROR:
		return "ERROR"
	default:
		return "?"
	}
}

// Entry is one structured log record.
type Entry struct {
	Time   string         `json:"time"`
	Level  string         `json:"level"`
	Event  string         `json:"event"`            // stable event key, e.g. "connect.up"
	Msg    string         `json:"msg"`              // human message
	Fields map[string]any `json:"fields,omitempty"` // structured context
}

// Logger writes structured entries to a console writer and (optionally) a JSON
// file. It is safe for concurrent use.
type Logger struct {
	mu       sync.Mutex
	console  io.Writer
	minLevel Level
	jsonFile *os.File
	useColor bool
}

// Options configure a Logger.
type Options struct {
	Console  io.Writer // default os.Stderr
	MinLevel Level     // default INFO
	JSONPath string    // if set, append JSON lines here
	Color    bool
}

// New builds a Logger.
func New(o Options) (*Logger, error) {
	if o.Console == nil {
		o.Console = os.Stderr
	}
	l := &Logger{console: o.Console, minLevel: o.MinLevel, useColor: o.Color}
	if o.JSONPath != "" {
		if err := os.MkdirAll(filepath.Dir(o.JSONPath), 0o700); err != nil {
			return nil, err
		}
		f, err := os.OpenFile(o.JSONPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
		if err != nil {
			return nil, err
		}
		l.jsonFile = f
	}
	return l, nil
}

// Close releases the JSON file if any.
func (l *Logger) Close() error {
	if l.jsonFile != nil {
		return l.jsonFile.Close()
	}
	return nil
}

// SetLevel changes the minimum level at runtime (e.g. debug toggle).
func (l *Logger) SetLevel(lv Level) { l.mu.Lock(); l.minLevel = lv; l.mu.Unlock() }

// log writes an entry at the given level.
func (l *Logger) log(lv Level, event, msg string, fields map[string]any) {
	if lv < l.minLevel {
		return
	}
	now := time.Now()
	e := Entry{Time: now.Format(time.RFC3339), Level: lv.String(), Event: event, Msg: msg, Fields: fields}

	l.mu.Lock()
	defer l.mu.Unlock()

	// Console line: "15:04:05 INFO  event  msg  k=v".
	stamp := now.Format("15:04:05")
	tag := lv.String()
	if l.useColor {
		tag = colorize(lv, tag)
	}
	line := fmt.Sprintf("%s %-5s %-16s %s", stamp, tag, event, msg)
	for k, v := range fields {
		line += fmt.Sprintf(" %s=%v", k, v)
	}
	fmt.Fprintln(l.console, line)

	if l.jsonFile != nil {
		if b, err := json.Marshal(e); err == nil {
			l.jsonFile.Write(b)
			l.jsonFile.Write([]byte{'\n'})
		}
	}
}

func colorize(lv Level, s string) string {
	switch lv {
	case DEBUG:
		return "\033[90m" + s + "\033[0m"
	case INFO:
		return "\033[36m" + s + "\033[0m"
	case WARN:
		return "\033[33m" + s + "\033[0m"
	case ERROR:
		return "\033[31m" + s + "\033[0m"
	}
	return s
}

// Debug/Info/Warn/Error log at each level. F is optional structured context.
func (l *Logger) Debug(event, msg string, f ...map[string]any) { l.log(DEBUG, event, msg, merge(f)) }
func (l *Logger) Info(event, msg string, f ...map[string]any)  { l.log(INFO, event, msg, merge(f)) }
func (l *Logger) Warn(event, msg string, f ...map[string]any)  { l.log(WARN, event, msg, merge(f)) }
func (l *Logger) Error(event, msg string, f ...map[string]any) { l.log(ERROR, event, msg, merge(f)) }

func merge(fs []map[string]any) map[string]any {
	if len(fs) == 0 {
		return nil
	}
	out := map[string]any{}
	for _, f := range fs {
		for k, v := range f {
			out[k] = v
		}
	}
	return out
}

// F is a shorthand for building a fields map.
func F(kv ...any) map[string]any {
	m := map[string]any{}
	for i := 0; i+1 < len(kv); i += 2 {
		if k, ok := kv[i].(string); ok {
			m[k] = kv[i+1]
		}
	}
	return m
}
