package objectstore

import (
	"context"
	"io"
)

// BlobStore 仅用于存放 Agent 侧压缩包等二进制归档（与 MySQL 中的 Agent 元数据分离）。
// 不要把其它业务文件不经约定塞进同一 bucket；通用备份如需落盘仍走 AGENTPARK_STORE 快照。
//
// S3 兼容实现可用于阿里云 OSS、腾讯云 COS、MinIO 等。
type BlobStore interface {
	Driver() string
	Put(ctx context.Context, key string, r io.Reader, size int64, contentType string) error
	Get(ctx context.Context, key string) (io.ReadCloser, error)
	Delete(ctx context.Context, key string) error
	Exists(ctx context.Context, key string) (bool, error)
}
