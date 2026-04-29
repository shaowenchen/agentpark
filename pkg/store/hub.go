package store

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/agentpark/agentpark/pkg/keys"
	"github.com/agentpark/agentpark/pkg/model"
)

var (
	ErrNotFound     = errors.New("not found")
	ErrShareRevoked = errors.New("share revoked")
	ErrShareExpired = errors.New("share expired")
)

// Hub 内存实现：按 workspace 隔离 Agent，并维护分享令牌。
type Hub struct {
	mu sync.RWMutex

	// apiKey -> workspaceID；空表示未配置 API Key（开发模式仅用 default）
	keys map[string]string

	workspaces map[string]*workspaceState
}

type workspaceState struct {
	agents         map[string]model.Agent
	externalToID   map[string]string // "origin\x00external_id" -> agent id
	sharesByToken  map[string]model.Share
}

func NewHub() *Hub {
	h := &Hub{
		keys:       make(map[string]string),
		workspaces: make(map[string]*workspaceState),
	}
	h.ensureWorkspace("default")
	return h
}

// RegisterAPIKey 将请求头中的 key 映射到独立 workspace（可多次调用注册多个 key）。
func (h *Hub) RegisterAPIKey(apiKey, workspaceID string) {
	apiKey = strings.TrimSpace(apiKey)
	workspaceID = strings.TrimSpace(workspaceID)
	if apiKey == "" || workspaceID == "" {
		return
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.keys == nil {
		h.keys = make(map[string]string)
	}
	h.keys[apiKey] = workspaceID
	h.ensureWorkspaceLocked(workspaceID)
}

// WorkspaceForKey 若 key 已注册则返回 workspace；否则返回 "", false。
func (h *Hub) WorkspaceForKey(apiKey string) (string, bool) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	ws, ok := h.keys[apiKey]
	return ws, ok
}

// AuthEnabled 是否要求客户端携带已注册的 API Key。
func (h *Hub) AuthEnabled() bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.keys) > 0
}

func (h *Hub) ensureWorkspace(id string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.ensureWorkspaceLocked(id)
}

func (h *Hub) ensureWorkspaceLocked(id string) {
	if h.workspaces[id] != nil {
		return
	}
	h.workspaces[id] = &workspaceState{
		agents:         make(map[string]model.Agent),
		externalToID:   make(map[string]string),
		sharesByToken:  make(map[string]model.Share),
	}
}

func extKey(origin, externalID string) string {
	return origin + "\x00" + externalID
}

// ListAgents 列出某 workspace 下全部 Agent。
func (h *Hub) ListAgents(workspaceID string) []model.Agent {
	h.mu.RLock()
	defer h.mu.RUnlock()
	ws := h.workspaces[workspaceID]
	if ws == nil {
		return nil
	}
	out := make([]model.Agent, 0, len(ws.agents))
	for _, a := range ws.agents {
		out = append(out, a)
	}
	return out
}

// GetAgent 按内部 id 读取。
func (h *Hub) GetAgent(workspaceID, id string) (model.Agent, error) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	ws := h.workspaces[workspaceID]
	if ws == nil {
		return model.Agent{}, ErrNotFound
	}
	a, ok := ws.agents[id]
	if !ok {
		return model.Agent{}, ErrNotFound
	}
	return a, nil
}

// CreateAgent 新建 Agent（无 external_id 或不要求幂等时使用）。
func (h *Hub) CreateAgent(workspaceID string, a model.Agent) model.Agent {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.ensureWorkspaceLocked(workspaceID)
	ws := h.workspaces[workspaceID]
	return h.createLocked(ws, workspaceID, a)
}

// UpsertByExternalID 供 OpenClaw / Hermes 插件同步：同一 origin + external_id 覆盖更新并 version++。
func (h *Hub) UpsertByExternalID(workspaceID string, a model.Agent) model.Agent {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.ensureWorkspaceLocked(workspaceID)
	ws := h.workspaces[workspaceID]
	if a.Origin == "" {
		a.Origin = model.OriginGeneric
	}
	if strings.TrimSpace(a.ExternalID) == "" {
		return h.createLocked(ws, workspaceID, a)
	}
	key := extKey(a.Origin, a.ExternalID)
	if id, ok := ws.externalToID[key]; ok {
		prev := ws.agents[id]
		a.ID = id
		a.Version = prev.Version + 1
		a.WorkspaceID = workspaceID
		a.UpdatedAt = time.Now().UTC()
		ws.agents[id] = a
		return a
	}
	a.ID = newAgentID()
	now := time.Now().UTC()
	a.Version = 1
	a.UpdatedAt = now
	a.WorkspaceID = workspaceID
	ws.agents[a.ID] = a
	ws.externalToID[key] = a.ID
	return a
}

func (h *Hub) createLocked(ws *workspaceState, workspaceID string, a model.Agent) model.Agent {
	if a.ID == "" {
		a.ID = newAgentID()
	} else if !keys.IsAgentID(a.ID) {
		a.ID = keys.AgentIDPrefix + a.ID
	}
	now := time.Now().UTC()
	a.Version = 1
	a.UpdatedAt = now
	a.WorkspaceID = workspaceID
	if a.Origin == "" {
		a.Origin = model.OriginGeneric
	}
	ws.agents[a.ID] = a
	if a.ExternalID != "" {
		ws.externalToID[extKey(a.Origin, a.ExternalID)] = a.ID
	}
	return a
}

// ReplaceAgent 按 id 全量替换（version++）。
func (h *Hub) ReplaceAgent(workspaceID, id string, a model.Agent) (model.Agent, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	ws := h.workspaces[workspaceID]
	if ws == nil {
		return model.Agent{}, ErrNotFound
	}
	prev, ok := ws.agents[id]
	if !ok {
		return model.Agent{}, ErrNotFound
	}
	// 维护 external 索引：若 external 变更则更新映射
	if prev.ExternalID != "" {
		delete(ws.externalToID, extKey(prev.Origin, prev.ExternalID))
	}
	a.ID = id
	a.WorkspaceID = workspaceID
	a.Version = prev.Version + 1
	a.UpdatedAt = time.Now().UTC()
	if a.Origin == "" {
		a.Origin = model.OriginGeneric
	}
	ws.agents[id] = a
	if a.ExternalID != "" {
		ws.externalToID[extKey(a.Origin, a.ExternalID)] = id
	}
	return a, nil
}

func (h *Hub) DeleteAgent(workspaceID, id string) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	ws := h.workspaces[workspaceID]
	if ws == nil {
		return ErrNotFound
	}
	a, ok := ws.agents[id]
	if !ok {
		return ErrNotFound
	}
	if a.ExternalID != "" {
		delete(ws.externalToID, extKey(a.Origin, a.ExternalID))
	}
	delete(ws.agents, id)
	return nil
}

// Snapshot 导出整个 workspace（备份）。
func (h *Hub) Snapshot(workspaceID string) model.WorkspaceBackup {
	h.mu.RLock()
	defer h.mu.RUnlock()
	ws := h.workspaces[workspaceID]
	var agents []model.Agent
	if ws != nil {
		agents = make([]model.Agent, 0, len(ws.agents))
		for _, a := range ws.agents {
			agents = append(agents, a)
		}
	}
	return model.WorkspaceBackup{
		Schema:      model.BackupSchema,
		WorkspaceID: workspaceID,
		ExportedAt:  time.Now().UTC(),
		Agents:      agents,
	}
}

// Restore 用备份覆盖该 workspace（与旧版「全量恢复」语义一致）。
func (h *Hub) Restore(workspaceID string, b model.WorkspaceBackup) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.ensureWorkspaceLocked(workspaceID)
	ws := h.workspaces[workspaceID]
	ws.agents = make(map[string]model.Agent, len(b.Agents))
	ws.externalToID = make(map[string]string)
	ws.sharesByToken = make(map[string]model.Share)
	for _, a := range b.Agents {
		if a.ID == "" {
			a.ID = newAgentID()
		} else if !keys.IsAgentID(a.ID) {
			a.ID = keys.AgentIDPrefix + a.ID
		}
		a.WorkspaceID = workspaceID
		if a.Origin == "" {
			a.Origin = model.OriginGeneric
		}
		ws.agents[a.ID] = a
		if a.ExternalID != "" {
			ws.externalToID[extKey(a.Origin, a.ExternalID)] = a.ID
		}
	}
}

// CreateShare 为某 Agent 生成只读分享令牌。
func (h *Hub) CreateShare(workspaceID, agentID string, expiresAt *time.Time) (model.Share, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	ws := h.workspaces[workspaceID]
	if ws == nil {
		return model.Share{}, ErrNotFound
	}
	if _, ok := ws.agents[agentID]; !ok {
		return model.Share{}, ErrNotFound
	}
	tok := newToken()
	sh := model.Share{
		Token:       tok,
		AgentID:     agentID,
		WorkspaceID: workspaceID,
		CreatedAt:   time.Now().UTC(),
		ExpiresAt:   expiresAt,
		Revoked:     false,
	}
	ws.sharesByToken[tok] = sh
	return sh, nil
}

// RevokeShare 撤销分享。
func (h *Hub) RevokeShare(workspaceID, token string) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	ws := h.workspaces[workspaceID]
	if ws == nil {
		return ErrNotFound
	}
	sh, ok := ws.sharesByToken[token]
	if !ok || sh.WorkspaceID != workspaceID {
		return ErrNotFound
	}
	sh.Revoked = true
	ws.sharesByToken[token] = sh
	return nil
}

// AgentByShareToken 公开接口：通过 token 取 Agent 快照（不含 workspace 敏感扩展时可在外层裁剪）。
func (h *Hub) AgentByShareToken(token string) (model.Agent, model.Share, error) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for _, ws := range h.workspaces {
		sh, ok := ws.sharesByToken[token]
		if !ok {
			continue
		}
		if sh.Revoked {
			return model.Agent{}, sh, ErrShareRevoked
		}
		if sh.ExpiresAt != nil && time.Now().UTC().After(*sh.ExpiresAt) {
			return model.Agent{}, sh, ErrShareExpired
		}
		ag, ok := ws.agents[sh.AgentID]
		if !ok {
			return model.Agent{}, sh, ErrNotFound
		}
		return ag, sh, nil
	}
	return model.Agent{}, model.Share{}, ErrNotFound
}

// RegisterNewUser 生成随机 API Key 与独立 workspace（无密码账户模型）。
func (h *Hub) RegisterNewUser() (apiKey, workspaceID string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.keys == nil {
		h.keys = make(map[string]string)
	}
	workspaceID = newUserWorkspaceID()
	for {
		apiKey = newUserAPIKey()
		if _, dup := h.keys[apiKey]; !dup {
			break
		}
	}
	h.keys[apiKey] = workspaceID
	h.ensureWorkspaceLocked(workspaceID)
	return apiKey, workspaceID
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

func randHex(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// newAgentID 生成 Agent 主键，固定前缀 agent-（与 keys.AgentIDPrefix 一致）。
func newAgentID() string {
	return keys.AgentIDPrefix + randHex(16)
}

// newUserAPIKey 生成用户 API Key，固定前缀 user-。
func newUserAPIKey() string {
	return keys.UserKeyPrefix + randHex(16)
}

// newUserWorkspaceID 注册用户独立空间 ID（与 user- 家族一致，避免与 api_key 字符串混淆用 user-ws-）。
func newUserWorkspaceID() string {
	return keys.UserKeyPrefix + "ws-" + randHex(8)
}

func newToken() string {
	b := make([]byte, 24)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
