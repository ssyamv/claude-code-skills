package orchestrator

import "github.com/ssyamv/claude-code-skills/xfchat-bootstrapper/internal/preflight"

type Orchestrator struct {
	Preflight preflight.Checker
}
