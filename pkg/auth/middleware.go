package auth

import (
	"net/http"
	"strings"

	"github.com/agentpark/agentpark/pkg/store"
)

// Middleware 在未配置任何 API Key 时，所有请求使用 workspace `default`；
// 配置后，除 /api/v1/public/ 外须在 Header 中携带 Bearer 或 X-API-Key。
func Middleware(hub *store.Hub) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			path := r.URL.Path
			if !strings.HasPrefix(path, "/api/") {
				next.ServeHTTP(w, r)
				return
			}
			if strings.HasPrefix(path, "/api/v1/public/") {
				next.ServeHTTP(w, r)
				return
			}
			if !hub.AuthEnabled() {
				next.ServeHTTP(w, authWith(r, "default"))
				return
			}
			key := extractAPIKey(r)
			if key == "" {
				http.Error(w, "missing API key", http.StatusUnauthorized)
				return
			}
			ws, ok := hub.WorkspaceForKey(key)
			if !ok {
				http.Error(w, "invalid API key", http.StatusForbidden)
				return
			}
			next.ServeHTTP(w, authWith(r, ws))
		})
	}
}

func authWith(r *http.Request, workspaceID string) *http.Request {
	return r.WithContext(WithWorkspaceID(r.Context(), workspaceID))
}

func extractAPIKey(r *http.Request) string {
	a := strings.TrimSpace(r.Header.Get("Authorization"))
	const prefix = "Bearer "
	if len(a) > len(prefix) && strings.EqualFold(a[:len(prefix)], prefix) {
		return strings.TrimSpace(a[len(prefix):])
	}
	return strings.TrimSpace(r.Header.Get("X-API-Key"))
}
