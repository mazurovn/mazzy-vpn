#!/usr/bin/env bash
# Build an AppImage without linuxdeploy's GTK dependency scan.
# The DEB is Tauri's already-tested payload; this only repackages it.
set -Eeuo pipefail

ROOT="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/../.." && pwd)"
DESKTOP="$ROOT/desktop"
TARGET_DIR="${CARGO_TARGET_DIR:-$DESKTOP/src-tauri/target}"
OUT_DIR="${1:-$TARGET_DIR/release/bundle/appimage}"
WORK="$(mktemp -d "${TMPDIR:-/tmp}/mazzy-appimage.XXXXXX")"
LOCK="${TMPDIR:-/tmp}/mazzy-vpn-appimage-build.lock"
if ! mkdir "$LOCK" 2>/dev/null; then
  die() { echo "manual-appimage: $*" >&2; exit 1; }
  die "another AppImage build is already running ($LOCK)"
fi
trap 'rmdir -- "$LOCK" 2>/dev/null || true; rm -rf -- "$WORK"' EXIT

die() { echo "manual-appimage: $*" >&2; exit 1; }
command -v cargo >/dev/null || die "cargo is required"
command -v dpkg-deb >/dev/null || die "dpkg-deb is required"
command -v curl >/dev/null || die "curl is required"

VERSION="$(sed -n 's/.*"version": "\([0-9.]*\)".*/\1/p' "$DESKTOP/src-tauri/tauri.conf.json" | head -1)"
[[ -n "$VERSION" ]] || die "cannot determine Desktop version"
mkdir -p "$OUT_DIR"

# Build only DEB when no existing artifact was supplied: linuxdeploy is
# deliberately not invoked.  Reusing a verified DEB also makes AppImage
# repackaging independent from Cargo locks and parallel release jobs.
DEB="${MAZZY_DEB_PATH:-}"
if [[ -z "$DEB" ]]; then
  (cd "$DESKTOP" && npm exec --offline -- tauri build --bundles deb --ci)
  DEB="$(find "$TARGET_DIR/release/bundle/deb" -maxdepth 1 -type f -name '*.deb' -print -quit)"
fi
[[ -s "$DEB" ]] || die "Tauri DEB build did not produce an artifact"

APPDIR="$WORK/Mazzy VPN Desktop.AppDir"
mkdir -p "$APPDIR"
dpkg-deb -x "$DEB" "$APPDIR"
[[ -x "$APPDIR/usr/bin/mazzy-vpn-desktop" ]] || die "DEB payload has no Desktop executable"
[[ -x "$APPDIR/usr/bin/mazzy-vpn" ]] || die "DEB payload has no bundled engine"
[[ -f "$APPDIR/usr/lib/mazzy-vpn/providers/v1/registry.json" ]] || die "provider registry missing"

cat > "$APPDIR/AppRun" <<'EOF'
#!/usr/bin/env bash
set -Eeuo pipefail
HERE="$(CDPATH= cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd -P)"
  # Keep AppImage discovery available to the Rust backend while preserving
  # the normal Tauri resource directory under usr/lib/mazzy-vpn.
  export APPIMAGE="$(CDPATH= cd -- "$HERE/../../.." && pwd -P)"
exec "$HERE/usr/bin/mazzy-vpn-desktop" "$@"
EOF
chmod 755 "$APPDIR/AppRun"
mkdir -p "$APPDIR/usr/share/applications" "$APPDIR/usr/share/icons/hicolor/128x128/apps"
cp "$DESKTOP/src-tauri/icons/128x128.png" \
  "$APPDIR/usr/share/icons/hicolor/128x128/apps/mazzy-vpn-desktop.png"
cat > "$APPDIR/usr/share/applications/mazzy-vpn-desktop.desktop" <<EOF
[Desktop Entry]
Type=Application
Name=Mazzy VPN Desktop
Exec=mazzy-vpn-desktop %U
Icon=mazzy-vpn-desktop
Categories=Network;Utility;
Terminal=false
EOF

TOOL="${APPIMAGETOOL:-}"
if [[ -z "$TOOL" ]]; then
  TOOL="$WORK/appimagetool.AppImage"
  url="${APPIMAGETOOL_URL:-https://github.com/AppImage/appimagetool/releases/download/continuous/appimagetool-x86_64.AppImage}"
  curl --fail --location --retry 3 --connect-timeout 15 --max-time 180 \
    --output "$TOOL" "$url" || die "cannot download appimagetool; set APPIMAGETOOL"
  chmod 755 "$TOOL"
fi
[[ -x "$TOOL" ]] || die "APPIMAGETOOL is not executable: $TOOL"
mkdir -p "$OUT_DIR"
OUTPUT="$OUT_DIR/Mazzy VPN Desktop-${VERSION}-x86_64.AppImage"
rm -f "$OUTPUT"
timeout "${MAZZY_APPIMAGETOOL_TIMEOUT_SEC:-180}s" \
  env ARCH=x86_64 APPIMAGE_EXTRACT_AND_RUN=1 "$TOOL" "$APPDIR" "$OUTPUT" \
  || die "appimagetool timed out or failed (linuxdeploy GTK scan was bypassed)"
[[ -s "$OUTPUT" ]] || die "appimagetool produced no artifact"
chmod 755 "$OUTPUT"
echo "$OUTPUT"
