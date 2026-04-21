package browser

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/chromedp"
)

type SessionBootstrap struct {
	NewSession  func(context.Context, BrowserProfile) (context.Context, func(), error)
	LoadPage    func(context.Context, string) error
	ReadCookies func(context.Context) ([]*http.Cookie, error)
	ReadToken   func(context.Context, string) (string, error)
}

func (b SessionBootstrap) Bootstrap(ctx context.Context, profile BrowserProfile, pageURL string) (SessionContext, error) {
	if b.NewSession == nil || b.LoadPage == nil || b.ReadCookies == nil || b.ReadToken == nil {
		return SessionContext{}, fmt.Errorf("session bootstrap is not configured")
	}

	sessionCtx, cleanup, err := b.NewSession(ctx, profile)
	if err != nil {
		return SessionContext{}, fmt.Errorf("create browser session: %w", err)
	}
	if cleanup != nil {
		defer cleanup()
	}

	if err := b.LoadPage(sessionCtx, pageURL); err != nil {
		return SessionContext{}, fmt.Errorf("load session page: %w", err)
	}

	cookies, err := b.ReadCookies(sessionCtx)
	if err != nil {
		return SessionContext{}, fmt.Errorf("read session cookies: %w", err)
	}

	token, err := b.ReadToken(sessionCtx, `meta[name="csrf-token"]`)
	if err != nil {
		return SessionContext{}, fmt.Errorf("read csrf token: %w", err)
	}

	return SessionContext{
		BaseURL:   originFromURL(pageURL),
		Cookies:   cookies,
		CSRFToken: token,
		Headers: map[string]string{
			"X-XSRF-TOKEN": token,
		},
	}, nil
}

func newProfileSession(ctx context.Context, profile BrowserProfile) (context.Context, func(), error) {
	allocCtx, cancelAlloc := chromedp.NewExecAllocator(ctx, SessionOptions(profile)...)
	taskCtx, cancelTask := chromedp.NewContext(allocCtx)

	return taskCtx, func() {
		cancelTask()
		cancelAlloc()
	}, nil
}

func readCookiesWithSession(ctx context.Context) ([]*http.Cookie, error) {
	var rawCookies []*network.Cookie
	if err := chromedp.Run(ctx, chromedp.ActionFunc(func(ctx context.Context) error {
		var err error
		rawCookies, err = network.GetCookies().Do(ctx)
		return err
	})); err != nil {
		return nil, err
	}

	cookies := make([]*http.Cookie, 0, len(rawCookies))
	for _, raw := range rawCookies {
		if raw == nil || raw.Name == "" {
			continue
		}
		cookie := &http.Cookie{
			Name:     raw.Name,
			Value:    raw.Value,
			Domain:   raw.Domain,
			Path:     raw.Path,
			Secure:   raw.Secure,
			HttpOnly: raw.HTTPOnly,
		}
		if raw.Expires > 0 {
			cookie.Expires = time.Unix(int64(raw.Expires), 0).UTC()
		}
		switch raw.SameSite {
		case "Strict":
			cookie.SameSite = http.SameSiteStrictMode
		case "Lax":
			cookie.SameSite = http.SameSiteLaxMode
		case "None":
			cookie.SameSite = http.SameSiteNoneMode
		}
		cookies = append(cookies, cookie)
	}
	return cookies, nil
}

func readTokenWithSession(ctx context.Context, selector string) (string, error) {
	var token string
	expr := fmt.Sprintf(`(() => {
		if (window.csrfToken) return window.csrfToken;
		const el = document.querySelector(%q);
		return el && el.content ? el.content : "";
	})()`, selector)
	if err := chromedp.Run(ctx, chromedp.Evaluate(expr, &token)); err != nil {
		return "", err
	}
	return token, nil
}
