package handlers

import (
	"context"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"tunnel-manager/models"
	"tunnel-manager/store"
)

// ---------------------------------------------------------------------------
// Recording

// auditTargetKey carries a handler-supplied description of the object it just
// touched, for operations whose target is not a path parameter (creates).
type auditTargetKey struct{}

type auditTargetHolder struct{ value string }

// SetAuditTarget lets a handler name the object it created so the entry the
// Audit middleware writes afterwards mentions it instead of only the action.
func SetAuditTarget(r *http.Request, target string) {
	if holder, ok := r.Context().Value(auditTargetKey{}).(*auditTargetHolder); ok {
		holder.value = target
	}
}

// statusRecorder captures the response status so the audit middleware can tell
// a completed operation from a rejected one.
type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (w *statusRecorder) WriteHeader(status int) {
	if w.status == 0 {
		w.status = status
	}
	w.ResponseWriter.WriteHeader(status)
}

func (w *statusRecorder) Write(data []byte) (int, error) {
	if w.status == 0 {
		w.status = http.StatusOK
	}
	return w.ResponseWriter.Write(data)
}

// Audit records one entry for a mutating request once the handler returns.
// target resolves the affected object from the request (usually a path
// parameter) and may be nil. It must be wrapped by Auth so the identity is
// present; read-only routes are left unwrapped.
func (m *Middleware) Audit(category, action string, target func(*http.Request) string, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		holder := &auditTargetHolder{}
		recorder := &statusRecorder{ResponseWriter: w}
		next(recorder, r.WithContext(context.WithValue(r.Context(), auditTargetKey{}, holder)))

		status := recorder.status
		if status == 0 {
			status = http.StatusOK
		}
		entry := models.AuditLog{
			Category: category,
			Action:   action,
			IP:       clientIP(r),
			Success:  status < http.StatusBadRequest,
		}
		if user := SessionUser(r); user != nil {
			entry.ActorID = user.ID
			entry.ActorName = user.Username
		}
		if target != nil {
			entry.Target = target(r)
		}
		if entry.Target == "" {
			entry.Target = holder.value
		}
		recordAuditEntry(m.Store, entry)
	}
}

// PathParam resolves an audit target from one chi path parameter.
func PathParam(name string) func(*http.Request) string {
	return func(r *http.Request) string { return chi.URLParam(r, name) }
}

// UserTarget resolves an account path parameter to its username, falling back
// to the raw id when the account no longer exists.
func (m *Middleware) UserTarget(param string) func(*http.Request) string {
	return func(r *http.Request) string {
		id := chi.URLParam(r, param)
		if m.Store != nil {
			if user, ok := m.Store.GetUserByID(id); ok {
				return user.Username
			}
		}
		return id
	}
}

// GroupTarget resolves a group path parameter to its display name.
func (m *Middleware) GroupTarget(param string) func(*http.Request) string {
	return func(r *http.Request) string {
		id := chi.URLParam(r, param)
		if m.Store != nil {
			for _, group := range m.Store.ListGroups() {
				if group.ID == id {
					return group.Name
				}
			}
		}
		return id
	}
}

// MonitorTarget resolves a monitor path parameter to its name.
func (m *Middleware) MonitorTarget(param string) func(*http.Request) string {
	return func(r *http.Request) string {
		id := chi.URLParam(r, param)
		if m.Store != nil {
			for _, monitor := range m.Store.GetConfig().Monitors {
				if monitor.ID == id {
					return monitor.Name
				}
			}
		}
		return id
	}
}

// recordAuditEntry persists one entry, logging (never failing the request) when
// the trail cannot be written.
func recordAuditEntry(st *store.Store, entry models.AuditLog) {
	if st == nil {
		return
	}
	if err := st.RecordAudit(entry); err != nil {
		log.Printf("record audit log: %v", err)
	}
}

// ---------------------------------------------------------------------------
// Reading

// AuditHandler serves the administrator audit trail.
type AuditHandler struct {
	store *store.Store
}

// NewAuditHandler creates the audit trail handler.
func NewAuditHandler(st *store.Store) *AuditHandler {
	return &AuditHandler{store: st}
}

// List handles GET /api/admin/audit-logs. Filters: actor (fuzzy account name),
// category, action, from and to (unix seconds); page and page_size paginate.
func (h *AuditHandler) List(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	page, err := h.store.QueryAuditLogs(models.AuditQuery{
		Actor:    strings.TrimSpace(query.Get("actor")),
		Category: strings.TrimSpace(query.Get("category")),
		Action:   strings.TrimSpace(query.Get("action")),
		From:     queryInt64(query.Get("from")),
		To:       queryInt64(query.Get("to")),
		Page:     int(queryInt64(query.Get("page"))),
		PageSize: int(queryInt64(query.Get("page_size"))),
	})
	if err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "读取审计日志失败"})
		return
	}
	writeJSON(w, http.StatusOK, page)
}

// Stats handles GET /api/admin/audit-logs/stats: totals for the recent window
// (days, default 7) plus today's count.
func (h *AuditHandler) Stats(w http.ResponseWriter, r *http.Request) {
	days := int(queryInt64(r.URL.Query().Get("days")))
	if days <= 0 {
		days = 7
	}
	if days > 90 {
		days = 90
	}
	now := time.Now()
	stats, err := h.store.AuditStats(now.AddDate(0, 0, -days).Unix(), startOfDay(now))
	if err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "读取审计统计失败"})
		return
	}
	writeJSON(w, http.StatusOK, stats)
}

// startOfDay returns the unix second at which the local day containing t began.
func startOfDay(t time.Time) int64 {
	year, month, day := t.Date()
	return time.Date(year, month, day, 0, 0, 0, 0, t.Location()).Unix()
}

// queryInt64 parses a numeric query parameter, defaulting to zero.
func queryInt64(value string) int64 {
	parsed, err := strconv.ParseInt(strings.TrimSpace(value), 10, 64)
	if err != nil {
		return 0
	}
	return parsed
}
