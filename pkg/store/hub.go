package store

import (
	"context"
	"strings"
	"time"

	"github.com/agentpark/agentpark/pkg/datastore"
	"github.com/agentpark/agentpark/pkg/model"
)

// 与 API 层 errors.Is 兼容：与 datastore 包内为同一哨兵值。
var (
	ErrNotFound     = datastore.ErrNotFound
	ErrShareRevoked = datastore.ErrShareRevoked
	ErrShareExpired = datastore.ErrShareExpired
)

// Hub 在 Store 之上提供与 HTTP 层相同的编排面；持久化细节由 Store 实现。
type Hub struct {
	store        datastore.Store
	stateBackend Backend
}

// NewHub 使用默认的内存 Store。
func NewHub() *Hub {
	return NewHubWithStore(datastore.NewMemory())
}

// NewHubWithStore 用于测试或注入其它 Store 实现（如未来的 SQL 驱动）。
func NewHubWithStore(st datastore.Store) *Hub {
	if st == nil {
		st = datastore.NewMemory()
	}
	return &Hub{store: st}
}

// SetStateBackend 设置快照读写的 Backend（如 jsonfile）；应在 Listen 之前调用一次。
func (h *Hub) SetStateBackend(b Backend) {
	h.stateBackend = b
}

// RegisterAPIKey 将请求头中的 key 映射到独立 workspace（可多次调用注册多个 key）。
func (h *Hub) RegisterAPIKey(apiKey, workspaceID string) {
	h.store.RegisterAPIKey(context.Background(), apiKey, workspaceID)
}

// WorkspaceForKey 若 key 已注册则返回 workspace；否则返回 "", false。
func (h *Hub) WorkspaceForKey(apiKey string) (string, bool) {
	return h.store.WorkspaceForKey(context.Background(), apiKey)
}

// AuthEnabled 是否要求客户端携带已注册的 API Key。
func (h *Hub) AuthEnabled() bool {
	return h.store.AuthEnabled(context.Background())
}

// ListAgents 列出某 workspace 下全部 Agent。
func (h *Hub) ListAgents(workspaceID string) []model.Agent {
	return h.store.ListAgents(context.Background(), workspaceID)
}

// GetAgent 按内部 id 读取。
func (h *Hub) GetAgent(workspaceID, id string) (model.Agent, error) {
	return h.store.GetAgent(context.Background(), workspaceID, id)
}

// CreateAgent 新建 Agent（无 external_id 或不要求幂等时使用）。
func (h *Hub) CreateAgent(workspaceID string, a model.Agent) model.Agent {
	return h.store.CreateAgent(context.Background(), workspaceID, a)
}

// UpsertByExternalID 供 OpenClaw / Hermes 插件同步：同一 origin + external_id 覆盖更新并 version++。
func (h *Hub) UpsertByExternalID(workspaceID string, a model.Agent) model.Agent {
	return h.store.UpsertByExternalID(context.Background(), workspaceID, a)
}

// ReplaceAgent 按 id 全量替换（version++）。
func (h *Hub) ReplaceAgent(workspaceID, id string, a model.Agent) (model.Agent, error) {
	return h.store.ReplaceAgent(context.Background(), workspaceID, id, a)
}

func (h *Hub) DeleteAgent(workspaceID, id string) error {
	return h.store.DeleteAgent(context.Background(), workspaceID, id)
}

// Snapshot 导出整个 workspace（备份）。
func (h *Hub) Snapshot(workspaceID string) model.WorkspaceBackup {
	return h.store.Snapshot(context.Background(), workspaceID)
}

// Restore 用备份覆盖该 workspace（与旧版「全量恢复」语义一致）。
func (h *Hub) Restore(workspaceID string, b model.WorkspaceBackup) {
	h.store.Restore(context.Background(), workspaceID, b)
}

// CreateShare 为某 Agent 生成只读分享令牌。
func (h *Hub) CreateShare(workspaceID, agentID string, expiresAt *time.Time) (model.Share, error) {
	return h.store.CreateShare(context.Background(), workspaceID, agentID, expiresAt)
}

// RevokeShare 撤销分享。
func (h *Hub) RevokeShare(workspaceID, token string) error {
	return h.store.RevokeShare(context.Background(), workspaceID, token)
}

// AgentByShareToken 公开接口：通过 token 取 Agent 快照（不含 workspace 敏感扩展时可在外层裁剪）。
func (h *Hub) AgentByShareToken(token string) (model.Agent, model.Share, error) {
	return h.store.AgentByShareToken(context.Background(), token)
}

// RegisterNewUser 生成随机 API Key 与独立 workspace（无密码账户模型）。
func (h *Hub) RegisterNewUser() (apiKey, workspaceID string) {
	return h.store.RegisterNewUser(context.Background())
}

// ForkFromShare 将他人分享的只读快照复制到当前 workspace（生成新 id，清除 external_id）。
func (h *Hub) ForkFromShare(workspaceID, shareToken, newName string) (model.Agent, error) {
	ag, _, err := h.AgentByShareToken(shareToken)
	if err != nil {
		return model.Agent{}, err
	}
	ag.ID = ""
	ag.ExternalID = ""
	ag.Version = 0
	if strings.TrimSpace(newName) != "" {
		ag.Name = strings.TrimSpace(newName)
	}
	return h.CreateAgent(workspaceID, ag), nil
}

// ExportState 导出当前快照（供 Backend 写入）。
func (h *Hub) ExportState() *State {
	return h.store.ExportSnapshot(context.Background())
}

// ApplyState 用快照覆盖 Store；s 为 nil 则忽略。
func (h *Hub) ApplyState(s *State) {
	if s == nil {
		return
	}
	h.store.ApplySnapshot(context.Background(), s)
}
