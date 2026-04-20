package larkcli

import (
	"os"
	"path/filepath"
	"testing"
)

func TestInstallerWritesExpectedBinaryPath(t *testing.T) {
	root := t.TempDir()
	installer := Installer{
		Platform: "darwin",
		WriteBinary: func(path string) error {
			return os.WriteFile(path, []byte("binary"), 0o755)
		},
	}

	path, err := installer.Install(root)
	if err != nil {
		t.Fatalf("install failed: %v", err)
	}

	expected := filepath.Join(root, "XfchatLarkCli", "bin", "lark-cli")
	if path != expected {
		t.Fatalf("expected %q, got %q", expected, path)
	}
}
