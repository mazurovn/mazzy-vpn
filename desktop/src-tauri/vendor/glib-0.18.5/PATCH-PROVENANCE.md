# glib 0.18.5 soundness backport

Mazzy VPN vendors the crates.io `glib` 0.18.5 source because Tauri 2.11.5 on
Linux still depends on the end-of-life GTK3 bindings, whose `glib ^0.18`
constraint cannot select the supported `glib >=0.20` line.

Source and review evidence:

- crates.io package: `glib 0.18.5`
- crates.io archive SHA-256:
  `233daaf6e83ae6a12a52055f568f9d7cf4671dabb78ff9560ab6da230ce00ee5`
- upstream repository: `https://github.com/gtk-rs/gtk-rs-core`
- upstream fix: `b5a4071e439bef2b5eea76c3aa25e5ae84839e34`
- upstream review: `https://github.com/gtk-rs/gtk-rs-core/pull/1343`
- advisory: `RUSTSEC-2024-0429` / `GHSA-wrw7-89jp-8q8g`

The backport changes only `glib/src/variant_iter.rs`: the local out-pointer is
made mutable and passed to GLib as `&mut p`. This is the exact upstream fix and
does not change the Rust API, crate features or native GLib ABI.

`tests/check-glib-backport.py` verifies the original archive checksum, compares
every vendored upstream file byte-for-byte, proves that this two-line backport
is the only source change and confirms that Cargo resolves `glib` from this
directory. Cargo-deny then audits the remaining registry dependency graph with
an empty advisory ignore list. The path override must be removed when Tauri
migrates to a supported GTK/glib line.
