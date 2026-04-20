package orchestrator

import (
	"context"
	"errors"
	"testing"

	"github.com/ssyamv/claude-code-skills/xfchat-bootstrapper/internal/config"
	runtimeerrors "github.com/ssyamv/claude-code-skills/xfchat-bootstrapper/internal/errors"
	"github.com/ssyamv/claude-code-skills/xfchat-bootstrapper/internal/state"
)

func TestRunThreadsLoadedStateIntoPlatformSetupRunner(t *testing.T) {
	var got state.BootstrapState

	orch := Orchestrator{
		LoadState: func() (state.BootstrapState, error) {
			return state.BootstrapState{
				Phase:  state.PhasePlatformSetup,
				AppID:  "cli_123",
				AppURL: "https://open.xfchat.iflytek.com/app/cli_123/baseinfo",
			}, nil
		},
		PlatformSetupRunner: runnerFunc(func(_ context.Context, current state.BootstrapState) error {
			got = current
			return nil
		}),
		Validate: func(context.Context) error { return nil },
	}

	if err := orch.Run(context.Background()); err != nil {
		t.Fatalf("run failed: %v", err)
	}
	if got.AppID != "cli_123" || got.AppURL == "" {
		t.Fatalf("expected loaded state to reach platform runner, got %#v", got)
	}
}

func TestRunAdvancesOAuthPhaseToValidation(t *testing.T) {
	var oauthCalled bool
	var validateCalled bool

	orch := Orchestrator{
		LoadState: func() (state.BootstrapState, error) {
			return state.BootstrapState{
				Phase:  state.PhaseOAuth,
				AppID:  "cli_123",
				AppURL: "https://open.xfchat.iflytek.com/app/cli_123/baseinfo",
			}, nil
		},
		OAuthRunner: runnerFunc(func(context.Context, state.BootstrapState) error {
			oauthCalled = true
			return nil
		}),
		Validate: func(context.Context) error {
			validateCalled = true
			return nil
		},
	}

	if err := orch.Run(context.Background()); err != nil {
		t.Fatalf("run failed: %v", err)
	}
	if !oauthCalled {
		t.Fatal("expected oauth to be called")
	}
	if !validateCalled {
		t.Fatal("expected validate to be called after oauth")
	}
}

func TestNewWiresValidateDefaultUnimplementedForValidatePhase(t *testing.T) {
	root := t.TempDir()

	orch := New(config.Config{
		InstallRoot: root,
		CallbackURL: "http://localhost:8080/callback",
	}, state.NewStore(root), "windows")
	orch.LoadState = func() (state.BootstrapState, error) {
		return state.BootstrapState{
			Phase:  state.PhaseValidate,
			AppID:  "cli_123",
			AppURL: "https://open.xfchat.iflytek.com/app/cli_123/baseinfo",
		}, nil
	}

	err := orch.Run(context.Background())
	if err == nil {
		t.Fatal("expected internal validate error")
	}
	if !errors.Is(err, runtimeerrors.ErrValidationUnimplemented) {
		t.Fatalf("expected validation unimplemented error, got %v", err)
	}
}
