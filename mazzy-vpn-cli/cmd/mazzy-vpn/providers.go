// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright © 2026 Nik m (@mazurovn). All rights reserved.

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sort"
	"sync"
	"time"
)

// aiProvider is a well-known AI provider/agent endpoint to check reachability.
type aiProvider struct {
	Name string
	URL  string
	Type string // "llm", "agent", "search"
}

// aiProviders is the built-in list of AI services to diagnose.
var aiProviders = []aiProvider{
	{"OpenAI API", "https://api.openai.com/v1/models", "llm"},
	{"ChatGPT", "https://chatgpt.com/", "llm"},
	{"Anthropic API", "https://api.anthropic.com/v1/models", "llm"},
	{"Claude", "https://claude.ai/", "llm"},
	{"Google Gemini", "https://generativelanguage.googleapis.com/", "llm"},
	{"Google AI Studio", "https://aistudio.google.com/", "llm"},
	{"NotebookLM", "https://notebooklm.google.com/", "agent"},
	{"Perplexity", "https://www.perplexity.ai/", "search"},
	{"Mistral API", "https://api.mistral.ai/v1/models", "llm"},
	{"Groq API", "https://api.groq.com/", "llm"},
	{"OpenRouter", "https://openrouter.ai/api/v1/models", "llm"},
	{"HuggingFace", "https://huggingface.co/", "llm"},
}

// providerResult is one reachability check.
type providerResult struct {
	Name      string `json:"name"`
	Type      string `json:"type"`
	Reachable bool   `json:"reachable"`
	Status    int    `json:"http_status,omitempty"`
	LatencyMS int64  `json:"latency_ms"`
	Err       string `json:"error,omitempty"`
}

// checkProvider probes one provider with a bounded HEAD/GET request.
func checkProvider(ctx context.Context, p aiProvider, timeout time.Duration) providerResult {
	res := providerResult{Name: p.Name, Type: p.Type}
	cctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	client := &http.Client{Timeout: timeout}
	start := time.Now()
	req, err := http.NewRequestWithContext(cctx, http.MethodGet, p.URL, nil)
	if err != nil {
		res.Err = err.Error()
		return res
	}
	req.Header.Set("User-Agent", "mazzy-vpn/diagnostics")
	resp, err := client.Do(req)
	res.LatencyMS = time.Since(start).Milliseconds()
	if err != nil {
		res.Err = shortErr(err.Error())
		return res
	}
	defer resp.Body.Close()
	res.Status = resp.StatusCode
	// Any HTTP response (even 401/403) proves the endpoint is REACHABLE; only a
	// transport error means blocked/unreachable.
	res.Reachable = true
	return res
}

// shortErr trims noisy transport error text.
func shortErr(s string) string {
	if len(s) > 80 {
		return s[:80] + "…"
	}
	return s
}

// cmdProviders checks AI provider/agent reachability from the current egress,
// with optional --type filtering (llm/agent/search) and --json output.
func cmdProviders(ctx context.Context, args []string) int {
	typeFilter := flagValue(args, "--type")

	var targets []aiProvider
	for _, p := range aiProviders {
		if typeFilter == "" || p.Type == typeFilter {
			targets = append(targets, p)
		}
	}
	if len(targets) == 0 {
		fmt.Fprintf(os.Stderr, "no providers of type %q (use llm/agent/search)\n", typeFilter)
		return 2
	}

	results := make([]providerResult, len(targets))
	var wg sync.WaitGroup
	sem := make(chan struct{}, 6)
	for i, p := range targets {
		wg.Add(1)
		sem <- struct{}{}
		go func(idx int, pr aiProvider) {
			defer func() { <-sem; wg.Done() }()
			results[idx] = checkProvider(ctx, pr, 6*time.Second)
		}(i, p)
	}
	wg.Wait()

	sort.SliceStable(results, func(i, j int) bool {
		if results[i].Reachable != results[j].Reachable {
			return results[i].Reachable
		}
		return results[i].Name < results[j].Name
	})

	if hasFlag(args, "--json") {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		_ = enc.Encode(results)
		return 0
	}

	fmt.Printf("Checking %d AI provider(s) from the current egress...\n\n", len(targets))
	fmt.Printf("%-20s %-8s %-10s %s\n", "PROVIDER", "TYPE", "LATENCY", "STATUS")
	ok := 0
	for _, r := range results {
		status := "✖ blocked/unreachable"
		lat := "-"
		if r.Reachable {
			status = fmt.Sprintf("✔ reachable (HTTP %d)", r.Status)
			lat = fmt.Sprintf("%d ms", r.LatencyMS)
			ok++
		}
		fmt.Printf("%-20s %-8s %-10s %s\n", r.Name, r.Type, lat, status)
	}
	fmt.Printf("\n%d/%d AI providers reachable from here.\n", ok, len(results))
	return 0
}
