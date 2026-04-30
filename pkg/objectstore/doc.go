// Package objectstore 仅用于存放与 Agent 相关的压缩包文件（如 zip / tar.gz 等二进制归档）。
//
// 用户、Agent 元数据与 JSON 快照仍在 MySQL / 文件快照中；本包不负责通用对象存储。
// 对象键建议使用 AgentPackageObjectKey，便于按 workspace / agent 划分路径。
package objectstore
