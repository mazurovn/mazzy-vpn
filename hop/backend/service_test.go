// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright © 2026 Nik m (@mazurovn). All rights reserved.

package main

import (
	"bufio"
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func requireTool(t *testing.T, name string) string {
	t.Helper()
	p, err := exec.LookPath(name)
	if err != nil {
		t.Skip(name + " is not installed")
	}
	return p
}

func withRoot(root string) []string {
	env := make([]string, 0, len(os.Environ())+1)
	for _, e := range os.Environ() {
		if !strings.HasPrefix(e, "MAZZY_HOP_TEST_ROOT=") {
			env = append(env, e)
		}
	}
	return append(env, "MAZZY_HOP_TEST_ROOT="+root)
}

func exitCode(err error) int {
	if err == nil {
		return 0
	}
	var ee *exec.ExitError
	if errors.As(err, &ee) {
		return ee.ExitCode()
	}
	return -1
}

func runCmd(bin string, env []string, args ...string) (int, string) {
	cmd := exec.Command(bin, args...)
	cmd.Env = env
	out, err := cmd.CombinedOutput()
	return exitCode(err), string(out)
}

func mustBuild(t *testing.T, dir, name string, args ...string) {
	t.Helper()
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		s := string(out)
		if len(s) > 4000 {
			s = s[len(s)-4000:]
		}
		t.Fatalf("%s: %v\n%s", name, err, s)
	}
}

func findExe(root, name string) string {
	var found string
	_ = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		if d.Name() != name && d.Name() != name+".exe" {
			return nil
		}
		info, statErr := d.Info()
		if statErr == nil && info.Mode()&0o111 != 0 {
			found = path
		}
		return nil
	})
	return found
}

func write0600(t *testing.T, path string, data []byte) {
	t.Helper()
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(path, 0o600); err != nil {
		t.Fatal(err)
	}
}

func writeCert(t *testing.T, dir, name string, serial int64) (*x509.Certificate, string, string) {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	tmpl := &x509.Certificate{
		SerialNumber:          big.NewInt(serial),
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(2 * time.Hour),
		IPAddresses:           []net.IP{net.ParseIP("127.0.0.1")},
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		IsCA:                  true,
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	leaf, err := x509.ParseCertificate(der)
	if err != nil {
		t.Fatal(err)
	}
	keyDER, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		t.Fatal(err)
	}
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: keyDER})
	certPath := filepath.Join(dir, name+".pem")
	keyPath := filepath.Join(dir, name+".key")
	write0600(t, certPath, certPEM)
	write0600(t, keyPath, keyPEM)
	return leaf, certPath, keyPath
}

func freePort(t *testing.T) int {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	n := ln.Addr().(*net.TCPAddr).Port
	if err := ln.Close(); err != nil {
		t.Fatal(err)
	}
	return n
}

func loopback(port int) string {
	return net.JoinHostPort("127.0.0.1", strconv.Itoa(port))
}

func readTo(conn net.Conn, marker string) (string, error) {
	var buf []byte
	one := make([]byte, 1)
	for len(buf) < 8192 {
		if _, err := io.ReadFull(conn, one); err != nil {
			return string(buf), err
		}
		buf = append(buf, one[0])
		if strings.HasSuffix(string(buf), marker) {
			return string(buf), nil
		}
	}
	return string(buf), errors.New("preface too long")
}

func contentLength(head string) int {
	for _, line := range strings.Split(head, "\r\n") {
		if strings.HasPrefix(strings.ToLower(line), "content-length:") {
			n, _ := strconv.Atoi(strings.TrimSpace(line[len("content-length:"):]))
			return n
		}
	}
	return 0
}

func readResponse(conn net.Conn) (string, error) {
	head, err := readTo(conn, "\r\n\r\n")
	if err != nil {
		return head, err
	}
	n := contentLength(head)
	body := make([]byte, n)
	if n > 0 {
		if _, err := io.ReadFull(conn, body); err != nil {
			return head, err
		}
	}
	return head + string(body), nil
}

func dialHop(addr string, leaf *x509.Certificate) (*tls.Conn, error) {
	raw, err := net.DialTimeout("tcp", addr, 3*time.Second)
	if err != nil {
		return nil, err
	}
	pool := x509.NewCertPool()
	pool.AddCert(leaf)
	conn := tls.Client(raw, &tls.Config{
		MinVersion: tls.VersionTLS13,
		ServerName: "127.0.0.1",
		RootCAs:    pool,
	})
	_ = conn.SetDeadline(time.Now().Add(20 * time.Second))
	if err := conn.Handshake(); err != nil {
		_ = raw.Close()
		return nil, err
	}
	if conn.ConnectionState().Version != tls.VersionTLS13 {
		_ = conn.Close()
		return nil, fmt.Errorf("tls version %x", conn.ConnectionState().Version)
	}
	return conn, nil
}

func writeConnect(conn net.Conn, target, token string) error {
	_, err := fmt.Fprintf(conn, "CONNECT %s HTTP/1.1\r\nHost: %s\r\nProxy-Authorization: Basic %s\r\n\r\n", target, target, token)
	return err
}

func startCmd(t *testing.T, env []string, out, errb *bytes.Buffer, bin string, args ...string) *exec.Cmd {
	t.Helper()
	cmd := exec.Command(bin, args...)
	cmd.Env = env
	cmd.Stdout = out
	cmd.Stderr = errb
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
		}
		_ = cmd.Wait()
	})
	return cmd
}

func waitReady(path string, cmd *exec.Cmd, errb *bytes.Buffer) error {
	deadline := time.Now().Add(8 * time.Second)
	for time.Now().Before(deadline) {
		if cmd.ProcessState != nil && cmd.ProcessState.Exited() {
			return fmt.Errorf("exited %d: %s", cmd.ProcessState.ExitCode(), errb.String())
		}
		b, err := os.ReadFile(path)
		if err == nil && strings.Contains(string(b), "ready") {
			return nil
		}
		time.Sleep(20 * time.Millisecond)
	}
	return fmt.Errorf("not ready: %s", errb.String())
}

func waitBody(url, cred, needle string) (string, error) {
	client := &http.Client{Timeout: time.Second}
	deadline := time.Now().Add(8 * time.Second)
	var last error
	for time.Now().Before(deadline) {
		req, err := http.NewRequest(http.MethodGet, url, nil)
		if err != nil {
			return "", err
		}
		if cred != "" {
			req.Header.Set("Authorization", "Bearer "+cred)
		}
		resp, err := client.Do(req)
		if err != nil {
			last = err
			time.Sleep(30 * time.Millisecond)
			continue
		}
		b, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		if resp.StatusCode == http.StatusOK && strings.Contains(string(b), needle) {
			return string(b), nil
		}
		last = fmt.Errorf("status %d", resp.StatusCode)
		time.Sleep(30 * time.Millisecond)
	}
	return "", last
}

func TestHopService(t *testing.T) {
	nim := requireTool(t, "nim")
	moon := requireTool(t, "moon")
	mojo := requireTool(t, "mojo")
	hop := ".."
	bin := t.TempDir()
	root := t.TempDir()
	env := withRoot(root)

	mustBuild(t, filepath.Join(hop, "core"), nim, "c", "-d:ssl", "--nimcache:"+filepath.Join(bin, "nimcache"), "-o:"+filepath.Join(bin, "mazzy-core"), "--hints:off", "mazzy_core.nim")
	mustBuild(t, filepath.Join(hop, "toeac"), moon, "build", "--target", "native")
	mustBuild(t, filepath.Join(hop, "front"), moon, "build", "--target", "native")
	mustBuild(t, hop, mojo, "build", filepath.Join("protect", "protect.mojo"), "-o", filepath.Join(bin, "mazzy-hop-protect"))
	mustBuild(t, filepath.Join(hop, "backend"), "go", "build", "-o", filepath.Join(bin, "mazzy-backend"), ".")

	core := filepath.Join(bin, "mazzy-core")
	protect := filepath.Join(bin, "mazzy-hop-protect")
	backend := filepath.Join(bin, "mazzy-backend")
	toeac := findExe(filepath.Join(hop, "toeac", "_build"), "mazzy-toeac")
	front := findExe(filepath.Join(hop, "front", "_build"), "mazzy-front")
	if toeac == "" || front == "" {
		t.Fatal("language binaries were not produced")
	}

	var coreOut, coreErr bytes.Buffer
	var backOut, backErr bytes.Buffer
	var frontOut, frontErr bytes.Buffer
	const hopCred = "hop-credential-ok!!"
	const adminCred = "admin-credential-ok"
	defer func() {
		if !t.Failed() {
			return
		}
		for _, pair := range []struct{ name, text string }{
			{"core-out", coreOut.String()},
			{"core-err", coreErr.String()},
			{"back-err", backErr.String()},
			{"front-err", frontErr.String()},
		} {
			if strings.Contains(pair.text, hopCred) || strings.Contains(pair.text, adminCred) {
				t.Log(pair.name, "leaked a credential")
				continue
			}
			s := pair.text
			if len(s) > 2000 {
				s = s[len(s)-2000:]
			}
			t.Log(pair.name, s)
		}
		_ = backOut
		_ = frontOut
	}()

	code, text := runCmd(core, env, "trust-config", filepath.Join(root, "missing.toeac"))
	if code != 78 || strings.Contains(text, hopCred) {
		t.Fatalf("missing file %d", code)
	}

	loose := filepath.Join(root, "loose.toeac")
	if err := os.WriteFile(loose, []byte("transport nested-https\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(loose, 0o644); err != nil {
		t.Fatal(err)
	}
	if code, _ := runCmd(core, env, "trust-config", loose); code != 78 {
		t.Fatalf("mode %d", code)
	}
	real := filepath.Join(root, "real.toeac")
	write0600(t, real, []byte("transport nested-https\n"))
	link := filepath.Join(root, "link.toeac")
	if err := os.Symlink(real, link); err != nil {
		t.Fatal(err)
	}
	if code, _ := runCmd(core, env, "trust-config", link); code != 78 {
		t.Fatalf("symlink %d", code)
	}
	outside := filepath.Join(t.TempDir(), "outside.toeac")
	write0600(t, outside, []byte("transport nested-https\n"))
	if code, _ := runCmd(core, env, "trust-config", outside); code != 78 {
		t.Fatalf("outside %d", code)
	}
	if code, _ := runCmd(core, env, "trust-config", real); code != 0 {
		t.Fatalf("trusted file %d", code)
	}

	example, err := os.ReadFile(filepath.Join(hop, "example.toeac"))
	if err != nil {
		t.Fatal(err)
	}
	placeholder := filepath.Join(root, "placeholder.toeac")
	write0600(t, placeholder, example)
	code, text = runCmd(core, env, "serve", "--config", placeholder, "--toeac", toeac, "--protect", protect)
	if code != 78 || !strings.Contains(text, "placeholder") {
		t.Fatalf("placeholder %d %q", code, text)
	}

	hopLeaf, hopCert, hopKey := writeCert(t, root, "hop", 1)
	upLeaf, upCert, upKey := writeCert(t, root, "up", 2)
	upTLS, err := tls.LoadX509KeyPair(upCert, upKey)
	if err != nil {
		t.Fatal(err)
	}
	upLn, err := tls.Listen("tcp", "127.0.0.1:0", &tls.Config{
		Certificates: []tls.Certificate{upTLS},
		MinVersion:   tls.VersionTLS13,
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = upLn.Close() })
	var upHits atomic.Int32
	go func() {
		for {
			c, err := upLn.Accept()
			if err != nil {
				return
			}
			upHits.Add(1)
			go func(c net.Conn) {
				defer c.Close()
				buf := make([]byte, 4)
				_, _ = io.ReadFull(c, buf)
				if string(buf) == "ping" {
					_, _ = c.Write([]byte("pong"))
				}
			}(c)
		}
	}()

	black, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = black.Close() })
	var blackHits atomic.Int32
	go func() {
		for {
			c, err := black.Accept()
			if err != nil {
				return
			}
			blackHits.Add(1)
			_ = c.Close()
		}
	}()

	hopPort := freePort(t)
	adminPort := freePort(t)
	frontPort := freePort(t)
	mcpPath := filepath.Join(root, "mcp.sock")
	statusPath := filepath.Join(root, "status.sock")
	cfgPath := filepath.Join(root, "hop.toeac")
	ready := filepath.Join(root, "ready")
	program := fmt.Sprintf(`transport nested-https

server {
  bind %q
  cert %q
  key %q
}

session {
  idle 30s
  max 8
}

doctor {
  every 60s
  claim-clean no
}

protect {
  credential %q
  allow %q
  require-auth yes
}

artifact {
  process "mazzy-core"
  owns mcp
}

artifact {
  process "mazzy-backend"
  owns admin
}

artifact {
  process "mazzy-front"
  owns a2a
}

logic {
  relay copy
  protect mojo
  doctor nim
}

mcp {
  socket %q
}

admin {
  bind %q
  credential %q
  status %q
}

a2a {
  bind %q
  admin %q
}
`, loopback(hopPort), hopCert, hopKey, hopCred, upLn.Addr().String(), mcpPath, loopback(adminPort), adminCred, statusPath, loopback(frontPort), "http://"+loopback(adminPort))
	write0600(t, cfgPath, []byte(program))

	coreCmd := startCmd(t, env, &coreOut, &coreErr, core, "serve", "--config", cfgPath, "--toeac", toeac, "--protect", protect, "--ready", ready)
	if err := waitReady(ready, coreCmd, &coreErr); err != nil {
		t.Fatal(err)
	}

	bad, err := dialHop(loopback(hopPort), hopLeaf)
	if err != nil {
		t.Fatal(err)
	}
	if err := writeConnect(bad, upLn.Addr().String(), base64.StdEncoding.EncodeToString([]byte("hop:wrong-password-value"))); err != nil {
		t.Fatal(err)
	}
	badBody, err := readResponse(bad)
	_ = bad.Close()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(badBody, "<title>ok</title>") || strings.Contains(badBody, "Connection Established") {
		t.Fatal("bad auth was not served the ordinary page")
	}
	time.Sleep(50 * time.Millisecond)
	if upHits.Load() != 0 || blackHits.Load() != 0 {
		t.Fatalf("probe dialed upstream=%d other=%d", upHits.Load(), blackHits.Load())
	}

	denied, err := dialHop(loopback(hopPort), hopLeaf)
	if err != nil {
		t.Fatal(err)
	}
	goodToken := base64.StdEncoding.EncodeToString([]byte("hop:" + hopCred))
	if err := writeConnect(denied, black.Addr().String(), goodToken); err != nil {
		t.Fatal(err)
	}
	deniedBody, err := readResponse(denied)
	_ = denied.Close()
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(deniedBody, "Connection Established") {
		t.Fatal("disallowed target was opened")
	}
	time.Sleep(50 * time.Millisecond)
	if blackHits.Load() != 0 {
		t.Fatal("disallowed target was dialed")
	}

	good, err := dialHop(loopback(hopPort), hopLeaf)
	if err != nil {
		t.Fatal(err)
	}
	defer good.Close()
	if err := writeConnect(good, upLn.Addr().String(), goodToken); err != nil {
		t.Fatal(err)
	}
	preface, err := readTo(good, "\r\n\r\n")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(preface, "200 Connection Established") {
		t.Fatal("authenticated session was not established")
	}
	pool := x509.NewCertPool()
	pool.AddCert(upLeaf)
	inner := tls.Client(good, &tls.Config{
		MinVersion: tls.VersionTLS13,
		ServerName: "127.0.0.1",
		RootCAs:    pool,
	})
	if err := inner.Handshake(); err != nil {
		t.Fatal(err)
	}
	if len(inner.ConnectionState().PeerCertificates) == 0 || !inner.ConnectionState().PeerCertificates[0].Equal(upLeaf) {
		t.Fatal("inner tls was not the upstream certificate")
	}
	if _, err := inner.Write([]byte("ping")); err != nil {
		t.Fatal(err)
	}
	pong := make([]byte, 4)
	if _, err := io.ReadFull(inner, pong); err != nil {
		t.Fatal(err)
	}
	if string(pong) != "pong" {
		t.Fatalf("relayed %q", pong)
	}
	_ = inner.Close()
	if upHits.Load() != 1 {
		t.Fatalf("upstream hits %d", upHits.Load())
	}

	mcp, err := net.DialTimeout("unix", mcpPath, 2*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer mcp.Close()
	_ = mcp.SetDeadline(time.Now().Add(3 * time.Second))
	mcpIn := bufio.NewReader(mcp)
	if _, err := io.WriteString(mcp, "{\"jsonrpc\":\"2.0\",\"id\":1,\"method\":\"initialize\"}\n"); err != nil {
		t.Fatal(err)
	}
	initLine, err := mcpIn.ReadString('\n')
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(initLine, "mazzy-core") || !strings.Contains(initLine, "2025-06-18") {
		t.Fatalf("initialize %s", initLine)
	}
	if _, err := io.WriteString(mcp, "{\"jsonrpc\":\"2.0\",\"id\":2,\"method\":\"tools/call\",\"params\":{\"name\":\"doctor\"}}\n"); err != nil {
		t.Fatal(err)
	}
	toolLine, err := mcpIn.ReadString('\n')
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(toolLine, "egress_clean false") || strings.Contains(toolLine, "egress_clean true") {
		t.Fatalf("mcp doctor %s", toolLine)
	}

	_ = startCmd(t, env, &backOut, &backErr, backend, "--config", cfgPath, "--trust", core, "--toeac", toeac)
	health, err := waitBody("http://"+loopback(adminPort)+"/v1/health", adminCred, `"egress_clean":false`)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(health, `"egress_clean":true`) || strings.Contains(health, hopCred) || strings.Contains(health, adminCred) {
		t.Fatal("admin health leaked or claimed clean")
	}
	anon, err := http.Get("http://" + loopback(adminPort) + "/v1/doctor")
	if err != nil {
		t.Fatal(err)
	}
	_ = anon.Body.Close()
	if anon.StatusCode != http.StatusUnauthorized {
		t.Fatalf("admin anon %d", anon.StatusCode)
	}

	_ = startCmd(t, env, &frontOut, &frontErr, front, "--config", cfgPath, "--trust", core, "--toeac", toeac)
	page, err := waitBody("http://"+loopback(frontPort)+"/", adminCred, "trusted")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(page, "not claimed clean") || strings.Contains(page, hopCred) {
		t.Fatal("status page")
	}
	anonFront, err := http.Get("http://" + loopback(frontPort) + "/")
	if err != nil {
		t.Fatal(err)
	}
	_ = anonFront.Body.Close()
	if anonFront.StatusCode != http.StatusUnauthorized {
		t.Fatalf("front anon %d", anonFront.StatusCode)
	}

	cardReq, err := http.NewRequest(http.MethodGet, "http://"+loopback(frontPort)+"/.well-known/agent-card.json", nil)
	if err != nil {
		t.Fatal(err)
	}
	cardReq.Header.Set("Authorization", "Bearer "+adminCred)
	cardResp, err := http.DefaultClient.Do(cardReq)
	if err != nil {
		t.Fatal(err)
	}
	cardBody, _ := io.ReadAll(cardResp.Body)
	_ = cardResp.Body.Close()
	if cardResp.StatusCode != http.StatusOK || !strings.Contains(string(cardBody), "mazzy-front") || strings.Contains(string(cardBody), "http") {
		t.Fatalf("agent card %d", cardResp.StatusCode)
	}

	a2aReq, err := http.NewRequest(http.MethodPost, "http://"+loopback(frontPort)+"/a2a", strings.NewReader(`{"jsonrpc":"2.0","id":4,"method":"message/send","params":{"message":{"parts":[{"kind":"text","text":"doctor"}]}}}`))
	if err != nil {
		t.Fatal(err)
	}
	a2aReq.Header.Set("Authorization", "Bearer "+adminCred)
	a2aResp, err := http.DefaultClient.Do(a2aReq)
	if err != nil {
		t.Fatal(err)
	}
	a2aBody, _ := io.ReadAll(a2aResp.Body)
	_ = a2aResp.Body.Close()
	if a2aResp.StatusCode != http.StatusOK || !strings.Contains(string(a2aBody), `"id":4`) || !strings.Contains(string(a2aBody), "trusted") || strings.Contains(string(a2aBody), `"egress_clean":true`) {
		t.Fatalf("a2a %d", a2aResp.StatusCode)
	}
	if strings.Contains(coreOut.String(), hopCred) || strings.Contains(coreErr.String(), hopCred) {
		t.Fatal("core log leaked a credential")
	}
}
