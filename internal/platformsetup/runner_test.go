package platformsetup

import (
	"context"
	"testing"

	"github.com/ssyamv/claude-code-skills/xfchat-bootstrapper/internal/browser"
	"github.com/ssyamv/claude-code-skills/xfchat-bootstrapper/internal/state"
)

func TestRunnerReturnsStateWithCredentialsAndAuthURL(t *testing.T) {
	runner := Runner{
		BootstrapSession: func(context.Context) (browser.SessionContext, error) {
			return browser.SessionContext{BaseURL: "https://open.xfchat.iflytek.com"}, nil
		},
		CreateApp: func(context.Context, browser.SessionContext) (Result, error) {
			return Result{
				AppID:     "cli_123",
				AppURL:    "https://open.xfchat.iflytek.com/app/cli_123/baseinfo",
				AppSecret: "secret-123",
				AuthURL:   "https://open.xfchat.iflytek.com/oauth/authorize?app_id=cli_123",
			}, nil
		},
	}

	got, err := runner.RunState(context.Background(), state.BootstrapState{Phase: state.PhasePlatformSetup})
	if err != nil {
		t.Fatalf("run state failed: %v", err)
	}
	if got.AppID != "cli_123" {
		t.Fatalf("expected app id to persist, got %#v", got)
	}
	if got.AppSecret != "secret-123" || got.AuthURL != "https://open.xfchat.iflytek.com/oauth/authorize?app_id=cli_123" {
		t.Fatalf("expected credential metadata to persist, got %#v", got)
	}
}
