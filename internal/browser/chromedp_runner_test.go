package browser

import (
	"context"
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
