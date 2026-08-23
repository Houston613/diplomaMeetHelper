package speech

import (
	"testing"
	"time"

	"diplomaMeetHelper/internal/config"
)

func TestNewClient_Mock(t *testing.T) {
	cfg := config.SpeechConfig{
		Provider: "mock",
		Mock: config.MockConfig{
			Delay:     10 * time.Millisecond,
			ErrorRate: 0,
		},
	}

	client, err := NewClient(cfg)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if client == nil {
		t.Fatal("expected non-nil speech client")
	}
}

func TestNewClient_Unsupported(t *testing.T) {
	cfg := config.SpeechConfig{
		Provider: "unknown_provider",
	}

	client, err := NewClient(cfg)
	if err == nil {
		t.Fatal("expected error for unsupported provider, got nil")
	}
	if client != nil {
		t.Fatal("expected nil client on error")
	}
}
