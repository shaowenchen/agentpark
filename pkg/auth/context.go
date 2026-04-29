package auth

import "context"

type workspaceCtxKey struct{}

// WithWorkspaceID 将当前请求的 workspace 写入 context（由中间件设置）。
func WithWorkspaceID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, workspaceCtxKey{}, id)
}

// WorkspaceID 读取 workspace；未设置时返回空字符串。
func WorkspaceID(ctx context.Context) string {
	s, _ := ctx.Value(workspaceCtxKey{}).(string)
	return s
}
