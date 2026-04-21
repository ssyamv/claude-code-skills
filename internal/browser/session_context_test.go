package browser

import (
	"net/http"
	"testing"
)

func TestSessionContextHeaderBuildIncludesCookiesAndCSRF(t *testing.T) {
	ctx := SessionContext{
		BaseURL:   "https://open.xfchat.iflytek.com",
		Cookies:   []*http.Cookie{{Name: "sid", Value: "cookie-123", Domain: "open.xfchat.iflytek.com", Path: "/"}},
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
	if got := header.Get("x-csrf-token"); got != "csrf-123" {
		t.Fatalf("expected lowercase csrf header, got %q", got)
	}
	if got := header.Get("Origin"); got != "https://open.xfchat.iflytek.com" {
		t.Fatalf("expected origin header, got %q", got)
	}
	if got := header.Get("Referer"); got != "https://open.xfchat.iflytek.com/app" {
		t.Fatalf("expected referer header, got %q", got)
	}
}

func TestSessionContextHeaderFiltersCookiesByRequestURL(t *testing.T) {
	ctx := SessionContext{
		BaseURL: "https://open.xfchat.iflytek.com",
		Cookies: []*http.Cookie{
			{Name: "sid", Value: "app", Domain: "open.xfchat.iflytek.com", Path: "/app"},
			{Name: "sid", Value: "secure", Domain: "open.xfchat.iflytek.com", Path: "/app", Secure: true},
			{Name: "sid", Value: "root", Domain: "open.xfchat.iflytek.com", Path: "/"},
			{Name: "drop", Value: "other", Domain: "other.example.com", Path: "/"},
		},
	}

	requestURL := "https://open.xfchat.iflytek.com/app/page"
	header := ctx.Header(requestURL)

	if got := header.Get("Cookie"); got != "sid=app; sid=secure; sid=root" {
		t.Fatalf("expected filtered cookie header, got %q", got)
	}
}

func TestSessionContextHeaderForRequestUsesRequestURLForCookieSelection(t *testing.T) {
	ctx := SessionContext{
		BaseURL: "https://open.xfchat.iflytek.com",
		Cookies: []*http.Cookie{
			{Name: "sid", Value: "api", Domain: "open.xfchat.iflytek.com", Path: "/api"},
			{Name: "sid", Value: "app", Domain: "open.xfchat.iflytek.com", Path: "/app"},
			{Name: "sid", Value: "root", Domain: "open.xfchat.iflytek.com", Path: "/"},
		},
	}

	header := ctx.HeaderForRequest("https://open.xfchat.iflytek.com/api/v1/query", "https://open.xfchat.iflytek.com/app")

	if got := header.Get("Cookie"); got != "sid=api; sid=root" {
		t.Fatalf("expected request-url cookie selection, got %q", got)
	}
	if got := header.Get("Referer"); got != "https://open.xfchat.iflytek.com/app" {
		t.Fatalf("expected explicit referer header, got %q", got)
	}
}

func TestSessionContextHeaderSelectsPathScopedCookiesAgainstRequestURL(t *testing.T) {
	ctx := SessionContext{
		BaseURL: "https://open.xfchat.iflytek.com",
		Cookies: []*http.Cookie{
			{Name: "sid", Value: "root", Domain: "open.xfchat.iflytek.com", Path: "/"},
			{Name: "sid", Value: "app", Domain: "open.xfchat.iflytek.com", Path: "/app"},
			{Name: "sid", Value: "admin", Domain: "open.xfchat.iflytek.com", Path: "/app/admin"},
		},
	}

	header := ctx.Header("https://open.xfchat.iflytek.com/app/admin/settings")

	if got := header.Get("Cookie"); got != "sid=admin; sid=app; sid=root" {
		t.Fatalf("expected path-scoped cookies ordered by specificity, got %q", got)
	}
}

func TestSessionContextHeaderCompatibilityWrapperUsesRefererAsRequestURL(t *testing.T) {
	ctx := SessionContext{
		BaseURL: "https://open.xfchat.iflytek.com",
		Cookies: []*http.Cookie{
			{Name: "sid", Value: "api", Domain: "open.xfchat.iflytek.com", Path: "/api"},
			{Name: "sid", Value: "app", Domain: "open.xfchat.iflytek.com", Path: "/app"},
			{Name: "sid", Value: "root", Domain: "open.xfchat.iflytek.com", Path: "/"},
		},
	}

	header := ctx.Header("https://open.xfchat.iflytek.com/api/v1/query")

	if got := header.Get("Cookie"); got != "sid=api; sid=root" {
		t.Fatalf("expected wrapper to keep request-url matching, got %q", got)
	}
	if got := header.Get("Referer"); got != "https://open.xfchat.iflytek.com/api/v1/query" {
		t.Fatalf("expected wrapper to set referer from the argument, got %q", got)
	}
}

func TestSessionContextHeaderDropsSecureCookiesOnHTTP(t *testing.T) {
	ctx := SessionContext{
		BaseURL: "http://open.xfchat.iflytek.com",
		Cookies: []*http.Cookie{
			{Name: "sid", Value: "app", Domain: "open.xfchat.iflytek.com", Path: "/app"},
			{Name: "sid", Value: "secure", Domain: "open.xfchat.iflytek.com", Path: "/app", Secure: true},
			{Name: "sid", Value: "root", Domain: "open.xfchat.iflytek.com", Path: "/"},
		},
	}

	header := ctx.Header("http://open.xfchat.iflytek.com/app/page")

	if got := header.Get("Cookie"); got != "sid=app; sid=root" {
		t.Fatalf("expected secure cookie to be dropped, got %q", got)
	}
}

func TestSessionContextHeaderDistinguishesHostOnlyAndDomainCookies(t *testing.T) {
	ctx := SessionContext{
		BaseURL: "https://example.com",
		Cookies: []*http.Cookie{
			{Name: "hostonly", Value: "base", Path: "/"},
			{Name: "domain", Value: "wide", Domain: "example.com", Path: "/"},
		},
	}

	baseHeader := ctx.Header("https://example.com/app")
	if got := baseHeader.Get("Cookie"); got != "hostonly=base; domain=wide" {
		t.Fatalf("expected both cookies on exact host, got %q", got)
	}

	subHeader := ctx.Header("https://sub.example.com/app")
	if got := subHeader.Get("Cookie"); got != "domain=wide" {
		t.Fatalf("expected only domain cookie on subdomain, got %q", got)
	}
}

func TestSessionContextHeaderPreservesExplicitXSRFHeader(t *testing.T) {
	ctx := SessionContext{
		BaseURL:   "https://open.xfchat.iflytek.com",
		Cookies:   []*http.Cookie{{Name: "sid", Value: "cookie-123", Domain: "open.xfchat.iflytek.com", Path: "/"}},
		CSRFToken: "csrf-123",
		Headers: map[string]string{
			"X-XSRF-TOKEN": "explicit-token",
		},
	}

	header := ctx.Header("https://open.xfchat.iflytek.com/app")

	if got := header.Get("X-XSRF-TOKEN"); got != "explicit-token" {
		t.Fatalf("expected explicit xsrf header to win, got %q", got)
	}
}
