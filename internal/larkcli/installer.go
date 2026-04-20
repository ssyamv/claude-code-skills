package larkcli

import (
	"os"
	"path/filepath"
)

type Installer struct {
	Platform    string
	WriteBinary func(path string) error
}

func (i Installer) Install(root string) (string, error) {
	name := "lark-cli"
	if i.Platform == "windows" {
		name = "lark-cli.exe"
	}

	target := filepath.Join(root, "XfchatLarkCli", "bin", name)
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return "", err
	}
	if err := i.WriteBinary(target); err != nil {
		return "", err
	}
	return target, nil
}
