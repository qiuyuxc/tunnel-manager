package store

import (
	"errors"
	"sort"
	"strings"
	"time"

	"tunnel-manager/models"
)

var (
	ErrOIDCProviderNotFound   = errors.New("OIDC 接入方不存在或已停用")
	ErrOIDCIdentityTaken      = errors.New("该 OIDC 身份已关联其他账号")
	ErrOIDCEmailProofRequired = errors.New("需要验证邮箱归属")
	ErrOIDCLastLogin          = errors.New("请先设置本站密码或绑定其他可用登录方式")
)

func (s *Store) ListOIDCProviders() []models.OIDCProvider {
	s.mu.RLock()
	defer s.mu.RUnlock()
	providers := append([]models.OIDCProvider{}, s.oidc.Providers...)
	sort.SliceStable(providers, func(i, j int) bool { return providers[i].Order < providers[j].Order })
	return providers
}

func (s *Store) GetOIDCProvider(id string) (models.OIDCProvider, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.oidcProviderLocked(id)
}

func (s *Store) oidcProviderLocked(id string) (models.OIDCProvider, bool) {
	for _, provider := range s.oidc.Providers {
		if provider.ID == id {
			return provider, true
		}
	}
	return models.OIDCProvider{}, false
}

func (s *Store) SaveOIDCProvider(provider models.OIDCProvider) (models.OIDCProvider, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	previous := append([]models.OIDCProvider(nil), s.oidc.Providers...)
	index := -1
	for i, existing := range s.oidc.Providers {
		if existing.ID == provider.ID {
			index = i
			provider.Revision = existing.Revision + 1
			break
		}
	}
	if provider.ID != "" && index < 0 {
		return models.OIDCProvider{}, ErrOIDCProviderNotFound
	}
	if index >= 0 {
		existing := s.oidc.Providers[index]
		if existing.Issuer != provider.Issuer || existing.ClientID != provider.ClientID {
			for _, identity := range s.oidc.Identities {
				if identity.ProviderID == provider.ID {
					return models.OIDCProvider{}, errors.New("已有账号关联，请新建接入方以更换 Issuer 或 Client ID")
				}
			}
		}
		s.oidc.Providers[index] = provider
	} else {
		if len(s.oidc.Providers) >= 32 {
			return models.OIDCProvider{}, errors.New("最多配置 32 个 OIDC 接入方")
		}
		provider.ID = newID()
		provider.Revision = 1
		s.oidc.Providers = append(s.oidc.Providers, provider)
	}
	if err := s.saveLocked(); err != nil {
		s.oidc.Providers = previous
		return models.OIDCProvider{}, err
	}
	return provider, nil
}

func (s *Store) DeleteOIDCProvider(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, identity := range s.oidc.Identities {
		if identity.ProviderID == id {
			return errors.New("接入方仍有关联账号，可先停用，全部解绑后再删除")
		}
	}
	for i, provider := range s.oidc.Providers {
		if provider.ID != id {
			continue
		}
		previous := append([]models.OIDCProvider(nil), s.oidc.Providers...)
		s.oidc.Providers = append(s.oidc.Providers[:i], s.oidc.Providers[i+1:]...)
		if err := s.saveLocked(); err != nil {
			s.oidc.Providers = previous
			return err
		}
		return nil
	}
	return ErrOIDCProviderNotFound
}

func (s *Store) ListOIDCIdentities(userID string) []models.OIDCIdentity {
	s.mu.RLock()
	defer s.mu.RUnlock()
	identities := []models.OIDCIdentity{}
	for _, identity := range s.oidc.Identities {
		if identity.UserID == userID {
			identities = append(identities, identity)
		}
	}
	return identities
}

// OIDCAccount resolves an existing stable identity before considering its
// current email. A linked identity cannot move accounts when its email changes.
func (s *Store) OIDCAccount(providerID, issuer, subject, email string) (models.User, bool, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, identity := range s.oidc.Identities {
		if identity.ProviderID == providerID && identity.Issuer == issuer && identity.Subject == subject {
			if user := s.findUserLocked(identity.UserID); user != nil {
				return copyUser(*user), true, true
			}
		}
	}
	for _, user := range s.users {
		if email != "" && strings.EqualFold(user.Email, email) {
			return copyUser(user), true, false
		}
	}
	return models.User{}, false, false
}

// ConfirmOIDCBindingPassword returns the exact credential proved by the caller.
// The later atomic binding rejects resets made during external authorization.
func (s *Store) ConfirmOIDCBindingPassword(userID, password string) (string, error) {
	user, ok := s.GetUserByID(userID)
	if !ok || user.Status != models.UserActive || user.PasswordHash == "" || password == "" {
		return "", errors.New("请先设置并确认本站密码")
	}
	valid, _ := verifyPassword(password, user.PasswordHash)
	if !valid {
		return "", errors.New("本站密码不正确")
	}
	return user.PasswordHash, nil
}

// ResolveOIDCAccount atomically rechecks provider revision, email ownership,
// account status and uniqueness before linking or provisioning an ordinary user.
func (s *Store) ResolveOIDCAccount(identity models.OIDCIdentity, revision uint64, emailProven, requireNewEmailProof bool, ownerID, passwordProof string) (models.User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	provider, ok := s.oidcProviderLocked(identity.ProviderID)
	if !ok || !provider.Enabled || provider.Revision != revision || provider.Issuer != identity.Issuer {
		return models.User{}, ErrOIDCProviderNotFound
	}
	for _, existing := range s.oidc.Identities {
		if existing.ProviderID == identity.ProviderID && existing.Issuer == identity.Issuer && existing.Subject == identity.Subject {
			if ownerID != "" && existing.UserID != ownerID {
				return models.User{}, ErrOIDCIdentityTaken
			}
			user := s.findUserLocked(existing.UserID)
			if user == nil || user.Status != models.UserActive {
				return models.User{}, errors.New("账号已禁用或不存在")
			}
			return copyUser(*user), nil
		}
	}
	var user *models.User
	if ownerID != "" {
		user = s.findUserLocked(ownerID)
		if user == nil {
			return models.User{}, ErrUserNotFound
		}
		if passwordProof == "" || user.PasswordHash != passwordProof {
			return models.User{}, errors.New("本站密码已变化，请重新确认后绑定")
		}
	} else {
		for i := range s.users {
			if identity.Email != "" && strings.EqualFold(s.users[i].Email, identity.Email) {
				user = &s.users[i]
				break
			}
		}
		if user != nil && !emailProven {
			return models.User{}, ErrOIDCEmailProofRequired
		}
		if user == nil && requireNewEmailProof && !emailProven {
			return models.User{}, ErrOIDCEmailProofRequired
		}
	}
	previousUsers := append([]models.User(nil), s.users...)
	previousIdentities := append([]models.OIDCIdentity(nil), s.oidc.Identities...)
	if user == nil {
		if identity.Email == "" {
			return models.User{}, errors.New("OIDC 未返回有效邮箱")
		}
		id := newID()
		created := models.User{ID: id, Username: "oidc_" + id[:12], Email: identity.Email, Role: models.RoleUser,
			GroupID: s.appSettings.DefaultGroupID, Status: models.UserActive, EmailVerified: emailProven, CreatedAt: time.Now().Unix()}
		if err := s.usernameFreeLocked(created.Username, ""); err != nil {
			return models.User{}, err
		}
		if err := s.emailFreeLocked(created.Email, ""); err != nil {
			return models.User{}, err
		}
		s.users = append(s.users, created)
		user = &s.users[len(s.users)-1]
	}
	if user.Status != models.UserActive {
		return models.User{}, errors.New("账号已禁用")
	}
	identity.UserID = user.ID
	identity.CreatedAt = time.Now().Unix()
	s.oidc.Identities = append(s.oidc.Identities, identity)
	result := copyUser(*user)
	if err := s.saveLocked(); err != nil {
		s.users = previousUsers
		s.oidc.Identities = previousIdentities
		return models.User{}, err
	}
	return result, nil
}

func (s *Store) UnlinkOIDC(userID, providerID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	user := s.findUserLocked(userID)
	if user == nil {
		return ErrUserNotFound
	}
	available := user.PasswordHash != "" && !user.PasswordLoginDisabled && !s.appSettings.PasswordLoginDisabled
	if !available {
		count, err := s.CountPasskeys(userID)
		if err != nil {
			return err
		}
		available = count > 0
	}
	for _, identity := range s.oidc.Identities {
		if identity.UserID == userID && identity.ProviderID != providerID {
			if provider, ok := s.oidcProviderLocked(identity.ProviderID); ok && provider.Enabled {
				available = true
			}
		}
	}
	if !available {
		return ErrOIDCLastLogin
	}
	previous := s.oidc.Identities
	remaining := make([]models.OIDCIdentity, 0, len(previous))
	for _, identity := range previous {
		if identity.UserID != userID || identity.ProviderID != providerID {
			remaining = append(remaining, identity)
		}
	}
	s.oidc.Identities = remaining
	if err := s.saveLocked(); err != nil {
		s.oidc.Identities = previous
		return err
	}
	return nil
}

// SetOIDCPassword atomically rechecks the account, exact binding and provider
// before installing the first password. A concurrent password reset wins.
func (s *Store) SetOIDCPassword(identity models.OIDCIdentity, revision uint64, passwordHash string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	provider, ok := s.oidcProviderLocked(identity.ProviderID)
	if !ok || !provider.Enabled || provider.Revision != revision || provider.Issuer != identity.Issuer {
		return ErrOIDCProviderNotFound
	}
	user := s.findUserLocked(identity.UserID)
	if user == nil || user.Status != models.UserActive || user.PasswordHash != "" || passwordHash == "" {
		return errors.New("账号状态已变化，请重新开始")
	}
	linked := false
	for _, existing := range s.oidc.Identities {
		if existing.ProviderID == identity.ProviderID && existing.Issuer == identity.Issuer && existing.Subject == identity.Subject && existing.UserID == identity.UserID {
			linked = true
			break
		}
	}
	if !linked {
		return errors.New("OIDC 身份已解绑，请重新开始")
	}
	user.PasswordHash = passwordHash
	if err := s.saveLocked(); err != nil {
		user.PasswordHash = ""
		return err
	}
	return nil
}
