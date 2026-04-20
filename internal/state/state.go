package state

import (
	"fmt"

	runtimeerrors "github.com/ssyamv/claude-code-skills/xfchat-bootstrapper/internal/errors"
)

type Phase string

type RecoveryKind string

const (
	RecoveryKindRetryable          RecoveryKind = "retryable"
	RecoveryKindUserActionable     RecoveryKind = "user_actionable"
	RecoveryKindPlatformActionable RecoveryKind = "platform_actionable"
)

const (
	PhasePlatformSetup Phase = "platform_setup"
	PhaseLocalInstall  Phase = "local_install"
	PhaseOAuth         Phase = "oauth"
	PhaseValidate      Phase = "validate"
)

// RecoveryError is the persisted recovery record for the last failure.
type RecoveryError struct {
	Kind    RecoveryKind `json:"kind"`
	Message string       `json:"message"`
}

func (r *RecoveryError) ToRuntimeError() (*runtimeerrors.BootstrapError, error) {
	if r == nil {
		return nil, nil
	}

	var kind runtimeerrors.Kind
	switch r.Kind {
	case RecoveryKindRetryable:
		kind = runtimeerrors.KindRetryable
	case RecoveryKindUserActionable:
		kind = runtimeerrors.KindUserActionable
	case RecoveryKindPlatformActionable:
		kind = runtimeerrors.KindPlatformActionable
	default:
		return nil, fmt.Errorf("unknown recovery kind %q", r.Kind)
	}

	return &runtimeerrors.BootstrapError{
		Kind:    kind,
		Message: r.Message,
	}, nil
}

// BootstrapState persists the resumable bootstrap progress.
type BootstrapState struct {
	Phase       Phase          `json:"phase"`
	AppID       string         `json:"app_id"`
	AppURL      string         `json:"app_url"`
	AuthSuccess bool           `json:"auth_success"`
	LastError   *RecoveryError `json:"last_error,omitempty"`
}
