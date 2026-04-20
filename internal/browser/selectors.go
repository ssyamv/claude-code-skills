package browser

type Selectors struct {
	CreateButton     string
	AppIDValue       string
	AppSecretReveal  string
	CallbackInput    string
	ScopeSearchInput string
	PublishButton    string
}

func DefaultSelectors(callbackURL string) Selectors {
	return Selectors{
		CreateButton:     `button[data-testid="create-app"]`,
		AppIDValue:       `[data-testid="app-id-value"]`,
		AppSecretReveal:  `button[data-testid="reveal-secret"]`,
		CallbackInput:    `input[value="` + callbackURL + `"]`,
		ScopeSearchInput: `input[placeholder*="scope"]`,
		PublishButton:    `button[data-testid="publish"]`,
	}
}
