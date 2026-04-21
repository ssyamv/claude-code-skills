package platformapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/ssyamv/claude-code-skills/xfchat-bootstrapper/internal/browser"
)

type DoFunc func(*http.Request) (*http.Response, error)

type Client struct {
	BaseURL string
	Do      DoFunc
}

var defaultHTTPClient = &http.Client{Timeout: 30 * time.Second}

func (c Client) CreateApp(ctx context.Context, session browser.SessionContext, in CreateAppRequest) (CreateAppResult, error) {
	var payload struct {
		Data struct {
			Apps []struct {
				AppID string `json:"appID"`
				Name  string `json:"name"`
			} `json:"apps"`
		} `json:"data"`
	}
	body := map[string]any{
		"Count":  10,
		"Cursor": 0,
		"QueryFilter": map[string]any{
			"filterAppSceneTypeList": []int{0},
		},
		"OrderBy": 0,
	}
	if err := c.doJSON(ctx, session, http.MethodPost, "/developers/v1/app/list", "/app", body, &payload); err != nil {
		return CreateAppResult{}, err
	}
	for _, app := range payload.Data.Apps {
		if app.Name == in.Name && app.AppID != "" {
			return CreateAppResult{
				AppID:  app.AppID,
				AppURL: joinURL(c.baseURL(session), "/app/"+url.PathEscape(app.AppID)+"/baseinfo"),
			}, nil
		}
	}
	return c.createApp(ctx, session, in)
}

func (c Client) createApp(ctx context.Context, session browser.SessionContext, in CreateAppRequest) (CreateAppResult, error) {
	var payload struct {
		Data struct {
			AppID  string `json:"app_id"`
			AppURL string `json:"app_url"`
		} `json:"data"`
	}
	if err := c.doJSON(ctx, session, http.MethodPost, "/api/apps", "/app", in, &payload); err != nil {
		return CreateAppResult{}, err
	}
	if payload.Data.AppID == "" {
		return CreateAppResult{}, fmt.Errorf("create app response missing app id")
	}
	appURL := payload.Data.AppURL
	if appURL == "" {
		appURL = joinURL(c.baseURL(session), "/app/"+url.PathEscape(payload.Data.AppID)+"/baseinfo")
	}
	return CreateAppResult{
		AppID:  payload.Data.AppID,
		AppURL: appURL,
	}, nil
}

func (c Client) EnsureRedirectURL(ctx context.Context, session browser.SessionContext, in EnsureRedirectURLRequest) error {
	var payload struct {
		Data struct {
			RedirectURL []string `json:"redirectURL"`
		} `json:"data"`
	}
	if err := c.doJSON(ctx, session, http.MethodPost, "/developers/v1/safe_setting/"+url.PathEscape(in.AppID), "/app/"+url.PathEscape(in.AppID)+"/safe", map[string]any{}, &payload); err != nil {
		return err
	}
	for _, existing := range payload.Data.RedirectURL {
		if existing == in.CallbackURL {
			return nil
		}
	}
	return fmt.Errorf("callback url %q is not configured", in.CallbackURL)
}

func (c Client) EnsureScopes(ctx context.Context, session browser.SessionContext, in EnsureScopesRequest) error {
	return c.doJSON(ctx, session, http.MethodPost, "/developers/v1/scope/all/"+url.PathEscape(in.AppID), "/app/"+url.PathEscape(in.AppID)+"/auth", map[string]any{}, nil)
}

func (c Client) GetAppCredentials(ctx context.Context, session browser.SessionContext, appID string) (CredentialsResult, error) {
	var payload struct {
		Data struct {
			AppSecret string `json:"secret"`
		} `json:"data"`
	}
	if err := c.doJSON(ctx, session, http.MethodPost, "/developers/v1/secret/"+url.PathEscape(appID), "/app/"+url.PathEscape(appID)+"/baseinfo", map[string]any{}, &payload); err != nil {
		return CredentialsResult{}, err
	}
	if payload.Data.AppSecret == "" {
		return CredentialsResult{}, fmt.Errorf("get app credentials response missing app secret")
	}
	return CredentialsResult{
		AppID:     appID,
		AppSecret: payload.Data.AppSecret,
		AppURL:    joinURL(c.baseURL(session), "/app/"+url.PathEscape(appID)+"/baseinfo"),
	}, nil
}

func (c Client) CreateVersion(ctx context.Context, session browser.SessionContext, appID string) error {
	return c.doJSON(ctx, session, http.MethodPost, "/developers/v1/app_version/list/"+url.PathEscape(appID), "/app/"+url.PathEscape(appID)+"/version", map[string]any{}, nil)
}

func (c Client) PublishVersion(ctx context.Context, session browser.SessionContext, in PublishVersionRequest) error {
	return c.doJSON(ctx, session, http.MethodPost, "/developers/v1/app_version/list/"+url.PathEscape(in.AppID), "/app/"+url.PathEscape(in.AppID)+"/version", map[string]any{}, nil)
}

func (c Client) doJSON(ctx context.Context, session browser.SessionContext, method, requestPath, refererPath string, in any, out any) error {
	requestURL, refererURL, err := c.resolveURLs(session, requestPath, refererPath)
	if err != nil {
		return err
	}

	var body io.Reader
	if in != nil {
		payload, err := json.Marshal(in)
		if err != nil {
			return err
		}
		body = bytes.NewReader(payload)
	}

	req, err := http.NewRequestWithContext(ctx, method, requestURL, body)
	if err != nil {
		return err
	}

	req.Header = session.HeaderForRequest(requestURL, refererURL)
	req.Header.Set("Accept", "application/json")
	if in != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	do := c.Do
	if do == nil {
		do = defaultHTTPClient.Do
	}
	resp, err := do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		snippet, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		snippet = bytes.TrimSpace(snippet)
		if len(snippet) > 0 {
			return fmt.Errorf("%s %s: unexpected status %d: %s", method, requestPath, resp.StatusCode, string(snippet))
		}
		return fmt.Errorf("%s %s: unexpected status %d", method, requestPath, resp.StatusCode)
	}

	if out == nil {
		return nil
	}
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return fmt.Errorf("%s %s: decode response: %w", method, requestPath, err)
	}
	return nil
}

func (c Client) resolveURLs(session browser.SessionContext, requestPath, refererPath string) (string, string, error) {
	baseURL := c.baseURL(session)
	if baseURL == "" {
		return "", "", fmt.Errorf("platform client base url is empty")
	}

	return joinURL(baseURL, requestPath), joinURL(baseURL, refererPath), nil
}

func (c Client) baseURL(session browser.SessionContext) string {
	baseURL := c.BaseURL
	if baseURL == "" {
		baseURL = session.BaseURL
	}
	return strings.TrimRight(baseURL, "/")
}

func joinURL(baseURL, path string) string {
	return strings.TrimRight(baseURL, "/") + "/" + strings.TrimLeft(path, "/")
}
