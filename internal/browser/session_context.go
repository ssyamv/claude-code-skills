package browser

import (
	"net/http"
	"net/url"
	"sort"
	"strings"
)

type SessionContext struct {
	BaseURL   string
	Cookies   []*http.Cookie
	CSRFToken string
	Headers   map[string]string
}

func (c SessionContext) Header(referer string) http.Header {
	return c.HeaderForRequest(referer, referer)
}

func (c SessionContext) HeaderForRequest(requestURL, referer string) http.Header {
	h := make(http.Header)

	for key, value := range c.Headers {
		if strings.TrimSpace(value) != "" {
			h.Set(key, value)
		}
	}

	if c.CSRFToken != "" && h.Get("X-XSRF-TOKEN") == "" {
		h.Set("X-XSRF-TOKEN", c.CSRFToken)
	}
	if c.CSRFToken != "" && h.Get("x-csrf-token") == "" {
		h.Set("x-csrf-token", c.CSRFToken)
	}

	if origin := originFromURL(c.BaseURL); origin != "" {
		h.Set("Origin", origin)
	}

	if referer != "" {
		h.Set("Referer", referer)
	}

	if cookieHeader := cookieHeaderForRequest(c.BaseURL, c.Cookies, requestURL); cookieHeader != "" {
		h.Set("Cookie", cookieHeader)
	}

	return h
}

func originFromURL(raw string) string {
	if raw == "" {
		return ""
	}

	parsed, err := url.Parse(raw)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return ""
	}

	return parsed.Scheme + "://" + parsed.Host
}

func cookieHeaderForRequest(baseURL string, cookies []*http.Cookie, rawURL string) string {
	target, err := url.Parse(rawURL)
	if err != nil || target.Scheme == "" || target.Host == "" {
		return ""
	}
	baseHost := ""
	if base := originFromURL(baseURL); base != "" {
		if parsedBase, err := url.Parse(base); err == nil {
			baseHost = parsedBase.Hostname()
		}
	}

	type matchedCookie struct {
		cookie      *http.Cookie
		pathLen     int
		specificity int
	}

	matches := make([]matchedCookie, 0, len(cookies))
	for _, cookie := range cookies {
		if !cookieMatchesURL(cookie, target, baseHost) {
			continue
		}
		path := cookie.Path
		if path == "" {
			path = "/"
		}
		matches = append(matches, matchedCookie{
			cookie:      cookie,
			pathLen:     len(path),
			specificity: cookieSpecificity(cookie, target.Hostname()),
		})
	}

	sort.SliceStable(matches, func(i, j int) bool {
		if matches[i].pathLen != matches[j].pathLen {
			return matches[i].pathLen > matches[j].pathLen
		}
		if matches[i].specificity != matches[j].specificity {
			return matches[i].specificity > matches[j].specificity
		}
		return false
	})

	parts := make([]string, 0, len(matches))
	for _, match := range matches {
		parts = append(parts, match.cookie.Name+"="+match.cookie.Value)
	}
	return strings.Join(parts, "; ")
}

func cookieMatchesURL(cookie *http.Cookie, target *url.URL, baseHost string) bool {
	if cookie == nil || cookie.Name == "" {
		return false
	}

	if cookie.Secure && !strings.EqualFold(target.Scheme, "https") {
		return false
	}

	if !cookieDomainMatches(cookie.Domain, target.Hostname(), baseHost) {
		return false
	}

	if !cookiePathMatches(target.Path, cookie.Path) {
		return false
	}

	return true
}

func cookieDomainMatches(cookieDomain, host, baseHost string) bool {
	host = strings.ToLower(host)
	cookieDomain = strings.ToLower(strings.TrimPrefix(cookieDomain, "."))
	if cookieDomain == "" {
		return baseHost != "" && host == strings.ToLower(baseHost)
	}
	if host == cookieDomain {
		return true
	}
	return strings.HasSuffix(host, "."+cookieDomain)
}

func cookieSpecificity(cookie *http.Cookie, host string) int {
	domain := strings.ToLower(strings.TrimPrefix(cookie.Domain, "."))
	if domain == "" || strings.EqualFold(domain, host) {
		return 2
	}
	return 1
}

func cookiePathMatches(requestPath, cookiePath string) bool {
	if cookiePath == "" {
		cookiePath = "/"
	}
	if requestPath == "" {
		requestPath = "/"
	}
	if requestPath == cookiePath {
		return true
	}
	if !strings.HasPrefix(requestPath, cookiePath) {
		return false
	}
	if strings.HasSuffix(cookiePath, "/") {
		return true
	}
	return len(requestPath) > len(cookiePath) && requestPath[len(cookiePath)] == '/'
}
