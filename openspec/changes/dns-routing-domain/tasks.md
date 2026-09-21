# Tasks

## 1. core/dns
- [x] 1.1 `resolvectl domain <iface> ~.` и `default-route yes` после `dns`
- [x] 1.2 Поле `Available` для тестов
- [x] 1.3 Тест `TestResolvectlSetsRoutingDomain`

## 2. Сборка/релиз
- [x] 2.1 `go mod vendor` в mazzy-vpn-cli
- [x] 2.2 Тесты core и cli, сборка 2.4.8
- [x] 2.3 CHANGELOG, тег v2.4.8, GitHub release, установка бинарника
- [ ] 2.4 Полевая проверка: `resolvectl status` показывает `DNS Domain: ~.` на vpnaw0; `resolvectl query api.ipify.org` → `-- link: vpnaw0`
