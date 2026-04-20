package browser

type WorkflowConfig struct {
	AppEntryURL    string
	CallbackURL    string
	RequiredScopes []string
}

type Workflow struct {
	cfg       WorkflowConfig
	selectors Selectors
}

func NewWorkflow(cfg WorkflowConfig) Workflow {
	return Workflow{
		cfg:       cfg,
		selectors: DefaultSelectors(cfg.CallbackURL),
	}
}

func (w Workflow) StepNames() []string {
	return []string{
		"open-app-entry",
		"create-app",
		"capture-app-credentials",
		"ensure-callback-url",
		"apply-required-scopes",
		"publish-app-version",
	}
}

func (w Workflow) AppEntryURL() string {
	return w.cfg.AppEntryURL
}

func (w Workflow) RequiredScopes() []string {
	return append([]string(nil), w.cfg.RequiredScopes...)
}

func (w Workflow) BaseInfoURL(appID string) string {
	return "https://open.xfchat.iflytek.com/app/" + appID + "/baseinfo"
}

func (w Workflow) AuthURL(appID string) string {
	return "https://open.xfchat.iflytek.com/app/" + appID + "/auth"
}

func (w Workflow) EventURL(appID string) string {
	return "https://open.xfchat.iflytek.com/app/" + appID + "/event"
}

func (w Workflow) SafeURL(appID string) string {
	return "https://open.xfchat.iflytek.com/app/" + appID + "/safe"
}

func (w Workflow) VersionURL(appID string) string {
	return "https://open.xfchat.iflytek.com/app/" + appID + "/version"
}
