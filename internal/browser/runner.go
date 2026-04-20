package browser

import (
	"context"
	"fmt"

	"github.com/ssyamv/claude-code-skills/xfchat-bootstrapper/internal/state"
)

type Runner struct {
	Workflow Workflow
	Automate AutomateFunc
}

func (r Runner) Run(ctx context.Context, current state.BootstrapState) (PlatformSetupResult, error) {
	result, err := r.run(ctx, current)
	if err != nil {
		return PlatformSetupResult{}, err
	}
	if result.AppID == "" {
		result.AppID = current.AppID
	}
	if result.AppURL == "" {
		result.AppURL = current.AppURL
	}
	return result, nil
}

func (r Runner) RunState(ctx context.Context, current state.BootstrapState) (state.BootstrapState, error) {
	result, err := r.Run(ctx, current)
	if err != nil {
		return state.BootstrapState{}, err
	}
	if result.AppID == "" {
		return state.BootstrapState{}, fmt.Errorf("platform setup metadata missing app id")
	}
	next := current
	next.AppID = result.AppID
	next.AppURL = result.AppURL
	return next, nil
}

func (r Runner) run(ctx context.Context, current state.BootstrapState) (PlatformSetupResult, error) {
	if r.Automate == nil {
		return PlatformSetupResult{}, errRunnerUnimplemented
	}
	return r.Automate(ctx, r.Workflow)
}
