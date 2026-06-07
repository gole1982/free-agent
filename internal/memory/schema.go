package memory

import (
	"database/sql"
	_ "modernc.org/sqlite"
)

func initDB(db *sql.DB) error {
	schema := `
CREATE TABLE IF NOT EXISTS conversations (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	title TEXT NOT NULL,
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS messages (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	conversation_id INTEGER NOT NULL,
	role TEXT NOT NULL,
	content TEXT NOT NULL,
	timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
	FOREIGN KEY (conversation_id) REFERENCES conversations(id)
);

CREATE TABLE IF NOT EXISTS agent_traits (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	agent_name TEXT NOT NULL UNIQUE,
	efficiency REAL DEFAULT 0.5,
	quality REAL DEFAULT 0.5,
	creativity REAL DEFAULT 0.5,
	collaboration REAL DEFAULT 0.5,
	updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS security_policies (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	policy_type TEXT NOT NULL,
	policy_content TEXT NOT NULL,
	enabled INTEGER DEFAULT 1,
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
	updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS agent_executions (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	conversation_id INTEGER NOT NULL,
	agent_name TEXT NOT NULL,
	task TEXT NOT NULL,
	result TEXT,
	status TEXT,
	started_at DATETIME DEFAULT CURRENT_TIMESTAMP,
	completed_at DATETIME,
	FOREIGN KEY (conversation_id) REFERENCES conversations(id)
);
`
	_, err := db.Exec(schema)
	return err
}
