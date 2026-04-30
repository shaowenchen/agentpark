package store

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
)

// JSONFileBackend 将 State 以 JSON 写入单文件（原子替换）。
type JSONFileBackend struct {
	Path string
}

func (b *JSONFileBackend) Kind() string { return "jsonfile" }

func (b *JSONFileBackend) Read(_ context.Context) (*State, error) {
	if b.Path == "" {
		return nil, errors.New("jsonfile backend: empty path")
	}
	data, err := os.ReadFile(b.Path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	var s State
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, err
	}
	if s.Version != 0 && s.Version != StateVersion {
		return nil, errors.New("jsonfile: unsupported state version")
	}
	return &s, nil
}

func (b *JSONFileBackend) Write(_ context.Context, s *State) error {
	if b.Path == "" {
		return errors.New("jsonfile backend: empty path")
	}
	if s == nil {
		return errors.New("jsonfile backend: nil state")
	}
	s.Version = StateVersion
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	dir := filepath.Dir(b.Path)
	if dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}
	tmp := b.Path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, b.Path)
}
