package diagnostics

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
)

type redactionRule struct {
	repl string
	re   *regexp.Regexp
}

var redactions = []redactionRule{
	{re: regexp.MustCompile(`(?i)(\bapp[_-]?secret=)([^\s\r\n]+)`), repl: `${1}[REDACTED]`},
	{re: regexp.MustCompile(`(?i)(\bclient[_-]?secret=)([^\s\r\n]+)`), repl: `${1}[REDACTED]`},
	{re: regexp.MustCompile(`(?i)(\baccess[_-]?token=)([^\s\r\n]+)`), repl: `${1}[REDACTED]`},
	{re: regexp.MustCompile(`(?i)(\brefresh[_-]?token=)([^\s\r\n]+)`), repl: `${1}[REDACTED]`},
	{re: regexp.MustCompile(`(?i)(\bpassword\s*[:=]\s*)([^\s\r\n]+)`), repl: `${1}[REDACTED]`},
	{re: regexp.MustCompile(`(?i)(\bCookie:\s*)([^\r\n]+)`), repl: `${1}[REDACTED]`},
	{re: regexp.MustCompile(`(?i)(\bX-XSRF-TOKEN:\s*)([^\r\n]+)`), repl: `${1}[REDACTED]`},
	{re: regexp.MustCompile(`(?i)(\bX-CSRF-Token:\s*)([^\r\n]+)`), repl: `${1}[REDACTED]`},
	{re: regexp.MustCompile(`(?i)("app[_-]?secret"\s*:\s*")([^"]+)(")`), repl: `${1}[REDACTED]${3}`},
	{re: regexp.MustCompile(`(?i)("client[_-]?secret"\s*:\s*")([^"]+)(")`), repl: `${1}[REDACTED]${3}`},
	{re: regexp.MustCompile(`(?i)("access[_-]?token"\s*:\s*")([^"]+)(")`), repl: `${1}[REDACTED]${3}`},
	{re: regexp.MustCompile(`(?i)("refresh[_-]?token"\s*:\s*")([^"]+)(")`), repl: `${1}[REDACTED]${3}`},
	{re: regexp.MustCompile(`(?i)(\bAuthorization:\s*Bearer\s+)([^\s\r\n]+)`), repl: `${1}[REDACTED]`},
	{re: regexp.MustCompile(`(?i)(\bBearer\s+)([^\s\r\n]+)`), repl: `${1}[REDACTED]`},
	{re: regexp.MustCompile(`(?i)(\bsecret=)([^\s\r\n]+)`), repl: `${1}[REDACTED]`},
	{re: regexp.MustCompile(`(?i)(--(?:app[_-]?)?secret(?:=|\s+))([^\s\r\n]+)`), repl: `${1}[REDACTED]`},
}

var bundleCreateEntry = func(zw *zip.Writer, name string) (io.Writer, error) {
	return zw.Create(name)
}

type redactingWriter struct {
	out io.Writer
}

func (w redactingWriter) Write(p []byte) (int, error) {
	_, err := w.out.Write([]byte(Redact(string(p))))
	if err != nil {
		return 0, err
	}
	return len(p), nil
}

func Redact(input string) string {
	for _, rule := range redactions {
		input = rule.re.ReplaceAllString(input, rule.repl)
	}
	return input
}

func WriteBundle(root string, files map[string][]byte) (string, error) {
	if err := os.MkdirAll(root, 0o755); err != nil {
		return "", err
	}

	target := filepath.Join(root, "support-bundle.zip")
	tmp, err := os.CreateTemp(root, "support-bundle-*.zip")
	if err != nil {
		return "", err
	}
	tmpName := tmp.Name()

	cleanup := func(cause error) (string, error) {
		_ = tmp.Close()
		_ = os.Remove(tmpName)
		return "", cause
	}

	zw := zip.NewWriter(tmp)
	for name, data := range files {
		w, err := bundleCreateEntry(zw, name)
		if err != nil {
			_ = zw.Close()
			return cleanup(err)
		}

		if _, err := w.Write([]byte(Redact(string(data)))); err != nil {
			_ = zw.Close()
			return cleanup(err)
		}
	}

	if err := zw.Close(); err != nil {
		return cleanup(err)
	}
	if err := tmp.Close(); err != nil {
		return cleanup(err)
	}

	if err := os.Remove(target); err != nil && !os.IsNotExist(err) {
		_ = os.Remove(tmpName)
		return "", fmt.Errorf("remove existing bundle: %w", err)
	}
	if err := os.Rename(tmpName, target); err != nil {
		_ = os.Remove(tmpName)
		return "", err
	}

	return target, nil
}
