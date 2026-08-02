# Безопасность и приватность

Полная двуязычная политика:
[PRIVACY.md](https://github.com/mazurovn/mazzy-vpn/blob/main/PRIVACY.md).

Mazzy VPN не использует обязательный cloud account и не собирает телеметрию.
Проверки endpoint выполняются только по явному действию пользователя или
включённой им проверке здоровья. Приватные профили хранятся локально.

## Что не публикуется

- приватные и preshared keys;
- OpenVPN client keys, auth-файлы и пароли;
- IPsec PSK и PPP credentials;
- реальные рабочие профили;
- личные локальные пути;
- endpoint list пользователя;
- журналы с данными авторизации.

## Защита runtime

- root-owned профили `600`, вся цепочка каталогов без group/world write;
- повторная проверка профиля внутри root service;
- запрет executable hooks/includes/plugins;
- один managed tunnel и одна изменяющая операция через общий runtime
  `.mutation.lock` в текущей непубликованной ветке;
- атомарное desired state;
- транзакционный rollback;
- redaction приватных параметров в расширенных журналах;
- Desktop CSP без remote scripts;
- Desktop enum allowlist без shell;
- status cache без endpoint, path и config content;
- строгая Rust-валидация status/profile cache до передачи в WebView;
- активный профиль определяется по opaque `profile_id` или точному basename;
  одинаковые display names не считаются достаточной идентичностью;
- отсутствие неявного публичного DNS: OpenVPN использует переданный сервером
  DNS либо явный `VPNCTL_OPENVPN_FALLBACK_DNS`.

Общий lock является переходным R0a, а не полноценным `mazzy-vpnd`: direct root
paths пока не имеют общего с API action journal, а rollback ещё не доказывает
восстановление routes, DNS, firewall и отсутствие leak.

Текущий Desktop Agent Control остаётся preview. Fixed enum/argv и отсутствие
shell уменьшают поверхность атаки, но renderer `window.confirm()` ещё не
является command-bound proof присутствия пользователя. До закрытия R0-2 нельзя
использовать этот preview как доверенную границу для high-risk remote actions.

## Проверки репозитория

```bash
./tests/audit-public.sh
gitleaks git --redact
npm audit --audit-level=high
cargo clippy --all-targets -- -D warnings
```

GitHub Actions закреплены полными commit SHA. Release-архивы должны иметь
SHA-256. Security reports: см. `SECURITY.md`; не публикуйте секрет в issue.

Лицензия AGPL позволяет изменять проект при сохранении условий лицензии и
исходного кода. Авторство Nik m и copyright notices нельзя выдавать за чужие.

---

<a id="english"></a>

# Security and privacy

Full bilingual policy:
[PRIVACY.md](https://github.com/mazurovn/mazzy-vpn/blob/main/PRIVACY.md).

Mazzy VPN has no mandatory cloud account and collects no telemetry. Endpoint
checks run only after an explicit user action or a health check the user
enabled. Private profiles remain local.

Private or preshared keys, OpenVPN client keys and auth files, IPsec/PPP
credentials, operational profiles, personal paths and user endpoint lists are
never published.

Runtime controls include root-owned mode-`600` profiles, a complete root-owned
directory chain with no group/world write, repeated
root-side validation, rejection of executable hooks/includes/plugins,
serialized mutations, atomic desired state, transactional rollback and log
redaction. Desktop uses a restrictive CSP, enum action allowlist, no shell and
a strictly validated cache with no endpoint, path or config content. OpenVPN
uses server-provided DNS or an explicit `VPNCTL_OPENVPN_FALLBACK_DNS`; it no
longer silently substitutes a public resolver.

The shared lock is a transitional unreleased R0a boundary, not the target
`mazzy-vpnd`; direct root paths do not yet share the API action journal and
rollback does not prove route/DNS/firewall/leak restoration. Desktop Agent
Control also remains preview-only: its renderer confirmation is not yet a
native command-bound approval proof for high-risk remote actions.

The repository runs public-tree audit, Gitleaks, npm audit, Rust tests and
Clippy. GitHub Actions are pinned to full commit SHAs. Do not put a secret in a
public issue; follow `SECURITY.md`.
