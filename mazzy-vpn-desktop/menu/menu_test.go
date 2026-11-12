// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright © 2026 Nik m (@mazurovn). All rights reserved.

package menu

import "testing"

func TestLabelsMatchTheCLIMenu(t *testing.T) {
	want := []string{
		"Связь", "Быстро", "Выбрать зону", "Переподключить", "Отключить",
		"Восстановить", "Сброс", "В фоне", "Диагностика", "Зонд",
		"Серверы", "Лучшая", "Адаптеры", "Сеть", "Разбор", "Путь",
		"Скрытность", "DNS", "Мимик", "Поставщики", "Профили", "Импорт",
		"Список", "Аудит", "Избранное", "Снять профиль", "Языки", "Система",
		"Врач", "Обновление", "Состояние",
	}
	got := Actions()
	if len(got) != len(want) {
		t.Fatalf("len %d", len(got))
	}
	for i, w := range want {
		if got[i].Label != w {
			t.Fatalf("item %d = %q, want %q", i, got[i].Label, w)
		}
	}
}

func TestDisabledActionsHaveReasons(t *testing.T) {
	for _, a := range Actions() {
		call := a.Run(Context{})
		switch a.Label {
		case "Зонд", "В фоне", "Аудит":
			if call.Disabled == "" || len(call.Argv) != 0 {
				t.Fatalf("%s call = %+v", a.Label, call)
			}
		}
	}
}

func TestConnectUsesSelectedProfile(t *testing.T) {
	var link Action
	for _, a := range Actions() {
		if a.Label == "Связь" {
			link = a
		}
	}
	off := link.Run(Context{})
	if off.Disabled == "" {
		t.Fatal("expected a reason without a profile")
	}
	on := link.Run(Context{Profile: "helsinki"})
	if on.Lifecycle != "connect" || len(on.Argv) != 1 || on.Argv[0][0] != "helsinki" {
		t.Fatalf("%+v", on)
	}
}
