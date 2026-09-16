package store

import (
	"errors"
	"testing"

	"tunnel-manager/models"
)

func TestPasskeyLifecycleAndOwnership(t *testing.T) {
	s := newTestStore(t, HashPassword("password"))
	adminID := s.AdminUserID()
	if err := s.CreateUser(models.User{Username: "bob", PasswordHash: HashPassword("password")}); err != nil {
		t.Fatalf("CreateUser() error = %v", err)
	}
	bob, ok := s.GetUserByUsername("bob")
	if !ok {
		t.Fatal("created account missing")
	}

	if err := s.AddPasskey(models.Passkey{ID: "cred-1", UserID: adminID, Name: "MacBook", Credential: `{"id":"AQ"}`}); err != nil {
		t.Fatalf("AddPasskey() error = %v", err)
	}
	if err := s.AddPasskey(models.Passkey{ID: "cred-2", UserID: bob.ID, Name: "Phone"}); err != nil {
		t.Fatalf("AddPasskey() error = %v", err)
	}

	entries, err := s.ListPasskeys(adminID)
	if err != nil {
		t.Fatalf("ListPasskeys() error = %v", err)
	}
	if len(entries) != 1 || entries[0].Name != "MacBook" {
		t.Fatalf("ListPasskeys() = %#v, want the administrator's single credential", entries)
	}
	if count, err := s.CountPasskeys(adminID); err != nil || count != 1 {
		t.Fatalf("CountPasskeys() = %d, %v, want 1", count, err)
	}
	if count, err := s.CountAdminsWithPasskeys(); err != nil || count != 1 {
		t.Fatalf("CountAdminsWithPasskeys() = %d, %v, want 1 (bob is not an administrator)", count, err)
	}

	// Another account can neither rename nor delete someone else's credential.
	if err := s.RenamePasskey("cred-1", bob.ID, "hijacked"); !errors.Is(err, ErrPasskeyNotFound) {
		t.Fatalf("RenamePasskey() cross-account error = %v, want ErrPasskeyNotFound", err)
	}
	if err := s.DeletePasskey("cred-1", bob.ID); !errors.Is(err, ErrPasskeyNotFound) {
		t.Fatalf("DeletePasskey() cross-account error = %v, want ErrPasskeyNotFound", err)
	}

	if err := s.RenamePasskey("cred-1", adminID, "Desktop"); err != nil {
		t.Fatalf("RenamePasskey() error = %v", err)
	}
	if err := s.UpdatePasskeyCredential("cred-1", `{"id":"AQ","counter":3}`, 12345); err != nil {
		t.Fatalf("UpdatePasskeyCredential() error = %v", err)
	}
	entries, err = s.ListPasskeys(adminID)
	if err != nil {
		t.Fatalf("ListPasskeys() error = %v", err)
	}
	if entries[0].Name != "Desktop" || entries[0].LastUsedAt != 12345 || entries[0].Credential != `{"id":"AQ","counter":3}` {
		t.Fatalf("stored credential = %#v, want the renamed and refreshed row", entries[0])
	}

	if err := s.DeletePasskey("cred-1", adminID); err != nil {
		t.Fatalf("DeletePasskey() error = %v", err)
	}
	if count, err := s.CountPasskeys(adminID); err != nil || count != 0 {
		t.Fatalf("CountPasskeys() after delete = %d, %v, want 0", count, err)
	}
	if count, err := s.CountAdminsWithPasskeys(); err != nil || count != 0 {
		t.Fatalf("CountAdminsWithPasskeys() after delete = %d, %v, want 0", count, err)
	}
}

func TestPasswordLoginSwitchPersistsAndRequiresAccount(t *testing.T) {
	s := newTestStore(t, HashPassword("password"))
	adminID := s.AdminUserID()

	if err := s.SetUserPasswordLoginDisabled("missing", true); !errors.Is(err, ErrUserNotFound) {
		t.Fatalf("SetUserPasswordLoginDisabled() error = %v, want ErrUserNotFound", err)
	}
	if err := s.SetUserPasswordLoginDisabled(adminID, true); err != nil {
		t.Fatalf("SetUserPasswordLoginDisabled() error = %v", err)
	}

	// The flag lives in the users table, which every save rewrites from the
	// cache, so a reload has to see it.
	reopened := NewStore(s.filePath)
	user, ok := reopened.GetUserByID(adminID)
	if !ok || !user.PasswordLoginDisabled {
		t.Fatalf("reloaded account = %#v, want the password switch disabled", user)
	}
	if view := reopened.ListUsers(); len(view) != 1 || !view[0].PasswordLoginDisabled {
		t.Fatalf("ListUsers() = %#v, want the switch exposed to the admin list", view)
	}

	if err := reopened.SetUserPasswordLoginDisabled(adminID, false); err != nil {
		t.Fatalf("SetUserPasswordLoginDisabled() error = %v", err)
	}
	if user, _ := reopened.GetUserByID(adminID); user.PasswordLoginDisabled {
		t.Fatal("password switch still disabled after re-enabling it")
	}
}

func TestDeleteUserRemovesPasskeys(t *testing.T) {
	s := newTestStore(t, HashPassword("password"))
	if err := s.CreateUser(models.User{Username: "bob", PasswordHash: HashPassword("password")}); err != nil {
		t.Fatalf("CreateUser() error = %v", err)
	}
	bob, _ := s.GetUserByUsername("bob")
	if err := s.AddPasskey(models.Passkey{ID: "cred-bob", UserID: bob.ID, Name: "Phone"}); err != nil {
		t.Fatalf("AddPasskey() error = %v", err)
	}

	if err := s.DeleteUser(bob.ID); err != nil {
		t.Fatalf("DeleteUser() error = %v", err)
	}
	if count, err := s.CountPasskeys(bob.ID); err != nil || count != 0 {
		t.Fatalf("CountPasskeys() after DeleteUser = %d, %v, want 0", count, err)
	}
}
