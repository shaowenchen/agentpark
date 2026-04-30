package datastore

import (
	"context"
	"time"

	"github.com/agentpark/agentpark/pkg/model"
)

// Store 描述 AgentPark 所需的持久化能力；新数据库驱动实现本接口即可接入 Hub。
//
// 约定：实现须对并发调用安全；ctx 预留取消/超时（内存实现可忽略）。
type Store interface {
	Driver() string

	RegisterAPIKey(ctx context.Context, apiKey, workspaceID string)
	WorkspaceForKey(ctx context.Context, apiKey string) (workspaceID string, ok bool)
	AuthEnabled(ctx context.Context) bool

	ListAgents(ctx context.Context, workspaceID string) []model.Agent
	GetAgent(ctx context.Context, workspaceID, id string) (model.Agent, error)
	CreateAgent(ctx context.Context, workspaceID string, a model.Agent) model.Agent
	UpsertByExternalID(ctx context.Context, workspaceID string, a model.Agent) model.Agent
	ReplaceAgent(ctx context.Context, workspaceID, id string, a model.Agent) (model.Agent, error)
	DeleteAgent(ctx context.Context, workspaceID, id string) error

	Snapshot(ctx context.Context, workspaceID string) model.WorkspaceBackup
	Restore(ctx context.Context, workspaceID string, b model.WorkspaceBackup)

	CreateShare(ctx context.Context, workspaceID, agentID string, expiresAt *time.Time) (model.Share, error)
	RevokeShare(ctx context.Context, workspaceID, token string) error
	AgentByShareToken(ctx context.Context, token string) (model.Agent, model.Share, error)

	RegisterNewUser(ctx context.Context) (apiKey, workspaceID string)

	ExportSnapshot(ctx context.Context) *Snapshot
	ApplySnapshot(ctx context.Context, s *Snapshot)
}
