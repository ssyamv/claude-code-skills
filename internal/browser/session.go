package browser

import (
	"context"

	"github.com/chromedp/chromedp"
)

type sessionAllocatorConfig struct {
	execPath    string
	userDataDir string
	headless    bool
	noFirstRun  bool
}

func sessionAllocatorConfigFromProfile(profile BrowserProfile) sessionAllocatorConfig {
	return sessionAllocatorConfig{
		execPath:    profile.BinaryPath,
		userDataDir: profile.UserDataDir,
		headless:    false,
		noFirstRun:  true,
	}
}

func SessionOptions(profile BrowserProfile) []chromedp.ExecAllocatorOption {
	cfg := sessionAllocatorConfigFromProfile(profile)

	return []chromedp.ExecAllocatorOption{
		chromedp.ExecPath(cfg.execPath),
		chromedp.UserDataDir(cfg.userDataDir),
		chromedp.Flag("headless", cfg.headless),
		chromedp.Flag("no-first-run", cfg.noFirstRun),
	}
}

func OpenURLWithProfile(ctx context.Context, profile BrowserProfile, url string) error {
	allocCtx, cancelAlloc := chromedp.NewExecAllocator(ctx, SessionOptions(profile)...)
	defer cancelAlloc()

	taskCtx, cancelTask := chromedp.NewContext(allocCtx)
	defer cancelTask()

	return chromedp.Run(taskCtx, chromedp.Navigate(url))
}
