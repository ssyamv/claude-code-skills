package browser

import (
	"context"
	"errors"
	"testing"

	"github.com/ssyamv/claude-code-skills/xfchat-bootstrapper/internal/orchestrator"
	"github.com/ssyamv/claude-code-skills/xfchat-bootstrapper/internal/state"
)

func TestRunnerReturnsSharedUnimplementedErrorWhenAutomateIsNil(t *testing.T) {
	runner := Runner{}

	result, err := runner.Run(context.Background(), state.BootstrapState{})
	if err == nil {
		t.Fatal("expected unimplemented error")
	}
	if !errors.Is(err, orchestrator.ErrPlatformSetupUnimplemented) {
		t.Fatalf("expected shared unimplemented error, got %v", err)
	}
	if result != (PlatformSetupResult{}) {
		t.Fatalf("expected empty result, got %#v", result)
	}
}
