package store

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
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

	"golang.org/x/crypto/argon2"

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

// Store manages application state with JSON file persistence
type Store struct {
	mu       sync.RWMutex
	filePath string
	config   models.Config
}

// NewStore creates a new Store with the given file path
func NewStore(filePath string) *Store {
	s := &Store{
		filePath: filePath,
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

func (s *Store) load() {
	data, err := os.ReadFile(s.filePath)
	if err != nil {
		// First run: use ADMIN_PASSWORD env var if set, otherwise generate random
		password := os.Getenv("ADMIN_PASSWORD")
		if password == "" {
			password = generateRandomPassword(12)
		}
		s.config.AdminPasswordHash = hashPassword(password)
		if err := s.saveLocked(); err != nil {
			log.Printf("save initial administrator password: %v", err)
		}
		log.Printf("========================================")
		log.Printf("  首次启动，已生成管理员账户：")
		log.Printf("  用户名: %s", s.config.AdminUsername)
		log.Printf("  密  码: %s", password)
		log.Printf("  请登录后立即修改密码！")
		log.Printf("========================================")
		return
	}
	var loaded models.Config
	if err := json.Unmarshal(data, &loaded); err != nil {
		log.Printf("load configuration: %v", err)
	} else {
		s.config = loaded
	}
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
	if s.config.AdminPasswordHash == "" {
		password := os.Getenv("ADMIN_PASSWORD")
		if password == "" {
			password = generateRandomPassword(12)
		}
		s.config.AdminPasswordHash = hashPassword(password)
		if err := s.saveLocked(); err != nil {
			log.Printf("save generated administrator password: %v", err)
		}
		log.Printf("========================================")
		log.Printf("  密码为空，已自动生成：")
		log.Printf("  用户名: %s", s.config.AdminUsername)
		log.Printf("  密  码: %s", password)
		log.Printf("========================================")
	}
}

// saveLocked persists the current configuration. The caller must hold s.mu for
// runtime mutations; load may call it before the store is published.
func (s *Store) saveLocked() error {
	data, err := json.MarshalIndent(s.config, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal configuration: %w", err)
	}

	dir := filepath.Dir(s.filePath)
	temp, err := os.CreateTemp(dir, ".config-*")
	if err != nil {
		return fmt.Errorf("create temporary configuration: %w", err)
	}
	tempPath := temp.Name()
	cleaned := false
	defer func() {
		if !cleaned {
			_ = temp.Close()
			_ = os.Remove(tempPath)
		}
	}()
	if err := temp.Chmod(0600); err != nil {
		return fmt.Errorf("set temporary configuration permissions: %w", err)
	}
	if _, err := temp.Write(data); err != nil {
		return fmt.Errorf("write temporary configuration: %w", err)
	}
	if err := temp.Sync(); err != nil {
		return fmt.Errorf("sync temporary configuration: %w", err)
	}
	if err := temp.Close(); err != nil {
		return fmt.Errorf("close temporary configuration: %w", err)
	}
	if err := os.Rename(tempPath, s.filePath); err != nil {
		return fmt.Errorf("replace configuration: %w", err)
	}
	cleaned = true

	directory, err := os.Open(dir)
	if err == nil {
		if syncErr := directory.Sync(); syncErr != nil {
			log.Printf("sync configuration directory: %v", syncErr)
		}
		_ = directory.Close()
	}
	return nil
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
	return config
}

// SetTunnelSelection sets the active tunnel and its display name.
func (s *Store) SetTunnelSelection(id, name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	previous := s.config
	s.config.TunnelID = id
	s.config.TunnelName = name
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

// GetAdminCredentials returns admin username and password hash
func (s *Store) GetAdminCredentials() (string, string) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.config.AdminUsername, s.config.AdminPasswordHash
}

// SetAdminCredentials sets admin username and password hash.
func (s *Store) SetAdminCredentials(username, passwordHash string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	previous := s.config
	s.config.AdminUsername = username
	s.config.AdminPasswordHash = passwordHash
	if err := s.saveLocked(); err != nil {
		s.config = previous
		return err
	}
	return nil
}

// SetAdminPasswordHash changes the password without overwriting a concurrent username update.
func (s *Store) SetAdminPasswordHash(passwordHash string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	previous := s.config.AdminPasswordHash
	s.config.AdminPasswordHash = passwordHash
	if err := s.saveLocked(); err != nil {
		s.config.AdminPasswordHash = previous
		return err
	}
	return nil
}

// SetAdminUsername changes only the administrator username. It intentionally
// leaves the current password hash untouched, including a migrated Argon2id hash.
func (s *Store) SetAdminUsername(username string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	previous := s.config
	s.config.AdminUsername = username
	if err := s.saveLocked(); err != nil {
		s.config = previous
		return err
	}
	return nil
}

// ValidatePassword checks a plaintext password against the stored hash. Successful
// validation of a legacy SHA-256 digest upgrades the stored hash to Argon2id.
func (s *Store) ValidatePassword(password, encodedHash string) bool {
	valid, legacy := verifyPassword(password, encodedHash)
	if !valid || !legacy {
		return valid
	}

	// Do not overwrite a password that changed between credential lookup and validation.
	s.mu.Lock()
	defer s.mu.Unlock()
	if subtle.ConstantTimeCompare([]byte(s.config.AdminPasswordHash), []byte(encodedHash)) == 1 {
		previous := s.config.AdminPasswordHash
		s.config.AdminPasswordHash = HashPassword(password)
		if err := s.saveLocked(); err != nil {
			s.config.AdminPasswordHash = previous
			log.Printf("save migrated administrator password hash: %v", err)
		}
	}
	return true
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

// GetTOTPState returns the persisted TOTP state without exposing recovery hashes.
func (s *Store) GetTOTPState() (enabled bool, encryptedSecret string, lastStep int64, recoveryCount int) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.config.TOTPEnabled, s.config.TOTPSecretEncrypted,
		s.config.TOTPLastAcceptedStep, len(s.config.TOTPRecoveryCodeHashes)
}

// EnableTOTP atomically persists a confirmed TOTP setup. The accepted setup
// step is recorded so the confirmation code cannot be replayed.
func (s *Store) EnableTOTP(encryptedSecret string, recoveryHashes []string, acceptedStep int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.config.TOTPEnabled {
		return ErrTOTPAlreadyEnabled
	}

	previous := s.config
	s.config.TOTPEnabled = true
	s.config.TOTPSecretEncrypted = encryptedSecret
	s.config.TOTPRecoveryCodeHashes = append([]string(nil), recoveryHashes...)
	s.config.TOTPLastAcceptedStep = acceptedStep
	if err := s.saveLocked(); err != nil {
		s.config = previous
		return err
	}
	return nil
}

// AdvanceTOTPStep records a newer accepted TOTP step and rejects replayed or
// older codes.
func (s *Store) AdvanceTOTPStep(step int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.config.TOTPEnabled {
		return ErrTOTPDisabled
	}
	if step <= s.config.TOTPLastAcceptedStep {
		return ErrTOTPReplay
	}

	previous := s.config.TOTPLastAcceptedStep
	s.config.TOTPLastAcceptedStep = step
	if err := s.saveLocked(); err != nil {
		s.config.TOTPLastAcceptedStep = previous
		return err
	}
	return nil
}

// ConsumeRecoveryCode removes one matching candidate hash. Comparison is
// constant-time and the candidate must already be hashed by the handler.
func (s *Store) ConsumeRecoveryCode(candidateHash string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.config.TOTPEnabled {
		return ErrTOTPDisabled
	}

	match := -1
	for i, storedHash := range s.config.TOTPRecoveryCodeHashes {
		if subtle.ConstantTimeCompare([]byte(storedHash), []byte(candidateHash)) == 1 {
			match = i
		}
	}
	if match < 0 {
		return ErrRecoveryCodeNotFound
	}

	previous := s.config.TOTPRecoveryCodeHashes
	remaining := make([]string, 0, len(previous)-1)
	remaining = append(remaining, previous[:match]...)
	remaining = append(remaining, previous[match+1:]...)
	s.config.TOTPRecoveryCodeHashes = remaining
	if err := s.saveLocked(); err != nil {
		s.config.TOTPRecoveryCodeHashes = previous
		return err
	}
	return nil
}

// DisableTOTPWithStep disables TOTP when presented with a newer valid step.
func (s *Store) DisableTOTPWithStep(step int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.config.TOTPEnabled {
		return ErrTOTPDisabled
	}
	if step <= s.config.TOTPLastAcceptedStep {
		return ErrTOTPReplay
	}
	return s.disableTOTPLocked()
}

// DisableTOTPWithRecovery disables TOTP using one stored recovery hash.
func (s *Store) DisableTOTPWithRecovery(candidateHash string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.config.TOTPEnabled {
		return ErrTOTPDisabled
	}

	matched := 0
	for _, storedHash := range s.config.TOTPRecoveryCodeHashes {
		matched |= subtle.ConstantTimeCompare([]byte(storedHash), []byte(candidateHash))
	}
	if matched != 1 {
		return ErrRecoveryCodeNotFound
	}
	return s.disableTOTPLocked()
}

func (s *Store) disableTOTPLocked() error {
	previous := s.config
	s.config.TOTPEnabled = false
	s.config.TOTPSecretEncrypted = ""
	s.config.TOTPRecoveryCodeHashes = nil
	s.config.TOTPLastAcceptedStep = 0
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
