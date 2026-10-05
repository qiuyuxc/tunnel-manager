package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"tunnel-manager/auth"
	"tunnel-manager/models"
)

func TestLabCloudflareConnectionIsolation(t *testing.T) {
	state := newTestStore(t)
	owner := state.AdminUserID()
	client := NewCloudflareClient("project-token", "project-account")
	client.SetSessionStore(state)
	if client.LabDNSAvailability(owner).Available || client.ForUser("unbound-user").LabDNSAvailability("unbound-user").Available {
		t.Fatal("an unbound identity inherited project credentials")
	}
	if _, err := client.ForUser("unbound-user").LabDNSForUser("unbound-user"); err == nil {
		t.Fatal("unbound user can construct a DNS writer")
	}
	bound := client.ForUser(owner)
	provider, err := bound.LabDNSForUser(owner)
	if err != nil || provider.(*labCloudflareDNS).token != "project-token" {
		t.Fatalf("primary administrator cannot reuse project credentials: %v", err)
	}
	key := bytes.Repeat([]byte{7}, 32)
	client.SetOAuth(NewCloudflareOAuth(state, key, CloudflareOAuthConfig{}))
	secret, err := auth.EncryptSecret(key, cloudflareAccessTokenPurpose, []byte("personal-token"))
	if err != nil {
		t.Fatal(err)
	}
	connectionID, err := state.CreateCFConnection(models.CFConnection{UserID: owner, Label: "personal", AccountID: "personal-account", AccessToken: secret, ExpiresAt: time.Now().Add(time.Hour).Unix()})
	if err != nil {
		t.Fatal(err)
	}
	if err := state.SetActiveCFConnection(owner, connectionID); err != nil {
		t.Fatal(err)
	}
	bound = client.ForUser(owner)
	provider, err = bound.LabDNSForUser(owner)
	if err != nil || provider.(*labCloudflareDNS).token != "personal-token" || provider.(*labCloudflareDNS).accountID != "personal-account" {
		t.Fatalf("incorrect bound account: %v", err)
	}
	if err := state.UpdateCFConnectionTokens(connectionID, secret, "", time.Now().Add(-time.Hour).Unix(), ""); err != nil {
		t.Fatal(err)
	}
	if _, err := bound.LabDNSForUser(owner); err == nil {
		t.Fatal("expired personal authorization silently fell back to project credentials")
	}
	if provider.(*labCloudflareDNS).token != "personal-token" {
		t.Fatal("an in-flight writer switched credentials")
	}
}

func TestLabCloudflareFindsOnlyCurrentAccountZone(t *testing.T) {
	for _, mode := range []string{"parent", "exact", "wrong-account", "similar-name", "permission-denied"} {
		t.Run(mode, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
				if request.Method != http.MethodGet || request.URL.Path != "/zones" || request.Header.Get("Authorization") != "Bearer test-token" || request.URL.Query().Get("account.id") != "current-account" {
					t.Fatal("zone query escaped its selected account")
				}
				if mode == "permission-denied" {
					http.Error(response, "forbidden", http.StatusForbidden)
					return
				}
				name := request.URL.Query().Get("name")
				account := "current-account"
				if mode == "parent" && name != "example.com" {
					fmt.Fprint(response, `{"success":true,"result":[]}`)
					return
				}
				if mode == "wrong-account" {
					account = "other-account"
				}
				if mode == "similar-name" {
					name = "notexample.com"
				}
				fmt.Fprintf(response, `{"success":true,"result":[{"id":"zone","name":%q,"status":"active","account":{"id":%q}}]}`, name, account)
			}))
			defer server.Close()
			client := &labCloudflareDNS{baseURL: server.URL, token: "test-token", accountID: "current-account", client: server.Client()}
			zone, err := client.FindZone(context.Background(), "probe.example.com")
			if mode == "parent" || mode == "exact" {
				want := "probe.example.com"
				if mode == "parent" {
					want = "example.com"
				}
				if err != nil || zone.Name != want {
					t.Fatalf("zone lookup: %#v %v", zone, err)
				}
			} else if err == nil {
				t.Fatal("found a zone outside the active account/domain")
			}
		})
	}
}

func TestLabCloudflareTXTIsAdditiveIdempotentAndBounded(t *testing.T) {
	for _, mode := range []string{"new", "existing", "quoted", "second-page", "denied", "redirect", "oversized"} {
		t.Run(mode, func(t *testing.T) {
			var writes, requests atomic.Int32
			name, token := "_tunnel-manager-verify.example.com", "tunnel-manager=instance.nonce"
			server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
				requests.Add(1)
				if request.Method == http.MethodPut || request.Method == http.MethodDelete {
					t.Error("automatic TXT must not replace or delete any existing record")
				}
				if mode == "denied" {
					http.Error(response, "forbidden", http.StatusForbidden)
					return
				}
				if mode == "redirect" {
					response.Header().Set("Location", "/credentials-must-not-follow")
					response.WriteHeader(http.StatusFound)
					return
				}
				if mode == "oversized" {
					fmt.Fprint(response, strings.Repeat("x", (1<<20)+2))
					return
				}
				if request.Method == http.MethodGet {
					if request.URL.Query().Get("name") != name || request.URL.Query().Get("type") != "TXT" {
						t.Error("TXT query is not scoped to the verification name")
					}
					content := "unrelated-existing-TXT"
					pages := 1
					if mode == "existing" {
						content = token
					}
					if mode == "quoted" {
						content = strconv.Quote(token)
					}
					if mode == "second-page" {
						pages = 2
						if request.URL.Query().Get("page") == "2" {
							content = token
						}
					}
					fmt.Fprintf(response, `{"success":true,"result":[{"id":"existing","type":"TXT","name":%q,"content":%q}],"result_info":{"total_pages":%d}}`, name, content, pages)
					return
				}
				writes.Add(1)
				var payload models.DNSRecordRequest
				if err := json.NewDecoder(request.Body).Decode(&payload); err != nil || payload.Type != "TXT" || payload.Name != name || payload.Content != token || payload.Proxied || payload.TTL != 60 {
					t.Error("unexpected TXT write")
				}
				fmt.Fprint(response, `{"success":true,"result":{"id":"new"}}`)
			}))
			defer server.Close()
			state := newTestStore(t)
			base := NewCloudflareClient("test-token", "account")
			base.SetSessionStore(state)
			base.baseURL, base.httpClient = server.URL, server.Client()
			provider, err := base.ForUser(state.AdminUserID()).LabDNSForUser(state.AdminUserID())
			if err != nil {
				t.Fatal(err)
			}
			created, err := provider.EnsureTXT(context.Background(), models.Zone{ID: "zone", Name: "example.com"}, name, token)
			if mode == "denied" || mode == "redirect" || mode == "oversized" {
				if err == nil || writes.Load() != 0 || requests.Load() != 1 {
					t.Fatalf("unsafe error handling: created=%v, requests=%d, writes=%d, err=%v", created, requests.Load(), writes.Load(), err)
				}
			} else if err != nil || created != (mode == "new") || (writes.Load() == 1) != (mode == "new") {
				t.Fatalf("unexpected creation: %v, writes=%d, err=%v", created, writes.Load(), err)
			}
		})
	}
}

func TestLabCloudflareCancelledLookupDoesNotWrite(t *testing.T) {
	started := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodGet {
			t.Error("cancelled lookup reached a TXT write")
		}
		close(started)
		<-request.Context().Done()
	}))
	defer server.Close()
	client := &labCloudflareDNS{baseURL: server.URL, token: "test", accountID: "account", client: server.Client()}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	done := make(chan error, 1)
	go func() {
		_, err := client.EnsureTXT(ctx, models.Zone{ID: "zone", Name: "example.com"}, "_tunnel-manager-verify.example.com", "tunnel-manager=token")
		done <- err
	}()
	select {
	case <-started:
	case <-ctx.Done():
		t.Fatal("lookup did not start")
	}
	cancel()
	if err := <-done; err == nil {
		t.Fatal("cancelled write succeeded")
	}
}
