package orchestrator

import (
	"context"

	runtimeerrors "github.com/ssyamv/claude-code-skills/xfchat-bootstrapper/internal/errors"
	"github.com/ssyamv/claude-code-skills/xfchat-bootstrapper/internal/state"
)

type Validator struct{}

func (Validator) Run(_ context.Context, current state.BootstrapState) error {
	if current.AppID == "" || !current.AuthSuccess {
		return runtimeerrors.ErrValidationFailed
	}
	return nil
}
