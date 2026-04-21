package orchestrator

import (
	"context"

	runtimeerrors "github.com/ssyamv/claude-code-skills/xfchat-bootstrapper/internal/errors"
	"github.com/ssyamv/claude-code-skills/xfchat-bootstrapper/internal/state"
)

var ErrPlatformSetupUnimplemented = runtimeerrors.ErrPlatformSetupUnimplemented
var ErrOAuthUnimplemented = runtimeerrors.ErrOAuthUnimplemented

// PlatformSetupRunner runs the internal platform setup phase.
type PlatformSetupRunner interface {
	Run(context.Context, state.BootstrapState) error
}

type platformSetupStateRunner interface {
	RunState(context.Context, state.BootstrapState) (state.BootstrapState, error)
}

type platformSetupCallbackStateRunner interface {
	RunStateWithCallbackURL(context.Context, state.BootstrapState, string) (state.BootstrapState, error)
}

// OAuthRunner runs the internal OAuth phase.
type OAuthRunner interface {
	Run(context.Context, state.BootstrapState) error
}

type oauthCallbackWaiterRunner interface {
	RunWithCallbackWaiter(context.Context, state.BootstrapState, CallbackWaiter) error
}

type runnerFunc func(context.Context, state.BootstrapState) error

func (f runnerFunc) Run(ctx context.Context, current state.BootstrapState) error {
	return f(ctx, current)
}
