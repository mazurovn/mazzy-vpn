# Профили и импорт

Приватные профили находятся вне Git:

```text
/etc/vpnctl/profiles/
├── amneziawg/
├── wireguard/
├── openvpn/
└── l2tp/
```

Каталоги имеют права `700`, файлы — `600`. Создать пользовательскую структуру:

```bash
mazzy-vpn init-config-dir ~/MazzyConfigs
```

Сначала всегда выполняйте dry-run:

```bash
mazzy-vpn import-dir ~/MazzyConfigs --dry-run
sudo mazzy-vpn import-dir ~/MazzyConfigs
```

Распознавание выполняется по содержимому, а не только по расширению. Проверяются
обязательные секции, endpoint, права и запрещённые директивы. OpenVPN include,
plugin, script hooks и вложенные configs отклоняются. WireGuard/AmneziaWG hooks
не исполняются. Изменённый одноимённый файл без `--force` сохраняется.

URI или JSON современных протоколов можно классифицировать без вывода входа:

```bash
printf '%s' 'vless://REDACTED' | mazzy-vpn protocols detect --stdin --json
mazzy-vpn protocols detect --stdin --json < custom-server.json
```

Это только безопасная классификация, не полная validation. Duplicate JSON keys,
несколько документов и смешанные protocol outbounds отклоняются. Для VLESS,
Hysteria 2, Mieru, NaiveProxy, TUIC, Shadowsocks 2022, Trojan, AnyTLS и
ShadowTLS реализованы закрытая managed schema и атомарный root-only import:

```bash
mazzy-vpn protocols managed-validate --stdin --json < profile.json
mazzy-vpn protocols managed-import profile.json --dry-run --json
sudo mazzy-vpn protocols managed-import profile.json --json
```

Managed profile — нейтральный формат Mazzy, а не произвольный engine JSON и не
автоматический import любого vendor share URI. Источник-symlink, duplicate key,
insecure TLS, пользовательские listener/path/route/mark отклоняются. Файл
сохраняется как `/etc/vpnctl/profiles/PROTOCOL/PROFILE_ID.json` с mode `600`,
каталог — `700`; JSON-ответ не содержит endpoint или credential. Статус import
поэтому `partial`. Подключение всех девяти пока `planned`: не завершены
поставка engine, service/TUN/DNS/routes, rollback и leak tests.

Endpoint для ping всегда читается из самого профиля: `remote` у OpenVPN,
`Endpoint=` у WireGuard/AmneziaWG и `gateway`/`remote` у L2TP. Списка серверов
в коде нет.

У VPN-форматов нет общего стандартного поля локации. Mazzy VPN читает
необязательные безопасные комментарии; если их нет, использует имя файла:

```text
# mazzy-name: AI Workspace
# mazzy-location: Belgium — Brussels
# mazzy-country-code: BE
```

Для `.nmconnection` отображаемое имя также берётся из `[connection] id=`.
`mazzy-country-code` должен быть двухбуквенным ISO-кодом. Он задаёт ожидаемую
страну для сравнения, но не подменяет наблюдаемую геолокацию: её отдельно
возвращают два внешних источника для фактического VPN egress. Без явного кода
фактическая страна отображается, но проверка соответствия остаётся `unknown`, а
общий итог — `warning`.

Никогда не добавляйте рабочие профили в issue, Wiki или Git. Для примеров
используйте только документационные адреса `192.0.2.0/24`, `198.51.100.0/24`,
`203.0.113.0/24` и маркеры `CHANGE_ME`.

---

<a id="english"></a>

# Profiles and import

Private profiles stay outside Git under `/etc/vpnctl/profiles/` in protocol
subdirectories. Directories use mode `700`; files use `600`.

```bash
mazzy-vpn init-config-dir ~/MazzyConfigs
mazzy-vpn import-dir ~/MazzyConfigs --dry-run
sudo mazzy-vpn import-dir ~/MazzyConfigs
```

Detection uses content, not only extensions. Required sections, endpoints,
permissions and unsafe directives are checked. OpenVPN includes, plugins,
script hooks and nested configs are rejected. WireGuard/AmneziaWG hooks are not
executed. A changed same-name file is preserved unless `--force` is explicit.

Modern share URIs or JSON can be classified without echoing the input:

```bash
printf '%s' 'vless://REDACTED' | mazzy-vpn protocols detect --stdin --json
mazzy-vpn protocols detect --stdin --json < custom-server.json
```

This is classification only, not full validation. Duplicate JSON keys, multiple
documents and mixed protocol outbounds are rejected. VLESS, Hysteria 2, Mieru,
NaiveProxy, TUIC, Shadowsocks 2022, Trojan, AnyTLS and ShadowTLS now have a
closed managed schema and atomic root-only import:

```bash
mazzy-vpn protocols managed-validate --stdin --json < profile.json
mazzy-vpn protocols managed-import profile.json --dry-run --json
sudo mazzy-vpn protocols managed-import profile.json --json
```

A managed profile is Mazzy's neutral format, not arbitrary engine JSON or an
automatic importer for every vendor share URI. Symlink sources, duplicate keys,
insecure TLS and user-controlled listeners, paths, routes or marks are rejected.
The response never contains an endpoint or credential. Import is therefore
`partial`; connection remains `planned` until engine supply, service/TUN/DNS/
routing, rollback and leak tests pass.

Ping endpoints always come from the profile itself: OpenVPN `remote`,
WireGuard/AmneziaWG `Endpoint=`, or L2TP `gateway`/`remote`. There is no
hard-coded server list.

VPN formats have no shared standard location field. Mazzy VPN therefore reads
optional safe comments and falls back to the filename:

```text
# mazzy-name: AI Workspace
# mazzy-location: Belgium — Brussels
# mazzy-country-code: BE
```

For `.nmconnection`, `[connection] id=` is also used as the display name. The
country code is only the expected value for comparison; two external sources
still provide the independently observed VPN-egress country. Without an
explicit code, actual country remains visible but profile match is `unknown`
and the overall verdict stays `warning`.

Never attach operational profiles to an issue, Wiki page or Git commit. Use
documentation address ranges and `CHANGE_ME` placeholders in examples.
