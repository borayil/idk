package aiprovider

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
)

const claudeDefaultBaseURL = "https://api.anthropic.com"
const claudeModel = "claude-haiku-4-5-20251001" // fast+cheap, this is a supplement not the main path

type claudeProvider struct {
	baseURL string
}

func newClaudeProvider(baseURL string) *claudeProvider {
	if baseURL == "" {
		baseURL = claudeDefaultBaseURL
	}
	return &claudeProvider{baseURL: baseURL}
}

func (p *claudeProvider) Name() string      { return "claude" }
func (p *claudeProvider) APIKeyEnv() string { return "ANTHROPIC_API_KEY" }

type claudeMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type claudeResponse struct {
	Content []struct {
		Text string `json:"text"`
	} `json:"content"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

func (p *claudeProvider) Suggest(ctx context.Context, req Request) ([]string, error) {
	apiKey := os.Getenv(p.APIKeyEnv())
	if apiKey == "" {
		return nil, fmt.Errorf("%s is not set", p.APIKeyEnv())
	}

	data, err := postJSON(ctx, p.baseURL+"/v1/messages", map[string]string{
		"x-api-key":         apiKey,
		"anthropic-version": "2023-06-01",
	}, map[string]any{
		"model":      claudeModel,
		"max_tokens": 150,
		"messages":   []claudeMessage{{Role: "user", Content: buildPrompt(req)}},
	})
	if err != nil {
		return nil, err
	}

	var parsed claudeResponse
	if err := json.Unmarshal(data, &parsed); err != nil {
		return nil, fmt.Errorf("unexpected response: %w", err)
	}
	if parsed.Error != nil {
		return nil, fmt.Errorf("claude api error: %s", parsed.Error.Message)
	}
	if len(parsed.Content) == 0 {
		return nil, nil
	}
	return parseSuggestions(parsed.Content[0].Text, req.MaxSuggestions), nil
}
