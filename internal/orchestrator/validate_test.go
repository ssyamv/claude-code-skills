package orchestrator

import (
	"context"
	"testing"

	"github.com/ssyamv/claude-code-skills/xfchat-bootstrapper/internal/state"
)

func TestValidateRequiresAppAndAuthSuccess(t *testing.T) {
	validator := Validator{}

	if err := validator.Run(context.Background(), state.BootstrapState{
		Phase:       state.PhaseValidate,
		AppID:       "cli_123",
		AuthSuccess: true,
	}); err != nil {
		t.Fatalf("validate failed: %v", err)
	}
}
