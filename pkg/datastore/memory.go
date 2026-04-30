package datastore

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/agentpark/agentpark/pkg/keys"
	"github.com/agentpark/agentpark/pkg/model"
)

// Memory 为 Store 的进程内实现，语义与原 Hub 内联 map 一致。
type Memory struct {
	mu sync.RWMutex

	keys       map[string]string
	workspaces map[string]*workspaceState
}

type workspaceState struct {
	agents        map[string]model.Agent
	externalToID  map[string]string // "origin\x00external_id" -> agent id
	sharesByToken map[string]model.Share
}

func extKey(origin, externalID string) string {
	return origin + "\x00" + externalID
}

// NewMemory 构造空存储并保证存在 default workspace。
func NewMemory() *Memory {
	m := &Memory{
		keys:       make(map[string]string),
		workspaces: make(map[string]*workspaceState),
	}
	m.mu.Lock()
	m.ensureWorkspaceLocked("default")
	m.mu.Unlock()
	return m
}

func (m *Memory) Driver() string { return "memory" }

func (m *Memory) RegisterAPIKey(_ context.Context, apiKey, workspaceID string) {
	apiKey = strings.TrimSpace(apiKey)
	workspaceID = strings.TrimSpace(workspaceID)
	if apiKey == "" || workspaceID == "" {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.keys == nil {
		m.keys = make(map[string]string)
	}
	m.keys[apiKey] = workspaceID
	m.ensureWorkspaceLocked(workspaceID)
}

func (m *Memory) WorkspaceForKey(_ context.Context, apiKey string) (string, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	ws, ok := m.keys[apiKey]
	return ws, ok
}

func (m *Memory) AuthEnabled(_ context.Context) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.keys) > 0
}

func (m *Memory) ensureWorkspaceLocked(id string) {
	if m.workspaces[id] != nil {
		return
	}
	m.workspaces[id] = &workspaceState{
		agents:        make(map[string]model.Agent),
		externalToID:  make(map[string]string),
		sharesByToken: make(map[string]model.Share),
	}
}

func (m *Memory) ListAgents(_ context.Context, workspaceID string) []model.Agent {
	m.mu.RLock()
	defer m.mu.RUnlock()
	ws := m.workspaces[workspaceID]
	if ws == nil {
		return nil
	}
	out := make([]model.Agent, 0, len(ws.agents))
	for _, a := range ws.agents {
		out = append(out, a)
	}
	return out
}

func (m *Memory) GetAgent(_ context.Context, workspaceID, id string) (model.Agent, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	ws := m.workspaces[workspaceID]
	if ws == nil {
		return model.Agent{}, ErrNotFound
	}
	a, ok := ws.agents[id]
	if !ok {
		return model.Agent{}, ErrNotFound
	}
	return a, nil
}

func (m *Memory) CreateAgent(_ context.Context, workspaceID string, a model.Agent) model.Agent {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ensureWorkspaceLocked(workspaceID)
	ws := m.workspaces[workspaceID]
	return m.createLocked(ws, workspaceID, a)
}

func (m *Memory) UpsertByExternalID(_ context.Context, workspaceID string, a model.Agent) model.Agent {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ensureWorkspaceLocked(workspaceID)
	ws := m.workspaces[workspaceID]
	if a.Origin == "" {
		a.Origin = model.OriginGeneric
	}
	if strings.TrimSpace(a.ExternalID) == "" {
		return m.createLocked(ws, workspaceID, a)
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
	a.ID = NewAgentID()
	now := time.Now().UTC()
	a.Version = 1
	a.UpdatedAt = now
	a.WorkspaceID = workspaceID
	ws.agents[a.ID] = a
	ws.externalToID[key] = a.ID
	return a
}

func (m *Memory) createLocked(ws *workspaceState, workspaceID string, a model.Agent) model.Agent {
	if a.ID == "" {
		a.ID = NewAgentID()
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

func (m *Memory) ReplaceAgent(_ context.Context, workspaceID, id string, a model.Agent) (model.Agent, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	ws := m.workspaces[workspaceID]
	if ws == nil {
		return model.Agent{}, ErrNotFound
	}
	prev, ok := ws.agents[id]
	if !ok {
		return model.Agent{}, ErrNotFound
	}
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

func (m *Memory) DeleteAgent(_ context.Context, workspaceID, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	ws := m.workspaces[workspaceID]
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

func (m *Memory) Snapshot(_ context.Context, workspaceID string) model.WorkspaceBackup {
	m.mu.RLock()
	defer m.mu.RUnlock()
	ws := m.workspaces[workspaceID]
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

func (m *Memory) Restore(_ context.Context, workspaceID string, b model.WorkspaceBackup) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ensureWorkspaceLocked(workspaceID)
	ws := m.workspaces[workspaceID]
	ws.agents = make(map[string]model.Agent, len(b.Agents))
	ws.externalToID = make(map[string]string)
	ws.sharesByToken = make(map[string]model.Share)
	for _, a := range b.Agents {
		if a.ID == "" {
			a.ID = NewAgentID()
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

func (m *Memory) CreateShare(_ context.Context, workspaceID, agentID string, expiresAt *time.Time) (model.Share, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	ws := m.workspaces[workspaceID]
	if ws == nil {
		return model.Share{}, ErrNotFound
	}
	if _, ok := ws.agents[agentID]; !ok {
		return model.Share{}, ErrNotFound
	}
	tok := NewShareToken()
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

func (m *Memory) RevokeShare(_ context.Context, workspaceID, token string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	ws := m.workspaces[workspaceID]
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

func (m *Memory) AgentByShareToken(_ context.Context, token string) (model.Agent, model.Share, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, ws := range m.workspaces {
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

func (m *Memory) RegisterNewUser(_ context.Context) (apiKey, workspaceID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.keys == nil {
		m.keys = make(map[string]string)
	}
	workspaceID = NewUserWorkspaceID()
	for {
		apiKey = NewUserAPIKey()
		if _, dup := m.keys[apiKey]; !dup {
			break
		}
	}
	m.keys[apiKey] = workspaceID
	m.ensureWorkspaceLocked(workspaceID)
	return apiKey, workspaceID
}

func (m *Memory) ExportSnapshot(_ context.Context) *Snapshot {
	m.mu.RLock()
	defer m.mu.RUnlock()
	s := &Snapshot{
		Version:    SnapshotVersion,
		Keys:       cloneKeys(m.keys),
		Workspaces: make(map[string]WorkspaceSnap, len(m.workspaces)),
	}
	for wid, ws := range m.workspaces {
		wsnap := WorkspaceSnap{
			Agents: make(map[string]model.Agent, len(ws.agents)),
			Shares: make(map[string]model.Share, len(ws.sharesByToken)),
		}
		for id, a := range ws.agents {
			wsnap.Agents[id] = a
		}
		for tok, sh := range ws.sharesByToken {
			wsnap.Shares[tok] = sh
		}
		s.Workspaces[wid] = wsnap
	}
	return s
}

func (m *Memory) ApplySnapshot(_ context.Context, s *Snapshot) {
	if s == nil {
		return
	}
	if s.Keys == nil {
		s.Keys = make(map[string]string)
	}
	if s.Workspaces == nil {
		s.Workspaces = make(map[string]WorkspaceSnap)
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.keys = cloneKeys(s.Keys)
	m.workspaces = make(map[string]*workspaceState, len(s.Workspaces))
	for wid, wsnap := range s.Workspaces {
		st := &workspaceState{
			agents:        make(map[string]model.Agent),
			externalToID:  make(map[string]string),
			sharesByToken: make(map[string]model.Share),
		}
		if wsnap.Agents != nil {
			for id, a := range wsnap.Agents {
				st.agents[id] = a
				if a.ExternalID != "" {
					oo := a.Origin
					if oo == "" {
						oo = model.OriginGeneric
					}
					st.externalToID[extKey(oo, a.ExternalID)] = id
				}
			}
		}
		if wsnap.Shares != nil {
			for tok, sh := range wsnap.Shares {
				st.sharesByToken[tok] = sh
			}
		}
		m.workspaces[wid] = st
	}
	if m.workspaces["default"] == nil {
		m.ensureWorkspaceLocked("default")
	}
}

func cloneKeys(keysMap map[string]string) map[string]string {
	if keysMap == nil {
		return make(map[string]string)
	}
	out := make(map[string]string, len(keysMap))
	for k, v := range keysMap {
		out[k] = v
	}
	return out
}

var _ Store = (*Memory)(nil)
