package browser

import (
	"context"
	"errors"
	"testing"

	"github.com/ssyamv/claude-code-skills/xfchat-bootstrapper/internal/orchestrator"
	"github.com/ssyamv/claude-code-skills/xfchat-bootstrapper/internal/state"
)

func TestRunnerReturnsInternalUnimplementedErrorInsteadOfShellingOut(t *testing.T) {
	runner := Runner{}

	err := runner.Run(context.Background(), state.BootstrapState{
		Phase: state.PhasePlatformSetup,
	})
	if err == nil {
		t.Fatal("expected internal unimplemented error")
	}
	if !errors.Is(err, orchestrator.ErrPlatformSetupUnimplemented) {
		t.Fatalf("expected unimplemented error, got %v", err)
	}
}
