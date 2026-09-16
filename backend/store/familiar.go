package store

import (
	"database/sql"
	"fmt"
	"net"
	"strings"
	"time"

	"tunnel-manager/db"
)

// How much of an address is kept when learning where an account signs in from.
// A /24 is one neighbourhood of IPv4 and a /64 is its IPv6 equivalent: wide
// enough that a dynamic address moving inside the pool still matches, and far
// narrower than the city an operator is picturing.
const (
	familiarIPv4Bits = 24
	familiarIPv6Bits = 64
)

// maxFamiliarSubnetsPerUser bounds one account's learned set. Roaming and
// dynamic addressing both add rows, and the set only has to cover the places
// someone actually signs in from.
const maxFamiliarSubnetsPerUser = 32

// familiarWindow is how long a network stays trusted after the last sign-in
// from it.
//
// One successful login must not widen the budget for a neighbourhood
// forever: a coffee shop or hotel network visited once would otherwise stay
// trusted indefinitely, and whoever used it next would inherit the
// relaxation without ever having signed in.
const familiarWindow = 90 * 24 * time.Hour

// SubnetOf reduces an address to the network that identifies where it came
// from, or "" when the address cannot be parsed.
func SubnetOf(ip string) string {
	parsed := net.ParseIP(strings.TrimSpace(ip))
	if parsed == nil {
		return ""
	}
	if v4 := parsed.To4(); v4 != nil {
		mask := net.CIDRMask(familiarIPv4Bits, 32)
		return (&net.IPNet{IP: v4.Mask(mask), Mask: mask}).String()
	}
	mask := net.CIDRMask(familiarIPv6Bits, 128)
	return (&net.IPNet{IP: parsed.Mask(mask), Mask: mask}).String()
}

// RememberFamiliarIP records that an account signed in from this address.
//
// It is called only after a successful sign-in, so the set grows from proven
// credentials alone: a failed guess never widens the account's own allowlist,
// which is what stops the relaxation from being something an attacker can
// award themselves by hammering the form.
func (s *Store) RememberFamiliarIP(userID, ip string) error {
	subnet := SubnetOf(ip)
	if userID == "" || subnet == "" {
		return nil
	}
	handle, err := db.Open(s.filePath)
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	defer handle.Close()

	now := time.Now().Unix()
	tx, err := handle.Begin()
	if err != nil {
		return fmt.Errorf("begin: %w", err)
	}
	defer tx.Rollback()

	_, err = tx.Exec(`INSERT INTO familiar_ips(user_id, subnet, first_seen, last_seen, seen_count)
VALUES(?, ?, ?, ?, 1)
ON CONFLICT(user_id, subnet) DO UPDATE SET
  last_seen = excluded.last_seen,
  seen_count = familiar_ips.seen_count + 1`, userID, subnet, now, now)
	if err != nil {
		return fmt.Errorf("remember familiar network: %w", err)
	}
	if err := pruneFamiliarLocked(tx, userID); err != nil {
		return err
	}
	return tx.Commit()
}

// IsFamiliarIP reports whether this address falls in a network the account has
// signed in from before. Unknown accounts and unparseable addresses are never
// familiar.
func (s *Store) IsFamiliarIP(userID, ip string) bool {
	subnet := SubnetOf(ip)
	if userID == "" || subnet == "" {
		return false
	}
	handle, err := db.Open(s.filePath)
	if err != nil {
		return false
	}
	defer handle.Close()

	var found int
	if err := handle.QueryRow(`SELECT 1 FROM familiar_ips WHERE user_id = ? AND subnet = ? AND last_seen >= ?`,
		userID, subnet, time.Now().Add(-familiarWindow).Unix()).Scan(&found); err != nil {
		return false
	}
	return true
}

// ListFamiliarIPs returns one account's learned networks, most recent first.
func (s *Store) ListFamiliarIPs(userID string) ([]string, error) {
	handle, err := db.Open(s.filePath)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}
	defer handle.Close()

	rows, err := handle.Query(`SELECT subnet FROM familiar_ips WHERE user_id = ? ORDER BY last_seen DESC`, userID)
	if err != nil {
		return nil, fmt.Errorf("load familiar networks: %w", err)
	}
	defer rows.Close()
	out := []string{}
	for rows.Next() {
		var subnet string
		if err := rows.Scan(&subnet); err != nil {
			return nil, fmt.Errorf("scan familiar network: %w", err)
		}
		out = append(out, subnet)
	}
	return out, rows.Err()
}

// deleteFamiliarIPsForUser drops the learned networks when an account goes
// away. Called from DeleteUser, since the table carries no foreign key to
// cascade through.
func (s *Store) deleteFamiliarIPsForUser(userID string) error {
	if userID == "" {
		return nil
	}
	handle, err := db.Open(s.filePath)
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	defer handle.Close()
	if _, err := handle.Exec(`DELETE FROM familiar_ips WHERE user_id = ?`, userID); err != nil {
		return fmt.Errorf("delete familiar networks: %w", err)
	}
	return nil
}

// pruneFamiliarLocked keeps one account's learned set bounded: expired
// networks are dropped outright rather than merely ignored, and the rest is
// capped so a roaming user cannot grow the table without bound.
func pruneFamiliarLocked(tx *sql.Tx, userID string) error {
	if _, err := tx.Exec(`DELETE FROM familiar_ips WHERE user_id = ? AND last_seen < ?`,
		userID, time.Now().Add(-familiarWindow).Unix()); err != nil {
		return fmt.Errorf("expire familiar networks: %w", err)
	}
	_, err := tx.Exec(`DELETE FROM familiar_ips WHERE user_id = ? AND subnet NOT IN (
  SELECT subnet FROM familiar_ips WHERE user_id = ? ORDER BY last_seen DESC LIMIT ?)`, userID, userID, maxFamiliarSubnetsPerUser)
	if err != nil {
		return fmt.Errorf("prune familiar networks: %w", err)
	}
	return nil
}
