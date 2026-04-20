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
	HomeDir  func() (string, error)
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

	home, err := homeDir()
	if err != nil {
		return BrowserProfile{}, fmt.Errorf("resolve home dir: %w", err)
	}

	for _, candidate := range browserCandidates(goos, home) {
		if binary, err := lookPath(candidate.binaryPath); err == nil {
			return BrowserProfile{
				BrowserName: candidate.browserName,
				BinaryPath:  binary,
				UserDataDir: userDataDir(goos, home, candidate.browserName),
			}, nil
		}
	}

	return BrowserProfile{}, errNotFound
}

type browserCandidate struct {
	browserName string
	binaryPath  string
}

func browserCandidates(goos, home string) []browserCandidate {
	switch goos {
	case "darwin":
		return []browserCandidate{
			{browserName: "chrome", binaryPath: "google-chrome"},
			{browserName: "chrome", binaryPath: "/Applications/Google Chrome.app/Contents/MacOS/Google Chrome"},
			{browserName: "edge", binaryPath: "msedge"},
			{browserName: "edge", binaryPath: "/Applications/Microsoft Edge.app/Contents/MacOS/Microsoft Edge"},
		}
	case "windows":
		return []browserCandidate{
			{browserName: "chrome", binaryPath: "google-chrome"},
			{browserName: "chrome", binaryPath: "chrome.exe"},
			{browserName: "chrome", binaryPath: filepath.Join(home, "AppData", "Local", "Google", "Chrome", "Application", "chrome.exe")},
			{browserName: "chrome", binaryPath: "C:\\Program Files\\Google\\Chrome\\Application\\chrome.exe"},
			{browserName: "chrome", binaryPath: "C:\\Program Files (x86)\\Google\\Chrome\\Application\\chrome.exe"},
			{browserName: "edge", binaryPath: "msedge"},
			{browserName: "edge", binaryPath: "msedge.exe"},
			{browserName: "edge", binaryPath: filepath.Join(home, "AppData", "Local", "Microsoft", "Edge", "Application", "msedge.exe")},
			{browserName: "edge", binaryPath: "C:\\Program Files\\Microsoft\\Edge\\Application\\msedge.exe"},
			{browserName: "edge", binaryPath: "C:\\Program Files (x86)\\Microsoft\\Edge\\Application\\msedge.exe"},
		}
	default:
		return nil
	}
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
