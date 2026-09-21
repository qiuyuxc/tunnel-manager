// Package setup persists the storage backend chosen by the first-run wizard so
// an instance can be installed without editing environment variables.
package setup

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"tunnel-manager/db"
)

// Driver is the storage backend an instance runs on.
type Driver string

const (
	// DriverSQLite is the embedded single-file backend.
	DriverSQLite Driver = "sqlite"
	// DriverPostgres is an external PostgreSQL server.
	DriverPostgres Driver = "postgres"
)

// DefaultPort is the PostgreSQL port the wizard suggests.
const DefaultPort = 5432

// FileName is the setup document written into the data directory.
const FileName = "setup.json"

// Source records where the storage configuration came from. Environment
// variables always win so container deployments stay in control.
type Source string

const (
	// SourceUnset means nothing is configured and the wizard must choose.
	SourceUnset Source = "unset"
	// SourceEnv means STORE_PATH or DATABASE_URL decided it.
	SourceEnv Source = "env"
	// SourceFile means the wizard wrote the setup document.
	SourceFile Source = "file"
)

// Postgres holds the connection fields the wizard asks for.
type Postgres struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Database string `json:"database"`
	User     string `json:"user"`
	Password string `json:"password"`
	SSLMode  string `json:"ssl_mode"`
}

// DSN renders the fields as a postgres:// URL.
func (p Postgres) DSN() string {
	port := p.Port
	if port == 0 {
		port = DefaultPort
	}
	sslMode := p.SSLMode
	if sslMode == "" {
		sslMode = "disable"
	}
	query := url.Values{"sslmode": {sslMode}}
	conn := url.URL{
		Scheme:   "postgres",
		User:     url.UserPassword(p.User, p.Password),
		Path:     "/" + p.Database,
		RawQuery: query.Encode(),
	}
	if strings.HasPrefix(p.Host, "/") {
		// A Unix socket directory holds a slash, which the URL authority
		// rejects (percent-encoding it fails too), so it travels as the host
		// parameter pgx understands.
		query.Set("host", p.Host)
		query.Set("port", strconv.Itoa(port))
		conn.RawQuery = query.Encode()
		return conn.String()
	}
	conn.Host = p.Host + ":" + strconv.Itoa(port)
	return conn.String()
}

// Storage is the storage configuration of one instance.
type Storage struct {
	Driver Driver `json:"driver"`
	// Path is the SQLite database file.
	Path string `json:"path,omitempty"`
	// URL is a ready-made PostgreSQL connection string, used when the DSN came
	// from the environment or was typed as a whole.
	URL string `json:"url,omitempty"`
	// Postgres is the structured form the wizard fills in; it wins over URL
	// because the wizard can only edit fields it understands.
	Postgres Postgres `json:"postgres,omitempty"`
}

// DSN returns the connection string for the configured storage. A legacy
// .json SQLite path resolves to the database file beside it, so testing and
// installing always touch the same file.
func (s Storage) DSN() string {
	if s.Driver == DriverPostgres {
		if s.Postgres.Host != "" {
			return s.Postgres.DSN()
		}
		return strings.TrimSpace(s.URL)
	}
	return db.ResolvePath(strings.TrimSpace(s.Path))
}

// Validate reports whether the storage can be used as configured.
func (s Storage) Validate() error {
	switch s.Driver {
	case DriverSQLite:
		if strings.TrimSpace(s.Path) == "" {
			return errors.New("请填写 SQLite 数据库文件路径")
		}
		if db.IsPostgresDSN(s.Path) {
			return errors.New("SQLite 路径不能是 postgres:// 连接串")
		}
		return nil
	case DriverPostgres:
		if s.Postgres.Host == "" {
			return errors.New("请填写 PostgreSQL 主机地址")
		}
		if s.Postgres.Database == "" {
			return errors.New("请填写数据库名")
		}
		if s.Postgres.User == "" {
			return errors.New("请填写数据库用户名")
		}
		if s.Postgres.Port != 0 && (s.Postgres.Port < 1 || s.Postgres.Port > 65535) {
			return errors.New("PostgreSQL 端口需在 1-65535 之间")
		}
		switch s.Postgres.SSLMode {
		case "", "disable", "allow", "prefer", "require", "verify-ca", "verify-full":
		default:
			return fmt.Errorf("不支持的 SSL 模式 %q", s.Postgres.SSLMode)
		}
		return nil
	default:
		return fmt.Errorf("未知的数据库类型 %q", s.Driver)
	}
}

// Redacted returns a copy safe to send to the browser.
func (s Storage) Redacted() Storage {
	out := s
	out.URL = ""
	out.Postgres.Password = ""
	if s.Driver == DriverPostgres && s.Postgres.Host == "" {
		// A DSN from the environment: keep its shape but drop the credentials.
		out.URL = redactURL(s.URL)
	}
	return out
}

// HasSecret reports whether a password is already stored, so the UI can show
// that a blank field keeps the existing one.
func (s Storage) HasSecret() bool {
	return s.Postgres.Password != "" || (s.Driver == DriverPostgres && s.URL != "")
}

var redactedSuffix = "://***"

// redactURL strips credentials from a connection string.
func redactURL(raw string) string {
	parsed, err := url.Parse(raw)
	if err != nil {
		return redactedSuffix
	}
	return parsed.Scheme + redactedSuffix + parsed.Host + parsed.Path
}

// State is the resolved storage configuration plus how it was decided.
type State struct {
	Storage Storage
	Source  Source
	// EnvLocked is true when environment variables, not the wizard, own the
	// storage configuration.
	EnvLocked bool
}

// Dir returns the directory holding the data files: DATA_DIR when set,
// otherwise "data" beside the working directory.
func Dir() string {
	if dir := strings.TrimSpace(os.Getenv("DATA_DIR")); dir != "" {
		return dir
	}
	return "data"
}

// File returns the path of the setup document.
func File() string {
	return filepath.Join(Dir(), FileName)
}

// Default returns the storage a fresh install starts from: a SQLite file in
// the data directory.
func Default() Storage {
	return Storage{Driver: DriverSQLite, Path: filepath.Join(Dir(), "tunnel-manager.db")}
}

// Resolve decides which storage the process should use, preferring the
// environment over the wizard's document.
func Resolve() (State, error) {
	if raw := strings.TrimSpace(os.Getenv("DATABASE_URL")); raw != "" {
		return State{Storage: Storage{Driver: DriverPostgres, URL: raw}, Source: SourceEnv, EnvLocked: true}, nil
	}
	if raw := strings.TrimSpace(os.Getenv("STORE_PATH")); raw != "" {
		storage := Storage{Driver: DriverSQLite, Path: db.ResolvePath(raw)}
		if db.IsPostgresDSN(raw) {
			storage = Storage{Driver: DriverPostgres, URL: raw}
		}
		return State{Storage: storage, Source: SourceEnv, EnvLocked: true}, nil
	}
	storage, ok, err := Load()
	if err != nil {
		return State{}, err
	}
	if !ok {
		return State{Storage: Default(), Source: SourceUnset}, nil
	}
	return State{Storage: storage, Source: SourceFile}, nil
}

// Load reads the wizard's storage document.
func Load() (Storage, bool, error) {
	path := File()
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return Storage{}, false, nil
	}
	if err != nil {
		return Storage{}, false, err
	}
	var stored Storage
	if err := json.Unmarshal(data, &stored); err != nil {
		return Storage{}, false, fmt.Errorf("解析 %s: %w", path, err)
	}
	return stored, true, nil
}

// Save writes the storage document atomically with owner-only permissions.
func Save(storage Storage) error {
	dir := Dir()
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(storage, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	temp, err := os.CreateTemp(dir, ".setup-*.json")
	if err != nil {
		return err
	}
	defer os.Remove(temp.Name())
	if err := temp.Chmod(0o600); err != nil {
		temp.Close()
		return err
	}
	if _, err := temp.Write(data); err != nil {
		temp.Close()
		return err
	}
	if err := temp.Close(); err != nil {
		return err
	}
	return os.Rename(temp.Name(), File())
}

// Remove deletes the wizard's document so a failed install can ask again.
func Remove() error {
	err := os.Remove(File())
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return err
}

// Check opens the storage and reports whether the server answers, so the
// wizard can validate an entry before installing with it.
func Check(storage Storage) error {
	if err := storage.Validate(); err != nil {
		return err
	}
	handle, err := db.Open(storage.DSN())
	if err != nil {
		return err
	}
	defer handle.Close()
	return nil
}

// EnsureDataDir creates the data directory (uploads and heartbeat logs live
// there) and returns it.
func EnsureDataDir(storage Storage, source Source) (string, error) {
	dir := DataDir(storage, source)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", err
	}
	return dir, nil
}

// DataDir returns the directory uploads and heartbeat logs belong to. It
// follows the storage the way the rest of the server does, so an existing
// deployment keeps writing beside its SQLite file.
func DataDir(storage Storage, source Source) string {
	if dir := strings.TrimSpace(os.Getenv("DATA_DIR")); dir != "" {
		return dir
	}
	if source == SourceEnv {
		if raw := strings.TrimSpace(os.Getenv("STORE_PATH")); raw != "" && !db.IsPostgresDSN(raw) {
			return filepath.Dir(raw)
		}
		return Dir()
	}
	if storage.Driver == DriverSQLite && storage.Path != "" {
		return filepath.Dir(storage.Path)
	}
	return Dir()
}
