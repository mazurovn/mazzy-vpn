# Security assessment — CodeQL Go alerts (CLI, v2.1.1)

CodeQL Go analysis (added to CI in the v2.1 line) raised two alert classes on the
CLI. This documents the disposition.

## go/log-injection (29 alerts) — FIXED

**Concern:** user-controlled values (profile names, paths, zone names, uplink,
control-plane IDs, timezone) printed to the terminal could carry ANSI escape
sequences or a BEL, spoofing output or moving the cursor.

**Fix:** `safeDisplay()` (in `commands.go`) strips ASCII control and C1
characters (tabs→space, printable Unicode preserved) and is applied to every
user-controlled display value across `connect`, `status`, `up`/`best`,
`catalog` (import/profiles/favorite/remove), `menu`, `control` and `stealth`.
Unit-tested (`TestSafeDisplayStripsControl`).

## go/path-injection (21 alerts) — ACCEPTED BY DESIGN

**Concern:** `os.ReadFile`/`os.Open` with user-influenced paths.

**Disposition:** these are the tool's core, intended behavior and are safe:

1. **`connect FILE` / `import FILE|DIR`** — the user *explicitly* names a file or
   directory to read, exactly like `cat`/`cp`. Reading the path the user asked
   for is the feature, not a vulnerability. The process runs with the invoking
   user's privileges; no privilege boundary is crossed by reading a path they
   already control.
2. **Managed catalog (`catalog.go`)** — profile *names* are never used to build
   filesystem paths from raw input. `safeName()` reduces any stem to
   `[A-Za-z0-9_-]` (so `../x` becomes `--x`), and `Get(name)` resolves by an
   exact match against stored entries, not by joining user input into a path.
   Directory traversal is therefore not reachable through catalog names.
3. **Config/state/identity files** — paths are derived from
   `os.UserConfigDir()` / `MAZZY_CONFIG_HOME`, not from untrusted network input.

No privilege escalation or traversal beyond the invoking user's own intent is
possible, so these alerts are accepted rather than "fixed" by adding artificial
allow-lists that would break the legitimate "open the file I named" workflow.

## Result

- log-injection: hardened with `safeDisplay` + test.
- path-injection: reviewed, safe-by-design (explicit user paths + sanitized
  catalog names), accepted.
