package browser

import (
	"context"

	"github.com/ssyamv/claude-code-skills/xfchat-bootstrapper/internal/orchestrator"
)

var errRunnerUnimplemented = orchestrator.ErrPlatformSetupUnimplemented

type PlatformSetupResult struct {
	AppID  string
	AppURL string
}

type AutomateFunc func(context.Context, Workflow) (PlatformSetupResult, error)
