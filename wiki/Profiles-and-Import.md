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

Never attach operational profiles to an issue, Wiki page or Git commit. Use
documentation address ranges and `CHANGE_ME` placeholders in examples.
