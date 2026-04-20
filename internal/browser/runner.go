package browser

import (
	"context"

	"github.com/ssyamv/claude-code-skills/xfchat-bootstrapper/internal/state"
)

type Runner struct {
	Workflow Workflow
	Automate AutomateFunc
}

func (r Runner) Run(ctx context.Context, current state.BootstrapState) (PlatformSetupResult, error) {
	_ = current

	if r.Automate == nil {
		return PlatformSetupResult{}, errRunnerUnimplemented
	}
	return r.Automate(ctx, r.Workflow)
}
