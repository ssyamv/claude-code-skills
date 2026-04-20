package orchestrator

import (
	"context"
	"errors"
	"testing"

	"github.com/ssyamv/claude-code-skills/xfchat-bootstrapper/internal/config"
	runtimeerrors "github.com/ssyamv/claude-code-skills/xfchat-bootstrapper/internal/errors"
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

func TestNewWiresInternalValidationDefaultForValidatePhase(t *testing.T) {
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
		t.Fatal("expected internal validation error")
	}
	if !errors.Is(err, runtimeerrors.ErrValidationUnimplemented) {
		t.Fatalf("expected internal validation sentinel, got %v", err)
	}
}

func TestNewAllowsRuntimeFallbackForLoadedPlatformPhase(t *testing.T) {
	var calls []string
	var executedState state.BootstrapState
	root := t.TempDir()

	orch := New(config.Config{
		InstallRoot: root,
		CallbackURL: "http://localhost:8080/callback",
	}, state.NewStore(root), "windows")
	orch.LoadState = func() (state.BootstrapState, error) {
		return state.BootstrapState{
			Phase:  state.PhasePlatformSetup,
			AppID:  "cli_123",
			AppURL: "https://open.xfchat.iflytek.com/app/cli_123/baseinfo",
		}, nil
	}
	orch.Execute = func(_ context.Context, current state.BootstrapState) error {
		calls = append(calls, "execute")
		executedState = current
		return nil
	}
	orch.Validate = func(context.Context) error {
		calls = append(calls, "validate")
		return runtimeerrors.ErrValidationUnimplemented
	}

	if err := orch.Run(context.Background()); err != nil {
		if !errors.Is(err, runtimeerrors.ErrValidationUnimplemented) {
			t.Fatalf("run failed: %v", err)
		}
	}
	if len(calls) != 2 || calls[0] != "execute" || calls[1] != "validate" {
		t.Fatalf("expected execute -> validate ordering, got %v", calls)
	}
	if executedState.Phase != state.PhasePlatformSetup || executedState.AppID != "cli_123" || executedState.AppURL == "" {
		t.Fatalf("expected loaded state to be threaded into execute, got %#v", executedState)
	}
}

func TestNewAllowsRuntimeFallbackForLoadedOAuthPhase(t *testing.T) {
	var calls []string
	var executedState state.BootstrapState
	root := t.TempDir()

	orch := New(config.Config{
		InstallRoot: root,
		CallbackURL: "http://localhost:8080/callback",
	}, state.NewStore(root), "windows")
	orch.LoadState = func() (state.BootstrapState, error) {
		return state.BootstrapState{
			Phase:  state.PhaseOAuth,
			AppID:  "cli_123",
			AppURL: "https://open.xfchat.iflytek.com/app/cli_123/baseinfo",
		}, nil
	}
	orch.Execute = func(_ context.Context, current state.BootstrapState) error {
		calls = append(calls, "execute")
		executedState = current
		return nil
	}
	orch.Validate = func(context.Context) error {
		calls = append(calls, "validate")
		return runtimeerrors.ErrValidationUnimplemented
	}

	if err := orch.Run(context.Background()); err != nil {
		if !errors.Is(err, runtimeerrors.ErrValidationUnimplemented) {
			t.Fatalf("run failed: %v", err)
		}
	}
	if len(calls) != 2 || calls[0] != "execute" || calls[1] != "validate" {
		t.Fatalf("expected execute -> validate ordering, got %v", calls)
	}
	if executedState.Phase != state.PhaseOAuth || executedState.AppID != "cli_123" || executedState.AppURL == "" {
		t.Fatalf("expected loaded state to be threaded into execute, got %#v", executedState)
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
