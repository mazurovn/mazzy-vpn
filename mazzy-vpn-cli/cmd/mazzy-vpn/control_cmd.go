// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright © 2026 Nik m (@mazurovn). All rights reserved.

package main

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/mazurovn/mazzy-vpn/core/control"
)

// identityFile is where this node's control-plane identity is stored.
func identityFile() string {
	if d := os.Getenv("MAZZY_CONFIG_HOME"); d != "" {
		return filepath.Join(d, "identity.json")
	}
	if h, err := os.UserConfigDir(); err == nil {
		return filepath.Join(h, "mazzy-vpn", "identity.json")
	}
	return filepath.Join(os.TempDir(), "mazzy-vpn", "identity.json")
}

// storedIdentity is the on-disk form (private key kept 0600).
type storedIdentity struct {
	ID      string `json:"id"`
	PrivKey string `json:"priv_key"` // base64 ed25519 private key
}

// cmdControl manages the node's control-plane identity and trust:
//
//	mazzy-vpn control id                 show this node's identity (ID + pubkey)
//	mazzy-vpn control pair <ID> <PUBKEY> trust a participant
//	mazzy-vpn control list               list trusted participants
func cmdControl(_ context.Context, args []string) int {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "usage: mazzy-vpn control id|pair <ID> <PUBKEY_B64>|list")
		return 2
	}
	switch args[0] {
	case "id":
		return controlShowID()
	case "pair":
		if len(args) < 3 {
			fmt.Fprintln(os.Stderr, "usage: mazzy-vpn control pair <ID> <PUBKEY_B64>")
			return 2
		}
		return controlPair(args[1], args[2])
	case "list":
		return controlList()
	default:
		fmt.Fprintf(os.Stderr, "unknown control subcommand: %s\n", args[0])
		return 2
	}
}

// nodeKeypair persists and returns a stable ed25519 keypair for this node.
func nodeKeypair() (ed25519.PublicKey, ed25519.PrivateKey, string, error) {
	p := identityFile()
	if data, err := os.ReadFile(p); err == nil {
		var s storedIdentity
		if json.Unmarshal(data, &s) == nil {
			if raw, err := base64.StdEncoding.DecodeString(s.PrivKey); err == nil && len(raw) == ed25519.PrivateKeySize {
				priv := ed25519.PrivateKey(raw)
				pub := priv.Public().(ed25519.PublicKey)
				return pub, priv, s.ID, nil
			}
		}
	}
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, nil, "", err
	}
	id := control.DeriveID(pub)
	if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
		return nil, nil, "", err
	}
	s := storedIdentity{ID: id, PrivKey: base64.StdEncoding.EncodeToString(priv)}
	data, _ := json.MarshalIndent(s, "", "  ")
	tmp := p + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return nil, nil, "", err
	}
	if err := os.Rename(tmp, p); err != nil {
		return nil, nil, "", err
	}
	return pub, priv, id, nil
}

func controlShowID() int {
	pub, _, id, err := nodeKeypair()
	if err != nil {
		fmt.Fprintln(os.Stderr, "identity error:", err)
		return 1
	}
	fmt.Printf("ID:      %s\n", id)
	fmt.Printf("PubKey:  %s\n", base64.StdEncoding.EncodeToString(pub))
	fmt.Println("\nShare these with a peer so they can 'mazzy-vpn control pair' with you.")
	return 0
}

// trustFile stores trusted participants.
func trustFile() string { return filepath.Join(filepath.Dir(identityFile()), "trust.json") }

type trustRecord struct {
	ID     string `json:"id"`
	PubKey string `json:"pub_key"`
}

func controlPair(id, pubB64 string) int {
	pub, err := base64.StdEncoding.DecodeString(pubB64)
	if err != nil || len(pub) != ed25519.PublicKeySize {
		fmt.Fprintln(os.Stderr, "invalid public key (expected base64 ed25519)")
		return 1
	}
	if !control.VerifyID(id, ed25519.PublicKey(pub)) {
		fmt.Fprintln(os.Stderr, "refused: public key does not match the claimed ID (impersonation)")
		return 1
	}
	var recs []trustRecord
	if data, err := os.ReadFile(trustFile()); err == nil {
		_ = json.Unmarshal(data, &recs)
	}
	for _, r := range recs {
		if r.ID == id {
			fmt.Println("already paired:", id)
			return 0
		}
	}
	recs = append(recs, trustRecord{ID: id, PubKey: pubB64})
	_ = os.MkdirAll(filepath.Dir(trustFile()), 0o700)
	data, _ := json.MarshalIndent(recs, "", "  ")
	if err := os.WriteFile(trustFile(), data, 0o600); err != nil {
		fmt.Fprintln(os.Stderr, "save trust:", err)
		return 1
	}
	fmt.Printf("✔ paired with %s\n", id)
	return 0
}

func controlList() int {
	data, err := os.ReadFile(trustFile())
	if err != nil {
		fmt.Println("No paired participants.")
		return 0
	}
	var recs []trustRecord
	_ = json.Unmarshal(data, &recs)
	if len(recs) == 0 {
		fmt.Println("No paired participants.")
		return 0
	}
	fmt.Printf("%-18s %s\n", "ID", "TRUST")
	for _, r := range recs {
		ok := control.VerifyID(r.ID, mustKey(r.PubKey))
		status := "paired"
		if !ok {
			status = "INVALID KEY"
		}
		fmt.Printf("%-18s %s\n", r.ID, status)
	}
	return 0
}

func mustKey(b64 string) ed25519.PublicKey {
	raw, _ := base64.StdEncoding.DecodeString(b64)
	return ed25519.PublicKey(raw)
}
