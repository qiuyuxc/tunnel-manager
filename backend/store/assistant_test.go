package store

import (
	"errors"
	"testing"

	"tunnel-manager/models"
)

func TestAIStoreIsolationReloadAndRollback(t *testing.T) {
	state := testStore(t)
	userID := state.AdminUserID()
	if err := state.UpdateAI(userID, func(global *models.AIState, user *models.AIUserState) error {
		global.SharedEnabled = true
		global.Shared = models.AIConnection{Endpoint: "https://example.com/v1", Model: "model", KeyEncrypted: "encrypted"}
		user.Connection.Endpoint = "https://personal.example.com/v1"
		user.Conversations = []models.AIConversation{{ID: "conversation", Messages: []models.AIMessage{{Role: "user", Content: "hello"}}, Tasks: []models.AITask{{ID: "task", Status: "running"}}}}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	_, _, other := state.AIStateFor("other")
	if other.Connection.Endpoint != "" || len(other.Conversations) != 0 {
		t.Fatal("cross-user data leak")
	}
	_, _, copy := state.AIStateFor(userID)
	copy.Conversations[0].Messages[0].Content = "mutation"
	_, _, actual := state.AIStateFor(userID)
	if actual.Conversations[0].Messages[0].Content != "hello" {
		t.Fatal("getter aliases stored memory")
	}
	reloaded := NewStore(state.dsn)
	enabled, shared, actual := reloaded.AIStateFor(userID)
	if !enabled || shared.KeyEncrypted != "encrypted" || actual.Conversations[0].Tasks[0].Status != "unknown" {
		t.Fatal("settings or interrupted task did not recover")
	}
	if err := state.UpdateAI(userID, func(global *models.AIState, user *models.AIUserState) error {
		global.SharedEnabled = false
		user.Connection.Model = "bad"
		return errors.New("reject")
	}); err == nil {
		t.Fatal("expected rejected update")
	}
	enabled, _, actual = state.AIStateFor(userID)
	if !enabled || actual.Connection.Model != "" {
		t.Fatal("callback failure changed state")
	}
	breakStore(t, state)
	if err := state.UpdateAI(userID, func(_ *models.AIState, user *models.AIUserState) error { user.Connection.Model = "bad"; return nil }); err == nil {
		t.Fatal("expected persistence failure")
	}
	_, _, actual = state.AIStateFor(userID)
	if actual.Connection.Model != "" {
		t.Fatal("persistence failure changed state")
	}
}

func TestAIDeletedUserDataIsRemoved(t *testing.T) {
	state := testStore(t)
	if err := state.CreateUser(models.User{ID: "member", Username: "member"}); err != nil {
		t.Fatal(err)
	}
	if err := state.UpdateAI("member", func(_ *models.AIState, user *models.AIUserState) error {
		user.Connection.KeyEncrypted = "retired-secret"
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if err := state.DeleteUser("member"); err != nil {
		t.Fatal(err)
	}
	_, _, current := state.AIStateFor("member")
	if current.Connection.KeyEncrypted != "" {
		t.Fatal("deleted user remains in memory")
	}
	reloaded := NewStore(state.dsn)
	_, _, current = reloaded.AIStateFor("member")
	if current.Connection.KeyEncrypted != "" {
		t.Fatal("deleted user remains persisted")
	}
}
