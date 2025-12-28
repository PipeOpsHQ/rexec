package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/lib/pq"
)

// Agent represents a registered external agent
type Agent struct {
	ID          string                 `json:"id"`
	UserID      string                 `json:"user_id"`
	Username    string                 `json:"username,omitempty"`
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	OS          string                 `json:"os"`
	Arch        string                 `json:"arch"`
	Shell       string                 `json:"shell"`
	Distro      string                 `json:"distro,omitempty"`
	Tags        []string               `json:"tags,omitempty"`
	Status      string                 `json:"status"`
	MFALocked   bool                   `json:"mfa_locked"`
	ConnectedAt time.Time              `json:"connected_at,omitempty"`
	LastPing    time.Time              `json:"last_ping,omitempty"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
	SystemInfo  map[string]interface{} `json:"system_info,omitempty"`
}

// CreateAgentsTable creates the agents table
func (s *PostgresStore) CreateAgentsTable(ctx context.Context) error {
	query := `
	CREATE TABLE IF NOT EXISTS agents (
		id UUID PRIMARY KEY,
		user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		name VARCHAR(255) NOT NULL,
		description TEXT,
		os VARCHAR(50),
		arch VARCHAR(50),
		shell VARCHAR(255),
		distro VARCHAR(100),
		tags TEXT[],
		created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
		updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
	);

	CREATE INDEX IF NOT EXISTS idx_agents_user_id ON agents(user_id);

	-- Add distro column if it doesn't exist (migration)
	DO $$
	BEGIN
		IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'agents' AND column_name = 'distro') THEN
			ALTER TABLE agents ADD COLUMN distro VARCHAR(100);
		END IF;
	END $$;
	`

	_, err := s.db.ExecContext(ctx, query)
	return err
}

// CreateAgent creates a new agent
func (s *PostgresStore) CreateAgent(ctx context.Context, id, userID, name, description, os, arch, shell string, tags []string) error {
	query := `
	INSERT INTO agents (id, user_id, name, description, os, arch, shell, tags)
	VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`

	_, err := s.db.ExecContext(ctx, query, id, userID, name, description, os, arch, shell, pq.Array(tags))
	return err
}

// GetAgent retrieves an agent by ID
func (s *PostgresStore) GetAgent(ctx context.Context, id string) (*Agent, error) {
	query := `
	SELECT id, user_id, name, COALESCE(description, ''), COALESCE(os, ''), COALESCE(arch, ''),
	       COALESCE(shell, ''), COALESCE(distro, ''), tags, COALESCE(mfa_locked, false),
	       created_at, updated_at, system_info
	FROM agents
	WHERE id = $1
	`

	var agent Agent
	var tags pq.StringArray
	var systemInfoJSON []byte

	err := s.db.QueryRowContext(ctx, query, id).Scan(
		&agent.ID, &agent.UserID, &agent.Name, &agent.Description,
		&agent.OS, &agent.Arch, &agent.Shell, &agent.Distro, &tags,
		&agent.MFALocked, &agent.CreatedAt, &agent.UpdatedAt, &systemInfoJSON,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	if len(systemInfoJSON) > 0 {
		json.Unmarshal(systemInfoJSON, &agent.SystemInfo)
	}

	agent.Tags = tags
	return &agent, nil
}

// SetAgentMFALock sets the MFA lock status for an agent
func (s *PostgresStore) SetAgentMFALock(ctx context.Context, id string, locked bool) error {
	query := `UPDATE agents SET mfa_locked = $2, updated_at = NOW() WHERE id = $1`
	_, err := s.db.ExecContext(ctx, query, id, locked)
	return err
}

// GetAgentsByUser retrieves all agents for a user
func (s *PostgresStore) GetAgentsByUser(ctx context.Context, userID string) ([]*Agent, error) {
	query := `
	SELECT id, user_id, name, COALESCE(description, ''), COALESCE(os, ''), COALESCE(arch, ''),
	       COALESCE(shell, ''), COALESCE(distro, ''), tags, created_at, updated_at, last_heartbeat, COALESCE(connected_instance_id, ''), system_info, COALESCE(mfa_locked, false)
	FROM agents
	WHERE user_id = $1
	ORDER BY created_at DESC
	`

	rows, err := s.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var agents []*Agent
	for rows.Next() {
		var agent Agent
		var tags pq.StringArray
		var lastHeartbeat sql.NullTime
		var connectedInstanceID string
		var systemInfoJSON []byte

		err := rows.Scan(
			&agent.ID, &agent.UserID, &agent.Name, &agent.Description,
			&agent.OS, &agent.Arch, &agent.Shell, &agent.Distro, &tags,
			&agent.CreatedAt, &agent.UpdatedAt, &lastHeartbeat, &connectedInstanceID, &systemInfoJSON,
			&agent.MFALocked,
		)
		if err != nil {
			return nil, err
		}

		if lastHeartbeat.Valid {
			agent.LastPing = lastHeartbeat.Time
		}

		if len(systemInfoJSON) > 0 {
			json.Unmarshal(systemInfoJSON, &agent.SystemInfo)
		}

		agent.Tags = tags
		agents = append(agents, &agent)
	}

	return agents, nil
}

// UpdateAgent updates an agent
func (s *PostgresStore) UpdateAgent(ctx context.Context, id, name, description string, tags []string) error {
	query := `
	UPDATE agents
	SET name = $2, description = $3, tags = $4, updated_at = NOW()
	WHERE id = $1
	`

	_, err := s.db.ExecContext(ctx, query, id, name, description, pq.Array(tags))
	return err
}

// DeleteAgent deletes an agent
func (s *PostgresStore) DeleteAgent(ctx context.Context, id string) error {
	query := `DELETE FROM agents WHERE id = $1`
	_, err := s.db.ExecContext(ctx, query, id)
	return err
}

// UpdateAgentHeartbeat updates the last_heartbeat and connected_instance_id for an agent
func (s *PostgresStore) UpdateAgentHeartbeat(ctx context.Context, id, instanceID string) error {
	query := `
	UPDATE agents
	SET last_heartbeat = NOW(), connected_instance_id = $2
	WHERE id = $1
	`
	_, err := s.db.ExecContext(ctx, query, id, instanceID)
	return err
}

// UpdateAgentSystemInfo updates the system info for an agent
func (s *PostgresStore) UpdateAgentSystemInfo(ctx context.Context, id string, systemInfo map[string]interface{}) error {
	systemInfoJSON, err := json.Marshal(systemInfo)
	if err != nil {
		return err
	}

	query := `
	UPDATE agents
	SET system_info = $2
	WHERE id = $1
	`
	_, err = s.db.ExecContext(ctx, query, id, systemInfoJSON)
	return err
}

// UpdateAgentMetadata updates the agent's metadata (OS, Arch, Shell, Distro)
func (s *PostgresStore) UpdateAgentMetadata(ctx context.Context, id, os, arch, shell, distro string) error {
	query := `
	UPDATE agents
	SET os = $2, arch = $3, shell = $4, distro = $5, updated_at = NOW()
	WHERE id = $1
	`
	_, err := s.db.ExecContext(ctx, query, id, os, arch, shell, distro)
	return err
}

// UpdateAgentStatus updates only the last_heartbeat (for pings)
func (s *PostgresStore) UpdateAgentStatus(ctx context.Context, id string) error {
	query := `UPDATE agents SET last_heartbeat = NOW() WHERE id = $1`
	_, err := s.db.ExecContext(ctx, query, id)
	return err
}

// DisconnectAgent clears the connected_instance_id
func (s *PostgresStore) DisconnectAgent(ctx context.Context, id string) error {
	query := `UPDATE agents SET connected_instance_id = NULL WHERE id = $1`
	_, err := s.db.ExecContext(ctx, query, id)
	return err
}

// GetAgentConnectedInstance returns the instance ID an agent is connected to, if online
func (s *PostgresStore) GetAgentConnectedInstance(ctx context.Context, id string) (string, error) {
	query := `
	SELECT connected_instance_id
	FROM agents
	WHERE id = $1 AND last_heartbeat > NOW() - INTERVAL '2 minutes'
	`
	var instanceID sql.NullString
	err := s.db.QueryRowContext(ctx, query, id).Scan(&instanceID)
	if err == sql.ErrNoRows {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return instanceID.String, nil
}

// GetAllAgents retrieves all agents (admin only)
func (s *PostgresStore) GetAllAgents(ctx context.Context) ([]*Agent, error) {
	query := `
	SELECT a.id, a.user_id, COALESCE(u.username, 'Unknown'), a.name, COALESCE(a.description, ''), COALESCE(a.os, ''), COALESCE(a.arch, ''),
	       COALESCE(a.shell, ''), COALESCE(a.distro, ''), a.tags, a.created_at, a.updated_at, a.last_heartbeat, COALESCE(a.connected_instance_id, ''), a.system_info, COALESCE(a.mfa_locked, false)
	FROM agents a
	LEFT JOIN users u ON a.user_id = u.id
	ORDER BY a.created_at DESC
	`

	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var agents []*Agent
	for rows.Next() {
		var agent Agent
		var tags pq.StringArray
		var lastHeartbeat sql.NullTime
		var connectedInstanceID string
		var systemInfoJSON []byte

		err := rows.Scan(
			&agent.ID, &agent.UserID, &agent.Username, &agent.Name, &agent.Description,
			&agent.OS, &agent.Arch, &agent.Shell, &agent.Distro, &tags,
			&agent.CreatedAt, &agent.UpdatedAt, &lastHeartbeat, &connectedInstanceID, &systemInfoJSON,
			&agent.MFALocked,
		)
		if err != nil {
			return nil, err
		}

		if lastHeartbeat.Valid {
			agent.LastPing = lastHeartbeat.Time
		}

		if len(systemInfoJSON) > 0 {
			json.Unmarshal(systemInfoJSON, &agent.SystemInfo)
		}

		agent.Tags = tags
		agents = append(agents, &agent)
	}

	return agents, nil
}
