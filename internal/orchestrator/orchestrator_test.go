package orchestrator

import (
	"context"
	"errors"
	"strings"
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
				Phase:       state.PhaseValidate,
				AppID:       "cli_123",
				AuthSuccess: true,
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
			AppID:     "cli_123",
			AppURL:    "https://open.xfchat.iflytek.com/app/cli_123/baseinfo",
			AppSecret: "secret-123",
			AuthURL:   "https://open.xfchat.iflytek.com/oauth/authorize?app_id=cli_123",
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
	if got.AppID != "cli_123" || got.AppURL == "" || got.AppSecret != "secret-123" || got.AuthURL == "" {
		t.Fatalf("expected app metadata to persist, got %#v", got)
	}
}

func TestRunInitializesMissingStateAtPlatformSetup(t *testing.T) {
	root := t.TempDir()
	store := state.NewStore(root)

	orch := New(config.Config{}, store, "windows")
	orch.LoadState = func() (state.BootstrapState, error) {
		return store.Load()
	}
	orch.PlatformSetupRunner = stateAdvanceRunnerFunc(func(_ context.Context, current state.BootstrapState) (state.BootstrapState, error) {
		if current.Phase != state.PhasePlatformSetup {
			t.Fatalf("expected initialized platform setup phase, got %#v", current)
		}
		next := current
		next.AppID = "cli_123"
		next.AppURL = "https://open.xfchat.iflytek.com/app/cli_123/baseinfo"
		next.AppSecret = "secret-123"
		next.AuthURL = "https://open.xfchat.iflytek.com/oauth/authorize?app_id=cli_123"
		return next, nil
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
	if got.Phase != state.PhaseValidate || !got.AuthSuccess || got.AppID != "cli_123" {
		t.Fatalf("expected initialized bootstrap to complete, got %#v", got)
	}
}

func TestRunUsesRuntimeCallbackURLForPlatformSetupAndOAuth(t *testing.T) {
	root := t.TempDir()
	store := state.NewStore(root)
	if err := store.Save(state.BootstrapState{
		Phase: state.PhasePlatformSetup,
	}); err != nil {
		t.Fatalf("seed state failed: %v", err)
	}

	waiter := waiterFunc{
		url: "http://127.0.0.1:18081/callback",
		wait: func(context.Context) (CallbackResult, error) {
			return CallbackResult{Code: "code-123"}, nil
		},
	}
	var gotSetupCallbackURL string
	var gotOAuthCallbackURL string
	var gotOAuthAuthURL string

	orch := New(config.Config{
		InstallRoot: root,
		CallbackURL: "http://localhost:8080/callback",
	}, store, "windows")
	orch.LoadState = func() (state.BootstrapState, error) {
		return store.Load()
	}
	orch.StartCallbackServer = func() (CallbackWaiter, error) {
		return waiter, nil
	}
	orch.PlatformSetupRunner = callbackStateAdvanceRunnerFunc(func(_ context.Context, current state.BootstrapState, callbackURL string) (state.BootstrapState, error) {
		gotSetupCallbackURL = callbackURL
		next := current
		next.AppID = "cli_123"
		next.AppURL = "https://open.xfchat.iflytek.com/app/cli_123/baseinfo"
		next.AppSecret = "secret-123"
		next.AuthURL = buildOAuthAuthorizationURL("cli_123", callbackURL, []string{"docx:document:readonly"})
		return next, nil
	})
	orch.OAuthRunner = callbackOAuthRunnerFunc(func(_ context.Context, current state.BootstrapState, callback CallbackWaiter) error {
		gotOAuthCallbackURL = callback.URL()
		gotOAuthAuthURL = current.AuthURL
		return nil
	})
	orch.Validate = func(context.Context) error { return nil }

	if err := orch.Run(context.Background()); err != nil {
		t.Fatalf("run failed: %v", err)
	}
	if gotSetupCallbackURL != "http://127.0.0.1:18081/callback" {
		t.Fatalf("expected runtime callback URL in setup, got %q", gotSetupCallbackURL)
	}
	if gotOAuthCallbackURL != "http://127.0.0.1:18081/callback" {
		t.Fatalf("expected same callback waiter in oauth, got %q", gotOAuthCallbackURL)
	}
	if !strings.Contains(gotOAuthAuthURL, "redirect_uri=http%3A%2F%2F127.0.0.1%3A18081%2Fcallback") {
		t.Fatalf("expected auth URL to use runtime redirect, got %q", gotOAuthAuthURL)
	}
}

type stateAdvanceRunnerFunc func(context.Context, state.BootstrapState) (state.BootstrapState, error)

func (f stateAdvanceRunnerFunc) RunState(ctx context.Context, current state.BootstrapState) (state.BootstrapState, error) {
	return f(ctx, current)
}

func (f stateAdvanceRunnerFunc) Run(context.Context, state.BootstrapState) error {
	return nil
}

type callbackStateAdvanceRunnerFunc func(context.Context, state.BootstrapState, string) (state.BootstrapState, error)

func (f callbackStateAdvanceRunnerFunc) RunStateWithCallbackURL(ctx context.Context, current state.BootstrapState, callbackURL string) (state.BootstrapState, error) {
	return f(ctx, current, callbackURL)
}

func (f callbackStateAdvanceRunnerFunc) RunState(ctx context.Context, current state.BootstrapState) (state.BootstrapState, error) {
	return f(ctx, current, "")
}

func (f callbackStateAdvanceRunnerFunc) Run(context.Context, state.BootstrapState) error {
	return nil
}

type callbackOAuthRunnerFunc func(context.Context, state.BootstrapState, CallbackWaiter) error

func (f callbackOAuthRunnerFunc) RunWithCallbackWaiter(ctx context.Context, current state.BootstrapState, callback CallbackWaiter) error {
	return f(ctx, current, callback)
}

func (f callbackOAuthRunnerFunc) Run(ctx context.Context, current state.BootstrapState) error {
	return f(ctx, current, waiterFunc{})
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
		t.Fatal("expected validation failure")
	}
	if !errors.Is(err, runtimeerrors.ErrValidationFailed) {
		t.Fatalf("expected validation failure, got %v", err)
	}
}

func TestNewAllowsRuntimeFallbackForLoadedPlatformPhase(t *testing.T) {
	var executeCalled bool
	var validateCalled bool
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
		executeCalled = true
		executedState = current
		return nil
	}
	orch.Validate = func(context.Context) error {
		validateCalled = true
		return nil
	}

	if err := orch.Run(context.Background()); err != nil {
		if !errors.Is(err, runtimeerrors.ErrValidationFailed) {
			t.Fatalf("run failed: %v", err)
		}
	}
	if !executeCalled {
		t.Fatal("expected execute to be called")
	}
	if validateCalled {
		t.Fatal("expected custom validate callback not to run when bootstrapper validation fails first")
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
		return nil
	}

	if err := orch.Run(context.Background()); err != nil {
		t.Fatalf("run failed: %v", err)
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
	store := state.NewStore(root)
	if err := store.Save(state.BootstrapState{
		Phase:       state.PhaseValidate,
		AppID:       "cli_123",
		AuthSuccess: true,
	}); err != nil {
		t.Fatalf("seed state failed: %v", err)
	}

	orch := New(config.Config{
		InstallRoot: root,
		CallbackURL: "http://localhost:8080/callback",
	}, store, "windows")

	if orch.LoadState == nil {
		t.Fatal("expected load state to be wired")
	}

	if err := orch.Run(context.Background()); err != nil {
		t.Fatalf("expected startup to succeed without external lark-cli, got %v", err)
	}
}
