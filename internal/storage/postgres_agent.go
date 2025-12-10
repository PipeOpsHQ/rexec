package storage

import (
	"context"
	"database/sql"
	"time"

	"github.com/lib/pq"
)

// Agent represents a registered external agent
type Agent struct {
	ID          string    `json:"id"`
	UserID      string    `json:"user_id"`
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	OS          string    `json:"os"`
	Arch        string    `json:"arch"`
	Shell       string    `json:"shell"`
	Tags        []string  `json:"tags,omitempty"`
	Status      string    `json:"status"`
	ConnectedAt time.Time `json:"connected_at,omitempty"`
	LastPing    time.Time `json:"last_ping,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
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
		tags TEXT[],
		created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
		updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
	);
	
	CREATE INDEX IF NOT EXISTS idx_agents_user_id ON agents(user_id);
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
	       COALESCE(shell, ''), tags, created_at, updated_at
	FROM agents
	WHERE id = $1
	`

	var agent Agent
	var tags pq.StringArray

	err := s.db.QueryRowContext(ctx, query, id).Scan(
		&agent.ID, &agent.UserID, &agent.Name, &agent.Description,
		&agent.OS, &agent.Arch, &agent.Shell, &tags,
		&agent.CreatedAt, &agent.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	agent.Tags = tags
	return &agent, nil
}

// GetAgentsByUser retrieves all agents for a user
func (s *PostgresStore) GetAgentsByUser(ctx context.Context, userID string) ([]*Agent, error) {
	query := `
	SELECT id, user_id, name, COALESCE(description, ''), COALESCE(os, ''), COALESCE(arch, ''), 
	       COALESCE(shell, ''), tags, created_at, updated_at
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

		err := rows.Scan(
			&agent.ID, &agent.UserID, &agent.Name, &agent.Description,
			&agent.OS, &agent.Arch, &agent.Shell, &tags,
			&agent.CreatedAt, &agent.UpdatedAt,
		)
		if err != nil {
			return nil, err
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

// GetAllAgents retrieves all agents (admin only)
func (s *PostgresStore) GetAllAgents(ctx context.Context) ([]*Agent, error) {
	query := `
	SELECT id, user_id, name, COALESCE(description, ''), COALESCE(os, ''), COALESCE(arch, ''), 
	       COALESCE(shell, ''), tags, created_at, updated_at
	FROM agents
	ORDER BY created_at DESC
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

		err := rows.Scan(
			&agent.ID, &agent.UserID, &agent.Name, &agent.Description,
			&agent.OS, &agent.Arch, &agent.Shell, &tags,
			&agent.CreatedAt, &agent.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}

		agent.Tags = tags
		agents = append(agents, &agent)
	}

	return agents, nil
}
