# Tasks

## 1. Реализация в daemon.go

- [x] 1.1 Добавить `probeHist`, `lastHSAge`, `hsChurn` и константы `probeWindow`, `probeLossLimit`, `hsChurnLimit`, `hsChurnAge`
- [x] 1.2 Хелперы `noteProbe(ok) int` и `noteHandshake(age, ok)`
- [x] 1.3 Ветка `s.Protected()`: `softFails--`, запись в окно, churn-учёт, флаг `degraded`, выход в реконнект без `continue`
- [x] 1.4 Ветка soft-fail: запись в окно, churn-учёт, расширенное условие эскалации и лог с `window loss`/`hs churn`
- [x] 1.5 Сброс накопителей при teardown

## 2. Проверка

- [x] 2.1 `gofmt`, `go vet ./cmd/mazzy-vpn/`
- [x] 2.2 `go test ./cmd/mazzy-vpn/` (-mod=vendor)
- [x] 2.3 Юнит-тест логики окна/churn (вынести хелперы в тестируемые функции)
- [ ] 2.4 Полевая проверка: после следующего запуска демона на AWG-зоне убедиться по `/run/mazzy-vpn/daemon.log`, что при флапе срабатывает `tunnel degraded` / `egress loss confirmed` и происходит failover

## 3. Выкладка

- [ ] 3.1 Собрать `mazzy-vpn` и заменить `~/.local/bin/mazzy-vpn` (после остановки текущего демона)
- [x] 3.2 Запись в CHANGELOG.md, bump версии (2.4.7)
- [ ] 3.3 Коммит в ветку `fix/degraded-tunnel-failover`, PR в `feat/go-rewrite`
