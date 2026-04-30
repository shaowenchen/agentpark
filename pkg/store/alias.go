package store

import "github.com/agentpark/agentpark/pkg/datastore"

// StateVersion 与磁盘 JSON 兼容（同 datastore.SnapshotVersion）。
const StateVersion = datastore.SnapshotVersion

// State、WorkspaceSnap 为快照类型的别名，便于现有代码与 JSON 标签保持不变。
type State = datastore.Snapshot
type WorkspaceSnap = datastore.WorkspaceSnap
