package state

import (
	"encoding/json"
	stderrors "errors"
	"os"
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

func TestReplaceFileRestoresOriginalOnFailure(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "state.tmp")
	dst := filepath.Join(dir, stateFileName)
	backup := dst + ".bak"

	if err := os.WriteFile(src, []byte("new"), 0o600); err != nil {
		t.Fatalf("write src: %v", err)
	}
	if err := os.WriteFile(dst, []byte("old"), 0o600); err != nil {
		t.Fatalf("write dst: %v", err)
	}

	oldRename := renameFile
	oldRemove := removeFile
	t.Cleanup(func() {
		renameFile = oldRename
		removeFile = oldRemove
	})

	call := 0
	renameFile = func(oldPath, newPath string) error {
		call++
		switch call {
		case 1:
			if oldPath != src || newPath != dst {
				t.Fatalf("unexpected first rename: %s -> %s", oldPath, newPath)
			}
			return os.ErrExist
		case 2:
			if oldPath != dst || newPath != backup {
				t.Fatalf("unexpected backup rename: %s -> %s", oldPath, newPath)
			}
			return os.Rename(oldPath, newPath)
		case 3:
			if oldPath != src || newPath != dst {
				t.Fatalf("unexpected retry rename: %s -> %s", oldPath, newPath)
			}
			return os.ErrInvalid
		case 4:
			if oldPath != backup || newPath != dst {
				t.Fatalf("unexpected restore rename: %s -> %s", oldPath, newPath)
			}
			return os.Rename(oldPath, newPath)
		default:
			t.Fatalf("unexpected rename call %d: %s -> %s", call, oldPath, newPath)
			return nil
		}
	}
	removeFile = func(path string) error {
		if path != backup {
			t.Fatalf("unexpected remove path: %s", path)
		}
		return os.Remove(path)
	}

	if err := replaceFile(src, dst); err == nil {
		t.Fatal("expected replaceFile to fail after retry and restore")
	}

	got, err := os.ReadFile(dst)
	if err != nil {
		t.Fatalf("read dst: %v", err)
	}
	if string(got) != "old" {
		t.Fatalf("expected original file restored, got %q", string(got))
	}
}

func TestLoadRecoversFromBackupWhenPrimaryIsMissing(t *testing.T) {
	dir := t.TempDir()
	store := NewStore(dir)
	expected := BootstrapState{
		Phase:  PhaseValidate,
		AppID:  "cli_123",
		AppURL: "https://open.xfchat.iflytek.com/app/cli_123/baseinfo",
	}

	backupPath := store.path + ".bak"
	data, err := json.MarshalIndent(expected, "", "  ")
	if err != nil {
		t.Fatalf("marshal expected state: %v", err)
	}
	if err := os.WriteFile(backupPath, data, 0o600); err != nil {
		t.Fatalf("write backup state: %v", err)
	}

	got, err := store.Load()
	if err != nil {
		t.Fatalf("load failed: %v", err)
	}
	if !reflect.DeepEqual(got, expected) {
		t.Fatalf("unexpected recovered state:\nwant: %#v\ngot:  %#v", expected, got)
	}

	primaryBytes, err := os.ReadFile(store.path)
	if err != nil {
		t.Fatalf("read restored primary: %v", err)
	}
	if string(primaryBytes) != string(data) {
		t.Fatalf("expected primary to be restored from backup, got %s", string(primaryBytes))
	}
}
