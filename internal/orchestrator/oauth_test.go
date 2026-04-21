package orchestrator

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"testing"

	"github.com/ssyamv/claude-code-skills/xfchat-bootstrapper/internal/browser"
	"github.com/ssyamv/claude-code-skills/xfchat-bootstrapper/internal/config"
	"github.com/ssyamv/claude-code-skills/xfchat-bootstrapper/internal/state"
)

func TestRunReturnsInternalOAuthErrorInsteadOfMissingBinary(t *testing.T) {
	orch := Orchestrator{
		LoadState: func() (state.BootstrapState, error) {
			return state.BootstrapState{
				Phase:   state.PhaseOAuth,
				AppID:   "cli_123",
				AppURL:  "https://open.xfchat.iflytek.com/app/cli_123/baseinfo",
				AuthURL: "https://open.xfchat.iflytek.com/oauth/authorize?app_id=cli_123",
			}, nil
		},
	}

	err := orch.Run(context.Background())
	if err == nil {
		t.Fatal("expected internal oauth error")
	}
	if !errors.Is(err, ErrOAuthUnimplemented) {
		t.Fatalf("expected internal oauth sentinel, got %v", err)
	}
}

func TestRunnerWaitsForCallbackSuccess(t *testing.T) {
	var openedURL string
	runner := Runner{
		StartCallbackServer: func() (CallbackWaiter, error) {
			return waiterFunc{
				url: "http://127.0.0.1:18080/callback",
				wait: func(context.Context) (CallbackResult, error) {
					return CallbackResult{Code: "ok"}, nil
				},
			}, nil
		},
		OpenAuthorization: func(_ context.Context, url string, current state.BootstrapState) error {
			openedURL = current.AuthURL
			if current.Phase != state.PhaseOAuth {
				t.Fatalf("expected oauth state, got %#v", current)
			}
			if current.AuthURL == "" {
				t.Fatal("expected auth url to be threaded into opener")
			}
			if url != "http://127.0.0.1:18080/callback" {
				t.Fatalf("expected callback url to be passed to opener, got %q", url)
			}
			return nil
		},
	}

	if err := runner.Run(context.Background(), state.BootstrapState{
		Phase:   state.PhaseOAuth,
		AuthURL: "https://open.xfchat.iflytek.com/oauth/authorize?app_id=cli_123",
	}); err != nil {
		t.Fatalf("oauth run failed: %v", err)
	}
	if openedURL != "https://open.xfchat.iflytek.com/oauth/authorize?app_id=cli_123" {
		t.Fatalf("expected auth url to be passed to opener, got %q", openedURL)
	}
}

func TestRunnerRejectsCallbackWithoutCode(t *testing.T) {
	runner := Runner{
		StartCallbackServer: func() (CallbackWaiter, error) {
			return waiterFunc{
				url: "http://127.0.0.1:18080/callback",
				wait: func(context.Context) (CallbackResult, error) {
					return CallbackResult{}, nil
				},
			}, nil
		},
		OpenAuthorization: func(context.Context, string, state.BootstrapState) error {
			return nil
		},
	}

	err := runner.Run(context.Background(), state.BootstrapState{
		Phase:   state.PhaseOAuth,
		AuthURL: "https://open.xfchat.iflytek.com/oauth/authorize?app_id=cli_123",
	})
	if err == nil {
		t.Fatal("expected missing code to fail")
	}
	if !strings.Contains(err.Error(), "missing code") {
		t.Fatalf("expected missing-code error, got %v", err)
	}
}

func TestRunnerClosesCallbackWaiterWhenOpenAuthorizationFails(t *testing.T) {
	openErr := errors.New("open failed")
	waiter := &closeTrackingWaiter{
		url: "http://127.0.0.1:18080/callback",
	}
	runner := Runner{
		StartCallbackServer: func() (CallbackWaiter, error) {
			return waiter, nil
		},
		OpenAuthorization: func(context.Context, string, state.BootstrapState) error {
			return openErr
		},
	}

	err := runner.Run(context.Background(), state.BootstrapState{
		Phase:   state.PhaseOAuth,
		AuthURL: "https://open.xfchat.iflytek.com/oauth/authorize?app_id=cli_123",
	})
	if !errors.Is(err, openErr) {
		t.Fatalf("expected opener error, got %v", err)
	}
	if !waiter.closed {
		t.Fatal("expected waiter to be closed after opener failure")
	}
}

type closeTrackingWaiter struct {
	url    string
	closed bool
}

func (w *closeTrackingWaiter) URL() string {
	return w.url
}

func (w *closeTrackingWaiter) Wait(context.Context) (CallbackResult, error) {
	return CallbackResult{}, fmt.Errorf("wait should not be called")
}

func (w *closeTrackingWaiter) Close() error {
	w.closed = true
	return nil
}

func TestBuildOAuthAuthorizationURLIncludesCallbackAndScopes(t *testing.T) {
	got := buildOAuthAuthorizationURL("cli_123", "http://localhost:8080/callback", []string{
		"docx:document:readonly",
	})

	parsed, err := url.Parse(got)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if parsed.Path != "/open-apis/authen/v1/authorize" {
		t.Fatalf("expected oauth authorize path, got %q", parsed.Path)
	}
	values := parsed.Query()
	if values.Get("app_id") != "cli_123" {
		t.Fatalf("expected app_id query param, got %q", values.Get("app_id"))
	}
	if values.Get("redirect_uri") != "http://localhost:8080/callback" {
		t.Fatalf("expected redirect uri query param, got %q", values.Get("redirect_uri"))
	}
	if values.Get("scope") != "docx:document:readonly" {
		t.Fatalf("expected scopes to be encoded, got %q", values.Get("scope"))
	}
	if strings.Contains(got, "/app/cli_123/auth") {
		t.Fatalf("expected real oauth url, got %q", got)
	}
	if strings.Contains(got, "/oauth/authorize") {
		t.Fatalf("expected lark-cli authen endpoint, got %q", got)
	}
}

func TestNewOAuthOpenerUsesDetachedHelperSeam(t *testing.T) {
	oldOpen := openOAuthURLWithProfile
	oldResolve := resolveBrowserProfileFn
	oldStartCallback := startCallbackServerFn
	t.Cleanup(func() {
		openOAuthURLWithProfile = oldOpen
		resolveBrowserProfileFn = oldResolve
		startCallbackServerFn = oldStartCallback
	})

	var gotProfile browser.BrowserProfile
	var gotURL string
	openOAuthURLWithProfile = func(_ context.Context, profile browser.BrowserProfile, url string) error {
		gotProfile = profile
		gotURL = url
		return nil
	}
	resolveBrowserProfileFn = func(string) (browser.BrowserProfile, error) {
		return browser.BrowserProfile{BinaryPath: "/bin/browser", UserDataDir: "/tmp/profile"}, nil
	}
	startCallbackServerFn = func() (CallbackWaiter, error) {
		return waiterFunc{
			url: "http://127.0.0.1:18080/callback",
			wait: func(context.Context) (CallbackResult, error) {
				return CallbackResult{Code: "ok"}, nil
			},
		}, nil
	}

	orch := New(config.Config{
		InstallRoot: "/tmp",
		CallbackURL: "http://localhost:8080/callback",
	}, nil, "windows")
	runner := orch.OAuthRunner.(Runner)
	runner.StartCallbackServer = nil
	orch.OAuthRunner = runner

	if err := orch.OAuthRunner.Run(context.Background(), state.BootstrapState{
		Phase:   state.PhaseOAuth,
		AuthURL: "https://open.xfchat.iflytek.com/oauth/authorize?app_id=cli_123",
	}); err != nil {
		t.Fatalf("oauth run failed: %v", err)
	}
	if gotProfile.BinaryPath != "/bin/browser" {
		t.Fatalf("expected resolved browser profile to be used, got %#v", gotProfile)
	}
	if gotURL != "https://open.xfchat.iflytek.com/oauth/authorize?app_id=cli_123" {
		t.Fatalf("expected auth url to be forwarded, got %q", gotURL)
	}
}
