package browser

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/chromedp/chromedp"
)

var oauthConfirmScript = `(() => {
	const exactPositive = ['授权', '确认授权', '同意', '允许', 'Authorize', 'Allow', 'Confirm'];
	const loosePositive = ['确认', '继续'];
	const negative = ['取消', '拒绝', '管理', '撤销', 'Cancel', 'Deny'];
	const visible = el => {
		const rect = el.getBoundingClientRect();
		const style = getComputedStyle(el);
		return rect.width > 0 && rect.height > 0 && style.visibility !== 'hidden' && style.display !== 'none';
	};
	const textOf = el => (el.innerText || el.textContent || el.getAttribute('aria-label') || '').trim();
	const nodes = Array.from(document.querySelectorAll('button, [role="button"], a'))
		.filter(visible)
		.map(el => ({ el, text: textOf(el) }))
		.filter(x => x.text && !negative.some(word => x.text.includes(word)));
	const candidates = nodes.filter(x => exactPositive.includes(x.text));
	if (!candidates.length) {
		candidates.push(...nodes.filter(x => loosePositive.some(word => x.text.includes(word))));
	}
	if (!candidates.length) {
		return { clicked: false, title: document.title, buttons: Array.from(document.querySelectorAll('button, [role="button"], a')).filter(visible).map(textOf).filter(Boolean).slice(0, 20) };
	}
	candidates[0].el.click();
	return { clicked: true, text: candidates[0].text };
})()`

type sessionAllocatorConfig struct {
	execPath    string
	userDataDir string
	headless    bool
	noFirstRun  bool
}

func sessionAllocatorConfigFromProfile(profile BrowserProfile) sessionAllocatorConfig {
	return sessionAllocatorConfig{
		execPath:    profile.BinaryPath,
		userDataDir: profile.UserDataDir,
		headless:    false,
		noFirstRun:  true,
	}
}

func SessionOptions(profile BrowserProfile) []chromedp.ExecAllocatorOption {
	cfg := sessionAllocatorConfigFromProfile(profile)

	return []chromedp.ExecAllocatorOption{
		chromedp.ExecPath(cfg.execPath),
		chromedp.UserDataDir(cfg.userDataDir),
		chromedp.Flag("headless", cfg.headless),
		chromedp.Flag("no-first-run", cfg.noFirstRun),
	}
}

func openURL(ctx context.Context, url string) error {
	return chromedp.Run(ctx, chromedp.Navigate(url))
}

var newProfileSessionFn = newProfileSession
var startDetachedBrowserProcessFn = startDetachedBrowserProcess
var prepareAutomationProfileFn = prepareAutomationProfile

func OpenURLWithProfile(ctx context.Context, profile BrowserProfile, url string) error {
	taskCtx, cleanup, err := newProfileSessionFn(ctx, profile)
	if err != nil {
		return err
	}
	defer cleanup()

	return openURL(taskCtx, url)
}

func OpenURLDetachedWithProfile(profile BrowserProfile, url string) error {
	if profile.BinaryPath == "" {
		return fmt.Errorf("browser binary path is empty")
	}
	if url == "" {
		return fmt.Errorf("url is empty")
	}

	return startDetachedBrowserProcessFn(profile, url)
}

func startDetachedBrowserProcess(profile BrowserProfile, url string) error {
	args := []string{"--new-window"}
	if strings.TrimSpace(profile.UserDataDir) != "" {
		args = append(args, "--user-data-dir="+profile.UserDataDir)
	}
	args = append(args, "--no-first-run", url)

	cmd := exec.Command(profile.BinaryPath, args...)
	if err := cmd.Start(); err != nil {
		return err
	}
	return nil
}

func OpenAndConfirmOAuthWithProfile(ctx context.Context, profile BrowserProfile, url string) error {
	if url == "" {
		return fmt.Errorf("url is empty")
	}

	automationProfile, cleanupProfile, err := prepareAutomationProfileFn(profile)
	if err != nil {
		return err
	}
	// Keep the automation browser alive for the callback wait; caller context controls lifetime.
	go func() {
		<-ctx.Done()
		cleanupProfile()
	}()

	taskCtx, cleanupSession, err := newProfileSessionFn(ctx, automationProfile)
	if err != nil {
		cleanupProfile()
		return err
	}

	if err := openURL(taskCtx, url); err != nil {
		cleanupSession()
		cleanupProfile()
		return err
	}

	go func() {
		defer cleanupSession()
		defer cleanupProfile()
		_ = clickOAuthConfirmLoop(taskCtx)
	}()
	return nil
}

func clickOAuthConfirmLoop(ctx context.Context) error {
	var last any
	for i := 0; i < 30; i++ {
		if err := chromedp.Run(ctx, chromedp.Evaluate(oauthConfirmScript, &last)); err != nil {
			return err
		}
		if result, ok := last.(map[string]any); ok {
			if clicked, _ := result["clicked"].(bool); clicked {
				return nil
			}
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(time.Second):
		}
	}
	return fmt.Errorf("oauth confirm button not found")
}

func NewSessionBootstrap(profile BrowserProfile) SessionBootstrap {
	return SessionBootstrap{
		NewSession: func(ctx context.Context, callProfile BrowserProfile) (context.Context, func(), error) {
			if callProfile == (BrowserProfile{}) {
				callProfile = profile
			}
			automationProfile, cleanupProfile, err := prepareAutomationProfileFn(callProfile)
			if err != nil {
				return nil, nil, err
			}
			sessionCtx, cleanupSession, err := newProfileSessionFn(ctx, automationProfile)
			if err != nil {
				cleanupProfile()
				return nil, nil, err
			}
			return sessionCtx, func() {
				cleanupSession()
				cleanupProfile()
			}, nil
		},
		LoadPage: func(ctx context.Context, pageURL string) error {
			return openURL(ctx, pageURL)
		},
		ReadCookies: func(ctx context.Context) ([]*http.Cookie, error) {
			return readCookiesWithSession(ctx)
		},
		ReadToken: func(ctx context.Context, selector string) (string, error) {
			return readTokenWithSession(ctx, selector)
		},
	}
}

func prepareAutomationProfile(profile BrowserProfile) (BrowserProfile, func(), error) {
	if profile.UserDataDir == "" {
		return profile, func() {}, nil
	}

	tmp, err := os.MkdirTemp("", "xfchat-browser-profile-*")
	if err != nil {
		return BrowserProfile{}, nil, err
	}
	cleanup := func() {
		_ = os.RemoveAll(tmp)
	}

	if err := copyBrowserProfile(profile.UserDataDir, tmp); err != nil {
		cleanup()
		return BrowserProfile{}, nil, err
	}

	profile.UserDataDir = tmp
	return profile, cleanup, nil
}

func copyBrowserProfile(source, target string) error {
	return filepath.WalkDir(source, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		rel, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}
		if shouldSkipProfileEntry(rel, entry) {
			if entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		targetPath := filepath.Join(target, rel)
		info, err := entry.Info()
		if err != nil {
			return err
		}

		if entry.IsDir() {
			return os.MkdirAll(targetPath, info.Mode().Perm())
		}
		if info.Mode().Type()&os.ModeSymlink != 0 {
			return nil
		}
		if !info.Mode().IsRegular() {
			return nil
		}

		return copyFile(path, targetPath, info.Mode().Perm())
	})
}

func shouldSkipProfileEntry(rel string, entry os.DirEntry) bool {
	name := entry.Name()
	if strings.HasPrefix(name, "Singleton") || name == "DevToolsActivePort" {
		return true
	}
	if name == "Crashpad" || name == "BrowserMetrics" || name == "CrashpadMetrics-active.pma" {
		return true
	}
	if name == "Cache" || name == "Code Cache" || name == "GPUCache" || name == "ShaderCache" || name == "GrShaderCache" || name == "GraphiteDawnCache" {
		return true
	}
	return strings.Contains(rel, string(filepath.Separator)+"Cache"+string(filepath.Separator))
}

func copyFile(source, target string, perm os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return err
	}

	in, err := os.Open(source)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.OpenFile(target, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, perm)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		_ = out.Close()
		return err
	}
	return out.Close()
}
