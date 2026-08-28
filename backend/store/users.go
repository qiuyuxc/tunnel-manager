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
	ErrBuiltinGroup  = errors.New("built-in group cannot be deleted")
	ErrGroupInUse    = errors.New("group still has members")
	ErrGroupNotFound = errors.New("group not found")
	ErrInviteInvalid = errors.New("invite code is invalid, expired or exhausted")
)

// newID returns a random 16-character hex identifier.
func newID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// seedUsers creates the built-in default group and the administrator account
// on first boot, migrating the legacy document credentials and TOTP state
// into the users table.
func (s *Store) seedUsers() {
	if len(s.users) > 0 {
		s.mu.Lock()
		defer s.mu.Unlock()
		s.resolveAdminIDLocked()
		return
	}

	now := time.Now().Unix()
	s.groups = append(s.groups, models.UserGroup{
		ID:          newID(),
		Name:        "默认用户组",
		Permissions: append([]string(nil), models.AllPermissions...),
		Builtin:     true,
		CreatedAt:   now,
	})
	if s.appSettings.DefaultGroupID == "" {
		s.appSettings.DefaultGroupID = s.groups[0].ID
	}
	// First boot: registration opens by default; administrators can close it
	// in the backend at any time.
	s.appSettings.RegistrationEnabled = true

	passwordHash := s.config.AdminPasswordHash
	if passwordHash == "" {
		password := os.Getenv("ADMIN_PASSWORD")
		if password == "" {
			password = "admin123"
		}
		passwordHash = hashPassword(password)
		log.Printf("========================================")
		log.Printf("  密码为空，已使用默认密码（请登录后立即修改）：")
		log.Printf("  用户名: %s", s.config.AdminUsername)
		log.Printf("  密  码: %s", password)
		log.Printf("========================================")
	}

	admin := models.User{
		ID:                     newID(),
		Username:               s.config.AdminUsername,
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
	for i := range s.config.Monitors {
		if s.config.Monitors[i].UserID == "" {
			s.config.Monitors[i].UserID = admin.ID
		}
	}
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

	if err := s.saveLocked(); err != nil {
		log.Printf("seed user tables: %v", err)
	}
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
	views := make([]models.UserView, 0, len(s.users))
	for i := range s.users {
		views = append(views, s.userViewLocked(&s.users[i]))
	}
	return views
}

func (s *Store) userViewLocked(user *models.User) models.UserView {
	view := models.UserView{
		ID:            user.ID,
		Username:      user.Username,
		Email:         user.Email,
		Role:          user.Role,
		GroupID:       user.GroupID,
		Status:        user.Status,
		EmailVerified: user.EmailVerified,
		TOTPEnabled:   user.TOTPEnabled,
		CreatedAt:     user.CreatedAt,
		LastLoginAt:   user.LastLoginAt,
		Permissions:   append([]string(nil), models.AllPermissions...),
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

// SetUserGroup reassigns an account's group.
func (s *Store) SetUserGroup(id, groupID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	user := s.findUserLocked(id)
	if user == nil {
		return ErrUserNotFound
	}
	if groupID != "" && s.findGroupLocked(groupID) == nil {
		return ErrGroupNotFound
	}
	previous := user.GroupID
	user.GroupID = groupID
	if err := s.saveLocked(); err != nil {
		user.GroupID = previous
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
func (s *Store) ListGroups() []models.UserGroup {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]models.UserGroup, len(s.groups))
	for i := range s.groups {
		out[i] = copyGroup(s.groups[i])
	}
	return out
}

func copyGroup(group models.UserGroup) models.UserGroup {
	group.Permissions = append([]string(nil), group.Permissions...)
	return group
}

func (s *Store) findGroupLocked(id string) *models.UserGroup {
	for i := range s.groups {
		if s.groups[i].ID == id {
			return &s.groups[i]
		}
	}
	return nil
}

// CreateGroup stores a new group with a validated permission set.
func (s *Store) CreateGroup(name string, permissions []string) (models.UserGroup, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	name = strings.TrimSpace(name)
	if name == "" {
		return models.UserGroup{}, ErrGroupNotFound
	}
	for i := range s.groups {
		if s.groups[i].Name == name {
			return models.UserGroup{}, ErrUsernameTaken
		}
	}
	group := models.UserGroup{
		ID:          newID(),
		Name:        name,
		Permissions: sanitizePermissions(permissions),
		CreatedAt:   time.Now().Unix(),
	}
	s.groups = append(s.groups, group)
	if err := s.saveLocked(); err != nil {
		s.groups = s.groups[:len(s.groups)-1]
		return models.UserGroup{}, err
	}
	return copyGroup(group), nil
}

// UpdateGroup renames a group or replaces its permission set. Built-in groups
// keep their name.
func (s *Store) UpdateGroup(id, name string, permissions []string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	group := s.findGroupLocked(id)
	if group == nil {
		return ErrGroupNotFound
	}
	previousName, previousPerms := group.Name, group.Permissions
	if !group.Builtin {
		name = strings.TrimSpace(name)
		if name != "" && name != group.Name {
			for i := range s.groups {
				if s.groups[i].ID != id && s.groups[i].Name == name {
					return ErrUsernameTaken
				}
			}
		}
		if name != "" {
			group.Name = name
		}
	}
	group.Permissions = sanitizePermissions(permissions)
	if err := s.saveLocked(); err != nil {
		group.Name, group.Permissions = previousName, previousPerms
		return err
	}
	return nil
}

// DeleteGroup removes a group that no account references.
func (s *Store) DeleteGroup(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	group := s.findGroupLocked(id)
	if group == nil {
		return ErrGroupNotFound
	}
	if group.Builtin {
		return ErrBuiltinGroup
	}
	for i := range s.users {
		if s.users[i].GroupID == id {
			return ErrGroupInUse
		}
	}
	idx := -1
	for i := range s.groups {
		if s.groups[i].ID == id {
			idx = i
			break
		}
	}
	previous := append([]models.UserGroup(nil), s.groups...)
	s.groups = append(s.groups[:idx], s.groups[idx+1:]...)
	changed := false
	if s.appSettings.DefaultGroupID == id {
		s.appSettings.DefaultGroupID = ""
		changed = true
	}
	if err := s.saveLocked(); err != nil {
		s.groups = previous
		return err
	}
	_ = changed
	return nil
}

// sanitizePermissions filters unknown keys and de-duplicates.
func sanitizePermissions(permissions []string) []string {
	allowed := map[string]bool{}
	for _, p := range models.AllPermissions {
		allowed[p] = true
	}
	out := make([]string, 0, len(permissions))
	seen := map[string]bool{}
	for _, p := range permissions {
		if allowed[p] && !seen[p] {
			seen[p] = true
			out = append(out, p)
		}
	}
	return out
}

// ---------------------------------------------------------------------------
// Invite codes

// ListInvites returns every invite code.
func (s *Store) ListInvites() []models.Invite {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]models.Invite, len(s.invites))
	copy(out, s.invites)
	return out
}

// CreateInvite stores a new invite code, generating one when absent.
func (s *Store) CreateInvite(invite models.Invite) (models.Invite, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if invite.Code == "" {
		invite.Code = newID() + newID()[:8]
	}
	invite.Code = strings.TrimSpace(invite.Code)
	for i := range s.invites {
		if s.invites[i].Code == invite.Code {
			return models.Invite{}, ErrInviteInvalid
		}
	}
	if invite.GroupID != "" && s.findGroupLocked(invite.GroupID) == nil {
		return models.Invite{}, ErrGroupNotFound
	}
	invite.CreatedAt = time.Now().Unix()
	invite.Enabled = true
	s.invites = append(s.invites, invite)
	if err := s.saveLocked(); err != nil {
		s.invites = s.invites[:len(s.invites)-1]
		return models.Invite{}, err
	}
	return invite, nil
}

// UpdateInvite enables or disables one code.
func (s *Store) UpdateInvite(code string, enabled bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.invites {
		if s.invites[i].Code == code {
			previous := s.invites[i].Enabled
			s.invites[i].Enabled = enabled
			if err := s.saveLocked(); err != nil {
				s.invites[i].Enabled = previous
				return err
			}
			return nil
		}
	}
	return ErrInviteInvalid
}

// DeleteInvite removes one code.
func (s *Store) DeleteInvite(code string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.invites {
		if s.invites[i].Code == code {
			previous := append([]models.Invite(nil), s.invites...)
			s.invites = append(s.invites[:i], s.invites[i+1:]...)
			if err := s.saveLocked(); err != nil {
				s.invites = previous
				return err
			}
			return nil
		}
	}
	return ErrInviteInvalid
}

// usableInviteLocked validates state without consuming a use.
func (s *Store) usableInviteLocked(code string) *models.Invite {
	now := time.Now().Unix()
	for i := range s.invites {
		invite := &s.invites[i]
		if invite.Code != code || !invite.Enabled {
			continue
		}
		if invite.ExpiresAt > 0 && now >= invite.ExpiresAt {
			continue
		}
		if invite.MaxUses > 0 && invite.UsedCount >= invite.MaxUses {
			continue
		}
		return invite
	}
	return nil
}

// ConsumeInvite validates an invite code and records one use, returning the
// group it grants.
func (s *Store) ConsumeInvite(code string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	invite := s.usableInviteLocked(strings.TrimSpace(code))
	if invite == nil {
		return "", ErrInviteInvalid
	}
	previousUsed := invite.UsedCount
	invite.UsedCount++
	if err := s.saveLocked(); err != nil {
		invite.UsedCount = previousUsed
		return "", err
	}
	return invite.GroupID, nil
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

// GetAppSettings returns the registration policy settings with defaults.
func (s *Store) GetAppSettings() models.AppSettings {
	s.mu.RLock()
	defer s.mu.RUnlock()
	settings := s.appSettings
	if settings.InviteMode == "" {
		settings.InviteMode = models.InviteModeOff
	}
	return settings
}

// SetAppSettings persists the registration policy settings.
func (s *Store) SetAppSettings(settings models.AppSettings) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	switch settings.InviteMode {
	case models.InviteModeOff, models.InviteModeOptional, models.InviteModeRequired:
	default:
		settings.InviteMode = models.InviteModeOff
	}
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
