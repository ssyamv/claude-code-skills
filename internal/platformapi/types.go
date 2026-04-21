package platformapi

type CreateAppRequest struct {
	Name string `json:"name"`
}

type CreateAppResult struct {
	AppID  string
	AppURL string
}

type CredentialsResult struct {
	AppID     string
	AppSecret string
	AppURL    string
}

type EnsureRedirectURLRequest struct {
	AppID       string `json:"app_id"`
	CallbackURL string `json:"callback_url"`
}

type EnsureScopesRequest struct {
	AppID  string   `json:"app_id"`
	Scopes []string `json:"scopes"`
}

type CreateVersionRequest struct {
	AppID string `json:"app_id"`
}

type PublishVersionRequest struct {
	AppID string `json:"app_id"`
}
