package orchestrator

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestCallbackServerCapturesSuccessRequest(t *testing.T) {
	server, err := NewEphemeralCallbackServer()
	if err != nil {
		t.Fatalf("new callback server: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	resultCh := make(chan CallbackResult, 1)
	errCh := make(chan error, 1)

	go func() {
		result, err := server.Wait(ctx)
		if err != nil {
			errCh <- err
			return
		}
		resultCh <- result
	}()

	resp, err := http.Get(fmt.Sprintf("%s?code=auth-code-123&state=state-456", server.URL()))
	if err != nil {
		t.Fatalf("callback request failed: %v", err)
	}
	defer resp.Body.Close()

	select {
	case err := <-errCh:
		t.Fatalf("wait failed: %v", err)
	case result := <-resultCh:
		if result.Code != "auth-code-123" {
			t.Fatalf("expected code to be captured, got %#v", result)
		}
		if result.State != "state-456" {
			t.Fatalf("expected state to be captured, got %#v", result)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for callback result")
	}
}

func TestStartCallbackServerFallsBackToEphemeralWhenDefaultPortUnavailable(t *testing.T) {
	occupied, err := net.Listen("tcp", defaultCallbackAddress)
	if err != nil {
		t.Skipf("default callback address is unavailable before test setup: %v", err)
	}
	defer occupied.Close()

	server, err := StartCallbackServer()
	if err != nil {
		t.Fatalf("start callback server with fallback: %v", err)
	}
	defer closeCallbackWaiter(server)

	if strings.Contains(server.URL(), ":8080/") {
		t.Fatalf("expected fallback callback URL to avoid default port, got %q", server.URL())
	}
	if !strings.HasPrefix(server.URL(), "http://127.0.0.1:") {
		t.Fatalf("expected localhost callback URL, got %q", server.URL())
	}
}
