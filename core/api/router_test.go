// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright © 2026 Nik m (@mazurovn). All rights reserved.

package api

import (
	"context"
	"encoding/json"
	"testing"
)

func testRouter() *Router {
	r := NewRouter(func() string { return "req-1" })
	r.Handle("status.get", func(_ context.Context, _ *Request) (any, error) {
		return map[string]string{"state": "down"}, nil
	})
	r.Handle("connect", func(_ context.Context, req *Request) (any, error) {
		return map[string]string{"accepted": req.ActionID}, nil
	})
	return r
}

func decode(t *testing.T, e *Envelope) map[string]any {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal(e.Marshal(), &m); err != nil {
		t.Fatalf("unmarshal envelope: %v", err)
	}
	return m
}

func TestReadOnlyOKEnvelope(t *testing.T) {
	r := testRouter()
	e := r.Dispatch(context.Background(), []byte(`{"api_version":"1.0","operation":"status.get"}`))
	m := decode(t, e)
	if m["status"] != "ok" || m["api_version"] != "1.0" || m["request_id"] != "req-1" {
		t.Fatalf("bad envelope: %v", m)
	}
	if _, hasErr := m["error"]; hasErr {
		t.Error("ok envelope must not carry error")
	}
}

func TestUnsupportedVersion(t *testing.T) {
	r := testRouter()
	e := r.Dispatch(context.Background(), []byte(`{"api_version":"9.9","operation":"status.get"}`))
	m := decode(t, e)
	if m["status"] != "error" {
		t.Fatal("expected error for bad version")
	}
	errObj := m["error"].(map[string]any)
	if errObj["code"] != CodeUnsupportedVersion {
		t.Errorf("code = %v, want %s", errObj["code"], CodeUnsupportedVersion)
	}
}

func TestReadOnlyRejectsCredentials(t *testing.T) {
	r := testRouter()
	e := r.Dispatch(context.Background(),
		[]byte(`{"api_version":"1.0","operation":"status.get","action_id":"x","authorization":"y"}`))
	m := decode(t, e)
	if m["status"] != "error" {
		t.Fatal("read-only op with credentials must be rejected")
	}
}

func TestMutationRequiresAuthorization(t *testing.T) {
	r := testRouter()
	e := r.Dispatch(context.Background(), []byte(`{"api_version":"1.0","operation":"connect"}`))
	m := decode(t, e)
	errObj := m["error"].(map[string]any)
	if errObj["code"] != CodePermissionDenied {
		t.Fatalf("mutation without auth must be permission-denied, got %v", errObj["code"])
	}
	if errObj["user_action_required"] != true {
		t.Error("expected user_action_required=true")
	}
}

func TestMutationWithAuthSucceeds(t *testing.T) {
	r := testRouter()
	e := r.Dispatch(context.Background(),
		[]byte(`{"api_version":"1.0","operation":"connect","action_id":"a1","authorization":"tok"}`))
	m := decode(t, e)
	if m["status"] != "ok" {
		t.Fatalf("authorized mutation should succeed: %v", m)
	}
	res := m["result"].(map[string]any)
	if res["accepted"] != "a1" {
		t.Errorf("handler did not receive action_id: %v", res)
	}
}

func TestUnknownOperation(t *testing.T) {
	r := testRouter()
	e := r.Dispatch(context.Background(), []byte(`{"api_version":"1.0","operation":"nope"}`))
	m := decode(t, e)
	if m["status"] != "error" {
		t.Fatal("unknown op must error")
	}
}

// TestDuplicateKeyRejected locks the request-smuggling defense: a duplicate
// top-level key (e.g. two "operation"s) must be rejected, not silently
// last-wins parsed.
func TestDuplicateKeyRejected(t *testing.T) {
	r := testRouter()
	raw := []byte(`{"api_version":"1.0","operation":"status.get","operation":"connect"}`)
	e := r.Dispatch(context.Background(), raw)
	m := decode(t, e)
	if m["status"] != "error" {
		t.Fatal("duplicate operation key must be rejected")
	}
	errObj := m["error"].(map[string]any)
	if errObj["code"] != CodeInvalidRequest {
		t.Errorf("code = %v, want %s", errObj["code"], CodeInvalidRequest)
	}
}

func TestMalformedJSON(t *testing.T) {
	r := testRouter()
	e := r.Dispatch(context.Background(), []byte(`{not json`))
	m := decode(t, e)
	if m["status"] != "error" {
		t.Fatal("malformed JSON must error")
	}
}

func TestUnknownFieldRejected(t *testing.T) {
	r := testRouter()
	e := r.Dispatch(context.Background(),
		[]byte(`{"api_version":"1.0","operation":"status.get","evil":"payload"}`))
	m := decode(t, e)
	if m["status"] != "error" {
		t.Fatal("unknown field must be rejected (DisallowUnknownFields)")
	}
}
