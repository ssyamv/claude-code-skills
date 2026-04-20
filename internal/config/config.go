package config

import (
	"os"
	"path/filepath"
	"runtime"
)

type Config struct {
	Brand          string
	CallbackURL    string
	RequiredScopes []string
	InstallRoot    string
}

func Default() Config {
	return Config{
		Brand:       "xfchat.iflytek.com",
		CallbackURL: "http://localhost:8080/callback",
		RequiredScopes: []string{
			"docs:document:readonly",
			"im:message:create_as_bot",
		},
		InstallRoot: defaultInstallRoot(),
	}
}

func defaultInstallRoot() string {
	switch runtime.GOOS {
	case "darwin":
		if home, err := os.UserHomeDir(); err == nil && home != "" {
			return filepath.Join(home, "Library", "Application Support")
		}
	case "windows":
		if dir, err := os.UserConfigDir(); err == nil && dir != "" {
			return dir
		}
	}

	if dir, err := os.UserCacheDir(); err == nil && dir != "" {
		return dir
	}
	return filepath.Join(os.TempDir(), "XfchatLarkCli")
}
