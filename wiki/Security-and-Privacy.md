# Безопасность и приватность

## Что не публикуется

- приватные и preshared keys;
- OpenVPN client keys, auth-файлы и пароли;
- IPsec PSK и PPP credentials;
- реальные рабочие профили;
- личные локальные пути;
- endpoint list пользователя;
- журналы с данными авторизации.

## Защита runtime

- профили `600`, каталоги `700`;
- повторная проверка профиля внутри root service;
- запрет executable hooks/includes/plugins;
- один managed tunnel и одна изменяющая операция;
- атомарное desired state;
- транзакционный rollback;
- redaction приватных параметров в расширенных журналах;
- Desktop CSP без remote scripts;
- Desktop enum allowlist без shell;
- status cache без endpoint, path и config content.

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

Private or preshared keys, OpenVPN client keys and auth files, IPsec/PPP
credentials, operational profiles, personal paths and user endpoint lists are
never published.

Runtime controls include mode-`600` profiles, mode-`700` directories, repeated
root-side validation, rejection of executable hooks/includes/plugins,
serialized mutations, atomic desired state, transactional rollback and log
redaction. Desktop uses a restrictive CSP, enum action allowlist, no shell and
a cache with no endpoint, path or config content.

The repository runs public-tree audit, Gitleaks, npm audit, Rust tests and
Clippy. GitHub Actions are pinned to full commit SHAs. Do not put a secret in a
public issue; follow `SECURITY.md`.
