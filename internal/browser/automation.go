package browser

import (
	"context"

	runtimeerrors "github.com/ssyamv/claude-code-skills/xfchat-bootstrapper/internal/errors"
)

var errRunnerUnimplemented = runtimeerrors.ErrPlatformSetupUnimplemented

type PlatformSetupResult struct {
	AppID     string
	AppURL    string
	AppSecret string
	AuthURL   string
}

type AutomateFunc func(context.Context, Workflow) (PlatformSetupResult, error)
