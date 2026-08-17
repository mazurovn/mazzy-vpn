// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright © 2026 Nik m (@mazurovn). All rights reserved.

package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
)

// Request is a parsed API request envelope.
type Request struct {
	APIVersion    string          `json:"api_version"`
	Operation     string          `json:"operation"`
	ActionID      string          `json:"action_id,omitempty"`
	Authorization string          `json:"authorization,omitempty"`
	DeadlineMS    int64           `json:"deadline_ms,omitempty"`
	Params        json.RawMessage `json:"params,omitempty"`
}

// readOnlyOps require no authorization. Parity with the bash allowlist
// (api.capabilities|status.get|profiles.list|protocols.list) plus probes.
var readOnlyOps = map[string]bool{
	"api.capabilities": true,
	"status.get":       true,
	"profiles.list":    true,
	"protocols.list":   true,
	"doctor.run":       true,
}

// mutationOps change state and MUST carry action_id + authorization.
var mutationOps = map[string]bool{
	"connect":    true,
	"disconnect": true,
	"reconnect":  true,
	"quick":      true,
}

// Handler implements one operation. It receives parsed params (may be nil) and
// returns a result to be wrapped in an OK envelope, or an error.
type Handler func(ctx context.Context, req *Request) (any, error)

// HandlerError lets a handler control the error envelope precisely.
type HandlerError struct {
	Code       string
	MessageKey string
	Retryable  bool
	UserAction bool
}

func (e *HandlerError) Error() string { return e.Code + ":" + e.MessageKey }

// Router dispatches requests to handlers with the read-only/mutation policy.
type Router struct {
	handlers map[string]Handler
	// RequestID generates a server-side request id.
	RequestID func() string
}

// NewRouter creates a Router.
func NewRouter(requestID func() string) *Router {
	return &Router{handlers: map[string]Handler{}, RequestID: requestID}
}

// Handle registers a handler for an operation.
func (r *Router) Handle(op string, h Handler) { r.handlers[op] = h }

// Dispatch parses raw JSON, enforces schema/version/authorization policy, and
// invokes the handler. It always returns a well-formed Envelope.
func (r *Router) Dispatch(ctx context.Context, raw []byte) *Envelope {
	rid := r.reqID()

	// Reject duplicate top-level keys (a real request-smuggling defense from
	// the bash api_request_has_unique_schema_keys).
	if err := rejectDuplicateKeys(raw); err != nil {
		return Err(rid, CodeInvalidRequest, "api.request.malformed", false, false)
	}

	var req Request
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		return Err(rid, CodeInvalidRequest, "api.request.malformed", false, false)
	}

	if req.APIVersion != Version {
		return Err(rid, CodeUnsupportedVersion, "api.version.unsupported", false, true)
	}
	if req.Operation == "" {
		return Err(rid, CodeInvalidRequest, "api.operation.missing", false, false)
	}

	isRead := readOnlyOps[req.Operation]
	isMutation := mutationOps[req.Operation]
	if !isRead && !isMutation {
		return Err(rid, CodeInvalidRequest, "api.operation.unknown", false, false)
	}

	// Read-only ops must NOT carry mutation credentials; mutations MUST.
	if isRead && (req.ActionID != "" || req.Authorization != "") {
		return Err(rid, CodeInvalidRequest, "api.readonly.credentials_forbidden", false, false)
	}
	if isMutation && (req.ActionID == "" || req.Authorization == "") {
		return Err(rid, CodePermissionDenied, "api.mutation.authorization_required", false, true)
	}

	h, ok := r.handlers[req.Operation]
	if !ok {
		return Err(rid, CodeBackendUnavailable, "api.operation.unimplemented", true, false)
	}

	result, err := h(ctx, &req)
	if err != nil {
		if he, ok := err.(*HandlerError); ok {
			return Err(rid, he.Code, he.MessageKey, he.Retryable, he.UserAction)
		}
		return Err(rid, CodeInternal, "api.internal", false, false)
	}
	env, err := OK(rid, result)
	if err != nil {
		return Err(rid, CodeInternal, "api.internal", false, false)
	}
	return env
}

func (r *Router) reqID() string {
	if r.RequestID != nil {
		return r.RequestID()
	}
	return "req"
}

// rejectDuplicateKeys walks the top-level object and fails if any key repeats.
// encoding/json silently takes the last value for duplicates, which can be used
// to smuggle a different operation past a validator; we reject outright.
func rejectDuplicateKeys(raw []byte) error {
	dec := json.NewDecoder(bytes.NewReader(raw))
	tok, err := dec.Token()
	if err != nil {
		return err
	}
	delim, ok := tok.(json.Delim)
	if !ok || delim != '{' {
		return fmt.Errorf("request must be a JSON object")
	}
	seen := map[string]bool{}
	depth := 0
	for dec.More() || depth > 0 {
		t, err := dec.Token()
		if err != nil {
			return err
		}
		switch d := t.(type) {
		case json.Delim:
			switch d {
			case '{', '[':
				depth++
			case '}', ']':
				depth--
			}
			continue
		case string:
			if depth == 0 {
				if seen[d] {
					return fmt.Errorf("duplicate key %q", d)
				}
				seen[d] = true
				// consume the value token(s)
				if err := skipValue(dec); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

// skipValue consumes a single JSON value (scalar or nested) from dec.
func skipValue(dec *json.Decoder) error {
	t, err := dec.Token()
	if err != nil {
		return err
	}
	if delim, ok := t.(json.Delim); ok && (delim == '{' || delim == '[') {
		depth := 1
		for depth > 0 {
			tt, err := dec.Token()
			if err != nil {
				return err
			}
			if dd, ok := tt.(json.Delim); ok {
				if dd == '{' || dd == '[' {
					depth++
				} else {
					depth--
				}
			}
		}
	}
	return nil
}
