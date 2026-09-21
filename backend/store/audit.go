package store

import (
	"fmt"
	"strings"
	"time"

	"tunnel-manager/db"
	"tunnel-manager/models"
)

// maxAuditFieldLength bounds the free-form columns so a single request cannot
// grow one row without limit.
const maxAuditFieldLength = 200

// RecordAudit appends one entry to the audit trail. Entries go straight to
// SQLite instead of through the in-memory cache: the trail is append-only and
// unbounded, whereas saveLocked rewrites every cached table on each mutation.
func (s *Store) RecordAudit(entry models.AuditLog) error {
	if entry.CreatedAt == 0 {
		entry.CreatedAt = time.Now().Unix()
	}
	handle, err := db.Open(s.dsn)
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	defer handle.Close()

	if _, err := handle.Exec(`INSERT INTO audit_logs(created_at, actor_id, actor_name, category, action, target, ip, success)
		VALUES(?,?,?,?,?,?,?,?)`,
		entry.CreatedAt, truncateAudit(entry.ActorID), truncateAudit(entry.ActorName), truncateAudit(entry.Category),
		truncateAudit(entry.Action), truncateAudit(entry.Target), truncateAudit(entry.IP), boolInt(entry.Success)); err != nil {
		return fmt.Errorf("insert audit log: %w", err)
	}
	return nil
}

// QueryAuditLogs returns one page of the trail, newest first, together with
// the total number of entries matching the filter.
func (s *Store) QueryAuditLogs(q models.AuditQuery) (models.AuditPage, error) {
	page := q.Page
	if page < 1 {
		page = 1
	}
	size := q.PageSize
	if size <= 0 {
		size = 50
	}
	if size > 200 {
		size = 200
	}
	out := models.AuditPage{Logs: []models.AuditLog{}, Page: page, PageSize: size}

	handle, err := db.Open(s.dsn)
	if err != nil {
		return out, fmt.Errorf("open database: %w", err)
	}
	defer handle.Close()

	where, args := auditFilter(q)
	if err := handle.QueryRow(`SELECT COUNT(*) FROM audit_logs`+where, args...).Scan(&out.Total); err != nil {
		return out, fmt.Errorf("count audit logs: %w", err)
	}

	rows, err := handle.Query(`SELECT id, created_at, actor_id, actor_name, category, action, target, ip, success
		FROM audit_logs`+where+` ORDER BY id DESC LIMIT ? OFFSET ?`,
		append(append([]interface{}{}, args...), size, (page-1)*size)...)
	if err != nil {
		return out, fmt.Errorf("load audit logs: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var entry models.AuditLog
		var success int
		if err := rows.Scan(&entry.ID, &entry.CreatedAt, &entry.ActorID, &entry.ActorName, &entry.Category,
			&entry.Action, &entry.Target, &entry.IP, &success); err != nil {
			return out, fmt.Errorf("scan audit log: %w", err)
		}
		entry.Success = success != 0
		out.Logs = append(out.Logs, entry)
	}
	return out, rows.Err()
}

// AuditStats summarizes the trail from the given unix second onwards, plus the
// number of entries recorded since todayFrom.
func (s *Store) AuditStats(from, todayFrom int64) (models.AuditStats, error) {
	stats := models.AuditStats{}
	handle, err := db.Open(s.dsn)
	if err != nil {
		return stats, fmt.Errorf("open database: %w", err)
	}
	defer handle.Close()

	if err := handle.QueryRow(`SELECT COUNT(*),
		COALESCE(SUM(CASE WHEN success = 0 THEN 1 ELSE 0 END), 0),
		COUNT(DISTINCT CASE WHEN actor_id != '' THEN actor_id END)
		FROM audit_logs WHERE created_at >= ?`, from).Scan(&stats.Total, &stats.Failed, &stats.Actors); err != nil {
		return stats, fmt.Errorf("summarize audit logs: %w", err)
	}
	if err := handle.QueryRow(`SELECT COUNT(*) FROM audit_logs WHERE created_at >= ?`, todayFrom).Scan(&stats.Today); err != nil {
		return stats, fmt.Errorf("count today audit logs: %w", err)
	}
	return stats, nil
}

// PruneAuditLogs deletes every entry older than cutoff and reports how many
// rows were removed.
func (s *Store) PruneAuditLogs(cutoff int64) (int64, error) {
	handle, err := db.Open(s.dsn)
	if err != nil {
		return 0, fmt.Errorf("open database: %w", err)
	}
	defer handle.Close()

	result, err := handle.Exec(`DELETE FROM audit_logs WHERE created_at < ?`, cutoff)
	if err != nil {
		return 0, fmt.Errorf("prune audit logs: %w", err)
	}
	removed, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("count pruned audit logs: %w", err)
	}
	return removed, nil
}

// auditFilter builds the WHERE clause shared by the count and page queries.
func auditFilter(q models.AuditQuery) (string, []interface{}) {
	clauses := []string{}
	args := []interface{}{}
	if actor := strings.TrimSpace(q.Actor); actor != "" {
		// Match the stored display name (which also covers renamed or deleted
		// accounts) and the exact account id.
		clauses = append(clauses, `(actor_name LIKE ? ESCAPE '\' OR actor_id = ?)`)
		args = append(args, "%"+escapeAuditLike(actor)+"%", actor)
	}
	if q.Category != "" {
		clauses = append(clauses, "category = ?")
		args = append(args, q.Category)
	}
	if q.Action != "" {
		clauses = append(clauses, "action = ?")
		args = append(args, q.Action)
	}
	if q.From > 0 {
		clauses = append(clauses, "created_at >= ?")
		args = append(args, q.From)
	}
	if q.To > 0 {
		clauses = append(clauses, "created_at <= ?")
		args = append(args, q.To)
	}
	if len(clauses) == 0 {
		return "", args
	}
	return " WHERE " + strings.Join(clauses, " AND "), args
}

// escapeAuditLike neutralizes the LIKE wildcards in user input.
func escapeAuditLike(value string) string {
	return strings.NewReplacer(`\`, `\\`, "%", `\%`, "_", `\_`).Replace(value)
}

// truncateAudit caps a free-form field without splitting a rune.
func truncateAudit(value string) string {
	runes := []rune(value)
	if len(runes) > maxAuditFieldLength {
		return string(runes[:maxAuditFieldLength])
	}
	return value
}
