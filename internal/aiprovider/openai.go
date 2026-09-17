package aiprovider

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
)

const openAIDefaultBaseURL = "https://api.openai.com"
const openAIModel = "gpt-4o-mini" // fast+cheap, this is a supplement not the main path

type openAIProvider struct {
	baseURL string
}

func newOpenAIProvider(baseURL string) *openAIProvider {
	if baseURL == "" {
		baseURL = openAIDefaultBaseURL
	}
	return &openAIProvider{baseURL: baseURL}
}

func (p *openAIProvider) Name() string      { return "openai" }
func (p *openAIProvider) APIKeyEnv() string { return "OPENAI_API_KEY" }

type openAIMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type openAIResponse struct {
	Choices []struct {
		Message openAIMessage `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

func (p *openAIProvider) Suggest(ctx context.Context, req Request) ([]string, error) {
	apiKey := os.Getenv(p.APIKeyEnv())
	if apiKey == "" {
		return nil, fmt.Errorf("%s is not set", p.APIKeyEnv())
	}

	data, err := postJSON(ctx, p.baseURL+"/v1/chat/completions", map[string]string{
		"authorization": "Bearer " + apiKey,
	}, map[string]any{
		"model":    openAIModel,
		"messages": []openAIMessage{{Role: "user", Content: buildPrompt(req)}},
	})
	if err != nil {
		return nil, err
	}

	var parsed openAIResponse
	if err := json.Unmarshal(data, &parsed); err != nil {
		return nil, fmt.Errorf("unexpected response: %w", err)
	}
	if parsed.Error != nil {
		return nil, fmt.Errorf("openai api error: %s", parsed.Error.Message)
	}
	if len(parsed.Choices) == 0 {
		return nil, nil
	}
	return parseSuggestions(parsed.Choices[0].Message.Content, req.MaxSuggestions), nil
}
