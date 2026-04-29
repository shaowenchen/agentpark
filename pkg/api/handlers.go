package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/agentpark/agentpark/pkg/store"
)

type Handlers struct {
	Store *store.Memory
}

func (h *Handlers) ListAgents(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	writeJSON(w, http.StatusOK, h.Store.List())
}

type createAgentBody struct {
	Name   string `json:"name"`
	System string `json:"system"`
}

func (h *Handlers) CreateAgent(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var body createAgentBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	body.Name = strings.TrimSpace(body.Name)
	if body.Name == "" {
		http.Error(w, "name required", http.StatusBadRequest)
		return
	}
	a := h.Store.Upsert(body.Name, strings.TrimSpace(body.System))
	writeJSON(w, http.StatusCreated, a)
}

func (h *Handlers) DeleteAgent(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	id := r.PathValue("id")
	if id == "" || strings.Contains(id, "/") {
		http.Error(w, "bad id", http.StatusBadRequest)
		return
	}
	if err := h.Store.Delete(id); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handlers) Backup(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	b := h.Store.Snapshot()
	writeJSON(w, http.StatusOK, b)
}

func (h *Handlers) Restore(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var b store.Backup
	if err := json.NewDecoder(r.Body).Decode(&b); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	h.Store.Restore(b)
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":          true,
		"restored_at": b.CreatedAt,
		"count":       len(b.Agents),
	})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
