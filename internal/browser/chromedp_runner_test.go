package browser

import (
	"context"
	"errors"
	"testing"
)

func TestChromedpRunnerUsesWorkflowEntryURL(t *testing.T) {
	runner := ChromedpRunner{
		Navigate: func(_ context.Context, url string) error {
			if url != "https://open.xfchat.iflytek.com/app" {
				t.Fatalf("unexpected navigate url: %q", url)
			}
			return nil
		},
	}

	err := runner.OpenEntry(context.Background(), NewWorkflow(WorkflowConfig{
		AppEntryURL: "https://open.xfchat.iflytek.com/app",
	}))
	if err != nil {
		t.Fatalf("open entry failed: %v", err)
	}
}

func TestChromedpRunnerCapturesPlatformMetadata(t *testing.T) {
	runner := ChromedpRunner{
		ExtractAppID: func(context.Context) (string, error) { return "cli_123", nil },
		ExtractAppURL: func(context.Context) (string, error) {
			return "https://open.xfchat.iflytek.com/app/cli_123/baseinfo", nil
		},
	}

	result, err := runner.CaptureMetadata(context.Background())
	if err != nil {
		t.Fatalf("capture failed: %v", err)
	}
	if result.AppID != "cli_123" {
		t.Fatalf("expected app id, got %#v", result)
	}
	if result.AppURL != "https://open.xfchat.iflytek.com/app/cli_123/baseinfo" {
		t.Fatalf("expected app url, got %#v", result)
	}
}

func TestChromedpRunnerRunsEntryThenCreateThenCapture(t *testing.T) {
	var calls []string
	runner := ChromedpRunner{
		Navigate: func(_ context.Context, url string) error {
			calls = append(calls, "navigate:"+url)
			return nil
		},
		ClickCreate: func(_ context.Context, wf Workflow) error {
			calls = append(calls, "create:"+wf.selectors.CreateButton)
			return nil
		},
		EnsureCallback: func(_ context.Context, wf Workflow) error {
			calls = append(calls, "callback:"+wf.selectors.CallbackInput)
			return nil
		},
		ApplyScopes: func(_ context.Context, wf Workflow) error {
			calls = append(calls, "scopes")
			for _, scope := range wf.RequiredScopes() {
				calls = append(calls, "scope:"+scope)
			}
			return nil
		},
		Publish: func(_ context.Context, wf Workflow) error {
			calls = append(calls, "publish:"+wf.selectors.PublishButton)
			return nil
		},
		ExtractAppID: func(context.Context) (string, error) {
			calls = append(calls, "app_id")
			return "cli_123", nil
		},
		ExtractAppURL: func(context.Context) (string, error) {
			calls = append(calls, "app_url")
			return "https://open.xfchat.iflytek.com/app/cli_123/baseinfo", nil
		},
	}

	result, err := runner.RunWorkflow(context.Background(), NewWorkflow(WorkflowConfig{
		AppEntryURL: "https://open.xfchat.iflytek.com/app",
		CallbackURL: "http://localhost:8080/callback",
		RequiredScopes: []string{
			"docs:document:readonly",
			"im:message:create_as_bot",
		},
	}))
	if err != nil {
		t.Fatalf("run workflow failed: %v", err)
	}
	if result.AppID != "cli_123" {
		t.Fatalf("expected app id, got %#v", result)
	}

	expected := []string{
		"navigate:https://open.xfchat.iflytek.com/app",
		"create:button[data-testid=\"create-app\"]",
		"callback:input[value=\"http://localhost:8080/callback\"]",
		"scopes",
		"scope:docs:document:readonly",
		"scope:im:message:create_as_bot",
		"publish:button[data-testid=\"publish\"]",
		"app_id",
		"app_url",
	}
	if len(calls) != len(expected) {
		t.Fatalf("expected %d calls, got %d (%v)", len(expected), len(calls), calls)
	}
	for i := range expected {
		if calls[i] != expected[i] {
			t.Fatalf("call %d: expected %q, got %q", i, expected[i], calls[i])
		}
	}
}

func TestNewDefaultAutomateReturnsResolverError(t *testing.T) {
	resolver := ProfileResolver{
		LookPath: func(string) (string, error) {
			return "", errNotFound
		},
	}

	automate := NewDefaultAutomate(resolver, "darwin")
	_, err := automate(context.Background(), NewWorkflow(WorkflowConfig{
		AppEntryURL: "https://open.xfchat.iflytek.com/app",
		CallbackURL: "http://localhost:8080/callback",
	}))
	if !errors.Is(err, errNotFound) {
		t.Fatalf("expected resolver error, got %v", err)
	}
}
