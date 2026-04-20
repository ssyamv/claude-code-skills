package main

import (
	"context"
	"log"
	"runtime"

	"github.com/ssyamv/claude-code-skills/xfchat-bootstrapper/internal/config"
	"github.com/ssyamv/claude-code-skills/xfchat-bootstrapper/internal/orchestrator"
	"github.com/ssyamv/claude-code-skills/xfchat-bootstrapper/internal/state"
)

func main() {
	cfg := config.Default()
	store := state.NewStore(cfg.InstallRoot)
	orch := orchestrator.New(cfg, store, runtime.GOOS)

	if err := orch.Run(context.Background()); err != nil {
		log.Fatal(err)
	}
}
