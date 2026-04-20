package orchestrator

import (
	"context"
	"errors"
	"testing"

	"github.com/ssyamv/claude-code-skills/xfchat-bootstrapper/internal/state"
)

func TestRunReturnsInternalOAuthErrorInsteadOfMissingBinary(t *testing.T) {
	orch := Orchestrator{
		LoadState: func() (state.BootstrapState, error) {
			return state.BootstrapState{
				Phase:  state.PhaseOAuth,
				AppID:  "cli_123",
				AppURL: "https://open.xfchat.iflytek.com/app/cli_123/baseinfo",
			}, nil
		},
	}

	err := orch.Run(context.Background())
	if err == nil {
		t.Fatal("expected internal oauth error")
	}
	if !errors.Is(err, ErrOAuthUnimplemented) {
		t.Fatalf("expected internal oauth sentinel, got %v", err)
	}
}

func TestRunnerWaitsForCallbackSuccess(t *testing.T) {
	runner := Runner{
		StartCallbackServer: func() (CallbackWaiter, error) {
			return waiterFunc{
				url: "http://127.0.0.1:18080/callback",
				wait: func(context.Context) (CallbackResult, error) {
					return CallbackResult{Code: "ok"}, nil
				},
			}, nil
		},
	}

	if err := runner.Run(context.Background(), state.BootstrapState{Phase: state.PhaseOAuth}); err != nil {
		t.Fatalf("oauth run failed: %v", err)
	}
}
