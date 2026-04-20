package orchestrator

import (
	"context"
	"fmt"

	"github.com/ssyamv/claude-code-skills/xfchat-bootstrapper/internal/state"
)

// Runner is the internal OAuth phase runner skeleton.
type Runner struct {
	StartCallbackServer func() (CallbackWaiter, error)
	OpenAuthorization   func(context.Context, string, state.BootstrapState) error
}

func (r Runner) Run(ctx context.Context, current state.BootstrapState) error {
	start := r.StartCallbackServer
	if start == nil {
		start = StartCallbackServer
	}

	waiter, err := start()
	if err != nil {
		return err
	}

	if r.OpenAuthorization != nil {
		if err := r.OpenAuthorization(ctx, waiter.URL(), current); err != nil {
			return err
		}
	}

	result, err := waiter.Wait(ctx)
	if err != nil {
		return err
	}
	if result.Error != "" {
		if result.ErrorDescription != "" {
			return fmt.Errorf("oauth callback failed: %s: %s", result.Error, result.ErrorDescription)
		}
		return fmt.Errorf("oauth callback failed: %s", result.Error)
	}
	return nil
}
