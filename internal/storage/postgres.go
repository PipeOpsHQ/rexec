package storage

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/rexec/rexec/internal/crypto"
	"github.com/rexec/rexec/internal/models"
)

// PostgresStore handles database operations
type PostgresStore struct {
	db        *sqlx.DB
	encryptor *crypto.Encryptor
}

// NewPostgresStore creates a new PostgreSQL store
func NewPostgresStore(databaseURL string, encryptor *crypto.Encryptor) (*PostgresStore, error) {
	db, err := sqlx.Connect("postgres", databaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Set connection pool settings
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	store := &PostgresStore{
		db:        db,
		encryptor: encryptor,
	}

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
			is_admin BOOLEAN DEFAULT false,
			mfa_enabled BOOLEAN DEFAULT false,
			mfa_secret VARCHAR(255) DEFAULT '',
			screen_lock_hash VARCHAR(255) DEFAULT '',
			screen_lock_enabled BOOLEAN DEFAULT false,
			lock_after_minutes INTEGER DEFAULT 5,
			lock_required_since TIMESTAMP WITH TIME ZONE,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
		);

		CREATE TABLE IF NOT EXISTS user_sessions (
			id VARCHAR(36) PRIMARY KEY,
			user_id VARCHAR(36) NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			ip_address VARCHAR(45),
			user_agent TEXT,
			token_issued_at TIMESTAMP WITH TIME ZONE,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
			last_seen_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
			revoked_at TIMESTAMP WITH TIME ZONE,
			revoked_reason TEXT
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
		container_id VARCHAR(64) NOT NULL,
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
	CREATE INDEX IF NOT EXISTS idx_user_sessions_user_id ON user_sessions(user_id);
	CREATE INDEX IF NOT EXISTS idx_user_sessions_revoked_at ON user_sessions(revoked_at);
	CREATE INDEX IF NOT EXISTS idx_containers_user_id ON containers(user_id);
	CREATE INDEX IF NOT EXISTS idx_sessions_user_id ON sessions(user_id);
	CREATE INDEX IF NOT EXISTS idx_ssh_keys_user_id ON ssh_keys(user_id);
	CREATE INDEX IF NOT EXISTS idx_ssh_keys_fingerprint ON ssh_keys(fingerprint);

	CREATE TABLE IF NOT EXISTS remote_hosts (
		id VARCHAR(36) PRIMARY KEY,
		user_id VARCHAR(36) NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		name VARCHAR(255) NOT NULL,
		hostname VARCHAR(255) NOT NULL,
		port INTEGER DEFAULT 22,
		username VARCHAR(255) NOT NULL,
		identity_file VARCHAR(255),
		created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
	);

	CREATE INDEX IF NOT EXISTS idx_remote_hosts_user_id ON remote_hosts(user_id);

	CREATE TABLE IF NOT EXISTS port_forwards (
		id VARCHAR(36) PRIMARY KEY,
		user_id VARCHAR(36) NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		container_id VARCHAR(64) NOT NULL REFERENCES containers(id) ON DELETE CASCADE,
		name VARCHAR(255) NOT NULL,
		container_port INTEGER NOT NULL,
		local_port INTEGER NOT NULL,
		protocol VARCHAR(10) DEFAULT 'tcp',
		is_active BOOLEAN DEFAULT true,
		created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
		UNIQUE (user_id, container_id, container_port, local_port)
	);

	CREATE INDEX IF NOT EXISTS idx_port_forwards_user_id ON port_forwards(user_id);
	CREATE INDEX IF NOT EXISTS idx_port_forwards_container_id ON port_forwards(container_id);
	CREATE INDEX IF NOT EXISTS idx_port_forwards_unique ON port_forwards(user_id, container_id, container_port, local_port);

	CREATE TABLE IF NOT EXISTS snippets (
		id VARCHAR(36) PRIMARY KEY,
		user_id VARCHAR(36) NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		name VARCHAR(255) NOT NULL,
		content TEXT NOT NULL,
		language VARCHAR(50) DEFAULT 'bash',
		created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
	);

	CREATE INDEX IF NOT EXISTS idx_snippets_user_id ON snippets(user_id);

	CREATE TABLE IF NOT EXISTS audit_logs (
		id VARCHAR(36) PRIMARY KEY,
		user_id VARCHAR(36) NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		action VARCHAR(255) NOT NULL,
		ip_address VARCHAR(45),
		user_agent TEXT,
		details TEXT,
		created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
	);

	CREATE INDEX IF NOT EXISTS idx_audit_logs_user_id ON audit_logs(user_id);
	`

	if _, err := s.db.Exec(createTables); err != nil {
		return err
	}

	// Step 2: Add optional columns if they don't exist
	addColumns := []string{
		`DO $$ BEGIN
				IF NOT EXISTS (SELECT 1 FROM information_schema.columns
					WHERE table_name='users' AND column_name='mfa_enabled') THEN
					ALTER TABLE users ADD COLUMN mfa_enabled BOOLEAN DEFAULT false;
				END IF;
			END $$`,
		`DO $$ BEGIN
				IF NOT EXISTS (SELECT 1 FROM information_schema.columns
					WHERE table_name='users' AND column_name='mfa_secret') THEN
					ALTER TABLE users ADD COLUMN mfa_secret VARCHAR(255) DEFAULT '';
				END IF;
			END $$`,
		`DO $$ BEGIN
				IF NOT EXISTS (SELECT 1 FROM information_schema.columns
					WHERE table_name='users' AND column_name='mfa_backup_codes') THEN
					ALTER TABLE users ADD COLUMN mfa_backup_codes TEXT DEFAULT '';
				END IF;
			END $$`,
		`DO $$ BEGIN
				IF NOT EXISTS (SELECT 1 FROM information_schema.columns
					WHERE table_name='users' AND column_name='screen_lock_hash') THEN
					ALTER TABLE users ADD COLUMN screen_lock_hash VARCHAR(255) DEFAULT '';
				END IF;
			END $$`,
		`DO $$ BEGIN
				IF NOT EXISTS (SELECT 1 FROM information_schema.columns
					WHERE table_name='users' AND column_name='screen_lock_enabled') THEN
					ALTER TABLE users ADD COLUMN screen_lock_enabled BOOLEAN DEFAULT false;
				END IF;
			END $$`,
		`DO $$ BEGIN
				IF NOT EXISTS (SELECT 1 FROM information_schema.columns
					WHERE table_name='users' AND column_name='lock_after_minutes') THEN
					ALTER TABLE users ADD COLUMN lock_after_minutes INTEGER DEFAULT 5;
				END IF;
			END $$`,
		`DO $$ BEGIN
				IF NOT EXISTS (SELECT 1 FROM information_schema.columns
					WHERE table_name='users' AND column_name='lock_required_since') THEN
					ALTER TABLE users ADD COLUMN lock_required_since TIMESTAMP WITH TIME ZONE;
				END IF;
			END $$`,
		`DO $$ BEGIN
				IF NOT EXISTS (SELECT 1 FROM information_schema.columns
					WHERE table_name='users' AND column_name='allowed_ips') THEN
					ALTER TABLE users ADD COLUMN allowed_ips TEXT;
				END IF;
		END $$`,
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
				WHERE table_name='users' AND column_name='first_name') THEN
				ALTER TABLE users ADD COLUMN first_name VARCHAR(255) DEFAULT '';
			END IF;
		END $$`,
		`DO $$ BEGIN
			IF NOT EXISTS (SELECT 1 FROM information_schema.columns
				WHERE table_name='users' AND column_name='last_name') THEN
				ALTER TABLE users ADD COLUMN last_name VARCHAR(255) DEFAULT '';
			END IF;
		END $$`,
		`DO $$ BEGIN
			IF NOT EXISTS (SELECT 1 FROM information_schema.columns
				WHERE table_name='containers' AND column_name='deleted_at') THEN
				ALTER TABLE containers ADD COLUMN deleted_at TIMESTAMP WITH TIME ZONE;
			END IF;
		END $$`,
		`DO $$ BEGIN
			IF NOT EXISTS (SELECT 1 FROM information_schema.columns
				WHERE table_name='containers' AND column_name='memory_mb') THEN
				ALTER TABLE containers ADD COLUMN memory_mb INTEGER DEFAULT 512;
			END IF;
		END $$`,
		`DO $$ BEGIN
			IF NOT EXISTS (SELECT 1 FROM information_schema.columns
				WHERE table_name='containers' AND column_name='cpu_shares') THEN
				ALTER TABLE containers ADD COLUMN cpu_shares INTEGER DEFAULT 512;
			END IF;
		END $$`,
		`DO $$ BEGIN
			IF NOT EXISTS (SELECT 1 FROM information_schema.columns
				WHERE table_name='containers' AND column_name='disk_mb') THEN
				ALTER TABLE containers ADD COLUMN disk_mb INTEGER DEFAULT 2048;
			END IF;
		END $$`,
		`DO $$ BEGIN
			IF NOT EXISTS (SELECT 1 FROM information_schema.columns
				WHERE table_name='containers' AND column_name='role') THEN
				ALTER TABLE containers ADD COLUMN role VARCHAR(50) DEFAULT 'standard';
			END IF;
		END $$`,
		`DO $$ BEGIN
			IF NOT EXISTS (SELECT 1 FROM information_schema.columns
				WHERE table_name='containers' AND column_name='shell_path') THEN
				ALTER TABLE containers ADD COLUMN shell_path VARCHAR(255) DEFAULT '';
			END IF;
		END $$`,
		`DO $$ BEGIN
			IF NOT EXISTS (SELECT 1 FROM information_schema.columns
				WHERE table_name='containers' AND column_name='has_tmux') THEN
				ALTER TABLE containers ADD COLUMN has_tmux BOOLEAN DEFAULT false;
			END IF;
		END $$`,
		`DO $$ BEGIN
			IF NOT EXISTS (SELECT 1 FROM information_schema.columns
				WHERE table_name='containers' AND column_name='shell_setup_done') THEN
				ALTER TABLE containers ADD COLUMN shell_setup_done BOOLEAN DEFAULT false;
			END IF;
		END $$`,
		`DO $$ BEGIN
			IF NOT EXISTS (SELECT 1 FROM information_schema.columns
				WHERE table_name='users' AND column_name='is_admin') THEN
				ALTER TABLE users ADD COLUMN is_admin BOOLEAN DEFAULT false;
			END IF;
		END $$`,
		`DO $$ BEGIN
			IF NOT EXISTS (SELECT 1 FROM information_schema.columns
				WHERE table_name='users' AND column_name='subscription_active') THEN
				ALTER TABLE users ADD COLUMN subscription_active BOOLEAN DEFAULT false;
			END IF;
		END $$`,
		`DO $$ BEGIN
			IF NOT EXISTS (SELECT 1 FROM information_schema.columns
				WHERE table_name='users' AND column_name='single_session_mode') THEN
				ALTER TABLE users ADD COLUMN single_session_mode BOOLEAN DEFAULT false;
			END IF;
		END $$`,
		`DO $$ BEGIN
			ALTER TABLE audit_logs ALTER COLUMN user_id DROP NOT NULL;
		EXCEPTION WHEN OTHERS THEN NULL;
		END $$`,
		`DO $$ BEGIN
			IF NOT EXISTS (SELECT 1 FROM information_schema.columns
				WHERE table_name='containers' AND column_name='mfa_locked') THEN
				ALTER TABLE containers ADD COLUMN mfa_locked BOOLEAN DEFAULT false;
			END IF;
		END $$`,
		`DO $$ BEGIN
			IF NOT EXISTS (SELECT 1 FROM information_schema.columns
				WHERE table_name='agents' AND column_name='mfa_locked') THEN
				ALTER TABLE agents ADD COLUMN mfa_locked BOOLEAN DEFAULT false;
			END IF;
		END $$`,
		`DO $$ BEGIN
			IF NOT EXISTS (SELECT 1 FROM information_schema.columns
				WHERE table_name='users' AND column_name='session_duration_minutes') THEN
				ALTER TABLE users ADD COLUMN session_duration_minutes INTEGER DEFAULT 0;
			END IF;
		END $$`,
	}

	for _, query := range addColumns {
		if _, err := s.db.Exec(query); err != nil {
			return err
		}
	}

	// Add snippet marketplace columns
	snippetColumns := []string{
		`DO $$ BEGIN
			IF NOT EXISTS (SELECT 1 FROM information_schema.columns
				WHERE table_name='snippets' AND column_name='is_public') THEN
				ALTER TABLE snippets ADD COLUMN is_public BOOLEAN DEFAULT false;
			END IF;
		END $$`,
		`DO $$ BEGIN
			IF NOT EXISTS (SELECT 1 FROM information_schema.columns
				WHERE table_name='snippets' AND column_name='description') THEN
				ALTER TABLE snippets ADD COLUMN description TEXT;
			END IF;
		END $$`,
		`DO $$ BEGIN
			IF NOT EXISTS (SELECT 1 FROM information_schema.columns
				WHERE table_name='snippets' AND column_name='usage_count') THEN
				ALTER TABLE snippets ADD COLUMN usage_count INTEGER DEFAULT 0;
			END IF;
		END $$`,
		`DO $$ BEGIN
			IF NOT EXISTS (SELECT 1 FROM information_schema.columns
				WHERE table_name='snippets' AND column_name='icon') THEN
				ALTER TABLE snippets ADD COLUMN icon VARCHAR(20);
			END IF;
		END $$`,
		`DO $$ BEGIN
			IF NOT EXISTS (SELECT 1 FROM information_schema.columns
				WHERE table_name='snippets' AND column_name='category') THEN
				ALTER TABLE snippets ADD COLUMN category VARCHAR(50);
			END IF;
		END $$`,
		`DO $$ BEGIN
			IF NOT EXISTS (SELECT 1 FROM information_schema.columns
				WHERE table_name='snippets' AND column_name='install_command') THEN
				ALTER TABLE snippets ADD COLUMN install_command TEXT;
			END IF;
		END $$`,
		`DO $$ BEGIN
			IF NOT EXISTS (SELECT 1 FROM information_schema.columns
				WHERE table_name='snippets' AND column_name='requires_install') THEN
				ALTER TABLE snippets ADD COLUMN requires_install BOOLEAN DEFAULT false;
			END IF;
		END $$`,
	}

	for _, query := range snippetColumns {
		if _, err := s.db.Exec(query); err != nil {
			return err
		}
	}

	// Create index for public snippets marketplace
	if _, err := s.db.Exec(`CREATE INDEX IF NOT EXISTS idx_snippets_public ON snippets(is_public) WHERE is_public = true`); err != nil {
		log.Printf("Warning: could not create public snippets index: %v", err)
	}

	// Step 3: Create indexes on optional columns (after they exist)
	createIndexes := `
	CREATE INDEX IF NOT EXISTS idx_users_stripe_customer_id ON users(stripe_customer_id);
	CREATE INDEX IF NOT EXISTS idx_users_pipeops_id ON users(pipeops_id);
	`

	if _, err := s.db.Exec(createIndexes); err != nil {
		return err
	}

	// Step 4: Drop sessions FK constraint if exists (sessions are ephemeral, no strict FK needed)
	dropSessionsFk := `
	DO $$ BEGIN
		ALTER TABLE sessions DROP CONSTRAINT IF EXISTS sessions_container_id_fkey;
	EXCEPTION WHEN undefined_object THEN NULL;
	END $$;
	`
	if _, err := s.db.Exec(dropSessionsFk); err != nil {
		// Non-fatal: constraint may not exist
		log.Printf("Warning: could not drop sessions FK (may not exist): %v", err)
	}

	// Step 5: Create collaboration and recording tables
	collabTables := `
	CREATE TABLE IF NOT EXISTS terminal_recordings (
		id VARCHAR(36) PRIMARY KEY,
		user_id VARCHAR(36) NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		container_id VARCHAR(64) NOT NULL,
		title VARCHAR(255) NOT NULL,
		duration_ms BIGINT DEFAULT 0,
		size_bytes BIGINT DEFAULT 0,
		data BYTEA,
		share_token VARCHAR(64) UNIQUE,
		is_public BOOLEAN DEFAULT false,
		created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
		expires_at TIMESTAMP WITH TIME ZONE,
		storage_type VARCHAR(20) DEFAULT 'database',
		storage_url TEXT
	);

	-- Add data column if missing (for existing installations)
	DO $$ BEGIN
		ALTER TABLE terminal_recordings ADD COLUMN IF NOT EXISTS data BYTEA;
	EXCEPTION WHEN duplicate_column THEN NULL;
	END $$;

	-- Add storage columns for R2/CDN support (for existing installations)
	DO $$ BEGIN
		ALTER TABLE terminal_recordings ADD COLUMN IF NOT EXISTS storage_type VARCHAR(20) DEFAULT 'database';
	EXCEPTION WHEN duplicate_column THEN NULL;
	END $$;

	DO $$ BEGIN
		ALTER TABLE terminal_recordings ADD COLUMN IF NOT EXISTS storage_url TEXT;
	EXCEPTION WHEN duplicate_column THEN NULL;
	END $$;

	CREATE TABLE IF NOT EXISTS collab_sessions (
		id VARCHAR(36) PRIMARY KEY,
		container_id VARCHAR(64) NOT NULL,
		owner_id VARCHAR(36) NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		share_code VARCHAR(10) UNIQUE NOT NULL,
		mode VARCHAR(20) DEFAULT 'view',
		max_users INTEGER DEFAULT 5,
		is_active BOOLEAN DEFAULT true,
		created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
		expires_at TIMESTAMP WITH TIME ZONE NOT NULL
	);

	CREATE TABLE IF NOT EXISTS collab_participants (
		id VARCHAR(36) PRIMARY KEY,
		session_id VARCHAR(36) NOT NULL REFERENCES collab_sessions(id) ON DELETE CASCADE,
		user_id VARCHAR(36) NOT NULL,
		username VARCHAR(255) NOT NULL,
		role VARCHAR(20) DEFAULT 'viewer',
		joined_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
		left_at TIMESTAMP WITH TIME ZONE
	);

	CREATE INDEX IF NOT EXISTS idx_recordings_user_id ON terminal_recordings(user_id);
	CREATE INDEX IF NOT EXISTS idx_recordings_share_token ON terminal_recordings(share_token);
	CREATE INDEX IF NOT EXISTS idx_collab_sessions_share_code ON collab_sessions(share_code);
	CREATE INDEX IF NOT EXISTS idx_collab_sessions_container ON collab_sessions(container_id);
	CREATE INDEX IF NOT EXISTS idx_collab_participants_session ON collab_participants(session_id);

	-- Deduplicate collab_participants before creating unique index (for existing installations with duplicates)
	DO $$ 
	BEGIN
		-- Only run deduplication if the unique index doesn't already exist
		IF NOT EXISTS (
			SELECT 1 FROM pg_indexes 
			WHERE indexname = 'idx_collab_participants_session_user'
		) THEN
			-- Delete duplicate entries, keeping only the most recent one per (session_id, user_id)
			DELETE FROM collab_participants a
			USING collab_participants b
			WHERE a.session_id = b.session_id 
			  AND a.user_id = b.user_id 
			  AND a.joined_at < b.joined_at;
		END IF;
	END $$;

	CREATE UNIQUE INDEX IF NOT EXISTS idx_collab_participants_session_user ON collab_participants(session_id, user_id);

	-- Agents table for BYOS (Bring Your Own Server)
	CREATE TABLE IF NOT EXISTS agents (
		id VARCHAR(36) PRIMARY KEY,
		user_id VARCHAR(36) NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		name VARCHAR(255) NOT NULL,
		description TEXT,
		os VARCHAR(50),
		arch VARCHAR(50),
		shell VARCHAR(255),
		distro VARCHAR(100),
		tags TEXT[],
		created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
		last_heartbeat TIMESTAMP WITH TIME ZONE,
		connected_instance_id VARCHAR(255)
	);

	CREATE INDEX IF NOT EXISTS idx_agents_user_id ON agents(user_id);

	-- API tokens table for CLI/API authentication
	CREATE TABLE IF NOT EXISTS api_tokens (
		id VARCHAR(36) PRIMARY KEY,
		user_id VARCHAR(36) NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		name VARCHAR(255) NOT NULL,
		token_hash VARCHAR(255) NOT NULL,
		token_prefix VARCHAR(12) NOT NULL,
		scopes TEXT[] DEFAULT ARRAY['read', 'write'],
		last_used_at TIMESTAMP WITH TIME ZONE,
		expires_at TIMESTAMP WITH TIME ZONE,
		created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
		revoked_at TIMESTAMP WITH TIME ZONE
	);

	CREATE INDEX IF NOT EXISTS idx_api_tokens_user_id ON api_tokens(user_id);
	CREATE INDEX IF NOT EXISTS idx_api_tokens_token_hash ON api_tokens(token_hash);

	-- Tutorials table for admin-created video tutorials
	CREATE TABLE IF NOT EXISTS tutorials (
		id VARCHAR(36) PRIMARY KEY,
		title VARCHAR(255) NOT NULL,
		description TEXT,
		type VARCHAR(20) DEFAULT 'video',
		content TEXT,
		video_url VARCHAR(512),
		thumbnail VARCHAR(512),
		duration VARCHAR(20),
		category VARCHAR(100) DEFAULT 'getting-started',
		display_order INT DEFAULT 0,
		is_published BOOLEAN DEFAULT false,
		created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
	);

	CREATE INDEX IF NOT EXISTS idx_tutorials_category ON tutorials(category);
	CREATE INDEX IF NOT EXISTS idx_tutorials_published ON tutorials(is_published);

	-- Add new columns if missing (for existing installations)
	DO $$ BEGIN
		IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='agents' AND column_name='last_heartbeat') THEN
			ALTER TABLE agents ADD COLUMN last_heartbeat TIMESTAMP WITH TIME ZONE;
		END IF;
		IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='agents' AND column_name='connected_instance_id') THEN
			ALTER TABLE agents ADD COLUMN connected_instance_id VARCHAR(255);
		END IF;
		IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='agents' AND column_name='system_info') THEN
			ALTER TABLE agents ADD COLUMN system_info JSONB;
		END IF;
		IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='agents' AND column_name='distro') THEN
			ALTER TABLE agents ADD COLUMN distro VARCHAR(100);
		END IF;
		IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='tutorials' AND column_name='type') THEN
			ALTER TABLE tutorials ADD COLUMN type VARCHAR(20) DEFAULT 'video';
		END IF;
		IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='tutorials' AND column_name='content') THEN
			ALTER TABLE tutorials ADD COLUMN content TEXT;
		END IF;
		IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='containers' AND column_name='provider') THEN
			ALTER TABLE containers ADD COLUMN provider VARCHAR(50) DEFAULT 'docker';
		END IF;
		IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='containers' AND column_name='vm_id') THEN
			ALTER TABLE containers ADD COLUMN vm_id VARCHAR(255);
		END IF;
		IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='containers' AND column_name='provider_config') THEN
			ALTER TABLE containers ADD COLUMN provider_config JSONB;
		END IF;
		ALTER TABLE tutorials ALTER COLUMN video_url DROP NOT NULL;
	END $$;
	`

	_, err := s.db.Exec(collabTables)
	if err != nil {
		return err
	}

	// Seed example snippets for marketplace
	return s.seedExampleSnippets()
}

// seedExampleSnippets creates a system user and populates the marketplace with example snippets
func (s *PostgresStore) seedExampleSnippets() error {
	ctx := context.Background()

	// Check if system user already exists
	systemUserID := "00000000-0000-0000-0000-000000000000"
	var exists bool
	err := s.db.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM users WHERE id = $1)", systemUserID).Scan(&exists)
	if err != nil {
		return fmt.Errorf("failed to check system user: %w", err)
	}

	if !exists {
		// Create system user for example snippets
		_, err = s.db.ExecContext(ctx, `
			INSERT INTO users (id, email, username, password_hash, tier, is_admin, created_at, updated_at)
			VALUES ($1, 'system@rexec.io', 'rexec', '', 'enterprise', false, NOW(), NOW())
		`, systemUserID)
		if err != nil {
			return fmt.Errorf("failed to create system user: %w", err)
		}
	}

	// Check if we already have seeded snippets
	var snippetCount int
	err = s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM snippets WHERE user_id = $1", systemUserID).Scan(&snippetCount)
	if err != nil {
		return fmt.Errorf("failed to count snippets: %w", err)
	}

	// Check if we can decrypt existing snippets (in case encryption key changed)
	if snippetCount > 0 {
		var sampleContent string
		err := s.db.QueryRowContext(ctx, "SELECT content FROM snippets WHERE user_id = $1 LIMIT 1", systemUserID).Scan(&sampleContent)
		if err == nil {
			_, decryptErr := s.encryptor.Decrypt(sampleContent)
			if decryptErr != nil {
				log.Printf("Warning: Failed to decrypt existing example snippets (key changed?): %v. Re-seeding...", decryptErr)
				// Delete all system snippets to re-seed with new key
				if _, delErr := s.db.ExecContext(ctx, "DELETE FROM snippets WHERE user_id = $1", systemUserID); delErr != nil {
					return fmt.Errorf("failed to clean up stale example snippets: %w", delErr)
				}
				snippetCount = 0
			}
		}
	}

	if snippetCount >= 100 {
		// Already seeded
		return nil
	}

	// Example snippets organized by role
	snippets := []struct {
		ID              string
		Name            string
		Content         string
		Language        string
		Description     string
		Icon            string
		Category        string
		InstallCommand  string
		RequiresInstall bool
	}{
		// System snippets
		{"seed-001", "System Info", "#!/bin/bash\necho '=== System Info ==='\nuname -a\necho ''\necho '=== Memory ==='\nfree -h\necho ''\necho '=== Disk ==='\ndf -h /\necho ''\necho '=== CPU ==='\nlscpu | grep -E '^(Model name|CPU(s)|Thread)'\necho ''\nneofetch 2>/dev/null || echo 'neofetch not installed'", "bash", "Display comprehensive system information including OS, memory, disk, and CPU details", "terminal", "system", "", false},
		{"seed-002", "Find Large Files", "#!/bin/bash\necho 'Finding files larger than 100MB...'\nfind . -type f -size +100M -exec ls -lh {} \\; 2>/dev/null | awk '{print $5, $9}' | sort -rh | head -20", "bash", "Find and list the 20 largest files over 100MB in current directory", "folder", "system", "", false},
		{"seed-003", "Process Monitor", "#!/bin/bash\necho 'Top 10 processes by CPU:'\nps aux --sort=-%cpu | head -11\necho ''\necho 'Top 10 processes by Memory:'\nps aux --sort=-%mem | head -11", "bash", "Show top 10 processes by CPU and memory usage", "chart", "system", "", false},
		{"seed-004", "Quick Backup", "#!/bin/bash\nDIR=${1:-.}\nBACKUP_NAME=\"backup_$(date +%Y%m%d_%H%M%S).tar.gz\"\ntar -czvf \"$BACKUP_NAME\" \"$DIR\"\necho \"Backup created: $BACKUP_NAME\"\nls -lh \"$BACKUP_NAME\"", "bash", "Create a timestamped tar.gz backup of current or specified directory", "archive", "system", "", false},
		{"seed-005", "Network Check", "#!/bin/bash\necho '=== Network Interfaces ==='\nip addr show | grep -E '^[0-9]+:|inet '\necho ''\necho '=== Connectivity Test ==='\nping -c 3 8.8.8.8\necho ''\necho '=== DNS Test ==='\nnslookup google.com", "bash", "Check network interfaces, connectivity, and DNS resolution", "network", "network", "", false},

		// Node.js/JavaScript snippets
		{"seed-006", "Node Project Init", "#!/bin/bash\nmkdir -p src tests\nnpm init -y\nnpm install --save-dev typescript @types/node jest ts-jest\ncat > tsconfig.json << 'EOF'\n{\n  \"compilerOptions\": {\n    \"target\": \"ES2020\",\n    \"module\": \"commonjs\",\n    \"outDir\": \"./dist\",\n    \"rootDir\": \"./src\",\n    \"strict\": true\n  }\n}\nEOF\necho 'console.log(\"Hello, TypeScript!\");' > src/index.ts\necho 'Node.js TypeScript project initialized!'", "bash", "Initialize a new Node.js project with TypeScript, Jest, and proper directory structure", "package", "nodejs", "npm install -g typescript ts-node", true},
		{"seed-007", "NPM Audit Fix", "#!/bin/bash\necho '=== Running npm audit ==='\nnpm audit\necho ''\necho '=== Fixing vulnerabilities ==='\nnpm audit fix\necho ''\necho '=== Checking outdated packages ==='\nnpm outdated", "bash", "Run npm security audit, fix vulnerabilities, and check for outdated packages", "shield", "nodejs", "", false},
		{"seed-008", "Express API Starter", "const express = require('express');\nconst app = express();\napp.use(express.json());\n\napp.get('/health', (req, res) => res.json({ status: 'ok' }));\n\napp.get('/api/items', (req, res) => {\n  res.json([{ id: 1, name: 'Item 1' }]);\n});\n\napp.post('/api/items', (req, res) => {\n  res.status(201).json({ id: 2, ...req.body });\n});\n\napp.listen(3000, () => console.log('Server running on :3000'));", "javascript", "Minimal Express.js REST API with health check and CRUD endpoints", "rocket", "nodejs", "npm install express", true},
		{"seed-009", "Package.json Scripts", "#!/bin/bash\ncat << 'EOF'\n// Add these to your package.json scripts:\n\"scripts\": {\n  \"dev\": \"nodemon src/index.ts\",\n  \"build\": \"tsc\",\n  \"start\": \"node dist/index.js\",\n  \"test\": \"jest\",\n  \"test:watch\": \"jest --watch\",\n  \"lint\": \"eslint src/**/*.ts\",\n  \"format\": \"prettier --write src/**/*.ts\"\n}\nEOF", "bash", "Common package.json scripts for TypeScript Node.js projects", "file", "nodejs", "", false},

		// Python/Data Science snippets
		{"seed-010", "Python Venv Setup", "#!/bin/bash\npython3 -m venv venv\nsource venv/bin/activate\npip install --upgrade pip\npip install pandas numpy matplotlib jupyter requests\npip freeze > requirements.txt\necho 'Virtual environment created and activated!'", "bash", "Create Python virtual environment with common data science packages", "python", "python", "", false},
		{"seed-011", "Data Analysis Template", "import pandas as pd\nimport numpy as np\nimport matplotlib.pyplot as plt\n\n# Load data\ndf = pd.read_csv('data.csv')\n\n# Quick overview\nprint('Shape:', df.shape)\nprint('\nColumns:', df.columns.tolist())\nprint('\nData types:')\nprint(df.dtypes)\nprint('\nSummary statistics:')\nprint(df.describe())\nprint('\nMissing values:')\nprint(df.isnull().sum())", "python", "Python data analysis template with pandas for quick dataset exploration", "chart", "python", "pip install pandas numpy matplotlib", true},
		{"seed-012", "Flask API Starter", "from flask import Flask, jsonify, request\n\napp = Flask(__name__)\n\nitems = [{'id': 1, 'name': 'Item 1'}]\n\n@app.route('/health')\ndef health():\n    return jsonify({'status': 'ok'})\n\n@app.route('/api/items', methods=['GET'])\ndef get_items():\n    return jsonify(items)\n\n@app.route('/api/items', methods=['POST'])\ndef create_item():\n    item = {'id': len(items) + 1, **request.json}\n    items.append(item)\n    return jsonify(item), 201\n\nif __name__ == '__main__':\n    app.run(debug=True, port=5000)", "python", "Minimal Flask REST API with health check and CRUD endpoints", "server", "python", "pip install flask", true},
		{"seed-013", "Jupyter Setup", "#!/bin/bash\npip install jupyterlab ipykernel\npython -m ipykernel install --user --name=myenv\njupyter lab --ip=0.0.0.0 --port=8888 --no-browser --allow-root", "bash", "Install JupyterLab and start it for remote access", "notebook", "python", "pip install jupyterlab", true},

		// Go/Gopher snippets
		{"seed-014", "Go Module Init", "#!/bin/bash\ngo mod init myproject\nmkdir -p cmd/myapp internal pkg\ncat > cmd/myapp/main.go << 'EOF'\npackage main\n\nimport \"fmt\"\n\nfunc main() {\n\tfmt.Println(\"Hello, Go!\")\n}\nEOF\ngo mod tidy\ngo build -o bin/myapp ./cmd/myapp\necho 'Go project initialized!'", "bash", "Initialize a Go module with standard project layout", "go", "golang", "", false},
		{"seed-015", "Go HTTP Server", "package main\n\nimport (\n\t\"encoding/json\"\n\t\"log\"\n\t\"net/http\"\n)\n\nfunc main() {\n\thttp.HandleFunc(\"/health\", func(w http.ResponseWriter, r *http.Request) {\n\t\tjson.NewEncoder(w).Encode(map[string]string{\"status\": \"ok\"})\n\t})\n\n\thttp.HandleFunc(\"/api/items\", func(w http.ResponseWriter, r *http.Request) {\n\t\titems := []map[string]interface{}{{\"id\": 1, \"name\": \"Item 1\"}}\n\t\tw.Header().Set(\"Content-Type\", \"application/json\")\n\t\tjson.NewEncoder(w).Encode(items)\n\t})\n\n\tlog.Println(\"Server starting on :8080\")\n\tlog.Fatal(http.ListenAndServe(\":8080\", nil))\n}", "go", "Simple Go HTTP server with JSON endpoints", "network", "golang", "", false},
		{"seed-016", "Go Test Template", "package mypackage\n\nimport \"testing\"\n\nfunc TestAdd(t *testing.T) {\n\ttests := []struct {\n\t\tname     string\n\t\ta, b     int\n\t\texpected int\n\t}{\n\t\t{\"positive\", 2, 3, 5},\n\t\t{\"negative\", -1, -1, -2},\n\t\t{\"zero\", 0, 0, 0},\n	}\n\n\tfor _, tt := range tests {\n\t\tt.Run(tt.name, func(t *testing.T) {\n\t\t\tresult := Add(tt.a, tt.b)\n\t\t\tif result != tt.expected {\n\t\t\t\tt.Errorf(\"Add(%d, %d) = %d; want %d\", tt.a, tt.b, result, tt.expected)\n\t\t\t}\n\t\t})\n\t}\n}", "go", "Go table-driven test template with subtests", "test", "golang", "", false},
		{"seed-017", "Go Build All", "#!/bin/bash\necho 'Building for multiple platforms...'\nGOOS=linux GOARCH=amd64 go build -o bin/app-linux-amd64 ./cmd/app\nGOOS=darwin GOARCH=amd64 go build -o bin/app-darwin-amd64 ./cmd/app\nGOOS=windows GOARCH=amd64 go build -o bin/app-windows-amd64.exe ./cmd/app\nls -la bin/\necho 'Cross-compilation complete!'", "bash", "Cross-compile Go application for Linux, macOS, and Windows", "build", "golang", "", false},

		// Neovim/Editor snippets
		{"seed-018", "Neovim Config Check", "#!/bin/bash\necho '=== Neovim Version ==='\nnvim --version | head -5\necho ''\necho '=== Config Location ==='\nls -la ~/.config/nvim/ 2>/dev/null || echo 'No config found'\necho ''\necho '=== Installed Plugins ==='\nls ~/.local/share/nvim/site/pack/ 2>/dev/null || echo 'No plugins found'", "bash", "Check Neovim version, config, and installed plugins", "edit", "editor", "", false},
		{"seed-019", "LazyVim Setup", "#!/bin/bash\n# Backup existing config\nmv ~/.config/nvim ~/.config/nvim.bak 2>/dev/null\nmv ~/.local/share/nvim ~/.local/share/nvim.bak 2>/dev/null\n\n# Clone LazyVim starter\ngit clone https://github.com/LazyVim/starter ~/.config/nvim\nrm -rf ~/.config/nvim/.git\n\necho 'LazyVim installed! Run nvim to complete setup.'", "bash", "Install LazyVim - a modern Neovim configuration", "rocket", "editor", "apt-get install -y neovim || brew install neovim", true},
		{"seed-019-nvchad", "NvChad Setup", "#!/bin/bash\n# Install NvChad - a Neovim configuration framework\n\nCONFIG_DIR=\"$HOME/.config/nvim\"\nNVCHAD_DIR=\"$HOME/.config/nvchad\"\n\necho \"Backing up existing Neovim config...\"\nmv \"$CONFIG_DIR\" \"$CONFIG_DIR\".bak 2>/dev/null || true\nmv \"$NVCHAD_DIR\" \"$NVCHAD_DIR\".bak 2>/dev/null || true\n\necho \"Cloning NvChad starter config...\"\ngit clone https://github.com/NvChad/NvChad \"$CONFIG_DIR\" --depth 1\nnvim --headless \"+MasonInstallAll\" \"+quit\" # Install LSPs and formatters\n\necho \"NvChad installed! Run nvim to complete setup.\"", "bash", "Install NvChad - a modular Neovim configuration framework", "rocket", "editor", "apt-get install -y neovim git || brew install neovim git", true},
		{"seed-019-lunarvim", "LunarVim Setup", "#!/bin/bash\n# Install LunarVim - an IDE-like Neovim experience\n\nLV_BRANCH='release-1.3/neovim-0.9'\nLAZY_PATH=\"$HOME/.local/share/lunarvim/lvim/bin\"\n\necho \"Backing up existing Neovim config...\"\nmv ~/.config/nvim ~/.config/nvim.bak 2>/dev/null || true\nmv ~/.local/share/lunarvim ~/.local/share/lunarvim.bak 2>/dev/null || true\nmv ~/.cache/lunarvim ~/.cache/lunarvim.bak 2>/dev/null || true\n\necho \"Installing LunarVim...\"\nLV_BRANCH=\"$LV_BRANCH\" bash <(curl -s https://raw.githubusercontent.com/lunarvim/lunarvim/master/utils/installer/install.sh)\n\necho \"Adding LunarVim to PATH for current session...\"\nexport PATH=\"$LAZY_PATH:$PATH\"\n\necho \"LunarVim installed! Run lvim to complete setup.\"", "bash", "Install LunarVim - a community-driven Neovim IDE experience", "rocket", "editor", "apt-get install -y neovim curl git || brew install neovim curl git", true},
		{"seed-020", "Ripgrep Search", "#!/bin/bash\n# Usage: ./script.sh 'pattern' [path]\nPATTERN=\"${1:-TODO}\"\nPATH=\"${2:-.}\"\n\necho \"Searching for '$PATTERN' in $PATH\"\nrg --color=always --line-number --heading \"$PATTERN\" \"$PATH\" | head -100", "bash", "Fast code search with ripgrep - colorized output with line numbers", "search", "editor", "apt-get install -y ripgrep || brew install ripgrep", true},

		// DevOps/YAML Herder snippets
		{"seed-021", "Docker Cleanup", "#!/bin/bash\necho '=== Removing stopped containers ==='\ndocker container prune -f\necho ''\necho '=== Removing unused images ==='\ndocker image prune -a -f\necho ''\necho '=== Removing unused volumes ==='\ndocker volume prune -f\necho ''\necho '=== Disk usage ==='\ndocker system df", "bash", "Clean up Docker resources - containers, images, volumes", "docker", "devops", "", false},
		{"seed-022", "Kubernetes Debug", "#!/bin/bash\nNS=${1:-default}\necho \"=== Pods in $NS ===\"\nkubectl get pods -n $NS\necho ''\necho '=== Recent Events ==='\nkubectl get events -n $NS --sort-by='.lastTimestamp' | tail -20\necho ''\necho '=== Failed Pods ==='\nkubectl get pods -n $NS --field-selector=status.phase!=Running,status.phase!=Succeeded", "bash", "Debug Kubernetes namespace - show pods, events, and failures", "kubernetes", "devops", "curl -LO https://dl.k8s.io/release/$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/linux/amd64/kubectl && chmod +x kubectl && mv kubectl /usr/local/bin/", true},
		{"seed-023", "Terraform Init", "#!/bin/bash\nterraform init\nterraform validate\nterraform fmt -recursive\nterraform plan -out=tfplan\necho ''\necho 'Review the plan above. Run: terraform apply tfplan'", "bash", "Initialize, validate, format, and plan Terraform configuration", "cloud", "devops", "apt-get install -y terraform || brew install terraform", true},
		{"seed-024", "Docker Compose Template", "version: '3.8'\n\nservices:\n  app:\n    build: .\n    ports:\n      - \"3000:3000\"\n    environment:\n      - NODE_ENV=development\n    volumes:\n      - .:/app\n      - /app/node_modules\n    depends_on:\n      - db\n      - redis\n\n  db:\n    image: postgres:15-alpine\n    environment:\n      POSTGRES_PASSWORD: secret\n      POSTGRES_DB: myapp\n    volumes:\n      - postgres_data:/var/lib/postgresql/data\n\n  redis:\n    image: redis:7-alpine\n\nvolumes:\n  postgres_data:", "yaml", "Docker Compose template with app, PostgreSQL, and Redis", "docker", "devops", "", false},
		{"seed-025", "Ansible Ping", "#!/bin/bash\nansible all -m ping -i inventory.ini\necho ''\necho '=== Host Facts ==='\nansible all -m setup -i inventory.ini -a 'filter=ansible_distribution*' | head -50", "bash", "Ansible ping all hosts and gather basic facts", "server", "devops", "pip install ansible", true},

		// AI/Vibe Coder snippets
		{"seed-026", "Claude Code", "#!/bin/bash\necho 'Starting Claude Code AI assistant...'\necho ''\necho 'Tips:'\necho '  - Use /help to see available commands'\necho '  - Describe what you want to build'\necho '  - Press Ctrl+C to exit'\necho ''\nclaude", "bash", "Launch Claude Code AI coding assistant", "ai", "ai", "npm install -g @anthropic-ai/claude-code", true},
		{"seed-027", "GitHub Copilot CLI", "#!/bin/bash\necho 'GitHub Copilot CLI commands:'\necho '  gh copilot suggest \"how to...\"  - Get command suggestions'\necho '  gh copilot explain \"command\"    - Explain a command'\necho ''\ngh copilot suggest \"find files modified in last 24 hours\"", "bash", "Use GitHub Copilot CLI for command suggestions and explanations", "github", "ai", "gh extension install github/gh-copilot", true},
		{"seed-028", "OpenCode Quick Start", "#!/bin/bash\necho 'Starting OpenCode AI coding assistant...'\necho ''\necho 'Tips:'\necho '  - Use /help to see available commands'\necho '  - Use /compact for token-efficient mode'\necho '  - Press Ctrl+C to exit'\necho ''\nopencode", "bash", "Launch OpenCode AI assistant with helpful tips", "code", "ai", "go install github.com/opencode-ai/opencode@latest", true},
		{"seed-029", "Aider Code Review", "#!/bin/bash\n# Use Aider for AI pair programming\necho 'Starting Aider AI pair programmer...'\necho ''\necho 'Tips:'\necho '  - Add files with /add filename'\necho '  - Ask for changes naturally'\necho '  - Use /diff to see changes'\necho ''\naider", "bash", "Launch Aider AI pair programming assistant", "ai", "ai", "pip install aider-chat", true},
		{"seed-030", "Gemini CLI", "#!/bin/bash\necho 'Gemini CLI - Google AI in your terminal'\necho ''\necho 'Usage:'\necho '  gemini \"your question here\"'\necho '  echo \"code\" | gemini \"explain this\"'\necho ''\ngemini \"What can you help me with?\"", "bash", "Use Google Gemini AI from the command line", "ai", "ai", "npm install -g @google/generative-ai-cli", true},
		{"seed-030-tgpt", "Install tgpt", "#!/bin/bash\necho 'Installing tgpt - Terminal GPT without API keys...'\ncurl -sSL https://raw.githubusercontent.com/aandrew-me/tgpt/main/install | bash -s /usr/local/bin\ntgpt --version\necho ''\necho 'Usage:'\necho '  tgpt \"your question here\"'\necho '  tgpt -i  # interactive mode'\necho '  tgpt -c \"generate code\"  # code mode'", "bash", "Install tgpt - ChatGPT in terminal without API keys", "ai", "ai", "", false},
		{"seed-030-aichat", "Install aichat", "#!/bin/bash\necho 'Installing aichat - All-in-one AI CLI tool...'\ncargo install aichat\necho ''\necho 'aichat installed!'\necho ''\necho 'Usage:'\necho '  aichat \"your question\"'\necho '  aichat -r coder \"write a function\"  # use coder role'\necho '  aichat -s  # start REPL session'", "bash", "Install aichat - All-in-one AI CLI supporting multiple LLM providers", "ai", "ai", "curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh -s -- -y && source $HOME/.cargo/env", true},
		{"seed-030-mods", "Install mods", "#!/bin/bash\necho 'Installing mods - AI on the command line...'\ngo install github.com/charmbracelet/mods@latest\necho ''\necho 'mods installed!'\necho ''\necho 'Usage:'\necho '  mods \"summarize this\"'\necho '  cat file.go | mods \"explain this code\"'\necho '  mods -f  # format output as markdown'", "bash", "Install mods - AI for the command line by Charm", "ai", "ai", "", false},
		{"seed-030-llm", "Install llm", "#!/bin/bash\necho 'Installing llm - CLI tool for LLMs...'\npip install --user llm\nllm --version\necho ''\necho 'Usage:'\necho '  llm \"your prompt\"'\necho '  llm -m gpt-4 \"complex question\"'\necho '  cat file.txt | llm \"summarize\"'\necho '  llm keys set openai  # set API key'", "bash", "Install llm - CLI utility for interacting with Large Language Models", "ai", "ai", "", false},
		{"seed-030-sgpt", "Install sgpt", "#!/bin/bash\necho 'Installing shell-gpt (sgpt)...'\npip install --user shell-gpt\necho ''\necho 'sgpt installed!'\necho ''\necho 'Usage:'\necho '  sgpt \"your question\"'\necho '  sgpt --shell \"list large files\"  # generate shell commands'\necho '  sgpt --code \"python function for...\"  # generate code'", "bash", "Install shell-gpt (sgpt) - ChatGPT in your terminal", "ai", "ai", "", false},

		// Tool Installations - Languages & Runtimes
		{"seed-031", "Install Rust", "#!/bin/bash\ncurl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh -s -- -y\nsource $HOME/.cargo/env\nrustc --version\ncargo --version\necho 'Rust installed successfully!'", "bash", "Install Rust programming language via rustup", "rust", "install", "", false},
		{"seed-032", "Install Go", "#!/bin/bash\nVERSION=1.22.0\ncurl -LO https://go.dev/dl/go${VERSION}.linux-amd64.tar.gz\nsudo rm -rf /usr/local/go\nsudo tar -C /usr/local -xzf go${VERSION}.linux-amd64.tar.gz\nexport PATH=$PATH:/usr/local/go/bin\ngo version\necho 'Go installed!'", "bash", "Install Go programming language from official release", "go", "install", "", false},
		{"seed-033", "Install Deno", "#!/bin/bash\ncurl -fsSL https://deno.land/install.sh | sh\nexport DENO_INSTALL=\"$HOME/.deno\"\nexport PATH=\"$DENO_INSTALL/bin:$PATH\"\ndeno --version\necho 'Deno installed!'", "bash", "Install Deno JavaScript/TypeScript runtime", "deno", "install", "", false},
		{"seed-034", "Install Bun", "#!/bin/bash\ncurl -fsSL https://bun.sh/install | bash\nexport BUN_INSTALL=\"$HOME/.bun\"\nexport PATH=\"$BUN_INSTALL/bin:$PATH\"\nbun --version\necho 'Bun installed!'", "bash", "Install Bun JavaScript runtime and bundler", "bun", "install", "", false},

		// Tool Installations - DevOps
		{"seed-035", "Install Docker", "#!/bin/bash\ncurl -fsSL https://get.docker.com | sh\nsudo usermod -aG docker $USER\ndocker --version\necho 'Docker installed! Log out and back in for group changes.'", "bash", "Install Docker using official convenience script", "docker", "install", "", false},
		{"seed-036", "Install kubectl", "#!/bin/bash\ncurl -LO \"https://dl.k8s.io/release/$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/linux/amd64/kubectl\"\nchmod +x kubectl\nsudo mv kubectl /usr/local/bin/\nkubectl version --client\necho 'kubectl installed!'", "bash", "Install kubectl Kubernetes CLI", "kubernetes", "install", "", false},
		{"seed-037", "Install Helm", "#!/bin/bash\ncurl https://raw.githubusercontent.com/helm/helm/main/scripts/get-helm-3 | bash\nhelm version\necho 'Helm installed!'", "bash", "Install Helm Kubernetes package manager", "helm", "install", "", false},
		{"seed-038", "Install Terraform", "#!/bin/bash\nsudo apt-get update && sudo apt-get install -y gnupg software-properties-common\nwget -O- https://apt.releases.hashicorp.com/gpg | sudo gpg --dearmor -o /usr/share/keyrings/hashicorp-archive-keyring.gpg\necho \"deb [signed-by=/usr/share/keyrings/hashicorp-archive-keyring.gpg] https://apt.releases.hashicorp.com $(lsb_release -cs) main\" | sudo tee /etc/apt/sources.list.d/hashicorp.list\nsudo apt update && sudo apt install -y terraform\nterraform version", "bash", "Install Terraform infrastructure-as-code tool", "cloud", "install", "", false},
		{"seed-039", "Install Ansible", "#!/bin/bash\npip install --user ansible ansible-lint\nansible --version\necho 'Ansible installed!'", "bash", "Install Ansible automation tool via pip", "server", "install", "", false},
		{"seed-040", "Install k9s", "#!/bin/bash\ncurl -sS https://webi.sh/k9s | sh\nexport PATH=\"$HOME/.local/bin:$PATH\"\nk9s version\necho 'k9s installed!'", "bash", "Install k9s Kubernetes TUI dashboard", "kubernetes", "install", "", false},

		// Tool Installations - CLI Tools
		{"seed-041", "Install lazygit", "#!/bin/bash\nLAZYGIT_VERSION=$(curl -s \"https://api.github.com/repos/jesseduffield/lazygit/releases/latest\" | grep -Po '\"tag_name\": \"v\\K[^\"]*')\ncurl -Lo lazygit.tar.gz \"https://github.com/jesseduffield/lazygit/releases/latest/download/lazygit_${LAZYGIT_VERSION}_Linux_x86_64.tar.gz\"\ntar xf lazygit.tar.gz lazygit\nsudo install lazygit /usr/local/bin\nrm lazygit lazygit.tar.gz\nlazygit --version", "bash", "Install lazygit TUI for git", "git", "install", "", false},
		{"seed-042", "Install lazydocker", "#!/bin/bash\ncurl https://raw.githubusercontent.com/jesseduffield/lazydocker/master/scripts/install_update_linux.sh | bash\nlazydocker --version\necho 'lazydocker installed!'", "bash", "Install lazydocker TUI for Docker", "docker", "install", "", false},
		{"seed-042-theme", "Lazygit VS Code Theme", "#!/bin/bash\n# Apply VS Code theme to Lazygit config\n\nCONFIG_DIR=\"$HOME/.config/lazygit\"\nmkdir -p \"$CONFIG_DIR\"\nCONFIG_FILE=\"$CONFIG_DIR/config.yml\"\n\necho \"Applying VS Code dark theme to $CONFIG_FILE...\"\n\n# Check if config exists\nif [ ! -f \"$CONFIG_FILE\" ]; then\n    touch \"$CONFIG_FILE\"\nfi\n\n# Append theme settings (VS Code Dark Modern colors)\ncat >> \"$CONFIG_FILE\" << 'EOF'\n\ngui:\n  theme:\n    activeBorderColor:\n      - \"#007acc\"\n      - bold\n    inactiveBorderColor:\n      - \"#5a5d5e\"\n    optionsTextColor:\n      - \"#007acc\"\n    selectedLineBgColor:\n      - \"#04395e\"\n    defaultFgColor:\n      - \"#cccccc\"\n    searchingActiveBorderColor:\n      - \"#ffcc00\"\n\nEOF\n\necho \"Theme applied! Restart lazygit to see changes.\"", "bash", "Apply VS Code Dark Modern theme colors to Lazygit configuration", "palette", "install", "", false},
		{"seed-043", "Install fzf", "#!/bin/bash\ngit clone --depth 1 https://github.com/junegunn/fzf.git ~/.fzf\n~/.fzf/install --all\necho 'fzf installed! Restart shell to use.'", "bash", "Install fzf fuzzy finder with shell integration", "search", "install", "", false},
		{"seed-044", "Install zoxide", "#!/bin/bash\ncurl -sS https://raw.githubusercontent.com/ajeetdsouza/zoxide/main/install.sh | bash\necho 'eval \"$(zoxide init bash)\"' >> ~/.bashrc\necho 'zoxide installed! Use z instead of cd.'", "bash", "Install zoxide smarter cd command", "folder", "install", "", false},
		{"seed-045", "Install eza", "#!/bin/bash\ncargo install eza\necho 'alias ls=\"eza --icons\"' >> ~/.bashrc\necho 'alias ll=\"eza -la --icons\"' >> ~/.bashrc\necho 'eza installed! Use ls for colorful file listings.'", "bash", "Install eza modern ls replacement (requires Rust)", "folder", "install", "curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh -s -- -y && source $HOME/.cargo/env", true},
		{"seed-046", "Install bat", "#!/bin/bash\ncargo install bat\necho 'alias cat=\"bat\"' >> ~/.bashrc\necho 'bat installed! Use cat for syntax highlighting.'", "bash", "Install bat - cat with syntax highlighting (requires Rust)", "file", "install", "curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh -s -- -y && source $HOME/.cargo/env", true},
		{"seed-047", "Install starship", "#!/bin/bash\ncurl -sS https://starship.rs/install.sh | sh -s -- -y\necho 'eval \"$(starship init bash)\"' >> ~/.bashrc\necho 'Starship prompt installed!'", "bash", "Install starship cross-shell prompt", "rocket", "install", "", false},
		{"seed-048", "Install bottom (btm)", "#!/bin/bash\ncargo install bottom\necho 'bottom (btm) installed! Run btm for system monitoring.'", "bash", "Install bottom system monitor TUI (requires Rust)", "chart", "install", "curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh -s -- -y && source $HOME/.cargo/env", true},
		{"seed-049", "Install httpie", "#!/bin/bash\npip install --user httpie\nhttp --version\necho 'HTTPie installed! Use http GET/POST for API testing.'", "bash", "Install HTTPie user-friendly HTTP client", "network", "install", "", false},
		{"seed-050", "Install jq + yq", "#!/bin/bash\nsudo apt-get update && sudo apt-get install -y jq\npip install --user yq\njq --version\nyq --version\necho 'jq and yq installed for JSON/YAML processing!'", "bash", "Install jq and yq for JSON/YAML processing", "file", "install", "", false},

		// Shell Enhancement Tools (from roles)
		{"seed-051", "Install zsh-autosuggestions", "#!/bin/bash\necho 'Installing zsh-autosuggestions...'\ngit clone https://github.com/zsh-users/zsh-autosuggestions ~/.zsh/zsh-autosuggestions\necho 'source ~/.zsh/zsh-autosuggestions/zsh-autosuggestions.zsh' >> ~/.zshrc\necho ''\necho 'zsh-autosuggestions installed!'\necho 'Restart your shell or run: source ~/.zshrc'", "bash", "Install zsh-autosuggestions for fish-like autosuggestions in zsh", "terminal", "install", "apt-get install -y zsh git || true", true},
		{"seed-052", "Install zsh-syntax-highlighting", "#!/bin/bash\necho 'Installing zsh-syntax-highlighting...'\ngit clone https://github.com/zsh-users/zsh-syntax-highlighting.git ~/.zsh/zsh-syntax-highlighting\necho 'source ~/.zsh/zsh-syntax-highlighting/zsh-syntax-highlighting.zsh' >> ~/.zshrc\necho ''\necho 'zsh-syntax-highlighting installed!'\necho 'Restart your shell or run: source ~/.zshrc'", "bash", "Install zsh-syntax-highlighting for fish-like syntax highlighting in zsh", "terminal", "install", "apt-get install -y zsh git || true", true},
		{"seed-053", "Install Oh My Zsh", "#!/bin/bash\necho 'Installing Oh My Zsh...'\nsh -c \"$(curl -fsSL https://raw.githubusercontent.com/ohmyzsh/ohmyzsh/master/tools/install.sh)\" \"\" --unattended\necho ''\necho 'Oh My Zsh installed!'\necho 'Restart your shell to use zsh with Oh My Zsh'", "bash", "Install Oh My Zsh - framework for managing zsh configuration", "terminal", "install", "apt-get install -y zsh curl git || true", true},
		{"seed-054", "Install tmux", "#!/bin/bash\necho 'Installing tmux...'\napt-get update && apt-get install -y tmux || brew install tmux\ntmux -V\necho ''\necho 'tmux installed!'\necho ''\necho 'Quick start:'\necho '  tmux new -s mysession  # create new session'\necho '  tmux attach -t mysession  # attach to session'\necho '  Ctrl+b d  # detach from session'\necho '  Ctrl+b c  # new window'\necho '  Ctrl+b n/p  # next/previous window'", "bash", "Install tmux terminal multiplexer", "terminal", "install", "", false},
		{"seed-055", "Tmux Config", "#!/bin/bash\necho 'Creating tmux configuration...'\ncat > ~/.tmux.conf << 'EOF'\n# Better prefix key\nset -g prefix C-a\nunbind C-b\nbind C-a send-prefix\n\n# Mouse support\nset -g mouse on\n\n# Start windows at 1\nset -g base-index 1\nsetw -g pane-base-index 1\n\n# Better colors\nset -g default-terminal \"xterm-256color\"\nset -ga terminal-overrides \",xterm-256color:Tc\"\n\n# Faster escape\nset -sg escape-time 0\n\n# Split panes with | and -\nbind | split-window -h\nbind - split-window -v\n\n# Reload config\nbind r source-file ~/.tmux.conf \\; display \"Config reloaded!\"\nEOF\necho 'tmux config created at ~/.tmux.conf'\necho 'Run: tmux source-file ~/.tmux.conf'", "bash", "Create a sensible tmux configuration file", "settings", "system", "", false},
		{"seed-056", "Install neofetch", "#!/bin/bash\necho 'Installing neofetch...'\napt-get update && apt-get install -y neofetch || brew install neofetch\necho ''\nneofetch\necho ''\necho 'neofetch installed! Run neofetch to see system info'", "bash", "Install neofetch - system information tool with ASCII art", "terminal", "install", "", false},
		{"seed-057", "Install htop", "#!/bin/bash\necho 'Installing htop...'\napt-get update && apt-get install -y htop || brew install htop\nhtop --version\necho ''\necho 'htop installed! Run htop for interactive process viewer'\necho ''\necho 'Keys:'\necho '  F2 - Setup'\necho '  F3 - Search'\necho '  F4 - Filter'\necho '  F5 - Tree view'\necho '  F9 - Kill process'\necho '  q - Quit'", "bash", "Install htop - interactive process viewer", "chart", "install", "", false},
		{"seed-058", "Install vim", "#!/bin/bash\necho 'Installing vim...'\napt-get update && apt-get install -y vim || brew install vim\nvim --version | head -1\necho ''\necho 'vim installed!'", "bash", "Install vim text editor", "edit", "install", "", false},
		{"seed-059", "Install nano", "#!/bin/bash\necho 'Installing nano...'\napt-get update && apt-get install -y nano || brew install nano\nnano --version\necho ''\necho 'nano installed!'", "bash", "Install nano text editor", "edit", "install", "", false},
		{"seed-060", "Install curl + wget", "#!/bin/bash\necho 'Installing curl and wget...'\napt-get update && apt-get install -y curl wget || brew install curl wget\ncurl --version | head -1\nwget --version | head -1\necho ''\necho 'curl and wget installed!'", "bash", "Install curl and wget for HTTP requests and downloads", "network", "install", "", false},
		{"seed-061", "Install git", "#!/bin/bash\necho 'Installing git...'\napt-get update && apt-get install -y git || brew install git\ngit --version\necho ''\necho 'git installed!'\necho ''\necho 'Configure git:'\necho '  git config --global user.name \"Your Name\"'\necho '  git config --global user.email \"you@example.com\"'", "bash", "Install git version control system", "git", "install", "", false},
		{"seed-062", "Install make + gcc", "#!/bin/bash\necho 'Installing build essentials (make, gcc)...'\napt-get update && apt-get install -y build-essential || brew install make gcc\nmake --version | head -1\ngcc --version | head -1\necho ''\necho 'Build tools installed!'", "bash", "Install make and gcc build tools", "build", "install", "", false},
		{"seed-063", "Install yarn", "#!/bin/bash\necho 'Installing yarn...'\nnpm install -g yarn\nyarn --version\necho ''\necho 'yarn installed!'\necho ''\necho 'Usage:'\necho '  yarn init  # create package.json'\necho '  yarn add <package>  # add dependency'\necho '  yarn install  # install all dependencies'", "bash", "Install yarn package manager for Node.js", "package", "install", "apt-get install -y nodejs npm || true", true},
		{"seed-064", "Install ripgrep", "#!/bin/bash\necho 'Installing ripgrep (rg)...'\napt-get update && apt-get install -y ripgrep || brew install ripgrep || cargo install ripgrep\nrg --version\necho ''\necho 'ripgrep installed!'\necho ''\necho 'Usage:'\necho '  rg \"pattern\"  # search in current dir'\necho '  rg -i \"pattern\"  # case insensitive'\necho '  rg -t py \"pattern\"  # search only Python files'\necho '  rg -l \"pattern\"  # list files only'", "bash", "Install ripgrep - fast recursive search tool", "search", "install", "", false},
		{"seed-065", "Install Node.js + npm", "#!/bin/bash\necho 'Installing Node.js and npm...'\ncurl -fsSL https://deb.nodesource.com/setup_lts.x | bash -\napt-get install -y nodejs || brew install node\nnode --version\nnpm --version\necho ''\necho 'Node.js and npm installed!'", "bash", "Install Node.js LTS and npm package manager", "nodejs", "install", "", false},
		{"seed-066", "Install Python 3 + pip", "#!/bin/bash\necho 'Installing Python 3 and pip...'\napt-get update && apt-get install -y python3 python3-pip python3-venv || brew install python3\npython3 --version\npip3 --version\necho ''\necho 'Python 3 and pip installed!'", "bash", "Install Python 3 with pip and venv", "python", "install", "", false},
		{"seed-067", "Install zsh", "#!/bin/bash\necho 'Installing zsh...'\napt-get update && apt-get install -y zsh || brew install zsh\nzsh --version\necho ''\necho 'zsh installed!'\necho ''\necho 'To set zsh as default shell:'\necho '  chsh -s $(which zsh)'", "bash", "Install zsh shell", "terminal", "install", "", false},
	}

	// Insert snippets
	for _, snip := range snippets {
		// Check if this snippet already exists
		var exists bool
		err = s.db.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM snippets WHERE id = $1)", snip.ID).Scan(&exists)
		if err != nil {
			log.Printf("Warning: failed to check snippet %s: %v", snip.ID, err)
			continue
		}

		if exists {
			continue
		}

		// Encrypt the content
		encryptedContent, err := s.encryptor.Encrypt(snip.Content)
		if err != nil {
			log.Printf("Warning: failed to encrypt snippet %s: %v", snip.Name, err)
			continue
		}

		_, err = s.db.ExecContext(ctx, `
			INSERT INTO snippets (id, user_id, name, content, language, is_public, description, icon, category, install_command, requires_install, usage_count, created_at)
			VALUES ($1, $2, $3, $4, $5, true, $6, $7, $8, $9, $10, 0, NOW())
		`, snip.ID, systemUserID, snip.Name, encryptedContent, snip.Language, snip.Description, snip.Icon, snip.Category, snip.InstallCommand, snip.RequiresInstall)
		if err != nil {
			log.Printf("Warning: failed to insert snippet %s: %v", snip.Name, err)
			continue
		}
	}

	log.Println("Example snippets seeded successfully")
	return nil
}

// Close closes the database connection
func (s *PostgresStore) Close() error {
	return s.db.Close()
}

// User operations

// CreateUser creates a new user
func (s *PostgresStore) CreateUser(ctx context.Context, user *models.User, passwordHash string) error {
	allowedIPs := strings.Join(user.AllowedIPs, ",")
	query := `
		INSERT INTO users (id, email, username, password_hash, tier, is_admin, allowed_ips, session_duration_minutes, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`
	_, err := s.db.ExecContext(ctx, query,
		user.ID,
		user.Email,
		user.Username,
		passwordHash,
		user.Tier,
		user.IsAdmin,
		allowedIPs,
		user.SessionDurationMinutes,
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
	var mfaEnabled bool
	var mfaSecret string
	var screenLockHash string
	var screenLockEnabled bool
	var lockAfterMinutes int
	var lockRequiredSince sql.NullTime
	var sessionDurationMinutes int
	var allowedIPs string
	var firstName, lastName sql.NullString

	query := `
		SELECT id, email, username, COALESCE(password_hash, ''), tier, COALESCE(is_admin, false),
		       COALESCE(pipeops_id, ''), COALESCE(mfa_enabled, false), COALESCE(mfa_secret, ''),
		       COALESCE(screen_lock_hash, ''), COALESCE(screen_lock_enabled, false), COALESCE(lock_after_minutes, 5),
		       lock_required_since, COALESCE(session_duration_minutes, 0),
		       COALESCE(allowed_ips, ''), COALESCE(first_name, ''), COALESCE(last_name, ''), created_at, updated_at
		FROM users WHERE email = $1
	`
	row := s.db.QueryRowContext(ctx, query, email)
	err := row.Scan(
		&user.ID,
		&user.Email,
		&user.Username,
		&passwordHash,
		&user.Tier,
		&user.IsAdmin,
		&pipeopsID,
		&mfaEnabled,
		&mfaSecret,
		&screenLockHash,
		&screenLockEnabled,
		&lockAfterMinutes,
		&lockRequiredSince,
		&sessionDurationMinutes,
		&allowedIPs,
		&firstName,
		&lastName,
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
	user.MFAEnabled = mfaEnabled
	user.MFASecret = mfaSecret
	user.ScreenLockHash = screenLockHash
	user.ScreenLockEnabled = screenLockEnabled
	user.LockAfterMinutes = lockAfterMinutes
	if lockRequiredSince.Valid {
		user.LockRequiredSince = &lockRequiredSince.Time
	}
	user.SessionDurationMinutes = sessionDurationMinutes
	user.FirstName = firstName.String
	user.LastName = lastName.String
	if allowedIPs != "" {
		user.AllowedIPs = strings.Split(allowedIPs, ",")
	}
	return &user, passwordHash, nil
}

// GetUserByID retrieves a user by ID
func (s *PostgresStore) GetUserByID(ctx context.Context, id string) (*models.User, error) {
	var user models.User
	var pipeopsID sql.NullString
	var mfaEnabled bool
	var mfaSecret string
	var screenLockHash string
	var screenLockEnabled bool
	var lockAfterMinutes int
	var lockRequiredSince sql.NullTime
	var sessionDurationMinutes int
	var allowedIPs string
	var firstName, lastName sql.NullString
	var singleSessionMode bool

	query := `
		SELECT id, email, username, tier, COALESCE(is_admin, false), COALESCE(pipeops_id, ''),
		       COALESCE(mfa_enabled, false), COALESCE(mfa_secret, ''),
		       COALESCE(screen_lock_hash, ''), COALESCE(screen_lock_enabled, false), COALESCE(lock_after_minutes, 5),
		       lock_required_since, COALESCE(session_duration_minutes, 0),
		       COALESCE(allowed_ips, ''),
		       COALESCE(first_name, ''), COALESCE(last_name, ''),
		       COALESCE(single_session_mode, false),
		       created_at, updated_at
		FROM users WHERE id = $1
	`
	row := s.db.QueryRowContext(ctx, query, id)
	err := row.Scan(
		&user.ID,
		&user.Email,
		&user.Username,
		&user.Tier,
		&user.IsAdmin,
		&pipeopsID,
		&mfaEnabled,
		&mfaSecret,
		&screenLockHash,
		&screenLockEnabled,
		&lockAfterMinutes,
		&lockRequiredSince,
		&sessionDurationMinutes,
		&allowedIPs,
		&firstName,
		&lastName,
		&singleSessionMode,
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
	user.MFAEnabled = mfaEnabled
	user.MFASecret = mfaSecret
	user.ScreenLockHash = screenLockHash
	user.ScreenLockEnabled = screenLockEnabled
	user.LockAfterMinutes = lockAfterMinutes
	if lockRequiredSince.Valid {
		user.LockRequiredSince = &lockRequiredSince.Time
	}
	user.SessionDurationMinutes = sessionDurationMinutes
	user.FirstName = firstName.String
	user.LastName = lastName.String
	user.SingleSessionMode = singleSessionMode
	if allowedIPs != "" {
		user.AllowedIPs = strings.Split(allowedIPs, ",")
	}
	return &user, nil
}

// UpdateUser updates a user
func (s *PostgresStore) UpdateUser(ctx context.Context, user *models.User) error {
	allowedIPs := strings.Join(user.AllowedIPs, ",")
	query := `
		UPDATE users SET username = $2, tier = $3, is_admin = $4, pipeops_id = $5, mfa_enabled = $6, allowed_ips = $7, first_name = $8, last_name = $9, session_duration_minutes = $10, updated_at = $11
		WHERE id = $1
	`
	_, err := s.db.ExecContext(ctx, query,
		user.ID,
		user.Username,
		user.Tier,
		user.IsAdmin,
		user.PipeOpsID,
		user.MFAEnabled,
		allowedIPs,
		user.FirstName,
		user.LastName,
		user.SessionDurationMinutes,
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

// UpdateSSHKeyLastUsed updates the last_used_at timestamp for an SSH key
func (s *PostgresStore) UpdateSSHKeyLastUsed(ctx context.Context, id string) error {
	query := `UPDATE ssh_keys SET last_used_at = NOW() WHERE id = $1`
	_, err := s.db.ExecContext(ctx, query, id)
	return err
}

// GetUserByUsername retrieves a user by their username
func (s *PostgresStore) GetUserByUsername(ctx context.Context, username string) (*models.User, error) {
	var user models.User
	var pipeopsID sql.NullString
	var mfaEnabled bool
	var mfaSecret string
	var screenLockHash string
	var screenLockEnabled bool
	var lockAfterMinutes int
	var lockRequiredSince sql.NullTime
	var sessionDurationMinutes int
	var allowedIPs string
	var firstName, lastName sql.NullString

	query := `
		SELECT id, email, username, tier, COALESCE(is_admin, false),
		       COALESCE(pipeops_id, ''), COALESCE(mfa_enabled, false), COALESCE(mfa_secret, ''),
		       COALESCE(screen_lock_hash, ''), COALESCE(screen_lock_enabled, false), COALESCE(lock_after_minutes, 5),
		       lock_required_since, COALESCE(session_duration_minutes, 0),
		       COALESCE(allowed_ips, ''), COALESCE(first_name, ''), COALESCE(last_name, ''), created_at, updated_at
		FROM users WHERE username = $1
	`
	row := s.db.QueryRowContext(ctx, query, username)
	err := row.Scan(
		&user.ID,
		&user.Email,
		&user.Username,
		&user.Tier,
		&user.IsAdmin,
		&pipeopsID,
		&mfaEnabled,
		&mfaSecret,
		&screenLockHash,
		&screenLockEnabled,
		&lockAfterMinutes,
		&lockRequiredSince,
		&sessionDurationMinutes,
		&allowedIPs,
		&firstName,
		&lastName,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("user not found")
	}
	if err != nil {
		return nil, err
	}

	user.PipeOpsID = pipeopsID.String
	user.MFAEnabled = mfaEnabled
	user.MFASecret = mfaSecret
	user.ScreenLockHash = screenLockHash
	user.ScreenLockEnabled = screenLockEnabled
	user.LockAfterMinutes = lockAfterMinutes
	if lockRequiredSince.Valid {
		user.LockRequiredSince = &lockRequiredSince.Time
	}
	user.SessionDurationMinutes = sessionDurationMinutes
	user.FirstName = firstName.String
	user.LastName = lastName.String
	if allowedIPs != "" {
		user.AllowedIPs = strings.Split(allowedIPs, ",")
	}

	return &user, nil
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
	Role       string    `db:"role"`
	Status     string    `db:"status"`
	DockerID   string    `db:"docker_id"`
	VolumeName string    `db:"volume_name"`
	MemoryMB   int64     `db:"memory_mb"`
	CPUShares  int64     `db:"cpu_shares"`
	DiskMB     int64     `db:"disk_mb"`
	MFALocked  bool      `db:"mfa_locked"`
	CreatedAt  time.Time `db:"created_at"`
	LastUsedAt time.Time `db:"last_used_at"`
}

// CreateContainer creates a container record
func (s *PostgresStore) CreateContainer(ctx context.Context, container *ContainerRecord) error {
	query := `
		INSERT INTO containers (id, user_id, name, image, role, status, docker_id, volume_name, memory_mb, cpu_shares, disk_mb, created_at, last_used_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
	`
	_, err := s.db.ExecContext(ctx, query,
		container.ID,
		container.UserID,
		container.Name,
		container.Image,
		container.Role,
		container.Status,
		container.DockerID,
		container.VolumeName,
		container.MemoryMB,
		container.CPUShares,
		container.DiskMB,
		container.CreatedAt,
		container.LastUsedAt,
	)
	return err
}

// GetContainersByUserID retrieves all non-deleted containers for a user
func (s *PostgresStore) GetContainersByUserID(ctx context.Context, userID string) ([]*ContainerRecord, error) {
	// TODO: Add pagination support if user container counts grow large
	query := `
		SELECT id, user_id, name, image, COALESCE(role, 'standard') as role, status, docker_id, volume_name,
		       COALESCE(memory_mb, 512) as memory_mb, COALESCE(cpu_shares, 512) as cpu_shares, COALESCE(disk_mb, 2048) as disk_mb,
		       COALESCE(mfa_locked, false) as mfa_locked, created_at, last_used_at
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
			&c.Role,
			&c.Status,
			&c.DockerID,
			&c.VolumeName,
			&c.MemoryMB,
			&c.CPUShares,
			&c.DiskMB,
			&c.MFALocked,
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
		SELECT id, user_id, name, image, COALESCE(role, 'standard') as role, status, docker_id, volume_name,
		       COALESCE(memory_mb, 512) as memory_mb, COALESCE(cpu_shares, 512) as cpu_shares, COALESCE(disk_mb, 2048) as disk_mb,
		       COALESCE(mfa_locked, false) as mfa_locked, created_at, last_used_at
		FROM containers WHERE id = $1 AND deleted_at IS NULL
	`
	row := s.db.QueryRowContext(ctx, query, id)
	err := row.Scan(
		&c.ID,
		&c.UserID,
		&c.Name,
		&c.Image,
		&c.Role,
		&c.Status,
		&c.DockerID,
		&c.VolumeName,
		&c.MemoryMB,
		&c.CPUShares,
		&c.DiskMB,
		&c.MFALocked,
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
		SELECT id, user_id, name, image, COALESCE(role, 'standard') as role, status, docker_id, volume_name,
		       COALESCE(memory_mb, 512) as memory_mb, COALESCE(cpu_shares, 512) as cpu_shares, COALESCE(disk_mb, 2048) as disk_mb,
		       COALESCE(mfa_locked, false) as mfa_locked, created_at, last_used_at
		FROM containers WHERE user_id = $1 AND docker_id = $2 AND deleted_at IS NULL
	`
	row := s.db.QueryRowContext(ctx, query, userID, dockerID)
	err := row.Scan(
		&c.ID,
		&c.UserID,
		&c.Name,
		&c.Image,
		&c.Role,
		&c.Status,
		&c.DockerID,
		&c.VolumeName,
		&c.MemoryMB,
		&c.CPUShares,
		&c.DiskMB,
		&c.MFALocked,
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

// GetContainerByDockerID retrieves a container by its Docker ID (for collab sessions)
func (s *PostgresStore) GetContainerByDockerID(ctx context.Context, dockerID string) (*ContainerRecord, error) {
	var c ContainerRecord
	query := `
		SELECT id, user_id, name, image, COALESCE(role, 'standard') as role, status, docker_id, volume_name,
		       COALESCE(memory_mb, 512) as memory_mb, COALESCE(cpu_shares, 512) as cpu_shares, COALESCE(disk_mb, 2048) as disk_mb,
		       COALESCE(mfa_locked, false) as mfa_locked, created_at, last_used_at
		FROM containers WHERE docker_id = $1 AND deleted_at IS NULL
	`
	row := s.db.QueryRowContext(ctx, query, dockerID)
	err := row.Scan(
		&c.ID,
		&c.UserID,
		&c.Name,
		&c.Image,
		&c.Role,
		&c.Status,
		&c.DockerID,
		&c.VolumeName,
		&c.MemoryMB,
		&c.CPUShares,
		&c.DiskMB,
		&c.MFALocked,
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

// SetContainerMFALock sets the MFA lock status for a container
func (s *PostgresStore) SetContainerMFALock(ctx context.Context, id string, locked bool) error {
	query := `UPDATE containers SET mfa_locked = $2 WHERE id = $1 AND deleted_at IS NULL`
	result, err := s.db.ExecContext(ctx, query, id, locked)
	if err != nil {
		return err
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("container not found")
	}
	return nil
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
	query := `UPDATE containers SET deleted_at = CURRENT_TIMESTAMP, status = 'deleted' WHERE id = $1`
	_, err := s.db.ExecContext(ctx, query, id)
	return err
}

// UpdateContainerSettings updates a container's name and resource settings
func (s *PostgresStore) UpdateContainerSettings(ctx context.Context, id, name string, memoryMB, cpuShares, diskMB int64) error {
	query := `UPDATE containers SET name = $2, memory_mb = $3, cpu_shares = $4, disk_mb = $5 WHERE id = $1 AND deleted_at IS NULL`
	result, err := s.db.ExecContext(ctx, query, id, name, memoryMB, cpuShares, diskMB)
	if err != nil {
		return fmt.Errorf("update query failed: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("no container found with id %s or it was deleted", id)
	}

	return nil
}

// HardDeleteContainer permanently deletes a container record (for cleanup)
func (s *PostgresStore) HardDeleteContainer(ctx context.Context, id string) error {
	query := `DELETE FROM containers WHERE id = $1`
	_, err := s.db.ExecContext(ctx, query, id)
	return err
}

// UpdateContainerShellMetadata updates shell-related metadata for faster terminal connections
func (s *PostgresStore) UpdateContainerShellMetadata(ctx context.Context, id, shellPath string, hasTmux, shellSetupDone bool) error {
	query := `UPDATE containers SET shell_path = $2, has_tmux = $3, shell_setup_done = $4 WHERE id = $1 AND deleted_at IS NULL`
	_, err := s.db.ExecContext(ctx, query, id, shellPath, hasTmux, shellSetupDone)
	return err
}

// GetContainerShellMetadata retrieves cached shell metadata for a container
func (s *PostgresStore) GetContainerShellMetadata(ctx context.Context, id string) (shellPath string, hasTmux bool, shellSetupDone bool, err error) {
	query := `SELECT COALESCE(shell_path, ''), COALESCE(has_tmux, false), COALESCE(shell_setup_done, false) FROM containers WHERE id = $1 AND deleted_at IS NULL`
	err = s.db.QueryRowContext(ctx, query, id).Scan(&shellPath, &hasTmux, &shellSetupDone)
	return
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
		SELECT id, user_id, name, image, COALESCE(role, 'standard') as role, status, docker_id, volume_name,
		       COALESCE(memory_mb, 512) as memory_mb, COALESCE(cpu_shares, 512) as cpu_shares, COALESCE(disk_mb, 2048) as disk_mb,
		       COALESCE(mfa_locked, false) as mfa_locked, created_at, last_used_at
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
			&c.Role,
			&c.Status,
			&c.DockerID,
			&c.VolumeName,
			&c.MemoryMB,
			&c.CPUShares,
			&c.DiskMB,
			&c.MFALocked,
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

// ============================================================================
// Terminal Recordings
// ============================================================================

// RecordingRecord represents a terminal recording in the database
type RecordingRecord struct {
	ID          string     `db:"id"`
	UserID      string     `db:"user_id"`
	ContainerID string     `db:"container_id"`
	Title       string     `db:"title"`
	Duration    int64      `db:"duration_ms"` // Duration in milliseconds
	Size        int64      `db:"size_bytes"`  // Size of recording data
	Data        []byte     `db:"data"`        // Recording data (gzipped asciicast) - only if StorageType='database'
	ShareToken  string     `db:"share_token"` // Public share link token
	IsPublic    bool       `db:"is_public"`   // Whether publicly accessible
	CreatedAt   time.Time  `db:"created_at"`
	ExpiresAt   *time.Time `db:"expires_at"`   // Optional expiration
	StorageType string     `db:"storage_type"` // 'database', 'r2', 's3'
	StorageURL  string     `db:"storage_url"`  // CDN URL for R2/S3 storage
}

// CreateRecording creates a new recording record
func (s *PostgresStore) CreateRecording(ctx context.Context, rec *RecordingRecord) error {
	// Default storage type if not set
	if rec.StorageType == "" {
		rec.StorageType = "database"
	}

	query := `
		INSERT INTO terminal_recordings (id, user_id, container_id, title, duration_ms, size_bytes, data, share_token, is_public, created_at, expires_at, storage_type, storage_url)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
	`
	_, err := s.db.ExecContext(ctx, query,
		rec.ID,
		rec.UserID,
		rec.ContainerID,
		rec.Title,
		rec.Duration,
		rec.Size,
		rec.Data,
		rec.ShareToken,
		rec.IsPublic,
		rec.CreatedAt,
		rec.ExpiresAt,
		rec.StorageType,
		rec.StorageURL,
	)
	return err
}

// GetRecordingsByUserID retrieves all recordings for a user
func (s *PostgresStore) GetRecordingsByUserID(ctx context.Context, userID string) ([]*RecordingRecord, error) {
	query := `
		SELECT id, user_id, container_id, title, duration_ms, size_bytes, share_token, is_public, created_at, expires_at, COALESCE(storage_type, 'database'), COALESCE(storage_url, '')
		FROM terminal_recordings WHERE user_id = $1
		ORDER BY created_at DESC
	`
	rows, err := s.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var recordings []*RecordingRecord
	for rows.Next() {
		var r RecordingRecord
		err := rows.Scan(
			&r.ID,
			&r.UserID,
			&r.ContainerID,
			&r.Title,
			&r.Duration,
			&r.Size,
			&r.ShareToken,
			&r.IsPublic,
			&r.CreatedAt,
			&r.ExpiresAt,
			&r.StorageType,
			&r.StorageURL,
		)
		if err != nil {
			return nil, err
		}
		recordings = append(recordings, &r)
	}
	return recordings, nil
}

// GetRecordingByID retrieves a recording by ID
func (s *PostgresStore) GetRecordingByID(ctx context.Context, id string) (*RecordingRecord, error) {
	var r RecordingRecord
	query := `
		SELECT id, user_id, container_id, title, duration_ms, size_bytes, share_token, is_public, created_at, expires_at, COALESCE(storage_type, 'database'), COALESCE(storage_url, '')
		FROM terminal_recordings WHERE id = $1
	`
	row := s.db.QueryRowContext(ctx, query, id)
	err := row.Scan(
		&r.ID,
		&r.UserID,
		&r.ContainerID,
		&r.Title,
		&r.Duration,
		&r.Size,
		&r.ShareToken,
		&r.IsPublic,
		&r.CreatedAt,
		&r.ExpiresAt,
		&r.StorageType,
		&r.StorageURL,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &r, nil
}

// GetRecordingByShareToken retrieves a public recording by share token
func (s *PostgresStore) GetRecordingByShareToken(ctx context.Context, token string) (*RecordingRecord, error) {
	var r RecordingRecord
	query := `
		SELECT id, user_id, container_id, title, duration_ms, size_bytes, share_token, is_public, created_at, expires_at, COALESCE(storage_type, 'database'), COALESCE(storage_url, '')
		FROM terminal_recordings
		WHERE share_token = $1 AND is_public = true AND (expires_at IS NULL OR expires_at > NOW())
	`
	row := s.db.QueryRowContext(ctx, query, token)
	err := row.Scan(
		&r.ID,
		&r.UserID,
		&r.ContainerID,
		&r.Title,
		&r.Duration,
		&r.Size,
		&r.ShareToken,
		&r.IsPublic,
		&r.CreatedAt,
		&r.ExpiresAt,
		&r.StorageType,
		&r.StorageURL,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &r, nil
}

// UpdateRecordingVisibility updates a recording's public visibility
func (s *PostgresStore) UpdateRecordingVisibility(ctx context.Context, id string, isPublic bool) error {
	query := `UPDATE terminal_recordings SET is_public = $2 WHERE id = $1`
	_, err := s.db.ExecContext(ctx, query, id, isPublic)
	return err
}

// DeleteRecording deletes a recording
func (s *PostgresStore) DeleteRecording(ctx context.Context, id string) error {
	query := `DELETE FROM terminal_recordings WHERE id = $1`
	_, err := s.db.ExecContext(ctx, query, id)
	return err
}

// GetRecordingData retrieves only the recording data for streaming
func (s *PostgresStore) GetRecordingData(ctx context.Context, id string) ([]byte, error) {
	var data []byte
	query := `SELECT data FROM terminal_recordings WHERE id = $1`
	err := s.db.QueryRowContext(ctx, query, id).Scan(&data)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return data, err
}

// GetRecordingDataByToken retrieves recording data by share token
func (s *PostgresStore) GetRecordingDataByToken(ctx context.Context, token string) ([]byte, error) {
	var data []byte
	query := `SELECT data FROM terminal_recordings WHERE share_token = $1 AND (expires_at IS NULL OR expires_at > NOW())`
	err := s.db.QueryRowContext(ctx, query, token).Scan(&data)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return data, err
}

// ============================================================================
// Collaboration Sessions
// ============================================================================

// CollabSessionRecord represents a collaborative terminal session
type CollabSessionRecord struct {
	ID          string    `db:"id"`
	ContainerID string    `db:"container_id"`
	OwnerID     string    `db:"owner_id"`
	ShareCode   string    `db:"share_code"` // Short code for joining (e.g., "ABC123")
	Mode        string    `db:"mode"`       // "view" or "control"
	MaxUsers    int       `db:"max_users"`
	IsActive    bool      `db:"is_active"`
	CreatedAt   time.Time `db:"created_at"`
	ExpiresAt   time.Time `db:"expires_at"`
}

// CollabParticipantRecord represents a participant in a collab session
type CollabParticipantRecord struct {
	ID        string     `db:"id"`
	SessionID string     `db:"session_id"`
	UserID    string     `db:"user_id"`
	Username  string     `db:"username"`
	Role      string     `db:"role"` // "owner", "editor", "viewer"
	JoinedAt  time.Time  `db:"joined_at"`
	LeftAt    *time.Time `db:"left_at"`
}

// CreateCollabSession creates a new collaboration session
func (s *PostgresStore) CreateCollabSession(ctx context.Context, session *CollabSessionRecord) error {
	query := `
		INSERT INTO collab_sessions (id, container_id, owner_id, share_code, mode, max_users, is_active, created_at, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`
	_, err := s.db.ExecContext(ctx, query,
		session.ID,
		session.ContainerID,
		session.OwnerID,
		session.ShareCode,
		session.Mode,
		session.MaxUsers,
		session.IsActive,
		session.CreatedAt,
		session.ExpiresAt,
	)
	return err
}

// GetCollabSessionByShareCode retrieves an active collab session by share code
func (s *PostgresStore) GetCollabSessionByShareCode(ctx context.Context, code string) (*CollabSessionRecord, error) {
	var session CollabSessionRecord
	query := `
		SELECT id, container_id, owner_id, share_code, mode, max_users, is_active, created_at, expires_at
		FROM collab_sessions
		WHERE share_code = $1 AND is_active = true AND expires_at > NOW()
	`
	row := s.db.QueryRowContext(ctx, query, code)
	err := row.Scan(
		&session.ID,
		&session.ContainerID,
		&session.OwnerID,
		&session.ShareCode,
		&session.Mode,
		&session.MaxUsers,
		&session.IsActive,
		&session.CreatedAt,
		&session.ExpiresAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &session, nil
}

// GetCollabSessionByContainerID retrieves the active collab session for a container
func (s *PostgresStore) GetCollabSessionByContainerID(ctx context.Context, containerID string) (*CollabSessionRecord, error) {
	var session CollabSessionRecord
	query := `
		SELECT id, container_id, owner_id, share_code, mode, max_users, is_active, created_at, expires_at
		FROM collab_sessions
		WHERE container_id = $1 AND is_active = true AND expires_at > NOW()
	`
	row := s.db.QueryRowContext(ctx, query, containerID)
	err := row.Scan(
		&session.ID,
		&session.ContainerID,
		&session.OwnerID,
		&session.ShareCode,
		&session.Mode,
		&session.MaxUsers,
		&session.IsActive,
		&session.CreatedAt,
		&session.ExpiresAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &session, nil
}

// GetCollabSessionByID retrieves a collab session by its ID
func (s *PostgresStore) GetCollabSessionByID(ctx context.Context, id string) (*CollabSessionRecord, error) {
	var session CollabSessionRecord
	query := `
		SELECT id, container_id, owner_id, share_code, mode, max_users, is_active, created_at, expires_at
		FROM collab_sessions
		WHERE id = $1
	`
	row := s.db.QueryRowContext(ctx, query, id)
	err := row.Scan(
		&session.ID,
		&session.ContainerID,
		&session.OwnerID,
		&session.ShareCode,
		&session.Mode,
		&session.MaxUsers,
		&session.IsActive,
		&session.CreatedAt,
		&session.ExpiresAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &session, nil
}

// EndCollabSession marks a collab session as inactive
func (s *PostgresStore) EndCollabSession(ctx context.Context, id string) error {
	query := `UPDATE collab_sessions SET is_active = false WHERE id = $1`
	_, err := s.db.ExecContext(ctx, query, id)
	return err
}

// AddCollabParticipant adds a participant to a collab session (upsert - safe to call multiple times)
func (s *PostgresStore) AddCollabParticipant(ctx context.Context, p *CollabParticipantRecord) error {
	query := `
		INSERT INTO collab_participants (id, session_id, user_id, username, role, joined_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (session_id, user_id) DO UPDATE SET
			username = EXCLUDED.username,
			role = EXCLUDED.role,
			left_at = NULL
	`
	_, err := s.db.ExecContext(ctx, query,
		p.ID,
		p.SessionID,
		p.UserID,
		p.Username,
		p.Role,
		p.JoinedAt,
	)
	return err
}

// GetCollabParticipants retrieves all active participants in a session
func (s *PostgresStore) GetCollabParticipants(ctx context.Context, sessionID string) ([]*CollabParticipantRecord, error) {
	query := `
		SELECT id, session_id, user_id, username, role, joined_at, left_at
		FROM collab_participants
		WHERE session_id = $1 AND left_at IS NULL
		ORDER BY joined_at
	`
	rows, err := s.db.QueryContext(ctx, query, sessionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var participants []*CollabParticipantRecord
	for rows.Next() {
		var p CollabParticipantRecord
		err := rows.Scan(
			&p.ID,
			&p.SessionID,
			&p.UserID,
			&p.Username,
			&p.Role,
			&p.JoinedAt,
			&p.LeftAt,
		)
		if err != nil {
			return nil, err
		}
		participants = append(participants, &p)
	}
	return participants, nil
}

// RemoveCollabParticipant marks a participant as left
func (s *PostgresStore) RemoveCollabParticipant(ctx context.Context, sessionID, userID string) error {
	query := `UPDATE collab_participants SET left_at = NOW() WHERE session_id = $1 AND user_id = $2 AND left_at IS NULL`
	_, err := s.db.ExecContext(ctx, query, sessionID, userID)
	return err
}

// GetActiveCollabSessionCount returns the count of active participants in a session
func (s *PostgresStore) GetActiveCollabSessionCount(ctx context.Context, sessionID string) (int, error) {
	var count int
	query := `SELECT COUNT(*) FROM collab_participants WHERE session_id = $1 AND left_at IS NULL`
	err := s.db.QueryRowContext(ctx, query, sessionID).Scan(&count)
	return count, err
}

// GetActiveCollabSessionsForParticipant returns all active collab sessions where the user
// is a participant (not the owner). This is used to show shared terminals on the dashboard.
func (s *PostgresStore) GetActiveCollabSessionsForParticipant(ctx context.Context, userID string) ([]*CollabSessionRecord, error) {
	query := `
		SELECT cs.id, cs.container_id, cs.owner_id, cs.share_code, cs.mode, cs.max_users, cs.is_active, cs.created_at, cs.expires_at
		FROM collab_sessions cs
		INNER JOIN collab_participants cp ON cs.id = cp.session_id
		WHERE cp.user_id = $1 
		  AND cp.left_at IS NULL 
		  AND cs.is_active = true 
		  AND cs.expires_at > NOW()
		  AND cs.owner_id != $1
		ORDER BY cp.joined_at DESC
	`
	rows, err := s.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sessions []*CollabSessionRecord
	for rows.Next() {
		var session CollabSessionRecord
		err := rows.Scan(
			&session.ID,
			&session.ContainerID,
			&session.OwnerID,
			&session.ShareCode,
			&session.Mode,
			&session.MaxUsers,
			&session.IsActive,
			&session.CreatedAt,
			&session.ExpiresAt,
		)
		if err != nil {
			return nil, err
		}
		sessions = append(sessions, &session)
	}
	return sessions, nil
}

// ============================================================================
// Remote Host operations (SSH Jump Hosts)
// ============================================================================

// CreateRemoteHost creates a new remote host record
func (s *PostgresStore) CreateRemoteHost(ctx context.Context, host *models.RemoteHost) error {
	// Encrypt sensitive fields
	var err error
	host.Hostname, err = s.encryptor.Encrypt(host.Hostname)
	if err != nil {
		return fmt.Errorf("failed to encrypt hostname: %w", err)
	}
	host.Username, err = s.encryptor.Encrypt(host.Username)
	if err != nil {
		return fmt.Errorf("failed to encrypt username: %w", err)
	}
	if host.IdentityFile != "" {
		host.IdentityFile, err = s.encryptor.Encrypt(host.IdentityFile)
		if err != nil {
			return fmt.Errorf("failed to encrypt identity file: %w", err)
		}
	}

	query := `
		INSERT INTO remote_hosts (id, user_id, name, hostname, port, username, identity_file, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`
	_, err = s.db.ExecContext(ctx, query,
		host.ID,
		host.UserID,
		host.Name,
		host.Hostname,
		host.Port,
		host.Username,
		host.IdentityFile,
		host.CreatedAt,
	)
	return err
}

// GetRemoteHostsByUserID retrieves all remote hosts for a user
func (s *PostgresStore) GetRemoteHostsByUserID(ctx context.Context, userID string) ([]*models.RemoteHost, error) {
	query := `
		SELECT id, user_id, name, hostname, port, username, COALESCE(identity_file, ''), created_at
		FROM remote_hosts WHERE user_id = $1
		ORDER BY created_at DESC
	`
	rows, err := s.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var hosts []*models.RemoteHost
	for rows.Next() {
		var h models.RemoteHost
		err := rows.Scan(
			&h.ID,
			&h.UserID,
			&h.Name,
			&h.Hostname,
			&h.Port,
			&h.Username,
			&h.IdentityFile,
			&h.CreatedAt,
		)
		if err != nil {
			return nil, err
		}

		// Decrypt fields
		h.Hostname, _ = s.encryptor.Decrypt(h.Hostname)
		h.Username, _ = s.encryptor.Decrypt(h.Username)
		if h.IdentityFile != "" {
			h.IdentityFile, _ = s.encryptor.Decrypt(h.IdentityFile)
		}

		hosts = append(hosts, &h)
	}
	return hosts, nil
}

// DeleteRemoteHost deletes a remote host by ID
func (s *PostgresStore) DeleteRemoteHost(ctx context.Context, id string) error {
	query := `DELETE FROM remote_hosts WHERE id = $1`
	_, err := s.db.ExecContext(ctx, query, id)
	return err
}

// GetRemoteHostByID retrieves a remote host by ID
func (s *PostgresStore) GetRemoteHostByID(ctx context.Context, id string) (*models.RemoteHost, error) {
	var h models.RemoteHost
	query := `
		SELECT id, user_id, name, hostname, port, username, COALESCE(identity_file, ''), created_at
		FROM remote_hosts WHERE id = $1
	`
	row := s.db.QueryRowContext(ctx, query, id)
	err := row.Scan(
		&h.ID,
		&h.UserID,
		&h.Name,
		&h.Hostname,
		&h.Port,
		&h.Username,
		&h.IdentityFile,
		&h.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	// Decrypt fields
	h.Hostname, _ = s.encryptor.Decrypt(h.Hostname)
	h.Username, _ = s.encryptor.Decrypt(h.Username)
	if h.IdentityFile != "" {
		h.IdentityFile, _ = s.encryptor.Decrypt(h.IdentityFile)
	}

	return &h, nil
}

// ============================================================================
// Port Forward operations
// ============================================================================

// CreatePortForward creates a new port forward record
func (s *PostgresStore) CreatePortForward(ctx context.Context, pf *models.PortForward) error {
	query := `
		INSERT INTO port_forwards (id, user_id, container_id, name, container_port, local_port, protocol, is_active, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`
	_, err := s.db.ExecContext(ctx, query,
		pf.ID,
		pf.UserID,
		pf.ContainerID,
		pf.Name,
		pf.ContainerPort,
		pf.LocalPort,
		pf.Protocol,
		pf.IsActive,
		pf.CreatedAt,
	)
	return err
}

// GetPortForwardsByUserIDAndContainerID retrieves active port forwards for a user and container
func (s *PostgresStore) GetPortForwardsByUserIDAndContainerID(ctx context.Context, userID, containerID string) ([]*models.PortForward, error) {
	query := `
		SELECT id, user_id, container_id, name, container_port, local_port, protocol, is_active, created_at
		FROM port_forwards WHERE user_id = $1 AND container_id = $2 AND is_active = true
		ORDER BY created_at DESC
	`
	rows, err := s.db.QueryContext(ctx, query, userID, containerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var forwards []*models.PortForward
	for rows.Next() {
		var pf models.PortForward
		err := rows.Scan(
			&pf.ID,
			&pf.UserID,
			&pf.ContainerID,
			&pf.Name,
			&pf.ContainerPort,
			&pf.LocalPort,
			&pf.Protocol,
			&pf.IsActive,
			&pf.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		forwards = append(forwards, &pf)
	}
	return forwards, nil
}

// GetPortForwardByID retrieves a port forward by ID
func (s *PostgresStore) GetPortForwardByID(ctx context.Context, id string) (*models.PortForward, error) {
	var pf models.PortForward
	query := `
		SELECT id, user_id, container_id, name, container_port, local_port, protocol, is_active, created_at
		FROM port_forwards WHERE id = $1
	`
	row := s.db.QueryRowContext(ctx, query, id)
	err := row.Scan(
		&pf.ID,
		&pf.UserID,
		&pf.ContainerID,
		&pf.Name,
		&pf.ContainerPort,
		&pf.LocalPort,
		&pf.Protocol,
		&pf.IsActive,
		&pf.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &pf, nil
}

// DeletePortForward marks a port forward as inactive (soft delete)
func (s *PostgresStore) DeletePortForward(ctx context.Context, id string) error {
	// We might want to just mark as inactive instead of truly deleting
	query := `UPDATE port_forwards SET is_active = false WHERE id = $1`
	_, err := s.db.ExecContext(ctx, query, id)
	return err
}

// ============================================================================
// Snippet operations
// ============================================================================

// CreateSnippet creates a new snippet record
func (s *PostgresStore) CreateSnippet(ctx context.Context, snippet *models.Snippet) error {
	// Encrypt content
	encryptedContent, err := s.encryptor.Encrypt(snippet.Content)
	if err != nil {
		return fmt.Errorf("failed to encrypt snippet content: %w", err)
	}
	snippet.Content = encryptedContent

	query := `
		INSERT INTO snippets (id, user_id, name, content, language, is_public, description, icon, category, install_command, requires_install, usage_count, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
	`
	_, err = s.db.ExecContext(ctx, query,
		snippet.ID,
		snippet.UserID,
		snippet.Name,
		snippet.Content,
		snippet.Language,
		snippet.IsPublic,
		snippet.Description,
		snippet.Icon,
		snippet.Category,
		snippet.InstallCommand,
		snippet.RequiresInstall,
		0, // usage_count starts at 0
		snippet.CreatedAt,
	)
	return err
}

// GetSnippetsByUserID retrieves all snippets for a user
func (s *PostgresStore) GetSnippetsByUserID(ctx context.Context, userID string) ([]*models.Snippet, error) {
	query := `
		SELECT id, user_id, name, content, language, is_public, COALESCE(description, ''), COALESCE(icon, ''), COALESCE(category, ''), COALESCE(install_command, ''), COALESCE(requires_install, false), COALESCE(usage_count, 0), created_at
		FROM snippets WHERE user_id = $1
		ORDER BY created_at DESC
	`
	rows, err := s.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var snippets []*models.Snippet
	for rows.Next() {
		var sn models.Snippet
		err := rows.Scan(
			&sn.ID,
			&sn.UserID,
			&sn.Name,
			&sn.Content,
			&sn.Language,
			&sn.IsPublic,
			&sn.Description,
			&sn.Icon,
			&sn.Category,
			&sn.InstallCommand,
			&sn.RequiresInstall,
			&sn.UsageCount,
			&sn.CreatedAt,
		)
		if err != nil {
			return nil, err
		}

		// Decrypt content
		decrypted, err := s.encryptor.Decrypt(sn.Content)
		if err == nil {
			sn.Content = decrypted
		} else if !isLikelyPlaintext(err) {
			sn.Content = "[Encrypted Content - Decryption Failed]"
		}

		snippets = append(snippets, &sn)
	}
	return snippets, nil
}

// GetSnippetByID retrieves a snippet by ID
func (s *PostgresStore) GetSnippetByID(ctx context.Context, id string) (*models.Snippet, error) {
	var sn models.Snippet
	query := `
		SELECT id, user_id, name, content, language, is_public, COALESCE(description, ''), COALESCE(icon, ''), COALESCE(category, ''), COALESCE(install_command, ''), COALESCE(requires_install, false), COALESCE(usage_count, 0), created_at
		FROM snippets WHERE id = $1
	`
	row := s.db.QueryRowContext(ctx, query, id)
	err := row.Scan(
		&sn.ID,
		&sn.UserID,
		&sn.Name,
		&sn.Content,
		&sn.Language,
		&sn.IsPublic,
		&sn.Description,
		&sn.Icon,
		&sn.Category,
		&sn.InstallCommand,
		&sn.RequiresInstall,
		&sn.UsageCount,
		&sn.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	// Decrypt content
	decrypted, err := s.encryptor.Decrypt(sn.Content)
	if err == nil {
		sn.Content = decrypted
	} else if !isLikelyPlaintext(err) {
		sn.Content = "[Encrypted Content - Decryption Failed]"
	}

	return &sn, nil
}

// UpdateSnippet updates a snippet
func (s *PostgresStore) UpdateSnippet(ctx context.Context, snippet *models.Snippet) error {
	// Encrypt content
	encryptedContent, err := s.encryptor.Encrypt(snippet.Content)
	if err != nil {
		return fmt.Errorf("failed to encrypt snippet content: %w", err)
	}

	query := `
		UPDATE snippets
		SET name = $1, content = $2, language = $3, is_public = $4, description = $5, icon = $6, category = $7, install_command = $8, requires_install = $9
		WHERE id = $10
	`
	_, err = s.db.ExecContext(ctx, query,
		snippet.Name,
		encryptedContent,
		snippet.Language,
		snippet.IsPublic,
		snippet.Description,
		snippet.Icon,
		snippet.Category,
		snippet.InstallCommand,
		snippet.RequiresInstall,
		snippet.ID,
	)
	return err
}

// GetPublicSnippets retrieves all public snippets for marketplace
func (s *PostgresStore) GetPublicSnippets(ctx context.Context, language, category, search, sort string) ([]*models.Snippet, error) {
	var args []interface{}
	argIdx := 1

	query := `
		SELECT s.id, s.user_id, COALESCE(u.username, 'Anonymous') as username, s.name, s.content, s.language,
		       COALESCE(s.description, ''), COALESCE(s.icon, ''), COALESCE(s.category, ''), COALESCE(s.install_command, ''), COALESCE(s.requires_install, false), COALESCE(s.usage_count, 0), s.created_at
		FROM snippets s
		LEFT JOIN users u ON s.user_id = u.id
		WHERE s.is_public = true
	`

	if language != "" && language != "all" {
		query += fmt.Sprintf(" AND s.language = $%d", argIdx)
		args = append(args, language)
		argIdx++
	}

	if category != "" && category != "all" {
		query += fmt.Sprintf(" AND s.category = $%d", argIdx)
		args = append(args, category)
		argIdx++
	}

	if search != "" {
		query += fmt.Sprintf(" AND (s.name ILIKE $%d OR s.description ILIKE $%d)", argIdx, argIdx)
		args = append(args, "%"+search+"%")
		argIdx++
	}

	switch sort {
	case "recent":
		query += " ORDER BY s.created_at DESC"
	case "name":
		query += " ORDER BY s.name ASC"
	default: // popular
		query += " ORDER BY s.usage_count DESC, s.created_at DESC"
	}

	query += " LIMIT 100"

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var snippets []*models.Snippet
	for rows.Next() {
		var sn models.Snippet
		err := rows.Scan(
			&sn.ID,
			&sn.UserID,
			&sn.Username,
			&sn.Name,
			&sn.Content,
			&sn.Language,
			&sn.Description,
			&sn.Icon,
			&sn.Category,
			&sn.InstallCommand,
			&sn.RequiresInstall,
			&sn.UsageCount,
			&sn.CreatedAt,
		)
		if err != nil {
			return nil, err
		}

		sn.IsPublic = true

		// Decrypt content
		decrypted, err := s.encryptor.Decrypt(sn.Content)
		if err == nil {
			sn.Content = decrypted
		} else if !isLikelyPlaintext(err) {
			sn.Content = "[Encrypted Content - Decryption Failed]"
		}

		snippets = append(snippets, &sn)
	}
	return snippets, nil
}

// isLikelyPlaintext returns true when a decrypt error suggests the stored value
// was never encrypted (legacy plaintext snippets).
func isLikelyPlaintext(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "illegal base64 data") ||
		strings.Contains(msg, "ciphertext too short")
}

// IncrementSnippetUsage increments the usage count for a snippet
func (s *PostgresStore) IncrementSnippetUsage(ctx context.Context, id string) error {
	query := `UPDATE snippets SET usage_count = COALESCE(usage_count, 0) + 1 WHERE id = $1`
	_, err := s.db.ExecContext(ctx, query, id)
	return err
}

// DeleteSnippet deletes a snippet by ID
func (s *PostgresStore) DeleteSnippet(ctx context.Context, id string) error {
	query := `DELETE FROM snippets WHERE id = $1`
	_, err := s.db.ExecContext(ctx, query, id)
	return err
}

// ============================================================================
// Tutorial operations
// ============================================================================

// CreateTutorial creates a new tutorial (admin only)
func (s *PostgresStore) CreateTutorial(ctx context.Context, tutorial *models.Tutorial) error {
	query := `
		INSERT INTO tutorials (id, title, description, type, content, video_url, thumbnail, duration, category, display_order, is_published, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
	`
	_, err := s.db.ExecContext(ctx, query,
		tutorial.ID,
		tutorial.Title,
		tutorial.Description,
		tutorial.Type,
		tutorial.Content,
		tutorial.VideoURL,
		tutorial.Thumbnail,
		tutorial.Duration,
		tutorial.Category,
		tutorial.Order,
		tutorial.IsPublished,
		tutorial.CreatedAt,
		tutorial.UpdatedAt,
	)
	return err
}

// GetTutorialByID retrieves a single tutorial by ID
func (s *PostgresStore) GetTutorialByID(ctx context.Context, id string) (*models.Tutorial, error) {
	var t models.Tutorial
	query := `
		SELECT id, title, COALESCE(description, ''), COALESCE(type, 'video'), COALESCE(content, ''), COALESCE(video_url, ''), COALESCE(thumbnail, ''), COALESCE(duration, ''), category, display_order, is_published, created_at, updated_at
		FROM tutorials WHERE id = $1
	`
	err := s.db.QueryRowContext(ctx, query, id).Scan(
		&t.ID,
		&t.Title,
		&t.Description,
		&t.Type,
		&t.Content,
		&t.VideoURL,
		&t.Thumbnail,
		&t.Duration,
		&t.Category,
		&t.Order,
		&t.IsPublished,
		&t.CreatedAt,
		&t.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

// GetPublishedTutorials retrieves all published tutorials ordered by category and display_order
func (s *PostgresStore) GetPublishedTutorials(ctx context.Context, category string) ([]*models.Tutorial, error) {
	var args []interface{}
	query := `
		SELECT id, title, COALESCE(description, ''), COALESCE(type, 'video'), COALESCE(content, ''), COALESCE(video_url, ''), COALESCE(thumbnail, ''), COALESCE(duration, ''), category, display_order, is_published, created_at, updated_at
		FROM tutorials
		WHERE is_published = true
	`
	if category != "" {
		query += ` AND category = $1`
		args = append(args, category)
	}
	query += ` ORDER BY created_at DESC`

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tutorials []*models.Tutorial
	for rows.Next() {
		var t models.Tutorial
		if err := rows.Scan(
			&t.ID,
			&t.Title,
			&t.Description,
			&t.Type,
			&t.Content,
			&t.VideoURL,
			&t.Thumbnail,
			&t.Duration,
			&t.Category,
			&t.Order,
			&t.IsPublished,
			&t.CreatedAt,
			&t.UpdatedAt,
		); err != nil {
			return nil, err
		}
		tutorials = append(tutorials, &t)
	}
	return tutorials, nil
}

// GetAllTutorials retrieves all tutorials (for admin)
func (s *PostgresStore) GetAllTutorials(ctx context.Context) ([]*models.Tutorial, error) {
	query := `
		SELECT id, title, COALESCE(description, ''), COALESCE(type, 'video'), COALESCE(content, ''), COALESCE(video_url, ''), COALESCE(thumbnail, ''), COALESCE(duration, ''), category, display_order, is_published, created_at, updated_at
		FROM tutorials
		ORDER BY created_at DESC
	`
	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tutorials []*models.Tutorial
	for rows.Next() {
		var t models.Tutorial
		if err := rows.Scan(
			&t.ID,
			&t.Title,
			&t.Description,
			&t.Type,
			&t.Content,
			&t.VideoURL,
			&t.Thumbnail,
			&t.Duration,
			&t.Category,
			&t.Order,
			&t.IsPublished,
			&t.CreatedAt,
			&t.UpdatedAt,
		); err != nil {
			return nil, err
		}
		tutorials = append(tutorials, &t)
	}
	return tutorials, nil
}

// UpdateTutorial updates an existing tutorial
func (s *PostgresStore) UpdateTutorial(ctx context.Context, tutorial *models.Tutorial) error {
	query := `
		UPDATE tutorials
		SET title = $1, description = $2, type = $3, content = $4, video_url = $5, thumbnail = $6, duration = $7, category = $8, display_order = $9, is_published = $10, updated_at = $11
		WHERE id = $12
	`
	_, err := s.db.ExecContext(ctx, query,
		tutorial.Title,
		tutorial.Description,
		tutorial.Type,
		tutorial.Content,
		tutorial.VideoURL,
		tutorial.Thumbnail,
		tutorial.Duration,
		tutorial.Category,
		tutorial.Order,
		tutorial.IsPublished,
		tutorial.UpdatedAt,
		tutorial.ID,
	)
	return err
}

// DeleteTutorial deletes a tutorial by ID
func (s *PostgresStore) DeleteTutorial(ctx context.Context, id string) error {
	query := `DELETE FROM tutorials WHERE id = $1`
	_, err := s.db.ExecContext(ctx, query, id)
	return err
}

// ============================================================================
// Audit Logs
// ============================================================================

// CreateAuditLog creates a new audit log entry
func (s *PostgresStore) CreateAuditLog(ctx context.Context, log *models.AuditLog) error {
	query := `
		INSERT INTO audit_logs (id, user_id, action, ip_address, user_agent, details, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`
	_, err := s.db.ExecContext(ctx, query,
		log.ID,
		log.UserID,
		log.Action,
		log.IPAddress,
		log.UserAgent,
		log.Details,
		log.CreatedAt,
	)
	return err
}

// GetAuditLogsByUserID retrieves audit logs for a user
func (s *PostgresStore) GetAuditLogsByUserID(ctx context.Context, userID string, limit, offset int) ([]*models.AuditLog, error) {
	query := `
		SELECT id, user_id, action, ip_address, user_agent, details, created_at
		FROM audit_logs WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`
	rows, err := s.db.QueryContext(ctx, query, userID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []*models.AuditLog
	for rows.Next() {
		var l models.AuditLog
		err := rows.Scan(
			&l.ID,
			&l.UserID,
			&l.Action,
			&l.IPAddress,
			&l.UserAgent,
			&l.Details,
			&l.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		logs = append(logs, &l)
	}
	return logs, nil
}

// EnableMFA enables MFA for a user with backup codes
func (s *PostgresStore) EnableMFA(ctx context.Context, userID, secret string, backupCodes []string) error {
	// Encrypt secret
	encryptedSecret, err := s.encryptor.Encrypt(secret)
	if err != nil {
		return fmt.Errorf("failed to encrypt MFA secret: %w", err)
	}

	// Encrypt backup codes (store as comma-separated encrypted string)
	backupCodesStr := strings.Join(backupCodes, ",")
	encryptedBackupCodes, err := s.encryptor.Encrypt(backupCodesStr)
	if err != nil {
		return fmt.Errorf("failed to encrypt backup codes: %w", err)
	}

	query := `UPDATE users SET mfa_enabled = true, mfa_secret = $2, mfa_backup_codes = $3, updated_at = $4 WHERE id = $1`
	_, err = s.db.ExecContext(ctx, query, userID, encryptedSecret, encryptedBackupCodes, time.Now())
	return err
}

// DisableMFA disables MFA for a user
func (s *PostgresStore) DisableMFA(ctx context.Context, userID string) error {
	query := `UPDATE users SET mfa_enabled = false, mfa_secret = '', mfa_backup_codes = '', updated_at = $2 WHERE id = $1`
	_, err := s.db.ExecContext(ctx, query, userID, time.Now())
	return err
}

// GetUserMFASecret retrieves the decrypted MFA secret for a user
func (s *PostgresStore) GetUserMFASecret(ctx context.Context, userID string) (string, error) {
	var secret string
	query := `SELECT mfa_secret FROM users WHERE id = $1`
	err := s.db.QueryRowContext(ctx, query, userID).Scan(&secret)
	if err != nil {
		return "", err
	}

	if secret == "" {
		return "", nil
	}

	return s.encryptor.Decrypt(secret)
}

// GetMFABackupCodes retrieves the backup codes for a user
func (s *PostgresStore) GetMFABackupCodes(ctx context.Context, userID string) ([]string, error) {
	var encryptedCodes string
	query := `SELECT COALESCE(mfa_backup_codes, '') FROM users WHERE id = $1`
	err := s.db.QueryRowContext(ctx, query, userID).Scan(&encryptedCodes)
	if err != nil {
		return nil, err
	}

	if encryptedCodes == "" {
		return []string{}, nil
	}

	decrypted, err := s.encryptor.Decrypt(encryptedCodes)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt backup codes: %w", err)
	}

	if decrypted == "" {
		return []string{}, nil
	}

	return strings.Split(decrypted, ","), nil
}

// UpdateMFABackupCodes updates the backup codes for a user
func (s *PostgresStore) UpdateMFABackupCodes(ctx context.Context, userID string, codes []string) error {
	codesStr := strings.Join(codes, ",")
	var encryptedCodes string
	var err error

	if codesStr != "" {
		encryptedCodes, err = s.encryptor.Encrypt(codesStr)
		if err != nil {
			return fmt.Errorf("failed to encrypt backup codes: %w", err)
		}
	}

	query := `UPDATE users SET mfa_backup_codes = $2, updated_at = $3 WHERE id = $1`
	_, err = s.db.ExecContext(ctx, query, userID, encryptedCodes, time.Now())
	return err
}

// GetMFABackupCodesCount returns how many backup codes remain for a user
func (s *PostgresStore) GetMFABackupCodesCount(ctx context.Context, userID string) (int, error) {
	codes, err := s.GetMFABackupCodes(ctx, userID)
	if err != nil {
		return 0, err
	}
	return len(codes), nil
}

// ============================================================================
// User Sessions (auth)
// ============================================================================

// CreateUserSession inserts a new tracked login session.
func (s *PostgresStore) CreateUserSession(ctx context.Context, session *models.UserSession) error {
	query := `
		INSERT INTO user_sessions (id, user_id, ip_address, user_agent, token_issued_at, created_at, last_seen_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`
	_, err := s.db.ExecContext(ctx, query,
		session.ID,
		session.UserID,
		session.IPAddress,
		session.UserAgent,
		session.CreatedAt,
		session.CreatedAt,
		session.LastSeenAt,
	)
	return err
}

// ListUserSessions returns all sessions for a user, newest first.
func (s *PostgresStore) ListUserSessions(ctx context.Context, userID string) ([]*models.UserSession, error) {
	query := `
		SELECT id, user_id, COALESCE(ip_address, ''), COALESCE(user_agent, ''), created_at,
		       COALESCE(last_seen_at, created_at), revoked_at, COALESCE(revoked_reason, '')
		FROM user_sessions
		WHERE user_id = $1
		ORDER BY created_at DESC
	`
	rows, err := s.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sessions []*models.UserSession
	for rows.Next() {
		var srec models.UserSession
		var revokedAt sql.NullTime
		if err := rows.Scan(
			&srec.ID,
			&srec.UserID,
			&srec.IPAddress,
			&srec.UserAgent,
			&srec.CreatedAt,
			&srec.LastSeenAt,
			&revokedAt,
			&srec.RevokedReason,
		); err != nil {
			return nil, err
		}
		if revokedAt.Valid {
			srec.RevokedAt = &revokedAt.Time
		}
		sessions = append(sessions, &srec)
	}
	return sessions, nil
}

// GetUserSession fetches a session by ID.
func (s *PostgresStore) GetUserSession(ctx context.Context, sessionID string) (*models.UserSession, error) {
	query := `
		SELECT id, user_id, COALESCE(ip_address, ''), COALESCE(user_agent, ''), created_at,
		       COALESCE(last_seen_at, created_at), revoked_at, COALESCE(revoked_reason, '')
		FROM user_sessions WHERE id = $1
	`
	var srec models.UserSession
	var revokedAt sql.NullTime
	err := s.db.QueryRowContext(ctx, query, sessionID).Scan(
		&srec.ID,
		&srec.UserID,
		&srec.IPAddress,
		&srec.UserAgent,
		&srec.CreatedAt,
		&srec.LastSeenAt,
		&revokedAt,
		&srec.RevokedReason,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if revokedAt.Valid {
		srec.RevokedAt = &revokedAt.Time
	}
	return &srec, nil
}

// TouchUserSession updates last_seen_at and refreshes IP/UA at most once per minute.
func (s *PostgresStore) TouchUserSession(ctx context.Context, sessionID, ip, ua string) error {
	query := `
		UPDATE user_sessions
		SET last_seen_at = NOW(),
		    ip_address = $2,
		    user_agent = $3
		WHERE id = $1
		  AND revoked_at IS NULL
		  AND (last_seen_at IS NULL OR last_seen_at < NOW() - INTERVAL '1 minute')
	`
	_, err := s.db.ExecContext(ctx, query, sessionID, ip, ua)
	return err
}

// RevokeUserSession marks a session as revoked.
func (s *PostgresStore) RevokeUserSession(ctx context.Context, userID, sessionID, reason string) error {
	query := `
		UPDATE user_sessions
		SET revoked_at = NOW(),
		    revoked_reason = $3
		WHERE id = $1 AND user_id = $2 AND revoked_at IS NULL
	`
	_, err := s.db.ExecContext(ctx, query, sessionID, userID, reason)
	return err
}

// RevokeOtherUserSessions revokes all sessions except the current one.
func (s *PostgresStore) RevokeOtherUserSessions(ctx context.Context, userID, currentSessionID, reason string) error {
	query := `
		UPDATE user_sessions
		SET revoked_at = NOW(),
		    revoked_reason = $3
		WHERE user_id = $1 AND id <> $2 AND revoked_at IS NULL
	`
	_, err := s.db.ExecContext(ctx, query, userID, currentSessionID, reason)
	return err
}

// ============================================================================
// Screen Lock (server-enforced)
// ============================================================================

// GetUserScreenLock returns the screen lock settings for a user.
func (s *PostgresStore) GetUserScreenLock(ctx context.Context, userID string) (hash string, enabled bool, lockAfterMinutes int, lockRequiredSince *time.Time, err error) {
	var lockSince sql.NullTime
	query := `
		SELECT COALESCE(screen_lock_hash, ''), COALESCE(screen_lock_enabled, false), COALESCE(lock_after_minutes, 5), lock_required_since
		FROM users WHERE id = $1
	`
	if err := s.db.QueryRowContext(ctx, query, userID).Scan(&hash, &enabled, &lockAfterMinutes, &lockSince); err != nil {
		return "", false, 0, nil, err
	}
	if lockSince.Valid {
		lockRequiredSince = &lockSince.Time
	}
	return hash, enabled, lockAfterMinutes, lockRequiredSince, nil
}

// SetScreenLockPasscode sets/updates the user's passcode hash and enables screen lock.
func (s *PostgresStore) SetScreenLockPasscode(ctx context.Context, userID, passcodeHash string, lockAfterMinutes int) error {
	query := `
		UPDATE users
		SET screen_lock_hash = $2,
		    screen_lock_enabled = true,
		    lock_after_minutes = $3,
		    updated_at = $4
		WHERE id = $1
	`
	_, err := s.db.ExecContext(ctx, query, userID, passcodeHash, lockAfterMinutes, time.Now())
	return err
}

// RemoveScreenLockPasscode disables screen lock and clears the stored hash.
func (s *PostgresStore) RemoveScreenLockPasscode(ctx context.Context, userID string) error {
	query := `
		UPDATE users
		SET screen_lock_hash = '',
		    screen_lock_enabled = false,
		    lock_required_since = NULL,
		    updated_at = $2
		WHERE id = $1
	`
	_, err := s.db.ExecContext(ctx, query, userID, time.Now())
	return err
}

// SetLockRequiredSince marks the account as locked from this time onward.
func (s *PostgresStore) SetLockRequiredSince(ctx context.Context, userID string, t time.Time) error {
	query := `
		UPDATE users
		SET lock_required_since = $2,
		    updated_at = $3
		WHERE id = $1
	`
	_, err := s.db.ExecContext(ctx, query, userID, t, time.Now())
	return err
}

// SetSingleSessionMode enables or disables single session mode for a user.
// When enabled, logging in from a new device automatically revokes all other sessions.
func (s *PostgresStore) SetSingleSessionMode(ctx context.Context, userID string, enabled bool) error {
	query := `
		UPDATE users
		SET single_session_mode = $2,
		    updated_at = $3
		WHERE id = $1
	`
	_, err := s.db.ExecContext(ctx, query, userID, enabled, time.Now())
	return err
}
