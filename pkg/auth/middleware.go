package auth

import (
	"net/http"
	"os"
	"strings"

	"github.com/agentpark/agentpark/pkg/store"
)

// Middleware envStrict 为 true（设置了 AGENTPARK_API_KEY）时，未携带合法 Key 的请求返回 401；
// 否则未携带 Key 时使用 workspace default（本地开发 / 匿名体验）。
// 以下路径免鉴权：公开目录、分享、注册。
func Middleware(hub *store.Hub, envStrict bool) func(http.Handler) http.Handler {
	if !envStrict {
		envStrict = strings.TrimSpace(os.Getenv("AGENTPARK_REQUIRE_KEY")) == "1"
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			path := r.URL.Path
			if !strings.HasPrefix(path, "/api/") {
				next.ServeHTTP(w, r)
				return
			}
			if isPublicAPI(r.Method, path) {
				next.ServeHTTP(w, r)
				return
			}
			key := extractAPIKey(r)
			if ws, ok := hub.WorkspaceForKey(key); ok {
				next.ServeHTTP(w, authWith(r, ws))
				return
			}
			if envStrict {
				http.Error(w, "missing or invalid API key", http.StatusUnauthorized)
				return
			}
			next.ServeHTTP(w, authWith(r, "default"))
		})
	}
}

func isPublicAPI(method, path string) bool {
	if strings.HasPrefix(path, "/api/v1/public/") {
		return true
	}
	if path == "/api/v1/auth/register" && method == http.MethodPost {
		return true
	}
	if path == "/api/v1/catalog/agents" && method == http.MethodGet {
		return true
	}
	return false
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
