package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/agentpark/agentpark/pkg/auth"
	"github.com/agentpark/agentpark/pkg/model"
	"github.com/agentpark/agentpark/pkg/store"
)

// Server HTTP API（v1 + 少量 legacy 路径）。
type Server struct {
	Hub *store.Hub
}

func (s *Server) ws(r *http.Request) string {
	return auth.WorkspaceID(r.Context())
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// --- v1 ---

func (s *Server) ListV1(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	writeJSON(w, http.StatusOK, s.Hub.ListAgents(s.ws(r)))
}

type agentInput struct {
	Name        string            `json:"name"`
	Description string            `json:"description"`
	System      string            `json:"system"`
	Origin      string            `json:"origin"`
	ExternalID  string            `json:"external_id"`
	Payload     json.RawMessage   `json:"payload"`
	Labels      map[string]string `json:"labels"`
}

func (s *Server) CreateV1(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var body agentInput
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(body.Name) == "" {
		http.Error(w, "name required", http.StatusBadRequest)
		return
	}
	a := model.Agent{
		Name:        strings.TrimSpace(body.Name),
		Description: strings.TrimSpace(body.Description),
		System:      strings.TrimSpace(body.System),
		Origin:      strings.TrimSpace(body.Origin),
		ExternalID:  strings.TrimSpace(body.ExternalID),
		Payload:     body.Payload,
		Labels:      body.Labels,
	}
	out := s.Hub.CreateAgent(s.ws(r), a)
	writeJSON(w, http.StatusCreated, out)
}

func (s *Server) SyncV1(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut && r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var body agentInput
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(body.Name) == "" {
		http.Error(w, "name required", http.StatusBadRequest)
		return
	}
	a := model.Agent{
		Name:        strings.TrimSpace(body.Name),
		Description: strings.TrimSpace(body.Description),
		System:      strings.TrimSpace(body.System),
		Origin:      strings.TrimSpace(body.Origin),
		ExternalID:  strings.TrimSpace(body.ExternalID),
		Payload:     body.Payload,
		Labels:      body.Labels,
	}
	if a.Origin == "" {
		a.Origin = model.OriginGeneric
	}
	out := s.Hub.UpsertByExternalID(s.ws(r), a)
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) GetV1(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	id := r.PathValue("id")
	a, err := s.Hub.GetAgent(s.ws(r), id)
	if err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	writeJSON(w, http.StatusOK, a)
}

func (s *Server) ReplaceV1(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	id := r.PathValue("id")
	var body agentInput
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(body.Name) == "" {
		http.Error(w, "name required", http.StatusBadRequest)
		return
	}
	a := model.Agent{
		Name:        strings.TrimSpace(body.Name),
		Description: strings.TrimSpace(body.Description),
		System:      strings.TrimSpace(body.System),
		Origin:      strings.TrimSpace(body.Origin),
		ExternalID:  strings.TrimSpace(body.ExternalID),
		Payload:     body.Payload,
		Labels:      body.Labels,
	}
	out, err := s.Hub.ReplaceAgent(s.ws(r), id, a)
	if err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) DeleteV1(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	id := r.PathValue("id")
	if err := s.Hub.DeleteAgent(s.ws(r), id); err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) BackupV1(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	b := s.Hub.Snapshot(s.ws(r))
	writeJSON(w, http.StatusOK, b)
}

func (s *Server) RestoreV1(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var raw json.RawMessage
	if err := json.NewDecoder(r.Body).Decode(&raw); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	b, err := decodeWorkspaceBackup(raw)
	if err != nil {
		http.Error(w, "invalid backup payload", http.StatusBadRequest)
		return
	}
	s.Hub.Restore(s.ws(r), b)
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":    true,
		"count": len(b.Agents),
	})
}

func decodeWorkspaceBackup(raw json.RawMessage) (model.WorkspaceBackup, error) {
	var probe struct {
		Schema string `json:"schema"`
	}
	_ = json.Unmarshal(raw, &probe)
	if probe.Schema != "" {
		var b model.WorkspaceBackup
		err := json.Unmarshal(raw, &b)
		return b, err
	}
	// 兼容旧 demo：`version` + `agents`（仅 name/system）
	var legacy struct {
		Agents []struct {
			ID        string    `json:"id"`
			Name      string    `json:"name"`
			System    string    `json:"system"`
			UpdatedAt time.Time `json:"updated_at"`
		} `json:"agents"`
	}
	if err := json.Unmarshal(raw, &legacy); err != nil {
		return model.WorkspaceBackup{}, err
	}
	out := model.WorkspaceBackup{
		Schema:     model.BackupSchema,
		ExportedAt: time.Now().UTC(),
		Agents:     make([]model.Agent, 0, len(legacy.Agents)),
	}
	for _, la := range legacy.Agents {
		out.Agents = append(out.Agents, model.Agent{
			ID:        la.ID,
			Name:      la.Name,
			System:    la.System,
			Origin:    model.OriginGeneric,
			UpdatedAt: la.UpdatedAt,
			Version:   1,
		})
	}
	return out, nil
}

type shareCreateBody struct {
	ExpiresAt *time.Time `json:"expires_at"`
}

func (s *Server) CreateShareV1(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	id := r.PathValue("id")
	var body shareCreateBody
	_ = json.NewDecoder(r.Body).Decode(&body)
	sh, err := s.Hub.CreateShare(s.ws(r), id, body.ExpiresAt)
	if err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	base := originBase(r)
	writeJSON(w, http.StatusCreated, map[string]any{
		"share":     sh,
		"share_url": base + "/api/v1/public/shares/" + sh.Token,
	})
}

func originBase(r *http.Request) string {
	if r.TLS != nil {
		return "https://" + r.Host
	}
	return "http://" + r.Host
}

func (s *Server) RevokeShareV1(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	tok := r.PathValue("token")
	if err := s.Hub.RevokeShare(s.ws(r), tok); err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) PublicShareV1(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	tok := r.PathValue("token")
	ag, sh, err := s.Hub.AgentByShareToken(tok)
	if err != nil {
		switch {
		case errors.Is(err, store.ErrShareRevoked):
			http.Error(w, "share revoked", http.StatusGone)
		case errors.Is(err, store.ErrShareExpired):
			http.Error(w, "share expired", http.StatusGone)
		default:
			http.Error(w, "not found", http.StatusNotFound)
		}
		return
	}
	ag.WorkspaceID = ""
	writeJSON(w, http.StatusOK, map[string]any{
		"agent": ag,
		"share": sh,
	})
}

type forkBody struct {
	ShareToken string `json:"share_token"`
	Name       string `json:"name"`
}

func (s *Server) ForkV1(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var body forkBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(body.ShareToken) == "" {
		http.Error(w, "share_token required", http.StatusBadRequest)
		return
	}
	a, err := s.Hub.ForkFromShare(s.ws(r), strings.TrimSpace(body.ShareToken), body.Name)
	if err != nil {
		if errors.Is(err, store.ErrShareRevoked) || errors.Is(err, store.ErrShareExpired) {
			http.Error(w, err.Error(), http.StatusGone)
			return
		}
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	writeJSON(w, http.StatusCreated, a)
}

// --- legacy /api/* ---

func (s *Server) LegacyList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	writeJSON(w, http.StatusOK, s.Hub.ListAgents(s.ws(r)))
}

type legacyCreate struct {
	Name   string `json:"name"`
	System string `json:"system"`
}

func (s *Server) LegacyCreate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var body legacyCreate
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(body.Name) == "" {
		http.Error(w, "name required", http.StatusBadRequest)
		return
	}
	a := s.Hub.CreateAgent(s.ws(r), model.Agent{
		Name:   strings.TrimSpace(body.Name),
		System: strings.TrimSpace(body.System),
		Origin: model.OriginGeneric,
	})
	writeJSON(w, http.StatusCreated, a)
}

func (s *Server) LegacyDelete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	id := r.PathValue("id")
	if err := s.Hub.DeleteAgent(s.ws(r), id); err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) LegacyBackup(w http.ResponseWriter, r *http.Request) {
	s.BackupV1(w, r)
}

func (s *Server) LegacyRestore(w http.ResponseWriter, r *http.Request) {
	s.RestoreV1(w, r)
}
