package mysqlstore

var migrateStatements = []string{
	`CREATE TABLE IF NOT EXISTS agentpark_workspaces (
  id VARCHAR(128) PRIMARY KEY
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`,
	`CREATE TABLE IF NOT EXISTS agentpark_api_keys (
  api_key VARCHAR(255) PRIMARY KEY,
  workspace_id VARCHAR(128) NOT NULL,
  INDEX idx_workspace (workspace_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`,
	`CREATE TABLE IF NOT EXISTS agentpark_agents (
  workspace_id VARCHAR(128) NOT NULL,
  agent_id VARCHAR(128) NOT NULL,
  body JSON NOT NULL,
  PRIMARY KEY (workspace_id, agent_id),
  INDEX idx_workspace (workspace_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`,
	`CREATE TABLE IF NOT EXISTS agentpark_external_idx (
  workspace_id VARCHAR(128) NOT NULL,
  origin VARCHAR(64) NOT NULL,
  external_id VARCHAR(768) NOT NULL,
  agent_id VARCHAR(128) NOT NULL,
  UNIQUE KEY uk_ext (workspace_id, origin, external_id),
  INDEX idx_agent (workspace_id, agent_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`,
	`CREATE TABLE IF NOT EXISTS agentpark_shares (
  token VARCHAR(128) PRIMARY KEY,
  workspace_id VARCHAR(128) NOT NULL,
  agent_id VARCHAR(128) NOT NULL,
  body JSON NOT NULL,
  INDEX idx_workspace (workspace_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`,
}
