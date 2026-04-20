package state

import (
	stderrors "errors"
	"path/filepath"
	"reflect"
	"testing"
)

func TestStoreRoundTrip(t *testing.T) {
	store := NewStore(filepath.Join(t.TempDir(), "nested"))
	input := BootstrapState{
		Phase:  PhasePlatformSetup,
		AppID:  "cli_123",
		AppURL: "https://open.xfchat.iflytek.com/app/cli_123/baseinfo",
		LastError: &RecoveryError{
			Kind:    RecoveryKindRetryable,
			Message: "temporary timeout",
		},
	}

	if err := store.Save(input); err != nil {
		t.Fatalf("save failed: %v", err)
	}

	got, err := store.Load()
	if err != nil {
		t.Fatalf("load failed: %v", err)
	}

	if !reflect.DeepEqual(got, input) {
		t.Fatalf("unexpected round trip state:\nwant: %#v\ngot:  %#v", input, got)
	}
}

func TestLoadMissingStateFileReturnsNotFound(t *testing.T) {
	store := NewStore(t.TempDir())

	got, err := store.Load()
	if !stderrors.Is(err, ErrStateNotFound) {
		t.Fatalf("expected ErrStateNotFound, got %v", err)
	}
	if !reflect.DeepEqual(got, BootstrapState{}) {
		t.Fatalf("expected zero state, got %#v", got)
	}
}

func TestRecoveryErrorToRuntimeErrorRejectsUnknownKind(t *testing.T) {
	rec := &RecoveryError{
		Kind:    RecoveryKind("bogus"),
		Message: "bad state",
	}

	got, err := rec.ToRuntimeError()
	if err == nil {
		t.Fatal("expected invalid kind to return an error")
	}
	if got != nil {
		t.Fatalf("expected no runtime error, got %#v", got)
	}
}
