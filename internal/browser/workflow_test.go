package browser

import "testing"

func TestWorkflowBuildsExpectedSteps(t *testing.T) {
	wf := NewWorkflow(WorkflowConfig{
		AppEntryURL: "https://open.xfchat.iflytek.com/app",
		CallbackURL: "http://localhost:8080/callback",
		RequiredScopes: []string{
			"docx:document:readonly",
		},
	})

	steps := wf.StepNames()
	expected := []string{
		"open-app-entry",
		"create-app",
		"capture-app-credentials",
		"ensure-callback-url",
		"apply-required-scopes",
		"publish-app-version",
	}

	if len(steps) != len(expected) {
		t.Fatalf("expected %d steps, got %d", len(expected), len(steps))
	}

	for i := range expected {
		if steps[i] != expected[i] {
			t.Fatalf("step %d: expected %q, got %q", i, expected[i], steps[i])
		}
	}
}

func TestWorkflowThreadsCallbackURLIntoSelectors(t *testing.T) {
	wf := NewWorkflow(WorkflowConfig{
		CallbackURL: "https://example.com/callback",
	})

	if wf.selectors.CallbackInput != `input[value="https://example.com/callback"], textarea` {
		t.Fatalf("expected callback selector to use config callback url, got %q", wf.selectors.CallbackInput)
	}
}

func TestWorkflowExposesConfiguredEntryURLAndScopes(t *testing.T) {
	wf := NewWorkflow(WorkflowConfig{
		AppEntryURL: "https://open.xfchat.iflytek.com/app",
		RequiredScopes: []string{
			"docx:document:readonly",
		},
	})

	if got := wf.AppEntryURL(); got != "https://open.xfchat.iflytek.com/app" {
		t.Fatalf("expected app entry url to round-trip from config, got %q", got)
	}

	scopes := wf.RequiredScopes()
	if len(scopes) != 1 {
		t.Fatalf("expected 1 scope, got %d", len(scopes))
	}
	if scopes[0] != "docx:document:readonly" {
		t.Fatalf("expected required scopes to round-trip from config, got %#v", scopes)
	}
}

func TestWorkflowBuildsKnownAppSubRoutes(t *testing.T) {
	wf := NewWorkflow(WorkflowConfig{})

	if got := wf.BaseInfoURL("cli_123"); got != "https://open.xfchat.iflytek.com/app/cli_123/baseinfo" {
		t.Fatalf("unexpected baseinfo url: %q", got)
	}
	if got := wf.AuthURL("cli_123"); got != "https://open.xfchat.iflytek.com/app/cli_123/auth" {
		t.Fatalf("unexpected auth url: %q", got)
	}
	if got := wf.EventURL("cli_123"); got != "https://open.xfchat.iflytek.com/app/cli_123/event" {
		t.Fatalf("unexpected event url: %q", got)
	}
	if got := wf.SafeURL("cli_123"); got != "https://open.xfchat.iflytek.com/app/cli_123/safe" {
		t.Fatalf("unexpected safe url: %q", got)
	}
	if got := wf.VersionURL("cli_123"); got != "https://open.xfchat.iflytek.com/app/cli_123/version" {
		t.Fatalf("unexpected version url: %q", got)
	}
}
