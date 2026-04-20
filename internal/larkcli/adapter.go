package larkcli

import (
	"bytes"
	"context"
	"os"
	"os/exec"
)

type Adapter struct {
	BinaryPath string
}

func (a Adapter) Run(ctx context.Context, args []string, stdin []byte) (string, string, error) {
	cmd := exec.CommandContext(ctx, a.BinaryPath, args...)
	cmd.Env = os.Environ()
	cmd.Env = append(cmd.Env, "LARK_CLI_NO_PROXY=1")
	cmd.Stdin = bytes.NewReader(stdin)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	return stdout.String(), stderr.String(), err
}
