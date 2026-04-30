package objectstore

import (
	"context"
	"io"
	"strings"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// S3Minio 使用 MinIO 客户端访问任意 S3 兼容 API，用于 Agent 压缩包对象（AWS S3、OSS、COS、MinIO 等）。
type S3Minio struct {
	cli    *minio.Client
	bucket string
	prefix string
}

// NewS3Minio endpoint 不含路径，例如 play.min.io 或 s3.amazonaws.com；prefix 可为空。
func NewS3Minio(endpoint, accessKey, secretKey, bucket, prefix string, useSSL bool) (*S3Minio, error) {
	ep := strings.TrimSpace(endpoint)
	ep = strings.TrimPrefix(ep, "https://")
	ep = strings.TrimPrefix(ep, "http://")
	cli, err := minio.New(ep, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: useSSL,
	})
	if err != nil {
		return nil, err
	}
	p := strings.Trim(prefix, "/")
	if p != "" {
		p += "/"
	}
	return &S3Minio{cli: cli, bucket: bucket, prefix: p}, nil
}

func (s *S3Minio) objectKey(key string) string {
	return s.prefix + strings.TrimPrefix(key, "/")
}

func (s *S3Minio) Driver() string { return "s3" }

func (s *S3Minio) Put(ctx context.Context, key string, r io.Reader, size int64, contentType string) error {
	opts := minio.PutObjectOptions{}
	if contentType != "" {
		opts.ContentType = contentType
	}
	if size < 0 {
		size = -1
	}
	_, err := s.cli.PutObject(ctx, s.bucket, s.objectKey(key), r, size, opts)
	return err
}

func (s *S3Minio) Get(ctx context.Context, key string) (io.ReadCloser, error) {
	o, err := s.cli.GetObject(ctx, s.bucket, s.objectKey(key), minio.GetObjectOptions{})
	if err != nil {
		return nil, err
	}
	return o, nil
}

func (s *S3Minio) Delete(ctx context.Context, key string) error {
	return s.cli.RemoveObject(ctx, s.bucket, s.objectKey(key), minio.RemoveObjectOptions{})
}

func (s *S3Minio) Exists(ctx context.Context, key string) (bool, error) {
	_, err := s.cli.StatObject(ctx, s.bucket, s.objectKey(key), minio.StatObjectOptions{})
	if err != nil {
		errResp := minio.ToErrorResponse(err)
		if errResp.Code == "NoSuchKey" || errResp.Code == "NotFound" {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

var _ BlobStore = (*S3Minio)(nil)
