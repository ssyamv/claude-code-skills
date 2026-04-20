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
				AppID:  "cli_123",
				AppURL: "https://open.xfchat.iflytek.com/app/cli_123/baseinfo",
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
}
