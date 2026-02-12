package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"
)

// Compile-time interface check.
var _ AIProvider = (*AnthropicProvider)(nil)

const anthropicAPIURL = "https://api.anthropic.com/v1/messages"

// AnthropicProvider implements AIProvider using the Anthropic Messages API.
type AnthropicProvider struct {
	apiKey string
	model  string
	client *http.Client
}

// NewAnthropicProvider creates an AnthropicProvider with a 60-second timeout
// HTTP client.
func NewAnthropicProvider(apiKey, model string) *AnthropicProvider {
	return &AnthropicProvider{
		apiKey: apiKey,
		model:  model,
		client: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

// anthropicRequest is the request body for the Anthropic Messages API.
type anthropicRequest struct {
	Model     string             `json:"model"`
	MaxTokens int                `json:"max_tokens"`
	System    string             `json:"system"`
	Messages  []anthropicMessage `json:"messages"`
}

// anthropicMessage is a single message in the Anthropic request.
type anthropicMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// anthropicResponse is the response body from the Anthropic Messages API.
type anthropicResponse struct {
	Content []struct {
		Text string `json:"text"`
	} `json:"content"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

// FilterAndRank selects and ranks blogs based on user preferences using the
// Anthropic Messages API.
func (p *AnthropicProvider) FilterAndRank(ctx context.Context, preferences string, blogs []BlogEntry) ([]RankedBlog, error) {
	systemPrompt, userPrompt := FilterAndRankPrompt(preferences, blogs)

	text, err := p.callAPI(ctx, systemPrompt, userPrompt)
	if err != nil {
		return nil, fmt.Errorf("anthropic filter-and-rank: %w", err)
	}

	cleaned := extractJSON(text)

	var ranked []RankedBlog
	if err := json.Unmarshal([]byte(cleaned), &ranked); err != nil {
		return nil, fmt.Errorf("anthropic filter-and-rank: parsing response JSON: %w", err)
	}

	return ranked, nil
}

// Summarize generates a concise summary of the given blog post using the
// Anthropic Messages API.
func (p *AnthropicProvider) Summarize(ctx context.Context, blog BlogEntry) (string, error) {
	content := blog.FullContent
	if content == "" {
		content = blog.Description
	}

	systemPrompt, userPrompt := SummarizePrompt(blog.Title, blog.Source, content)

	text, err := p.callAPI(ctx, systemPrompt, userPrompt)
	if err != nil {
		return "", fmt.Errorf("anthropic summarize: %w", err)
	}

	return text, nil
}

// callAPI makes an HTTP request to the Anthropic Messages API and returns
// the text content from the first content block.
func (p *AnthropicProvider) callAPI(ctx context.Context, systemPrompt, userPrompt string) (string, error) {
	reqBody := anthropicRequest{
		Model:     p.model,
		MaxTokens: 1024,
		System:    systemPrompt,
		Messages: []anthropicMessage{
			{Role: "user", Content: userPrompt},
		},
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("marshaling request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, anthropicAPIURL, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("x-api-key", p.apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")
	req.Header.Set("content-type", "application/json")

	slog.Debug("calling Anthropic API", "model", p.model)

	resp, err := p.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("sending request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("reading response body: %w", err)
	}

	var apiResp anthropicResponse
	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		return "", fmt.Errorf("parsing response (status %d): %w", resp.StatusCode, err)
	}

	if apiResp.Error != nil {
		return "", fmt.Errorf("API error (status %d): %s", resp.StatusCode, apiResp.Error.Message)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	if len(apiResp.Content) == 0 {
		return "", fmt.Errorf("empty response: no content blocks returned")
	}

	return apiResp.Content[0].Text, nil
}
