package datastore

import "github.com/agentpark/agentpark/pkg/model"

// SnapshotVersion 与磁盘 JSON 快照格式兼容。
const SnapshotVersion = 1

// Snapshot 为可序列化的全量视图（与具体数据库引擎无关）。
type Snapshot struct {
	Version    int                      `json:"version"`
	Keys       map[string]string        `json:"keys"` // api_key -> workspace_id
	Workspaces map[string]WorkspaceSnap `json:"workspaces"`
}

// WorkspaceSnap 单个 workspace 的可序列化视图。
type WorkspaceSnap struct {
	Agents map[string]model.Agent `json:"agents"`
	Shares map[string]model.Share `json:"shares"`
}
