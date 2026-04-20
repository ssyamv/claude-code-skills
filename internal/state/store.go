package state

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
)

const stateFileName = "bootstrap-state.json"

var ErrStateNotFound = errors.New("bootstrap state not found")

var (
	writeFile      = os.WriteFile
	renameFile     = os.Rename
	createTempFile = os.CreateTemp
	removeFile     = os.Remove
)

// Store persists bootstrap state as a JSON file.
type Store struct {
	path string
}

func NewStore(root string) *Store {
	return &Store{path: filepath.Join(root, stateFileName)}
}

func (s *Store) Save(in BootstrapState) error {
	data, err := json.MarshalIndent(in, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return err
	}

	tmp, err := createTempFile(filepath.Dir(s.path), filepath.Base(s.path)+".*.tmp")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	if err := tmp.Close(); err != nil {
		_ = removeFile(tmpPath)
		return err
	}

	if err := writeFile(tmpPath, data, 0o600); err != nil {
		_ = removeFile(tmpPath)
		return err
	}
	if err := replaceFile(tmpPath, s.path); err != nil {
		_ = removeFile(tmpPath)
		return err
	}
	return nil
}

func replaceFile(sourcePath, targetPath string) error {
	if err := renameFile(sourcePath, targetPath); err == nil {
		return nil
	}

	backupPath := targetPath + ".bak"
	if err := removeFile(backupPath); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}

	if err := renameFile(targetPath, backupPath); err != nil {
		return err
	}

	if err := renameFile(sourcePath, targetPath); err != nil {
		restoreErr := renameFile(backupPath, targetPath)
		if restoreErr != nil {
			return errors.Join(err, restoreErr)
		}
		return err
	}

	if err := removeFile(backupPath); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	return nil
}

func (s *Store) Load() (BootstrapState, error) {
	data, err := os.ReadFile(s.path)
	if err == nil {
		return decodeState(data)
	}
	if !errors.Is(err, fs.ErrNotExist) {
		return BootstrapState{}, err
	}

	backupPath := s.path + ".bak"
	data, err = os.ReadFile(backupPath)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return BootstrapState{}, ErrStateNotFound
		}
		return BootstrapState{}, err
	}

	out, err := decodeState(data)
	if err != nil {
		return BootstrapState{}, err
	}
	_ = renameFile(backupPath, s.path)
	return out, nil
}

func decodeState(data []byte) (BootstrapState, error) {
	var out BootstrapState
	if err := json.Unmarshal(data, &out); err != nil {
		return BootstrapState{}, err
	}
	return out, nil
}
