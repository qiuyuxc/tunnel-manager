package store

import (
	"errors"
	"fmt"
	"time"

	"tunnel-manager/db"
	"tunnel-manager/models"
)

// ErrPasskeyNotFound reports a passkey that does not exist or does not belong
// to the account asking for it.
var ErrPasskeyNotFound = errors.New("passkey not found")

// ListPasskeys returns one account's passkeys, oldest first.
func (s *Store) ListPasskeys(userID string) ([]models.Passkey, error) {
	handle, err := db.Open(s.dsn)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}
	defer handle.Close()

	rows, err := handle.Query(`SELECT id, user_id, name, credential, created_at, last_used_at
		FROM passkeys WHERE user_id = ? ORDER BY created_at, id`, userID)
	if err != nil {
		return nil, fmt.Errorf("load passkeys: %w", err)
	}
	defer rows.Close()
	out := []models.Passkey{}
	for rows.Next() {
		var entry models.Passkey
		if err := rows.Scan(&entry.ID, &entry.UserID, &entry.Name, &entry.Credential, &entry.CreatedAt, &entry.LastUsedAt); err != nil {
			return nil, fmt.Errorf("scan passkey: %w", err)
		}
		out = append(out, entry)
	}
	return out, rows.Err()
}

// passkeyCounts reports how many passkeys each account has bound, for the
// administrator's account list.
func (s *Store) passkeyCounts() (map[string]int, error) {
	handle, err := db.Open(s.dsn)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}
	defer handle.Close()

	rows, err := handle.Query(`SELECT user_id, COUNT(*) FROM passkeys GROUP BY user_id`)
	if err != nil {
		return nil, fmt.Errorf("count passkeys: %w", err)
	}
	defer rows.Close()
	counts := map[string]int{}
	for rows.Next() {
		var userID string
		var count int
		if err := rows.Scan(&userID, &count); err != nil {
			return nil, fmt.Errorf("scan passkey count: %w", err)
		}
		counts[userID] = count
	}
	return counts, rows.Err()
}

// CountPasskeys reports how many passkeys an account has bound.
func (s *Store) CountPasskeys(userID string) (int, error) {
	handle, err := db.Open(s.dsn)
	if err != nil {
		return 0, fmt.Errorf("open database: %w", err)
	}
	defer handle.Close()

	var count int
	if err := handle.QueryRow(`SELECT COUNT(*) FROM passkeys WHERE user_id = ?`, userID).Scan(&count); err != nil {
		return 0, fmt.Errorf("count passkeys: %w", err)
	}
	return count, nil
}

// AddPasskey stores a newly registered credential.
func (s *Store) AddPasskey(entry models.Passkey) error {
	if entry.CreatedAt == 0 {
		entry.CreatedAt = time.Now().Unix()
	}
	handle, err := db.Open(s.dsn)
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	defer handle.Close()

	if _, err := handle.Exec(`INSERT INTO passkeys(id, user_id, name, credential, created_at, last_used_at)
		VALUES(?,?,?,?,?,?)`,
		entry.ID, entry.UserID, entry.Name, entry.Credential, entry.CreatedAt, entry.LastUsedAt); err != nil {
		return fmt.Errorf("save passkey: %w", err)
	}
	return nil
}

// RenamePasskey updates the label of one of the account's passkeys.
func (s *Store) RenamePasskey(id, userID, name string) error {
	return s.execPasskey(`UPDATE passkeys SET name = ? WHERE id = ? AND user_id = ?`, name, id, userID)
}

// DeletePasskey removes one of the account's passkeys.
func (s *Store) DeletePasskey(id, userID string) error {
	return s.execPasskey(`DELETE FROM passkeys WHERE id = ? AND user_id = ?`, id, userID)
}

// UpdatePasskeyCredential rewrites a stored credential after an assertion,
// which is how the signature counter (and the backup flags) stay current.
func (s *Store) UpdatePasskeyCredential(id, credential string, usedAt int64) error {
	handle, err := db.Open(s.dsn)
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	defer handle.Close()

	if _, err := handle.Exec(`UPDATE passkeys SET credential = ?, last_used_at = ? WHERE id = ?`,
		credential, usedAt, id); err != nil {
		return fmt.Errorf("update passkey: %w", err)
	}
	return nil
}

// execPasskey runs a statement scoped to one credential and reports a missing
// row as ErrPasskeyNotFound.
func (s *Store) execPasskey(query string, args ...interface{}) error {
	handle, err := db.Open(s.dsn)
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	defer handle.Close()

	result, err := handle.Exec(query, args...)
	if err != nil {
		return fmt.Errorf("update passkey: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("count passkey rows: %w", err)
	}
	if affected == 0 {
		return ErrPasskeyNotFound
	}
	return nil
}

// SetUserPasswordLoginDisabled flips the per-account password switch. Users live
// in the in-memory cache (every save rewrites the table), so this goes through
// the cache rather than straight to SQLite.
func (s *Store) SetUserPasswordLoginDisabled(userID string, disabled bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	user := s.findUserLocked(userID)
	if user == nil {
		return ErrUserNotFound
	}
	if user.PasswordLoginDisabled == disabled {
		return nil
	}
	previous := user.PasswordLoginDisabled
	user.PasswordLoginDisabled = disabled
	if err := s.saveLocked(); err != nil {
		user.PasswordLoginDisabled = previous
		return err
	}
	return nil
}

// CountAdminsWithPasskeys reports how many active administrators have bound at
// least one passkey. The panel refuses to disable password sign-in globally
// while this is zero, so an install cannot lock itself out.
func (s *Store) CountAdminsWithPasskeys() (int, error) {
	s.mu.RLock()
	admins := map[string]bool{}
	for i := range s.users {
		if s.users[i].Role == models.RoleAdmin && s.users[i].Status == models.UserActive {
			admins[s.users[i].ID] = true
		}
	}
	s.mu.RUnlock()
	if len(admins) == 0 {
		return 0, nil
	}

	handle, err := db.Open(s.dsn)
	if err != nil {
		return 0, fmt.Errorf("open database: %w", err)
	}
	defer handle.Close()

	rows, err := handle.Query(`SELECT DISTINCT user_id FROM passkeys`)
	if err != nil {
		return 0, fmt.Errorf("load passkey owners: %w", err)
	}
	defer rows.Close()
	count := 0
	for rows.Next() {
		var userID string
		if err := rows.Scan(&userID); err != nil {
			return 0, fmt.Errorf("scan passkey owner: %w", err)
		}
		if admins[userID] {
			count++
		}
	}
	return count, rows.Err()
}

// deletePasskeysForUser drops every credential of a removed account. The
// passkeys table has no foreign key (saveLocked rewrites users on every save,
// which would cascade-delete credentials), so deletion is explicit.
func (s *Store) deletePasskeysForUser(userID string) error {
	handle, err := db.Open(s.dsn)
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	defer handle.Close()

	if _, err := handle.Exec(`DELETE FROM passkeys WHERE user_id = ?`, userID); err != nil {
		return fmt.Errorf("delete user passkeys: %w", err)
	}
	return nil
}
