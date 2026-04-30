package objectstore

import (
	"path"
	"strings"
)

// AgentPackageObjectKey 生成 S3 对象键，用于某一 Agent 的压缩包文件。
// filename 应包含扩展名（如 bundle.zip）；会做 path.Base 防止路径穿越。
func AgentPackageObjectKey(workspaceID, agentID, filename string) string {
	base := path.Base(strings.ReplaceAll(filename, "\\", "/"))
	if base == "." || base == "/" {
		base = "archive.zip"
	}
	return path.Join("workspaces", workspaceID, "agents", agentID, "packages", base)
}
