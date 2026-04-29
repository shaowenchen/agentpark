package store

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"sync"
	"time"
)

var ErrNotFound = errors.New("not found")

type Agent struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	System    string    `json:"system"` // 系统提示 / 角色设定
	UpdatedAt time.Time `json:"updated_at"`
}

type Backup struct {
	Version   string    `json:"version"`
	CreatedAt time.Time `json:"created_at"`
	Agents    []Agent   `json:"agents"`
}

type Memory struct {
	mu     sync.RWMutex
	agents map[string]Agent
}

func NewMemory() *Memory {
	return &Memory{agents: make(map[string]Agent)}
}

func (m *Memory) List() []Agent {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]Agent, 0, len(m.agents))
	for _, a := range m.agents {
		out = append(out, a)
	}
	return out
}

func (m *Memory) Get(id string) (Agent, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	a, ok := m.agents[id]
	if !ok {
		return Agent{}, ErrNotFound
	}
	return a, nil
}

func (m *Memory) Upsert(name, system string) Agent {
	m.mu.Lock()
	defer m.mu.Unlock()
	now := time.Now().UTC()
	id := newID()
	a := Agent{
		ID:        id,
		Name:      name,
		System:    system,
		UpdatedAt: now,
	}
	m.agents[id] = a
	return a
}

func (m *Memory) Delete(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.agents[id]; !ok {
		return ErrNotFound
	}
	delete(m.agents, id)
	return nil
}

func (m *Memory) Snapshot() Backup {
	m.mu.RLock()
	defer m.mu.RUnlock()
	agents := make([]Agent, 0, len(m.agents))
	for _, a := range m.agents {
		agents = append(agents, a)
	}
	return Backup{
		Version:   "1",
		CreatedAt: time.Now().UTC(),
		Agents:    agents,
	}
}

func (m *Memory) Restore(b Backup) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.agents = make(map[string]Agent, len(b.Agents))
	for _, a := range b.Agents {
		if a.ID == "" {
			a.ID = newID()
		}
		m.agents[a.ID] = a
	}
}

func newID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
