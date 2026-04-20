package browser

import (
	"context"

	runtimeerrors "github.com/ssyamv/claude-code-skills/xfchat-bootstrapper/internal/errors"
)

var errRunnerUnimplemented = runtimeerrors.ErrPlatformSetupUnimplemented

type PlatformSetupResult struct {
	AppID  string
	AppURL string
}

type AutomateFunc func(context.Context, Workflow) (PlatformSetupResult, error)
