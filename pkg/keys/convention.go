// Package keys 约定用户密钥与 Agent 实体 ID 的前缀，便于插件与网关解析、路由分流。
package keys

import "strings"

const (
	// UserKeyPrefix 用户 API Key（Bearer / X-API-Key）必须以此前缀开头。
	UserKeyPrefix = "user-"
	// AgentIDPrefix 所有 Agent 的主键 id 必须以此前缀开头（含目录示例与同步创建）。
	AgentIDPrefix = "agent-"
)

// IsUserKey 判断是否为用户侧密钥格式。
func IsUserKey(s string) bool {
	return strings.HasPrefix(strings.TrimSpace(s), UserKeyPrefix)
}

// IsAgentID 判断是否为 Agent 主键格式。
func IsAgentID(s string) bool {
	return strings.HasPrefix(s, AgentIDPrefix)
}
