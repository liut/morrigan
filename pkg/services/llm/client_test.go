package llm

import (
	"testing"
)

func TestNewClient(t *testing.T) {
	tests := []struct {
		name      string
		opts      []Option
		wantErr   bool
		checkFunc func(*testing.T, Client)
	}{
		{
			name:    "default provider",
			wantErr: false,
			checkFunc: func(t *testing.T, c Client) {
				if c == nil {
					t.Error("expected non-nil client")
				}
			},
		},
		{
			name: "with custom model",
			opts: []Option{WithModel("gpt-4")},
			wantErr: false,
			checkFunc: func(t *testing.T, c Client) {
				// Client created successfully
			},
		},
		{
			name:    "anthropic provider",
			opts:    []Option{WithProvider("anthropic")},
			wantErr: false,
		},
		{
			name:    "gemini not implemented",
			opts:    []Option{WithProvider("gemini")},
			wantErr: true,
		},
		{
			name: "openrouter provider",
			opts: []Option{WithProvider("openrouter")},
			wantErr: false,
		},
		{
			name: "ollama provider",
			opts: []Option{WithProvider("ollama")},
			wantErr: false,
		},
		{
			name: "unknown provider",
			opts: []Option{WithProvider("unknown-provider")},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, err := NewClient(tt.opts...)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewClient() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.checkFunc != nil && err == nil {
				tt.checkFunc(t, c)
			}
		})
	}
}
