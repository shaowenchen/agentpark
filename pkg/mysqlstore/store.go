package mysqlstore

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/agentpark/agentpark/pkg/datastore"
	"github.com/agentpark/agentpark/pkg/keys"
	"github.com/agentpark/agentpark/pkg/model"

	_ "github.com/go-sql-driver/mysql"
)

// Store 实现 datastore.Store，使用 MySQL 持久化 API Key、Agent、分享与 external_id 索引。
type Store struct {
	db *sql.DB
}

// Open 连接 MySQL、执行建表并保证存在 default workspace。DSN 示例：
//
//	user:pass@tcp(127.0.0.1:3306)/agentpark?parseTime=true&charset=utf8mb4&collation=utf8mb4_unicode_ci
func Open(dsn string) (*Store, error) {
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, err
	}
	st := &Store{db: db}
	for _, q := range migrateStatements {
		if _, err := db.Exec(q); err != nil {
			_ = db.Close()
			return nil, err
		}
	}
	ctx := context.Background()
	if _, err := db.ExecContext(ctx, `INSERT IGNORE INTO agentpark_workspaces(id) VALUES ('default')`); err != nil {
		_ = db.Close()
		return nil, err
	}
	return st, nil
}

// Close 关闭连接池。
func (s *Store) Close() error { return s.db.Close() }

func (s *Store) Driver() string { return "mysql" }

func originOrGeneric(o string) string {
	if o == "" {
		return model.OriginGeneric
	}
	return o
}

func (s *Store) ensureWorkspace(ctx context.Context, tx *sql.Tx, workspaceID string) error {
	var exec execer = s.db
	if tx != nil {
		exec = tx
	}
	_, err := exec.ExecContext(ctx, `INSERT IGNORE INTO agentpark_workspaces(id) VALUES (?)`, workspaceID)
	return err
}

type execer interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
}

func (s *Store) RegisterAPIKey(ctx context.Context, apiKey, workspaceID string) {
	apiKey = strings.TrimSpace(apiKey)
	workspaceID = strings.TrimSpace(workspaceID)
	if apiKey == "" || workspaceID == "" {
		return
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return
	}
	defer func() { _ = tx.Rollback() }()
	if err := s.ensureWorkspace(ctx, tx, workspaceID); err != nil {
		return
	}
	_, err = tx.ExecContext(ctx,
		`INSERT INTO agentpark_api_keys(api_key, workspace_id) VALUES (?, ?)
		 ON DUPLICATE KEY UPDATE workspace_id = VALUES(workspace_id)`,
		apiKey, workspaceID)
	if err != nil {
		return
	}
	_ = tx.Commit()
}

func (s *Store) WorkspaceForKey(ctx context.Context, apiKey string) (string, bool) {
	var ws string
	err := s.db.QueryRowContext(ctx,
		`SELECT workspace_id FROM agentpark_api_keys WHERE api_key = ?`, apiKey).Scan(&ws)
	if errors.Is(err, sql.ErrNoRows) {
		return "", false
	}
	if err != nil {
		return "", false
	}
	return ws, true
}

func (s *Store) AuthEnabled(ctx context.Context) bool {
	var n int
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM agentpark_api_keys`).Scan(&n); err != nil {
		return false
	}
	return n > 0
}

func (s *Store) ListAgents(ctx context.Context, workspaceID string) []model.Agent {
	rows, err := s.db.QueryContext(ctx,
		`SELECT body FROM agentpark_agents WHERE workspace_id = ?`, workspaceID)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var out []model.Agent
	for rows.Next() {
		var raw []byte
		if err := rows.Scan(&raw); err != nil {
			continue
		}
		var a model.Agent
		if err := json.Unmarshal(raw, &a); err != nil {
			continue
		}
		out = append(out, a)
	}
	return out
}

func (s *Store) GetAgent(ctx context.Context, workspaceID, id string) (model.Agent, error) {
	var raw []byte
	err := s.db.QueryRowContext(ctx,
		`SELECT body FROM agentpark_agents WHERE workspace_id = ? AND agent_id = ?`,
		workspaceID, id).Scan(&raw)
	if errors.Is(err, sql.ErrNoRows) {
		return model.Agent{}, datastore.ErrNotFound
	}
	if err != nil {
		return model.Agent{}, err
	}
	var a model.Agent
	if err := json.Unmarshal(raw, &a); err != nil {
		return model.Agent{}, err
	}
	return a, nil
}

func normalizeNewAgent(workspaceID string, a model.Agent) model.Agent {
	if a.ID == "" {
		a.ID = datastore.NewAgentID()
	} else if !keys.IsAgentID(a.ID) {
		a.ID = keys.AgentIDPrefix + a.ID
	}
	now := time.Now().UTC()
	a.Version = 1
	a.UpdatedAt = now
	a.WorkspaceID = workspaceID
	if a.Origin == "" {
		a.Origin = model.OriginGeneric
	}
	return a
}

func (s *Store) CreateAgent(ctx context.Context, workspaceID string, a model.Agent) model.Agent {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return a
	}
	defer func() { _ = tx.Rollback() }()
	if err := s.ensureWorkspace(ctx, tx, workspaceID); err != nil {
		return model.Agent{}
	}
	a = normalizeNewAgent(workspaceID, a)
	raw, err := json.Marshal(a)
	if err != nil {
		return model.Agent{}
	}
	if _, err := tx.ExecContext(ctx,
		`INSERT INTO agentpark_agents(workspace_id, agent_id, body) VALUES (?,?,?)`,
		workspaceID, a.ID, raw); err != nil {
		return model.Agent{}
	}
	if a.ExternalID != "" {
		_, _ = tx.ExecContext(ctx,
			`INSERT INTO agentpark_external_idx(workspace_id, origin, external_id, agent_id) VALUES (?,?,?,?)`,
			workspaceID, originOrGeneric(a.Origin), a.ExternalID, a.ID)
	}
	if err := tx.Commit(); err != nil {
		return model.Agent{}
	}
	return a
}

func (s *Store) UpsertByExternalID(ctx context.Context, workspaceID string, a model.Agent) model.Agent {
	if a.Origin == "" {
		a.Origin = model.OriginGeneric
	}
	if strings.TrimSpace(a.ExternalID) == "" {
		return s.CreateAgent(ctx, workspaceID, a)
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return model.Agent{}
	}
	defer func() { _ = tx.Rollback() }()
	if err := s.ensureWorkspace(ctx, tx, workspaceID); err != nil {
		return model.Agent{}
	}

	origin := originOrGeneric(a.Origin)
	var existingID string
	err = tx.QueryRowContext(ctx,
		`SELECT agent_id FROM agentpark_external_idx WHERE workspace_id = ? AND origin = ? AND external_id = ?`,
		workspaceID, origin, a.ExternalID).Scan(&existingID)

	if err == nil {
		var raw []byte
		err = tx.QueryRowContext(ctx,
			`SELECT body FROM agentpark_agents WHERE workspace_id = ? AND agent_id = ? FOR UPDATE`,
			workspaceID, existingID).Scan(&raw)
		if err != nil {
			return model.Agent{}
		}
		var prev model.Agent
		if json.Unmarshal(raw, &prev) != nil {
			return model.Agent{}
		}
		a.ID = existingID
		a.Version = prev.Version + 1
		a.WorkspaceID = workspaceID
		a.UpdatedAt = time.Now().UTC()
		raw2, err := json.Marshal(a)
		if err != nil {
			return model.Agent{}
		}
		if _, err := tx.ExecContext(ctx,
			`UPDATE agentpark_agents SET body = ? WHERE workspace_id = ? AND agent_id = ?`,
			raw2, workspaceID, existingID); err != nil {
			return model.Agent{}
		}
		if err := tx.Commit(); err != nil {
			return model.Agent{}
		}
		return a
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return model.Agent{}
	}

	a.ID = datastore.NewAgentID()
	now := time.Now().UTC()
	a.Version = 1
	a.UpdatedAt = now
	a.WorkspaceID = workspaceID
	raw, err := json.Marshal(a)
	if err != nil {
		return model.Agent{}
	}
	if _, err := tx.ExecContext(ctx,
		`INSERT INTO agentpark_agents(workspace_id, agent_id, body) VALUES (?,?,?)`,
		workspaceID, a.ID, raw); err != nil {
		return model.Agent{}
	}
	if _, err := tx.ExecContext(ctx,
		`INSERT INTO agentpark_external_idx(workspace_id, origin, external_id, agent_id) VALUES (?,?,?,?)`,
		workspaceID, origin, a.ExternalID, a.ID); err != nil {
		return model.Agent{}
	}
	if err := tx.Commit(); err != nil {
		return model.Agent{}
	}
	return a
}

func (s *Store) ReplaceAgent(ctx context.Context, workspaceID, id string, a model.Agent) (model.Agent, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return model.Agent{}, err
	}
	defer func() { _ = tx.Rollback() }()

	var prevRaw []byte
	err = tx.QueryRowContext(ctx,
		`SELECT body FROM agentpark_agents WHERE workspace_id = ? AND agent_id = ? FOR UPDATE`,
		workspaceID, id).Scan(&prevRaw)
	if errors.Is(err, sql.ErrNoRows) {
		return model.Agent{}, datastore.ErrNotFound
	}
	if err != nil {
		return model.Agent{}, err
	}
	var prev model.Agent
	if err := json.Unmarshal(prevRaw, &prev); err != nil {
		return model.Agent{}, err
	}

	_, _ = tx.ExecContext(ctx,
		`DELETE FROM agentpark_external_idx WHERE workspace_id = ? AND agent_id = ?`,
		workspaceID, id)

	a.ID = id
	a.WorkspaceID = workspaceID
	a.Version = prev.Version + 1
	a.UpdatedAt = time.Now().UTC()
	if a.Origin == "" {
		a.Origin = model.OriginGeneric
	}
	raw, err := json.Marshal(a)
	if err != nil {
		return model.Agent{}, err
	}
	if _, err := tx.ExecContext(ctx,
		`UPDATE agentpark_agents SET body = ? WHERE workspace_id = ? AND agent_id = ?`,
		raw, workspaceID, id); err != nil {
		return model.Agent{}, err
	}
	if a.ExternalID != "" {
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO agentpark_external_idx(workspace_id, origin, external_id, agent_id) VALUES (?,?,?,?)`,
			workspaceID, originOrGeneric(a.Origin), a.ExternalID, id); err != nil {
			return model.Agent{}, err
		}
	}
	if err := tx.Commit(); err != nil {
		return model.Agent{}, err
	}
	return a, nil
}

func (s *Store) DeleteAgent(ctx context.Context, workspaceID, id string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	_, _ = tx.ExecContext(ctx,
		`DELETE FROM agentpark_external_idx WHERE workspace_id = ? AND agent_id = ?`,
		workspaceID, id)
	res, err := tx.ExecContext(ctx,
		`DELETE FROM agentpark_agents WHERE workspace_id = ? AND agent_id = ?`,
		workspaceID, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return datastore.ErrNotFound
	}
	return tx.Commit()
}

func (s *Store) Snapshot(ctx context.Context, workspaceID string) model.WorkspaceBackup {
	list := s.ListAgents(ctx, workspaceID)
	return model.WorkspaceBackup{
		Schema:      model.BackupSchema,
		WorkspaceID: workspaceID,
		ExportedAt:  time.Now().UTC(),
		Agents:      list,
	}
}

func (s *Store) Restore(ctx context.Context, workspaceID string, b model.WorkspaceBackup) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.ExecContext(ctx, `INSERT IGNORE INTO agentpark_workspaces(id) VALUES (?)`, workspaceID); err != nil {
		return
	}
	_, _ = tx.ExecContext(ctx, `DELETE FROM agentpark_external_idx WHERE workspace_id = ?`, workspaceID)
	_, _ = tx.ExecContext(ctx, `DELETE FROM agentpark_shares WHERE workspace_id = ?`, workspaceID)
	_, _ = tx.ExecContext(ctx, `DELETE FROM agentpark_agents WHERE workspace_id = ?`, workspaceID)

	for _, a := range b.Agents {
		if a.ID == "" {
			a.ID = datastore.NewAgentID()
		} else if !keys.IsAgentID(a.ID) {
			a.ID = keys.AgentIDPrefix + a.ID
		}
		a.WorkspaceID = workspaceID
		if a.Origin == "" {
			a.Origin = model.OriginGeneric
		}
		raw, err := json.Marshal(a)
		if err != nil {
			continue
		}
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO agentpark_agents(workspace_id, agent_id, body) VALUES (?,?,?)`,
			workspaceID, a.ID, raw); err != nil {
			continue
		}
		if a.ExternalID != "" {
			_, _ = tx.ExecContext(ctx,
				`INSERT INTO agentpark_external_idx(workspace_id, origin, external_id, agent_id) VALUES (?,?,?,?)`,
				workspaceID, originOrGeneric(a.Origin), a.ExternalID, a.ID)
		}
	}
	_ = tx.Commit()
}

func (s *Store) CreateShare(ctx context.Context, workspaceID, agentID string, expiresAt *time.Time) (model.Share, error) {
	_, err := s.GetAgent(ctx, workspaceID, agentID)
	if err != nil {
		return model.Share{}, err
	}
	tok := datastore.NewShareToken()
	sh := model.Share{
		Token:       tok,
		AgentID:     agentID,
		WorkspaceID: workspaceID,
		CreatedAt:   time.Now().UTC(),
		ExpiresAt:   expiresAt,
		Revoked:     false,
	}
	raw, err := json.Marshal(sh)
	if err != nil {
		return model.Share{}, err
	}
	_, err = s.db.ExecContext(ctx,
		`INSERT INTO agentpark_shares(token, workspace_id, agent_id, body) VALUES (?,?,?,?)`,
		tok, workspaceID, agentID, raw)
	if err != nil {
		return model.Share{}, err
	}
	return sh, nil
}

func (s *Store) RevokeShare(ctx context.Context, workspaceID, token string) error {
	var raw []byte
	err := s.db.QueryRowContext(ctx,
		`SELECT body FROM agentpark_shares WHERE token = ? AND workspace_id = ?`,
		token, workspaceID).Scan(&raw)
	if errors.Is(err, sql.ErrNoRows) {
		return datastore.ErrNotFound
	}
	if err != nil {
		return err
	}
	var sh model.Share
	if err := json.Unmarshal(raw, &sh); err != nil {
		return err
	}
	if sh.WorkspaceID != workspaceID {
		return datastore.ErrNotFound
	}
	sh.Revoked = true
	raw2, err := json.Marshal(sh)
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx,
		`UPDATE agentpark_shares SET body = ? WHERE token = ? AND workspace_id = ?`,
		raw2, token, workspaceID)
	return err
}

func (s *Store) AgentByShareToken(ctx context.Context, token string) (model.Agent, model.Share, error) {
	var agentRaw, shareRaw []byte
	err := s.db.QueryRowContext(ctx, `
		SELECT a.body, s.body
		FROM agentpark_shares s
		INNER JOIN agentpark_agents a ON a.workspace_id = s.workspace_id AND a.agent_id = s.agent_id
		WHERE s.token = ?`, token).Scan(&agentRaw, &shareRaw)
	if errors.Is(err, sql.ErrNoRows) {
		return model.Agent{}, model.Share{}, datastore.ErrNotFound
	}
	if err != nil {
		return model.Agent{}, model.Share{}, err
	}
	var ag model.Agent
	var sh model.Share
	if err := json.Unmarshal(agentRaw, &ag); err != nil {
		return model.Agent{}, model.Share{}, err
	}
	if err := json.Unmarshal(shareRaw, &sh); err != nil {
		return model.Agent{}, model.Share{}, err
	}
	if sh.Revoked {
		return model.Agent{}, sh, datastore.ErrShareRevoked
	}
	if sh.ExpiresAt != nil && time.Now().UTC().After(*sh.ExpiresAt) {
		return model.Agent{}, sh, datastore.ErrShareExpired
	}
	return ag, sh, nil
}

func (s *Store) RegisterNewUser(ctx context.Context) (apiKey, workspaceID string) {
	for attempts := 0; attempts < 64; attempts++ {
		apiKey = datastore.NewUserAPIKey()
		workspaceID = datastore.NewUserWorkspaceID()
		tx, err := s.db.BeginTx(ctx, nil)
		if err != nil {
			continue
		}
		if _, err := tx.ExecContext(ctx, `INSERT IGNORE INTO agentpark_workspaces(id) VALUES (?)`, workspaceID); err != nil {
			_ = tx.Rollback()
			continue
		}
		_, err = tx.ExecContext(ctx,
			`INSERT INTO agentpark_api_keys(api_key, workspace_id) VALUES (?, ?)`, apiKey, workspaceID)
		if err == nil {
			_ = tx.Commit()
			return apiKey, workspaceID
		}
		_ = tx.Rollback()
	}
	return datastore.NewUserAPIKey(), datastore.NewUserWorkspaceID()
}

func (s *Store) ExportSnapshot(ctx context.Context) *datastore.Snapshot {
	snap := &datastore.Snapshot{
		Version:    datastore.SnapshotVersion,
		Keys:       make(map[string]string),
		Workspaces: make(map[string]datastore.WorkspaceSnap),
	}

	rows, err := s.db.QueryContext(ctx, `SELECT api_key, workspace_id FROM agentpark_api_keys`)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var k, ws string
			if rows.Scan(&k, &ws) == nil {
				snap.Keys[k] = ws
			}
		}
	}

	wsRows, err := s.db.QueryContext(ctx, `
		SELECT DISTINCT wid FROM (
			SELECT id AS wid FROM agentpark_workspaces
			UNION
			SELECT DISTINCT workspace_id AS wid FROM agentpark_agents
			UNION
			SELECT DISTINCT workspace_id AS wid FROM agentpark_shares
		) x`)
	if err != nil {
		return snap
	}
	defer wsRows.Close()
	wsSeen := make(map[string]struct{})
	for wsRows.Next() {
		var wid string
		if wsRows.Scan(&wid) != nil {
			continue
		}
		wsSeen[wid] = struct{}{}
	}
	for wid := range wsSeen {
		wsnap := datastore.WorkspaceSnap{
			Agents: make(map[string]model.Agent),
			Shares: make(map[string]model.Share),
		}
		arows, err := s.db.QueryContext(ctx,
			`SELECT agent_id, body FROM agentpark_agents WHERE workspace_id = ?`, wid)
		if err == nil {
			for arows.Next() {
				var aid string
				var raw []byte
				if arows.Scan(&aid, &raw) != nil {
					continue
				}
				var a model.Agent
				if json.Unmarshal(raw, &a) != nil {
					continue
				}
				wsnap.Agents[aid] = a
			}
			arows.Close()
		}
		srows, err := s.db.QueryContext(ctx,
			`SELECT token, body FROM agentpark_shares WHERE workspace_id = ?`, wid)
		if err == nil {
			for srows.Next() {
				var tok string
				var raw []byte
				if srows.Scan(&tok, &raw) != nil {
					continue
				}
				var sh model.Share
				if json.Unmarshal(raw, &sh) != nil {
					continue
				}
				wsnap.Shares[tok] = sh
			}
			srows.Close()
		}
		snap.Workspaces[wid] = wsnap
	}

	if _, ok := snap.Workspaces["default"]; !ok {
		snap.Workspaces["default"] = datastore.WorkspaceSnap{
			Agents: make(map[string]model.Agent),
			Shares: make(map[string]model.Share),
		}
	}
	return snap
}

func (s *Store) ApplySnapshot(ctx context.Context, snap *datastore.Snapshot) {
	if snap == nil {
		return
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return
	}
	defer func() { _ = tx.Rollback() }()

	_, _ = tx.ExecContext(ctx, `DELETE FROM agentpark_external_idx`)
	_, _ = tx.ExecContext(ctx, `DELETE FROM agentpark_shares`)
	_, _ = tx.ExecContext(ctx, `DELETE FROM agentpark_agents`)
	_, _ = tx.ExecContext(ctx, `DELETE FROM agentpark_api_keys`)
	_, _ = tx.ExecContext(ctx, `DELETE FROM agentpark_workspaces`)

	if snap.Keys == nil {
		snap.Keys = make(map[string]string)
	}
	if snap.Workspaces == nil {
		snap.Workspaces = make(map[string]datastore.WorkspaceSnap)
	}

	for wid := range snap.Workspaces {
		_, _ = tx.ExecContext(ctx, `INSERT IGNORE INTO agentpark_workspaces(id) VALUES (?)`, wid)
	}
	for _, wid := range snap.Keys {
		_, _ = tx.ExecContext(ctx, `INSERT IGNORE INTO agentpark_workspaces(id) VALUES (?)`, wid)
	}
	for k, wid := range snap.Keys {
		_, _ = tx.ExecContext(ctx,
			`INSERT INTO agentpark_api_keys(api_key, workspace_id) VALUES (?, ?)`, k, wid)
	}

	for wid, wsnap := range snap.Workspaces {
		if wsnap.Agents != nil {
			for aid, a := range wsnap.Agents {
				if a.Origin == "" {
					a.Origin = model.OriginGeneric
				}
				raw, err := json.Marshal(a)
				if err != nil {
					continue
				}
				_, _ = tx.ExecContext(ctx,
					`INSERT INTO agentpark_agents(workspace_id, agent_id, body) VALUES (?,?,?)`,
					wid, aid, raw)
				if a.ExternalID != "" {
					oo := a.Origin
					if oo == "" {
						oo = model.OriginGeneric
					}
					_, _ = tx.ExecContext(ctx,
						`INSERT INTO agentpark_external_idx(workspace_id, origin, external_id, agent_id) VALUES (?,?,?,?)`,
						wid, oo, a.ExternalID, aid)
				}
			}
		}
		if wsnap.Shares != nil {
			for tok, sh := range wsnap.Shares {
				raw, err := json.Marshal(sh)
				if err != nil {
					continue
				}
				_, _ = tx.ExecContext(ctx,
					`INSERT INTO agentpark_shares(token, workspace_id, agent_id, body) VALUES (?,?,?,?)`,
					tok, wid, sh.AgentID, raw)
			}
		}
	}

	_, _ = tx.ExecContext(ctx, `INSERT IGNORE INTO agentpark_workspaces(id) VALUES ('default')`)
	_ = tx.Commit()
}

var _ datastore.Store = (*Store)(nil)
