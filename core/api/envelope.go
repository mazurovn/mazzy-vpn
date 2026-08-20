// SPDX-License-Identifier: PolyForm-Noncommercial-1.0.0
// Copyright © 2026 Nik m (@mazurovn). All rights reserved.

// Package api provides the machine-first JSON control API for mazzy-core. It is
// the AI-native surface: agents and harnesses drive the VPN programmatically
// through a stable, versioned envelope instead of parsing human output.
//
// It replaces the bash socat+jq API with native Go (encoding/json, net). The
// envelope shape preserves parity with the v1 contract:
//
//	{ "api_version": "1.0", "request_id": "...", "status": "ok",    "result": {...} }
//	{ "api_version": "1.0", "request_id": "...", "status": "error", "error":  {...} }
//
// Operations are split into read-only (no authorization) and mutations (must
// carry action_id + authorization), mirroring the bash dispatch.
package api

import (
	"encoding/json"
	"fmt"
)

// Version is the API schema version.
const Version = "1.0"

// Status is the envelope status.
type Status string

const (
	StatusOK    Status = "ok"
	StatusError Status = "error"
)

// Envelope is the top-level response.
type Envelope struct {
	APIVersion string          `json:"api_version"`
	RequestID  string          `json:"request_id"`
	Status     Status          `json:"status"`
	Result     json.RawMessage `json:"result,omitempty"`
	Error      *APIError       `json:"error,omitempty"`
}

// APIError is a structured, localizable error. message_key is a stable key the
// client maps to a localized string; no host paths or secrets are exposed.
type APIError struct {
	Code               string `json:"code"`
	MessageKey         string `json:"message_key"`
	Retryable          bool   `json:"retryable"`
	UserActionRequired bool   `json:"user_action_required"`
}

// Error codes, parity with bash.
const (
	CodeInvalidRequest     = "invalid-request"
	CodePermissionDenied   = "permission-denied"
	CodeBackendUnavailable = "backend-unavailable"
	CodeUnsupportedVersion = "unsupported-version"
	CodeConflict           = "conflict"
	CodeInternal           = "internal"
)

// OK builds a success envelope from an already-marshaled result.
func OK(requestID string, result any) (*Envelope, error) {
	raw, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("marshal result: %w", err)
	}
	return &Envelope{
		APIVersion: Version,
		RequestID:  requestID,
		Status:     StatusOK,
		Result:     raw,
	}, nil
}

// Err builds an error envelope.
func Err(requestID, code, messageKey string, retryable, userAction bool) *Envelope {
	return &Envelope{
		APIVersion: Version,
		RequestID:  requestID,
		Status:     StatusError,
		Error: &APIError{
			Code:               code,
			MessageKey:         messageKey,
			Retryable:          retryable,
			UserActionRequired: userAction,
		},
	}
}

// Marshal renders the envelope as compact JSON.
func (e *Envelope) Marshal() []byte {
	b, _ := json.Marshal(e)
	return b
}
