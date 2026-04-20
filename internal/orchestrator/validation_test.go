package orchestrator

import (
	"context"
	"errors"
	"testing"

	"github.com/ssyamv/claude-code-skills/xfchat-bootstrapper/internal/config"
	runtimeerrors "github.com/ssyamv/claude-code-skills/xfchat-bootstrapper/internal/errors"
	"github.com/ssyamv/claude-code-skills/xfchat-bootstrapper/internal/state"
)

func TestRunUsesInternalValidationWhenPhaseValidate(t *testing.T) {
	root := t.TempDir()

	orch := New(config.Config{
		InstallRoot: root,
		CallbackURL: "http://localhost:8080/callback",
	}, state.NewStore(root), "windows")
	orch.LoadState = func() (state.BootstrapState, error) {
		return state.BootstrapState{
			Phase: state.PhaseValidate,
		}, nil
	}
	orch.Execute = func(context.Context, state.BootstrapState) error {
		t.Fatal("expected validate phase to avoid shell-based execution")
		return nil
	}

	err := orch.Run(context.Background())
	if err == nil {
		t.Fatal("expected validate phase to fail bootstrapper-owned validation")
	}
	if !errors.Is(err, runtimeerrors.ErrValidationFailed) {
		t.Fatalf("expected validation failure, got %v", err)
	}
}
