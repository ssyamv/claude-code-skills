package orchestrator

import (
	"context"
	"errors"

	"github.com/ssyamv/claude-code-skills/xfchat-bootstrapper/internal/config"
	runtimeerrors "github.com/ssyamv/claude-code-skills/xfchat-bootstrapper/internal/errors"
	"github.com/ssyamv/claude-code-skills/xfchat-bootstrapper/internal/state"
)

type Orchestrator struct {
	LoadState           func() (state.BootstrapState, error)
	PlatformSetupRunner PlatformSetupRunner
	OAuthRunner         OAuthRunner
	Validate            func(context.Context) error
	Execute             func(context.Context, state.BootstrapState) error
}

func New(cfg config.Config, store *state.Store, platform string) Orchestrator {
	_ = cfg
	_ = platform
	return Orchestrator{
		LoadState: store.Load,
		Validate:  func(context.Context) error { return runtimeerrors.ErrValidationUnimplemented },
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
		return o.runValidate(ctx)
	}

	if current.Phase == state.PhaseOAuth {
		if err := o.runOAuth(ctx, current); err != nil {
			return err
		}
		return o.runValidate(ctx)
	}

	if err := o.runPlatformSetup(ctx, current); err != nil {
		return err
	}

	return o.runValidate(ctx)
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

func (o Orchestrator) runValidate(ctx context.Context) error {
	if o.Validate == nil {
		return nil
	}
	return o.Validate(ctx)
}

func (o Orchestrator) runPlatformSetup(ctx context.Context, current state.BootstrapState) error {
	if o.PlatformSetupRunner != nil {
		return o.PlatformSetupRunner.Run(ctx, current)
	}
	if o.Execute != nil {
		return o.Execute(ctx, current)
	}
	return nil
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
