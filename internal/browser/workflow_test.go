package browser

import "testing"

func TestWorkflowBuildsExpectedSteps(t *testing.T) {
	wf := NewWorkflow(WorkflowConfig{
		AppEntryURL: "https://open.xfchat.iflytek.com/app",
		CallbackURL: "http://localhost:8080/callback",
		RequiredScopes: []string{
			"docs:document:readonly",
			"im:message:create_as_bot",
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

	if wf.selectors.CallbackInput != `input[value="https://example.com/callback"]` {
		t.Fatalf("expected callback selector to use config callback url, got %q", wf.selectors.CallbackInput)
	}
}

func TestWorkflowExposesConfiguredEntryURLAndScopes(t *testing.T) {
	wf := NewWorkflow(WorkflowConfig{
		AppEntryURL: "https://open.xfchat.iflytek.com/app",
		RequiredScopes: []string{
			"docs:document:readonly",
			"im:message:create_as_bot",
		},
	})

	if got := wf.AppEntryURL(); got != "https://open.xfchat.iflytek.com/app" {
		t.Fatalf("expected app entry url to round-trip from config, got %q", got)
	}

	scopes := wf.RequiredScopes()
	if len(scopes) != 2 {
		t.Fatalf("expected 2 scopes, got %d", len(scopes))
	}
	if scopes[0] != "docs:document:readonly" || scopes[1] != "im:message:create_as_bot" {
		t.Fatalf("expected required scopes to round-trip from config, got %#v", scopes)
	}
}
