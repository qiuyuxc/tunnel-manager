package store

import (
	"testing"
	"time"

	"tunnel-manager/models"
)

func TestAuditTrailRecordsFiltersAndPaginates(t *testing.T) {
	s := newTestStore(t, HashPassword("password"))
	base := time.Now().Unix() - 3600
	entries := []models.AuditLog{
		{ActorID: "u1", ActorName: "alice", Category: models.AuditCategoryUser, Action: models.AuditActionUserCreate, Target: "bob", IP: "10.0.0.1", Success: true, CreatedAt: base},
		{ActorID: "u1", ActorName: "alice", Category: models.AuditCategoryTunnel, Action: models.AuditActionTunnelDelete, Target: "tunnel-a", IP: "10.0.0.1", Success: true, CreatedAt: base + 10},
		{ActorID: "u2", ActorName: "carol", Category: models.AuditCategoryAuth, Action: models.AuditActionLoginFailed, Target: "carol", IP: "10.0.0.2", CreatedAt: base + 20},
	}
	for _, entry := range entries {
		if err := s.RecordAudit(entry); err != nil {
			t.Fatalf("RecordAudit() error = %v", err)
		}
	}

	page, err := s.QueryAuditLogs(models.AuditQuery{})
	if err != nil {
		t.Fatalf("QueryAuditLogs() error = %v", err)
	}
	if page.Total != 3 || len(page.Logs) != 3 {
		t.Fatalf("QueryAuditLogs() = %d logs of %d, want 3 of 3", len(page.Logs), page.Total)
	}
	if page.Logs[0].ActorName != "carol" || page.Logs[2].ActorName != "alice" {
		t.Fatalf("QueryAuditLogs() order = %q..%q, want newest first", page.Logs[0].ActorName, page.Logs[2].ActorName)
	}
	if page.Logs[0].Success || !page.Logs[1].Success {
		t.Fatalf("QueryAuditLogs() success flags = %v, %v, want false, true", page.Logs[0].Success, page.Logs[1].Success)
	}

	if got := queryAuditTotal(t, s, models.AuditQuery{Actor: "ali"}); got != 2 {
		t.Fatalf("actor filter total = %d, want 2", got)
	}
	if got := queryAuditTotal(t, s, models.AuditQuery{Actor: "u2"}); got != 1 {
		t.Fatalf("actor id filter total = %d, want 1", got)
	}
	if got := queryAuditTotal(t, s, models.AuditQuery{Category: models.AuditCategoryUser}); got != 1 {
		t.Fatalf("category filter total = %d, want 1", got)
	}
	if got := queryAuditTotal(t, s, models.AuditQuery{Action: models.AuditActionTunnelDelete}); got != 1 {
		t.Fatalf("action filter total = %d, want 1", got)
	}
	if got := queryAuditTotal(t, s, models.AuditQuery{From: base + 15}); got != 1 {
		t.Fatalf("from filter total = %d, want 1", got)
	}
	if got := queryAuditTotal(t, s, models.AuditQuery{To: base + 5}); got != 1 {
		t.Fatalf("to filter total = %d, want 1", got)
	}
	if got := queryAuditTotal(t, s, models.AuditQuery{Category: models.AuditCategoryUser, Action: models.AuditActionTunnelDelete}); got != 0 {
		t.Fatalf("combined filter total = %d, want 0", got)
	}

	first, err := s.QueryAuditLogs(models.AuditQuery{Page: 1, PageSize: 2})
	if err != nil {
		t.Fatalf("QueryAuditLogs() page 1 error = %v", err)
	}
	if len(first.Logs) != 2 || first.Total != 3 {
		t.Fatalf("page 1 = %d logs of %d, want 2 of 3", len(first.Logs), first.Total)
	}
	second, err := s.QueryAuditLogs(models.AuditQuery{Page: 2, PageSize: 2})
	if err != nil {
		t.Fatalf("QueryAuditLogs() page 2 error = %v", err)
	}
	if len(second.Logs) != 1 || second.Logs[0].ActorName != "alice" {
		t.Fatalf("page 2 = %#v, want the oldest entry", second.Logs)
	}
	capped, err := s.QueryAuditLogs(models.AuditQuery{PageSize: 5000})
	if err != nil {
		t.Fatalf("QueryAuditLogs() capped error = %v", err)
	}
	if capped.PageSize != 200 {
		t.Fatalf("page size = %d, want the 200 row cap", capped.PageSize)
	}
}

func TestAuditActorSearchTreatsWildcardsLiterally(t *testing.T) {
	s := newTestStore(t, HashPassword("password"))
	for _, name := range []string{"alice", "bob"} {
		if err := s.RecordAudit(models.AuditLog{ActorName: name, Category: models.AuditCategoryUser, Action: models.AuditActionUserCreate, Success: true}); err != nil {
			t.Fatalf("RecordAudit() error = %v", err)
		}
	}
	if got := queryAuditTotal(t, s, models.AuditQuery{Actor: "%"}); got != 0 {
		t.Fatalf("wildcard search total = %d, want 0", got)
	}
	if got := queryAuditTotal(t, s, models.AuditQuery{Actor: "li"}); got != 1 {
		t.Fatalf("substring search total = %d, want 1", got)
	}
}

func TestAuditStatsAndPrune(t *testing.T) {
	s := newTestStore(t, HashPassword("password"))
	now := time.Now()
	stale := now.AddDate(0, 0, -30).Unix()
	fresh := []models.AuditLog{
		{ActorID: "u1", ActorName: "alice", Category: models.AuditCategoryUser, Action: models.AuditActionUserCreate, Success: true},
		{ActorID: "u2", ActorName: "carol", Category: models.AuditCategoryAuth, Action: models.AuditActionLoginFailed},
	}
	for _, entry := range fresh {
		if err := s.RecordAudit(entry); err != nil {
			t.Fatalf("RecordAudit() error = %v", err)
		}
	}
	if err := s.RecordAudit(models.AuditLog{ActorID: "u1", ActorName: "alice", Category: models.AuditCategoryTunnel, Action: models.AuditActionTunnelDelete, Success: true, CreatedAt: stale}); err != nil {
		t.Fatalf("RecordAudit() error = %v", err)
	}

	stats, err := s.AuditStats(now.AddDate(0, 0, -7).Unix(), startOfLocalDay(now))
	if err != nil {
		t.Fatalf("AuditStats() error = %v", err)
	}
	if stats.Total != 2 || stats.Failed != 1 || stats.Today != 2 || stats.Actors != 2 {
		t.Fatalf("AuditStats() = %#v, want 2 total, 1 failed, 2 today, 2 actors", stats)
	}

	removed, err := s.PruneAuditLogs(now.AddDate(0, 0, -7).Unix())
	if err != nil {
		t.Fatalf("PruneAuditLogs() error = %v", err)
	}
	if removed != 1 {
		t.Fatalf("PruneAuditLogs() removed %d rows, want 1", removed)
	}
	if got := queryAuditTotal(t, s, models.AuditQuery{}); got != 2 {
		t.Fatalf("total after prune = %d, want 2", got)
	}
}

func TestAuditRetentionDefaults(t *testing.T) {
	settings := models.AppSettings{}
	if got := settings.EffectiveAuditRetentionDays(); got != models.DefaultAuditRetentionDays {
		t.Fatalf("EffectiveAuditRetentionDays() = %d, want the %d day default", got, models.DefaultAuditRetentionDays)
	}
	settings.AuditRetentionDays = -1
	if got := settings.EffectiveAuditRetentionDays(); got != -1 {
		t.Fatalf("EffectiveAuditRetentionDays() = %d, want -1 (keep forever)", got)
	}
	settings.AuditRetentionDays = 30
	if got := settings.EffectiveAuditRetentionDays(); got != 30 {
		t.Fatalf("EffectiveAuditRetentionDays() = %d, want 30", got)
	}
}

func queryAuditTotal(t *testing.T, s *Store, q models.AuditQuery) int {
	t.Helper()
	page, err := s.QueryAuditLogs(q)
	if err != nil {
		t.Fatalf("QueryAuditLogs(%#v) error = %v", q, err)
	}
	return page.Total
}

func startOfLocalDay(t time.Time) int64 {
	year, month, day := t.Date()
	return time.Date(year, month, day, 0, 0, 0, 0, t.Location()).Unix()
}
