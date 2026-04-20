package browser

type fakeRunner struct {
	steps []string
}

func (r *fakeRunner) Run(step string) {
	r.steps = append(r.steps, step)
}
