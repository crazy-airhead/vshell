package db

func (db *DB) migrate() error {
	migrations := []string{
		`CREATE TABLE IF NOT EXISTS groups (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			parent_id TEXT REFERENCES groups(id) ON DELETE CASCADE,
			sort_order INTEGER DEFAULT 0,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS connections (
			id TEXT PRIMARY KEY,
			group_id TEXT REFERENCES groups(id) ON DELETE SET NULL,
			name TEXT NOT NULL,
			host TEXT NOT NULL,
			port INTEGER DEFAULT 22,
			username TEXT NOT NULL,
			auth_type TEXT NOT NULL,
			password TEXT,
			private_key TEXT,
			key_passphrase TEXT,
			proxy_type TEXT,
			proxy_addr TEXT,
			jump_host_id TEXT,
			upload_path TEXT DEFAULT '/',
			default_cmd TEXT,
			sort_order INTEGER DEFAULT 0,
			color TEXT,
			last_used_at DATETIME,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS quick_commands (
			id TEXT PRIMARY KEY,
			name TEXT,
			command TEXT NOT NULL,
			connection_id TEXT,
			sort_order INTEGER DEFAULT 0,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS port_forwards (
			id TEXT PRIMARY KEY,
			connection_id TEXT NOT NULL,
			type TEXT NOT NULL,
			local_host TEXT,
			local_port INTEGER,
			remote_host TEXT,
			remote_port INTEGER,
			auto_start INTEGER DEFAULT 0
		)`,
	}

	for _, m := range migrations {
		if _, err := db.Exec(m); err != nil {
			return err
		}
	}
	return nil
}
