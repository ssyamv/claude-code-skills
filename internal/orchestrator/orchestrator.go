package orchestrator

import (
	"context"
	"errors"

	"github.com/ssyamv/claude-code-skills/xfchat-bootstrapper/internal/config"
	"github.com/ssyamv/claude-code-skills/xfchat-bootstrapper/internal/state"
)

type Orchestrator struct {
	LoadState func() (state.BootstrapState, error)
	Validate  func(context.Context) error
	Execute   func(context.Context, state.BootstrapState) error
}

func New(cfg config.Config, store *state.Store, platform string) Orchestrator {
	_ = cfg
	_ = platform
	return Orchestrator{
		LoadState: store.Load,
	}
}

func (o Orchestrator) Run(ctx context.Context) error {
	current, err := o.loadState()
	if err != nil {
		return err
	}

	if current.Phase == state.PhaseValidate {
		return o.runValidate(ctx)
	}

	if err := o.runExecute(ctx, current); err != nil {
		return err
	}

	return o.runValidate(ctx)
}

func (o Orchestrator) loadState() (state.BootstrapState, error) {
	if o.LoadState == nil {
		return state.BootstrapState{}, nil
	}

	current, err := o.LoadState()
	if err != nil && !errors.Is(err, state.ErrStateNotFound) {
		return state.BootstrapState{}, err
	}
	return current, nil
}

func (o Orchestrator) runValidate(ctx context.Context) error {
	if o.Validate == nil {
		return nil
	}
	return o.Validate(ctx)
}

func (o Orchestrator) runExecute(ctx context.Context, current state.BootstrapState) error {
	if o.Execute == nil {
		return nil
	}
	return o.Execute(ctx, current)
}
