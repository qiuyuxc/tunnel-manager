package store

import (
	"errors"
	"sync"
	"testing"

	"tunnel-manager/models"
)

func testOIDCProvider(t *testing.T, s *Store) models.OIDCProvider {
	t.Helper()
	p, err := s.SaveOIDCProvider(models.OIDCProvider{Name: "Identity", Issuer: "https://id.example.com", ClientID: "panel", Enabled: true})
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func TestOIDCEmailOwnershipAndStableIdentity(t *testing.T) {
	s := testStore(t)
	p := testOIDCProvider(t, s)
	existing := models.User{ID: "existing", Username: "member", Email: "member@example.com", PasswordHash: HashPassword("password"), Role: models.RoleAdmin}
	if err := s.CreateUser(existing); err != nil {
		t.Fatal(err)
	}
	id := models.OIDCIdentity{ProviderID: p.ID, Issuer: p.Issuer, Subject: "member-sub", Email: "MEMBER@example.com"}
	if _, err := s.ResolveOIDCAccount(id, p.Revision, false, false, "", ""); !errors.Is(err, ErrOIDCEmailProofRequired) {
		t.Fatalf("unverified email: %v", err)
	}
	u, err := s.ResolveOIDCAccount(id, p.Revision, true, false, "", "")
	if err != nil || u.ID != existing.ID || u.Role != existing.Role {
		t.Fatalf("existing account: %+v, %v", u, err)
	}
	id.Email = "changed@example.com"
	u, err = s.ResolveOIDCAccount(id, p.Revision, false, true, "", "")
	if err != nil || u.ID != existing.ID || u.Email != existing.Email {
		t.Fatalf("stable identity: %+v, %v", u, err)
	}
	if _, err := s.ResolveOIDCAccount(id, p.Revision, true, false, s.AdminUserID(), ""); !errors.Is(err, ErrOIDCIdentityTaken) {
		t.Fatalf("identity transfer: %v", err)
	}
}

func TestOIDCProvisioningIsAtomicAndOrdinary(t *testing.T) {
	s := testStore(t)
	p := testOIDCProvider(t, s)
	id := models.OIDCIdentity{ProviderID: p.ID, Issuer: p.Issuer, Subject: "new-sub", Email: "new@example.com"}
	if _, err := s.ResolveOIDCAccount(id, p.Revision, false, true, "", ""); !errors.Is(err, ErrOIDCEmailProofRequired) {
		t.Fatalf("new email: %v", err)
	}
	var wg sync.WaitGroup
	results := make(chan models.User, 8)
	errorsSeen := make(chan error, 8)
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			u, err := s.ResolveOIDCAccount(id, p.Revision, true, true, "", "")
			results <- u
			errorsSeen <- err
		}()
	}
	wg.Wait()
	close(results)
	close(errorsSeen)
	for err := range errorsSeen {
		if err != nil {
			t.Fatal(err)
		}
	}
	userID := ""
	for u := range results {
		if userID == "" {
			userID = u.ID
		}
		if u.ID != userID || u.Role != models.RoleUser || u.PasswordHash != "" || !u.EmailVerified || u.GroupID != s.GetAppSettings().DefaultGroupID {
			t.Fatalf("provisioned account: %+v", u)
		}
	}
	if len(s.ListUsers()) != 2 || len(s.ListOIDCIdentities(userID)) != 1 {
		t.Fatal("concurrent provisioning duplicated account or binding")
	}
	for _, view := range s.ListUsers() {
		if view.ID == userID && (view.HasPassword || view.OIDCIdentities != 1) {
			t.Fatalf("OIDC login methods projection: %+v", view)
		}
	}
	reopened := NewStore(s.dsn)
	if u, exists, linked := reopened.OIDCAccount(p.ID, p.Issuer, id.Subject, ""); !exists || !linked || u.ID != userID {
		t.Fatal("binding did not persist")
	}
}

func TestOIDCConfigurationAndUnlinkProtection(t *testing.T) {
	s := testStore(t)
	p := testOIDCProvider(t, s)
	id := models.OIDCIdentity{ProviderID: p.ID, Issuer: p.Issuer, Subject: "new-sub", Email: "new@example.com"}
	u, err := s.ResolveOIDCAccount(id, p.Revision, true, false, "", "")
	if err != nil {
		t.Fatal(err)
	}
	if err := s.UnlinkOIDC(u.ID, p.ID); !errors.Is(err, ErrOIDCLastLogin) {
		t.Fatalf("last login: %v", err)
	}
	changed := p
	changed.Issuer = "https://other.example.com"
	if _, err := s.SaveOIDCProvider(changed); err == nil {
		t.Fatal("changed bound issuer")
	}
	if err := s.DeleteOIDCProvider(p.ID); err == nil {
		t.Fatal("deleted bound provider")
	}
	p.Enabled = false
	if _, err := s.SaveOIDCProvider(p); err != nil {
		t.Fatal(err)
	}
	if _, err := s.ResolveOIDCAccount(id, p.Revision, true, false, "", ""); !errors.Is(err, ErrOIDCProviderNotFound) {
		t.Fatalf("stale provider: %v", err)
	}
	if err := s.SetUserPasswordHash(u.ID, HashPassword("local-password")); err != nil {
		t.Fatal(err)
	}
	if err := s.UnlinkOIDC(u.ID, p.ID); err != nil {
		t.Fatal(err)
	}
	if err := s.DeleteOIDCProvider(p.ID); err != nil {
		t.Fatal(err)
	}
}

func TestOIDCProvisioningRollsBackFailedSave(t *testing.T) {
	s := testStore(t)
	p := testOIDCProvider(t, s)
	breakStore(t, s)
	id := models.OIDCIdentity{ProviderID: p.ID, Issuer: p.Issuer, Subject: "new-sub", Email: "new@example.com"}
	if _, err := s.ResolveOIDCAccount(id, p.Revision, true, false, "", ""); err == nil {
		t.Fatal("expected save error")
	}
	if _, exists, _ := s.OIDCAccount(p.ID, p.Issuer, id.Subject, id.Email); exists {
		t.Fatal("failed save left a partial account")
	}
}

func TestOIDCPasswordSetupCannotOverwriteExistingPassword(t *testing.T) {
	s := testStore(t)
	p := testOIDCProvider(t, s)
	id := models.OIDCIdentity{ProviderID: p.ID, Issuer: p.Issuer, Subject: "password-sub", Email: "password@example.com"}
	u, err := s.ResolveOIDCAccount(id, p.Revision, true, false, "", "")
	if err != nil {
		t.Fatal(err)
	}
	id.UserID = u.ID
	if err := s.SetOIDCPassword(id, p.Revision, HashPassword("first-password")); err != nil {
		t.Fatal(err)
	}
	if err := s.SetOIDCPassword(id, p.Revision, HashPassword("other-password")); err == nil {
		t.Fatal("overwrote existing password")
	}
	saved, _ := s.GetUserByID(u.ID)
	if valid, _ := verifyPassword("first-password", saved.PasswordHash); !valid {
		t.Fatal("password changed after rejected setup")
	}
}

func TestOIDCBindingPasswordProofIsAtomic(t *testing.T) {
	s := testStore(t)
	p := testOIDCProvider(t, s)
	uid := s.AdminUserID()
	if err := s.SetUserPasswordHash(uid, HashPassword("current-password")); err != nil {
		t.Fatal(err)
	}
	identity := models.OIDCIdentity{ProviderID: p.ID, Issuer: p.Issuer, Subject: "new-binding"}
	if _, err := s.ConfirmOIDCBindingPassword(uid, "wrong-password"); err == nil {
		t.Fatal("accepted incorrect password")
	}
	if _, err := s.ResolveOIDCAccount(identity, p.Revision, true, false, uid, ""); err == nil {
		t.Fatal("bound identity without password proof")
	}
	proof, err := s.ConfirmOIDCBindingPassword(uid, "current-password")
	if err != nil {
		t.Fatal(err)
	}
	if err := s.SetUserPasswordHash(uid, HashPassword("reset-password")); err != nil {
		t.Fatal(err)
	}
	if _, err := s.ResolveOIDCAccount(identity, p.Revision, true, false, uid, proof); err == nil {
		t.Fatal("accepted proof from before password reset")
	}
	proof, err = s.ConfirmOIDCBindingPassword(uid, "reset-password")
	if err != nil {
		t.Fatal(err)
	}
	user, err := s.ResolveOIDCAccount(identity, p.Revision, true, false, uid, proof)
	if err != nil || user.ID != uid || len(s.ListOIDCIdentities(uid)) != 1 {
		t.Fatalf("confirmed binding: %+v %v", user, err)
	}
}
