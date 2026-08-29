// SPDX-License-Identifier: PolyForm-Noncommercial-1.0.0
// Copyright © 2026 Nik m (@mazurovn). All rights reserved.

package main

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/user"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"
)

// updateRepo is the GitHub repository releases are pulled from.
const updateRepo = "mazurovn/mazzy-vpn"

// ghRelease is the subset of the GitHub Releases API we use.
type ghRelease struct {
	TagName string `json:"tag_name"`
	Name    string `json:"name"`
	Assets  []struct {
		Name               string `json:"name"`
		BrowserDownloadURL string `json:"browser_download_url"`
	} `json:"assets"`
}

// cmdUpdate checks GitHub for a newer release and, with --apply, downloads and
// installs it. Without --apply it only reports what is available (dry run).
// --menu adds the interactive middle ground used by the TUI/menu: check, then
// offer to install right away (elevating itself when not root).
func cmdUpdate(ctx context.Context, args []string) int {
	apply := hasFlag(args, "--apply")
	menu := hasFlag(args, "--menu")
	t := translator()

	fmt.Println(t.Tf("cli.update.current", version))
	fmt.Println(t.Tf("cli.update.checking", updateRepo))

	rel, err := latestRelease(ctx)
	if err != nil {
		fmt.Fprintln(os.Stderr, "update check failed:", err)
		return 1
	}
	fmt.Println(t.Tf("cli.update.latest", rel.TagName))

	if !isNewer(rel.TagName, version) {
		fmt.Println(t.T("cli.update.uptodate"))
		return 0
	}
	fmt.Println(t.Tf("cli.update.available", rel.TagName))

	// Find the linux-amd64 tarball asset.
	asset := ""
	assetName := fmt.Sprintf("mazzy-vpn-go-%s-linux-%s.tar.gz",
		strings.TrimPrefix(rel.TagName, "v"), runtime.GOARCH)
	var fallback string
	for _, a := range rel.Assets {
		if a.Name == assetName {
			asset = a.BrowserDownloadURL
		}
		if strings.HasPrefix(a.Name, "mazzy-vpn-go-") && strings.HasSuffix(a.Name, ".tar.gz") {
			fallback = a.BrowserDownloadURL
		}
	}
	if asset == "" {
		asset = fallback
	}
	if asset == "" {
		fmt.Fprintln(os.Stderr, "no linux tarball asset in the release")
		return 1
	}

	if !apply && menu {
		// Interactive: offer the install right here. The privileged re-exec
		// (sudo password prompt) happens on the caller's real terminal.
		fmt.Printf("Install %s now? [y/N] ", rel.TagName)
		var answer string
		_, _ = fmt.Fscanln(os.Stdin, &answer)
		if !strings.EqualFold(strings.TrimSpace(answer), "y") {
			fmt.Println(t.T("cli.menu.cancelled"))
			return 0
		}
		if os.Geteuid() != 0 {
			return runPrivileged(ctx, "update", "--apply")
		}
		apply = true
	}
	if !apply {
		fmt.Println(t.T("cli.update.hint_apply"))
		return 0
	}

	if os.Geteuid() != 0 {
		fmt.Fprintln(os.Stderr, t.T("cli.update.needs_root"))
		return 1
	}

	fmt.Println(t.Tf("cli.update.downloading", assetName))
	bin, gotSum, err := downloadBinaryFromTarball(ctx, asset)
	if err != nil {
		fmt.Fprintln(os.Stderr, "download failed:", err)
		return 1
	}

	// Verify the extracted binary against the release's published SHA256SUMS
	// asset when present (audit P2-2). The old code computed a sha but never
	// checked it, trusting only a size heuristic. If the release ships no
	// checksum asset we proceed (best-effort) but say so, so a supply-chain
	// mismatch cannot pass silently when checksums ARE published.
	if want, ok := expectedTarballSum(ctx, rel, assetName); ok {
		if !strings.EqualFold(want, gotSum) {
			_ = os.Remove(bin)
			fmt.Fprintf(os.Stderr, "checksum mismatch: published %s but downloaded %s; refusing to install\n", short(want), short(gotSum))
			return 1
		}
		fmt.Printf("  ✔ checksum verified (%s)\n", short(gotSum))
	} else {
		fmt.Println("  ⚠ no SHA256SUMS asset in the release; installing unverified (size-checked only)")
	}

	// Install atomically next to the current binary.
	self, err := os.Executable()
	if err != nil {
		fmt.Fprintln(os.Stderr, "cannot locate current binary:", err)
		return 1
	}
	if err := replaceBinary(self, bin); err != nil {
		fmt.Fprintln(os.Stderr, "install failed:", err)
		return 1
	}
	// Keep the machine's OTHER canonical install in sync: this host convention
	// is /usr/local/bin (sudo PATH, systemd) + the invoking user's ~/.local/bin.
	// Updating only os.Executable() left the twin behind on every release.
	syncCompanionInstalls(self, bin)
	fmt.Println(t.Tf("cli.update.updated", rel.TagName))
	return 0
}

// syncCompanionInstalls best-effort updates the standard install locations
// other than the one just replaced. Targets must already EXIST as regular
// files (Lstat: never through a symlink) — this refreshes known installs, it
// does not create new ones.
func syncCompanionInstalls(self, newBin string) {
	targets := []string{"/usr/local/bin/mazzy-vpn"}
	if su := os.Getenv("SUDO_USER"); su != "" && su != "root" && sudoUserNameOK(su) {
		if u, err := user.Lookup(su); err == nil && u.HomeDir != "" {
			targets = append(targets, filepath.Join(u.HomeDir, ".local", "bin", "mazzy-vpn"))
		}
	} else if home, err := os.UserHomeDir(); err == nil {
		targets = append(targets, filepath.Join(home, ".local", "bin", "mazzy-vpn"))
	}
	selfReal, _ := filepath.EvalSymlinks(self)
	for _, tgt := range targets {
		tgtReal, err := filepath.EvalSymlinks(tgt)
		if err == nil && tgtReal == selfReal {
			continue // already replaced
		}
		fi, err := os.Lstat(tgt)
		if err != nil || !fi.Mode().IsRegular() {
			continue // absent or a symlink: do not touch
		}
		if err := replaceBinary(tgt, newBin); err != nil {
			fmt.Fprintf(os.Stderr, "  ⚠ could not update %s: %v\n", tgt, err)
		} else {
			fmt.Printf("  ✔ also updated %s\n", tgt)
		}
	}
}

// sudoUserNameOK matches conventional Unix user names (no traversal via a
// crafted SUDO_USER), mirroring the settings/catalog hardening.
var sudoUserNameRePattern = regexp.MustCompile(`^[a-z_][a-z0-9_-]{0,31}\$?$`)

func sudoUserNameOK(name string) bool { return sudoUserNameRePattern.MatchString(name) }

// latestRelease fetches the newest GitHub release.
func latestRelease(ctx context.Context) (*ghRelease, error) {
	url := "https://api.github.com/repos/" + updateRepo + "/releases/latest"
	cctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(cctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "mazzy-vpn-updater")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("github API returned %d", resp.StatusCode)
	}
	var rel ghRelease
	if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
		return nil, err
	}
	return &rel, nil
}

// isNewer reports whether tag (e.g. "v2.1.0") is newer than cur (e.g.
// "2.0.0-dev"). It compares dotted numeric components; a genuine prerelease
// suffix (-dev/-rc/-beta/-alpha) on cur is treated as older than the same clean
// release. A local/fork build marker (e.g. "-vpn.local") is NOT a prerelease and
// must not be nagged to "update" to an equal upstream tag (audit P2-1).
func isNewer(tag, cur string) bool {
	tv := parseVer(tag)
	cv := parseVer(cur)
	for i := 0; i < 3; i++ {
		if tv[i] != cv[i] {
			return tv[i] > cv[i]
		}
	}
	// Equal numeric: only a real prerelease current is older than a clean tag.
	return isPrerelease(cur) && !isPrerelease(tag)
}

// isPrerelease reports whether a version string carries a genuine prerelease
// marker. Local/build metadata (".local", "+build", "-vpn.local") does not count,
// so a project-local build is treated as equal to its numeric release.
func isPrerelease(v string) bool {
	lower := strings.ToLower(v)
	if strings.Contains(lower, ".local") || strings.Contains(lower, "+") {
		return false
	}
	for _, marker := range []string{"-dev", "-rc", "-alpha", "-beta", "-pre", "-snapshot"} {
		if strings.Contains(lower, marker) {
			return true
		}
	}
	return false
}

// parseVer extracts up to 3 numeric components from a version string.
func parseVer(s string) [3]int {
	s = strings.TrimPrefix(strings.TrimSpace(s), "v")
	if i := strings.IndexAny(s, "-+"); i >= 0 {
		s = s[:i]
	}
	var out [3]int
	for i, part := range strings.SplitN(s, ".", 3) {
		if i > 2 {
			break
		}
		n := 0
		fmt.Sscanf(part, "%d", &n)
		out[i] = n
	}
	return out
}

// downloadBinaryFromTarball fetches a release tarball and extracts the
// mazzy-vpn binary to a temp file, returning its path AND the full lowercase
// sha256 of the extracted binary so the caller can verify it against the
// release's published checksum (audit P2-2).
func downloadBinaryFromTarball(ctx context.Context, url string) (path, sum string, err error) {
	cctx, cancel := context.WithTimeout(ctx, 120*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(cctx, http.MethodGet, url, nil)
	if err != nil {
		return "", "", err
	}
	req.Header.Set("User-Agent", "mazzy-vpn-updater")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return "", "", fmt.Errorf("download returned %d", resp.StatusCode)
	}
	gz, err := gzip.NewReader(resp.Body)
	if err != nil {
		return "", "", err
	}
	defer gz.Close()
	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", "", err
		}
		if filepath.Base(hdr.Name) != "mazzy-vpn" || hdr.Typeflag != tar.TypeReg {
			continue
		}
		tmp, err := os.CreateTemp("", "mazzy-vpn-update-*")
		if err != nil {
			return "", "", err
		}
		h := sha256.New()
		if _, err := io.Copy(io.MultiWriter(tmp, h), tr); err != nil {
			_ = tmp.Close()
			_ = os.Remove(tmp.Name())
			return "", "", err
		}
		// A failed Close() after writing means the file may be truncated (e.g.
		// disk full): never return a half-written binary for installation.
		if err := tmp.Close(); err != nil {
			_ = os.Remove(tmp.Name())
			return "", "", fmt.Errorf("flush update binary: %w", err)
		}
		_ = os.Chmod(tmp.Name(), 0o755)
		full := hex.EncodeToString(h.Sum(nil))
		fmt.Printf("  sha256: %s\n", short(full))
		return tmp.Name(), full, nil
	}
	return "", "", fmt.Errorf("mazzy-vpn binary not found in tarball")
}

// short returns the first 16 hex chars of a checksum for compact display.
func short(sum string) string {
	if len(sum) > 16 {
		return sum[:16]
	}
	return sum
}

// expectedTarballSum fetches the release's SHA256SUMS asset (if any) and returns
// the checksum recorded for assetName. ok is false when no checksum asset is
// published or the asset is not listed, so the caller can decide policy. Lines
// are the standard `<hex>  <name>` sha256sum format.
func expectedTarballSum(ctx context.Context, rel *ghRelease, assetName string) (sum string, ok bool) {
	sumsURL := ""
	for _, a := range rel.Assets {
		if strings.EqualFold(a.Name, "SHA256SUMS") || strings.HasSuffix(a.Name, "SHA256SUMS") {
			sumsURL = a.BrowserDownloadURL
			break
		}
	}
	if sumsURL == "" {
		return "", false
	}
	cctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(cctx, http.MethodGet, sumsURL, nil)
	if err != nil {
		return "", false
	}
	req.Header.Set("User-Agent", "mazzy-vpn-updater")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", false
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return "", false
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return "", false
	}
	return matchSumLine(string(body), assetName)
}

// matchSumLine finds the sha256 recorded for assetName in a standard SHA256SUMS
// body (`<hex>  <name>` lines). The name column may carry a leading '*' (binary
// mode) or a path prefix. Extracted as a pure function so the parser is unit-
// testable without a network fetch.
func matchSumLine(body, assetName string) (sum string, ok bool) {
	for _, line := range strings.Split(body, "\n") {
		f := strings.Fields(line)
		if len(f) < 2 {
			continue
		}
		name := strings.TrimPrefix(f[len(f)-1], "*")
		if filepath.Base(name) == assetName {
			return strings.ToLower(f[0]), true
		}
	}
	return "", false
}

// replaceBinary atomically swaps the running binary with the new one, keeping a
// single, consistent rollback path (audit P2-3): the pre-update binary is moved
// aside to `<target>.old`; on any copy failure it is restored and the temp
// backup removed, so we never leave a half-installed binary OR an orphaned
// `.old`. Both the rename and the cross-device (copy) branch share this cleanup.
func replaceBinary(target, newBin string) error {
	// Basic sanity: the new binary must be a non-empty ELF-ish file.
	fi, err := os.Stat(newBin)
	if err != nil || fi.Size() < 1_000_000 {
		return fmt.Errorf("downloaded binary looks invalid")
	}
	backup := target + ".old"
	_ = os.Remove(backup)

	renamed := true
	if err := os.Rename(target, backup); err != nil {
		// Cross-device or busy: we cannot move the old binary aside, so there is
		// no backup to keep. Copy the new binary directly over the target.
		renamed = false
	}

	if err := copyFile(newBin, target); err != nil {
		if renamed {
			_ = os.Rename(backup, target) // restore the original from the backup
		}
		return err
	}
	if renamed {
		_ = os.Remove(backup) // success: drop the backup
	}
	return nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o755)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		_ = out.Close()
		return err
	}
	// fsync before close so the replaced binary is durably on disk; a crash
	// mid-update must not leave a truncated, unrunnable mazzy-vpn behind.
	if err := out.Sync(); err != nil {
		_ = out.Close()
		return err
	}
	return out.Close()
}
