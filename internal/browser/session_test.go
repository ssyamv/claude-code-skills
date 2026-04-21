package browser

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSessionOptionsUseResolvedProfile(t *testing.T) {
	profile := BrowserProfile{
		BrowserName: "chrome",
		BinaryPath:  "/opt/google/chrome/chrome",
		UserDataDir: "/tmp/chrome-profile",
	}

	cfg := sessionAllocatorConfigFromProfile(profile)

	if cfg.execPath != profile.BinaryPath {
		t.Fatalf("expected exec path %q, got %q", profile.BinaryPath, cfg.execPath)
	}
	if cfg.userDataDir != profile.UserDataDir {
		t.Fatalf("expected user data dir %q, got %q", profile.UserDataDir, cfg.userDataDir)
	}
	if cfg.headless {
		t.Fatal("expected headless to be disabled")
	}
	if !cfg.noFirstRun {
		t.Fatal("expected no-first-run to be enabled")
	}
}

func TestResolveProfileUsesFallbackBinaryPaths(t *testing.T) {
	tests := []struct {
		name        string
		goos        string
		wantBrowser string
		wantBinary  string
		wantUserDir string
	}{
		{
			name:        "darwin chrome app bundle",
			goos:        "darwin",
			wantBrowser: "chrome",
			wantBinary:  "/Applications/Google Chrome.app/Contents/MacOS/Google Chrome",
			wantUserDir: filepath.Join("/Users/alice", "Library", "Application Support", "Google", "Chrome"),
		},
		{
			name:        "windows edge app bundle",
			goos:        "windows",
			wantBrowser: "edge",
			wantBinary:  filepath.Join("C:\\Users\\alice", "AppData", "Local", "Microsoft", "Edge", "Application", "msedge.exe"),
			wantUserDir: filepath.Join("C:\\Users\\alice", "AppData", "Local", "Microsoft", "Edge", "User Data"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resolver := ProfileResolver{
				LookPath: func(name string) (string, error) {
					if name == tt.wantBinary {
						return name, nil
					}
					return "", errNotFound
				},
				HomeDir: func() (string, error) {
					return "/Users/alice", nil
				},
			}
			if tt.goos == "windows" {
				resolver.HomeDir = func() (string, error) {
					return "C:\\Users\\alice", nil
				}
			}

			profile, err := resolver.Resolve(tt.goos)
			if err != nil {
				t.Fatalf("resolve failed: %v", err)
			}
			if profile.BrowserName != tt.wantBrowser {
				t.Fatalf("expected browser %q, got %q", tt.wantBrowser, profile.BrowserName)
			}
			if profile.BinaryPath != tt.wantBinary {
				t.Fatalf("expected binary path %q, got %q", tt.wantBinary, profile.BinaryPath)
			}
			if profile.UserDataDir != tt.wantUserDir {
				t.Fatalf("expected user data dir %q, got %q", tt.wantUserDir, profile.UserDataDir)
			}
		})
	}
}

func TestResolveProfileStillReportsUnsupportedGOOS(t *testing.T) {
	resolver := ProfileResolver{
		LookPath: func(string) (string, error) {
			return "", errNotFound
		},
	}

	if _, err := resolver.Resolve("linux"); err == nil {
		t.Fatal("expected unsupported platform failure")
	}
}

func TestOpenURLDetachedWithProfileBuildsDetachedCommand(t *testing.T) {
	oldStart := startDetachedBrowserProcessFn
	t.Cleanup(func() {
		startDetachedBrowserProcessFn = oldStart
	})

	var gotProfile BrowserProfile
	var gotURL string
	startDetachedBrowserProcessFn = func(profile BrowserProfile, url string) error {
		gotProfile = profile
		gotURL = url
		return nil
	}

	profile := BrowserProfile{
		BinaryPath:  "/Applications/Google Chrome.app/Contents/MacOS/Google Chrome",
		UserDataDir: "/tmp/chrome-profile",
	}
	if err := OpenURLDetachedWithProfile(profile, "https://open.xfchat.iflytek.com/oauth/authorize?app_id=cli_123"); err != nil {
		t.Fatalf("detached open failed: %v", err)
	}
	if gotProfile.BinaryPath != profile.BinaryPath {
		t.Fatalf("expected profile to be forwarded, got %#v", gotProfile)
	}
	if gotURL != "https://open.xfchat.iflytek.com/oauth/authorize?app_id=cli_123" {
		t.Fatalf("expected url to be forwarded, got %q", gotURL)
	}
}

func TestOpenURLDetachedWithProfileRejectsEmptyInputs(t *testing.T) {
	if err := OpenURLDetachedWithProfile(BrowserProfile{}, "https://open.xfchat.iflytek.com/oauth/authorize?app_id=cli_123"); err == nil {
		t.Fatal("expected empty binary path to fail")
	}
	if err := OpenURLDetachedWithProfile(BrowserProfile{BinaryPath: "/bin/browser"}, ""); err == nil {
		t.Fatal("expected empty url to fail")
	}
}

func TestPrepareAutomationProfileCopiesProfileToNonDefaultDirectory(t *testing.T) {
	source := t.TempDir()
	if err := os.WriteFile(filepath.Join(source, "Local State"), []byte("state"), 0o600); err != nil {
		t.Fatalf("write local state: %v", err)
	}
	if err := os.Symlink("stale-lock", filepath.Join(source, "SingletonLock")); err != nil {
		t.Fatalf("write singleton lock: %v", err)
	}
	if err := os.WriteFile(filepath.Join(source, "DevToolsActivePort"), []byte("9222"), 0o600); err != nil {
		t.Fatalf("write devtools port: %v", err)
	}
	cacheDir := filepath.Join(source, "Default", "Cache")
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		t.Fatalf("mkdir cache: %v", err)
	}
	if err := os.WriteFile(filepath.Join(cacheDir, "entry"), []byte("cache"), 0o600); err != nil {
		t.Fatalf("write cache: %v", err)
	}

	got, cleanup, err := prepareAutomationProfile(BrowserProfile{
		BrowserName: "chrome",
		BinaryPath:  "/bin/browser",
		UserDataDir: source,
	})
	if err != nil {
		t.Fatalf("prepare profile failed: %v", err)
	}
	defer cleanup()

	if got.UserDataDir == "" || got.UserDataDir == source {
		t.Fatalf("expected copied user data dir, got %#v", got)
	}
	if _, err := os.Stat(filepath.Join(got.UserDataDir, "Local State")); err != nil {
		t.Fatalf("expected local state to be copied: %v", err)
	}
	if _, err := os.Lstat(filepath.Join(got.UserDataDir, "SingletonLock")); !os.IsNotExist(err) {
		t.Fatalf("expected singleton lock to be skipped, got %v", err)
	}
	if _, err := os.Stat(filepath.Join(got.UserDataDir, "DevToolsActivePort")); !os.IsNotExist(err) {
		t.Fatalf("expected devtools port to be skipped, got %v", err)
	}
	if _, err := os.Stat(filepath.Join(got.UserDataDir, "Default", "Cache", "entry")); !os.IsNotExist(err) {
		t.Fatalf("expected cache entry to be skipped, got %v", err)
	}
}
