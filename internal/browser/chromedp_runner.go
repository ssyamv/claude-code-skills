package browser

import "context"

type ChromedpRunner struct {
	Navigate      func(context.Context, string) error
	ExtractAppID  func(context.Context) (string, error)
	ExtractAppURL func(context.Context) (string, error)
}

func (r ChromedpRunner) OpenEntry(ctx context.Context, wf Workflow) error {
	if r.Navigate == nil {
		return errRunnerUnimplemented
	}
	return r.Navigate(ctx, wf.AppEntryURL())
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
