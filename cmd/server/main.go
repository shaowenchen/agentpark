package main

import (
	"embed"
	"io/fs"
	"log"
	"net/http"

	"github.com/agentpark/agentpark/pkg/api"
	"github.com/agentpark/agentpark/pkg/store"
)

//go:embed all:web
var webFS embed.FS

func main() {
	st := store.NewMemory()
	_ = st.Upsert("Demo 助手", "你是一个简洁、专业的助手。")

	h := &api.Handlers{Store: st}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/agents", h.ListAgents)
	mux.HandleFunc("POST /api/agents", h.CreateAgent)
	mux.HandleFunc("DELETE /api/agents/{id}", h.DeleteAgent)
	mux.HandleFunc("GET /api/backup", h.Backup)
	mux.HandleFunc("POST /api/restore", h.Restore)

	sub, err := fs.Sub(webFS, "web")
	if err != nil {
		log.Fatal(err)
	}
	mux.Handle("/", http.FileServer(http.FS(sub)))

	addr := ":8080"
	log.Printf("AgentPark demo: http://127.0.0.1%s", addr)
	log.Fatal(http.ListenAndServe(addr, logRequests(mux)))
}

func logRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("%s %s", r.Method, r.URL.Path)
		next.ServeHTTP(w, r)
	})
}
