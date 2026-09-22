package store

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"log"
	"os"
	"strings"
	"time"

	"tunnel-manager/models"
)

var (
	ErrUsernameTaken = errors.New("username already taken")
	ErrEmailTaken    = errors.New("email already taken")
	ErrUserNotFound  = errors.New("user not found")
	ErrLastAdmin     = errors.New("cannot remove or disable the last administrator")
)

// newID returns a random 16-character hex identifier.
func newID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// seedUsers creates the built-in default group on first boot and migrates a
// legacy document's administrator credentials and TOTP state into the users
// table. A fresh install gets no administrator here: the setup wizard creates
// one from a password the operator typed, which is also why no password is
// ever generated and printed.
func (s *Store) seedUsers() {
	if len(s.users) > 0 {
		s.mu.Lock()
		defer s.mu.Unlock()
		s.resolveAdminIDLocked()
		if s.assignOrphanMonitorsToAdminLocked() {
			if err := s.saveLocked(); err != nil {
				log.Printf("assign orphaned monitors to administrator: %v", err)
			}
		}
		return
	}

	s.ensureDefaultGroupLocked()

	passwordHash := s.config.AdminPasswordHash
	if passwordHash == "" {
		if password := os.Getenv("ADMIN_PASSWORD"); password != "" {
			passwordHash = hashPassword(password)
		}
	}
	if passwordHash == "" {
		// Nothing to migrate and no ADMIN_PASSWORD: the instance stays
		// uninstalled until the setup wizard supplies the first account.
		if err := s.saveLocked(); err != nil {
			log.Printf("seed user tables: %v", err)
		}
		return
	}

	s.attachAdminLocked(s.config.AdminUsername, passwordHash)

	if err := s.saveLocked(); err != nil {
		log.Printf("seed user tables: %v", err)
	}
}

// CreateAdmin adds the first administrator account with a password the
// operator chose in the setup wizard. It refuses to touch a store that already
// has accounts, so an installed panel can never be taken over through it.
func (s *Store) CreateAdmin(username, password string) error {
	username = strings.TrimSpace(username)
	if username == "" {
		return errors.New("请填写管理员用户名")
	}
	if len(password) < 6 || len(password) > 1024 {
		return errors.New("密码长度需在 6-1024 位之间")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.users) > 0 {
		return errors.New("该数据库已有账户，无法重复初始化")
	}
	s.ensureDefaultGroupLocked()
	s.attachAdminLocked(username, hashPassword(password))
	return s.saveLocked()
}

// Installed reports whether any account exists, which is what separates a
// fresh install (the wizard runs) from a configured panel.
func (s *Store) Installed() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.users) > 0
}

// ensureDefaultGroupLocked creates the built-in group when the user tables are
// still empty.
func (s *Store) ensureDefaultGroupLocked() {
	if len(s.groups) > 0 {
		return
	}
	group := models.UserGroup{
		ID:          newID(),
		Name:        "默认用户组",
		Permissions: append([]string(nil), models.AllPermissions...),
		Builtin:     true,
		CreatedAt:   time.Now().Unix(),
	}
	s.groups = append(s.groups, group)
}

// attachAdminLocked appends an administrator built from the stored settings,
// migrating the legacy global OAuth connection and clearing the secrets the
// settings document used to carry. The caller holds the lock and saves.
func (s *Store) attachAdminLocked(username, passwordHash string) {
	now := time.Now().Unix()
	admin := models.User{
		ID:                     newID(),
		Username:               username,
		PasswordHash:           passwordHash,
		Role:                   models.RoleAdmin,
		Status:                 models.UserActive,
		EmailVerified:          true,
		TOTPEnabled:            s.config.TOTPEnabled,
		TOTPSecretEncrypted:    s.config.TOTPSecretEncrypted,
		TOTPLastAcceptedStep:   s.config.TOTPLastAcceptedStep,
		TOTPRecoveryCodeHashes: append([]string(nil), s.config.TOTPRecoveryCodeHashes...),
		CreatedAt:              now,
	}
	// Migrate the legacy global OAuth connection to the administrator.
	if s.config.CFOAuthAccessToken != "" {
		conn := models.CFConnection{
			ID:           newID(),
			UserID:       admin.ID,
			Label:        "默认连接",
			AccountID:    s.config.CFAccountID,
			AccountName:  s.config.CFAccountName,
			AccessToken:  s.config.CFOAuthAccessToken,
			RefreshToken: s.config.CFOAuthRefreshToken,
			ExpiresAt:    s.config.CFOAuthExpiresAt,
			Scope:        s.config.CFOAuthScope,
			CreatedAt:    now,
		}
		s.cfConns = append(s.cfConns, conn)
		admin.ActiveCFConnectionID = conn.ID
	}
	s.users = append(s.users, admin)
	s.adminID = admin.ID
	s.prefs[admin.ID] = models.UserPrefs{
		TunnelID:         s.config.TunnelID,
		TunnelName:       s.config.TunnelName,
		ServiceURL:       s.config.ServiceURL,
		SelectedZoneID:   s.config.SelectedZoneID,
		SelectedZoneName: s.config.SelectedZoneName,
	}
	s.assignOrphanMonitorsToAdminLocked()
	// Administrator secrets now live in the users table only.
	s.config.AdminPasswordHash = ""
	s.config.TOTPEnabled = false
	s.config.TOTPSecretEncrypted = ""
	s.config.TOTPLastAcceptedStep = 0
	s.config.TOTPRecoveryCodeHashes = nil
	s.config.CFOAuthAccessToken = ""
	s.config.CFOAuthRefreshToken = ""
	s.config.CFOAuthExpiresAt = 0
	s.config.CFOAuthScope = ""
	s.config.CFAccountID = ""
	s.config.CFAccountName = ""
}

func (s *Store) resolveAdminIDLocked() {
	if s.adminID != "" {
		return
	}
	for i := range s.users {
		if s.users[i].Role == models.RoleAdmin {
			s.adminID = s.users[i].ID
			return
		}
	}
}

// assignOrphanMonitorsToAdminLocked claims monitors created before ownership
// was persisted. The caller must hold s.mu once the store is published.
func (s *Store) assignOrphanMonitorsToAdminLocked() bool {
	if s.adminID == "" {
		return false
	}
	changed := false
	for i := range s.config.Monitors {
		if s.config.Monitors[i].UserID != "" {
			continue
		}
		s.config.Monitors[i].UserID = s.adminID
		changed = true
	}
	return changed
}

// AdminUserID returns the seeded administrator account id.
func (s *Store) AdminUserID() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	s.resolveAdminIDLocked()
	return s.adminID
}

// findUserLocked returns the user with the given id; caller holds s.mu.
func (s *Store) findUserLocked(id string) *models.User {
	for i := range s.users {
		if s.users[i].ID == id {
			return &s.users[i]
		}
	}
	return nil
}

// adminUserLocked returns the administrator account; caller holds s.mu.
func (s *Store) adminUserLocked() *models.User {
	s.resolveAdminIDLocked()
	if s.adminID == "" {
		return nil
	}
	return s.findUserLocked(s.adminID)
}

// usernameFreeLocked reports whether a username is unused apart from excludeID.
func (s *Store) usernameFreeLocked(username, excludeID string) error {
	for i := range s.users {
		if s.users[i].ID != excludeID && strings.EqualFold(s.users[i].Username, username) {
			return ErrUsernameTaken
		}
	}
	return nil
}

// emailFreeLocked reports whether an email is unused apart from excludeID.
func (s *Store) emailFreeLocked(email, excludeID string) error {
	if email == "" {
		return nil
	}
	for i := range s.users {
		if s.users[i].ID != excludeID && strings.EqualFold(s.users[i].Email, email) {
			return ErrEmailTaken
		}
	}
	return nil
}

// GetAdminCredentials returns the administrator username and password hash.
func (s *Store) GetAdminCredentials() (string, string) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	admin := s.adminUserLocked()
	if admin == nil {
		return "", ""
	}
	return admin.Username, admin.PasswordHash
}

// SetAdminCredentials sets the administrator username and password hash.
func (s *Store) SetAdminCredentials(username, passwordHash string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	admin := s.adminUserLocked()
	if admin == nil {
		return ErrUserNotFound
	}
	if err := s.usernameFreeLocked(username, admin.ID); err != nil {
		return err
	}
	prevName, prevHash := admin.Username, admin.PasswordHash
	admin.Username, admin.PasswordHash = username, passwordHash
	if err := s.saveLocked(); err != nil {
		admin.Username, admin.PasswordHash = prevName, prevHash
		return err
	}
	return nil
}

// SetAdminPasswordHash changes the administrator password without touching a
// concurrent username update.
func (s *Store) SetAdminPasswordHash(passwordHash string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	admin := s.adminUserLocked()
	if admin == nil {
		return ErrUserNotFound
	}
	previous := admin.PasswordHash
	admin.PasswordHash = passwordHash
	if err := s.saveLocked(); err != nil {
		admin.PasswordHash = previous
		return err
	}
	return nil
}

// SetAdminUsername changes only the administrator username. It intentionally
// leaves the current password hash untouched, including a migrated Argon2id hash.
func (s *Store) SetAdminUsername(username string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	admin := s.adminUserLocked()
	if admin == nil {
		return ErrUserNotFound
	}
	if err := s.usernameFreeLocked(username, admin.ID); err != nil {
		return err
	}
	previous := admin.Username
	admin.Username = username
	if err := s.saveLocked(); err != nil {
		admin.Username = previous
		return err
	}
	return nil
}

// ValidatePassword checks a plaintext password against the administrator's
// stored hash. Successful validation of a legacy SHA-256 digest upgrades the
// stored hash to Argon2id.
func (s *Store) ValidatePassword(password, encodedHash string) bool {
	return s.ValidateUserPassword(s.AdminUserID(), password, encodedHash)
}

// ValidateUserPassword checks a plaintext password against one account's
// stored hash, upgrading legacy SHA-256 digests to Argon2id on success.
func (s *Store) ValidateUserPassword(userID, password, encodedHash string) bool {
	valid, legacy := verifyPassword(password, encodedHash)
	if !valid || !legacy {
		return valid
	}

	// Do not overwrite a password that changed between credential lookup and
	// validation.
	s.mu.Lock()
	defer s.mu.Unlock()
	user := s.findUserLocked(userID)
	if user == nil {
		return false
	}
	if subtle.ConstantTimeCompare([]byte(user.PasswordHash), []byte(encodedHash)) == 1 {
		previous := user.PasswordHash
		user.PasswordHash = HashPassword(password)
		if err := s.saveLocked(); err != nil {
			user.PasswordHash = previous
			log.Printf("save migrated password hash for %s: %v", userID, err)
		}
	}
	return true
}

// ---------------------------------------------------------------------------
// User management

// CreateUser stores a new account. Username uniqueness is enforced case
// folder-insensitively; email uniqueness when a non-empty email is given.
func (s *Store) CreateUser(user models.User) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if user.ID == "" {
		user.ID = newID()
	}
	if user.CreatedAt == 0 {
		user.CreatedAt = time.Now().Unix()
	}
	user.Username = strings.TrimSpace(user.Username)
	user.Email = strings.TrimSpace(strings.ToLower(user.Email))
	if user.Status == "" {
		user.Status = models.UserActive
	}
	if user.Role == "" {
		user.Role = models.RoleUser
	}
	if err := s.usernameFreeLocked(user.Username, ""); err != nil {
		return err
	}
	if err := s.emailFreeLocked(user.Email, ""); err != nil {
		return err
	}
	s.users = append(s.users, user)
	if err := s.saveLocked(); err != nil {
		s.users = s.users[:len(s.users)-1]
		return err
	}
	return nil
}

// GetUserByID returns a copy of one account.
func (s *Store) GetUserByID(id string) (models.User, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	user := s.findUserLocked(id)
	if user == nil {
		return models.User{}, false
	}
	return copyUser(*user), true
}

// GetUserByUsername returns a copy of one account by login name.
func (s *Store) GetUserByUsername(username string) (models.User, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for i := range s.users {
		if strings.EqualFold(s.users[i].Username, username) {
			return copyUser(s.users[i]), true
		}
	}
	return models.User{}, false
}

// GetUserByEmail returns a copy of one account by email address.
func (s *Store) GetUserByEmail(email string) (models.User, bool) {
	email = strings.TrimSpace(strings.ToLower(email))
	s.mu.RLock()
	defer s.mu.RUnlock()
	for i := range s.users {
		if s.users[i].Email != "" && strings.EqualFold(s.users[i].Email, email) {
			return copyUser(s.users[i]), true
		}
	}
	return models.User{}, false
}

func copyUser(user models.User) models.User {
	user.TOTPRecoveryCodeHashes = append([]string(nil), user.TOTPRecoveryCodeHashes...)
	return user
}

// ListUsers returns the API-safe view of every account.
func (s *Store) ListUsers() []models.UserView {
	s.mu.RLock()
	defer s.mu.RUnlock()
	// Bound credentials live outside the cache, so the counts come from one
	// grouped query instead of one query per account.
	counts, err := s.passkeyCounts()
	if err != nil {
		log.Printf("count passkeys per account: %v", err)
	}
	views := make([]models.UserView, 0, len(s.users))
	for i := range s.users {
		view := s.userViewLocked(&s.users[i])
		view.Passkeys = counts[s.users[i].ID]
		views = append(views, view)
	}
	return views
}

func (s *Store) userViewLocked(user *models.User) models.UserView {
	view := models.UserView{
		ID:            user.ID,
		Username:      user.Username,
		Nickname:      user.Nickname,
		Avatar:        user.Avatar,
		Email:         user.Email,
		Role:          user.Role,
		GroupID:       user.GroupID,
		Status:        user.Status,
		EmailVerified: user.EmailVerified,
		TOTPEnabled:   user.TOTPEnabled,
		CreatedAt:     user.CreatedAt,
		LastLoginAt:   user.LastLoginAt,
		Permissions:   append([]string(nil), models.AllPermissions...),
		// Passkeys is filled in by ListUsers, which batches the counts.
		PasswordLoginDisabled: user.PasswordLoginDisabled,
	}
	if group := s.findGroupLocked(user.GroupID); group != nil {
		view.GroupName = group.Name
		view.Permissions = append([]string(nil), group.Permissions...)
	}
	return view
}

// SetUserStatus activates or disables one account.
func (s *Store) SetUserStatus(id, status string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	user := s.findUserLocked(id)
	if user == nil {
		return ErrUserNotFound
	}
	if user.Role == models.RoleAdmin && status != models.UserActive {
		admins := 0
		for i := range s.users {
			if s.users[i].Role == models.RoleAdmin && s.users[i].Status == models.UserActive {
				admins++
			}
		}
		if admins <= 1 {
			return ErrLastAdmin
		}
	}
	previous := user.Status
	user.Status = status
	if err := s.saveLocked(); err != nil {
		user.Status = previous
		return err
	}
	return nil
}

// SetUserPasswordHash replaces one account's password hash.
func (s *Store) SetUserPasswordHash(id, passwordHash string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	user := s.findUserLocked(id)
	if user == nil {
		return ErrUserNotFound
	}
	previous := user.PasswordHash
	user.PasswordHash = passwordHash
	if err := s.saveLocked(); err != nil {
		user.PasswordHash = previous
		return err
	}
	return nil
}

// SetUserEmailVerified records whether an address has been confirmed.
func (s *Store) SetUserEmailVerified(id string, verified bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	user := s.findUserLocked(id)
	if user == nil {
		return ErrUserNotFound
	}
	previous := user.EmailVerified
	user.EmailVerified = verified
	if err := s.saveLocked(); err != nil {
		user.EmailVerified = previous
		return err
	}
	return nil
}

// SetUsername changes one account's login name with uniqueness enforcement.
func (s *Store) SetUsername(id, username string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	user := s.findUserLocked(id)
	if user == nil {
		return ErrUserNotFound
	}
	if err := s.usernameFreeLocked(username, id); err != nil {
		return err
	}
	previous := user.Username
	user.Username = username
	if err := s.saveLocked(); err != nil {
		user.Username = previous
		return err
	}
	return nil
}

// UpdateUserLogin stamps the last successful login time.
func (s *Store) UpdateUserLogin(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	user := s.findUserLocked(id)
	if user == nil {
		return
	}
	user.LastLoginAt = time.Now().Unix()
	if err := s.saveLocked(); err != nil {
		user.LastLoginAt = 0
	}
}

// DeleteUser removes one account and its sessions and preferences.
func (s *Store) DeleteUser(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	idx := -1
	for i := range s.users {
		if s.users[i].ID == id {
			idx = i
			break
		}
	}
	if idx < 0 {
		return ErrUserNotFound
	}
	if s.users[idx].Role == models.RoleAdmin {
		admins := 0
		for i := range s.users {
			if s.users[i].Role == models.RoleAdmin {
				admins++
			}
		}
		if admins <= 1 {
			return ErrLastAdmin
		}
	}
	previousUsers := append([]models.User(nil), s.users...)
	previousSessions := append([]sessionRecord(nil), s.sessions...)
	previousPrefs := len(s.prefs)
	s.users = append(s.users[:idx], s.users[idx+1:]...)
	sessions := s.sessions[:0]
	for _, sess := range s.sessions {
		if sess.UserID != id {
			sessions = append(sessions, sess)
		}
	}
	s.sessions = sessions
	delete(s.prefs, id)
	if s.adminID == id {
		s.adminID = ""
	}
	if err := s.saveLocked(); err != nil {
		s.users = previousUsers
		s.sessions = previousSessions
		if previousPrefs > len(s.prefs) {
			s.prefs[id] = models.UserPrefs{}
		}
		return err
	}
	// Bound passkeys and learned networks are stored outside the cached
	// configuration, so they are removed explicitly once the account is really
	// gone.
	if err := s.deletePasskeysForUser(id); err != nil {
		log.Printf("delete passkeys of removed account %s: %v", id, err)
	}
	if err := s.deleteFamiliarIPsForUser(id); err != nil {
		log.Printf("delete familiar networks of removed account %s: %v", id, err)
	}
	return nil
}

// ---------------------------------------------------------------------------
// Per-user TOTP (replaces the former single-administrator state)

// GetTOTPState returns one account's persisted TOTP state.
func (s *Store) GetTOTPState(userID string) (enabled bool, encryptedSecret string, lastStep int64, recoveryCount int) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	user := s.findUserLocked(userID)
	if user == nil {
		return false, "", 0, 0
	}
	return user.TOTPEnabled, user.TOTPSecretEncrypted, user.TOTPLastAcceptedStep, len(user.TOTPRecoveryCodeHashes)
}

// EnableTOTP atomically persists a confirmed TOTP setup. The accepted setup
// step is recorded so the confirmation code cannot be replayed.
func (s *Store) EnableTOTP(userID, encryptedSecret string, recoveryHashes []string, acceptedStep int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	user := s.findUserLocked(userID)
	if user == nil {
		return ErrUserNotFound
	}
	if user.TOTPEnabled {
		return ErrTOTPAlreadyEnabled
	}
	previousEnabled, previousSecret, previousStep, previousHashes := user.TOTPEnabled, user.TOTPSecretEncrypted, user.TOTPLastAcceptedStep, user.TOTPRecoveryCodeHashes
	user.TOTPEnabled = true
	user.TOTPSecretEncrypted = encryptedSecret
	user.TOTPRecoveryCodeHashes = append([]string(nil), recoveryHashes...)
	user.TOTPLastAcceptedStep = acceptedStep
	if err := s.saveLocked(); err != nil {
		user.TOTPEnabled, user.TOTPSecretEncrypted, user.TOTPLastAcceptedStep, user.TOTPRecoveryCodeHashes = previousEnabled, previousSecret, previousStep, previousHashes
		return err
	}
	return nil
}

// AdvanceTOTPStep records a newer accepted TOTP step and rejects replayed or
// older codes.
func (s *Store) AdvanceTOTPStep(userID string, step int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	user := s.findUserLocked(userID)
	if user == nil || !user.TOTPEnabled {
		return ErrTOTPDisabled
	}
	if step <= user.TOTPLastAcceptedStep {
		return ErrTOTPReplay
	}
	previous := user.TOTPLastAcceptedStep
	user.TOTPLastAcceptedStep = step
	if err := s.saveLocked(); err != nil {
		user.TOTPLastAcceptedStep = previous
		return err
	}
	return nil
}

// ConsumeRecoveryCode removes one matching candidate hash. Comparison is
// constant-time and the candidate must already be hashed by the handler.
func (s *Store) ConsumeRecoveryCode(userID, candidateHash string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	user := s.findUserLocked(userID)
	if user == nil || !user.TOTPEnabled {
		return ErrTOTPDisabled
	}
	match := -1
	for i, storedHash := range user.TOTPRecoveryCodeHashes {
		if subtle.ConstantTimeCompare([]byte(storedHash), []byte(candidateHash)) == 1 {
			match = i
		}
	}
	if match < 0 {
		return ErrRecoveryCodeNotFound
	}
	previous := append([]string(nil), user.TOTPRecoveryCodeHashes...)
	remaining := make([]string, 0, len(previous)-1)
	remaining = append(remaining, previous[:match]...)
	remaining = append(remaining, previous[match+1:]...)
	user.TOTPRecoveryCodeHashes = remaining
	if err := s.saveLocked(); err != nil {
		user.TOTPRecoveryCodeHashes = previous
		return err
	}
	return nil
}

// DisableTOTPWithStep disables TOTP when presented with a newer valid step.
func (s *Store) DisableTOTPWithStep(userID string, step int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	user := s.findUserLocked(userID)
	if user == nil || !user.TOTPEnabled {
		return ErrTOTPDisabled
	}
	if step <= user.TOTPLastAcceptedStep {
		return ErrTOTPReplay
	}
	return s.disableTOTPLocked(user)
}

// DisableTOTPWithRecovery disables TOTP using one stored recovery hash.
func (s *Store) DisableTOTPWithRecovery(userID, candidateHash string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	user := s.findUserLocked(userID)
	if user == nil || !user.TOTPEnabled {
		return ErrTOTPDisabled
	}
	matched := 0
	for _, storedHash := range user.TOTPRecoveryCodeHashes {
		matched |= subtle.ConstantTimeCompare([]byte(storedHash), []byte(candidateHash))
	}
	if matched != 1 {
		return ErrRecoveryCodeNotFound
	}
	return s.disableTOTPLocked(user)
}

func (s *Store) disableTOTPLocked(user *models.User) error {
	previousEnabled, previousSecret, previousStep, previousHashes := user.TOTPEnabled, user.TOTPSecretEncrypted, user.TOTPLastAcceptedStep, user.TOTPRecoveryCodeHashes
	user.TOTPEnabled = false
	user.TOTPSecretEncrypted = ""
	user.TOTPRecoveryCodeHashes = nil
	user.TOTPLastAcceptedStep = 0
	if err := s.saveLocked(); err != nil {
		user.TOTPEnabled, user.TOTPSecretEncrypted, user.TOTPLastAcceptedStep, user.TOTPRecoveryCodeHashes = previousEnabled, previousSecret, previousStep, previousHashes
		return err
	}
	return nil
}

// ---------------------------------------------------------------------------
// Sessions (database-backed, survive restarts)

// CreateSession persists one session token hash with its expiry.
func (s *Store) CreateSession(tokenHash, userID string, expiresAt int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.findUserLocked(userID) == nil {
		return ErrUserNotFound
	}
	s.pruneSessionsLocked()
	s.sessions = append(s.sessions, sessionRecord{
		TokenHash: tokenHash,
		UserID:    userID,
		CreatedAt: time.Now().Unix(),
		ExpiresAt: expiresAt,
	})
	if err := s.saveLocked(); err != nil {
		s.sessions = s.sessions[:len(s.sessions)-1]
		return err
	}
	return nil
}

// GetSessionUser resolves a token hash to the authenticated identity while
// the session is unexpired and the account active. now is supplied by the
// caller so tests can drive the clock.
func (s *Store) GetSessionUser(tokenHash string, now int64) (models.SessionUser, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for i := range s.sessions {
		sess := s.sessions[i]
		if sess.TokenHash != tokenHash {
			continue
		}
		if sess.ExpiresAt > 0 && now >= sess.ExpiresAt {
			return models.SessionUser{}, false
		}
		user := s.findUserLocked(sess.UserID)
		if user == nil || user.Status != models.UserActive {
			return models.SessionUser{}, false
		}
		return s.sessionUserLocked(user), true
	}
	return models.SessionUser{}, false
}

func (s *Store) sessionUserLocked(user *models.User) models.SessionUser {
	su := models.SessionUser{
		ID:          user.ID,
		Username:    user.Username,
		Nickname:    user.Nickname,
		Avatar:      user.Avatar,
		Email:       user.Email,
		Role:        user.Role,
		Permissions: []string{},
	}
	if !user.EmailVerified {
		su.Email = ""
	}
	if user.Role == models.RoleAdmin {
		su.Permissions = append(su.Permissions, models.AllPermissions...)
		return su
	}
	if group := s.findGroupLocked(user.GroupID); group != nil {
		su.Permissions = append(su.Permissions, group.Permissions...)
	}
	return su
}

// DeleteSession removes one session (logout).
func (s *Store) DeleteSession(tokenHash string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.sessions {
		if s.sessions[i].TokenHash == tokenHash {
			previous := append([]sessionRecord(nil), s.sessions...)
			s.sessions = append(s.sessions[:i], s.sessions[i+1:]...)
			if err := s.saveLocked(); err != nil {
				s.sessions = previous
				return err
			}
			return nil
		}
	}
	return nil
}

// DeleteUserSessions revokes every session of one account (password change).
func (s *Store) DeleteUserSessions(userID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	remaining := make([]sessionRecord, 0, len(s.sessions))
	removed := false
	for _, sess := range s.sessions {
		if sess.UserID == userID {
			removed = true
			continue
		}
		remaining = append(remaining, sess)
	}
	if !removed {
		return nil
	}
	previous := append([]sessionRecord(nil), s.sessions...)
	s.sessions = remaining
	if err := s.saveLocked(); err != nil {
		s.sessions = previous
		return err
	}
	return nil
}

func (s *Store) pruneSessionsLocked() {
	if len(s.sessions) == 0 {
		return
	}
	now := time.Now().Unix()
	remaining := s.sessions[:0]
	for _, sess := range s.sessions {
		if sess.ExpiresAt <= 0 || now < sess.ExpiresAt {
			remaining = append(remaining, sess)
		}
	}
	s.sessions = remaining
}

// ---------------------------------------------------------------------------
// User groups

// ListGroups returns every user group.
func (s *Store) findGroupLocked(id string) *models.UserGroup {
	for i := range s.groups {
		if s.groups[i].ID == id {
			return &s.groups[i]
		}
	}
	return nil
}

// ---------------------------------------------------------------------------
// Email verification codes

// LastCodeSentWithin reports whether a code for the email was created less
// than d ago, throttling resend requests.
func (s *Store) LastCodeSentWithin(email, purpose string, d time.Duration) bool {
	email = strings.ToLower(strings.TrimSpace(email))
	s.mu.RLock()
	defer s.mu.RUnlock()
	for i := range s.verifyCodes {
		if s.verifyCodes[i].Email == email && s.verifyCodes[i].Purpose == purpose {
			return time.Since(time.Unix(s.verifyCodes[i].CreatedAt, 0)) < d
		}
	}
	return false
}

// PutVerifyCode stores a hashed code for the email, replacing prior codes.
func (s *Store) PutVerifyCode(email, purpose, codeHash string, ttl time.Duration) {
	email = strings.ToLower(strings.TrimSpace(email))
	now := time.Now().Unix()
	s.mu.Lock()
	defer s.mu.Unlock()
	remaining := s.verifyCodes[:0]
	for _, rec := range s.verifyCodes {
		if rec.Email != email || rec.Purpose != purpose {
			remaining = append(remaining, rec)
		}
	}
	s.verifyCodes = append(remaining, verifyCodeRecord{
		Email:     email,
		Purpose:   purpose,
		CodeHash:  codeHash,
		CreatedAt: now,
		ExpiresAt: now + int64(ttl.Seconds()),
	})
	if err := s.saveLocked(); err != nil {
		log.Printf("save verification code: %v", err)
	}
}

// ConsumeVerifyCode removes a matching unexpired code and reports whether the
// hash matched.
func (s *Store) ConsumeVerifyCode(email, purpose, codeHash string) bool {
	email = strings.ToLower(strings.TrimSpace(email))
	now := time.Now().Unix()
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.verifyCodes {
		rec := s.verifyCodes[i]
		if rec.Email != email || rec.Purpose != purpose {
			continue
		}
		// Expired codes are dropped either way.
		s.verifyCodes = append(s.verifyCodes[:i], s.verifyCodes[i+1:]...)
		if rec.ExpiresAt > 0 && now >= rec.ExpiresAt {
			_ = s.saveLocked()
			return false
		}
		if rec.CodeHash == codeHash {
			_ = s.saveLocked()
			return true
		}
		_ = s.saveLocked()
		return false
	}
	return false
}

// ---------------------------------------------------------------------------
// Application settings and SMTP relay

// GetAppSettings returns the panel settings.
func (s *Store) GetAppSettings() models.AppSettings {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.appSettings
}

// SetAppSettings persists the panel settings.
func (s *Store) SetAppSettings(settings models.AppSettings) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	previous := s.appSettings
	s.appSettings = settings
	if err := s.saveLocked(); err != nil {
		s.appSettings = previous
		return err
	}
	return nil
}

// GetSMTPSettings returns the relay configuration (password encrypted).
func (s *Store) GetSMTPSettings() models.SMTPSettings {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.smtp
}

// SetSMTPSettings persists the relay configuration.
func (s *Store) SetSMTPSettings(settings models.SMTPSettings) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	previous := s.smtp
	s.smtp = settings
	if err := s.saveLocked(); err != nil {
		s.smtp = previous
		return err
	}
	return nil
}

// ---------------------------------------------------------------------------
// Per-user preferences

// GetUserPrefs returns one account's selections.
func (s *Store) GetUserPrefs(userID string) models.UserPrefs {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.prefs[userID]
}

// SetUserTunnelSelection stores one account's active tunnel.
func (s *Store) SetUserTunnelSelection(userID, id, name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.findUserLocked(userID) == nil {
		return ErrUserNotFound
	}
	prefs := s.prefs[userID]
	previous := prefs
	prefs.TunnelID, prefs.TunnelName = id, name
	s.prefs[userID] = prefs
	if err := s.saveLocked(); err != nil {
		s.prefs[userID] = previous
		return err
	}
	return nil
}

// SetUserServiceURL stores one account's forwarding service URL.
func (s *Store) SetUserServiceURL(userID, serviceURL string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.findUserLocked(userID) == nil {
		return ErrUserNotFound
	}
	prefs := s.prefs[userID]
	previous := prefs
	prefs.ServiceURL = serviceURL
	s.prefs[userID] = prefs
	if err := s.saveLocked(); err != nil {
		s.prefs[userID] = previous
		return err
	}
	return nil
}

// GetUserNotifySettings returns the API-safe projection of one account's
// notification preferences.
func (s *Store) GetUserNotifySettings(userID string) (models.NotifySettingsView, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.findUserLocked(userID) == nil {
		return models.NotifySettingsView{}, false
	}
	prefs := s.prefs[userID]
	events := map[string]bool{}
	for _, event := range models.AllNotifyEvents {
		// First visit: default every event to on; an explicit save always
		// records the user's choice.
		events[event] = prefs.NotifyEvents == nil || prefs.NotifyEvents[event]
	}
	return models.NotifySettingsView{
		Channels:       append([]string(nil), prefs.NotifyChannels...),
		Events:         events,
		Emails:         prefs.NotifyEmails,
		TGBotTokenSet:  prefs.TGBotTokenEncrypted != "",
		TGNotifyChatID: prefs.TGNotifyChatID,
		TGRemoteBotSet: prefs.TGRemoteTokenEncrypted != "",
	}, true
}

// SetUserNotifySettings stores one account's notification preferences.
// tgBotTokenEncrypted must already be encrypted by the caller; pass the
// previously stored value to keep an existing token unchanged.
func (s *Store) SetUserNotifySettings(userID string, channels []string, events map[string]bool, emails, tgBotTokenEncrypted, tgNotifyChatID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.findUserLocked(userID) == nil {
		return ErrUserNotFound
	}
	prefs := s.prefs[userID]
	previous := prefs
	prefs.NotifyChannels = append([]string(nil), channels...)
	prefs.NotifyEvents = events
	prefs.NotifyEmails = emails
	prefs.TGBotTokenEncrypted = tgBotTokenEncrypted
	prefs.TGNotifyChatID = tgNotifyChatID
	s.prefs[userID] = prefs
	if err := s.saveLocked(); err != nil {
		s.prefs[userID] = previous
		return err
	}
	return nil
}

// ClearTunnelSelectionIfUsed drops the selection of every user pointing at
// the deleted tunnel.
func (s *Store) ClearTunnelSelectionIfUsed(tunnelID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	changed := false
	for uid := range s.prefs {
		if s.prefs[uid].TunnelID == tunnelID {
			prefs := s.prefs[uid]
			prefs.TunnelID, prefs.TunnelName = "", ""
			s.prefs[uid] = prefs
			changed = true
		}
	}
	if !changed {
		return nil
	}
	return s.saveLocked()
}

// SetUserEmail binds or replaces one account's email address. Uniqueness is
// enforced case-insensitively; pass verified=false to mark it pending.
func (s *Store) SetUserEmail(id, email string, verified bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	user := s.findUserLocked(id)
	if user == nil {
		return ErrUserNotFound
	}
	email = strings.TrimSpace(strings.ToLower(email))
	if err := s.emailFreeLocked(email, id); err != nil {
		return err
	}
	prevEmail, prevVerified := user.Email, user.EmailVerified
	user.Email = email
	user.EmailVerified = verified
	if err := s.saveLocked(); err != nil {
		user.Email, user.EmailVerified = prevEmail, prevVerified
		return err
	}
	return nil
}

// SetUserProfile updates one account's display nickname and avatar URL.
// Empty values clear the corresponding field.
func (s *Store) SetUserProfile(id, nickname, avatar string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	user := s.findUserLocked(id)
	if user == nil {
		return ErrUserNotFound
	}
	prevNickname, prevAvatar := user.Nickname, user.Avatar
	user.Nickname = strings.TrimSpace(nickname)
	user.Avatar = strings.TrimSpace(avatar)
	if err := s.saveLocked(); err != nil {
		user.Nickname, user.Avatar = prevNickname, prevAvatar
		return err
	}
	return nil
}
