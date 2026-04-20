package config

import (
	"reflect"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := Default()

	if cfg.Brand != "xfchat.iflytek.com" {
		t.Fatalf("expected default brand xfchat.iflytek.com, got %q", cfg.Brand)
	}
	if cfg.CallbackURL != "http://localhost:8080/callback" {
		t.Fatalf("expected callback URL to be preconfigured, got %q", cfg.CallbackURL)
	}
	expectedScopes := []string{
		"docs:document:readonly",
		"im:message:create_as_bot",
	}
	if !reflect.DeepEqual(cfg.RequiredScopes, expectedScopes) {
		t.Fatalf("expected required scopes %v, got %v", expectedScopes, cfg.RequiredScopes)
	}
}
