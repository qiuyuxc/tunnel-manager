package handlers

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"tunnel-manager/db"
	"tunnel-manager/setup"
	"tunnel-manager/store"
)

// SetupHandler serves the first-run wizard: it reports the storage an instance
// would use, checks a candidate database, and installs the panel with the
// administrator account the operator typed. It is only mounted while the
// instance is uninstalled.
type SetupHandler struct {
	state      setup.State
	restart    func()
	mu         sync.Mutex
	installing bool
}

// NewSetupHandler wires the wizard to the storage resolved at startup; restart
// re-executes the binary once installation finishes.
func NewSetupHandler(state setup.State, restart func()) *SetupHandler {
	return &SetupHandler{state: state, restart: restart}
}

// setupStoragePayload is the wizard's storage form, flattened so the browser
// never has to build a connection string itself.
type setupStoragePayload struct {
	Driver   string `json:"driver"`
	Path     string `json:"path"`
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Database string `json:"database"`
	User     string `json:"user"`
	Password string `json:"password"`
	SSLMode  string `json:"ssl_mode"`
}

// Status reports whether the wizard is needed and what it should prefill.
func (h *SetupHandler) Status(w http.ResponseWriter, r *http.Request) {
	storage := h.state.Storage.Redacted()
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"needs_setup": true,
		"source":      h.state.Source,
		"env_locked":  h.state.EnvLocked,
		"storage":     storage,
		"has_secret":  h.state.Storage.HasSecret(),
		"data_dir":    setup.Dir(),
		"setup_file":  setup.File(),
	})
}

// Test checks a candidate storage without installing anything.
func (h *SetupHandler) Test(w http.ResponseWriter, r *http.Request) {
	var payload setupStoragePayload
	if err := readJSON(r, &payload); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "请求格式不正确"})
		return
	}
	storage, err := h.resolve(payload)
	if err != nil {
		writeJSON(w, http.StatusOK, map[string]interface{}{"ok": false, "error": err.Error()})
		return
	}
	if err := checkCandidate(storage); err != nil {
		writeJSON(w, http.StatusOK, map[string]interface{}{"ok": false, "error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"ok": true, "message": "连接成功"})
}

// Complete installs the panel: it stores the chosen database, applies the
// schema and creates the administrator account.
func (h *SetupHandler) Complete(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		Storage setupStoragePayload `json:"storage"`
		Admin   struct {
			Username string `json:"username"`
			Password string `json:"password"`
		} `json:"admin"`
	}
	if err := readJSON(r, &payload); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "请求格式不正确"})
		return
	}
	username := strings.TrimSpace(payload.Admin.Username)
	password := payload.Admin.Password
	if username == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "请填写管理员用户名"})
		return
	}
	if len(password) < 6 || len(password) > 1024 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "密码长度需在 6-1024 位之间"})
		return
	}
	storage, err := h.resolve(payload.Storage)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	if err := checkCandidate(storage); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	h.mu.Lock()
	defer h.mu.Unlock()
	if h.installing {
		writeJSON(w, http.StatusConflict, map[string]string{"error": "安装正在进行，请稍候"})
		return
	}
	h.installing = true

	if !h.state.EnvLocked {
		if err := setup.Save(storage); err != nil {
			h.installing = false
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "保存安装配置失败：" + err.Error()})
			return
		}
	}
	if err := installStorage(storage, username, password); err != nil {
		if !h.state.EnvLocked {
			// Leave no half-written configuration behind: an unusable
			// document would make the next boot skip the wizard.
			_ = removeSetupFile()
		}
		h.installing = false
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "初始化数据库失败：" + err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"ok":         true,
		"restarting": true,
		"message":    "初始化完成，正在启动面板…",
	})
	if h.restart != nil {
		go h.restart()
	}
}

// resolve turns a form payload into storage, keeping the stored password when
// the field is left blank and ignoring the form entirely when the environment
// owns the configuration.
func (h *SetupHandler) resolve(payload setupStoragePayload) (setup.Storage, error) {
	if h.state.EnvLocked {
		return h.state.Storage, nil
	}
	switch payload.Driver {
	case string(setup.DriverPostgres):
		password := payload.Password
		if password == "" {
			password = h.state.Storage.Postgres.Password
		}
		port := payload.Port
		if port == 0 {
			port = setup.DefaultPort
		}
		storage := setup.Storage{
			Driver: setup.DriverPostgres,
			Postgres: setup.Postgres{
				Host:     strings.TrimSpace(payload.Host),
				Port:     port,
				Database: strings.TrimSpace(payload.Database),
				User:     strings.TrimSpace(payload.User),
				Password: password,
				SSLMode:  strings.TrimSpace(payload.SSLMode),
			},
		}
		return storage, storage.Validate()
	case string(setup.DriverSQLite):
		path := strings.TrimSpace(payload.Path)
		if path == "" {
			path = setup.Default().Path
		}
		storage := setup.Storage{Driver: setup.DriverSQLite, Path: path}
		return storage, storage.Validate()
	default:
		return setup.Storage{}, fmt.Errorf("未知的数据库类型 %q", payload.Driver)
	}
}

// checkCandidate reports whether a database answers, with a message that says
// which backend failed.
func checkCandidate(storage setup.Storage) error {
	if storage.Driver == setup.DriverSQLite {
		if err := setup.Check(storage); err != nil {
			return fmt.Errorf("无法使用 SQLite 数据库：%w", err)
		}
		return nil
	}
	if err := setup.Check(storage); err != nil {
		return fmt.Errorf("无法连接 PostgreSQL：%w", err)
	}
	return nil
}

// installStorage opens the chosen database, applies the schema and creates the
// administrator account.
func installStorage(storage setup.Storage, username, password string) error {
	dsn := storage.DSN()
	if !db.IsPostgresDSN(dsn) {
		if err := ensureParentDir(dsn); err != nil {
			return err
		}
	}
	st := store.NewStore(dsn)
	if st.Installed() {
		return errors.New("该数据库已有账户，请直接登录或换一个数据库")
	}
	return st.CreateAdmin(username, password)
}

// removeSetupFile drops the wizard's document after a failed install.
func removeSetupFile() error {
	return setup.Remove()
}

// ensureParentDir creates the directory a SQLite file will live in.
func ensureParentDir(path string) error {
	dir := filepath.Dir(path)
	if dir == "" || dir == "." {
		return nil
	}
	return os.MkdirAll(dir, 0o700)
}
