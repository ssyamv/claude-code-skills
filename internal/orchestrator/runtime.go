package orchestrator

import (
	"context"
	"errors"

	"github.com/ssyamv/claude-code-skills/xfchat-bootstrapper/internal/state"
)

var ErrPlatformSetupUnimplemented = errors.New("platform setup runner not implemented")

var ErrOAuthUnimplemented = errors.New("oauth runner not implemented")

var ErrValidationUnimplemented = errors.New("validation runner not implemented")

// PlatformSetupRunner runs the internal platform setup phase.
type PlatformSetupRunner interface {
	Run(context.Context, state.BootstrapState) error
}

// OAuthRunner runs the internal OAuth phase.
type OAuthRunner interface {
	Run(context.Context, state.BootstrapState) error
}

type runnerFunc func(context.Context, state.BootstrapState) error

func (f runnerFunc) Run(ctx context.Context, current state.BootstrapState) error {
	return f(ctx, current)
}
