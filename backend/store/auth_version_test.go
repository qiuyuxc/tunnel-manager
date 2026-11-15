package store

import (
	"errors"
	"path/filepath"
	"testing"
	"time"
)

func TestPasswordRevocationIsAtomicAndPersists(t *testing.T) {
	s := testStore(t)
	owner, _ := s.GetUserByID(s.AdminUserID())
	expiry := time.Now().Add(time.Hour).Unix()
	if err := s.CreateSession("before", owner.ID, expiry, owner.AuthVersion); err != nil {
		t.Fatal(err)
	}
	if err := s.SetUserPasswordHash(owner.ID, HashPassword("new-password")); err != nil {
		t.Fatal(err)
	}
	if _, valid := s.GetSessionUser("before", time.Now().Unix()); valid {
		t.Fatal("pre-reset session survived")
	}
	if err := s.CreateSession("late", owner.ID, expiry, owner.AuthVersion); !errors.Is(err, ErrAuthStateChanged) {
		t.Fatalf("late session: %v", err)
	}
	reloaded := NewStore(s.dsn)
	current, _ := reloaded.GetUserByID(owner.ID)
	if current.AuthVersion != owner.AuthVersion+1 {
		t.Fatalf("version not persisted: %d", current.AuthVersion)
	}
	if err := reloaded.CreateSession("after", owner.ID, expiry, current.AuthVersion); err != nil {
		t.Fatalf("fresh login: %v", err)
	}
	if _, valid := reloaded.GetSessionUser("after", time.Now().Unix()); !valid {
		t.Fatal("fresh session unusable")
	}
}

func TestFailedPasswordRevocationRollsBackVersionAndSessions(t *testing.T) {
	s := testStore(t)
	owner, _ := s.GetUserByID(s.AdminUserID())
	if err := s.CreateSession("existing", owner.ID, time.Now().Add(time.Hour).Unix(), owner.AuthVersion); err != nil {
		t.Fatal(err)
	}
	s.dsn = filepath.Join(t.TempDir(), "missing", "store.db")
	if err := s.SetUserPasswordHash(owner.ID, HashPassword("new-password")); err == nil {
		t.Fatal("expected persistence failure")
	}
	current, _ := s.GetUserByID(owner.ID)
	if current.AuthVersion != owner.AuthVersion || current.PasswordHash != owner.PasswordHash {
		t.Fatal("failed save changed credential state")
	}
	if _, valid := s.GetSessionUser("existing", time.Now().Unix()); !valid {
		t.Fatal("failed save lost session")
	}
}
