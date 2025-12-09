package storage

import (
	"context"
	"database/sql"
	"fmt"
	"log"
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
		expires_at TIMESTAMP WITH TIME ZONE
	);
	
	-- Add data column if missing (for existing installations)
	DO $$ BEGIN
		ALTER TABLE terminal_recordings ADD COLUMN IF NOT EXISTS data BYTEA;
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

	if snippetCount >= 30 {
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
		// 🖥️ Standard/System snippets
		{"seed-001", "System Info", "#!/bin/bash\necho '=== System Info ==='\nuname -a\necho ''\necho '=== Memory ==='\nfree -h\necho ''\necho '=== Disk ==='\ndf -h /\necho ''\necho '=== CPU ==='\nlscpu | grep -E '^(Model name|CPU\\(s\\)|Thread)'\necho ''\nneofetch 2>/dev/null || echo 'neofetch not installed'", "bash", "Display comprehensive system information including OS, memory, disk, and CPU details", "🖥️", "system", "", false},
		{"seed-002", "Find Large Files", "#!/bin/bash\necho 'Finding files larger than 100MB...'\nfind . -type f -size +100M -exec ls -lh {} \\; 2>/dev/null | awk '{print $5, $9}' | sort -rh | head -20", "bash", "Find and list the 20 largest files over 100MB in current directory", "📁", "system", "", false},
		{"seed-003", "Process Monitor", "#!/bin/bash\necho 'Top 10 processes by CPU:'\nps aux --sort=-%cpu | head -11\necho ''\necho 'Top 10 processes by Memory:'\nps aux --sort=-%mem | head -11", "bash", "Show top 10 processes by CPU and memory usage", "📊", "system", "", false},
		{"seed-004", "Quick Backup", "#!/bin/bash\nDIR=${1:-.}\nBACKUP_NAME=\"backup_$(date +%Y%m%d_%H%M%S).tar.gz\"\ntar -czvf \"$BACKUP_NAME\" \"$DIR\"\necho \"Backup created: $BACKUP_NAME\"\nls -lh \"$BACKUP_NAME\"", "bash", "Create a timestamped tar.gz backup of current or specified directory", "💾", "system", "", false},
		{"seed-005", "Network Check", "#!/bin/bash\necho '=== Network Interfaces ==='\nip addr show | grep -E '^[0-9]+:|inet '\necho ''\necho '=== Connectivity Test ==='\nping -c 3 8.8.8.8\necho ''\necho '=== DNS Test ==='\nnslookup google.com", "bash", "Check network interfaces, connectivity, and DNS resolution", "🌐", "network", "", false},

		// 📦 Node.js/JavaScript snippets
		{"seed-006", "Node Project Init", "#!/bin/bash\nmkdir -p src tests\nnpm init -y\nnpm install --save-dev typescript @types/node jest ts-jest\ncat > tsconfig.json << 'EOF'\n{\n  \"compilerOptions\": {\n    \"target\": \"ES2020\",\n    \"module\": \"commonjs\",\n    \"outDir\": \"./dist\",\n    \"rootDir\": \"./src\",\n    \"strict\": true\n  }\n}\nEOF\necho 'console.log(\"Hello, TypeScript!\");' > src/index.ts\necho 'Node.js TypeScript project initialized!'", "bash", "Initialize a new Node.js project with TypeScript, Jest, and proper directory structure", "📦", "nodejs", "npm install -g typescript ts-node", true},
		{"seed-007", "NPM Audit Fix", "#!/bin/bash\necho '=== Running npm audit ==='\nnpm audit\necho ''\necho '=== Fixing vulnerabilities ==='\nnpm audit fix\necho ''\necho '=== Checking outdated packages ==='\nnpm outdated", "bash", "Run npm security audit, fix vulnerabilities, and check for outdated packages", "🔒", "nodejs", "", false},
		{"seed-008", "Express API Starter", "const express = require('express');\nconst app = express();\napp.use(express.json());\n\napp.get('/health', (req, res) => res.json({ status: 'ok' }));\n\napp.get('/api/items', (req, res) => {\n  res.json([{ id: 1, name: 'Item 1' }]);\n});\n\napp.post('/api/items', (req, res) => {\n  res.status(201).json({ id: 2, ...req.body });\n});\n\napp.listen(3000, () => console.log('Server running on :3000'));", "javascript", "Minimal Express.js REST API with health check and CRUD endpoints", "🚀", "nodejs", "npm install express", true},
		{"seed-009", "Package.json Scripts", "#!/bin/bash\ncat << 'EOF'\n// Add these to your package.json scripts:\n\"scripts\": {\n  \"dev\": \"nodemon src/index.ts\",\n  \"build\": \"tsc\",\n  \"start\": \"node dist/index.js\",\n  \"test\": \"jest\",\n  \"test:watch\": \"jest --watch\",\n  \"lint\": \"eslint src/**/*.ts\",\n  \"format\": \"prettier --write src/**/*.ts\"\n}\nEOF", "bash", "Common package.json scripts for TypeScript Node.js projects", "📝", "nodejs", "", false},

		// 🐍 Python/Data Science snippets
		{"seed-010", "Python Venv Setup", "#!/bin/bash\npython3 -m venv venv\nsource venv/bin/activate\npip install --upgrade pip\npip install pandas numpy matplotlib jupyter requests\npip freeze > requirements.txt\necho 'Virtual environment created and activated!'", "bash", "Create Python virtual environment with common data science packages", "🐍", "python", "", false},
		{"seed-011", "Data Analysis Template", "import pandas as pd\nimport numpy as np\nimport matplotlib.pyplot as plt\n\n# Load data\ndf = pd.read_csv('data.csv')\n\n# Quick overview\nprint('Shape:', df.shape)\nprint('\\nColumns:', df.columns.tolist())\nprint('\\nData types:')\nprint(df.dtypes)\nprint('\\nSummary statistics:')\nprint(df.describe())\nprint('\\nMissing values:')\nprint(df.isnull().sum())", "python", "Python data analysis template with pandas for quick dataset exploration", "📈", "python", "pip install pandas numpy matplotlib", true},
		{"seed-012", "Flask API Starter", "from flask import Flask, jsonify, request\n\napp = Flask(__name__)\n\nitems = [{'id': 1, 'name': 'Item 1'}]\n\n@app.route('/health')\ndef health():\n    return jsonify({'status': 'ok'})\n\n@app.route('/api/items', methods=['GET'])\ndef get_items():\n    return jsonify(items)\n\n@app.route('/api/items', methods=['POST'])\ndef create_item():\n    item = {'id': len(items) + 1, **request.json}\n    items.append(item)\n    return jsonify(item), 201\n\nif __name__ == '__main__':\n    app.run(debug=True, port=5000)", "python", "Minimal Flask REST API with health check and CRUD endpoints", "🌶️", "python", "pip install flask", true},
		{"seed-013", "Jupyter Setup", "#!/bin/bash\npip install jupyterlab ipykernel\npython -m ipykernel install --user --name=myenv\njupyter lab --ip=0.0.0.0 --port=8888 --no-browser --allow-root", "bash", "Install JupyterLab and start it for remote access", "📓", "python", "pip install jupyterlab", true},

		// 🐹 Go/Gopher snippets
		{"seed-014", "Go Module Init", "#!/bin/bash\ngo mod init myproject\nmkdir -p cmd/myapp internal pkg\ncat > cmd/myapp/main.go << 'EOF'\npackage main\n\nimport \"fmt\"\n\nfunc main() {\n\tfmt.Println(\"Hello, Go!\")\n}\nEOF\ngo mod tidy\ngo build -o bin/myapp ./cmd/myapp\necho 'Go project initialized!'", "bash", "Initialize a Go module with standard project layout", "🐹", "golang", "", false},
		{"seed-015", "Go HTTP Server", "package main\n\nimport (\n\t\"encoding/json\"\n\t\"log\"\n\t\"net/http\"\n)\n\nfunc main() {\n\thttp.HandleFunc(\"/health\", func(w http.ResponseWriter, r *http.Request) {\n\t\tjson.NewEncoder(w).Encode(map[string]string{\"status\": \"ok\"})\n\t})\n\n\thttp.HandleFunc(\"/api/items\", func(w http.ResponseWriter, r *http.Request) {\n\t\titems := []map[string]interface{}{{\"id\": 1, \"name\": \"Item 1\"}}\n\t\tw.Header().Set(\"Content-Type\", \"application/json\")\n\t\tjson.NewEncoder(w).Encode(items)\n\t})\n\n\tlog.Println(\"Server starting on :8080\")\n\tlog.Fatal(http.ListenAndServe(\":8080\", nil))\n}", "go", "Simple Go HTTP server with JSON endpoints", "🌐", "golang", "", false},
		{"seed-016", "Go Test Template", "package mypackage\n\nimport \"testing\"\n\nfunc TestAdd(t *testing.T) {\n\ttests := []struct {\n\t\tname     string\n\t\ta, b     int\n\t\texpected int\n\t}{\n\t\t{\"positive\", 2, 3, 5},\n\t\t{\"negative\", -1, -1, -2},\n\t\t{\"zero\", 0, 0, 0},\n\t}\n\n\tfor _, tt := range tests {\n\t\tt.Run(tt.name, func(t *testing.T) {\n\t\t\tresult := Add(tt.a, tt.b)\n\t\t\tif result != tt.expected {\n\t\t\t\tt.Errorf(\"Add(%d, %d) = %d; want %d\", tt.a, tt.b, result, tt.expected)\n\t\t\t}\n\t\t})\n\t}\n}", "go", "Go table-driven test template with subtests", "🧪", "golang", "", false},
		{"seed-017", "Go Build All", "#!/bin/bash\necho 'Building for multiple platforms...'\nGOOS=linux GOARCH=amd64 go build -o bin/app-linux-amd64 ./cmd/app\nGOOS=darwin GOARCH=amd64 go build -o bin/app-darwin-amd64 ./cmd/app\nGOOS=windows GOARCH=amd64 go build -o bin/app-windows-amd64.exe ./cmd/app\nls -la bin/\necho 'Cross-compilation complete!'", "bash", "Cross-compile Go application for Linux, macOS, and Windows", "🔨", "golang", "", false},

		// ✏️ Neovim/Editor snippets
		{"seed-018", "Neovim Config Check", "#!/bin/bash\necho '=== Neovim Version ==='\nnvim --version | head -5\necho ''\necho '=== Config Location ==='\nls -la ~/.config/nvim/ 2>/dev/null || echo 'No config found'\necho ''\necho '=== Installed Plugins ==='\nls ~/.local/share/nvim/site/pack/ 2>/dev/null || echo 'No plugins found'", "bash", "Check Neovim version, config, and installed plugins", "✏️", "editor", "", false},
		{"seed-019", "LazyVim Setup", "#!/bin/bash\n# Backup existing config\nmv ~/.config/nvim ~/.config/nvim.bak 2>/dev/null\nmv ~/.local/share/nvim ~/.local/share/nvim.bak 2>/dev/null\n\n# Clone LazyVim starter\ngit clone https://github.com/LazyVim/starter ~/.config/nvim\nrm -rf ~/.config/nvim/.git\n\necho 'LazyVim installed! Run nvim to complete setup.'", "bash", "Install LazyVim - a modern Neovim configuration", "🚀", "editor", "apt-get install -y neovim || brew install neovim", true},
		{"seed-020", "Ripgrep Search", "#!/bin/bash\n# Usage: ./script.sh 'pattern' [path]\nPATTERN=\"${1:-TODO}\"\nPATH=\"${2:-.}\"\n\necho \"Searching for '$PATTERN' in $PATH\"\nrg --color=always --line-number --heading \"$PATTERN\" \"$PATH\" | head -100", "bash", "Fast code search with ripgrep - colorized output with line numbers", "🔍", "editor", "apt-get install -y ripgrep || brew install ripgrep", true},

		// ☸️ DevOps/YAML Herder snippets
		{"seed-021", "Docker Cleanup", "#!/bin/bash\necho '=== Removing stopped containers ==='\ndocker container prune -f\necho ''\necho '=== Removing unused images ==='\ndocker image prune -a -f\necho ''\necho '=== Removing unused volumes ==='\ndocker volume prune -f\necho ''\necho '=== Disk usage ==='\ndocker system df", "bash", "Clean up Docker resources - containers, images, volumes", "🐳", "devops", "", false},
		{"seed-022", "Kubernetes Debug", "#!/bin/bash\nNS=${1:-default}\necho \"=== Pods in $NS ===\"\nkubectl get pods -n $NS\necho ''\necho '=== Recent Events ==='\nkubectl get events -n $NS --sort-by='.lastTimestamp' | tail -20\necho ''\necho '=== Failed Pods ==='\nkubectl get pods -n $NS --field-selector=status.phase!=Running,status.phase!=Succeeded", "bash", "Debug Kubernetes namespace - show pods, events, and failures", "☸️", "devops", "curl -LO https://dl.k8s.io/release/$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/linux/amd64/kubectl && chmod +x kubectl && mv kubectl /usr/local/bin/", true},
		{"seed-023", "Terraform Init", "#!/bin/bash\nterraform init\nterraform validate\nterraform fmt -recursive\nterraform plan -out=tfplan\necho ''\necho 'Review the plan above. Run: terraform apply tfplan'", "bash", "Initialize, validate, format, and plan Terraform configuration", "🏗️", "devops", "apt-get install -y terraform || brew install terraform", true},
		{"seed-024", "Docker Compose Template", "version: '3.8'\n\nservices:\n  app:\n    build: .\n    ports:\n      - \"3000:3000\"\n    environment:\n      - NODE_ENV=development\n    volumes:\n      - .:/app\n      - /app/node_modules\n    depends_on:\n      - db\n      - redis\n\n  db:\n    image: postgres:15-alpine\n    environment:\n      POSTGRES_PASSWORD: secret\n      POSTGRES_DB: myapp\n    volumes:\n      - postgres_data:/var/lib/postgresql/data\n\n  redis:\n    image: redis:7-alpine\n\nvolumes:\n  postgres_data:", "yaml", "Docker Compose template with app, PostgreSQL, and Redis", "🐳", "devops", "", false},
		{"seed-025", "Ansible Ping", "#!/bin/bash\nansible all -m ping -i inventory.ini\necho ''\necho '=== Host Facts ==='\nansible all -m setup -i inventory.ini -a 'filter=ansible_distribution*' | head -50", "bash", "Ansible ping all hosts and gather basic facts", "📡", "devops", "pip install ansible", true},

		// 🤖 AI/Vibe Coder snippets
		{"seed-026", "Claude Code", "#!/bin/bash\necho 'Starting Claude Code AI assistant...'\necho ''\necho 'Tips:'\necho '  - Use /help to see available commands'\necho '  - Describe what you want to build'\necho '  - Press Ctrl+C to exit'\necho ''\nclaude", "bash", "Launch Claude Code AI coding assistant", "🤖", "ai", "npm install -g @anthropic-ai/claude-code", true},
		{"seed-027", "GitHub Copilot CLI", "#!/bin/bash\necho 'GitHub Copilot CLI commands:'\necho '  gh copilot suggest \"how to...\"  - Get command suggestions'\necho '  gh copilot explain \"command\"    - Explain a command'\necho ''\ngh copilot suggest \"find files modified in last 24 hours\"", "bash", "Use GitHub Copilot CLI for command suggestions and explanations", "🐙", "ai", "gh extension install github/gh-copilot", true},
		{"seed-028", "OpenCode Quick Start", "#!/bin/bash\necho 'Starting OpenCode AI coding assistant...'\necho ''\necho 'Tips:'\necho '  - Use /help to see available commands'\necho '  - Use /compact for token-efficient mode'\necho '  - Press Ctrl+C to exit'\necho ''\nopencode", "bash", "Launch OpenCode AI assistant with helpful tips", "💻", "ai", "go install github.com/opencode-ai/opencode@latest", true},
		{"seed-029", "Aider Code Review", "#!/bin/bash\n# Use Aider for AI pair programming\necho 'Starting Aider AI pair programmer...'\necho ''\necho 'Tips:'\necho '  - Add files with /add filename'\necho '  - Ask for changes naturally'\necho '  - Use /diff to see changes'\necho ''\naider", "bash", "Launch Aider AI pair programming assistant", "👨‍💻", "ai", "pip install aider-chat", true},
		{"seed-030", "Gemini CLI", "#!/bin/bash\necho 'Gemini CLI - Google AI in your terminal'\necho ''\necho 'Usage:'\necho '  gemini \"your question here\"'\necho '  echo \"code\" | gemini \"explain this\"'\necho ''\ngemini \"What can you help me with?\"", "bash", "Use Google Gemini AI from the command line", "💎", "ai", "npm install -g @anthropic-ai/claude-code && pip install google-generativeai", true},
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
	query := `
		INSERT INTO users (id, email, username, password_hash, tier, is_admin, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`
	_, err := s.db.ExecContext(ctx, query,
		user.ID,
		user.Email,
		user.Username,
		passwordHash,
		user.Tier,
		user.IsAdmin,
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
		SELECT id, email, username, COALESCE(password_hash, ''), tier, COALESCE(is_admin, false),
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
		&user.IsAdmin,
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
		SELECT id, email, username, tier, COALESCE(is_admin, false), COALESCE(pipeops_id, ''), created_at, updated_at
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
		UPDATE users SET username = $2, tier = $3, is_admin = $4, pipeops_id = $5, updated_at = $6
		WHERE id = $1
	`
	_, err := s.db.ExecContext(ctx, query,
		user.ID,
		user.Username,
		user.Tier,
		user.IsAdmin,
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
	Role       string    `db:"role"`
	Status     string    `db:"status"`
	DockerID   string    `db:"docker_id"`
	VolumeName string    `db:"volume_name"`
	MemoryMB   int64     `db:"memory_mb"`
	CPUShares  int64     `db:"cpu_shares"`
	DiskMB     int64     `db:"disk_mb"`
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
	query := `
		SELECT id, user_id, name, image, COALESCE(role, 'standard') as role, status, docker_id, volume_name,
		       COALESCE(memory_mb, 512) as memory_mb, COALESCE(cpu_shares, 512) as cpu_shares, COALESCE(disk_mb, 2048) as disk_mb,
		       created_at, last_used_at
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
		       created_at, last_used_at
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
		       created_at, last_used_at
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
		       created_at, last_used_at
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
		       created_at, last_used_at
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
	Duration    int64      `db:"duration_ms"`    // Duration in milliseconds
	Size        int64      `db:"size_bytes"`     // Size of recording data
	Data        []byte     `db:"data"`           // Recording data (gzipped asciicast)
	ShareToken  string     `db:"share_token"`    // Public share link token
	IsPublic    bool       `db:"is_public"`      // Whether publicly accessible
	CreatedAt   time.Time  `db:"created_at"`
	ExpiresAt   *time.Time `db:"expires_at"`     // Optional expiration
}

// CreateRecording creates a new recording record
func (s *PostgresStore) CreateRecording(ctx context.Context, rec *RecordingRecord) error {
	query := `
		INSERT INTO terminal_recordings (id, user_id, container_id, title, duration_ms, size_bytes, data, share_token, is_public, created_at, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
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
	)
	return err
}

// GetRecordingsByUserID retrieves all recordings for a user
func (s *PostgresStore) GetRecordingsByUserID(ctx context.Context, userID string) ([]*RecordingRecord, error) {
	query := `
		SELECT id, user_id, container_id, title, duration_ms, size_bytes, share_token, is_public, created_at, expires_at
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
		SELECT id, user_id, container_id, title, duration_ms, size_bytes, share_token, is_public, created_at, expires_at
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
		SELECT id, user_id, container_id, title, duration_ms, size_bytes, share_token, is_public, created_at, expires_at
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
	ShareCode   string    `db:"share_code"`      // Short code for joining (e.g., "ABC123")
	Mode        string    `db:"mode"`            // "view" or "control"
	MaxUsers    int       `db:"max_users"`
	IsActive    bool      `db:"is_active"`
	CreatedAt   time.Time `db:"created_at"`
	ExpiresAt   time.Time `db:"expires_at"`
}

// CollabParticipantRecord represents a participant in a collab session
type CollabParticipantRecord struct {
	ID        string    `db:"id"`
	SessionID string    `db:"session_id"`
	UserID    string    `db:"user_id"`
	Username  string    `db:"username"`
	Role      string    `db:"role"`       // "owner", "editor", "viewer"
	JoinedAt  time.Time `db:"joined_at"`
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

// AddCollabParticipant adds a participant to a collab session
func (s *PostgresStore) AddCollabParticipant(ctx context.Context, p *CollabParticipantRecord) error {
	query := `
		INSERT INTO collab_participants (id, session_id, user_id, username, role, joined_at)
		VALUES ($1, $2, $3, $4, $5, $6)
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
func (s *PostgresStore) GetPublicSnippets(ctx context.Context, language, search, sort string) ([]*models.Snippet, error) {
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
		}

		snippets = append(snippets, &sn)
	}
	return snippets, nil
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
