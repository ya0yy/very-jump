package database

import (
	"database/sql"
	"log"
	_ "modernc.org/sqlite"
	
)

// Init 初始化数据库
func Init(dbPath string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", dbPath+"?_journal_mode=WAL&_synchronous=NORMAL")
	if err != nil {
		return nil, err
	}

	// 测试连接
	if err := db.Ping(); err != nil {
		return nil, err
	}

	// 运行迁移
	if err := runMigrations(db); err != nil {
		return nil, err
	}

	log.Println("Database initialized successfully")
	return db, nil
}

// runMigrations 运行数据库迁移
func runMigrations(db *sql.DB) error {
	migrations := []string{
		createUsersTable,
		createServersTable,
		createUserServerPermissionsTable,
		createSessionsTable,
		createAuditLogsTable,
		alterAuditLogsAddSuccessColumn, // 添加 success 列
		createExtraAuditTables,         // 新的审计表
		insertDefaultAdmin,
	}

	for _, migration := range migrations {
		if _, err := db.Exec(migration); err != nil {
			// Ignore "duplicate column name" error
			if strings.Contains(err.Error(), "duplicate column name") {
				log.Printf("Ignoring duplicate column error: %v", err)
				continue
			}
			return err
		}
	}

	return nil
}

const createUsersTable = `
CREATE TABLE IF NOT EXISTS users (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    username VARCHAR(50) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    role VARCHAR(20) DEFAULT 'user',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
`

const createServersTable = `
CREATE TABLE IF NOT EXISTS servers (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name VARCHAR(100) NOT NULL,
    host VARCHAR(255) NOT NULL,
    port INTEGER DEFAULT 22,
    username VARCHAR(100) NOT NULL,
    auth_type VARCHAR(20) DEFAULT 'password',
    password VARCHAR(255),
    private_key TEXT,
    description TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
`

const createUserServerPermissionsTable = `
CREATE TABLE IF NOT EXISTS user_server_permissions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL,
    server_id INTEGER NOT NULL,
    permission VARCHAR(20) DEFAULT 'read',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id),
    FOREIGN KEY (server_id) REFERENCES servers(id),
    UNIQUE(user_id, server_id)
);
`

const createSessionsTable = `
CREATE TABLE IF NOT EXISTS sessions (
    id VARCHAR(36) PRIMARY KEY,
    user_id INTEGER NOT NULL,
    server_id INTEGER NOT NULL,
    start_time DATETIME DEFAULT CURRENT_TIMESTAMP,
    end_time DATETIME,
    status VARCHAR(20) DEFAULT 'active',
    client_ip VARCHAR(45),
    recording_file VARCHAR(255),
    FOREIGN KEY (user_id) REFERENCES users(id),
    FOREIGN KEY (server_id) REFERENCES servers(id)
);
`

const createAuditLogsTable = `
CREATE TABLE IF NOT EXISTS audit_logs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL,
    action VARCHAR(100) NOT NULL,
    resource_type VARCHAR(50),
    resource_id VARCHAR(100),
    details TEXT,
    ip_address VARCHAR(45),
    user_agent TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id)
);
`

const alterAuditLogsAddSuccessColumn = `
-- Add success column to audit_logs table
ALTER TABLE audit_logs ADD COLUMN success BOOLEAN DEFAULT TRUE;
`

const createExtraAuditTables = `
-- 终端会话统计表
CREATE TABLE IF NOT EXISTS terminal_sessions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    session_id TEXT UNIQUE NOT NULL,
    user_id INTEGER NOT NULL,
    server_id INTEGER NOT NULL,
    start_time DATETIME NOT NULL,
    end_time DATETIME,
    duration INTEGER DEFAULT 0,
    command_count INTEGER DEFAULT 0,
    ip_address TEXT,
    status TEXT DEFAULT 'active',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id),
    FOREIGN KEY (server_id) REFERENCES servers(id)
);

-- 安全告警表
CREATE TABLE IF NOT EXISTS security_alerts (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER,
    server_id INTEGER,
    alert_type TEXT NOT NULL,
    severity TEXT NOT NULL DEFAULT 'medium',
    description TEXT NOT NULL,
    details TEXT,
    ip_address TEXT,
    session_id TEXT,
    resolved BOOLEAN DEFAULT FALSE,
    resolved_by INTEGER,
    resolved_at DATETIME,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id),
    FOREIGN KEY (server_id) REFERENCES servers(id),
    FOREIGN KEY (resolved_by) REFERENCES users(id)
);

-- 创建索引
CREATE INDEX IF NOT EXISTS idx_audit_logs_user_id ON audit_logs(user_id);
CREATE INDEX IF NOT EXISTS idx_audit_logs_action ON audit_logs(action);
CREATE INDEX IF NOT EXISTS idx_audit_logs_created_at ON audit_logs(created_at);
CREATE INDEX IF NOT EXISTS idx_terminal_sessions_user_id ON terminal_sessions(user_id);
CREATE INDEX IF NOT EXISTS idx_terminal_sessions_status ON terminal_sessions(status);
CREATE INDEX IF NOT EXISTS idx_security_alerts_resolved ON security_alerts(resolved);
`

const insertDefaultAdmin = `
INSERT OR IGNORE INTO users (username, password_hash, role) 
VALUES ('admin', '$2a$10$u4V8qHD8a4YyP0ylvUjAb.hhJ8KdhJ32.rV1jOcxyAoinpJu64vo2', 'admin');
`
