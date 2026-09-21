package store

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"tunnel-manager/db"
)

// postgresTestDSN points the suite at a throwaway PostgreSQL server. Without
// it every test runs on SQLite, which stays the default backend.
var postgresTestDSN = os.Getenv("TM_TEST_POSTGRES_DSN")

// isPostgresTest reports whether the suite runs against PostgreSQL.
func isPostgresTest() bool { return postgresTestDSN != "" }

// schemaCounter keeps generated PostgreSQL schema names unique per test.
var schemaCounter atomic.Int64

// testDSN returns a database private to one test: a temporary SQLite file, or
// a dedicated PostgreSQL schema when TM_TEST_POSTGRES_DSN is set.
func testDSN(t *testing.T) string {
	t.Helper()
	if !isPostgresTest() {
		return filepath.Join(t.TempDir(), "config.json")
	}
	schema := fmt.Sprintf("tm_test_%d_%s", schemaCounter.Add(1), testNameSlug(t.Name()))
	handle, err := db.Open(postgresTestDSN)
	if err != nil {
		t.Fatalf("connect test postgres: %v", err)
	}
	defer handle.Close()
	if _, err := handle.Exec("CREATE SCHEMA " + schema); err != nil {
		t.Fatalf("create test schema: %v", err)
	}
	t.Cleanup(func() {
		cleanup, err := db.Open(postgresTestDSN)
		if err != nil {
			return
		}
		defer cleanup.Close()
		_, _ = cleanup.Exec("DROP SCHEMA IF EXISTS " + schema + " CASCADE")
	})
	return dsnWithSchema(t, schema)
}

// dsnWithSchema scopes a PostgreSQL DSN to one schema through search_path.
func dsnWithSchema(t *testing.T, schema string) string {
	t.Helper()
	parsed, err := url.Parse(postgresTestDSN)
	if err != nil || parsed.Scheme == "" {
		t.Fatalf("TM_TEST_POSTGRES_DSN must be a postgres:// URL, got %q", postgresTestDSN)
	}
	query := parsed.Query()
	query.Set("search_path", schema)
	parsed.RawQuery = query.Encode()
	return parsed.String()
}

// testNameSlug reduces a test name to characters valid in a schema name.
func testNameSlug(name string) string {
	var slug strings.Builder
	for _, char := range strings.ToLower(name) {
		switch {
		case char >= 'a' && char <= 'z', char >= '0' && char <= '9':
			slug.WriteRune(char)
		case char == '_':
			slug.WriteRune(char)
		}
	}
	if slug.Len() == 0 {
		return "case"
	}
	return slug.String()
}

// breakStore points a store at an unreachable database so save failures can be
// exercised on both backends.
func breakStore(t *testing.T, s *Store) {
	t.Helper()
	if isPostgresTest() {
		s.dsn = "postgres://postgres@127.0.0.1:59999/tunnel_manager_missing?sslmode=disable&connect_timeout=1"
		return
	}
	s.dsn = filepath.Join(t.TempDir(), "missing", "config.json")
}

// testStore returns an installed store: a fresh database with an administrator
// account, matching what the setup wizard leaves behind.
func testStore(t *testing.T) *Store {
	t.Helper()
	st := NewStore(testDSN(t))
	if !st.Installed() {
		if err := st.CreateAdmin("admin", "password"); err != nil {
			t.Fatalf("create administrator: %v", err)
		}
	}
	return st
}
