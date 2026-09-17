package aiprovider

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestParseSuggestionsSanitizes(t *testing.T) {
	raw := "git push\n`git pull`\n\n" + strings.Repeat("x", 250) + "\ngit stash"
	got := parseSuggestions(raw, 2)
	want := []string{"git push", "git pull"}
	if len(got) != len(want) {
		t.Fatalf("got %v want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("got %v want %v", got, want)
		}
	}
}

func TestParseSuggestionsRejectsOverlong(t *testing.T) {
	got := parseSuggestions(strings.Repeat("x", 300), 5)
	if len(got) != 0 {
		t.Errorf("expected overlong line rejected, got %v", got)
	}
}

func TestBuildPromptIncludesContext(t *testing.T) {
	req := Request{Branch: "main", ChangedFiles: []string{"main.go"}, RecentCommands: []string{"go build"}, MaxSuggestions: 2}
	prompt := buildPrompt(req)
	for _, want := range []string{"main", "main.go", "go build", "one per line"} {
		if !strings.Contains(prompt, want) {
			t.Errorf("expected prompt to contain %q, got: %s", want, prompt)
		}
	}
}

func TestClaudeSuggestSuccess(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("x-api-key") != "test-key" {
			t.Errorf("missing api key header")
		}
		json.NewEncoder(w).Encode(map[string]any{
			"content": []map[string]string{{"text": "git push\ngit pull"}},
		})
	}))
	defer srv.Close()

	t.Setenv("ANTHROPIC_API_KEY", "test-key")
	p := newClaudeProvider(srv.URL)
	got, err := p.Suggest(context.Background(), Request{MaxSuggestions: 2})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0] != "git push" || got[1] != "git pull" {
		t.Errorf("got %v", got)
	}
}

func TestClaudeSuggestMissingKey(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "")
	p := newClaudeProvider("http://unused")
	_, err := p.Suggest(context.Background(), Request{MaxSuggestions: 2})
	if err == nil {
		t.Fatal("expected error for missing API key")
	}
}

func TestClaudeSuggestAPIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]string{"message": "invalid api key"},
		})
	}))
	defer srv.Close()

	t.Setenv("ANTHROPIC_API_KEY", "bad-key")
	p := newClaudeProvider(srv.URL)
	_, err := p.Suggest(context.Background(), Request{MaxSuggestions: 2})
	if err == nil || !strings.Contains(err.Error(), "invalid api key") {
		t.Errorf("expected api error surfaced, got %v", err)
	}
}

func TestClaudeSuggestTimeout(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
		json.NewEncoder(w).Encode(map[string]any{"content": []map[string]string{{"text": "git push"}}})
	}))
	defer srv.Close()

	t.Setenv("ANTHROPIC_API_KEY", "test-key")
	p := newClaudeProvider(srv.URL)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	_, err := p.Suggest(ctx, Request{MaxSuggestions: 2})
	if err == nil {
		t.Fatal("expected context deadline error")
	}
}

func TestOpenAISuggestSuccess(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("authorization") != "Bearer test-key" {
			t.Errorf("missing auth header, got %q", r.Header.Get("authorization"))
		}
		json.NewEncoder(w).Encode(map[string]any{
			"choices": []map[string]any{{"message": map[string]string{"role": "assistant", "content": "npm test"}}},
		})
	}))
	defer srv.Close()

	t.Setenv("OPENAI_API_KEY", "test-key")
	p := newOpenAIProvider(srv.URL)
	got, err := p.Suggest(context.Background(), Request{MaxSuggestions: 2})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0] != "npm test" {
		t.Errorf("got %v", got)
	}
}

func TestAutoDetectPrefersClaudeThenOpenAI(t *testing.T) {
	env := map[string]string{"ANTHROPIC_API_KEY": "x", "OPENAI_API_KEY": "y"}
	p, err := AutoDetect(func(k string) string { return env[k] })
	if err != nil {
		t.Fatal(err)
	}
	if p.Name() != "claude" {
		t.Errorf("expected claude preferred, got %s", p.Name())
	}

	delete(env, "ANTHROPIC_API_KEY")
	p, err = AutoDetect(func(k string) string { return env[k] })
	if err != nil {
		t.Fatal(err)
	}
	if p.Name() != "openai" {
		t.Errorf("expected openai fallback, got %s", p.Name())
	}
}

func TestAutoDetectNoneSet(t *testing.T) {
	_, err := AutoDetect(func(string) string { return "" })
	if err == nil {
		t.Fatal("expected error when no key is set")
	}
}

func TestResolveUnknownProvider(t *testing.T) {
	_, err := Resolve("bogus")
	if err == nil {
		t.Fatal("expected error for unknown provider")
	}
}
