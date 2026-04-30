package objectstore

import (
	"fmt"
	"os"
	"strings"
)

// Config 解析结果，便于日志。
type Config struct {
	Kind   string // memory | s3
	Bucket string
	Prefix string
}

// FromEnv 根据 AGENTPARK_BLOB_STORE 构造 BlobStore（仅用于 Agent 压缩包，见包文档）。
//
//	AGENTPARK_BLOB_STORE=memory|s3（默认 memory）
//	S3: AGENTPARK_S3_ENDPOINT, AGENTPARK_S3_BUCKET, AGENTPARK_S3_ACCESS_KEY, AGENTPARK_S3_SECRET_KEY
//	    AGENTPARK_S3_PREFIX（可选） AGENTPARK_S3_USE_SSL=1（默认 1）
func FromEnv() (BlobStore, Config, error) {
	kind := strings.ToLower(strings.TrimSpace(os.Getenv("AGENTPARK_BLOB_STORE")))
	if kind == "" {
		kind = "memory"
	}
	switch kind {
	case "memory", "none", "noop":
		return NewMemory(), Config{Kind: "memory"}, nil
	case "s3":
		endpoint := strings.TrimSpace(os.Getenv("AGENTPARK_S3_ENDPOINT"))
		bucket := strings.TrimSpace(os.Getenv("AGENTPARK_S3_BUCKET"))
		ak := strings.TrimSpace(os.Getenv("AGENTPARK_S3_ACCESS_KEY"))
		sk := strings.TrimSpace(os.Getenv("AGENTPARK_S3_SECRET_KEY"))
		prefix := strings.TrimSpace(os.Getenv("AGENTPARK_S3_PREFIX"))
		if endpoint == "" || bucket == "" || ak == "" || sk == "" {
			return nil, Config{}, fmt.Errorf("AGENTPARK_BLOB_STORE=s3 requires AGENTPARK_S3_ENDPOINT, AGENTPARK_S3_BUCKET, AGENTPARK_S3_ACCESS_KEY, AGENTPARK_S3_SECRET_KEY")
		}
		useSSL := true
		if v := strings.TrimSpace(os.Getenv("AGENTPARK_S3_USE_SSL")); v == "0" || strings.EqualFold(v, "false") {
			useSSL = false
		}
		cli, err := NewS3Minio(endpoint, ak, sk, bucket, prefix, useSSL)
		if err != nil {
			return nil, Config{}, err
		}
		return cli, Config{Kind: "s3", Bucket: bucket, Prefix: prefix}, nil
	default:
		return nil, Config{}, fmt.Errorf("AGENTPARK_BLOB_STORE: unsupported %q (memory|s3)", kind)
	}
}
