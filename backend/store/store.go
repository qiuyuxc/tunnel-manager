package store

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/argon2"

	"tunnel-manager/db"
	"tunnel-manager/models"
)

const (
	argon2Time       uint32 = 1
	argon2Memory     uint32 = 19 * 1024
	argon2Threads    uint8  = 1
	argon2KeyLen     uint32 = 32
	argon2SaltLen           = 16
	maxArgon2Memory         = 32 * 1024
	maxArgon2Time           = 10
	maxArgon2Threads        = 16
	maxArgon2SaltLen        = 64
	maxArgon2KeyLen         = 64
)

var (
	ErrTOTPDisabled         = errors.New("TOTP is disabled")
	ErrTOTPAlreadyEnabled   = errors.New("TOTP is already enabled")
	ErrTOTPReplay           = errors.New("TOTP step has already been accepted")
	ErrRecoveryCodeNotFound = errors.New("recovery code is invalid or already used")
)

// sessionRecord is one authenticated session, stored by token hash.
type sessionRecord struct {
	TokenHash string
	UserID    string
	CreatedAt int64
	ExpiresAt int64
}

// verifyCodeRecord is one hashed email verification code.
type verifyCodeRecord struct {
	Email     string
	Purpose   string
	CodeHash  string
	CreatedAt int64
	ExpiresAt int64
}

// Store manages application state with SQLite persistence and an in-memory
// cache. All accessor methods operate on the cache; saveLocked persists the
// full state inside a single transaction.
type Store struct {
	mu          sync.RWMutex
	filePath    string
	config      models.Config
	users       []models.User
	groups      []models.UserGroup
	invites     []models.Invite
	sessions    []sessionRecord
	verifyCodes []verifyCodeRecord
	prefs       map[string]models.UserPrefs
	appSettings models.AppSettings
	smtp        models.SMTPSettings
	adminID     string
}

// settingsKey is the app_settings row holding the flat settings document.
const settingsKey = "config"

// NewStore creates a new Store backed by the SQLite database at the given
// path. A path ending in .json is treated as a legacy configuration file:
// the database is stored next to it with a .db extension and the JSON
// document is imported once on first run.
func NewStore(filePath string) *Store {
	resolved := ResolveDBPath(filePath)
	if dir := filepath.Dir(resolved); dir != "" {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			log.Fatalf("create store directory: %v", err)
		}
	}
	s := &Store{
		filePath: resolved,
		prefs:    map[string]models.UserPrefs{},
		config: models.Config{
			PreferredCNAME: "cf.090227.xyz",
			CNAMEPresets: []models.CNAMEPreset{
				{Name: "默认优选", Value: "cf.090227.xyz"},
			},
			SiteName:        "Tunnel Manager",
			SiteDescription: "Cloudflare 隧道管理中心",
			AdminUsername:   "admin",
			TGMode:          "polling",
			TGApiEndpoint:   "https://api.telegram.org",
		},
	}
	s.load()
	return s
}

// ResolveDBPath maps a legacy .json store path onto the .db database path.
func ResolveDBPath(path string) string {
	if strings.HasSuffix(path, ".json") {
		return strings.TrimSuffix(path, ".json") + ".db"
	}
	return path
}

func (s *Store) load() {
	handle, err := db.Open(s.filePath)
	if err != nil {
		log.Fatalf("open store database %s: %v", s.filePath, err)
	}
	defer handle.Close()
	if err := db.Migrate(handle); err != nil {
		log.Fatalf("migrate store database %s: %v", s.filePath, err)
	}

	fresh, err := s.loadFromDB(handle)
	if err != nil {
		log.Fatalf("load store database %s: %v", s.filePath, err)
	}
	if !fresh {
		s.applyDefaults(false)
	} else if legacy := legacyJSONPath(s.filePath); legacy != "" {
		cfg, err := readLegacyConfig(legacy)
		if err == nil {
			s.config = cfg
			s.applyDefaults(true)
			if err := s.saveLocked(); err != nil {
				log.Fatalf("import legacy configuration: %v", err)
			}
			log.Printf("已从 %s 迁移配置到 SQLite，原文件保留作备份", legacy)
		} else {
			log.Printf("读取旧配置 %s 失败，按全新安装初始化: %v", legacy, err)
			s.bootstrapAdminPassword()
		}
	} else {
		s.bootstrapAdminPassword()
	}
	s.seedUsers()
}

// bootstrapAdminPassword generates the first-run administrator password and
// prints the banner. The account row itself is created by seedUsers.
func (s *Store) bootstrapAdminPassword() {
	password := os.Getenv("ADMIN_PASSWORD")
	if password == "" {
		password = generateRandomPassword(12)
	}
	s.config.AdminPasswordHash = hashPassword(password)
	if err := s.saveLocked(); err != nil {
		log.Fatalf("save initial administrator password: %v", err)
	}
	log.Printf("========================================")
	log.Printf("  首次启动，已生成管理员账户：")
	log.Printf("  用户名: %s", s.config.AdminUsername)
	log.Printf("  密  码: %s", password)
	log.Printf("  请登录后立即修改密码！")
	log.Printf("========================================")
}

// legacyJSONPath returns the legacy JSON configuration to import for a
// database path, or "" when none exists.
func legacyJSONPath(dbPath string) string {
	var candidates []string
	if jsonPath := strings.TrimSuffix(dbPath, ".db") + ".json"; jsonPath != dbPath {
		candidates = append(candidates, jsonPath)
	}
	candidates = append(candidates, filepath.Join(filepath.Dir(dbPath), "config.json"))
	seen := map[string]bool{}
	for _, candidate := range candidates {
		if seen[candidate] {
			continue
		}
		seen[candidate] = true
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
			return candidate
		}
	}
	return ""
}

func readLegacyConfig(path string) (models.Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return models.Config{}, err
	}
	var cfg models.Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return models.Config{}, err
	}
	return cfg, nil
}

// applyDefaults fills legacy/empty fields on the cached configuration. The
// generatePassword flag keeps the first-run password generation out of the
// regular load path, which reports a dedicated banner instead.
func (s *Store) applyDefaults(generatePassword bool) {
	if s.config.PreferredCNAME == "" {
		s.config.PreferredCNAME = "cf.090227.xyz"
	}
	if len(s.config.CNAMEPresets) == 0 {
		s.config.CNAMEPresets = []models.CNAMEPreset{{Name: "默认优选", Value: s.config.PreferredCNAME}}
	}
	if s.config.SiteName == "" {
		s.config.SiteName = "Tunnel Manager"
	}
	if s.config.SiteDescription == "" {
		s.config.SiteDescription = "Cloudflare 隧道管理中心"
	}
	if s.config.AdminUsername == "" {
		s.config.AdminUsername = "admin"
	}
	if s.config.TGMode == "" {
		s.config.TGMode = "polling"
	}
	if s.config.TGApiEndpoint == "" {
		s.config.TGApiEndpoint = "https://api.telegram.org"
	}
	// Administrator credentials are owned by the users table (seedUsers); the
	// settings document no longer carries them.
}

// loadFromDB reads the settings document, CNAME presets and monitors into
// the cache. It reports whether the database is still empty (fresh install).
func (s *Store) loadFromDB(handle *sql.DB) (bool, error) {
	var doc string
	err := handle.QueryRow(`SELECT value FROM app_settings WHERE key = ?`, settingsKey).Scan(&doc)
	if errors.Is(err, sql.ErrNoRows) {
		return true, nil
	}
	if err != nil {
		return false, fmt.Errorf("read settings document: %w", err)
	}
	var stored models.Config
	if err := json.Unmarshal([]byte(doc), &stored); err != nil {
		return false, fmt.Errorf("decode settings document: %w", err)
	}
	stored.CNAMEPresets = nil
	stored.Monitors = nil
	s.config = stored

	presets, err := loadCNAMEPresets(handle)
	if err != nil {
		return false, err
	}
	s.config.CNAMEPresets = presets

	monitors, err := loadMonitors(handle)
	if err != nil {
		return false, err
	}
	s.config.Monitors = monitors

	users, prefs, err := loadUsers(handle)
	if err != nil {
		return false, err
	}
	s.users = users
	if prefs == nil {
		prefs = map[string]models.UserPrefs{}
	}
	s.prefs = prefs
	if s.groups, err = loadGroups(handle); err != nil {
		return false, err
	}
	if s.sessions, err = loadSessions(handle); err != nil {
		return false, err
	}
	if s.invites, err = loadInvites(handle); err != nil {
		return false, err
	}
	if s.verifyCodes, err = loadVerifyCodes(handle); err != nil {
		return false, err
	}
	if appDoc, ok, loadErr := loadSetting(handle, "app"); loadErr != nil {
		return false, loadErr
	} else if ok {
		_ = json.Unmarshal([]byte(appDoc), &s.appSettings)
	}
	if smtpDoc, ok, loadErr := loadSetting(handle, "smtp"); loadErr != nil {
		return false, loadErr
	} else if ok {
		_ = json.Unmarshal([]byte(smtpDoc), &s.smtp)
	}
	return false, nil
}

// saveLocked persists the cached configuration to SQLite in a single
// transaction: the settings document plus full replacements of the CNAME
// preset and monitor tables. The caller must hold s.mu for runtime
// mutations; load may call it before the store is published.
func (s *Store) saveLocked() error {
	handle, err := db.Open(s.filePath)
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	defer handle.Close()

	doc, err := json.Marshal(settingsDocument(s.config))
	if err != nil {
		return fmt.Errorf("marshal settings document: %w", err)
	}

	tx, err := handle.Begin()
	if err != nil {
		return fmt.Errorf("begin configuration transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.Exec(`INSERT INTO app_settings(key, value) VALUES(?, ?)
		ON CONFLICT(key) DO UPDATE SET value = excluded.value`, settingsKey, string(doc)); err != nil {
		return fmt.Errorf("save settings document: %w", err)
	}
	if err := replaceCNAMEPresets(tx, s.config.CNAMEPresets); err != nil {
		return err
	}
	if err := replaceMonitors(tx, s.config.Monitors); err != nil {
		return err
	}
	if err := replaceUsers(tx, s.users, s.prefs); err != nil {
		return err
	}
	if err := replaceGroups(tx, s.groups); err != nil {
		return err
	}
	if err := replaceSessions(tx, s.sessions); err != nil {
		return err
	}
	if err := replaceInvites(tx, s.invites); err != nil {
		return err
	}
	if err := replaceVerifyCodes(tx, s.verifyCodes); err != nil {
		return err
	}
	appJSON, err := json.Marshal(s.appSettings)
	if err != nil {
		return fmt.Errorf("marshal app settings: %w", err)
	}
	if err := upsertSetting(tx, "app", string(appJSON)); err != nil {
		return err
	}
	smtpJSON, err := json.Marshal(s.smtp)
	if err != nil {
		return fmt.Errorf("marshal smtp settings: %w", err)
	}
	if err := upsertSetting(tx, "smtp", string(smtpJSON)); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit configuration: %w", err)
	}
	return nil
}

// settingsDocument returns the flat settings portion of the configuration;
// monitors and CNAME presets live in their own tables.
func settingsDocument(c models.Config) models.Config {
	doc := c
	doc.Monitors = nil
	doc.CNAMEPresets = nil
	return doc
}

func (s *Store) restoreAndLog(previous models.Config, operation string) {
	if err := s.saveLocked(); err != nil {
		s.config = previous
		log.Printf("%s: %v", operation, err)
	}
}

// GetConfig returns the current configuration
func (s *Store) GetConfig() models.Config {
	s.mu.RLock()
	defer s.mu.RUnlock()
	config := s.config
	config.TOTPRecoveryCodeHashes = append([]string(nil), s.config.TOTPRecoveryCodeHashes...)
	config.CNAMEPresets = append([]models.CNAMEPreset(nil), s.config.CNAMEPresets...)
	config.Monitors = append([]models.Monitor(nil), s.config.Monitors...)
	return config
}

// AddMonitor appends a monitor project.
func (s *Store) AddMonitor(m models.Monitor) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	m.CreatedAt = time.Now().Unix()
	if m.IntervalSec == 0 {
		m.IntervalSec = 60
	}
	previous := s.config
	s.config.Monitors = append(s.config.Monitors, m)
	if err := s.saveLocked(); err != nil {
		s.config = previous
		return err
	}
	return nil
}

// ErrMonitorNotFound is returned when a monitor id does not exist.
var ErrMonitorNotFound = errors.New("monitor not found")

// MutateMonitor applies fn to one stored monitor under the lock; fn returns
// false when the id is unknown. Saving failures roll the change back.
func (s *Store) MutateMonitor(id string, fn func(*models.Monitor) bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	idx := -1
	for i := range s.config.Monitors {
		if s.config.Monitors[i].ID == id {
			idx = i
			break
		}
	}
	if idx < 0 {
		return ErrMonitorNotFound
	}
	if !fn(&s.config.Monitors[idx]) {
		return ErrMonitorNotFound
	}
	previous := s.config
	if err := s.saveLocked(); err != nil {
		s.config = previous
		return err
	}
	return nil
}

// RemoveMonitor deletes a monitor project by id. It is implemented directly
// instead of through MutateMonitor, whose callback can only edit one entry
// in place and cannot shrink the slice.
func (s *Store) RemoveMonitor(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	idx := -1
	for i := range s.config.Monitors {
		if s.config.Monitors[i].ID == id {
			idx = i
			break
		}
	}
	if idx < 0 {
		return ErrMonitorNotFound
	}
	previous := s.config
	remaining := append([]models.Monitor(nil), s.config.Monitors...)
	s.config.Monitors = append(remaining[:idx], remaining[idx+1:]...)
	if err := s.saveLocked(); err != nil {
		s.config = previous
		return err
	}
	return nil
}

// FindMonitorByToken locates a published monitor by its public token.
func (s *Store) FindMonitorByToken(token string) (models.Monitor, bool) {
	cfg := s.GetConfig()
	for _, m := range cfg.Monitors {
		slugHit := m.PublicSlug != "" && m.PublicSlug == token
		if m.PublishEnabled && ((m.PublicToken != "" && m.PublicToken == token) || slugHit) {
			return m, true
		}
	}
	return models.Monitor{}, false
}

// SetZoneSelection sets the active zone used by Telegram DNS commands.
func (s *Store) SetZoneSelection(id, name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	previous := s.config
	s.config.SelectedZoneID = id
	s.config.SelectedZoneName = name
	if err := s.saveLocked(); err != nil {
		s.config = previous
		return err
	}
	return nil
}

// SetServiceURL sets the forwarding service URL
func (s *Store) SetServiceURL(url string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	previous := s.config
	s.config.ServiceURL = url
	s.restoreAndLog(previous, "save service URL")
}

// SetPreferredCNAME sets the global preferred CNAME
func (s *Store) SetPreferredCNAME(cname string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	previous := s.config
	s.config.PreferredCNAME = cname
	s.restoreAndLog(previous, "save preferred CNAME")
}

// SetSiteSettings updates public-facing site branding.
func (s *Store) SetSiteSettings(name, description, icon string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	previous := s.config
	s.config.SiteName = name
	s.config.SiteDescription = description
	s.config.SiteIcon = icon
	if err := s.saveLocked(); err != nil {
		s.config = previous
		return err
	}
	return nil
}

// SetCNAMEPresets replaces the reusable CNAME options.
func (s *Store) SetCNAMEPresets(items []models.CNAMEPreset) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	previous := s.config
	s.config.CNAMEPresets = append([]models.CNAMEPreset(nil), items...)
	if err := s.saveLocked(); err != nil {
		s.config = previous
		return err
	}
	return nil
}

// HashPassword returns an Argon2id PHC string for a password.
func HashPassword(password string) string {
	salt := make([]byte, argon2SaltLen)
	if _, err := rand.Read(salt); err != nil {
		panic(fmt.Sprintf("generate password salt: %v", err))
	}
	key := argon2.IDKey([]byte(password), salt, argon2Time, argon2Memory, argon2Threads, argon2KeyLen)
	return fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version, argon2Memory, argon2Time, argon2Threads,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(key))
}

func verifyPassword(password, encodedHash string) (valid, legacy bool) {
	if isLegacySHA256(encodedHash) {
		digest := sha256.Sum256([]byte(password))
		expected, _ := hex.DecodeString(encodedHash)
		return subtle.ConstantTimeCompare(digest[:], expected) == 1, true
	}

	parts := strings.Split(encodedHash, "$")
	if len(parts) != 6 || parts[0] != "" || parts[1] != "argon2id" {
		return false, false
	}
	if parts[2] != "v="+strconv.Itoa(argon2.Version) {
		return false, false
	}

	var memory, timeCost uint32
	var threads uint8
	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &memory, &timeCost, &threads); err != nil ||
		parts[3] != fmt.Sprintf("m=%d,t=%d,p=%d", memory, timeCost, threads) ||
		memory == 0 || memory > maxArgon2Memory || timeCost == 0 || timeCost > maxArgon2Time || threads == 0 || threads > maxArgon2Threads {
		return false, false
	}
	if len(parts[4]) > base64.RawStdEncoding.EncodedLen(maxArgon2SaltLen) || len(parts[5]) > base64.RawStdEncoding.EncodedLen(maxArgon2KeyLen) {
		return false, false
	}
	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil || len(salt) == 0 || len(salt) > maxArgon2SaltLen {
		return false, false
	}
	expected, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil || len(expected) == 0 || len(expected) > maxArgon2KeyLen {
		return false, false
	}
	actual := argon2.IDKey([]byte(password), salt, timeCost, memory, threads, uint32(len(expected)))
	return subtle.ConstantTimeCompare(actual, expected) == 1, false
}

func isLegacySHA256(encodedHash string) bool {
	if len(encodedHash) != sha256.Size*2 {
		return false
	}
	_, err := hex.DecodeString(encodedHash)
	return err == nil
}

// hashPassword is an internal alias for convenience
func hashPassword(password string) string {
	return HashPassword(password)
}

// SetCloudflareOAuth stores encrypted OAuth credentials and their expiry.
func (s *Store) SetCloudflareOAuth(accessToken, refreshToken string, expiresAt time.Time, scope string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	previous := s.config
	s.config.CFOAuthAccessToken = accessToken
	s.config.CFOAuthRefreshToken = refreshToken
	if expiresAt.IsZero() {
		s.config.CFOAuthExpiresAt = 0
	} else {
		s.config.CFOAuthExpiresAt = expiresAt.Unix()
	}
	s.config.CFOAuthScope = scope
	if err := s.saveLocked(); err != nil {
		s.config = previous
		return err
	}
	return nil
}

// SetCloudflareAccount selects the account used for account-scoped API calls.
func (s *Store) SetCloudflareAccount(id, name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	previous := s.config
	if s.config.CFAccountID != id {
		s.config.TunnelID = ""
		s.config.TunnelName = ""
		s.clearAllTunnelSelectionsLocked()
	}
	s.config.CFAccountID = id
	s.config.CFAccountName = name
	if err := s.saveLocked(); err != nil {
		s.config = previous
		return err
	}
	return nil
}

// clearAllTunnelSelectionsLocked drops every user's tunnel selection; the
// caller holds s.mu and saves afterwards.
func (s *Store) clearAllTunnelSelectionsLocked() {
	for uid := range s.prefs {
		prefs := s.prefs[uid]
		prefs.TunnelID = ""
		prefs.TunnelName = ""
		s.prefs[uid] = prefs
	}
}

// ClearCloudflareOAuth removes OAuth credentials and the OAuth account selection.
func (s *Store) ClearCloudflareOAuth() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	previous := s.config
	s.config.CFOAuthAccessToken = ""
	s.config.CFOAuthRefreshToken = ""
	s.config.CFOAuthExpiresAt = 0
	s.config.CFOAuthScope = ""
	s.config.CFAccountID = ""
	s.config.CFAccountName = ""
	s.config.TunnelID = ""
	s.config.TunnelName = ""
	s.clearAllTunnelSelectionsLocked()
	if err := s.saveLocked(); err != nil {
		s.config = previous
		return err
	}
	return nil
}

// SetTelegramSettings saves all bot settings atomically
func (s *Store) SetTelegramSettings(enabled bool, token, adminIDs, mode, webhookURL, apiEndpoint string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	previous := s.config
	s.config.TGBotEnabled = enabled
	if token != "" {
		s.config.TGBotToken = token
	}
	s.config.TGAdminIDs = adminIDs
	if mode != "" {
		s.config.TGMode = mode
	}
	s.config.TGWebhookURL = webhookURL
	if apiEndpoint != "" {
		s.config.TGApiEndpoint = apiEndpoint
	}
	s.restoreAndLog(previous, "save Telegram settings")
}

// SetTelegramWebhookSecret persists the webhook verification secret
func (s *Store) SetTelegramWebhookSecret(secret string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	previous := s.config
	s.config.TGWebhookSecret = secret
	s.restoreAndLog(previous, "save Telegram webhook secret")
}

func generateRandomPassword(length int) string {
	b := make([]byte, length)
	rand.Read(b)
	return hex.EncodeToString(b)[:length]
}

func loadCNAMEPresets(handle *sql.DB) ([]models.CNAMEPreset, error) {
	rows, err := handle.Query(`SELECT name, value FROM cname_presets ORDER BY position`)
	if err != nil {
		return nil, fmt.Errorf("load CNAME presets: %w", err)
	}
	defer rows.Close()
	presets := []models.CNAMEPreset{}
	for rows.Next() {
		var preset models.CNAMEPreset
		if err := rows.Scan(&preset.Name, &preset.Value); err != nil {
			return nil, fmt.Errorf("scan CNAME preset: %w", err)
		}
		presets = append(presets, preset)
	}
	return presets, rows.Err()
}

func replaceCNAMEPresets(tx *sql.Tx, presets []models.CNAMEPreset) error {
	if _, err := tx.Exec(`DELETE FROM cname_presets`); err != nil {
		return fmt.Errorf("clear CNAME presets: %w", err)
	}
	for i, preset := range presets {
		if _, err := tx.Exec(`INSERT INTO cname_presets(position, name, value) VALUES(?, ?, ?)`,
			i, preset.Name, preset.Value); err != nil {
			return fmt.Errorf("save CNAME preset: %w", err)
		}
	}
	return nil
}

const monitorColumns = `id, position, name, interval_sec, publish_enabled, public_token, public_slug, public_title, public_icon, public_theme, announcement, created_at`

func loadMonitors(handle *sql.DB) ([]models.Monitor, error) {
	rows, err := handle.Query(`SELECT ` + monitorColumns + ` FROM monitors ORDER BY position`)
	if err != nil {
		return nil, fmt.Errorf("load monitors: %w", err)
	}
	defer rows.Close()
	monitors := []models.Monitor{}
	for rows.Next() {
		var m models.Monitor
		var publishEnabled int
		if err := rows.Scan(&m.ID, new(int), &m.Name, &m.IntervalSec, &publishEnabled,
			&m.PublicToken, &m.PublicSlug, &m.PublicTitle, &m.PublicIcon, &m.PublicTheme,
			&m.Announcement, &m.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan monitor: %w", err)
		}
		m.PublishEnabled = publishEnabled != 0
		monitors = append(monitors, m)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate monitors: %w", err)
	}

	targetRows, err := handle.Query(`SELECT monitor_id, id, position, name, url, type, method, created_at, link_enabled
		FROM monitor_targets ORDER BY monitor_id, position`)
	if err != nil {
		return nil, fmt.Errorf("load monitor targets: %w", err)
	}
	defer targetRows.Close()
	byID := make(map[string]int, len(monitors))
	for i := range monitors {
		byID[monitors[i].ID] = i
		monitors[i].Targets = []models.MonitorTarget{}
	}
	for targetRows.Next() {
		var monitorID string
		var t models.MonitorTarget
		var linkEnabled int
		if err := targetRows.Scan(&monitorID, &t.ID, new(int), &t.Name, &t.URL, &t.Type, &t.Method, &t.CreatedAt, &linkEnabled); err != nil {
			return nil, fmt.Errorf("scan monitor target: %w", err)
		}
		t.LinkEnabled = linkEnabled != 0
		if idx, ok := byID[monitorID]; ok {
			monitors[idx].Targets = append(monitors[idx].Targets, t)
		}
	}
	if err := targetRows.Err(); err != nil {
		return nil, fmt.Errorf("iterate monitor targets: %w", err)
	}
	return monitors, nil
}

func replaceMonitors(tx *sql.Tx, monitors []models.Monitor) error {
	if _, err := tx.Exec(`DELETE FROM monitor_targets`); err != nil {
		return fmt.Errorf("clear monitor targets: %w", err)
	}
	if _, err := tx.Exec(`DELETE FROM monitors`); err != nil {
		return fmt.Errorf("clear monitors: %w", err)
	}
	for i, m := range monitors {
		if _, err := tx.Exec(`INSERT INTO monitors(`+monitorColumns+`) VALUES(?,?,?,?,?,?,?,?,?,?,?,?)`,
			m.ID, i, m.Name, m.IntervalSec, boolInt(m.PublishEnabled), m.PublicToken, m.PublicSlug,
			m.PublicTitle, m.PublicIcon, m.PublicTheme, m.Announcement, m.CreatedAt); err != nil {
			return fmt.Errorf("save monitor %s: %w", m.ID, err)
		}
		for j, t := range m.Targets {
			if _, err := tx.Exec(`INSERT INTO monitor_targets(id, monitor_id, position, name, url, type, method, created_at, link_enabled)
				VALUES(?,?,?,?,?,?,?,?,?)`,
				t.ID, m.ID, j, t.Name, t.URL, t.Type, t.Method, t.CreatedAt, boolInt(t.LinkEnabled)); err != nil {
				return fmt.Errorf("save monitor target %s: %w", t.ID, err)
			}
		}
	}
	return nil
}

func boolInt(v bool) int {
	if v {
		return 1
	}
	return 0
}
