package orchestrator

import (
	"context"
	"errors"
	"path/filepath"

	"github.com/ssyamv/claude-code-skills/xfchat-bootstrapper/internal/config"
	"github.com/ssyamv/claude-code-skills/xfchat-bootstrapper/internal/larkcli"
	"github.com/ssyamv/claude-code-skills/xfchat-bootstrapper/internal/state"
)

type Orchestrator struct {
	LoadState func() (state.BootstrapState, error)
	Validate  func(context.Context) error
	Execute   func(context.Context, state.BootstrapState) error
}

func New(cfg config.Config, store *state.Store, platform string) Orchestrator {
	cli := larkcli.Adapter{
		BinaryPath: binaryPath(cfg.InstallRoot, platform),
	}

	return Orchestrator{
		LoadState: store.Load,
		Validate: func(ctx context.Context) error {
			_, _, err := cli.Run(ctx, []string{"validate", "--callback-url", cfg.CallbackURL}, nil)
			return err
		},
		Execute: func(ctx context.Context, current state.BootstrapState) error {
			args := []string{"bootstrap", "--install-root", cfg.InstallRoot}
			if current.Phase != "" {
				args = append(args, "--resume-phase", string(current.Phase))
			}
			if current.AppID != "" {
				args = append(args, "--app-id", current.AppID)
			}
			if current.AppURL != "" {
				args = append(args, "--app-url", current.AppURL)
			}

			if _, _, err := cli.Run(ctx, args, nil); err != nil {
				return err
			}

			return store.Save(state.BootstrapState{
				Phase:     state.PhaseValidate,
				AppID:     current.AppID,
				AppURL:    current.AppURL,
				LastError: current.LastError,
			})
		},
	}
}

func (o Orchestrator) Run(ctx context.Context) error {
	current, err := o.loadState()
	if err != nil {
		return err
	}

	if current.Phase == state.PhaseValidate {
		return o.runValidate(ctx)
	}

	if err := o.runExecute(ctx, current); err != nil {
		return err
	}

	return o.runValidate(ctx)
}

func (o Orchestrator) loadState() (state.BootstrapState, error) {
	if o.LoadState == nil {
		return state.BootstrapState{}, nil
	}

	current, err := o.LoadState()
	if err != nil && !errors.Is(err, state.ErrStateNotFound) {
		return state.BootstrapState{}, err
	}
	return current, nil
}

func (o Orchestrator) runValidate(ctx context.Context) error {
	if o.Validate == nil {
		return nil
	}
	return o.Validate(ctx)
}

func (o Orchestrator) runExecute(ctx context.Context, current state.BootstrapState) error {
	if o.Execute == nil {
		return nil
	}
	return o.Execute(ctx, current)
}

func binaryPath(installRoot, platform string) string {
	name := "lark-cli"
	if platform == "windows" {
		name = "lark-cli.exe"
	}
	return filepath.Join(installRoot, "XfchatLarkCli", "bin", name)
}
