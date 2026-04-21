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

var startCallbackServerFn = StartCallbackServer

func (r Runner) Run(ctx context.Context, current state.BootstrapState) error {
	start := r.StartCallbackServer
	if start == nil {
		start = startCallbackServerFn
	}
	if r.OpenAuthorization == nil {
		return ErrOAuthUnimplemented
	}

	waiter, err := start()
	if err != nil {
		return err
	}

	return r.RunWithCallbackWaiter(ctx, current, waiter)
}

func (r Runner) RunWithCallbackWaiter(ctx context.Context, current state.BootstrapState, waiter CallbackWaiter) error {
	if r.OpenAuthorization == nil {
		closeCallbackWaiter(waiter)
		return ErrOAuthUnimplemented
	}

	if err := r.OpenAuthorization(ctx, waiter.URL(), current); err != nil {
		closeCallbackWaiter(waiter)
		return err
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
	if result.Code == "" {
		return fmt.Errorf("oauth callback missing code")
	}
	return nil
}

type callbackCloser interface {
	Close() error
}

func closeCallbackWaiter(waiter CallbackWaiter) {
	if closer, ok := waiter.(callbackCloser); ok {
		_ = closer.Close()
	}
}
