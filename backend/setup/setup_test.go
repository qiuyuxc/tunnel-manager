package setup

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The wizard renders the connection fields as a URL the driver can open, so
// both the ordinary host case and the Unix socket case are pinned here.
func TestPostgresDSN(t *testing.T) {
	cases := []struct {
		name string
		pg   Postgres
		want string
	}{
		{
			name: "tcp with password",
			pg:   Postgres{Host: "db.internal", Port: 5433, Database: "panel", User: "tm", Password: "s3cret", SSLMode: "require"},
			want: "postgres://tm:s3cret@db.internal:5433/panel?sslmode=require",
		},
		{
			name: "defaults the port and ssl mode",
			pg:   Postgres{Host: "127.0.0.1", Database: "panel", User: "tm", Password: "p"},
			want: "postgres://tm:p@127.0.0.1:5432/panel?sslmode=disable",
		},
		{
			name: "unix socket directory",
			pg:   Postgres{Host: "/var/run/postgresql", Port: 5433, Database: "panel", User: "tm", Password: "p"},
			want: "postgres://tm:p@/panel?host=%2Fvar%2Frun%2Fpostgresql&port=5433&sslmode=disable",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.pg.DSN(); got != tc.want {
				t.Fatalf("DSN() = %q, want %q", got, tc.want)
			}
		})
	}
}

// A socket host keeps the slash out of the URL authority, the only form the
// driver parses.
func TestPostgresDSNSocketHostStaysOutOfAuthority(t *testing.T) {
	dsn := Postgres{Host: "/run/postgresql", Database: "panel", User: "tm"}.DSN()
	if strings.Contains(strings.SplitN(dsn, "?", 2)[0], "%2F") {
		t.Fatalf("socket path leaked into the URL authority: %q", dsn)
	}
}

func TestStorageValidate(t *testing.T) {
	sqlite := Storage{Driver: DriverSQLite, Path: filepath.Join(t.TempDir(), "panel.db")}
	if err := sqlite.Validate(); err != nil {
		t.Fatalf("sqlite storage should validate: %v", err)
	}
	if err := (Storage{Driver: DriverSQLite, Path: "postgres://u@h/db"}).Validate(); err == nil {
		t.Fatal("a connection string in the SQLite path should be rejected")
	}
	postgres := Storage{Driver: DriverPostgres, Postgres: Postgres{Host: "/var/run/postgresql", Database: "panel", User: "tm"}}
	if err := postgres.Validate(); err != nil {
		t.Fatalf("socket storage should validate: %v", err)
	}
	badSSL := Storage{Driver: DriverPostgres, Postgres: Postgres{Host: "db", Database: "panel", User: "tm", SSLMode: "sometimes"}}
	if err := badSSL.Validate(); err == nil {
		t.Fatal("an unknown ssl mode should be rejected")
	}
	badPort := Storage{Driver: DriverPostgres, Postgres: Postgres{Host: "db", Port: 70000, Database: "panel", User: "tm"}}
	if err := badPort.Validate(); err == nil {
		t.Fatal("an out-of-range port should be rejected")
	}
}

// The browser must never see a stored password, but it does need to know that
// one exists so a blank field can mean "keep it".
func TestRedactedKeepsNoSecret(t *testing.T) {
	storage := Storage{Driver: DriverPostgres, Postgres: Postgres{Host: "db", Port: 5432, Database: "panel", User: "tm", Password: "s3cret"}}
	redacted := storage.Redacted()
	if redacted.Postgres.Password != "" {
		t.Fatal("redacted storage still carries the password")
	}
	if !storage.HasSecret() {
		t.Fatal("stored password should be reported as a secret")
	}
	if redacted.HasSecret() {
		t.Fatal("redacted storage should not report a secret")
	}
	env := Storage{Driver: DriverPostgres, URL: "postgres://tm:s3cret@db:5432/panel"}
	if got := env.Redacted().URL; strings.Contains(got, "s3cret") {
		t.Fatalf("environment DSN leaked its password: %q", got)
	}
}

// Resolve lets the environment win over the wizard's document, which is what
// keeps container deployments in control of their storage.
func TestResolvePrefersEnvironment(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("DATA_DIR", dir)
	t.Setenv("STORE_PATH", "")
	t.Setenv("DATABASE_URL", "")
	if err := Save(Default()); err != nil {
		t.Fatalf("save setup document: %v", err)
	}
	state, err := Resolve()
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if state.Source != SourceFile || state.EnvLocked {
		t.Fatalf("document should drive the storage, got source=%q locked=%v", state.Source, state.EnvLocked)
	}

	t.Setenv("DATABASE_URL", "postgres://tm@db:5432/panel")
	state, err = Resolve()
	if err != nil {
		t.Fatalf("resolve with environment: %v", err)
	}
	if state.Source != SourceEnv || !state.EnvLocked || state.Storage.Driver != DriverPostgres {
		t.Fatalf("environment should win, got source=%q locked=%v driver=%q", state.Source, state.EnvLocked, state.Storage.Driver)
	}
}

// Save writes owner-only permissions so a stored database password stays
// private on a shared host.
func TestSaveUsesOwnerOnlyPermissions(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("DATA_DIR", dir)
	if err := Save(Storage{Driver: DriverSQLite, Path: filepath.Join(dir, "panel.db")}); err != nil {
		t.Fatalf("save: %v", err)
	}
	info, err := os.Stat(filepath.Join(dir, FileName))
	if err != nil {
		t.Fatalf("stat setup document: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Fatalf("setup document mode = %o, want 600", perm)
	}
}
