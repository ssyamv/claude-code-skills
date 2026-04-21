package browser

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"testing"
)

func TestBootstrapSessionUsesSharedSessionAndPageURL(t *testing.T) {
	type sessionKey struct{}

	pageURL := "https://open.xfchat.iflytek.com/app"
	expectedSessionID := "shared-session"
	expectedSelector := `meta[name="csrf-token"]`

	var calls []string
	cleanupCalled := false

	bootstrap := SessionBootstrap{
		NewSession: func(ctx context.Context, profile BrowserProfile) (context.Context, func(), error) {
			if profile.BrowserName != "chrome" {
				t.Fatalf("expected browser profile to be passed through, got %#v", profile)
			}
			sessionCtx := context.WithValue(ctx, sessionKey{}, expectedSessionID)
			return sessionCtx, func() { cleanupCalled = true }, nil
		},
		LoadPage: func(ctx context.Context, gotURL string) error {
			if gotURL != pageURL {
				t.Fatalf("expected pageURL %q, got %q", pageURL, gotURL)
			}
			if got := ctx.Value(sessionKey{}); got != expectedSessionID {
				t.Fatalf("expected shared session context, got %#v", got)
			}
			calls = append(calls, "load")
			return nil
		},
		ReadCookies: func(ctx context.Context) ([]*http.Cookie, error) {
			if got := ctx.Value(sessionKey{}); got != expectedSessionID {
				t.Fatalf("expected shared session context, got %#v", got)
			}
			calls = append(calls, "cookies")
			return []*http.Cookie{{Name: "sid", Value: "cookie-123"}}, nil
		},
		ReadToken: func(ctx context.Context, selector string) (string, error) {
			if got := ctx.Value(sessionKey{}); got != expectedSessionID {
				t.Fatalf("expected shared session context, got %#v", got)
			}
			if selector != expectedSelector {
				t.Fatalf("expected selector %q, got %q", expectedSelector, selector)
			}
			calls = append(calls, "token")
			return "csrf-123", nil
		},
	}

	got, err := bootstrap.Bootstrap(context.Background(), BrowserProfile{BrowserName: "chrome"}, pageURL)
	if err != nil {
		t.Fatalf("bootstrap failed: %v", err)
	}
	if got.CSRFToken != "csrf-123" {
		t.Fatalf("expected csrf token, got %#v", got)
	}
	if len(got.Cookies) != 1 || got.Cookies[0].Name != "sid" {
		t.Fatalf("expected cookie capture, got %#v", got)
	}
	if got.BaseURL != "https://open.xfchat.iflytek.com" {
		t.Fatalf("expected normalized base URL, got %#v", got.BaseURL)
	}
	if want := []string{"load", "cookies", "token"}; fmt.Sprint(calls) != fmt.Sprint(want) {
		t.Fatalf("expected call order %v, got %v", want, calls)
	}
	if !cleanupCalled {
		t.Fatal("expected session cleanup to run")
	}
}

func TestBootstrapSessionCleansUpOnFailure(t *testing.T) {
	stageErr := errors.New("stage failure")

	tests := []struct {
		name       string
		loadErr    error
		cookiesErr error
		tokenErr   error
	}{
		{name: "load page failure", loadErr: stageErr},
		{name: "cookie failure", cookiesErr: stageErr},
		{name: "token failure", tokenErr: stageErr},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cleanupCalled := false
			bootstrap := SessionBootstrap{
				NewSession: func(ctx context.Context, profile BrowserProfile) (context.Context, func(), error) {
					return ctx, func() { cleanupCalled = true }, nil
				},
				LoadPage: func(context.Context, string) error {
					if tt.loadErr != nil {
						return tt.loadErr
					}
					return nil
				},
				ReadCookies: func(context.Context) ([]*http.Cookie, error) {
					if tt.cookiesErr != nil {
						return nil, tt.cookiesErr
					}
					return []*http.Cookie{{Name: "sid", Value: "cookie-123"}}, nil
				},
				ReadToken: func(context.Context, string) (string, error) {
					if tt.tokenErr != nil {
						return "", tt.tokenErr
					}
					return "csrf-123", nil
				},
			}

			_, err := bootstrap.Bootstrap(context.Background(), BrowserProfile{BrowserName: "chrome"}, "https://open.xfchat.iflytek.com/app")
			if !errors.Is(err, stageErr) {
				t.Fatalf("expected stage error, got %v", err)
			}
			if !cleanupCalled {
				t.Fatal("expected cleanup on failure")
			}
		})
	}
}

func TestNewSessionBootstrapFallsBackToConstructorProfile(t *testing.T) {
	defaultProfile := BrowserProfile{
		BrowserName: "chrome",
		BinaryPath:  "/opt/google/chrome",
		UserDataDir: "/tmp/chrome-profile",
	}

	bootstrap := NewSessionBootstrap(defaultProfile)

	var seen BrowserProfile
	original := newProfileSessionFn
	originalPrepare := prepareAutomationProfileFn
	newProfileSessionFn = func(ctx context.Context, profile BrowserProfile) (context.Context, func(), error) {
		seen = profile
		return ctx, func() {}, nil
	}
	prepareAutomationProfileFn = func(profile BrowserProfile) (BrowserProfile, func(), error) {
		return profile, func() {}, nil
	}
	t.Cleanup(func() {
		newProfileSessionFn = original
		prepareAutomationProfileFn = originalPrepare
	})

	_, cleanup, err := bootstrap.NewSession(context.Background(), BrowserProfile{})
	if err != nil {
		t.Fatalf("new session failed: %v", err)
	}
	if cleanup != nil {
		cleanup()
	}
	if seen != defaultProfile {
		t.Fatalf("expected constructor profile fallback, got %#v", seen)
	}
}
