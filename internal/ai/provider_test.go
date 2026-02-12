package ai

import (
	"testing"
)

func TestNewProvider(t *testing.T) {
	tests := []struct {
		name      string
		cfg       ProviderConfig
		wantErr   bool
		wantType  string
	}{
		{
			name: "anthropic provider",
			cfg: ProviderConfig{
				Provider: "anthropic",
				APIKey:   "test-key",
				Model:    "claude-haiku-4-5",
			},
			wantErr:  false,
			wantType: "*ai.AnthropicProvider",
		},
		{
			name: "openai provider",
			cfg: ProviderConfig{
				Provider: "openai",
				APIKey:   "test-key",
				Model:    "gpt-4o-mini",
			},
			wantErr:  false,
			wantType: "*ai.OpenAIProvider",
		},
		{
			name: "unsupported provider",
			cfg: ProviderConfig{
				Provider: "invalid",
				APIKey:   "test-key",
				Model:    "some-model",
			},
			wantErr: true,
		},
		{
			name: "empty provider",
			cfg: ProviderConfig{
				Provider: "",
				APIKey:   "test-key",
				Model:    "some-model",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider, err := NewProvider(tt.cfg)

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if provider != nil {
					t.Fatal("expected nil provider when error occurs")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if provider == nil {
				t.Fatal("expected non-nil provider")
			}

			// Verify the concrete type via type assertion.
			switch tt.wantType {
			case "*ai.AnthropicProvider":
				if _, ok := provider.(*AnthropicProvider); !ok {
					t.Errorf("expected *AnthropicProvider, got %T", provider)
				}
			case "*ai.OpenAIProvider":
				if _, ok := provider.(*OpenAIProvider); !ok {
					t.Errorf("expected *OpenAIProvider, got %T", provider)
				}
			}
		})
	}
}
