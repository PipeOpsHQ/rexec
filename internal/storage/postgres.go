package storage

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/rexec/rexec/internal/models"
)

// PostgresStore handles database operations
type PostgresStore struct {
	db *sqlx.DB
}

// NewPostgresStore creates a new PostgreSQL store
func NewPostgresStore(databaseURL string) (*PostgresStore, error) {
	db, err := sqlx.Connect("postgres", databaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Set connection pool settings
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	store := &PostgresStore{db: db}

	// Run migrations
	if err := store.migrate(); err != nil {
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	return store, nil
}

// migrate creates the database schema
func (s *PostgresStore) migrate() error {
	// Step 1: Create tables with basic columns
	createTables := `
	CREATE TABLE IF NOT EXISTS users (
		id VARCHAR(36) PRIMARY KEY,
		email VARCHAR(255) UNIQUE NOT NULL,
		username VARCHAR(255) NOT NULL,
		password_hash VARCHAR(255) DEFAULT '',
		tier VARCHAR(50) DEFAULT 'free',
		created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS containers (
		id VARCHAR(64) PRIMARY KEY,
		user_id VARCHAR(36) NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		name VARCHAR(255) NOT NULL,
		image VARCHAR(100) NOT NULL,
		status VARCHAR(50) DEFAULT 'creating',
		docker_id VARCHAR(64),
		volume_name VARCHAR(255),
		created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
		last_used_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS sessions (
		id VARCHAR(36) PRIMARY KEY,
		user_id VARCHAR(36) NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		container_id VARCHAR(64) NOT NULL REFERENCES containers(id) ON DELETE CASCADE,
		created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
		last_ping_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS ssh_keys (
		id VARCHAR(36) PRIMARY KEY,
		user_id VARCHAR(36) NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		name VARCHAR(255) NOT NULL,
		public_key TEXT NOT NULL,
		fingerprint VARCHAR(255) NOT NULL,
		created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
		last_used_at TIMESTAMP WITH TIME ZONE
	);

	CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);
	CREATE INDEX IF NOT EXISTS idx_containers_user_id ON containers(user_id);
	CREATE INDEX IF NOT EXISTS idx_sessions_user_id ON sessions(user_id);
	CREATE INDEX IF NOT EXISTS idx_ssh_keys_user_id ON ssh_keys(user_id);
	CREATE INDEX IF NOT EXISTS idx_ssh_keys_fingerprint ON ssh_keys(fingerprint);
	`

	if _, err := s.db.Exec(createTables); err != nil {
		return err
	}

	// Step 2: Add optional columns if they don't exist
	addColumns := []string{
		`DO $$ BEGIN
			IF NOT EXISTS (SELECT 1 FROM information_schema.columns
				WHERE table_name='users' AND column_name='stripe_customer_id') THEN
				ALTER TABLE users ADD COLUMN stripe_customer_id VARCHAR(255);
			END IF;
		END $$`,
		`DO $$ BEGIN
			IF NOT EXISTS (SELECT 1 FROM information_schema.columns
				WHERE table_name='users' AND column_name='pipeops_id') THEN
				ALTER TABLE users ADD COLUMN pipeops_id VARCHAR(255);
			END IF;
		END $$`,
		`DO $$ BEGIN
			IF NOT EXISTS (SELECT 1 FROM information_schema.columns
				WHERE table_name='containers' AND column_name='deleted_at') THEN
				ALTER TABLE containers ADD COLUMN deleted_at TIMESTAMP WITH TIME ZONE;
			END IF;
		END $$`,
	}

	for _, query := range addColumns {
		if _, err := s.db.Exec(query); err != nil {
			return err
		}
	}

	// Step 3: Create indexes on optional columns (after they exist)
	createIndexes := `
	CREATE INDEX IF NOT EXISTS idx_users_stripe_customer_id ON users(stripe_customer_id);
	CREATE INDEX IF NOT EXISTS idx_users_pipeops_id ON users(pipeops_id);
	`

	_, err := s.db.Exec(createIndexes)
	return err
}

// Close closes the database connection
func (s *PostgresStore) Close() error {
	return s.db.Close()
}

// User operations

// CreateUser creates a new user
func (s *PostgresStore) CreateUser(ctx context.Context, user *models.User, passwordHash string) error {
	query := `
		INSERT INTO users (id, email, username, password_hash, tier, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`
	_, err := s.db.ExecContext(ctx, query,
		user.ID,
		user.Email,
		user.Username,
		passwordHash,
		user.Tier,
		user.CreatedAt,
		user.UpdatedAt,
	)
	return err
}

// GetUserByEmail retrieves a user by email
func (s *PostgresStore) GetUserByEmail(ctx context.Context, email string) (*models.User, string, error) {
	var user models.User
	var passwordHash string
	var pipeopsID sql.NullString

	query := `
		SELECT id, email, username, COALESCE(password_hash, ''), tier,
		       COALESCE(pipeops_id, ''), created_at, updated_at
		FROM users WHERE email = $1
	`
	row := s.db.QueryRowContext(ctx, query, email)
	err := row.Scan(
		&user.ID,
		&user.Email,
		&user.Username,
		&passwordHash,
		&user.Tier,
		&pipeopsID,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, "", nil
	}
	if err != nil {
		return nil, "", err
	}
	user.PipeOpsID = pipeopsID.String
	return &user, passwordHash, nil
}

// GetUserByID retrieves a user by ID
func (s *PostgresStore) GetUserByID(ctx context.Context, id string) (*models.User, error) {
	var user models.User
	var pipeopsID sql.NullString

	query := `
		SELECT id, email, username, tier, COALESCE(pipeops_id, ''), created_at, updated_at
		FROM users WHERE id = $1
	`
	row := s.db.QueryRowContext(ctx, query, id)
	err := row.Scan(
		&user.ID,
		&user.Email,
		&user.Username,
		&user.Tier,
		&pipeopsID,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	user.PipeOpsID = pipeopsID.String
	return &user, nil
}

// UpdateUser updates a user
func (s *PostgresStore) UpdateUser(ctx context.Context, user *models.User) error {
	query := `
		UPDATE users SET username = $2, tier = $3, pipeops_id = $4, updated_at = $5
		WHERE id = $1
	`
	_, err := s.db.ExecContext(ctx, query,
		user.ID,
		user.Username,
		user.Tier,
		user.PipeOpsID,
		time.Now(),
	)
	return err
}

// UpdateUserTier updates a user's subscription tier
func (s *PostgresStore) UpdateUserTier(ctx context.Context, userID, tier string) error {
	query := `UPDATE users SET tier = $2, updated_at = $3 WHERE id = $1`
	_, err := s.db.ExecContext(ctx, query, userID, tier, time.Now())
	return err
}

// GetUserStripeCustomerID retrieves a user's Stripe customer ID
func (s *PostgresStore) GetUserStripeCustomerID(ctx context.Context, userID string) (string, error) {
	var customerID sql.NullString
	query := `SELECT stripe_customer_id FROM users WHERE id = $1`
	err := s.db.QueryRowContext(ctx, query, userID).Scan(&customerID)
	if err != nil {
		return "", err
	}
	return customerID.String, nil
}

// SetUserStripeCustomerID sets a user's Stripe customer ID
func (s *PostgresStore) SetUserStripeCustomerID(ctx context.Context, userID, customerID string) error {
	query := `UPDATE users SET stripe_customer_id = $2, updated_at = $3 WHERE id = $1`
	_, err := s.db.ExecContext(ctx, query, userID, customerID, time.Now())
	return err
}

// GetUserIDByStripeCustomerID retrieves a user ID by Stripe customer ID
func (s *PostgresStore) GetUserIDByStripeCustomerID(ctx context.Context, customerID string) (string, error) {
	var userID string
	query := `SELECT id FROM users WHERE stripe_customer_id = $1`
	err := s.db.QueryRowContext(ctx, query, customerID).Scan(&userID)
	if err == sql.ErrNoRows {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return userID, nil
}

// UserRecord represents a user record from the database
type UserRecord struct {
	ID               string    `db:"id"`
	Email            string    `db:"email"`
	Username         string    `db:"username"`
	Tier             string    `db:"tier"`
	StripeCustomerID string    `db:"stripe_customer_id"`
	CreatedAt        time.Time `db:"created_at"`
	UpdatedAt        time.Time `db:"updated_at"`
}

// SSH Key operations

// SSHKeyRecord represents an SSH key in the database
type SSHKeyRecord struct {
	ID          string     `db:"id"`
	UserID      string     `db:"user_id"`
	Name        string     `db:"name"`
	PublicKey   string     `db:"public_key"`
	Fingerprint string     `db:"fingerprint"`
	CreatedAt   time.Time  `db:"created_at"`
	LastUsedAt  *time.Time `db:"last_used_at"`
}

// CreateSSHKey creates a new SSH key record
func (s *PostgresStore) CreateSSHKey(ctx context.Context, key *SSHKeyRecord) error {
	query := `
		INSERT INTO ssh_keys (id, user_id, name, public_key, fingerprint, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`
	_, err := s.db.ExecContext(ctx, query,
		key.ID,
		key.UserID,
		key.Name,
		key.PublicKey,
		key.Fingerprint,
		key.CreatedAt,
	)
	return err
}

// GetSSHKeysByUserID retrieves all SSH keys for a user
func (s *PostgresStore) GetSSHKeysByUserID(ctx context.Context, userID string) ([]*SSHKeyRecord, error) {
	query := `
		SELECT id, user_id, name, public_key, fingerprint, created_at, last_used_at
		FROM ssh_keys WHERE user_id = $1
		ORDER BY created_at DESC
	`
	rows, err := s.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var keys []*SSHKeyRecord
	for rows.Next() {
		var k SSHKeyRecord
		err := rows.Scan(
			&k.ID,
			&k.UserID,
			&k.Name,
			&k.PublicKey,
			&k.Fingerprint,
			&k.CreatedAt,
			&k.LastUsedAt,
		)
		if err != nil {
			return nil, err
		}
		keys = append(keys, &k)
	}
	return keys, nil
}

// GetSSHKeyByID retrieves an SSH key by ID
func (s *PostgresStore) GetSSHKeyByID(ctx context.Context, id string) (*SSHKeyRecord, error) {
	var k SSHKeyRecord
	query := `
		SELECT id, user_id, name, public_key, fingerprint, created_at, last_used_at
		FROM ssh_keys WHERE id = $1
	`
	row := s.db.QueryRowContext(ctx, query, id)
	err := row.Scan(
		&k.ID,
		&k.UserID,
		&k.Name,
		&k.PublicKey,
		&k.Fingerprint,
		&k.CreatedAt,
		&k.LastUsedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &k, nil
}

// GetSSHKeyByFingerprint retrieves an SSH key by fingerprint
func (s *PostgresStore) GetSSHKeyByFingerprint(ctx context.Context, fingerprint string) (*SSHKeyRecord, error) {
	var k SSHKeyRecord
	query := `
		SELECT id, user_id, name, public_key, fingerprint, created_at, last_used_at
		FROM ssh_keys WHERE fingerprint = $1
	`
	row := s.db.QueryRowContext(ctx, query, fingerprint)
	err := row.Scan(
		&k.ID,
		&k.UserID,
		&k.Name,
		&k.PublicKey,
		&k.Fingerprint,
		&k.CreatedAt,
		&k.LastUsedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &k, nil
}

// DeleteSSHKey deletes an SSH key by ID
func (s *PostgresStore) DeleteSSHKey(ctx context.Context, id string) error {
	query := `DELETE FROM ssh_keys WHERE id = $1`
	_, err := s.db.ExecContext(ctx, query, id)
	return err
}

// TouchSSHKey updates the last_used_at timestamp
func (s *PostgresStore) TouchSSHKey(ctx context.Context, id string) error {
	query := `UPDATE ssh_keys SET last_used_at = $2 WHERE id = $1`
	_, err := s.db.ExecContext(ctx, query, id, time.Now())
	return err
}

// GetAllUserSSHPublicKeys returns all public keys for a user as a single authorized_keys string
func (s *PostgresStore) GetAllUserSSHPublicKeys(ctx context.Context, userID string) (string, error) {
	keys, err := s.GetSSHKeysByUserID(ctx, userID)
	if err != nil {
		return "", err
	}

	var result string
	for _, key := range keys {
		result += key.PublicKey + "\n"
	}
	return result, nil
}

// Container operations

// ContainerRecord represents a container in the database
type ContainerRecord struct {
	ID         string    `db:"id"`
	UserID     string    `db:"user_id"`
	Name       string    `db:"name"`
	Image      string    `db:"image"`
	Status     string    `db:"status"`
	DockerID   string    `db:"docker_id"`
	VolumeName string    `db:"volume_name"`
	CreatedAt  time.Time `db:"created_at"`
	LastUsedAt time.Time `db:"last_used_at"`
}

// CreateContainer creates a container record
func (s *PostgresStore) CreateContainer(ctx context.Context, container *ContainerRecord) error {
	query := `
		INSERT INTO containers (id, user_id, name, image, status, docker_id, volume_name, created_at, last_used_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`
	_, err := s.db.ExecContext(ctx, query,
		container.ID,
		container.UserID,
		container.Name,
		container.Image,
		container.Status,
		container.DockerID,
		container.VolumeName,
		container.CreatedAt,
		container.LastUsedAt,
	)
	return err
}

// GetContainersByUserID retrieves all non-deleted containers for a user
func (s *PostgresStore) GetContainersByUserID(ctx context.Context, userID string) ([]*ContainerRecord, error) {
	query := `
		SELECT id, user_id, name, image, status, docker_id, volume_name, created_at, last_used_at
		FROM containers WHERE user_id = $1 AND deleted_at IS NULL
		ORDER BY created_at DESC
	`
	rows, err := s.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var containers []*ContainerRecord
	for rows.Next() {
		var c ContainerRecord
		err := rows.Scan(
			&c.ID,
			&c.UserID,
			&c.Name,
			&c.Image,
			&c.Status,
			&c.DockerID,
			&c.VolumeName,
			&c.CreatedAt,
			&c.LastUsedAt,
		)
		if err != nil {
			return nil, err
		}
		containers = append(containers, &c)
	}
	return containers, nil
}

// GetContainerByID retrieves a non-deleted container by ID
func (s *PostgresStore) GetContainerByID(ctx context.Context, id string) (*ContainerRecord, error) {
	var c ContainerRecord
	query := `
		SELECT id, user_id, name, image, status, docker_id, volume_name, created_at, last_used_at
		FROM containers WHERE id = $1 AND deleted_at IS NULL
	`
	row := s.db.QueryRowContext(ctx, query, id)
	err := row.Scan(
		&c.ID,
		&c.UserID,
		&c.Name,
		&c.Image,
		&c.Status,
		&c.DockerID,
		&c.VolumeName,
		&c.CreatedAt,
		&c.LastUsedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &c, nil
}

// GetContainerByUserAndDockerID retrieves a non-deleted container by user ID and docker ID
func (s *PostgresStore) GetContainerByUserAndDockerID(ctx context.Context, userID, dockerID string) (*ContainerRecord, error) {
	var c ContainerRecord
	query := `
		SELECT id, user_id, name, image, status, docker_id, volume_name, created_at, last_used_at
		FROM containers WHERE user_id = $1 AND docker_id = $2 AND deleted_at IS NULL
	`
	row := s.db.QueryRowContext(ctx, query, userID, dockerID)
	err := row.Scan(
		&c.ID,
		&c.UserID,
		&c.Name,
		&c.Image,
		&c.Status,
		&c.DockerID,
		&c.VolumeName,
		&c.CreatedAt,
		&c.LastUsedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &c, nil
}

// UpdateContainerStatus updates a container's status (only for non-deleted containers)
func (s *PostgresStore) UpdateContainerStatus(ctx context.Context, id, status string) error {
	query := `UPDATE containers SET status = $2, last_used_at = $3 WHERE id = $1 AND deleted_at IS NULL`
	_, err := s.db.ExecContext(ctx, query, id, status, time.Now())
	return err
}

// UpdateContainerDockerID updates a container's Docker ID (only for non-deleted containers)
func (s *PostgresStore) UpdateContainerDockerID(ctx context.Context, id, dockerID string) error {
	query := `UPDATE containers SET docker_id = $2 WHERE id = $1 AND deleted_at IS NULL`
	_, err := s.db.ExecContext(ctx, query, id, dockerID)
	return err
}

// UpdateContainerError updates a container's error message (stored in status field with error: prefix)
func (s *PostgresStore) UpdateContainerError(ctx context.Context, id, errorMsg string) error {
	// Store error in a way that can be retrieved - we'll use status field with error prefix
	// In a production system, you'd add an error_message column
	query := `UPDATE containers SET status = $2, last_used_at = $3 WHERE id = $1`
	_, err := s.db.ExecContext(ctx, query, id, "error", time.Now())
	if err != nil {
		return err
	}
	// Log the actual error for debugging
	log.Printf("Container %s creation failed: %s", id, errorMsg)
	return nil
}

// DeleteContainer soft deletes a container record by setting deleted_at
func (s *PostgresStore) DeleteContainer(ctx context.Context, id string) error {
	query := `UPDATE containers SET deleted_at = CURRENT_TIMESTAMP, status = 'stopped' WHERE id = $1`
	_, err := s.db.ExecContext(ctx, query, id)
	return err
}

// UpdateContainerSettings updates a container's name and resource settings
func (s *PostgresStore) UpdateContainerSettings(ctx context.Context, id, name string, memoryMB, cpuShares, diskMB int64) error {
	query := `UPDATE containers SET name = $2, memory_mb = $3, cpu_shares = $4, disk_mb = $5 WHERE id = $1 AND deleted_at IS NULL`
	_, err := s.db.ExecContext(ctx, query, id, name, memoryMB, cpuShares, diskMB)
	return err
}

// HardDeleteContainer permanently deletes a container record (for cleanup)
func (s *PostgresStore) HardDeleteContainer(ctx context.Context, id string) error {
	query := `DELETE FROM containers WHERE id = $1`
	_, err := s.db.ExecContext(ctx, query, id)
	return err
}

// RestoreContainer restores a soft-deleted container
func (s *PostgresStore) RestoreContainer(ctx context.Context, id string) error {
	query := `UPDATE containers SET deleted_at = NULL WHERE id = $1`
	_, err := s.db.ExecContext(ctx, query, id)
	return err
}

// GetAllContainers retrieves all non-deleted containers (for loading on startup)
func (s *PostgresStore) GetAllContainers(ctx context.Context) ([]*ContainerRecord, error) {
	query := `
		SELECT id, user_id, name, image, status, docker_id, volume_name, created_at, last_used_at
		FROM containers WHERE deleted_at IS NULL
	`
	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var containers []*ContainerRecord
	for rows.Next() {
		var c ContainerRecord
		err := rows.Scan(
			&c.ID,
			&c.UserID,
			&c.Name,
			&c.Image,
			&c.Status,
			&c.DockerID,
			&c.VolumeName,
			&c.CreatedAt,
			&c.LastUsedAt,
		)
		if err != nil {
			return nil, err
		}
		containers = append(containers, &c)
	}
	return containers, nil
}

// TouchContainer updates the last_used_at timestamp (only for non-deleted containers)
func (s *PostgresStore) TouchContainer(ctx context.Context, id string) error {
	query := `UPDATE containers SET last_used_at = $2 WHERE id = $1 AND deleted_at IS NULL`
	_, err := s.db.ExecContext(ctx, query, id, time.Now())
	return err
}
