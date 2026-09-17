// Package aiprovider is the opt-in "idk enable ai" suggestion supplement. Off unless a user
// explicitly enables it and has the matching API key set.
package aiprovider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Request is the context sent to a provider when asking for extra command suggestions.
type Request struct {
	Branch         string
	ChangedFiles   []string
	RecentCommands []string // already-ranked heuristic candidates, most relevant first
	MaxSuggestions int
}

// Provider is an LLM backend that can suggest extra shell commands.
type Provider interface {
	Name() string
	APIKeyEnv() string
	Suggest(ctx context.Context, req Request) ([]string, error)
}

// httpClient has a defense-in-depth timeout; the real deadline comes from the caller's ctx.
var httpClient = &http.Client{Timeout: 5 * time.Second}

// postJSON marshals body, posts it with the given headers, and returns the raw response body.
// Shared by every provider so each one only has to know its own request/response shapes.
func postJSON(ctx context.Context, url string, headers map[string]string, body any) ([]byte, error) {
	b, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(b))
	if err != nil {
		return nil, err
	}
	req.Header.Set("content-type", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}

// Resolve returns the Provider for a config name ("claude" or "openai").
func Resolve(name string) (Provider, error) {
	switch name {
	case "claude":
		return newClaudeProvider(""), nil
	case "openai":
		return newOpenAIProvider(""), nil
	default:
		return nil, fmt.Errorf("unknown ai provider %q (supported: claude, openai)", name)
	}
}

// AutoDetect picks a provider by which API key is already set, preferring claude.
// Used by `idk enable ai` when no provider is named explicitly.
func AutoDetect(getenv func(string) string) (Provider, error) {
	candidates := []Provider{newClaudeProvider(""), newOpenAIProvider("")}
	for _, p := range candidates {
		if getenv(p.APIKeyEnv()) != "" {
			return p, nil
		}
	}
	names := make([]string, len(candidates))
	envs := make([]string, len(candidates))
	for i, p := range candidates {
		names[i] = p.Name()
		envs[i] = p.APIKeyEnv()
	}
	return nil, fmt.Errorf("no API key found; set one of [%s] (for %s respectively) or pass a provider explicitly",
		strings.Join(envs, ", "), strings.Join(names, ", "))
}

func buildPrompt(req Request) string {
	var b strings.Builder
	b.WriteString("You suggest shell commands for a developer's command-line tool. ")
	fmt.Fprintf(&b, "Suggest up to %d additional shell commands the user might want to run next, ", req.MaxSuggestions)
	b.WriteString("that are NOT already in the recent-commands list below. ")
	b.WriteString("Respond with ONLY the raw commands, one per line, no explanation, no numbering, no backticks, no markdown.\n\n")
	if req.Branch != "" {
		fmt.Fprintf(&b, "Git branch: %s\n", req.Branch)
	}
	if len(req.ChangedFiles) > 0 {
		fmt.Fprintf(&b, "Changed files: %s\n", strings.Join(req.ChangedFiles, ", "))
	}
	if len(req.RecentCommands) > 0 {
		b.WriteString("Recent commands:\n")
		for _, c := range req.RecentCommands {
			fmt.Fprintf(&b, "- %s\n", c)
		}
	}
	return b.String()
}

// parseSuggestions sanitizes raw LLM text into safe-ish command candidates. idk will `eval`
// whatever comes back, so empty/absurd/markdown-wrapped lines get dropped rather than trusted.
func parseSuggestions(raw string, max int) []string {
	var out []string
	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(line)
		line = strings.Trim(line, "`")
		line = strings.TrimSpace(line)
		if line == "" || len(line) > 200 {
			continue
		}
		out = append(out, line)
		if len(out) >= max {
			break
		}
	}
	return out
}
