package store

import (
	"path/filepath"
	"testing"

	"tunnel-manager/models"
)

func TestDisconnectOnlyClearsOwnersTunnelAndRollsBackOnFailure(t *testing.T) {
	for _, fail := range []bool{false, true} {
		s := testStore(t)
		ownerID := s.AdminUserID()
		if err := s.CreateUser(models.User{Username: "other", Email: "other@example.com", Role: models.RoleUser}); err != nil {
			t.Fatal(err)
		}
		other, _ := s.GetUserByUsername("other")
		id, err := s.CreateCFConnection(models.CFConnection{UserID: ownerID, AccountID: "account", AccessToken: "encrypted"})
		if err != nil {
			t.Fatal(err)
		}
		if err := s.SetActiveCFConnection(ownerID, id); err != nil {
			t.Fatal(err)
		}
		for uid, name := range map[string]string{ownerID: "Owner", other.ID: "Other"} {
			if err := s.SetUserTunnelSelection(uid, "tunnel-"+name, name); err != nil {
				t.Fatal(err)
			}
		}
		if fail {
			s.dsn = filepath.Join(t.TempDir(), "missing", "store.db")
		}
		err = s.DeleteCFConnection(ownerID, id)
		if (err != nil) != fail {
			t.Fatalf("fail=%v: error=%v", fail, err)
		}
		if p := s.GetUserPrefs(other.ID); p.TunnelID != "tunnel-Other" || p.TunnelName != "Other" {
			t.Fatalf("another account lost its tunnel: %#v", p)
		}
		p := s.GetUserPrefs(ownerID)
		_, active := s.ActiveCFConnection(ownerID)
		if fail && (p.TunnelID != "tunnel-Owner" || !active) {
			t.Fatalf("failed delete mutated state: %#v, active=%v", p, active)
		}
		if !fail && (p.TunnelID != "" || active) {
			t.Fatalf("owner selection survived delete: %#v, active=%v", p, active)
		}
	}
}

func TestCFConnectionLookupEnforcesOwnership(t *testing.T) {
	s := testStore(t)
	id, err := s.CreateCFConnection(models.CFConnection{UserID: s.AdminUserID(), AccessToken: "encrypted"})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := s.CFConnection("another-owner", id); ok {
		t.Fatal("connection credentials crossed accounts")
	}
	if _, ok := s.CFConnection(s.AdminUserID(), id); !ok {
		t.Fatal("owner cannot resolve own connection")
	}
}
