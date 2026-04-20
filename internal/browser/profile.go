package browser

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

var errNotFound = errors.New("browser not found")

type BrowserProfile struct {
	BrowserName string
	BinaryPath  string
	UserDataDir string
}

type ProfileResolver struct {
	LookPath func(string) (string, error)
	HomeDir   func() (string, error)
}

func (r ProfileResolver) Resolve(goos string) (BrowserProfile, error) {
	lookPath := r.LookPath
	if lookPath == nil {
		lookPath = func(name string) (string, error) {
			return "", errNotFound
		}
	}
	homeDir := r.HomeDir
	if homeDir == nil {
		homeDir = os.UserHomeDir
	}

	if !supportedGOOS(goos) {
		return BrowserProfile{}, fmt.Errorf("unsupported platform: %s", goos)
	}

	if binary, err := lookPath("google-chrome"); err == nil {
		home, err := homeDir()
		if err != nil {
			return BrowserProfile{}, fmt.Errorf("resolve home dir: %w", err)
		}
		return BrowserProfile{
			BrowserName: "chrome",
			BinaryPath:  binary,
			UserDataDir: userDataDir(goos, home, "chrome"),
		}, nil
	}

	if binary, err := lookPath("msedge"); err == nil {
		home, err := homeDir()
		if err != nil {
			return BrowserProfile{}, fmt.Errorf("resolve home dir: %w", err)
		}
		return BrowserProfile{
			BrowserName: "edge",
			BinaryPath:  binary,
			UserDataDir: userDataDir(goos, home, "edge"),
		}, nil
	}

	return BrowserProfile{}, errNotFound
}

func supportedGOOS(goos string) bool {
	switch goos {
	case "darwin", "windows":
		return true
	default:
		return false
	}
}

func userDataDir(goos, home, browser string) string {
	switch goos {
	case "darwin":
		if browser == "chrome" {
			return filepath.Join(home, "Library", "Application Support", "Google", "Chrome")
		}
		return filepath.Join(home, "Library", "Application Support", "Microsoft Edge")
	case "windows":
		if browser == "chrome" {
			return filepath.Join(home, "AppData", "Local", "Google", "Chrome", "User Data")
		}
		return filepath.Join(home, "AppData", "Local", "Microsoft", "Edge", "User Data")
	default:
		return ""
	}
}
