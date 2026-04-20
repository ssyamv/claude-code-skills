package orchestrator

import (
	"context"

	"github.com/ssyamv/claude-code-skills/xfchat-bootstrapper/internal/state"
)

// Runner is the internal OAuth phase runner skeleton.
type Runner struct{}

func (Runner) Run(context.Context, state.BootstrapState) error {
	return ErrOAuthUnimplemented
}
