package store

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"tunnel-manager/models"
)

// upsertSetting stores one key in app_settings.
func upsertSetting(tx *sql.Tx, key, value string) error {
	_, err := tx.Exec(`INSERT INTO app_settings(key, value) VALUES(?, ?)
		ON CONFLICT(key) DO UPDATE SET value = excluded.value`, key, value)
	if err != nil {
		return fmt.Errorf("save setting %s: %w", key, err)
	}
	return nil
}

// loadSetting reads one optional key from app_settings.
func loadSetting(handle *sql.DB, key string) (string, bool, error) {
	var value string
	err := handle.QueryRow(`SELECT value FROM app_settings WHERE key = ?`, key).Scan(&value)
	if err == sql.ErrNoRows {
		return "", false, nil
	}
	if err != nil {
		return "", false, fmt.Errorf("load setting %s: %w", key, err)
	}
	return value, true, nil
}

// replaceUsers syncs the users and preferences tables.
func replaceUsers(tx *sql.Tx, users []models.User, prefs map[string]models.UserPrefs) error {
	if _, err := tx.Exec(`DELETE FROM user_prefs`); err != nil {
		return fmt.Errorf("clear user prefs: %w", err)
	}
	if _, err := tx.Exec(`DELETE FROM users`); err != nil {
		return fmt.Errorf("clear users: %w", err)
	}
	for _, u := range users {
		if _, err := tx.Exec(`INSERT INTO users(id, username, email, password_hash, role, group_id, status,
			email_verified, totp_enabled, totp_secret_encrypted, totp_last_accepted_step, totp_recovery_code_hashes,
			created_at, last_login_at, active_cf_connection_id) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
			u.ID, u.Username, u.Email, u.PasswordHash, u.Role, u.GroupID, u.Status,
			boolInt(u.EmailVerified), boolInt(u.TOTPEnabled), u.TOTPSecretEncrypted, u.TOTPLastAcceptedStep,
			strings.Join(u.TOTPRecoveryCodeHashes, "\n"), u.CreatedAt, u.LastLoginAt, u.ActiveCFConnectionID); err != nil {
			return fmt.Errorf("save user %s: %w", u.ID, err)
		}
		p := prefs[u.ID]
		if _, err := tx.Exec(`INSERT INTO user_prefs(user_id, tunnel_id, tunnel_name, service_url, selected_zone_id, selected_zone_name)
			VALUES(?,?,?,?,?,?)`, u.ID, p.TunnelID, p.TunnelName, p.ServiceURL, p.SelectedZoneID, p.SelectedZoneName); err != nil {
			return fmt.Errorf("save prefs %s: %w", u.ID, err)
		}
	}
	return nil
}

// loadUsers reads users and their preferences.
func loadUsers(handle *sql.DB) ([]models.User, map[string]models.UserPrefs, error) {
	rows, err := handle.Query(`SELECT id, username, email, password_hash, role, group_id, status,
		email_verified, totp_enabled, totp_secret_encrypted, totp_last_accepted_step, totp_recovery_code_hashes,
		created_at, last_login_at, active_cf_connection_id FROM users`)
	if err != nil {
		return nil, nil, fmt.Errorf("load users: %w", err)
	}
	defer rows.Close()
	users := []models.User{}
	for rows.Next() {
		var u models.User
		var emailVerified, totpEnabled int
		var recoveryHashes string
		if err := rows.Scan(&u.ID, &u.Username, &u.Email, &u.PasswordHash, &u.Role, &u.GroupID, &u.Status,
			&emailVerified, &totpEnabled, &u.TOTPSecretEncrypted, &u.TOTPLastAcceptedStep, &recoveryHashes,
			&u.CreatedAt, &u.LastLoginAt, &u.ActiveCFConnectionID); err != nil {
			return nil, nil, fmt.Errorf("scan user: %w", err)
		}
		u.EmailVerified = emailVerified != 0
		u.TOTPEnabled = totpEnabled != 0
		if recoveryHashes != "" {
			u.TOTPRecoveryCodeHashes = strings.Split(recoveryHashes, "\n")
		}
		users = append(users, u)
	}
	if err := rows.Err(); err != nil {
		return nil, nil, fmt.Errorf("iterate users: %w", err)
	}

	prefs := map[string]models.UserPrefs{}
	prefRows, err := handle.Query(`SELECT user_id, tunnel_id, tunnel_name, service_url, selected_zone_id, selected_zone_name FROM user_prefs`)
	if err != nil {
		return nil, nil, fmt.Errorf("load user prefs: %w", err)
	}
	defer prefRows.Close()
	for prefRows.Next() {
		var userID string
		var p models.UserPrefs
		if err := prefRows.Scan(&userID, &p.TunnelID, &p.TunnelName, &p.ServiceURL, &p.SelectedZoneID, &p.SelectedZoneName); err != nil {
			return nil, nil, fmt.Errorf("scan user prefs: %w", err)
		}
		prefs[userID] = p
	}
	return users, prefs, prefRows.Err()
}

// replaceGroups syncs the user_groups table.
func replaceGroups(tx *sql.Tx, groups []models.UserGroup) error {
	if _, err := tx.Exec(`DELETE FROM user_groups`); err != nil {
		return fmt.Errorf("clear groups: %w", err)
	}
	for _, g := range groups {
		perms, err := json.Marshal(g.Permissions)
		if err != nil {
			return fmt.Errorf("encode group permissions: %w", err)
		}
		builtin := 0
		if g.Builtin {
			builtin = 1
		}
		if _, err := tx.Exec(`INSERT INTO user_groups(id, name, permissions, builtin, created_at) VALUES(?,?,?,?,?)`,
			g.ID, g.Name, string(perms), builtin, g.CreatedAt); err != nil {
			return fmt.Errorf("save group %s: %w", g.ID, err)
		}
	}
	return nil
}

// loadGroups reads the user_groups table.
func loadGroups(handle *sql.DB) ([]models.UserGroup, error) {
	rows, err := handle.Query(`SELECT id, name, permissions, builtin, created_at FROM user_groups ORDER BY created_at`)
	if err != nil {
		return nil, fmt.Errorf("load groups: %w", err)
	}
	defer rows.Close()
	groups := []models.UserGroup{}
	for rows.Next() {
		var g models.UserGroup
		var perms string
		var builtin int
		if err := rows.Scan(&g.ID, &g.Name, &perms, &builtin, &g.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan group: %w", err)
		}
		g.Builtin = builtin != 0
		if perms != "" {
			_ = json.Unmarshal([]byte(perms), &g.Permissions)
		}
		groups = append(groups, g)
	}
	return groups, rows.Err()
}

// replaceSessions syncs the sessions table.
func replaceSessions(tx *sql.Tx, sessions []sessionRecord) error {
	if _, err := tx.Exec(`DELETE FROM sessions`); err != nil {
		return fmt.Errorf("clear sessions: %w", err)
	}
	for _, sess := range sessions {
		if _, err := tx.Exec(`INSERT INTO sessions(token_hash, user_id, created_at, expires_at) VALUES(?,?,?,?)`,
			sess.TokenHash, sess.UserID, sess.CreatedAt, sess.ExpiresAt); err != nil {
			return fmt.Errorf("save session: %w", err)
		}
	}
	return nil
}

// loadSessions reads the sessions table.
func loadSessions(handle *sql.DB) ([]sessionRecord, error) {
	rows, err := handle.Query(`SELECT token_hash, user_id, created_at, expires_at FROM sessions`)
	if err != nil {
		return nil, fmt.Errorf("load sessions: %w", err)
	}
	defer rows.Close()
	sessions := []sessionRecord{}
	for rows.Next() {
		var sess sessionRecord
		if err := rows.Scan(&sess.TokenHash, &sess.UserID, &sess.CreatedAt, &sess.ExpiresAt); err != nil {
			return nil, fmt.Errorf("scan session: %w", err)
		}
		sessions = append(sessions, sess)
	}
	return sessions, rows.Err()
}

// replaceInvites syncs the invites table.
func replaceInvites(tx *sql.Tx, invites []models.Invite) error {
	if _, err := tx.Exec(`DELETE FROM invites`); err != nil {
		return fmt.Errorf("clear invites: %w", err)
	}
	for _, invite := range invites {
		enabled := 0
		if invite.Enabled {
			enabled = 1
		}
		if _, err := tx.Exec(`INSERT INTO invites(code, group_id, max_uses, used_count, expires_at, enabled, created_at)
			VALUES(?,?,?,?,?,?,?)`, invite.Code, invite.GroupID, invite.MaxUses, invite.UsedCount, invite.ExpiresAt, enabled, invite.CreatedAt); err != nil {
			return fmt.Errorf("save invite %s: %w", invite.Code, err)
		}
	}
	return nil
}

// loadInvites reads the invites table.
func loadInvites(handle *sql.DB) ([]models.Invite, error) {
	rows, err := handle.Query(`SELECT code, group_id, max_uses, used_count, expires_at, enabled, created_at FROM invites`)
	if err != nil {
		return nil, fmt.Errorf("load invites: %w", err)
	}
	defer rows.Close()
	invites := []models.Invite{}
	for rows.Next() {
		var invite models.Invite
		var enabled int
		if err := rows.Scan(&invite.Code, &invite.GroupID, &invite.MaxUses, &invite.UsedCount, &invite.ExpiresAt, &enabled, &invite.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan invite: %w", err)
		}
		invite.Enabled = enabled != 0
		invites = append(invites, invite)
	}
	return invites, rows.Err()
}

// replaceVerifyCodes syncs the verify_codes table.
func replaceVerifyCodes(tx *sql.Tx, codes []verifyCodeRecord) error {
	if _, err := tx.Exec(`DELETE FROM verify_codes`); err != nil {
		return fmt.Errorf("clear verify codes: %w", err)
	}
	for _, code := range codes {
		if _, err := tx.Exec(`INSERT INTO verify_codes(email, purpose, code_hash, created_at, expires_at) VALUES(?,?,?,?,?)`,
			code.Email, code.Purpose, code.CodeHash, code.CreatedAt, code.ExpiresAt); err != nil {
			return fmt.Errorf("save verify code: %w", err)
		}
	}
	return nil
}

// loadVerifyCodes reads the verify_codes table.
func loadVerifyCodes(handle *sql.DB) ([]verifyCodeRecord, error) {
	rows, err := handle.Query(`SELECT email, purpose, code_hash, created_at, expires_at FROM verify_codes`)
	if err != nil {
		return nil, fmt.Errorf("load verify codes: %w", err)
	}
	defer rows.Close()
	codes := []verifyCodeRecord{}
	for rows.Next() {
		var code verifyCodeRecord
		if err := rows.Scan(&code.Email, &code.Purpose, &code.CodeHash, &code.CreatedAt, &code.ExpiresAt); err != nil {
			return nil, fmt.Errorf("scan verify code: %w", err)
		}
		codes = append(codes, code)
	}
	return codes, rows.Err()
}

// replaceCFConnections syncs the cf_connections table.
func replaceCFConnections(tx *sql.Tx, conns []models.CFConnection) error {
	if _, err := tx.Exec(`DELETE FROM cf_connections`); err != nil {
		return fmt.Errorf("clear cf connections: %w", err)
	}
	for _, conn := range conns {
		if _, err := tx.Exec(`INSERT INTO cf_connections(id, user_id, label, account_id, account_name,
			access_token, refresh_token, expires_at, scope, created_at) VALUES(?,?,?,?,?,?,?,?,?,?)`,
			conn.ID, conn.UserID, conn.Label, conn.AccountID, conn.AccountName,
			conn.AccessToken, conn.RefreshToken, conn.ExpiresAt, conn.Scope, conn.CreatedAt); err != nil {
			return fmt.Errorf("save cf connection %s: %w", conn.ID, err)
		}
	}
	return nil
}

// replaceAlertLogs syncs the alert_logs table.
func replaceAlertLogs(tx *sql.Tx, logs []models.AlertLog) error {
	if _, err := tx.Exec(`DELETE FROM alert_logs`); err != nil {
		return fmt.Errorf("clear alert logs: %w", err)
	}
	for _, entry := range logs {
		if _, err := tx.Exec(`INSERT INTO alert_logs(monitor_id, target_id, target_name, state, http_code, error, notified, detail, created_at)
			VALUES(?,?,?,?,?,?,?,?,?)`,
			entry.MonitorID, entry.TargetID, entry.TargetName, entry.State, entry.HTTPCode, entry.Error,
			boolInt(entry.Notified), entry.Detail, entry.CreatedAt); err != nil {
			return fmt.Errorf("save alert log: %w", err)
		}
	}
	return nil
}

// loadAlertLogs reads the alert_logs table.
func loadAlertLogs(handle *sql.DB) ([]models.AlertLog, error) {
	rows, err := handle.Query(`SELECT monitor_id, target_id, target_name, state, http_code, error, notified, detail, created_at FROM alert_logs`)
	if err != nil {
		return nil, fmt.Errorf("load alert logs: %w", err)
	}
	defer rows.Close()
	logs := []models.AlertLog{}
	for rows.Next() {
		var entry models.AlertLog
		var notified int
		if err := rows.Scan(&entry.MonitorID, &entry.TargetID, &entry.TargetName, &entry.State, &entry.HTTPCode, &entry.Error, &notified, &entry.Detail, &entry.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan alert log: %w", err)
		}
		entry.Notified = notified != 0
		logs = append(logs, entry)
	}
	return logs, rows.Err()
}

// loadCFConnections reads the cf_connections table.
func loadCFConnections(handle *sql.DB) ([]models.CFConnection, error) {
	rows, err := handle.Query(`SELECT id, user_id, label, account_id, account_name,
		access_token, refresh_token, expires_at, scope, created_at FROM cf_connections`)
	if err != nil {
		return nil, fmt.Errorf("load cf connections: %w", err)
	}
	defer rows.Close()
	conns := []models.CFConnection{}
	for rows.Next() {
		var conn models.CFConnection
		if err := rows.Scan(&conn.ID, &conn.UserID, &conn.Label, &conn.AccountID, &conn.AccountName,
			&conn.AccessToken, &conn.RefreshToken, &conn.ExpiresAt, &conn.Scope, &conn.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan cf connection: %w", err)
		}
		conns = append(conns, conn)
	}
	return conns, rows.Err()
}
