// Package datastore 定义与传输/展示无关的持久化抽象，便于接入多种存储实现（内存、MySQL 等）。
//
// 当前提供 Store 接口与 Memory 实现；MySQL 实现在 pkg/mysqlstore。业务编排仍在 pkg/store.Hub。
package datastore
