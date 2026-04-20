package browser

import "testing"

func TestResolveProfilePrefersChromeThenEdge(t *testing.T) {
	resolver := ProfileResolver{
		LookPath: func(name string) (string, error) {
			if name == "google-chrome" {
				return "/Applications/Google Chrome.app/Contents/MacOS/Google Chrome", nil
			}
			return "", errNotFound
		},
	}

	profile, err := resolver.Resolve("darwin")
	if err != nil {
		t.Fatalf("resolve failed: %v", err)
	}
	if profile.BrowserName != "chrome" {
		t.Fatalf("expected chrome profile, got %#v", profile)
	}
	if profile.UserDataDir == "" {
		t.Fatalf("expected user data dir, got %#v", profile)
	}
}

func TestResolveProfileFailsWhenHomeDirLookupFails(t *testing.T) {
	resolver := ProfileResolver{
		LookPath: func(name string) (string, error) {
			if name == "google-chrome" {
				return "/Applications/Google Chrome.app/Contents/MacOS/Google Chrome", nil
			}
			return "", errNotFound
		},
		HomeDir: func() (string, error) {
			return "", errNotFound
		},
	}

	if _, err := resolver.Resolve("darwin"); err == nil {
		t.Fatal("expected home dir lookup failure")
	}
}

func TestResolveProfileFailsOnUnsupportedGOOS(t *testing.T) {
	resolver := ProfileResolver{
		LookPath: func(name string) (string, error) {
			if name == "google-chrome" {
				return "/usr/bin/google-chrome", nil
			}
			return "", errNotFound
		},
	}

	if _, err := resolver.Resolve("linux"); err == nil {
		t.Fatal("expected unsupported platform failure")
	}
}
