package clob

import (
	"testing"

	"go.uber.org/zap"
)

func TestNewClient(t *testing.T) {
	logger := zap.NewNop().Sugar()
	client := NewClient("", logger)

	if client.baseURL != BaseURL {
		t.Errorf("expected BaseURL %s, got %s", BaseURL, client.baseURL)
	}
}

func TestWithCredentials(t *testing.T) {
	logger := zap.NewNop().Sugar()
	client := NewClient("", logger)

	creds := &Credentials{
		APIKey:     "test-key",
		PrivateKey: "0x0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef", // 64 hex chars
	}

	client.WithCredentials(creds)

	if client.creds != creds {
		t.Error("credentials not set")
	}
	if client.signer == nil {
		t.Error("signer should be initialized")
	}
}
