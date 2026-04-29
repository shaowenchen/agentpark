package main

import (
	"embed"
	"io/fs"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/agentpark/agentpark/pkg/api"
	"github.com/agentpark/agentpark/pkg/auth"
	"github.com/agentpark/agentpark/pkg/model"
	"github.com/agentpark/agentpark/pkg/store"
)

//go:embed all:web
var webFS embed.FS

func main() {
	hub := store.NewHub()
	loadAPIKeys(hub)
	hub.SeedCatalog()

	_ = hub.CreateAgent("default", model.Agent{
		Name:   "Demo 助手",
		System: "你是一个简洁、专业的助手。",
		Origin: model.OriginGeneric,
	})

	srv := &api.Server{Hub: hub}

	mux := http.NewServeMux()

	mux.HandleFunc("GET /api/v1/agents", srv.ListV1)
	mux.HandleFunc("POST /api/v1/agents", srv.CreateV1)
	mux.HandleFunc("POST /api/v1/agents/fork", srv.ForkV1)
	mux.HandleFunc("PUT /api/v1/sync/agents", srv.SyncV1)
	mux.HandleFunc("POST /api/v1/sync/agents", srv.SyncV1)
	mux.HandleFunc("GET /api/v1/agents/{id}", srv.GetV1)
	mux.HandleFunc("PUT /api/v1/agents/{id}", srv.ReplaceV1)
	mux.HandleFunc("DELETE /api/v1/agents/{id}", srv.DeleteV1)
	mux.HandleFunc("POST /api/v1/agents/{id}/shares", srv.CreateShareV1)
	mux.HandleFunc("GET /api/v1/backup", srv.BackupV1)
	mux.HandleFunc("POST /api/v1/restore", srv.RestoreV1)
	mux.HandleFunc("DELETE /api/v1/shares/{token}", srv.RevokeShareV1)
	mux.HandleFunc("GET /api/v1/public/shares/{token}", srv.PublicShareV1)
	mux.HandleFunc("GET /api/v1/public/catalog/agents/{id}", srv.PublicCatalogAgent)

	mux.HandleFunc("POST /api/v1/auth/register", srv.Register)
	mux.HandleFunc("GET /api/v1/catalog/agents", srv.CatalogList)
	mux.HandleFunc("POST /api/v1/catalog/agents/{id}/install", srv.InstallCatalogAgent)

	mux.HandleFunc("GET /api/agents", srv.LegacyList)
	mux.HandleFunc("POST /api/agents", srv.LegacyCreate)
	mux.HandleFunc("DELETE /api/agents/{id}", srv.LegacyDelete)
	mux.HandleFunc("GET /api/backup", srv.LegacyBackup)
	mux.HandleFunc("POST /api/restore", srv.LegacyRestore)

	sub, err := fs.Sub(webFS, "web")
	if err != nil {
		log.Fatal(err)
	}
	mux.Handle("/", http.FileServer(http.FS(sub)))

	addr := ":8080"
	if p := os.Getenv("PORT"); p != "" {
		addr = ":" + strings.TrimPrefix(p, ":")
	}

	envStrict := strings.TrimSpace(os.Getenv("AGENTPARK_API_KEY")) != ""
	handler := auth.Middleware(hub, envStrict)(mux)
	if envStrict {
		log.Printf("AgentPark: http://127.0.0.1%s (strict: AGENTPARK_API_KEY required for /api/*)", addr)
	} else {
		log.Printf("AgentPark: http://127.0.0.1%s (匿名 default workspace；POST /api/v1/auth/register 获取独立密钥)", addr)
	}
	log.Fatal(http.ListenAndServe(addr, logRequests(handler)))
}

func loadAPIKeys(hub *store.Hub) {
	if k := strings.TrimSpace(os.Getenv("AGENTPARK_API_KEY")); k != "" {
		hub.RegisterAPIKey(k, "default")
	}
	if list := strings.TrimSpace(os.Getenv("AGENTPARK_API_KEYS")); list != "" {
		for _, pair := range strings.Split(list, ",") {
			pair = strings.TrimSpace(pair)
			kv := strings.SplitN(pair, ":", 2)
			if len(kv) != 2 {
				continue
			}
			hub.RegisterAPIKey(strings.TrimSpace(kv[0]), strings.TrimSpace(kv[1]))
		}
	}
}

func logRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("%s %s", r.Method, r.URL.Path)
		next.ServeHTTP(w, r)
	})
}
