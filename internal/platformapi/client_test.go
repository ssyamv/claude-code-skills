package platformapi

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/ssyamv/claude-code-skills/xfchat-bootstrapper/internal/browser"
)

func TestClientCreateAppUsesSessionHeadersAndParsesResponse(t *testing.T) {
	t.Helper()

	var gotMethod string
	var gotPath string
	var gotCookie string
	var gotReferer string
	var gotOrigin string
	var gotXSRF string
	var gotContentType string
	var gotBody string

	client := Client{
		BaseURL: "https://open.xfchat.iflytek.com",
		Do: func(req *http.Request) (*http.Response, error) {
			gotMethod = req.Method
			gotPath = req.URL.Path
			gotCookie = req.Header.Get("Cookie")
			gotReferer = req.Header.Get("Referer")
			gotOrigin = req.Header.Get("Origin")
			gotXSRF = req.Header.Get("X-XSRF-TOKEN")
			gotContentType = req.Header.Get("Content-Type")
			body, _ := io.ReadAll(req.Body)
			gotBody = string(body)

			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(`{"data":{"apps":[{"appID":"cli_123","name":"lark_cli"}]}}`)),
			}, nil
		},
	}

	result, err := client.CreateApp(context.Background(), browser.SessionContext{
		BaseURL: "https://open.xfchat.iflytek.com",
		Cookies: []*http.Cookie{
			{Name: "sid", Value: "api-cookie", Domain: "open.xfchat.iflytek.com", Path: "/developers"},
			{Name: "sid", Value: "app-cookie", Domain: "open.xfchat.iflytek.com", Path: "/app"},
			{Name: "sid", Value: "root", Domain: "open.xfchat.iflytek.com", Path: "/"},
		},
		Headers: map[string]string{"X-XSRF-TOKEN": "csrf-123"},
	}, CreateAppRequest{Name: "lark_cli"})
	if err != nil {
		t.Fatalf("create app failed: %v", err)
	}

	if gotMethod != http.MethodPost {
		t.Fatalf("expected post, got %q", gotMethod)
	}
	if gotPath != "/developers/v1/app/list" {
		t.Fatalf("expected app list path, got %q", gotPath)
	}
	if gotCookie != "sid=api-cookie; sid=root" {
		t.Fatalf("expected request-url cookie selection, got %q", gotCookie)
	}
	if gotReferer != "https://open.xfchat.iflytek.com/app" {
		t.Fatalf("expected referer header, got %q", gotReferer)
	}
	if gotOrigin != "https://open.xfchat.iflytek.com" {
		t.Fatalf("expected origin header, got %q", gotOrigin)
	}
	if gotXSRF != "csrf-123" {
		t.Fatalf("expected xsrf header, got %q", gotXSRF)
	}
	if gotContentType != "application/json" {
		t.Fatalf("expected json content type, got %q", gotContentType)
	}
	if !strings.Contains(gotBody, `"filterAppSceneTypeList":[0]`) {
		t.Fatalf("expected app-list filter in body, got %q", gotBody)
	}
	if result.AppID != "cli_123" {
		t.Fatalf("expected app id, got %#v", result)
	}
	if result.AppURL != "https://open.xfchat.iflytek.com/app/cli_123/baseinfo" {
		t.Fatalf("expected app url, got %#v", result)
	}
}

func TestClientEnsureRedirectURLReturnsErrorOnUnexpectedStatus(t *testing.T) {
	t.Helper()

	var gotMethod string
	var gotPath string
	var gotCookie string
	var gotReferer string
	var gotOrigin string
	var gotXSRF string
	var gotContentType string
	var gotBody string

	client := Client{
		BaseURL: "https://open.xfchat.iflytek.com",
		Do: func(req *http.Request) (*http.Response, error) {
			gotMethod = req.Method
			gotPath = req.URL.Path
			gotCookie = req.Header.Get("Cookie")
			gotReferer = req.Header.Get("Referer")
			gotOrigin = req.Header.Get("Origin")
			gotXSRF = req.Header.Get("X-XSRF-TOKEN")
			gotContentType = req.Header.Get("Content-Type")
			body, _ := io.ReadAll(req.Body)
			gotBody = string(body)

			return &http.Response{
				StatusCode: http.StatusServiceUnavailable,
				Body:       io.NopCloser(strings.NewReader(`{"error":"unavailable"}`)),
			}, nil
		},
	}

	err := client.EnsureRedirectURL(context.Background(), browser.SessionContext{
		BaseURL: "https://open.xfchat.iflytek.com",
		Cookies: []*http.Cookie{{Name: "sid", Value: "cookie-123", Domain: "open.xfchat.iflytek.com", Path: "/developers/v1"}},
		Headers: map[string]string{"X-XSRF-TOKEN": "csrf-123"},
	}, EnsureRedirectURLRequest{
		AppID:       "cli_123",
		CallbackURL: "http://localhost:8080/callback",
	})
	if err == nil {
		t.Fatal("expected redirect update to fail")
	}
	if !strings.Contains(err.Error(), "unexpected status 503") {
		t.Fatalf("expected unexpected-status error, got %v", err)
	}
	if gotMethod != http.MethodPost {
		t.Fatalf("expected post request, got %q", gotMethod)
	}
	if gotPath != "/developers/v1/safe_setting/cli_123" {
		t.Fatalf("expected safe endpoint path, got %q", gotPath)
	}
	if gotCookie != "sid=cookie-123" {
		t.Fatalf("expected request-scoped cookie selection, got %q", gotCookie)
	}
	if gotReferer != "https://open.xfchat.iflytek.com/app/cli_123/safe" {
		t.Fatalf("expected referer header for safe page, got %q", gotReferer)
	}
	if gotOrigin != "https://open.xfchat.iflytek.com" {
		t.Fatalf("expected origin header, got %q", gotOrigin)
	}
	if gotXSRF != "csrf-123" {
		t.Fatalf("expected xsrf header to be forwarded, got %q", gotXSRF)
	}
	if gotContentType != "application/json" {
		t.Fatalf("expected json content type, got %q", gotContentType)
	}
	if gotBody != "{}" {
		t.Fatalf("expected empty request body, got %q", gotBody)
	}
}

func TestClientEnsureScopesReturnsErrorOnUnexpectedStatus(t *testing.T) {
	t.Helper()

	var gotMethod string
	var gotPath string
	var gotCookie string
	var gotBody string

	client := Client{
		BaseURL: "https://open.xfchat.iflytek.com",
		Do: func(req *http.Request) (*http.Response, error) {
			gotMethod = req.Method
			gotPath = req.URL.Path
			gotCookie = req.Header.Get("Cookie")
			body, _ := io.ReadAll(req.Body)
			gotBody = string(body)

			return &http.Response{
				StatusCode: http.StatusBadRequest,
				Body:       io.NopCloser(strings.NewReader(`{"error":"bad request"}`)),
			}, nil
		},
	}

	err := client.EnsureScopes(context.Background(), browser.SessionContext{
		BaseURL: "https://open.xfchat.iflytek.com",
		Cookies: []*http.Cookie{
			{Name: "sid", Value: "scope-cookie", Domain: "open.xfchat.iflytek.com", Path: "/developers/v1/scope"},
			{Name: "sid", Value: "root", Domain: "open.xfchat.iflytek.com", Path: "/"},
		},
	}, EnsureScopesRequest{
		AppID:  "cli_123",
		Scopes: []string{"docx:document:readonly"},
	})
	if err == nil {
		t.Fatal("expected scope update to fail")
	}
	if !strings.Contains(err.Error(), "unexpected status 400") {
		t.Fatalf("expected unexpected-status error, got %v", err)
	}
	if gotMethod != http.MethodPost {
		t.Fatalf("expected post request, got %q", gotMethod)
	}
	if gotPath != "/developers/v1/scope/all/cli_123" {
		t.Fatalf("expected auth endpoint path, got %q", gotPath)
	}
	if gotCookie != "sid=scope-cookie; sid=root" {
		t.Fatalf("expected request-url cookie selection, got %q", gotCookie)
	}
	if gotBody != "{}" {
		t.Fatalf("expected empty request body, got %q", gotBody)
	}
}

func TestClientPublishVersionReturnsErrorOnUnexpectedStatus(t *testing.T) {
	t.Helper()

	var gotMethod string
	var gotPath string
	var gotCookie string
	var gotReferer string
	var gotOrigin string
	var gotXSRF string
	var gotContentType string
	var gotBody string

	client := Client{
		BaseURL: "https://open.xfchat.iflytek.com",
		Do: func(req *http.Request) (*http.Response, error) {
			gotMethod = req.Method
			gotPath = req.URL.Path
			gotCookie = req.Header.Get("Cookie")
			gotReferer = req.Header.Get("Referer")
			gotOrigin = req.Header.Get("Origin")
			gotXSRF = req.Header.Get("X-XSRF-TOKEN")
			gotContentType = req.Header.Get("Content-Type")
			body, _ := io.ReadAll(req.Body)
			gotBody = string(body)

			return &http.Response{
				StatusCode: http.StatusInternalServerError,
				Body:       io.NopCloser(strings.NewReader(`{"error":"server error"}`)),
			}, nil
		},
	}

	err := client.PublishVersion(context.Background(), browser.SessionContext{
		BaseURL: "https://open.xfchat.iflytek.com",
		Cookies: []*http.Cookie{{Name: "sid", Value: "cookie-123", Domain: "open.xfchat.iflytek.com", Path: "/developers/v1/app_version"}},
		Headers: map[string]string{"X-XSRF-TOKEN": "csrf-456"},
	}, PublishVersionRequest{
		AppID: "cli_123",
	})
	if err == nil {
		t.Fatal("expected publish request to fail")
	}
	if !strings.Contains(err.Error(), "unexpected status 500") {
		t.Fatalf("expected unexpected-status error, got %v", err)
	}
	if gotMethod != http.MethodPost {
		t.Fatalf("expected post request, got %q", gotMethod)
	}
	if gotPath != "/developers/v1/app_version/list/cli_123" {
		t.Fatalf("expected version publish endpoint path, got %q", gotPath)
	}
	if gotCookie != "sid=cookie-123" {
		t.Fatalf("expected request-scoped cookie selection, got %q", gotCookie)
	}
	if gotReferer != "https://open.xfchat.iflytek.com/app/cli_123/version" {
		t.Fatalf("expected referer header for version page, got %q", gotReferer)
	}
	if gotOrigin != "https://open.xfchat.iflytek.com" {
		t.Fatalf("expected origin header, got %q", gotOrigin)
	}
	if gotXSRF != "csrf-456" {
		t.Fatalf("expected xsrf header to be forwarded, got %q", gotXSRF)
	}
	if gotContentType != "application/json" {
		t.Fatalf("expected json content type, got %q", gotContentType)
	}
	if gotBody != "{}" {
		t.Fatalf("expected empty request body, got %q", gotBody)
	}
}
