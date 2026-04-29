package store

import (
	"time"

	"github.com/agentpark/agentpark/pkg/model"
)

// CatalogWorkspaceID 公开「商店」目录 workspace，仅用于展示与一键安装。
const CatalogWorkspaceID = "_catalog"

// SeedCatalog 写入若干热门示例 Agent（固定 id 便于前端链接）。
func (h *Hub) SeedCatalog() {
	now := time.Now().UTC()
	demos := []model.Agent{
		{
			ID:          "agent-demo-writing",
			Name:        "写作副驾",
			Description: "长文大纲、润色与多语言改写。",
			Origin:      model.OriginHermes,
			System:      "你是一位专业编辑，擅长结构化写作与简洁表达。",
			Version:     1,
			UpdatedAt:   now,
		},
		{
			ID:          "agent-demo-ops",
			Name:        "运维哨兵",
			Description: "日志摘要、告警解读与 Runbook 草稿。",
			Origin:      model.OriginOpenClaw,
			System:      "你熟悉常见云厂商与 Kubernetes，回答偏操作步骤与安全注意。",
			Version:     1,
			UpdatedAt:   now,
		},
		{
			ID:          "agent-demo-review",
			Name:        "代码评审",
			Description: "PR 风险点、测试建议与命名风格提示。",
			Origin:      model.OriginGeneric,
			System:      "你做代码审查时具体指出问题位置与可执行修改建议，语气克制。",
			Version:     1,
			UpdatedAt:   now,
		},
		{
			ID:          "agent-demo-meeting",
			Name:        "会议秘书",
			Description: "议程提炼、行动项与跟进邮件草稿。",
			Origin:      model.OriginHermes,
			System:      "你从杂乱的会议记录中提取决策、负责人与截止时间。",
			Version:     1,
			UpdatedAt:   now,
		},
	}
	for _, a := range demos {
		_ = h.CreateAgent(CatalogWorkspaceID, a)
	}
}

// CloneFromCatalog 将商店中的 Agent 复制到用户 workspace（新 id，清除 external_id）。
func (h *Hub) CloneFromCatalog(dstWorkspace, catalogAgentID string) (model.Agent, error) {
	src, err := h.GetAgent(CatalogWorkspaceID, catalogAgentID)
	if err != nil {
		return model.Agent{}, err
	}
	src.ID = ""
	src.ExternalID = ""
	src.Version = 0
	src.WorkspaceID = ""
	return h.CreateAgent(dstWorkspace, src), nil
}
