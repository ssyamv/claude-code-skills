package browser

import (
	"context"
	"testing"

	"github.com/ssyamv/claude-code-skills/xfchat-bootstrapper/internal/state"
)

func TestRunnerReturnsPlatformSetupMetadata(t *testing.T) {
	runner := Runner{
		Automate: func(context.Context, Workflow) (PlatformSetupResult, error) {
			return PlatformSetupResult{
				AppID:     "cli_123",
				AppURL:    "https://open.xfchat.iflytek.com/app/cli_123/baseinfo",
				AppSecret: "secret-123",
				AuthURL:   "https://open.xfchat.iflytek.com/oauth/authorize?app_id=cli_123&redirect_uri=http%3A%2F%2Flocalhost%3A8080%2Fcallback",
			}, nil
		},
		Workflow: NewWorkflow(WorkflowConfig{
			AppEntryURL: "https://open.xfchat.iflytek.com/app",
			CallbackURL: "http://localhost:8080/callback",
		}),
	}

	result, err := runner.Run(context.Background(), state.BootstrapState{})
	if err != nil {
		t.Fatalf("run failed: %v", err)
	}
	if result.AppID != "cli_123" {
		t.Fatalf("expected app id, got %#v", result)
	}
	if result.AppURL != "https://open.xfchat.iflytek.com/app/cli_123/baseinfo" {
		t.Fatalf("expected app url, got %#v", result)
	}
	if result.AppSecret != "secret-123" {
		t.Fatalf("expected app secret, got %#v", result)
	}
	if result.AuthURL != "https://open.xfchat.iflytek.com/oauth/authorize?app_id=cli_123&redirect_uri=http%3A%2F%2Flocalhost%3A8080%2Fcallback" {
		t.Fatalf("expected auth url, got %#v", result)
	}
}

func TestRunnerPreservesFallbackSecretAndAuthURL(t *testing.T) {
	runner := Runner{
		Automate: func(context.Context, Workflow) (PlatformSetupResult, error) {
			return PlatformSetupResult{
				AppID: "cli_123",
			}, nil
		},
	}

	result, err := runner.Run(context.Background(), state.BootstrapState{
		AppID:     "legacy-id",
		AppURL:    "https://open.xfchat.iflytek.com/app/legacy-id/baseinfo",
		AppSecret: "fallback-secret",
		AuthURL:   "https://open.xfchat.iflytek.com/oauth/authorize?app_id=legacy-id",
	})
	if err != nil {
		t.Fatalf("run failed: %v", err)
	}
	if result.AppSecret != "fallback-secret" {
		t.Fatalf("expected fallback secret to survive, got %#v", result)
	}
	if result.AuthURL != "https://open.xfchat.iflytek.com/oauth/authorize?app_id=legacy-id" {
		t.Fatalf("expected fallback auth url to survive, got %#v", result)
	}
}
