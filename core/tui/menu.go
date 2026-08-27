// SPDX-License-Identifier: PolyForm-Noncommercial-1.0.0
// Copyright © 2026 Nik m (@mazurovn). All rights reserved.

// Package tui provides the interactive menu model for the mazzy-core CLI. The
// menu state and action dispatch are separated from terminal rendering so the
// logic is unit-testable without a TTY. A concrete renderer (plain line-based
// or a full-screen library) drives this model.
package tui

import (
	"github.com/mazurovn/mazzy-vpn/core/i18n"
)

// Action identifies a menu action independent of its localized label.
type Action string

const (
	ActConnect    Action = "connect"
	ActDisconnect Action = "disconnect"
	ActStatus     Action = "status"
	ActVerify     Action = "verify"
	ActDoctor     Action = "doctor"
	ActLanguage   Action = "language"
	ActQuit       Action = "quit"
)

// Item is one menu entry: a stable action plus its message key.
type Item struct {
	Action Action
	Key    string // i18n key for the label
}

// mainItems is the canonical main-menu order.
var mainItems = []Item{
	{ActConnect, "menu.connect"},
	{ActDisconnect, "menu.disconnect"},
	{ActStatus, "menu.status"},
	{ActVerify, "menu.verify"},
	{ActDoctor, "menu.doctor"},
	{ActLanguage, "menu.language"},
	{ActQuit, "menu.quit"},
}

// Menu is the interactive menu model.
type Menu struct {
	tr    *i18n.Translator
	items []Item
}

// NewMenu builds a Menu for the given language.
func NewMenu(lang i18n.Lang) *Menu {
	return &Menu{tr: i18n.NewTranslator(lang), items: mainItems}
}

// Title returns the localized menu title.
func (m *Menu) Title() string { return m.tr.T("menu.title") }

// Prompt returns the localized action prompt.
func (m *Menu) Prompt() string { return m.tr.T("prompt.choose") }

// Lines returns the localized, numbered menu lines (1-based), e.g.
// "1. Connect". Rendering-agnostic: a renderer prints these.
func (m *Menu) Lines() []string {
	out := make([]string, len(m.items))
	for i, it := range m.items {
		out[i] = numberLabel(i+1, m.tr.T(it.Key))
	}
	return out
}

// Select maps a 1-based choice to its Action. ok is false when the choice is
// out of range (so the renderer can reprompt without panicking).
func (m *Menu) Select(choice int) (Action, bool) {
	if choice < 1 || choice > len(m.items) {
		return "", false
	}
	return m.items[choice-1].Action, true
}

// SetLanguage switches the menu language (used by the language action).
func (m *Menu) SetLanguage(lang i18n.Lang) {
	m.tr = i18n.NewTranslator(lang)
}

// Len returns the number of menu items.
func (m *Menu) Len() int { return len(m.items) }

// numberLabel formats "N. label" without importing fmt for a hot path.
func numberLabel(n int, label string) string {
	return itoa(n) + ". " + label
}

// itoa converts a small positive int to its decimal string.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}
