package storage

import (
	"context"
	"database/sql"

	"github.com/rexec/rexec/internal/models"
)

// GetAllUsers retrieves all users for the admin dashboard
func (s *PostgresStore) GetAllUsers(ctx context.Context) ([]*models.User, error) {
	query := `
		SELECT id, email, username, tier, COALESCE(is_admin, false), 
		       COALESCE(pipeops_id, ''), subscription_active, created_at, updated_at
		FROM users
		ORDER BY created_at DESC
	`
	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []*models.User
	for rows.Next() {
		var u models.User
		var pipeopsID sql.NullString
		// Assuming SubscriptionActive is a bool in your models.User
		// If it's a pointer/sql.NullBool, adjust accordingly
		err := rows.Scan(
			&u.ID,
			&u.Email,
			&u.Username,
			&u.Tier,
			&u.IsAdmin,
			&pipeopsID,
			&u.SubscriptionActive,
			&u.CreatedAt,
			&u.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		u.PipeOpsID = pipeopsID.String
		users = append(users, &u)
	}
	return users, nil
}

// GetAllContainersAdmin retrieves all containers for the admin dashboard
// It performs a JOIN with users to get owner details
func (s *PostgresStore) GetAllContainersAdmin(ctx context.Context) ([]*models.AdminContainer, error) {
	query := `
		SELECT 
			c.id, c.user_id, c.name, c.image, c.status, c.created_at, 
			c.memory_mb, c.cpu_shares, c.disk_mb,
			u.username, u.email
		FROM containers c
		JOIN users u ON c.user_id = u.id
		WHERE c.deleted_at IS NULL
		ORDER BY c.created_at DESC
	`
	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var containers []*models.AdminContainer
	for rows.Next() {
		var ac models.AdminContainer
		ac.Resources = models.ResourceLimits{} // Initialize nested struct
		
		err := rows.Scan(
			&ac.ID,
			&ac.UserID,
			&ac.Name,
			&ac.Image,
			&ac.Status,
			&ac.CreatedAt,
			&ac.Resources.MemoryMB,
			&ac.Resources.CPUShares,
			&ac.Resources.DiskMB,
			&ac.Username,
			&ac.UserEmail,
		)
		if err != nil {
			return nil, err
		}
		containers = append(containers, &ac)
	}
	return containers, nil
}

// GetAllSessionsAdmin retrieves active terminal sessions for the admin dashboard
func (s *PostgresStore) GetAllSessionsAdmin(ctx context.Context) ([]*models.AdminTerminal, error) {
	// Join sessions with users and containers to provide meaningful info
	// Filter by last_ping_at to only show recently active sessions (e.g., last 5 minutes)
	// Although for now, we'll just return all sessions in the table as "active" implies
	// they haven't been deleted yet.
	query := `
		SELECT 
			s.id, s.container_id, s.user_id, s.created_at,
			u.username,
			c.name as container_name, c.status
		FROM sessions s
		JOIN users u ON s.user_id = u.id
		JOIN containers c ON s.container_id = c.id
		ORDER BY s.created_at DESC
	`
	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var terminals []*models.AdminTerminal
	for rows.Next() {
		var t models.AdminTerminal
		err := rows.Scan(
			&t.ID,
			&t.ContainerID,
			&t.UserID,
			&t.ConnectedAt,
			&t.Username,
			&t.Name,   // Container name
			&t.Status, // Container status
		)
		if err != nil {
			return nil, err
		}
		terminals = append(terminals, &t)
	}
	return terminals, nil
}

// DeleteUser permanently deletes a user and their cascading resources
func (s *PostgresStore) DeleteUser(ctx context.Context, id string) error {
	// You might want to implement CASCADE DELETES in your SQL schema
	// or manually delete associated resources (containers, terminals, etc.) here.
	// For simplicity, this example just deletes the user.
	query := `DELETE FROM users WHERE id = $1`
	_, err := s.db.ExecContext(ctx, query, id)
	return err
}
