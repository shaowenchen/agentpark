package model

import (
	"encoding/json"
	"time"
)

// Agent 表示一条可备份、可共享的 Agent 快照（来源无关的通用壳 + 各框架专有 payload）。
type Agent struct {
	ID          string          `json:"id"`
	WorkspaceID string          `json:"workspace_id,omitempty"`
	Name        string          `json:"name"`
	Origin      string          `json:"origin"` // openclaw | hermes | generic
	ExternalID  string          `json:"external_id,omitempty"`
	Description string          `json:"description,omitempty"`
	System      string          `json:"system,omitempty"`
	Payload     json.RawMessage `json:"payload,omitempty"`
	Labels      map[string]string `json:"labels,omitempty"`
	Version     int             `json:"version"`
	UpdatedAt   time.Time       `json:"updated_at"`
}

// Share 描述一条「只读分享给他人」的链接能力。
type Share struct {
	Token       string     `json:"token"`
	AgentID     string     `json:"agent_id"`
	WorkspaceID string     `json:"workspace_id,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
	Revoked     bool       `json:"revoked"`
}

// WorkspaceBackup 用于整库导出/导入（备份恢复）。
type WorkspaceBackup struct {
	Schema      string    `json:"schema"`
	WorkspaceID string    `json:"workspace_id"`
	ExportedAt  time.Time `json:"exported_at"`
	Agents      []Agent   `json:"agents"`
}

const BackupSchema = "agentpark.workspace.v1"
