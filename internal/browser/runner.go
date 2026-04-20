package browser

import (
	"context"

	"github.com/ssyamv/claude-code-skills/xfchat-bootstrapper/internal/orchestrator"
	"github.com/ssyamv/claude-code-skills/xfchat-bootstrapper/internal/state"
)

type Runner struct{}

func (Runner) Run(context.Context, state.BootstrapState) error {
	return orchestrator.ErrPlatformSetupUnimplemented
}
