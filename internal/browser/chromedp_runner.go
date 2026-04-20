package browser

import "context"

type ChromedpRunner struct {
	Navigate func(context.Context, string) error
}

func (r ChromedpRunner) OpenEntry(ctx context.Context, wf Workflow) error {
	if r.Navigate == nil {
		return errRunnerUnimplemented
	}
	return r.Navigate(ctx, wf.AppEntryURL())
}
