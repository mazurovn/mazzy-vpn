// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright © 2026 Nik m (@mazurovn). All rights reserved.

// Package api is the window's client for /run/mazzy-vpn/api-v1.sock.
// Lifecycle calls go there as the current user. This package does not
// start pkexec or sudo.
package api

import (
	"bufio"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"strings"
	"time"
)

const (
	maxResponse = 1 << 20
	apiVersion  = "1.0"
)

// Kind classifies one socket round trip.
type Kind int

const (
	// KindUnavailable means the socket was not reached. The tunnel was not attempted.
	KindUnavailable Kind = iota
	// KindRefused means the service answered and rejected the request before a tunnel change.
	KindRefused
	// KindFailed means the service accepted the command and the tunnel did not come up.
	KindFailed
	// KindAccepted means the service reported success.
	KindAccepted
)

// Outcome is the Russian line the window may show, plus the classification.
type Outcome struct {
	Kind   Kind
	Line   string
	Action string
}

// Client dials one Unix socket.
type Client struct {
	Path string
	Dial func(ctx context.Context, network, addr string) (net.Conn, error)
}

// ProfileID is the opaque id the installed engine derives for a catalog file.
// It matches the desktop compatibility id: SHA-256 of protocol, a zero byte,
// and the file name, first 16 bytes, prefixed with "profile-".
func ProfileID(protocol, fileName string) string {
	sum := sha256.Sum256([]byte(protocol + "\x00" + fileName))
	return "profile-" + hex.EncodeToString(sum[:16])
}

// Connect sends lifecycle.connect.
func (c Client) Connect(ctx context.Context, profileID string) Outcome {
	return c.mutate(ctx, "lifecycle.connect", 60_000, map[string]string{"profile_id": profileID})
}

// Reconnect sends lifecycle.reconnect.
func (c Client) Reconnect(ctx context.Context) Outcome {
	return c.mutate(ctx, "lifecycle.reconnect", 30_000, map[string]any{})
}

// Disconnect sends lifecycle.disconnect.
func (c Client) Disconnect(ctx context.Context) Outcome {
	return c.mutate(ctx, "lifecycle.disconnect", 30_000, map[string]any{})
}

func (c Client) mutate(ctx context.Context, operation string, deadline int, payload any) Outcome {
	body, action, err := envelope(operation, deadline, payload)
	if err != nil {
		return Outcome{Kind: KindUnavailable, Line: serviceDown}
	}
	out := c.round(ctx, body)
	if out.Kind == kindIndeterminate {
		out = c.round(ctx, body)
	}
	if out.Action == "" {
		out.Action = action
	}
	if out.Kind == kindIndeterminate {
		return Outcome{Kind: KindUnavailable, Line: noAnswer, Action: action}
	}
	return Outcome{Kind: out.Kind, Line: out.Line, Action: out.Action}
}

const (
	serviceDown = "Служба не запущена. Нужен сокет /run/mazzy-vpn/api-v1.sock."
	noGroup     = "Нет группы mazzy-vpn. Подключение без пароля только у этой группы. После добавления нужен новый вход в сеанс."
	noAnswer    = "Нет ответа службы. Повтор через пароль не делается."
	rejected    = "Служба отклонила запрос."
	notFound    = "Профиль не найден в службе."
	engineDown  = "Служба приняла команду, туннель не поднялся."
)

type rawOutcome struct {
	Kind   Kind
	Line   string
	Action string
}

const kindIndeterminate Kind = 99

func (c Client) round(ctx context.Context, body []byte) rawOutcome {
	conn, err := c.dial(ctx)
	if err != nil {
		return rawOutcome{Kind: KindUnavailable, Line: dialLine(err)}
	}
	defer conn.Close()
	deadline, _ := ctx.Deadline()
	if !deadline.IsZero() {
		_ = conn.SetDeadline(deadline)
	}
	if _, err := conn.Write(append(body, '\n')); err != nil {
		return rawOutcome{Kind: kindIndeterminate, Line: noAnswer}
	}
	reader := bufio.NewReaderSize(conn, 4096)
	line, err := reader.ReadBytes('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return rawOutcome{Kind: kindIndeterminate, Line: noAnswer}
	}
	if len(line) == 0 || len(line) > maxResponse {
		return rawOutcome{Kind: kindIndeterminate, Line: noAnswer}
	}
	return classify(body, line)
}

func (c Client) dial(ctx context.Context) (net.Conn, error) {
	if c.Dial != nil {
		return c.Dial(ctx, "unix", c.Path)
	}
	d := net.Dialer{Timeout: 400 * time.Millisecond}
	return d.DialContext(ctx, "unix", c.Path)
}

func dialLine(err error) string {
	msg := err.Error()
	if os.IsNotExist(err) || strings.Contains(msg, "no such file") || strings.Contains(msg, "connection refused") {
		return serviceDown
	}
	if os.IsPermission(err) || strings.Contains(msg, "permission denied") {
		return noGroup
	}
	return noAnswer
}

func envelope(operation string, deadline int, payload any) ([]byte, string, error) {
	action := ident("action")
	req := map[string]any{
		"api_version":   apiVersion,
		"request_id":    ident("request"),
		"operation":     operation,
		"action_id":     action,
		"authorization": "system-mutate",
		"deadline_ms":   deadline,
		"payload":       payload,
	}
	body, err := json.Marshal(req)
	return body, action, err
}

func ident(prefix string) string {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		return prefix + "-0123456789abcdef"
	}
	return prefix + "-" + hex.EncodeToString(b[:])
}

type response struct {
	APIVersion string `json:"api_version"`
	RequestID  string `json:"request_id"`
	Status     string `json:"status"`
	Result     struct {
		State      string `json:"state"`
		MessageKey string `json:"message_key"`
		ActionID   string `json:"action_id"`
	} `json:"result"`
	Error struct {
		Code       string `json:"code"`
		MessageKey string `json:"message_key"`
	} `json:"error"`
}

func classify(request, reply []byte) rawOutcome {
	var sent struct {
		RequestID string `json:"request_id"`
		ActionID  string `json:"action_id"`
	}
	if err := json.Unmarshal(request, &sent); err != nil {
		return rawOutcome{Kind: kindIndeterminate, Line: noAnswer}
	}
	var resp response
	if err := json.Unmarshal(reply, &resp); err != nil {
		return rawOutcome{Kind: kindIndeterminate, Line: noAnswer}
	}
	if resp.APIVersion != apiVersion || resp.RequestID != sent.RequestID {
		return rawOutcome{Kind: kindIndeterminate, Line: noAnswer}
	}
	action := resp.Result.ActionID
	if action == "" {
		action = sent.ActionID
	}
	if resp.Status != "ok" {
		return rawOutcome{Kind: KindRefused, Line: refuseLine(resp.Error.Code), Action: action}
	}
	switch resp.Result.State {
	case "succeeded":
		return rawOutcome{Kind: KindAccepted, Line: "подключено", Action: action}
	case "failed", "timed-out", "rolled-back", "rollback-failed":
		return rawOutcome{Kind: KindFailed, Line: engineDown, Action: action}
	default:
		return rawOutcome{Kind: kindIndeterminate, Line: noAnswer, Action: action}
	}
}

func refuseLine(code string) string {
	switch code {
	case "permission-denied", "authorization-required":
		return noGroup
	case "backend-unavailable":
		return serviceDown
	case "not-found":
		return notFound
	default:
		return rejected
	}
}

// Scrub drops polkit's English denial if a child process ever prints it.
func Scrub(text string) string {
	if strings.Contains(text, "Not authorized") || strings.Contains(text, "This incident has been reported") {
		return rejected + " Повтор через пароль не делается."
	}
	return text
}

// ErrNotCLI is returned when the window is asked to elevate.
var ErrNotCLI = fmt.Errorf("окно не повышает права")
