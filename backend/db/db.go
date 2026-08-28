// Package db owns SQLite connection handling and schema migrations for the
// application store.
package db

import (
	"database/sql"
	"fmt"
	"os"
	"time"

	_ "modernc.org/sqlite"
)

// schemaV1 creates the initial tables: a key/value settings document,
// ordered CNAME presets and monitor projects with their probe targets.
const schemaV1 = `CREATE TABLE app_settings (
	key   TEXT PRIMARY KEY,
	value TEXT NOT NULL
);

CREATE TABLE cname_presets (
	position INTEGER PRIMARY KEY,
	name     TEXT NOT NULL,
	value    TEXT NOT NULL
);

CREATE TABLE monitors (
	id              TEXT PRIMARY KEY,
	position        INTEGER NOT NULL DEFAULT 0,
	name            TEXT NOT NULL,
	interval_sec    INTEGER NOT NULL DEFAULT 60,
	publish_enabled INTEGER NOT NULL DEFAULT 0,
	public_token    TEXT NOT NULL DEFAULT '',
	public_slug     TEXT NOT NULL DEFAULT '',
	public_title    TEXT NOT NULL DEFAULT '',
	public_icon     TEXT NOT NULL DEFAULT '',
	public_theme    TEXT NOT NULL DEFAULT '',
	announcement    TEXT NOT NULL DEFAULT '',
	created_at      INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE monitor_targets (
	id           TEXT PRIMARY KEY,
	monitor_id   TEXT NOT NULL REFERENCES monitors(id) ON DELETE CASCADE,
	position     INTEGER NOT NULL DEFAULT 0,
	name         TEXT NOT NULL DEFAULT '',
	url          TEXT NOT NULL DEFAULT '',
	type         TEXT NOT NULL DEFAULT '',
	method       TEXT NOT NULL DEFAULT '',
	created_at   INTEGER NOT NULL DEFAULT 0,
	link_enabled INTEGER NOT NULL DEFAULT 0
);

CREATE INDEX idx_monitor_targets_monitor ON monitor_targets(monitor_id);
`

// schemaV2 adds multi-user support: users with per-user TOTP state and
// preferences, database-backed sessions, user groups, invite codes and
// email verification codes.
const schemaV2 = `ALTER TABLE monitors ADD COLUMN user_id TEXT NOT NULL DEFAULT '';

CREATE TABLE user_groups (
	id          TEXT PRIMARY KEY,
	name        TEXT NOT NULL UNIQUE,
	permissions TEXT NOT NULL DEFAULT '',
	builtin     INTEGER NOT NULL DEFAULT 0,
	created_at  INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE users (
	id                        TEXT PRIMARY KEY,
	username                  TEXT NOT NULL UNIQUE,
	email                     TEXT NOT NULL DEFAULT '',
	password_hash             TEXT NOT NULL,
	role                      TEXT NOT NULL DEFAULT 'user',
	group_id                  TEXT NOT NULL DEFAULT '',
	status                    TEXT NOT NULL DEFAULT 'active',
	email_verified            INTEGER NOT NULL DEFAULT 0,
	totp_enabled              INTEGER NOT NULL DEFAULT 0,
	totp_secret_encrypted     TEXT NOT NULL DEFAULT '',
	totp_last_accepted_step   INTEGER NOT NULL DEFAULT 0,
	totp_recovery_code_hashes TEXT NOT NULL DEFAULT '',
	created_at                INTEGER NOT NULL DEFAULT 0,
	last_login_at             INTEGER NOT NULL DEFAULT 0
);

CREATE UNIQUE INDEX idx_users_email ON users(email) WHERE email != '';

CREATE TABLE user_prefs (
	user_id            TEXT PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
	tunnel_id          TEXT NOT NULL DEFAULT '',
	tunnel_name        TEXT NOT NULL DEFAULT '',
	service_url        TEXT NOT NULL DEFAULT '',
	selected_zone_id   TEXT NOT NULL DEFAULT '',
	selected_zone_name TEXT NOT NULL DEFAULT ''
);

CREATE TABLE sessions (
	token_hash TEXT PRIMARY KEY,
	user_id    TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
	created_at INTEGER NOT NULL DEFAULT 0,
	expires_at INTEGER NOT NULL DEFAULT 0
);

CREATE INDEX idx_sessions_user ON sessions(user_id);
CREATE INDEX idx_sessions_expiry ON sessions(expires_at);

CREATE TABLE invites (
	code       TEXT PRIMARY KEY,
	group_id   TEXT NOT NULL DEFAULT '',
	max_uses   INTEGER NOT NULL DEFAULT 0,
	used_count INTEGER NOT NULL DEFAULT 0,
	expires_at INTEGER NOT NULL DEFAULT 0,
	enabled    INTEGER NOT NULL DEFAULT 1,
	created_at INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE verify_codes (
	email      TEXT NOT NULL,
	purpose    TEXT NOT NULL,
	code_hash  TEXT NOT NULL,
	created_at INTEGER NOT NULL DEFAULT 0,
	expires_at INTEGER NOT NULL DEFAULT 0,
	PRIMARY KEY (email, purpose)
);
`

type migration struct {
	version int
	stmts   string
}

// migrations lists every schema version in order. Append new entries instead
// of editing existing ones.
var migrations = []migration{
	{version: 1, stmts: schemaV1},
	{version: 2, stmts: schemaV2},
}

// Open opens (creating when missing) the SQLite database at path with the
// pragmas used across the application. The database file is created with
// 0600 permissions. Callers are responsible for ensuring the parent
// directory exists.
func Open(path string) (*sql.DB, error) {
	dsn := "file:" + path + "?_pragma=busy_timeout(10000)&_pragma=journal_mode(WAL)&_pragma=foreign_keys(1)&_pragma=synchronous(NORMAL)"
	handle, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}
	// Single connection: the store serializes writes itself and SQLite is
	// fastest here without intra-process write contention.
	handle.SetMaxOpenConns(1)
	if err := handle.Ping(); err != nil {
		handle.Close()
		return nil, fmt.Errorf("open database %s: %w", path, err)
	}
	// SQLite creates files with process-default permissions; tighten the
	// main file (best effort, WAL sidecars included when present).
	_ = os.Chmod(path, 0o600)
	_ = os.Chmod(path+"-wal", 0o600)
	_ = os.Chmod(path+"-shm", 0o600)
	return handle, nil
}

// Migrate applies pending schema migrations inside per-version transactions.
func Migrate(handle *sql.DB) error {
	if _, err := handle.Exec(`CREATE TABLE IF NOT EXISTS schema_migrations (
		version    INTEGER PRIMARY KEY,
		applied_at INTEGER NOT NULL
	)`); err != nil {
		return fmt.Errorf("create migrations table: %w", err)
	}

	var current int
	if err := handle.QueryRow(`SELECT COALESCE(MAX(version), 0) FROM schema_migrations`).Scan(&current); err != nil {
		return fmt.Errorf("read schema version: %w", err)
	}

	for _, m := range migrations {
		if m.version <= current {
			continue
		}
		tx, err := handle.Begin()
		if err != nil {
			return fmt.Errorf("begin migration %d: %w", m.version, err)
		}
		if _, err := tx.Exec(m.stmts); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("apply migration %d: %w", m.version, err)
		}
		if _, err := tx.Exec(`INSERT INTO schema_migrations(version, applied_at) VALUES(?, ?)`, m.version, time.Now().Unix()); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("record migration %d: %w", m.version, err)
		}
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("commit migration %d: %w", m.version, err)
		}
	}
	return nil
}
