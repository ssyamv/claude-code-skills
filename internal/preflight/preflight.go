package preflight

import (
	"fmt"
	"runtime"
	"strings"
)

type Result struct {
	Supported bool
	Reason    string
}

type Checker struct {
	DetectDefaultBrowser func() (string, error)
	CheckPort8080        func() error
	CheckWritableRoot    func() error
}

func (c Checker) Run() (Result, error) {
	if !isSupportedPlatform(runtime.GOOS) {
		return Result{Supported: false, Reason: "platform must be macOS or Windows"}, nil
	}

	browser, err := c.detectDefaultBrowser()
	if err != nil {
		return Result{}, err
	}
	if !isSupportedBrowser(browser) {
		return Result{Supported: false, Reason: "default browser must be Chrome or Edge for release 1"}, nil
	}

	if err := c.checkPort8080(); err != nil {
		return Result{Supported: false, Reason: "port 8080 is unavailable"}, nil
	}
	if err := c.checkWritableRoot(); err != nil {
		return Result{Supported: false, Reason: "install directory is not writable"}, nil
	}

	return Result{Supported: true}, nil
}

func (c Checker) detectDefaultBrowser() (string, error) {
	if c.DetectDefaultBrowser == nil {
		return "", fmt.Errorf("DetectDefaultBrowser is not configured")
	}
	return c.DetectDefaultBrowser()
}

func (c Checker) checkPort8080() error {
	if c.CheckPort8080 == nil {
		return fmt.Errorf("CheckPort8080 is not configured")
	}
	return c.CheckPort8080()
}

func (c Checker) checkWritableRoot() error {
	if c.CheckWritableRoot == nil {
		return fmt.Errorf("CheckWritableRoot is not configured")
	}
	return c.CheckWritableRoot()
}

func isSupportedPlatform(goos string) bool {
	return goos == "darwin" || goos == "windows"
}

func isSupportedBrowser(browser string) bool {
	switch strings.ToLower(strings.TrimSpace(browser)) {
	case "chrome", "google chrome", "edge", "msedge", "microsoft edge":
		return true
	default:
		return false
	}
}
