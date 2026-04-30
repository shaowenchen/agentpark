package datastore

import (
	"crypto/rand"
	"encoding/hex"

	"github.com/agentpark/agentpark/pkg/keys"
)

func randHex(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// NewAgentID 生成 agent- 前缀的主键（与 Memory / SQL 实现共用）。
func NewAgentID() string {
	return keys.AgentIDPrefix + randHex(16)
}

// NewUserAPIKey 生成 user- 前缀的 API Key。
func NewUserAPIKey() string {
	return keys.UserKeyPrefix + randHex(16)
}

// NewUserWorkspaceID 生成注册用户 workspace id。
func NewUserWorkspaceID() string {
	return keys.UserKeyPrefix + "ws-" + randHex(8)
}

// NewShareToken 生成分享令牌。
func NewShareToken() string {
	b := make([]byte, 24)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
