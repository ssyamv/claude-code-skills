package browser

import (
	"context"
	"strings"

	"github.com/chromedp/chromedp"
)

type ChromedpRunner struct {
	Navigate      func(context.Context, string) error
	ClickCreate   func(context.Context, Workflow) error
	ExtractAppID  func(context.Context) (string, error)
	ExtractAppURL func(context.Context) (string, error)
}

func (r ChromedpRunner) OpenEntry(ctx context.Context, wf Workflow) error {
	if r.Navigate == nil {
		return errRunnerUnimplemented
	}
	return r.Navigate(ctx, wf.AppEntryURL())
}

func (r ChromedpRunner) RunWorkflow(ctx context.Context, wf Workflow) (PlatformSetupResult, error) {
	if err := r.OpenEntry(ctx, wf); err != nil {
		return PlatformSetupResult{}, err
	}
	if r.ClickCreate != nil {
		if err := r.ClickCreate(ctx, wf); err != nil {
			return PlatformSetupResult{}, err
		}
	}
	return r.CaptureMetadata(ctx)
}

func (r ChromedpRunner) CaptureMetadata(ctx context.Context) (PlatformSetupResult, error) {
	if r.ExtractAppID == nil && r.ExtractAppURL == nil {
		return PlatformSetupResult{}, errRunnerUnimplemented
	}

	var result PlatformSetupResult

	if r.ExtractAppID != nil {
		appID, err := r.ExtractAppID(ctx)
		if err != nil {
			return PlatformSetupResult{}, err
		}
		result.AppID = appID
	}

	if r.ExtractAppURL != nil {
		appURL, err := r.ExtractAppURL(ctx)
		if err != nil {
			return PlatformSetupResult{}, err
		}
		result.AppURL = appURL
	}

	return result, nil
}

func NewDefaultAutomate(resolver ProfileResolver, goos string) AutomateFunc {
	return func(ctx context.Context, wf Workflow) (PlatformSetupResult, error) {
		profile, err := resolver.Resolve(goos)
		if err != nil {
			return PlatformSetupResult{}, err
		}

		allocCtx, cancelAlloc := chromedp.NewExecAllocator(ctx, SessionOptions(profile)...)
		defer cancelAlloc()

		taskCtx, cancelTask := chromedp.NewContext(allocCtx)
		defer cancelTask()

		runner := ChromedpRunner{
			Navigate: func(ctx context.Context, url string) error {
				return chromedp.Run(ctx, chromedp.Navigate(url))
			},
			ClickCreate: func(ctx context.Context, wf Workflow) error {
				return chromedp.Run(ctx, chromedp.Click(wf.selectors.CreateButton, chromedp.NodeVisible))
			},
			ExtractAppID: func(ctx context.Context) (string, error) {
				var appID string
				if err := chromedp.Run(ctx, chromedp.Text(wf.selectors.AppIDValue, &appID, chromedp.NodeVisible)); err != nil {
					return "", err
				}
				return strings.TrimSpace(appID), nil
			},
			ExtractAppURL: func(ctx context.Context) (string, error) {
				var url string
				if err := chromedp.Run(ctx, chromedp.Location(&url)); err != nil {
					return "", err
				}
				return url, nil
			},
		}

		if err := runner.OpenEntry(taskCtx, wf); err != nil {
			return PlatformSetupResult{}, err
		}
		return runner.RunWorkflow(taskCtx, wf)
	}
}
