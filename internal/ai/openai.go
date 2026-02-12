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
var _ AIProvider = (*OpenAIProvider)(nil)

const openaiAPIURL = "https://api.openai.com/v1/chat/completions"

// OpenAIProvider implements AIProvider using the OpenAI Chat Completions API.
type OpenAIProvider struct {
	apiKey string
	model  string
	client *http.Client
}

// NewOpenAIProvider creates an OpenAIProvider with a 60-second timeout
// HTTP client.
func NewOpenAIProvider(apiKey, model string) *OpenAIProvider {
	return &OpenAIProvider{
		apiKey: apiKey,
		model:  model,
		client: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

// openaiRequest is the request body for the OpenAI Chat Completions API.
type openaiRequest struct {
	Model    string           `json:"model"`
	Messages []openaiMessage  `json:"messages"`
}

// openaiMessage is a single message in the OpenAI request.
type openaiMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// openaiResponse is the response body from the OpenAI Chat Completions API.
type openaiResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

// FilterAndRank selects and ranks blogs based on user preferences using the
// OpenAI Chat Completions API.
func (p *OpenAIProvider) FilterAndRank(ctx context.Context, preferences string, blogs []BlogEntry) ([]RankedBlog, error) {
	systemPrompt, userPrompt := FilterAndRankPrompt(preferences, blogs)

	text, err := p.callAPI(ctx, systemPrompt, userPrompt)
	if err != nil {
		return nil, fmt.Errorf("openai filter-and-rank: %w", err)
	}

	cleaned := extractJSON(text)

	var ranked []RankedBlog
	if err := json.Unmarshal([]byte(cleaned), &ranked); err != nil {
		return nil, fmt.Errorf("openai filter-and-rank: parsing response JSON: %w", err)
	}

	return ranked, nil
}

// Summarize generates a concise summary of the given blog post using the
// OpenAI Chat Completions API.
func (p *OpenAIProvider) Summarize(ctx context.Context, blog BlogEntry) (string, error) {
	content := blog.FullContent
	if content == "" {
		content = blog.Description
	}

	systemPrompt, userPrompt := SummarizePrompt(blog.Title, blog.Source, content)

	text, err := p.callAPI(ctx, systemPrompt, userPrompt)
	if err != nil {
		return "", fmt.Errorf("openai summarize: %w", err)
	}

	return text, nil
}

// callAPI makes an HTTP request to the OpenAI Chat Completions API and
// returns the text content from the first choice.
func (p *OpenAIProvider) callAPI(ctx context.Context, systemPrompt, userPrompt string) (string, error) {
	reqBody := openaiRequest{
		Model: p.model,
		Messages: []openaiMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userPrompt},
		},
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("marshaling request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, openaiAPIURL, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+p.apiKey)
	req.Header.Set("Content-Type", "application/json")

	slog.Debug("calling OpenAI API", "model", p.model)

	resp, err := p.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("sending request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("reading response body: %w", err)
	}

	var apiResp openaiResponse
	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		return "", fmt.Errorf("parsing response (status %d): %w", resp.StatusCode, err)
	}

	if apiResp.Error != nil {
		return "", fmt.Errorf("API error (status %d): %s", resp.StatusCode, apiResp.Error.Message)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	if len(apiResp.Choices) == 0 {
		return "", fmt.Errorf("empty response: no choices returned")
	}

	return apiResp.Choices[0].Message.Content, nil
}
