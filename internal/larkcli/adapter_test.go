package larkcli

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"
)

func TestAdapterRun(t *testing.T) {
	exe, err := os.Executable()
	if err != nil {
		t.Fatalf("resolve executable: %v", err)
	}

	adapter := Adapter{BinaryPath: exe}
	stdout, stderr, err := adapter.Run(context.Background(), []string{"-test.run=TestHelperProcess", "--", "helper", "alpha", "beta"}, []byte("payload"))
	if err != nil {
		t.Fatalf("run failed: %v", err)
	}

	if stdout != "stdout:alpha,beta" {
		t.Fatalf("unexpected stdout: %q", stdout)
	}
	if stderr != "stderr:payload|no-proxy:1" {
		t.Fatalf("unexpected stderr: %q", stderr)
	}
}

func TestHelperProcess(t *testing.T) {
	if !containsArg(os.Args, "helper") {
		return
	}

	if os.Getenv("LARK_CLI_NO_PROXY") != "1" {
		fmt.Fprint(os.Stdout, "missing-no-proxy")
		os.Exit(0)
	}

	args := os.Args[1:]
	for i, arg := range args {
		if arg == "--" {
			args = args[i+1:]
			break
		}
	}
	if len(args) > 0 && args[0] == "helper" {
		args = args[1:]
	}

	fmt.Fprintf(os.Stdout, "stdout:%s", strings.Join(args, ","))
	stdin, err := io.ReadAll(os.Stdin)
	if err != nil {
		fmt.Fprintf(os.Stderr, "stderr:read-failed")
		os.Exit(0)
	}
	fmt.Fprintf(os.Stderr, "stderr:%s|no-proxy:%s", string(stdin), os.Getenv("LARK_CLI_NO_PROXY"))
	os.Exit(0)
}

func containsArg(args []string, want string) bool {
	for _, arg := range args {
		if arg == want {
			return true
		}
	}
	return false
}
