package store

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"tunnel-manager/models"
)

func TestHashPasswordUsesArgon2idAndValidates(t *testing.T) {
	hash := HashPassword("correct horse battery staple")
	if !strings.HasPrefix(hash, "$argon2id$") {
		t.Fatalf("HashPassword() = %q, want Argon2id PHC string", hash)
	}

	s := newTestStore(t, hash)
	if !s.ValidatePassword("correct horse battery staple", hash) {
		t.Fatal("ValidatePassword() rejected correct Argon2id password")
	}
	if s.ValidatePassword("wrong password", hash) {
		t.Fatal("ValidatePassword() accepted incorrect Argon2id password")
	}
}

func TestValidatePasswordMigratesLegacySHA256(t *testing.T) {
	password := "legacy password"
	legacy := sha256Hex(password)
	s := newTestStore(t, legacy)

	if !s.ValidatePassword(password, legacy) {
		t.Fatal("ValidatePassword() rejected correct legacy password")
	}
	_, migrated := s.GetAdminCredentials()
	if !strings.HasPrefix(migrated, "$argon2id$") {
		t.Fatalf("stored hash = %q, want migrated Argon2id PHC string", migrated)
	}
	if !s.ValidatePassword(password, migrated) {
		t.Fatal("ValidatePassword() rejected migrated password")
	}
}

func TestLegacyConfigWithoutTOTPFieldsIsDisabled(t *testing.T) {
	s := newTestStore(t, HashPassword("password"))
	enabled, secret, step, count := s.GetTOTPState(s.AdminUserID())
	if enabled || secret != "" || step != 0 || count != 0 {
		t.Fatalf("GetTOTPState() = (%v, %q, %d, %d), want disabled zero state", enabled, secret, step, count)
	}
}

func TestEnableTOTPPersistsState(t *testing.T) {
	s, path := newStoreWithConfig(t, models.Config{
		AdminUsername:     "admin",
		AdminPasswordHash: HashPassword("password"),
	})
	hashes := []string{sha256Hex("recovery-one"), sha256Hex("recovery-two")}
	if err := s.EnableTOTP(s.AdminUserID(), "v1:encrypted-secret", hashes, 123); err != nil {
		t.Fatalf("EnableTOTP() error = %v", err)
	}

	reloaded := NewStore(path)
	enabled, secret, step, count := reloaded.GetTOTPState(reloaded.AdminUserID())
	if !enabled || secret != "v1:encrypted-secret" || step != 123 || count != 2 {
		t.Fatalf("persisted state = (%v, %q, %d, %d)", enabled, secret, step, count)
	}
	data, err := os.ReadFile(ResolveDBPath(path))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "recovery-one") || strings.Contains(string(data), "recovery-two") {
		t.Fatal("config contains plaintext recovery code")
	}
}

func TestAdvanceTOTPStepRejectsReplay(t *testing.T) {
	s := newEnabledTOTPStore(t, 50, []string{"hash"})
	if err := s.AdvanceTOTPStep(s.AdminUserID(), 51); err != nil {
		t.Fatalf("AdvanceTOTPStep(51) error = %v", err)
	}
	for _, step := range []int64{51, 50, 49} {
		if err := s.AdvanceTOTPStep(s.AdminUserID(), step); !errors.Is(err, ErrTOTPReplay) {
			t.Errorf("AdvanceTOTPStep(%d) error = %v, want ErrTOTPReplay", step, err)
		}
	}
}

func TestConsumeRecoveryCodeRejectsReuse(t *testing.T) {
	candidate := sha256Hex("candidate")
	s := newEnabledTOTPStore(t, 10, []string{sha256Hex("other"), candidate})
	if err := s.ConsumeRecoveryCode(s.AdminUserID(), candidate); err != nil {
		t.Fatalf("ConsumeRecoveryCode() error = %v", err)
	}
	if err := s.ConsumeRecoveryCode(s.AdminUserID(), candidate); !errors.Is(err, ErrRecoveryCodeNotFound) {
		t.Fatalf("reused ConsumeRecoveryCode() error = %v, want ErrRecoveryCodeNotFound", err)
	}
	_, _, _, count := s.GetTOTPState(s.AdminUserID())
	if count != 1 {
		t.Fatalf("recovery count = %d, want 1", count)
	}
	reloaded := NewStore(s.filePath)
	if err := reloaded.ConsumeRecoveryCode(reloaded.AdminUserID(), candidate); !errors.Is(err, ErrRecoveryCodeNotFound) {
		t.Fatalf("reloaded ConsumeRecoveryCode() error = %v, want ErrRecoveryCodeNotFound", err)
	}
}

func TestConcurrentSameTOTPStepExactlyOneSucceeds(t *testing.T) {
	s := newEnabledTOTPStore(t, 100, []string{"hash"})
	errs := runConcurrently(20, func() error { return s.AdvanceTOTPStep(s.AdminUserID(), 101) })
	assertExactlyOneSuccess(t, errs, ErrTOTPReplay)
}

func TestConcurrentSameRecoveryCodeExactlyOneSucceeds(t *testing.T) {
	candidate := sha256Hex("one-time-code")
	s := newEnabledTOTPStore(t, 100, []string{candidate})
	errs := runConcurrently(20, func() error { return s.ConsumeRecoveryCode(s.AdminUserID(), candidate) })
	assertExactlyOneSuccess(t, errs, ErrRecoveryCodeNotFound)
}

func TestDisableTOTPWithStepClearsAllState(t *testing.T) {
	s := newEnabledTOTPStore(t, 20, []string{"hash-one", "hash-two"})
	if err := s.DisableTOTPWithStep(s.AdminUserID(), 20); !errors.Is(err, ErrTOTPReplay) {
		t.Fatalf("DisableTOTPWithStep(replay) error = %v, want ErrTOTPReplay", err)
	}
	if err := s.DisableTOTPWithStep(s.AdminUserID(), 21); err != nil {
		t.Fatalf("DisableTOTPWithStep() error = %v", err)
	}
	assertTOTPDisabled(t, s)
}

func TestDisableTOTPWithRecoveryClearsAllState(t *testing.T) {
	candidate := sha256Hex("disable-code")
	s := newEnabledTOTPStore(t, 20, []string{candidate})
	if err := s.DisableTOTPWithRecovery(s.AdminUserID(), sha256Hex("wrong")); !errors.Is(err, ErrRecoveryCodeNotFound) {
		t.Fatalf("DisableTOTPWithRecovery(wrong) error = %v, want ErrRecoveryCodeNotFound", err)
	}
	if err := s.DisableTOTPWithRecovery(s.AdminUserID(), candidate); err != nil {
		t.Fatalf("DisableTOTPWithRecovery() error = %v", err)
	}
	assertTOTPDisabled(t, s)
}

func TestAuthSensitiveSaveFailuresRollBack(t *testing.T) {
	candidate := sha256Hex("candidate")
	tests := []struct {
		name   string
		mutate func(*Store) error
	}{
		{"enable", func(s *Store) error { return s.EnableTOTP(s.AdminUserID(), "encrypted", []string{candidate}, 7) }},
		{"advance", func(s *Store) error { return s.AdvanceTOTPStep(s.AdminUserID(), 8) }},
		{"consume", func(s *Store) error { return s.ConsumeRecoveryCode(s.AdminUserID(), candidate) }},
		{"disable step", func(s *Store) error { return s.DisableTOTPWithStep(s.AdminUserID(), 8) }},
		{"disable recovery", func(s *Store) error { return s.DisableTOTPWithRecovery(s.AdminUserID(), candidate) }},
		{"username", func(s *Store) error { return s.SetAdminUsername("changed") }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			initial := models.Config{AdminUsername: "admin", AdminPasswordHash: HashPassword("password")}
			if tt.name != "enable" {
				initial.TOTPEnabled = true
				initial.TOTPSecretEncrypted = "encrypted"
				initial.TOTPRecoveryCodeHashes = []string{candidate}
				initial.TOTPLastAcceptedStep = 7
			}
			s, _ := newStoreWithConfig(t, initial)
			before := cloneConfig(t, s.GetConfig())
			s.filePath = filepath.Join(t.TempDir(), "missing", "config.json")

			if err := tt.mutate(s); err == nil {
				t.Fatal("mutation succeeded with unwritable path")
			}
			after := s.GetConfig()
			if !configsEqual(before, after) {
				t.Fatalf("state changed on save failure\nbefore: %#v\nafter:  %#v", before, after)
			}
		})
	}
}

func TestValidatePasswordMigrationRollsBackOnSaveFailure(t *testing.T) {
	password := "legacy password"
	legacy := sha256Hex(password)
	s := newTestStore(t, legacy)
	s.filePath = filepath.Join(t.TempDir(), "missing", "config.json")

	if !s.ValidatePassword(password, legacy) {
		t.Fatal("ValidatePassword() rejected valid legacy password")
	}
	_, stored := s.GetAdminCredentials()
	if stored != legacy {
		t.Fatalf("stored hash = %q, want legacy hash after failed save", stored)
	}
}

func TestSetAdminUsernamePreservesMigratedPasswordHash(t *testing.T) {
	password := "legacy password"
	legacy := sha256Hex(password)
	s := newTestStore(t, legacy)
	if !s.ValidatePassword(password, legacy) {
		t.Fatal("ValidatePassword() rejected valid legacy password")
	}
	_, migrated := s.GetAdminCredentials()
	if !strings.HasPrefix(migrated, "$argon2id$") {
		t.Fatalf("migrated hash = %q", migrated)
	}

	if err := s.SetAdminUsername("new-admin"); err != nil {
		t.Fatalf("SetAdminUsername() error = %v", err)
	}
	username, after := s.GetAdminCredentials()
	if username != "new-admin" || after != migrated {
		t.Fatalf("credentials = (%q, %q), want username changed and hash preserved", username, after)
	}
	reloaded := NewStore(s.filePath)
	username, after = reloaded.GetAdminCredentials()
	if username != "new-admin" || after != migrated {
		t.Fatalf("persisted credentials = (%q, %q), want username changed and hash preserved", username, after)
	}
}

func TestVerifyPasswordRejectsUnsafeArgon2Parameters(t *testing.T) {
	salt := "c2FsdA"
	key := "a2V5"
	for _, params := range []string{
		"m=32769,t=1,p=1",
		"m=19456,t=11,p=1",
		"m=19456,t=1,p=17",
	} {
		hash := "$argon2id$v=19$" + params + "$" + salt + "$" + key
		if valid, legacy := verifyPassword("password", hash); valid || legacy {
			t.Fatalf("verifyPassword(%q) = (%v, %v)", params, valid, legacy)
		}
	}
	oversizedSalt := strings.Repeat("A", 100)
	if valid, _ := verifyPassword("password", "$argon2id$v=19$m=19456,t=1,p=1$"+oversizedSalt+"$"+key); valid {
		t.Fatal("verifyPassword accepted oversized salt")
	}
	oversizedKey := strings.Repeat("A", 100)
	if valid, _ := verifyPassword("password", "$argon2id$v=19$m=19456,t=1,p=1$"+salt+"$"+oversizedKey); valid {
		t.Fatal("verifyPassword accepted oversized output")
	}
}

func TestGetUserByIDDeepCopiesRecoveryHashes(t *testing.T) {
	s := newEnabledTOTPStore(t, 1, []string{"one", "two"})
	user, ok := s.GetUserByID(s.AdminUserID())
	if !ok {
		t.Fatal("admin user missing")
	}
	user.TOTPRecoveryCodeHashes[0] = "changed"
	again, ok := s.GetUserByID(s.AdminUserID())
	if !ok {
		t.Fatal("admin user missing")
	}
	if got := again.TOTPRecoveryCodeHashes[0]; got != "one" {
		t.Fatalf("stored recovery hash changed through GetUserByID: %q", got)
	}
}

func TestGetConfigDeepCopiesCNAMEPresets(t *testing.T) {
	s, _ := newStoreWithConfig(t, models.Config{
		AdminUsername:     "admin",
		AdminPasswordHash: HashPassword("password"),
		PreferredCNAME:    "default.example.com",
		CNAMEPresets: []models.CNAMEPreset{
			{Name: "线路 A", Value: "a.example.com"},
		},
	})
	config := s.GetConfig()
	config.CNAMEPresets[0].Value = "changed.example.com"
	if got := s.GetConfig().CNAMEPresets[0].Value; got != "a.example.com" {
		t.Fatalf("stored CNAME preset changed through GetConfig: %q", got)
	}
}

func TestLegacyConfigReceivesBrandingAndCNAMEPresetDefaults(t *testing.T) {
	s, _ := newStoreWithConfig(t, models.Config{
		AdminUsername:     "admin",
		AdminPasswordHash: HashPassword("password"),
		PreferredCNAME:    "legacy.example.com",
	})
	config := s.GetConfig()
	if config.SiteName != "Tunnel Manager" || config.SiteDescription == "" {
		t.Fatalf("site defaults = (%q, %q)", config.SiteName, config.SiteDescription)
	}
	if len(config.CNAMEPresets) != 1 || config.CNAMEPresets[0].Value != "legacy.example.com" {
		t.Fatalf("CNAME defaults = %#v", config.CNAMEPresets)
	}
}

func TestSiteCNAMEAndTunnelSettingsPersist(t *testing.T) {
	s := newTestStore(t, HashPassword("password"))
	if err := s.SetSiteSettings("My Panel", "Operations", "https://example.com/icon.png"); err != nil {
		t.Fatal(err)
	}
	presets := []models.CNAMEPreset{{Name: "线路 A", Value: "a.example.com"}}
	if err := s.SetCNAMEPresets(presets); err != nil {
		t.Fatal(err)
	}
	if err := s.SetUserTunnelSelection(s.AdminUserID(), "tunnel-id", "Production"); err != nil {
		t.Fatal(err)
	}

	reloaded := NewStore(s.filePath)
	config := reloaded.GetConfig()
	if config.SiteName != "My Panel" || config.SiteDescription != "Operations" || config.SiteIcon != "https://example.com/icon.png" {
		t.Fatalf("persisted site settings = %#v", config)
	}
	prefs := reloaded.GetUserPrefs(reloaded.AdminUserID())
	if prefs.TunnelID != "tunnel-id" || prefs.TunnelName != "Production" {
		t.Fatalf("persisted tunnel prefs = (%q, %q)", prefs.TunnelID, prefs.TunnelName)
	}
	if len(config.CNAMEPresets) != 1 || config.CNAMEPresets[0] != presets[0] {
		t.Fatalf("persisted presets = %#v", config.CNAMEPresets)
	}
}

func TestDatabaseFileHasPrivatePermissions(t *testing.T) {
	s, path := newStoreWithConfig(t, models.Config{AdminUsername: "admin", AdminPasswordHash: HashPassword("password")})
	dbPath := ResolveDBPath(path)
	if err := s.SetAdminUsername("new-admin"); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0600 {
		t.Fatalf("database permissions = %o, want 600", info.Mode().Perm())
	}
	matches, err := filepath.Glob(filepath.Join(filepath.Dir(dbPath), ".config-*"))
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 0 {
		t.Fatalf("temporary files remain: %v", matches)
	}
	reloaded := NewStore(path)
	username, _ := reloaded.GetAdminCredentials()
	if username != "new-admin" {
		t.Fatalf("reloaded username = %q", username)
	}
}

func TestSetAdminPasswordHashPreservesUsername(t *testing.T) {
	s := newTestStore(t, HashPassword("old-password"))
	if err := s.SetAdminUsername("concurrent-name"); err != nil {
		t.Fatal(err)
	}
	newHash := HashPassword("new-password")
	if err := s.SetAdminPasswordHash(newHash); err != nil {
		t.Fatal(err)
	}
	username, hash := s.GetAdminCredentials()
	if username != "concurrent-name" || hash != newHash {
		t.Fatalf("credentials = (%q, %q)", username, hash)
	}
}

func TestSetAdminCredentialsReturnsSaveFailure(t *testing.T) {
	s := newTestStore(t, HashPassword("password"))
	before := s.GetConfig()
	s.filePath = filepath.Join(t.TempDir(), "missing", "config.json")
	if err := s.SetAdminCredentials("changed", HashPassword("changed")); err == nil {
		t.Fatal("SetAdminCredentials succeeded with missing parent")
	}
	if !configsEqual(before, s.GetConfig()) {
		t.Fatal("SetAdminCredentials did not roll back after save failure")
	}
}

func TestChangingCloudflareAccountClearsTunnelSelection(t *testing.T) {
	s := newTestStore(t, HashPassword("password"))
	if err := s.SetCloudflareAccount("account-one", "One"); err != nil {
		t.Fatalf("SetCloudflareAccount() error = %v", err)
	}
	if err := s.SetUserTunnelSelection(s.AdminUserID(), "tunnel-one", "Tunnel One"); err != nil {
		t.Fatalf("SetTunnelSelection() error = %v", err)
	}
	if err := s.SetCloudflareAccount("account-two", "Two"); err != nil {
		t.Fatalf("SetCloudflareAccount() error = %v", err)
	}
	config := s.GetConfig()
	if config.TunnelID != "" || config.TunnelName != "" {
		t.Fatalf("tunnel selection survived account change: %#v", config)
	}
}

func sha256Hex(password string) string {
	digest := sha256.Sum256([]byte(password))
	return fmt.Sprintf("%x", digest)
}

func newTestStore(t *testing.T, passwordHash string) *Store {
	t.Helper()
	s, _ := newStoreWithConfig(t, models.Config{
		AdminUsername:     "admin",
		AdminPasswordHash: passwordHash,
	})
	return s
}

func newEnabledTOTPStore(t *testing.T, step int64, hashes []string) *Store {
	t.Helper()
	s, _ := newStoreWithConfig(t, models.Config{
		AdminUsername:          "admin",
		AdminPasswordHash:      HashPassword("password"),
		TOTPEnabled:            true,
		TOTPSecretEncrypted:    "v1:encrypted-secret",
		TOTPRecoveryCodeHashes: append([]string(nil), hashes...),
		TOTPLastAcceptedStep:   step,
	})
	return s
}

func newStoreWithConfig(t *testing.T, config models.Config) (*Store, string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.json")
	data, err := json.Marshal(config)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
	return NewStore(path), path
}

func TestMonitorsAndTargetsPersistAcrossReload(t *testing.T) {
	s, path := newTestStoreWithMonitors(t)
	monitors := s.GetConfig().Monitors
	if len(monitors) != 2 {
		t.Fatalf("monitor count = %d, want 2", len(monitors))
	}

	reloaded := NewStore(path).GetConfig().Monitors
	if len(reloaded) != 2 {
		t.Fatalf("reloaded monitor count = %d, want 2", len(reloaded))
	}
	for i := range monitors {
		if monitors[i].ID != reloaded[i].ID || monitors[i].Name != reloaded[i].Name {
			t.Fatalf("monitor %d = (%q, %q), want (%q, %q)", i, reloaded[i].ID, reloaded[i].Name, monitors[i].ID, monitors[i].Name)
		}
		if monitors[i].PublishEnabled != reloaded[i].PublishEnabled {
			t.Fatalf("monitor %s publish = %v, want %v", monitors[i].ID, reloaded[i].PublishEnabled, monitors[i].PublishEnabled)
		}
		if len(reloaded[i].Targets) != len(monitors[i].Targets) {
			t.Fatalf("monitor %s target count = %d, want %d", monitors[i].ID, len(reloaded[i].Targets), len(monitors[i].Targets))
		}
		for j, target := range monitors[i].Targets {
			if reloaded[i].Targets[j] != target {
				t.Fatalf("monitor %s target %d = %#v, want %#v", monitors[i].ID, j, reloaded[i].Targets[j], target)
			}
		}
	}

	// Mutating the reloaded store must not resurrect stale legacy state.
	if err := reloadedStore(t, path).RemoveMonitor(monitors[0].ID); err != nil {
		t.Fatal(err)
	}
	if got := NewStore(path).GetConfig().Monitors; len(got) != 1 || got[0].ID != monitors[1].ID {
		t.Fatalf("monitors after removal = %#v", got)
	}
}

func reloadedStore(t *testing.T, path string) *Store {
	t.Helper()
	return NewStore(path)
}

func newTestStoreWithMonitors(t *testing.T) (*Store, string) {
	t.Helper()
	s := newTestStore(t, HashPassword("password"))
	first := models.Monitor{
		ID:             "mon-one",
		Name:           "First",
		IntervalSec:    60,
		PublishEnabled: true,
		PublicToken:    "token-one",
		PublicSlug:     "first",
		Targets: []models.MonitorTarget{
			{ID: "t-one", Name: "A", URL: "https://a.example.com", Type: "http", CreatedAt: 11, LinkEnabled: true},
			{ID: "t-two", Name: "B", URL: "https://b.example.com", Type: "tcp"},
		},
	}
	second := models.Monitor{
		ID:          "mon-two",
		Name:        "Second",
		IntervalSec: 120,
		Targets: []models.MonitorTarget{
			{ID: "t-three", Name: "C", URL: "10.0.0.1:22", Type: "tcp", CreatedAt: 33},
		},
	}
	if err := s.AddMonitor(first); err != nil {
		t.Fatal(err)
	}
	if err := s.AddMonitor(second); err != nil {
		t.Fatal(err)
	}
	return s, s.filePath
}

func runConcurrently(count int, operation func() error) []error {
	start := make(chan struct{})
	errs := make([]error, count)
	var wg sync.WaitGroup
	wg.Add(count)
	for i := range errs {
		go func(index int) {
			defer wg.Done()
			<-start
			errs[index] = operation()
		}(i)
	}
	close(start)
	wg.Wait()
	return errs
}

func assertExactlyOneSuccess(t *testing.T, errs []error, wantFailure error) {
	t.Helper()
	successes := 0
	for _, err := range errs {
		if err == nil {
			successes++
			continue
		}
		if !errors.Is(err, wantFailure) {
			t.Errorf("operation error = %v, want %v", err, wantFailure)
		}
	}
	if successes != 1 {
		t.Fatalf("successful operations = %d, want exactly 1", successes)
	}
}

func assertTOTPDisabled(t *testing.T, s *Store) {
	t.Helper()
	enabled, secret, step, count := s.GetTOTPState(s.AdminUserID())
	if enabled || secret != "" || step != 0 || count != 0 {
		t.Fatalf("GetTOTPState() = (%v, %q, %d, %d), want disabled zero state", enabled, secret, step, count)
	}
	config := s.GetConfig()
	if config.TOTPRecoveryCodeHashes != nil {
		t.Fatalf("recovery hashes = %#v, want nil", config.TOTPRecoveryCodeHashes)
	}
}

func cloneConfig(t *testing.T, config models.Config) models.Config {
	t.Helper()
	data, err := json.Marshal(config)
	if err != nil {
		t.Fatal(err)
	}
	var clone models.Config
	if err := json.Unmarshal(data, &clone); err != nil {
		t.Fatal(err)
	}
	return clone
}

func configsEqual(left, right models.Config) bool {
	leftJSON, _ := json.Marshal(left)
	rightJSON, _ := json.Marshal(right)
	return string(leftJSON) == string(rightJSON)
}
