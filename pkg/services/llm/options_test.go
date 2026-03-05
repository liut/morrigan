package llm

import (
	"testing"
	"time"
)

func TestOptions(t *testing.T) {
	cfg := applyOptions(
		WithProvider("custom"),
		WithBaseURL("https://custom.api.com/v1"),
		WithAPIKey("test-key"),
		WithModel("custom-model"),
		WithMaxTokens(2048),
		WithTemperature(0.5),
		WithTimeout(30*time.Second),
	)

	if cfg.provider != "custom" {
		t.Errorf("provider = %v, want custom", cfg.provider)
	}
	if cfg.baseURL != "https://custom.api.com/v1" {
		t.Errorf("baseURL = %v, want https://custom.api.com/v1", cfg.baseURL)
	}
	if cfg.apiKey != "test-key" {
		t.Errorf("apiKey = %v, want test-key", cfg.apiKey)
	}
	if cfg.model != "custom-model" {
		t.Errorf("model = %v, want custom-model", cfg.model)
	}
	if cfg.maxTokens != 2048 {
		t.Errorf("maxTokens = %v, want 2048", cfg.maxTokens)
	}
	if cfg.temperature != 0.5 {
		t.Errorf("temperature = %v, want 0.5", cfg.temperature)
	}
	if cfg.timeout != 30*time.Second {
		t.Errorf("timeout = %v, want 30s", cfg.timeout)
	}
}
