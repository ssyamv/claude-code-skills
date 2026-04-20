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
