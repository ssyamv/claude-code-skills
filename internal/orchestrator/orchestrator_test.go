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

func TestRunAdvancesStateAfterPlatformSetupAndOAuth(t *testing.T) {
	root := t.TempDir()
	store := state.NewStore(root)
	if err := store.Save(state.BootstrapState{
		Phase: state.PhasePlatformSetup,
	}); err != nil {
		t.Fatalf("seed state failed: %v", err)
	}

	orch := New(config.Config{}, store, "windows")
	orch.LoadState = func() (state.BootstrapState, error) {
		return store.Load()
	}
	orch.PlatformSetupRunner = stateAdvanceRunnerFunc(func(context.Context, state.BootstrapState) (state.BootstrapState, error) {
		return state.BootstrapState{
			AppID:  "cli_123",
			AppURL: "https://open.xfchat.iflytek.com/app/cli_123/baseinfo",
		}, nil
	})
	orch.OAuthRunner = runnerFunc(func(context.Context, state.BootstrapState) error {
		return nil
	})
	orch.Validate = func(context.Context) error { return nil }

	if err := orch.Run(context.Background()); err != nil {
		t.Fatalf("run failed: %v", err)
	}

	got, err := store.Load()
	if err != nil {
		t.Fatalf("load failed: %v", err)
	}
	if got.Phase != state.PhaseValidate {
		t.Fatalf("expected final phase validate, got %#v", got)
	}
	if !got.AuthSuccess {
		t.Fatalf("expected oauth success to be persisted, got %#v", got)
	}
	if got.AppID != "cli_123" || got.AppURL == "" {
		t.Fatalf("expected app metadata to persist, got %#v", got)
	}
}

type stateAdvanceRunnerFunc func(context.Context, state.BootstrapState) (state.BootstrapState, error)

func (f stateAdvanceRunnerFunc) RunState(ctx context.Context, current state.BootstrapState) (state.BootstrapState, error) {
	return f(ctx, current)
}

func (f stateAdvanceRunnerFunc) Run(context.Context, state.BootstrapState) error {
	return nil
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
	orch.PlatformSetupRunner = nil
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
	orch.OAuthRunner = nil
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
