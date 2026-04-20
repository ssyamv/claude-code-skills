package errors

// Kind classifies a bootstrap failure for recovery and presentation.
type Kind string

const (
	KindRetryable          Kind = "retryable"
	KindUserActionable     Kind = "user_actionable"
	KindPlatformActionable Kind = "platform_actionable"
)

// BootstrapError captures the failure kind and user-facing message.
type BootstrapError struct {
	Kind    Kind   `json:"kind"`
	Message string `json:"message"`
}

func (e *BootstrapError) Error() string {
	if e == nil {
		return ""
	}
	return e.Message
}
