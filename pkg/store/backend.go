package store

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Backend 存储抽象：实现可插拔的读/写快照（内存、本地 JSON、后续 S3/PG 等）。
// 业务逻辑仍集中在 Hub；Backend 只负责 State 的持久化与恢复。
type Backend interface {
	// Kind 返回实现标识，用于日志与运维（如 memory、jsonfile）。
	Kind() string
	Read(ctx context.Context) (*State, error)
	Write(ctx context.Context, s *State) error
}

// MemoryBackend 不落盘；Read 恒为空，Write 恒为 no-op。
type MemoryBackend struct{}

func (MemoryBackend) Kind() string { return "memory" }

func (MemoryBackend) Read(_ context.Context) (*State, error) {
	return nil, nil
}

func (MemoryBackend) Write(_ context.Context, _ *State) error {
	return nil
}

// BackendConfig 由环境变量解析，便于测试与 main 注入。
type BackendConfig struct {
	Kind string // memory | jsonfile
	Path string // jsonfile 时文件路径
}

// BackendFromEnv 根据 AGENTPARK_STORE / AGENTPARK_DATA / AGENTPARK_DISABLE_PERSIST 构造 Backend。
//
//	AGENTPARK_STORE=memory|jsonfile（默认 jsonfile，除非显式 memory）
//	AGENTPARK_DATA  jsonfile 文件路径，默认 data/agentpark.json
//	AGENTPARK_DISABLE_PERSIST=1 等价于 memory
func BackendFromEnv() (Backend, BackendConfig, error) {
	if strings.TrimSpace(os.Getenv("AGENTPARK_DISABLE_PERSIST")) == "1" {
		return MemoryBackend{}, BackendConfig{Kind: "memory"}, nil
	}
	kind := strings.ToLower(strings.TrimSpace(os.Getenv("AGENTPARK_STORE")))
	if kind == "" {
		kind = "jsonfile"
	}
	if kind == "memory" || kind == "none" || kind == "noop" {
		return MemoryBackend{}, BackendConfig{Kind: "memory"}, nil
	}
	if kind != "jsonfile" {
		return nil, BackendConfig{}, fmt.Errorf("AGENTPARK_STORE: unsupported %q (supported: memory, jsonfile)", kind)
	}
	path := strings.TrimSpace(os.Getenv("AGENTPARK_DATA"))
	if path == "" {
		path = filepath.Join("data", "agentpark.json")
	}
	path = filepath.Clean(path)
	return &JSONFileBackend{Path: path}, BackendConfig{Kind: "jsonfile", Path: path}, nil
}
