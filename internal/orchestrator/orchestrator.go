package orchestrator

import (
	"context"
	"errors"
	"net/url"
	"os/exec"
	"strings"

	"github.com/ssyamv/claude-code-skills/xfchat-bootstrapper/internal/browser"
	"github.com/ssyamv/claude-code-skills/xfchat-bootstrapper/internal/config"
	"github.com/ssyamv/claude-code-skills/xfchat-bootstrapper/internal/platformapi"
	"github.com/ssyamv/claude-code-skills/xfchat-bootstrapper/internal/platformsetup"
	"github.com/ssyamv/claude-code-skills/xfchat-bootstrapper/internal/state"
)

type Orchestrator struct {
	LoadState           func() (state.BootstrapState, error)
	SaveState           func(state.BootstrapState) error
	StartCallbackServer func() (CallbackWaiter, error)
	PlatformSetupRunner PlatformSetupRunner
	OAuthRunner         OAuthRunner
	Validate            func(context.Context) error
	Execute             func(context.Context, state.BootstrapState) error
}

func New(cfg config.Config, store *state.Store, platform string) Orchestrator {
	workflow := browser.NewWorkflow(browser.WorkflowConfig{
		AppEntryURL:    "https://open.xfchat.iflytek.com/app",
		CallbackURL:    cfg.CallbackURL,
		RequiredScopes: cfg.RequiredScopes,
	})
	profileResolver := browser.ProfileResolver{
		LookPath: exec.LookPath,
	}
	platformClient := platformapi.Client{
		BaseURL: "https://open.xfchat.iflytek.com",
	}
	platformRunner := platformsetup.Runner{
		BootstrapSession: func(ctx context.Context) (browser.SessionContext, error) {
			profile, err := profileResolver.Resolve(platform)
			if err != nil {
				return browser.SessionContext{}, err
			}
			bootstrap := browser.NewSessionBootstrap(profile)
			return bootstrap.Bootstrap(ctx, profile, workflow.AppEntryURL())
		},
		CreateAppWithCallbackURL: func(ctx context.Context, session browser.SessionContext, callbackURL string) (platformsetup.Result, error) {
			if callbackURL == "" {
				callbackURL = cfg.CallbackURL
			}
			created, err := platformClient.CreateApp(ctx, session, platformapi.CreateAppRequest{
				Name: "lark_cli",
			})
			if err != nil {
				return platformsetup.Result{}, err
			}

			if err := platformClient.EnsureRedirectURL(ctx, session, platformapi.EnsureRedirectURLRequest{
				AppID:       created.AppID,
				CallbackURL: callbackURL,
			}); err != nil {
				return platformsetup.Result{}, err
			}

			if err := platformClient.EnsureScopes(ctx, session, platformapi.EnsureScopesRequest{
				AppID:  created.AppID,
				Scopes: workflow.RequiredScopes(),
			}); err != nil {
				return platformsetup.Result{}, err
			}

			if err := platformClient.CreateVersion(ctx, session, created.AppID); err != nil {
				return platformsetup.Result{}, err
			}
			if err := platformClient.PublishVersion(ctx, session, platformapi.PublishVersionRequest{
				AppID: created.AppID,
			}); err != nil {
				return platformsetup.Result{}, err
			}

			credentials, err := platformClient.GetAppCredentials(ctx, session, created.AppID)
			if err != nil {
				return platformsetup.Result{}, err
			}

			appURL := credentials.AppURL
			if appURL == "" {
				appURL = created.AppURL
			}
			if appURL == "" {
				appURL = workflow.BaseInfoURL(created.AppID)
			}

			appID := credentials.AppID
			if appID == "" {
				appID = created.AppID
			}

			return platformsetup.Result{
				AppID:     appID,
				AppURL:    appURL,
				AppSecret: credentials.AppSecret,
				AuthURL:   buildOAuthAuthorizationURL(appID, callbackURL, workflow.RequiredScopes()),
			}, nil
		},
	}
	return Orchestrator{
		LoadState:           store.Load,
		SaveState:           store.Save,
		StartCallbackServer: StartCallbackServer,
		PlatformSetupRunner: platformRunner,
		OAuthRunner: Runner{
			StartCallbackServer: StartCallbackServer,
			OpenAuthorization: func(ctx context.Context, callbackURL string, current state.BootstrapState) error {
				_ = callbackURL
				if current.AuthURL == "" {
					return ErrOAuthUnimplemented
				}
				profile, err := resolveBrowserProfileFn(platform)
				if err != nil {
					return err
				}
				return openOAuthURLWithProfile(ctx, profile, current.AuthURL)
			},
		},
	}
}

var openOAuthURLWithProfile = browser.OpenAndConfirmOAuthWithProfile
var resolveBrowserProfileFn = func(platform string) (browser.BrowserProfile, error) {
	return (browser.ProfileResolver{LookPath: exec.LookPath}).Resolve(platform)
}

func buildOAuthAuthorizationURL(appID, callbackURL string, scopes []string) string {
	values := url.Values{}
	values.Set("app_id", appID)
	if callbackURL != "" {
		values.Set("redirect_uri", callbackURL)
	}
	authPath := "/open-apis/authen/v1/index"
	if len(scopes) > 0 {
		values.Set("scope", strings.Join(scopes, " "))
		authPath = "/open-apis/authen/v1/authorize"
	}
	return "https://open.xfchat.iflytek.com" + authPath + "?" + values.Encode()
}

func (o Orchestrator) Run(ctx context.Context) error {
	current, loaded, err := o.loadState()
	if err != nil {
		return err
	}
	if !loaded {
		current = state.BootstrapState{Phase: state.PhasePlatformSetup}
		if err := o.saveState(current); err != nil {
			return err
		}
	}

	if current.Phase == state.PhaseValidate {
		return o.runValidate(ctx, current)
	}

	if current.Phase == state.PhaseOAuth {
		if err := o.runOAuth(ctx, current); err != nil {
			return err
		}
		current.AuthSuccess = true
		current.Phase = state.PhaseValidate
		if err := o.saveState(current); err != nil {
			return err
		}
		return o.runValidate(ctx, current)
	}

	callback, err := o.startCallbackServer()
	if err != nil {
		return err
	}

	next, advanced, err := o.runPlatformSetup(ctx, current, callback.URL())
	if err != nil {
		closeCallbackWaiter(callback)
		return err
	}

	if advanced {
		next.Phase = state.PhaseOAuth
		if err := o.saveState(next); err != nil {
			closeCallbackWaiter(callback)
			return err
		}

		if err := o.runOAuthWithCallbackWaiter(ctx, next, callback); err != nil {
			return err
		}

		next.AuthSuccess = true
		next.Phase = state.PhaseValidate
		if err := o.saveState(next); err != nil {
			return err
		}

		return o.runValidate(ctx, next)
	}

	closeCallbackWaiter(callback)
	return o.runValidate(ctx, current)
}

func (o Orchestrator) loadState() (state.BootstrapState, bool, error) {
	if o.LoadState == nil {
		return state.BootstrapState{}, false, nil
	}

	current, err := o.LoadState()
	if err != nil && !errors.Is(err, state.ErrStateNotFound) {
		return state.BootstrapState{}, false, err
	}
	if err != nil {
		return state.BootstrapState{}, false, nil
	}
	return current, true, nil
}

func (o Orchestrator) runValidate(ctx context.Context, current state.BootstrapState) error {
	if err := (Validator{}).Run(ctx, current); err != nil {
		return err
	}
	if o.Validate == nil {
		return nil
	}
	return o.Validate(ctx)
}

func (o Orchestrator) runPlatformSetup(ctx context.Context, current state.BootstrapState, callbackURL string) (state.BootstrapState, bool, error) {
	if runner, ok := o.PlatformSetupRunner.(platformSetupCallbackStateRunner); ok {
		next, err := runner.RunStateWithCallbackURL(ctx, current, callbackURL)
		return next, true, err
	}
	if runner, ok := o.PlatformSetupRunner.(platformSetupStateRunner); ok {
		next, err := runner.RunState(ctx, current)
		return next, true, err
	}
	if o.PlatformSetupRunner != nil {
		return current, false, o.PlatformSetupRunner.Run(ctx, current)
	}
	if o.Execute != nil {
		return current, false, o.Execute(ctx, current)
	}
	return current, false, nil
}

func (o Orchestrator) runOAuth(ctx context.Context, current state.BootstrapState) error {
	if o.OAuthRunner != nil {
		return o.OAuthRunner.Run(ctx, current)
	}
	if o.Execute != nil {
		return o.Execute(ctx, current)
	}
	return ErrOAuthUnimplemented
}

func (o Orchestrator) runOAuthWithCallbackWaiter(ctx context.Context, current state.BootstrapState, callback CallbackWaiter) error {
	if runner, ok := o.OAuthRunner.(oauthCallbackWaiterRunner); ok {
		return runner.RunWithCallbackWaiter(ctx, current, callback)
	}
	closeCallbackWaiter(callback)
	return o.runOAuth(ctx, current)
}

func (o Orchestrator) startCallbackServer() (CallbackWaiter, error) {
	start := o.StartCallbackServer
	if start == nil {
		start = StartCallbackServer
	}
	return start()
}

func (o Orchestrator) saveState(current state.BootstrapState) error {
	if o.SaveState == nil {
		return nil
	}
	return o.SaveState(current)
}
