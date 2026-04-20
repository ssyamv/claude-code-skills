package orchestrator

import (
	"context"
	"errors"
	"os/exec"

	"github.com/ssyamv/claude-code-skills/xfchat-bootstrapper/internal/browser"
	"github.com/ssyamv/claude-code-skills/xfchat-bootstrapper/internal/config"
	"github.com/ssyamv/claude-code-skills/xfchat-bootstrapper/internal/state"
)

type Orchestrator struct {
	LoadState           func() (state.BootstrapState, error)
	SaveState           func(state.BootstrapState) error
	PlatformSetupRunner PlatformSetupRunner
	OAuthRunner         OAuthRunner
	Validate            func(context.Context) error
	Execute             func(context.Context, state.BootstrapState) error
}

func New(cfg config.Config, store *state.Store, platform string) Orchestrator {
	_ = platform
	platformRunner := browserPlatformSetupRunner{
		Runner: browser.Runner{
			Workflow: browser.NewWorkflow(browser.WorkflowConfig{
				AppEntryURL:    "https://open.xfchat.iflytek.com/app",
				CallbackURL:    cfg.CallbackURL,
				RequiredScopes: cfg.RequiredScopes,
			}),
			Automate: browser.NewDefaultAutomate(browser.ProfileResolver{
				LookPath: exec.LookPath,
			}, platform),
		},
	}
	return Orchestrator{
		LoadState:           store.Load,
		SaveState:           store.Save,
		PlatformSetupRunner: platformRunner,
		OAuthRunner:         Runner{StartCallbackServer: StartCallbackServer},
	}
}

func (o Orchestrator) Run(ctx context.Context) error {
	current, loaded, err := o.loadState()
	if err != nil {
		return err
	}
	if !loaded {
		return nil
	}

	if current.Phase == state.PhaseValidate {
		return o.runValidate(ctx, current)
	}

	if current.Phase == state.PhaseOAuth {
		if err := o.runOAuth(ctx, current); err != nil {
			return err
		}
		current.AuthSuccess = true
		current.Phase = state.PhaseValidate
		if err := o.saveState(current); err != nil {
			return err
		}
		return o.runValidate(ctx, current)
	}

	next, advanced, err := o.runPlatformSetup(ctx, current)
	if err != nil {
		return err
	}

	if advanced {
		next.Phase = state.PhaseOAuth
		if err := o.saveState(next); err != nil {
			return err
		}

		if err := o.runOAuth(ctx, next); err != nil {
			return err
		}

		next.AuthSuccess = true
		next.Phase = state.PhaseValidate
		if err := o.saveState(next); err != nil {
			return err
		}

		return o.runValidate(ctx, next)
	}

	return o.runValidate(ctx, current)
}

func (o Orchestrator) loadState() (state.BootstrapState, bool, error) {
	if o.LoadState == nil {
		return state.BootstrapState{}, false, nil
	}

	current, err := o.LoadState()
	if err != nil && !errors.Is(err, state.ErrStateNotFound) {
		return state.BootstrapState{}, false, err
	}
	if err != nil {
		return state.BootstrapState{}, false, nil
	}
	return current, true, nil
}

func (o Orchestrator) runValidate(ctx context.Context, current state.BootstrapState) error {
	if err := (Validator{}).Run(ctx, current); err != nil {
		return err
	}
	if o.Validate == nil {
		return nil
	}
	return o.Validate(ctx)
}

func (o Orchestrator) runPlatformSetup(ctx context.Context, current state.BootstrapState) (state.BootstrapState, bool, error) {
	if runner, ok := o.PlatformSetupRunner.(platformSetupStateRunner); ok {
		next, err := runner.RunState(ctx, current)
		return next, true, err
	}
	if o.PlatformSetupRunner != nil {
		return current, false, o.PlatformSetupRunner.Run(ctx, current)
	}
	if o.Execute != nil {
		return current, false, o.Execute(ctx, current)
	}
	return current, false, nil
}

func (o Orchestrator) runOAuth(ctx context.Context, current state.BootstrapState) error {
	if o.OAuthRunner != nil {
		return o.OAuthRunner.Run(ctx, current)
	}
	if o.Execute != nil {
		return o.Execute(ctx, current)
	}
	return ErrOAuthUnimplemented
}

func (o Orchestrator) saveState(current state.BootstrapState) error {
	if o.SaveState == nil {
		return nil
	}
	return o.SaveState(current)
}

type browserPlatformSetupRunner struct {
	Runner browser.Runner
}

func (r browserPlatformSetupRunner) Run(ctx context.Context, current state.BootstrapState) error {
	_, err := r.Runner.Run(ctx, current)
	return err
}

func (r browserPlatformSetupRunner) RunState(ctx context.Context, current state.BootstrapState) (state.BootstrapState, error) {
	return r.Runner.RunState(ctx, current)
}
