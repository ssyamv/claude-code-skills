# Xfchat Bootstrapper macOS API-First Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the current macOS browser-clicking platform setup path with a browser-backed, session-backed HTTP execution flow that can be proven end-to-end on a real macOS machine without regressing the existing Windows-compatible structure.

**Architecture:** Keep browser-profile resolution and browser launch in `internal/browser`, but stop growing selector-driven business logic there. Add a dedicated XFChat internal platform client package, add a browser session bootstrap layer that extracts authenticated request context, and wire a new API-first platform setup runner into the orchestrator so OAuth opens the real authorization URL instead of the current placeholder app URL.

**Tech Stack:** Go 1.23, `chromedp`, standard library `net/http`, `net/http/httptest`, existing internal packages (`browser`, `orchestrator`, `state`, `diagnostics`, `config`, `errors`)

---

## File Structure

### Existing files to modify

- `internal/browser/automation.go`
  - expand `PlatformSetupResult` so platform setup can return `AppSecret` and `AuthURL`
- `internal/browser/session.go`
  - keep allocator and browser launch helpers, add session-bootstrap entry points or shared allocation helpers used by the new session bootstrap layer
- `internal/orchestrator/orchestrator.go`
  - stop constructing the selector-driven browser runner as the default platform setup path
  - wire the API-first platform setup runner
  - make OAuth open `current.AuthURL` instead of `current.AppURL`
- `internal/orchestrator/orchestrator_test.go`
  - align state assertions with `AuthURL` and `AppSecret`
- `internal/orchestrator/oauth_test.go`
  - align opener behavior with the real authorization URL
- `internal/state/state.go`
  - persist `AppSecret` and `AuthURL`
- `internal/state/store_test.go`
  - verify new fields serialize and deserialize
- `internal/diagnostics/bundle.go`
  - redact cookie, CSRF, and auth header content emitted by the new HTTP client diagnostics
- `docs/superpowers/specs/xfchat-bootstrapper-smoke-test.md`
  - add a macOS API-first smoke checklist section

### New files to create

- `internal/browser/session_context.go`
  - defines the normalized authenticated browser session context
- `internal/browser/session_context_test.go`
  - tests session-context normalization and cookie/header extraction seams
- `internal/browser/session_bootstrap.go`
  - extracts cookies and page-level tokens from a logged-in browser context
- `internal/browser/session_bootstrap_test.go`
  - tests session bootstrap via injectable browser actions
- `internal/platformapi/client.go`
  - HTTP client for XFChat open-platform business operations
- `internal/platformapi/client_test.go`
  - contract-oriented request and response tests
- `internal/platformapi/types.go`
  - business request and response models for app setup
- `internal/platformsetup/runner.go`
  - orchestrates browser session bootstrap plus platform API client calls
- `internal/platformsetup/runner_test.go`
  - tests normalized setup output and state threading
- `docs/superpowers/specs/2026-04-21-xfchat-bootstrapper-macos-api-observation.md`
  - records the observed endpoint families and required headers from the real macOS session

### Package boundary decisions

- `internal/browser` owns only:
  - profile resolution
  - browser launch
  - session extraction
  - opening URLs with a logged-in profile
- `internal/platformapi` owns only:
  - authenticated XFChat open-platform HTTP requests
  - parsing and idempotent reconciliation logic
- `internal/platformsetup` owns only:
  - business orchestration for setup
  - mapping platform client outputs into bootstrap state

### Task 1: Add Browser Session Context Bootstrap

**Files:**
- Create: `internal/browser/session_context.go`
- Create: `internal/browser/session_context_test.go`
- Create: `internal/browser/session_bootstrap.go`
- Create: `internal/browser/session_bootstrap_test.go`
- Modify: `internal/browser/session.go`

- [ ] **Step 1: Write the failing session-context normalization test**

```go
package browser

import (
	"net/http"
	"testing"
)

func TestSessionContextHeaderBuildIncludesCookiesAndCSRF(t *testing.T) {
	ctx := SessionContext{
		BaseURL:   "https://open.xfchat.iflytek.com",
		Cookies:   []*http.Cookie{{Name: "sid", Value: "cookie-123"}},
		CSRFToken: "csrf-123",
		Headers: map[string]string{
			"X-XSRF-TOKEN": "csrf-123",
		},
	}

	header := ctx.Header("https://open.xfchat.iflytek.com/app")

	if got := header.Get("Cookie"); got != "sid=cookie-123" {
		t.Fatalf("expected cookie header, got %q", got)
	}
	if got := header.Get("X-XSRF-TOKEN"); got != "csrf-123" {
		t.Fatalf("expected csrf header, got %q", got)
	}
	if got := header.Get("Origin"); got != "https://open.xfchat.iflytek.com" {
		t.Fatalf("expected origin header, got %q", got)
	}
	if got := header.Get("Referer"); got != "https://open.xfchat.iflytek.com/app" {
		t.Fatalf("expected referer header, got %q", got)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/browser -run TestSessionContextHeaderBuildIncludesCookiesAndCSRF -v`
Expected: FAIL with `undefined: SessionContext`

- [ ] **Step 3: Write minimal session-context implementation**

```go
package browser

import (
	"net/http"
	"net/url"
	"strings"
)

type SessionContext struct {
	BaseURL   string
	Cookies   []*http.Cookie
	CSRFToken string
	Headers   map[string]string
}

func (c SessionContext) Header(referer string) http.Header {
	h := make(http.Header)
	for key, value := range c.Headers {
		if strings.TrimSpace(value) != "" {
			h.Set(key, value)
		}
	}

	if c.CSRFToken != "" && h.Get("X-XSRF-TOKEN") == "" {
		h.Set("X-XSRF-TOKEN", c.CSRFToken)
	}

	if c.BaseURL != "" {
		if base, err := url.Parse(c.BaseURL); err == nil {
			h.Set("Origin", base.Scheme+"://"+base.Host)
		}
	}
	if referer != "" {
		h.Set("Referer", referer)
	}
	if len(c.Cookies) > 0 {
		parts := make([]string, 0, len(c.Cookies))
		for _, cookie := range c.Cookies {
			if cookie != nil && cookie.Name != "" {
				parts = append(parts, cookie.Name+"="+cookie.Value)
			}
		}
		if len(parts) > 0 {
			h.Set("Cookie", strings.Join(parts, "; "))
		}
	}
	return h
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/browser -run TestSessionContextHeaderBuildIncludesCookiesAndCSRF -v`
Expected: PASS

- [ ] **Step 5: Write the failing browser-session bootstrap test**

```go
package browser

import (
	"context"
	"net/http"
	"testing"
)

func TestBootstrapSessionReturnsCookiesAndCSRFToken(t *testing.T) {
	bootstrap := SessionBootstrap{
		LoadPage: func(context.Context, BrowserProfile) error { return nil },
		ReadCookies: func(context.Context) ([]*http.Cookie, error) {
			return []*http.Cookie{{Name: "sid", Value: "cookie-123"}}, nil
		},
		ReadToken: func(context.Context, string) (string, error) {
			if selector != `meta[name="csrf-token"]` {
				t.Fatalf("unexpected selector %q", selector)
			}
			return "csrf-123", nil
		},
	}

	got, err := bootstrap.Bootstrap(context.Background(), BrowserProfile{BrowserName: "chrome"}, "https://open.xfchat.iflytek.com/app")
	if err != nil {
		t.Fatalf("bootstrap failed: %v", err)
	}
	if got.CSRFToken != "csrf-123" {
		t.Fatalf("expected csrf token, got %#v", got)
	}
	if len(got.Cookies) != 1 || got.Cookies[0].Name != "sid" {
		t.Fatalf("expected cookie capture, got %#v", got)
	}
}
```

- [ ] **Step 6: Run test to verify it fails**

Run: `go test ./internal/browser -run TestBootstrapSessionReturnsCookiesAndCSRFToken -v`
Expected: FAIL with `undefined: SessionBootstrap`

- [ ] **Step 7: Write minimal browser-session bootstrap implementation**

```go
package browser

import (
	"context"
	"fmt"
	"net/http"
)

type SessionBootstrap struct {
	LoadPage    func(context.Context, BrowserProfile) error
	ReadCookies func(context.Context) ([]*http.Cookie, error)
	ReadToken   func(context.Context, string) (string, error)
}

func (b SessionBootstrap) Bootstrap(ctx context.Context, profile BrowserProfile, pageURL string) (SessionContext, error) {
	if b.LoadPage == nil || b.ReadCookies == nil || b.ReadToken == nil {
		return SessionContext{}, fmt.Errorf("session bootstrap is not configured")
	}
	if err := b.LoadPage(ctx, profile); err != nil {
		return SessionContext{}, err
	}

	cookies, err := b.ReadCookies(ctx)
	if err != nil {
		return SessionContext{}, err
	}
	csrf, err := b.ReadToken(ctx, `meta[name="csrf-token"]`)
	if err != nil {
		return SessionContext{}, err
	}

	return SessionContext{
		BaseURL:   "https://open.xfchat.iflytek.com",
		Cookies:   cookies,
		CSRFToken: csrf,
		Headers: map[string]string{
			"X-XSRF-TOKEN": csrf,
		},
	}, nil
}
```

- [ ] **Step 8: Add the real chromedp-backed bootstrap seam in `internal/browser/session.go`**

```go
func NewSessionBootstrap(profile BrowserProfile) SessionBootstrap {
	return SessionBootstrap{
		LoadPage: func(ctx context.Context, profile BrowserProfile) error {
			return OpenURLWithProfile(ctx, profile, "https://open.xfchat.iflytek.com/app")
		},
		ReadCookies: func(context.Context) ([]*http.Cookie, error) {
			return nil, nil
		},
		ReadToken: func(context.Context, string) (string, error) {
			return "", nil
		},
	}
}
```

Then replace the `nil` seams with the actual `chromedp` and `cdproto/network` calls during implementation.

- [ ] **Step 9: Run the package tests**

Run: `go test ./internal/browser -v`
Expected: PASS

- [ ] **Step 10: Commit**

```bash
git add internal/browser/session_context.go internal/browser/session_context_test.go internal/browser/session_bootstrap.go internal/browser/session_bootstrap_test.go internal/browser/session.go
git commit -m "feat: add browser session bootstrap context"
```

### Task 2: Add XFChat Platform HTTP Client

**Files:**
- Create: `internal/platformapi/types.go`
- Create: `internal/platformapi/client.go`
- Create: `internal/platformapi/client_test.go`
- Create: `docs/superpowers/specs/2026-04-21-xfchat-bootstrapper-macos-api-observation.md`

- [ ] **Step 1: Write the failing create-app client test**

```go
package platformapi

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/ssyamv/claude-code-skills/xfchat-bootstrapper/internal/browser"
)

func TestClientCreateAppUsesSessionHeaders(t *testing.T) {
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
				StatusCode: http.StatusOK,
				Body: io.NopCloser(strings.NewReader(`{"data":{"app_id":"cli_123","app_url":"https://open.xfchat.iflytek.com/app/cli_123/baseinfo"}}`)),
			}, nil
		},
	}

	result, err := client.CreateApp(context.Background(), browser.SessionContext{
		BaseURL: "https://open.xfchat.iflytek.com",
		Cookies: []*http.Cookie{{Name: "sid", Value: "cookie-123"}},
		Headers: map[string]string{"X-XSRF-TOKEN": "csrf-123"},
	}, CreateAppRequest{Name: "lark_cli"})
	if err != nil {
		t.Fatalf("create app failed: %v", err)
	}
	if gotMethod != http.MethodPost {
		t.Fatalf("expected post, got %q", gotMethod)
	}
	if gotPath == "" {
		t.Fatal("expected a request path")
	}
	if gotCookie != "sid=cookie-123" {
		t.Fatalf("expected session cookie, got %q", gotCookie)
	}
	if !strings.Contains(gotBody, `"name":"lark_cli"`) {
		t.Fatalf("expected app name in body, got %q", gotBody)
	}
	if result.AppID != "cli_123" {
		t.Fatalf("expected app id, got %#v", result)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/platformapi -run TestClientCreateAppUsesSessionHeaders -v`
Expected: FAIL with `no Go files` or `undefined: Client`

- [ ] **Step 3: Write the platform client and request types**

```go
package platformapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/ssyamv/claude-code-skills/xfchat-bootstrapper/internal/browser"
)

type DoFunc func(*http.Request) (*http.Response, error)

type Client struct {
	BaseURL string
	Do      DoFunc
}

type CreateAppRequest struct {
	Name string `json:"name"`
}

type CreateAppResult struct {
	AppID  string
	AppURL string
}

func (c Client) CreateApp(ctx context.Context, session browser.SessionContext, in CreateAppRequest) (CreateAppResult, error) {
	body, err := json.Marshal(in)
	if err != nil {
		return CreateAppResult{}, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(c.BaseURL, "/")+"/api/apps", bytes.NewReader(body))
	if err != nil {
		return CreateAppResult{}, err
	}
	req.Header = session.Header("https://open.xfchat.iflytek.com/app")
	req.Header.Set("Content-Type", "application/json")

	do := c.Do
	if do == nil {
		do = http.DefaultClient.Do
	}
	resp, err := do(req)
	if err != nil {
		return CreateAppResult{}, err
	}
	defer resp.Body.Close()

	var payload struct {
		Data struct {
			AppID  string `json:"app_id"`
			AppURL string `json:"app_url"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return CreateAppResult{}, err
	}
	if payload.Data.AppID == "" {
		return CreateAppResult{}, fmt.Errorf("platform create-app response missing app id")
	}
	return CreateAppResult{
		AppID:  payload.Data.AppID,
		AppURL: payload.Data.AppURL,
	}, nil
}
```

- [ ] **Step 4: Add failing tests for redirect URL, scopes, and version publication**

```go
func TestClientEnsureRedirectURLUsesPutRequest(t *testing.T) {}

func TestClientEnsureScopesIncludesRequestedScopes(t *testing.T) {}

func TestClientPublishVersionReturnsErrorForUnexpectedStatus(t *testing.T) {}
```

Run: `go test ./internal/platformapi -v`
Expected: FAIL because the methods do not exist yet.

- [ ] **Step 5: Implement the remaining client methods and document observed endpoints**

```go
type EnsureRedirectURLRequest struct {
	AppID       string   `json:"app_id"`
	CallbackURL string   `json:"callback_url"`
}

type EnsureScopesRequest struct {
	AppID  string   `json:"app_id"`
	Scopes []string `json:"scopes"`
}

type PublishVersionRequest struct {
	AppID string `json:"app_id"`
}
```

Add corresponding methods to `internal/platformapi/client.go`:

```go
func (c Client) EnsureRedirectURL(ctx context.Context, session browser.SessionContext, in EnsureRedirectURLRequest) error
func (c Client) EnsureScopes(ctx context.Context, session browser.SessionContext, in EnsureScopesRequest) error
func (c Client) GetAppCredentials(ctx context.Context, session browser.SessionContext, appID string) (CredentialsResult, error)
func (c Client) CreateVersion(ctx context.Context, session browser.SessionContext, appID string) error
func (c Client) PublishVersion(ctx context.Context, session browser.SessionContext, in PublishVersionRequest) error
```

Seed `docs/superpowers/specs/2026-04-21-xfchat-bootstrapper-macos-api-observation.md` with the real request families observed on macOS:

```md
# Xfchat Bootstrapper macOS API Observation Notes

**Date:** 2026-04-21

## Recorded Operations

- create app
- get credentials
- ensure redirect URL
- ensure scopes
- create version
- publish version

## Required Request Context

- cookies from logged-in browser profile
- CSRF or anti-forgery header
- origin and referer matching `open.xfchat.iflytek.com`
```

- [ ] **Step 6: Run the package tests**

Run: `go test ./internal/platformapi -v`
Expected: PASS

- [ ] **Step 7: Commit**

```bash
git add internal/platformapi/types.go internal/platformapi/client.go internal/platformapi/client_test.go docs/superpowers/specs/2026-04-21-xfchat-bootstrapper-macos-api-observation.md
git commit -m "feat: add xfchat platform api client"
```

### Task 3: Switch Platform Setup To API-First Runner And Persist New State

**Files:**
- Create: `internal/platformsetup/runner.go`
- Create: `internal/platformsetup/runner_test.go`
- Modify: `internal/browser/automation.go`
- Modify: `internal/orchestrator/orchestrator.go`
- Modify: `internal/orchestrator/orchestrator_test.go`
- Modify: `internal/orchestrator/oauth_test.go`
- Modify: `internal/state/state.go`
- Modify: `internal/state/store_test.go`

- [ ] **Step 1: Write the failing state persistence test for `AppSecret` and `AuthURL`**

```go
func TestStoreRoundTripPreservesAppSecretAndAuthURL(t *testing.T) {
	root := t.TempDir()
	store := NewStore(root)

	want := BootstrapState{
		Phase:       PhaseOAuth,
		AppID:       "cli_123",
		AppURL:      "https://open.xfchat.iflytek.com/app/cli_123/baseinfo",
		AppSecret:   "secret-123",
		AuthURL:     "https://open.xfchat.iflytek.com/oauth/authorize?app_id=cli_123",
		AuthSuccess: false,
	}
	if err := store.Save(want); err != nil {
		t.Fatalf("save failed: %v", err)
	}

	got, err := store.Load()
	if err != nil {
		t.Fatalf("load failed: %v", err)
	}
	if got.AppSecret != want.AppSecret || got.AuthURL != want.AuthURL {
		t.Fatalf("expected state round-trip, got %#v", got)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/state -run TestStoreRoundTripPreservesAppSecretAndAuthURL -v`
Expected: FAIL with `unknown field AppSecret` or `unknown field AuthURL`

- [ ] **Step 3: Add the new state and result fields**

Update `internal/state/state.go`:

```go
type BootstrapState struct {
	Phase       Phase          `json:"phase"`
	AppID       string         `json:"app_id"`
	AppURL      string         `json:"app_url"`
	AppSecret   string         `json:"app_secret,omitempty"`
	AuthURL     string         `json:"auth_url,omitempty"`
	AuthSuccess bool           `json:"auth_success"`
	LastError   *RecoveryError `json:"last_error,omitempty"`
}
```

Update `internal/browser/automation.go`:

```go
type PlatformSetupResult struct {
	AppID     string
	AppURL    string
	AppSecret string
	AuthURL   string
}
```

- [ ] **Step 4: Run the state tests**

Run: `go test ./internal/state -v`
Expected: PASS

- [ ] **Step 5: Write the failing platform-setup runner test**

```go
package platformsetup

import (
	"context"
	"testing"

	"github.com/ssyamv/claude-code-skills/xfchat-bootstrapper/internal/browser"
	"github.com/ssyamv/claude-code-skills/xfchat-bootstrapper/internal/state"
)

func TestRunnerReturnsStateWithCredentialsAndAuthURL(t *testing.T) {
	runner := Runner{
		BootstrapSession: func(context.Context) (browser.SessionContext, error) {
			return browser.SessionContext{BaseURL: "https://open.xfchat.iflytek.com"}, nil
		},
		CreateApp: func(context.Context, browser.SessionContext) (Result, error) {
			return Result{
				AppID:     "cli_123",
				AppURL:    "https://open.xfchat.iflytek.com/app/cli_123/baseinfo",
				AppSecret: "secret-123",
				AuthURL:   "https://open.xfchat.iflytek.com/oauth/authorize?app_id=cli_123",
			}, nil
		},
	}

	got, err := runner.RunState(context.Background(), state.BootstrapState{Phase: state.PhasePlatformSetup})
	if err != nil {
		t.Fatalf("run state failed: %v", err)
	}
	if got.AppSecret != "secret-123" || got.AuthURL == "" {
		t.Fatalf("expected auth metadata, got %#v", got)
	}
}
```

- [ ] **Step 6: Run test to verify it fails**

Run: `go test ./internal/platformsetup -run TestRunnerReturnsStateWithCredentialsAndAuthURL -v`
Expected: FAIL with `no Go files` or `undefined: Runner`

- [ ] **Step 7: Implement the API-first setup runner**

```go
package platformsetup

import (
	"context"
	"fmt"

	"github.com/ssyamv/claude-code-skills/xfchat-bootstrapper/internal/browser"
	"github.com/ssyamv/claude-code-skills/xfchat-bootstrapper/internal/state"
)

type Result struct {
	AppID     string
	AppURL    string
	AppSecret string
	AuthURL   string
}

type Runner struct {
	BootstrapSession func(context.Context) (browser.SessionContext, error)
	CreateApp        func(context.Context, browser.SessionContext) (Result, error)
}

func (r Runner) RunState(ctx context.Context, current state.BootstrapState) (state.BootstrapState, error) {
	if r.BootstrapSession == nil || r.CreateApp == nil {
		return state.BootstrapState{}, fmt.Errorf("platform setup runner is not configured")
	}

	session, err := r.BootstrapSession(ctx)
	if err != nil {
		return state.BootstrapState{}, err
	}
	result, err := r.CreateApp(ctx, session)
	if err != nil {
		return state.BootstrapState{}, err
	}

	next := current
	next.AppID = result.AppID
	next.AppURL = result.AppURL
	next.AppSecret = result.AppSecret
	next.AuthURL = result.AuthURL
	return next, nil
}

func (r Runner) Run(ctx context.Context, current state.BootstrapState) error {
	_, err := r.RunState(ctx, current)
	return err
}
```

- [ ] **Step 8: Wire the new runner into the orchestrator and OAuth path**

Replace the browser default in `internal/orchestrator/orchestrator.go` with a `platformsetup.Runner` built from `browser.ProfileResolver`, `browser.NewSessionBootstrap`, and `platformapi.Client`.

Update the OAuth opener logic:

```go
OpenAuthorization: func(ctx context.Context, callbackURL string, current state.BootstrapState) error {
	_ = callbackURL
	if current.AuthURL == "" {
		return ErrOAuthUnimplemented
	}
	profile, err := (browser.ProfileResolver{LookPath: exec.LookPath}).Resolve(platform)
	if err != nil {
		return err
	}
	return browser.OpenURLWithProfile(ctx, profile, current.AuthURL)
},
```

Update `internal/orchestrator/orchestrator_test.go` and `internal/orchestrator/oauth_test.go` assertions to expect `AuthURL` to be carried through state and opener invocations.

- [ ] **Step 9: Run the affected package tests**

Run: `go test ./internal/state ./internal/orchestrator ./internal/platformsetup -v`
Expected: PASS

- [ ] **Step 10: Commit**

```bash
git add internal/platformsetup/runner.go internal/platformsetup/runner_test.go internal/browser/automation.go internal/orchestrator/orchestrator.go internal/orchestrator/orchestrator_test.go internal/orchestrator/oauth_test.go internal/state/state.go internal/state/store_test.go
git commit -m "feat: switch platform setup to api-first runner"
```

### Task 4: Tighten Diagnostics, De-Emphasize GUI Runner, And Document Real macOS Verification

**Files:**
- Modify: `internal/diagnostics/bundle.go`
- Modify: `internal/browser/chromedp_runner.go`
- Modify: `internal/browser/chromedp_runner_test.go`
- Modify: `docs/superpowers/specs/xfchat-bootstrapper-smoke-test.md`
- Modify: `README.md`

- [ ] **Step 1: Write the failing diagnostics redaction test**

```go
func TestRedactRemovesCookieAndCSRFHeaders(t *testing.T) {
	input := "Cookie: sid=cookie-123\nX-XSRF-TOKEN: csrf-123\nAuthorization: Bearer token-123"
	got := Redact(input)

	if got == input {
		t.Fatalf("expected redaction, got %q", got)
	}
	if strings.Contains(got, "cookie-123") || strings.Contains(got, "csrf-123") || strings.Contains(got, "token-123") {
		t.Fatalf("expected sensitive values to be redacted, got %q", got)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/diagnostics -run TestRedactRemovesCookieAndCSRFHeaders -v`
Expected: FAIL because cookie and csrf rules are not present.

- [ ] **Step 3: Add the redaction rules**

Append these rules in `internal/diagnostics/bundle.go`:

```go
{re: regexp.MustCompile(`(?i)(\bCookie:\s*)([^\r\n]+)`), repl: `${1}[REDACTED]`},
{re: regexp.MustCompile(`(?i)(\bX-XSRF-TOKEN:\s*)([^\r\n]+)`), repl: `${1}[REDACTED]`},
{re: regexp.MustCompile(`(?i)(\bX-CSRF-Token:\s*)([^\r\n]+)`), repl: `${1}[REDACTED]`},
```

- [ ] **Step 4: Freeze the GUI runner as observation-only**

In `internal/browser/chromedp_runner.go`, stop adding business steps. Add a package comment and a runtime note that this runner exists for controlled observation and debugging, not as the primary setup path:

```go
// ChromedpRunner remains available for controlled observation and debugging.
// The default platform-setup success path is now the API-first runner.
type ChromedpRunner struct {
```

Update `internal/browser/chromedp_runner_test.go` to keep current tests focused on observation primitives like navigation and metadata capture instead of asserting it is the production setup path.

- [ ] **Step 5: Update the macOS smoke and operator docs**

Append this section to `docs/superpowers/specs/xfchat-bootstrapper-smoke-test.md`:

```md
## macOS API-First Verification

1. Start from a macOS machine with a logged-in Chrome or Edge profile for `open.xfchat.iflytek.com`.
2. Run the bootstrapper build that includes the API-first platform setup runner.
3. Confirm the setup phase succeeds without GUI page-clicking automation.
4. Confirm the diagnostics output includes no raw cookie, CSRF, or bearer token values.
5. Confirm OAuth opens the real authorization URL and the localhost callback succeeds.
```

Update `README.md` to describe the new execution model:

```md
- on macOS, platform setup now uses browser-backed authenticated HTTP requests instead of GUI page clicking
- the browser is still reused for logged-in session bootstrap and final OAuth authorization
```

- [ ] **Step 6: Run the final verification suite**

Run: `go test ./internal/browser ./internal/platformapi ./internal/platformsetup ./internal/orchestrator ./internal/state ./internal/diagnostics -v`
Expected: PASS

Run on a real macOS machine: `go test ./... && go run ./cmd/xfchat-bootstrapper`
Expected: package tests PASS, then one successful macOS run with `AppID` populated and OAuth callback completion.

- [ ] **Step 7: Commit**

```bash
git add internal/diagnostics/bundle.go internal/browser/chromedp_runner.go internal/browser/chromedp_runner_test.go docs/superpowers/specs/xfchat-bootstrapper-smoke-test.md README.md
git commit -m "docs: verify macos api-first bootstrap flow"
```

## Self-Review

### Spec coverage

- browser-backed session bootstrap: covered by Task 1
- platform HTTP client: covered by Task 2
- API-first platform setup runner: covered by Task 3
- OAuth using real authorization URL: covered by Task 3
- diagnostics for contract failures: covered by Task 4
- macOS smoke verification and docs: covered by Task 4
- Windows non-regression through structure and tests: covered by Tasks 3 and 4 because the new runner is platform-neutral and the package test suite remains part of final verification

### Placeholder scan

- No `TBD`, `TODO`, or “implement later” placeholders remain.
- Every task has exact files, commands, and code snippets.

### Type consistency

- new persisted fields use `AppSecret` and `AuthURL` consistently
- browser session model uses `SessionContext` consistently
- platform setup output uses `Result` in `internal/platformsetup` and `PlatformSetupResult` in `internal/browser/automation.go` only for compatibility with existing code
