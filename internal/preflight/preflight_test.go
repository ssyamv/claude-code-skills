package preflight

import (
	"errors"
	"testing"

	"github.com/ssyamv/claude-code-skills/xfchat-bootstrapper/internal/browser"
)

func TestCheckRejectsUnsupportedBrowser(t *testing.T) {
	checker := Checker{
		DetectDefaultBrowser:  func() (string, error) { return "safari", nil },
		ResolveBrowserProfile: func(string) (browser.BrowserProfile, error) { return browser.BrowserProfile{}, nil },
		CheckPort8080:         func() error { return nil },
		CheckWritableRoot:     func() error { return nil },
	}

	result, err := checker.Run()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Supported {
		t.Fatal("expected unsupported browser to fail support check")
	}
	if result.Reason == "" {
		t.Fatal("expected actionable failure reason")
	}
}

func TestCheckRejectsMissingBrowserProfile(t *testing.T) {
	checker := Checker{
		DetectDefaultBrowser: func() (string, error) { return "chrome", nil },
		ResolveBrowserProfile: func(string) (browser.BrowserProfile, error) {
			return browser.BrowserProfile{}, errors.New("not found")
		},
		CheckPort8080:     func() error { return nil },
		CheckWritableRoot: func() error { return nil },
	}

	result, err := checker.Run()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Supported {
		t.Fatal("expected missing browser profile to fail support check")
	}
	if result.Reason != "browser profile could not be resolved" {
		t.Fatalf("unexpected reason: %q", result.Reason)
	}
}
