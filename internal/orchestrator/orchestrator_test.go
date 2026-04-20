package orchestrator

import (
	"context"
	"errors"
	"testing"

	"github.com/ssyamv/claude-code-skills/xfchat-bootstrapper/internal/config"
	"github.com/ssyamv/claude-code-skills/xfchat-bootstrapper/internal/state"
)

func TestRunResumesAtValidatePhase(t *testing.T) {
	var validateCalled bool
	var executeCalled bool

	orch := Orchestrator{
		LoadState: func() (state.BootstrapState, error) {
			return state.BootstrapState{
				Phase: state.PhaseValidate,
				AppID: "cli_123",
			}, nil
		},
		Validate: func(context.Context) error {
			validateCalled = true
			return nil
		},
		Execute: func(context.Context, state.BootstrapState) error {
			executeCalled = true
			return errors.New("execute should not be called")
		},
	}

	if err := orch.Run(context.Background()); err != nil {
		t.Fatalf("run failed: %v", err)
	}
	if !validateCalled {
		t.Fatal("expected validate to be called")
	}
	if executeCalled {
		t.Fatal("expected execute not to be called")
	}
}

func TestRunExecutesThenValidatesForEarlierPhase(t *testing.T) {
	var validateCalled bool
	var executeCalled bool
	var executedState state.BootstrapState

	orch := Orchestrator{
		LoadState: func() (state.BootstrapState, error) {
			return state.BootstrapState{
				Phase:  state.PhasePlatformSetup,
				AppID:  "cli_123",
				AppURL: "https://open.xfchat.iflytek.com/app/cli_123/baseinfo",
			}, nil
		},
		Validate: func(context.Context) error {
			validateCalled = true
			return nil
		},
		Execute: func(_ context.Context, current state.BootstrapState) error {
			executeCalled = true
			executedState = current
			return nil
		},
	}

	if err := orch.Run(context.Background()); err != nil {
		t.Fatalf("run failed: %v", err)
	}
	if !executeCalled {
		t.Fatal("expected execute to be called")
	}
	if executedState.Phase != state.PhasePlatformSetup || executedState.AppID != "cli_123" || executedState.AppURL == "" {
		t.Fatalf("expected loaded state to be threaded into execute, got %#v", executedState)
	}
	if !validateCalled {
		t.Fatal("expected validate to be called after execute")
	}
}

func TestNewDoesNotRequireExternalLarkCLIToStart(t *testing.T) {
	root := t.TempDir()

	orch := New(config.Config{
		InstallRoot: root,
		CallbackURL: "http://localhost:8080/callback",
	}, state.NewStore(root), "windows")

	if orch.LoadState == nil {
		t.Fatal("expected load state to be wired")
	}

	if err := orch.Run(context.Background()); err != nil {
		t.Fatalf("expected startup to succeed without external lark-cli, got %v", err)
	}
}
