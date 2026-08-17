// SPDX-License-Identifier: AGPL-3.0-or-later
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
	"path/filepath"
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
func cmdUpdate(ctx context.Context, args []string) int {
	apply := hasFlag(args, "--apply")

	fmt.Printf("Current version: %s\n", version)
	fmt.Printf("Checking %s for updates...\n", updateRepo)

	rel, err := latestRelease(ctx)
	if err != nil {
		fmt.Fprintln(os.Stderr, "update check failed:", err)
		return 1
	}
	fmt.Printf("Latest release : %s\n", rel.TagName)

	if !isNewer(rel.TagName, version) {
		fmt.Println("You are up to date.")
		return 0
	}
	fmt.Printf("A newer version is available: %s\n", rel.TagName)

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

	if !apply {
		fmt.Println("\nRun 'sudo mazzy-vpn update --apply' to download and install it.")
		return 0
	}

	if os.Geteuid() != 0 {
		fmt.Fprintln(os.Stderr, "installing an update needs root: sudo mazzy-vpn update --apply")
		return 1
	}

	fmt.Printf("Downloading %s...\n", assetName)
	bin, err := downloadBinaryFromTarball(ctx, asset)
	if err != nil {
		fmt.Fprintln(os.Stderr, "download failed:", err)
		return 1
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
	fmt.Printf("✔ Updated to %s. Restart mazzy-vpn to use it.\n", rel.TagName)
	return 0
}

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
// "2.0.0-dev"). It compares dotted numeric components; a -dev/-rc suffix on cur
// is treated as older than the same release.
func isNewer(tag, cur string) bool {
	tv := parseVer(tag)
	cv := parseVer(cur)
	for i := 0; i < 3; i++ {
		if tv[i] != cv[i] {
			return tv[i] > cv[i]
		}
	}
	// Equal numeric: a "-dev"/"-rc" current is older than a clean tag.
	return strings.ContainsAny(cur, "-") && !strings.ContainsAny(tag, "-")
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
// mazzy-vpn binary to a temp file, returning its path.
func downloadBinaryFromTarball(ctx context.Context, url string) (string, error) {
	cctx, cancel := context.WithTimeout(ctx, 120*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(cctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "mazzy-vpn-updater")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("download returned %d", resp.StatusCode)
	}
	gz, err := gzip.NewReader(resp.Body)
	if err != nil {
		return "", err
	}
	defer gz.Close()
	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", err
		}
		if filepath.Base(hdr.Name) != "mazzy-vpn" || hdr.Typeflag != tar.TypeReg {
			continue
		}
		tmp, err := os.CreateTemp("", "mazzy-vpn-update-*")
		if err != nil {
			return "", err
		}
		h := sha256.New()
		if _, err := io.Copy(io.MultiWriter(tmp, h), tr); err != nil {
			tmp.Close()
			os.Remove(tmp.Name())
			return "", err
		}
		tmp.Close()
		_ = os.Chmod(tmp.Name(), 0o755)
		fmt.Printf("  sha256: %s\n", hex.EncodeToString(h.Sum(nil))[:16])
		return tmp.Name(), nil
	}
	return "", fmt.Errorf("mazzy-vpn binary not found in tarball")
}

// replaceBinary atomically swaps the running binary with the new one.
func replaceBinary(target, newBin string) error {
	// Basic sanity: the new binary must be a non-empty ELF-ish file.
	fi, err := os.Stat(newBin)
	if err != nil || fi.Size() < 1_000_000 {
		return fmt.Errorf("downloaded binary looks invalid")
	}
	backup := target + ".old"
	_ = os.Remove(backup)
	if err := os.Rename(target, backup); err != nil {
		// Cross-device or busy: try copy instead.
		if err2 := copyFile(newBin, target); err2 != nil {
			return err2
		}
		return nil
	}
	if err := copyFile(newBin, target); err != nil {
		_ = os.Rename(backup, target) // rollback
		return err
	}
	_ = os.Remove(backup)
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
		out.Close()
		return err
	}
	return out.Close()
}
