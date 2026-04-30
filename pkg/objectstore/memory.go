package objectstore

import (
	"bytes"
	"context"
	"io"
	"sync"
)

// Memory 进程内 BlobStore（仅占位 Agent 压缩包），用于测试或未启用 S3 时。
type Memory struct {
	mu   sync.RWMutex
	data map[string][]byte
}

func NewMemory() *Memory {
	return &Memory{data: make(map[string][]byte)}
}

func (m *Memory) Driver() string { return "memory" }

func (m *Memory) Put(ctx context.Context, key string, r io.Reader, _ int64, _ string) error {
	b, err := io.ReadAll(r)
	if err != nil {
		return err
	}
	_ = ctx
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.data == nil {
		m.data = make(map[string][]byte)
	}
	m.data[key] = bytes.Clone(b)
	return nil
}

func (m *Memory) Get(ctx context.Context, key string) (io.ReadCloser, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	b, ok := m.data[key]
	if !ok {
		return nil, errNotFound{}
	}
	_ = ctx
	return io.NopCloser(bytes.NewReader(bytes.Clone(b))), nil
}

func (m *Memory) Delete(ctx context.Context, key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.data, key)
	_ = ctx
	return nil
}

func (m *Memory) Exists(ctx context.Context, key string) (bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	_, ok := m.data[key]
	_ = ctx
	return ok, nil
}

type errNotFound struct{}

func (errNotFound) Error() string { return "agent package object not found" }

var _ BlobStore = (*Memory)(nil)
