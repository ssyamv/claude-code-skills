package diagnostics

import (
	"archive/zip"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ssyamv/claude-code-skills/xfchat-bootstrapper/internal/state"
)

func TestRedactSecrets(t *testing.T) {
	input := "app_secret=super-secret-value\nstdout: cli_123"

	got := Redact(input)

	if got == input {
		t.Fatal("expected secret redaction to modify output")
	}
	if !strings.Contains(got, "[REDACTED]") {
		t.Fatalf("expected redaction marker in output, got %q", got)
	}
	if strings.Contains(got, "super-secret-value") {
		t.Fatalf("secret leaked into redacted output: %q", got)
	}
}

func TestRedactCommonSecretShapes(t *testing.T) {
	input := "password: hunter2\n{\"access_token\":\"tok-123\"}\nAuthorization: Bearer abc.def.ghi"

	got := Redact(input)

	for _, secret := range []string{"hunter2", "tok-123", "abc.def.ghi"} {
		if strings.Contains(got, secret) {
			t.Fatalf("secret %q leaked into redacted output: %q", secret, got)
		}
	}
	if strings.Count(got, "[REDACTED]") < 3 {
		t.Fatalf("expected multiple redactions, got %q", got)
	}
}

func TestRedactRemovesCookieAndCSRFHeaders(t *testing.T) {
	input := "Cookie: sid=cookie-123\nX-XSRF-TOKEN: csrf-123\nX-CSRF-Token: csrf-456\nAuthorization: Bearer token-123"

	got := Redact(input)

	if got == input {
		t.Fatalf("expected redaction, got %q", got)
	}
	for _, secret := range []string{"cookie-123", "csrf-123", "csrf-456", "token-123"} {
		if strings.Contains(got, secret) {
			t.Fatalf("expected sensitive value %q to be redacted, got %q", secret, got)
		}
	}
}

func TestRedactRemovesJSONAppSecretFromState(t *testing.T) {
	data, err := json.Marshal(state.BootstrapState{
		AppID:     "cli_123",
		AppSecret: "secret-123",
		AuthURL:   "https://open.xfchat.iflytek.com/oauth/authorize?app_id=cli_123",
	})
	if err != nil {
		t.Fatalf("marshal state failed: %v", err)
	}

	got := Redact(string(data))

	if strings.Contains(got, "secret-123") {
		t.Fatalf("app secret leaked into redacted state: %q", got)
	}
	if !strings.Contains(got, `"app_secret":"[REDACTED]"`) {
		t.Fatalf("expected app_secret JSON redaction, got %q", got)
	}
}

func TestWriteBundleCreatesRedactedZip(t *testing.T) {
	root := t.TempDir()
	files := map[string][]byte{
		"logs/output.log": []byte("app_secret=super-secret-value\nstdout: cli_123"),
		"logs/other.log":  []byte("harmless output"),
	}

	got, err := WriteBundle(root, files)
	if err != nil {
		t.Fatalf("WriteBundle failed: %v", err)
	}

	want := filepath.Join(root, "support-bundle.zip")
	if got != want {
		t.Fatalf("expected bundle path %q, got %q", want, got)
	}

	zr, err := zip.OpenReader(got)
	if err != nil {
		t.Fatalf("open zip failed: %v", err)
	}
	defer zr.Close()

	if len(zr.File) != len(files) {
		t.Fatalf("expected %d files in bundle, got %d", len(files), len(zr.File))
	}

	gotFiles := make(map[string]string, len(zr.File))
	for _, f := range zr.File {
		rc, err := f.Open()
		if err != nil {
			t.Fatalf("open zip entry %q failed: %v", f.Name, err)
		}

		data, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			t.Fatalf("read zip entry %q failed: %v", f.Name, err)
		}

		gotFiles[f.Name] = string(data)
	}

	if got := gotFiles["logs/output.log"]; !strings.Contains(got, "[REDACTED]") {
		t.Fatalf("expected redacted output.log contents, got %q", got)
	}
	if got := gotFiles["logs/output.log"]; strings.Contains(got, "super-secret-value") {
		t.Fatalf("secret leaked into bundle entry output.log: %q", got)
	}
	if got := gotFiles["logs/other.log"]; got != "harmless output" {
		t.Fatalf("unexpected other.log contents: %q", got)
	}
}

func TestWriteBundleRemovesPartialBundleOnError(t *testing.T) {
	root := t.TempDir()
	origCreateEntry := bundleCreateEntry
	bundleCreateEntry = func(zw *zip.Writer, name string) (io.Writer, error) {
		return nil, errors.New("forced create failure")
	}
	t.Cleanup(func() {
		bundleCreateEntry = origCreateEntry
	})

	got, err := WriteBundle(root, map[string][]byte{
		"logs/output.log": []byte("data"),
	})
	if err == nil {
		t.Fatalf("expected WriteBundle to fail, got path %q", got)
	}

	target := filepath.Join(root, "support-bundle.zip")
	if _, statErr := os.Stat(target); !os.IsNotExist(statErr) {
		t.Fatalf("expected no bundle file left behind, stat err=%v", statErr)
	}
}
