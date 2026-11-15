package services

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"tunnel-manager/auth"
	"tunnel-manager/models"
)

func TestCloudflareReadStopsWhenPageRequestIsCancelled(t *testing.T) {
	started := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(started)
		<-r.Context().Done()
	}))
	defer server.Close()
	client := NewCloudflareClient("test-token", "account")
	client.baseURL = server.URL
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() { _, err := client.WithContext(ctx).ListTunnels(); done <- err }()
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("request did not start")
	}
	cancel()
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("request error = %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("upstream call outlived its page request")
	}
}

func oauthConnectionForTest(t *testing.T, oauth *CloudflareOAuth, label string, expired bool) models.CFConnection {
	t.Helper()
	access, err := auth.EncryptSecret(oauth.encryptionKey, cloudflareAccessTokenPurpose, []byte("access-"+label))
	if err != nil {
		t.Fatal(err)
	}
	refresh, err := auth.EncryptSecret(oauth.encryptionKey, cloudflareRefreshTokenPurpose, []byte("refresh-"+label))
	if err != nil {
		t.Fatal(err)
	}
	expires := oauth.now().Add(time.Hour).Unix()
	if expired {
		expires = oauth.now().Add(-time.Minute).Unix()
	}
	conn := models.CFConnection{UserID: oauth.store.AdminUserID(), Label: label, AccountID: label, AccessToken: access, RefreshToken: refresh, ExpiresAt: expires}
	conn.ID, err = oauth.store.CreateCFConnection(conn)
	if err != nil {
		t.Fatal(err)
	}
	return conn
}

func TestConcurrentCloudflareRefreshUsesRotatedTokenOnce(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"access_token": "fresh", "refresh_token": "rotated", "expires_in": 3600})
	}))
	defer server.Close()
	oauth := NewCloudflareOAuth(newOAuthTestStore(t), bytes.Repeat([]byte{4}, 32), CloudflareOAuthConfig{ClientID: "client", ClientSecret: "secret"})
	oauth.tokenEndpoint = server.URL
	conn := oauthConnectionForTest(t, oauth, "same-grant", true)
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			token, err := oauth.AccessTokenFor(conn)
			if err != nil || token != "fresh" {
				t.Errorf("token=%q err=%v", token, err)
			}
		}()
	}
	wg.Wait()
	if calls.Load() != 1 {
		t.Fatalf("same stale snapshot caused %d refreshes", calls.Load())
	}
}

func TestSlowCloudflareRefreshDoesNotBlockOtherConnection(t *testing.T) {
	started := make(chan struct{})
	release := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(started)
		<-release
		_ = json.NewEncoder(w).Encode(map[string]any{"access_token": "fresh", "expires_in": 3600})
	}))
	defer server.Close()
	oauth := NewCloudflareOAuth(newOAuthTestStore(t), bytes.Repeat([]byte{5}, 32), CloudflareOAuthConfig{ClientID: "client", ClientSecret: "secret"})
	oauth.tokenEndpoint = server.URL
	slow := oauthConnectionForTest(t, oauth, "slow", true)
	fast := oauthConnectionForTest(t, oauth, "fast", false)
	// Avoid the deliberate hash collisions of the fixed memory bound.
	slotFor := func(id string) uint32 {
		var hash uint32 = 2166136261
		for _, b := range []byte(id) {
			hash = (hash ^ uint32(b)) * 16777619
		}
		return hash % 64
	}
	for slotFor(slow.ID) == slotFor(fast.ID) {
		fast = oauthConnectionForTest(t, oauth, "fast", false)
	}
	done := make(chan error, 1)
	go func() { _, err := oauth.AccessTokenFor(slow); done <- err }()
	<-started
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	token, err := oauth.AccessTokenForContext(ctx, fast)
	close(release)
	if slowErr := <-done; slowErr != nil {
		t.Fatal(slowErr)
	}
	if err != nil || token != "access-fast" {
		t.Fatalf("unrelated connection blocked: token=%q error=%v", token, err)
	}
}

func TestDeletedCloudflareConnectionCannotUsePinnedCredentials(t *testing.T) {
	oauth := NewCloudflareOAuth(newOAuthTestStore(t), bytes.Repeat([]byte{6}, 32), CloudflareOAuthConfig{})
	conn := oauthConnectionForTest(t, oauth, "removed", false)
	if err := oauth.store.DeleteCFConnection(conn.UserID, conn.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := oauth.AccessTokenFor(conn); !errors.Is(err, ErrCloudflareOAuthNotConnected) {
		t.Fatalf("removed grant remained usable: %v", err)
	}
}
