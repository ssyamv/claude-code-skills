package errors

import "testing"

func TestBootstrapErrorKindValues(t *testing.T) {
	if KindRetryable != "retryable" {
		t.Fatalf("expected retryable kind, got %q", KindRetryable)
	}
	if KindUserActionable != "user_actionable" {
		t.Fatalf("expected user actionable kind, got %q", KindUserActionable)
	}
	if KindPlatformActionable != "platform_actionable" {
		t.Fatalf("expected platform actionable kind, got %q", KindPlatformActionable)
	}
}

func TestBootstrapErrorError(t *testing.T) {
	err := &BootstrapError{
		Kind:    KindRetryable,
		Message: "temporary timeout",
	}

	if got := err.Error(); got != "temporary timeout" {
		t.Fatalf("expected message to be returned, got %q", got)
	}
}
