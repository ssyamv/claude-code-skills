package browser

import (
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
