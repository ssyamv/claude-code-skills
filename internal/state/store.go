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
	return os.WriteFile(s.path, data, 0o600)
}

func (s *Store) Load() (BootstrapState, error) {
	data, err := os.ReadFile(s.path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return BootstrapState{}, ErrStateNotFound
		}
		return BootstrapState{}, err
	}

	var out BootstrapState
	if err := json.Unmarshal(data, &out); err != nil {
		return BootstrapState{}, err
	}
	return out, nil
}
