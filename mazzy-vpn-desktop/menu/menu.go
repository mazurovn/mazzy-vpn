// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright © 2026 Nik m (@mazurovn). All rights reserved.

// Package menu is the desktop copy of the CLI menu. Labels are the Russian
// names of the product commands. An action that the CLI cannot run is
// returned disabled with the reason; this package does not invent a backend.
package menu

import "strings"

// Call is one menu activation.
type Call struct {
	Label    string
	Disabled string
	Confirm  bool
	// Argv is one or more CLI argument lists, run in order.
	// Settings is set when the action reads the settings file instead of the CLI.
	Argv     [][]string
	Settings bool
	// TraceZone asks the caller to run core/pathtrace for this profile name.
	TraceZone string
}

// Context is what the menu knows about the current selection.
type Context struct {
	Profile   string
	Catalog   string
	Path      string
	Confirmed bool
}

// Action is one row.
type Action struct {
	Label string
	Run   func(Context) Call
}

// Actions is the CLI menu in the order Nikolay listed.
func Actions() []Action {
	return []Action{
		{Label: "Связь", Run: connectSelected},
		{Label: "Быстро", Run: func(Context) Call {
			return live("Быстро", []string{"up", "--best"})
		}},
		{Label: "Выбрать зону", Run: func(Context) Call {
			return live("Выбрать зону", []string{"test"})
		}},
		{Label: "Переподключить", Run: reconnect},
		{Label: "Отключить", Run: func(Context) Call {
			return live("Отключить", []string{"disconnect"})
		}},
		{Label: "Восстановить", Run: confirmCmd("Восстановить", "recover")},
		{Label: "Сброс", Run: confirmArgs("Сброс", "recover", "--reset-catalog")},
		{Label: "В фоне", Run: disabled("В фоне", "daemon не возвращает управление — окно его не запускает")},
		{Label: "Диагностика", Run: func(Context) Call { return live("Диагностика", []string{"doctor"}) }},
		{Label: "Зонд", Run: disabled("Зонд", "команды probe нет — список серверов открывает «Серверы»")},
		{Label: "Серверы", Run: func(Context) Call { return live("Серверы", []string{"test"}) }},
		{Label: "Лучшая", Run: func(Context) Call { return live("Лучшая", []string{"best"}) }},
		{Label: "Адаптеры", Run: func(Context) Call { return live("Адаптеры", []string{"adapters"}) }},
		{Label: "Сеть", Run: func(Context) Call { return live("Сеть", []string{"netdiag"}) }},
		{Label: "Разбор", Run: func(Context) Call { return live("Разбор", []string{"diagnose"}) }},
		{Label: "Путь", Run: trace},
		{Label: "Скрытность", Run: func(Context) Call { return live("Скрытность", []string{"stealth"}) }},
		{Label: "DNS", Run: func(Context) Call { return live("DNS", []string{"dns-check"}) }},
		{Label: "Мимик", Run: func(Context) Call { return live("Мимик", []string{"mimic"}) }},
		{Label: "Поставщики", Run: func(Context) Call { return live("Поставщики", []string{"providers"}) }},
		{Label: "Профили", Run: func(Context) Call { return live("Профили", []string{"profiles"}) }},
		{Label: "Импорт", Run: importPath},
		{Label: "Список", Run: listDir},
		{Label: "Аудит", Run: disabled("Аудит", "в mazzy-vpn нет команды audit")},
		{Label: "Избранное", Run: favorite},
		{Label: "Снять профиль", Run: removeProfile},
		{Label: "Языки", Run: func(Context) Call { return live("Языки", []string{"language", "--list"}) }},
		{Label: "Система", Run: func(Context) Call { return Call{Label: "Система", Settings: true} }},
		{Label: "Врач", Run: func(Context) Call { return live("Врач", []string{"doctor"}) }},
		{Label: "Обновление", Run: func(Context) Call { return live("Обновление", []string{"update"}) }},
		{Label: "Состояние", Run: func(Context) Call { return live("Состояние", []string{"status"}) }},
	}
}

func live(label string, args []string) Call {
	return Call{Label: label, Argv: [][]string{args}}
}

func disabled(label, reason string) func(Context) Call {
	return func(Context) Call { return Call{Label: label, Disabled: reason} }
}

func connectSelected(ctx Context) Call {
	if ctx.Profile == "" {
		return Call{Label: "Связь", Disabled: "выберите сервер в списке"}
	}
	return live("Связь", []string{"up", ctx.Profile})
}

func reconnect(Context) Call {
	return Call{Label: "Переподключить", Argv: [][]string{
		{"netdiag"},
		{"disconnect"},
		{"up", "--best"},
	}}
}

func confirmCmd(label, cmd string) func(Context) Call {
	return confirmArgs(label, cmd)
}

func confirmArgs(label string, args ...string) func(Context) Call {
	return func(ctx Context) Call {
		if !ctx.Confirmed {
			return Call{Label: label, Confirm: true, Disabled: "нажмите ещё раз, чтобы подтвердить"}
		}
		return live(label, args)
	}
}

func trace(ctx Context) Call {
	if ctx.Profile == "" {
		return Call{Label: "Путь", Disabled: "выберите сервер: mazzy-vpn trace ZONE"}
	}
	return Call{Label: "Путь", Argv: [][]string{{"trace", ctx.Profile}}, TraceZone: ctx.Profile}
}

func importPath(ctx Context) Call {
	path := strings.TrimSpace(ctx.Path)
	if path == "" {
		return Call{Label: "Импорт", Disabled: "введите путь к файлу или папке"}
	}
	return live("Импорт", []string{"import", path})
}

func listDir(ctx Context) Call {
	dir := strings.TrimSpace(ctx.Path)
	if dir == "" {
		dir = ctx.Catalog
	}
	if dir == "" {
		return Call{Label: "Список", Disabled: "введите каталог для mazzy-vpn list"}
	}
	return live("Список", []string{"list", dir})
}

func favorite(ctx Context) Call {
	if ctx.Profile == "" {
		return Call{Label: "Избранное", Disabled: "выберите сервер"}
	}
	return live("Избранное", []string{"favorite", ctx.Profile})
}

func removeProfile(ctx Context) Call {
	if ctx.Profile == "" {
		return Call{Label: "Снять профиль", Disabled: "выберите сервер"}
	}
	if !ctx.Confirmed {
		return Call{Label: "Снять профиль", Confirm: true, Disabled: "нажмите ещё раз, чтобы снять " + ctx.Profile}
	}
	return live("Снять профиль", []string{"remove", ctx.Profile})
}
