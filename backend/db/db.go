// Package db owns database connection handling and schema migrations for the
// application store. SQLite is the default backend; a PostgreSQL server can be
// used instead by pointing the store at a postgres:// URL.
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

// schemaV4 adds monitor alerting: per-monitor alert configuration, the
// persisted last probe state per target (edge detection across restarts)
// and an alert delivery log.
const schemaV4 = `ALTER TABLE monitors ADD COLUMN alert_enabled INTEGER NOT NULL DEFAULT 0;
ALTER TABLE monitors ADD COLUMN alert_emails TEXT NOT NULL DEFAULT '';
ALTER TABLE monitor_targets ADD COLUMN last_state TEXT NOT NULL DEFAULT '';

CREATE TABLE alert_logs (
	id          INTEGER PRIMARY KEY AUTOINCREMENT,
	monitor_id  TEXT NOT NULL,
	target_id   TEXT NOT NULL,
	target_name TEXT NOT NULL DEFAULT '',
	state       TEXT NOT NULL DEFAULT '',
	http_code   INTEGER NOT NULL DEFAULT 0,
	error       TEXT NOT NULL DEFAULT '',
	notified    INTEGER NOT NULL DEFAULT 0,
	detail      TEXT NOT NULL DEFAULT '',
	created_at  INTEGER NOT NULL DEFAULT 0
);

CREATE INDEX idx_alert_logs_monitor ON alert_logs(monitor_id, created_at);
`

// schemaV3 adds per-user Cloudflare OAuth connections so one account can
// authorize several Cloudflare accounts and switch between them.
const schemaV3 = `CREATE TABLE cf_connections (
	id            TEXT PRIMARY KEY,
	user_id       TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
	label         TEXT NOT NULL DEFAULT '',
	account_id    TEXT NOT NULL DEFAULT '',
	account_name  TEXT NOT NULL DEFAULT '',
	access_token  TEXT NOT NULL DEFAULT '',
	refresh_token TEXT NOT NULL DEFAULT '',
	expires_at    INTEGER NOT NULL DEFAULT 0,
	scope         TEXT NOT NULL DEFAULT '',
	created_at    INTEGER NOT NULL DEFAULT 0
);

CREATE INDEX idx_cf_connections_user ON cf_connections(user_id);

ALTER TABLE users ADD COLUMN active_cf_connection_id TEXT NOT NULL DEFAULT '';
`

// schemaV5 adds per-user profile fields: an optional display nickname and
// a custom avatar URL (typically an uploaded /uploads/... image).
const schemaV5 = `ALTER TABLE users ADD COLUMN nickname TEXT NOT NULL DEFAULT '';
ALTER TABLE users ADD COLUMN avatar TEXT NOT NULL DEFAULT '';`

// schemaV6 adds per-user notification preferences: delivery channels,
// enabled events, multiple recipient emails and a per-user Telegram bot.
const schemaV6 = `ALTER TABLE user_prefs ADD COLUMN notify_channels TEXT NOT NULL DEFAULT '';
ALTER TABLE user_prefs ADD COLUMN notify_events TEXT NOT NULL DEFAULT '';
ALTER TABLE user_prefs ADD COLUMN notify_emails TEXT NOT NULL DEFAULT '';
ALTER TABLE user_prefs ADD COLUMN tg_bot_token_encrypted TEXT NOT NULL DEFAULT '';
ALTER TABLE user_prefs ADD COLUMN tg_notify_chat_id TEXT NOT NULL DEFAULT '';`

// schemaV7 adds per-user Telegram remote-control preferences: an enable
// flag, the Telegram user IDs allowed to operate the bot, and a per-user
// preferred CNAME (mirrors the global Telegram bot fields).
const schemaV7 = `ALTER TABLE user_prefs ADD COLUMN tg_remote_enabled INTEGER NOT NULL DEFAULT 0;
ALTER TABLE user_prefs ADD COLUMN tg_operator_ids TEXT NOT NULL DEFAULT '';
ALTER TABLE user_prefs ADD COLUMN preferred_cname TEXT NOT NULL DEFAULT '';`

// schemaV8 separates the notification bot token from the remote-control bot
// token, so an account can use two different bots (or reuse one for both).
const schemaV8 = `ALTER TABLE user_prefs ADD COLUMN tg_remote_token_encrypted TEXT NOT NULL DEFAULT '';`

// schemaV9 adds an optional custom domain per monitor: visitors reaching the
// panel with this Host are routed to the monitor's public status page.
const schemaV9 = `ALTER TABLE monitors ADD COLUMN public_domain TEXT NOT NULL DEFAULT '';`

// schemaV10 records whether a monitor custom domain uses a direct proxied
// Tunnel CNAME or the SaaS custom-hostname preferred route.
const schemaV10 = `ALTER TABLE monitors ADD COLUMN public_domain_mode TEXT NOT NULL DEFAULT 'simple';`

// schemaV11 adds the preferred-mode auxiliary origin hostname and an optional
// per-monitor preferred CNAME override.
const schemaV11 = `ALTER TABLE monitors ADD COLUMN public_aux_domain TEXT NOT NULL DEFAULT '';
ALTER TABLE monitors ADD COLUMN public_preferred_cname TEXT NOT NULL DEFAULT '';`

// schemaV12 adds per-user Telegram remote-control mode and webhook settings:
// polling (default) or webhook, plus the public HTTPS base URL and the
// generated verification secret. The secret never leaves the backend.
const schemaV12 = `ALTER TABLE user_prefs ADD COLUMN tg_remote_mode TEXT NOT NULL DEFAULT 'polling';
ALTER TABLE user_prefs ADD COLUMN tg_webhook_url TEXT NOT NULL DEFAULT '';
ALTER TABLE user_prefs ADD COLUMN tg_webhook_secret TEXT NOT NULL DEFAULT '';`

// schemaV13 adds the administrator audit trail: one append-only row per
// mutating operation, filterable by account, action and time range.
const schemaV13 = `CREATE TABLE audit_logs (
	id         INTEGER PRIMARY KEY AUTOINCREMENT,
	created_at INTEGER NOT NULL DEFAULT 0,
	actor_id   TEXT NOT NULL DEFAULT '',
	actor_name TEXT NOT NULL DEFAULT '',
	category   TEXT NOT NULL DEFAULT '',
	action     TEXT NOT NULL DEFAULT '',
	target     TEXT NOT NULL DEFAULT '',
	ip         TEXT NOT NULL DEFAULT '',
	success    INTEGER NOT NULL DEFAULT 1
);

CREATE INDEX idx_audit_logs_created ON audit_logs(created_at);
CREATE INDEX idx_audit_logs_actor ON audit_logs(actor_id, created_at);
CREATE INDEX idx_audit_logs_action ON audit_logs(category, action, created_at);
`

// schemaV14 adds passkeys (WebAuthn credentials) and the per-account
// password-login switch. The passkeys table deliberately carries no foreign key
// to users: saveLocked rewrites the whole users table (DELETE + INSERT) on every
// configuration change, which would cascade-delete every bound credential.
const schemaV14 = `CREATE TABLE passkeys (
	id           TEXT PRIMARY KEY,
	user_id      TEXT NOT NULL,
	name         TEXT NOT NULL DEFAULT '',
	credential   TEXT NOT NULL DEFAULT '',
	created_at   INTEGER NOT NULL DEFAULT 0,
	last_used_at INTEGER NOT NULL DEFAULT 0
);

CREATE INDEX idx_passkeys_user ON passkeys(user_id);

ALTER TABLE users ADD COLUMN password_login_disabled INTEGER NOT NULL DEFAULT 0;
`

// schemaV15 records the networks each account has signed in from, so a later
// attempt from the same neighbourhood can be told apart from one that is not.
// Like passkeys it carries no foreign key to users: saveLocked rewrites the
// users table wholesale, which would cascade-delete every learned network.
const schemaV15 = `CREATE TABLE familiar_ips (
	user_id    TEXT NOT NULL,
	subnet     TEXT NOT NULL,
	first_seen INTEGER NOT NULL DEFAULT 0,
	last_seen  INTEGER NOT NULL DEFAULT 0,
	seen_count INTEGER NOT NULL DEFAULT 0,
	PRIMARY KEY (user_id, subnet)
);

CREATE INDEX idx_familiar_ips_user ON familiar_ips(user_id);
`

var migrations = []migration{
	{version: 1, stmts: schemaV1},
	{version: 2, stmts: schemaV2},
	{version: 3, stmts: schemaV3},
	{version: 4, stmts: schemaV4},
	{version: 5, stmts: schemaV5},
	{version: 6, stmts: schemaV6},
	{version: 7, stmts: schemaV7},
	{version: 8, stmts: schemaV8},
	{version: 9, stmts: schemaV9},
	{version: 10, stmts: schemaV10},
	{version: 11, stmts: schemaV11},
	{version: 12, stmts: schemaV12},
	{version: 13, stmts: schemaV13},
	{version: 14, stmts: schemaV14},
	{version: 15, stmts: schemaV15},
}

// Open connects to the database named by dsn. A postgres:// or postgresql://
// URL selects the PostgreSQL backend; any other value is a SQLite file path,
// created when missing. Callers are responsible for ensuring the parent
// directory of a SQLite file exists.
func Open(dsn string) (*sql.DB, error) {
	if DetectDialect(dsn) == Postgres {
		return openPostgres(dsn)
	}
	return openSQLite(dsn)
}

// openSQLite opens (creating when missing) the SQLite database at path with
// the pragmas used across the application. The database file is created with
// 0600 permissions.
func openSQLite(path string) (*sql.DB, error) {
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
// The migration scripts are written in SQLite form and translated for the
// given dialect.
func Migrate(handle *sql.DB, dialect Dialect) error {
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
		for _, stmt := range splitStatements(translateSchema(m.stmts, dialect)) {
			if _, err := tx.Exec(stmt); err != nil {
				_ = tx.Rollback()
				return fmt.Errorf("apply migration %d: %w", m.version, err)
			}
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
