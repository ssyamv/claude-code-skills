package platformsetup

import (
	"context"
	"fmt"

	"github.com/ssyamv/claude-code-skills/xfchat-bootstrapper/internal/browser"
	"github.com/ssyamv/claude-code-skills/xfchat-bootstrapper/internal/state"
)

type Result struct {
	AppID     string
	AppURL    string
	AppSecret string
	AuthURL   string
}

type Runner struct {
	BootstrapSession         func(context.Context) (browser.SessionContext, error)
	CreateApp                func(context.Context, browser.SessionContext) (Result, error)
	CreateAppWithCallbackURL func(context.Context, browser.SessionContext, string) (Result, error)
}

func (r Runner) Run(ctx context.Context, current state.BootstrapState) error {
	_, err := r.RunState(ctx, current)
	return err
}

func (r Runner) RunState(ctx context.Context, current state.BootstrapState) (state.BootstrapState, error) {
	return r.RunStateWithCallbackURL(ctx, current, "")
}

func (r Runner) RunStateWithCallbackURL(ctx context.Context, current state.BootstrapState, callbackURL string) (state.BootstrapState, error) {
	if r.BootstrapSession == nil || (r.CreateApp == nil && r.CreateAppWithCallbackURL == nil) {
		return state.BootstrapState{}, fmt.Errorf("platform setup runner is not configured")
	}

	session, err := r.BootstrapSession(ctx)
	if err != nil {
		return state.BootstrapState{}, err
	}

	var result Result
	if r.CreateAppWithCallbackURL != nil {
		result, err = r.CreateAppWithCallbackURL(ctx, session, callbackURL)
	} else {
		result, err = r.CreateApp(ctx, session)
	}
	if err != nil {
		return state.BootstrapState{}, err
	}
	if result.AppID == "" {
		return state.BootstrapState{}, fmt.Errorf("platform setup result missing app id")
	}

	next := current
	next.AppID = result.AppID
	next.AppURL = result.AppURL
	next.AppSecret = result.AppSecret
	next.AuthURL = result.AuthURL
	return next, nil
}
