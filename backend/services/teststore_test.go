package services

import (
	"path/filepath"
	"testing"

	"tunnel-manager/store"
)

// newTestStore opens a throwaway store with the administrator account the
// setup wizard creates on a fresh install, so a test starts from the same
// state as an installed panel.
func newTestStore(t *testing.T) *store.Store {
	t.Helper()
	return newTestStoreAt(t, filepath.Join(t.TempDir(), "config.json"))
}

// newTestStoreAt is newTestStore for the tests that reopen the same document,
// such as reload and migration checks.
func newTestStoreAt(t *testing.T, path string) *store.Store {
	t.Helper()
	st := store.NewStore(path)
	if err := st.CreateAdmin("admin", "admin123"); err != nil {
		t.Fatalf("create test administrator: %v", err)
	}
	return st
}
