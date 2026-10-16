package services

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	jose "github.com/go-jose/go-jose/v4"
	"golang.org/x/oauth2"
	"tunnel-manager/models"
)

func TestOIDCExchangeVerifiesSignedIdentityAndUserInfo(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	signer, err := jose.NewSigner(jose.SigningKey{Algorithm: jose.RS256, Key: key}, (&jose.SignerOptions{}).WithHeader("kid", "test"))
	if err != nil {
		t.Fatal(err)
	}
	for _, scenario := range []string{"verified", "userinfo", "nonce", "audience", "issuer", "expired", "userinfo-subject", "signature", "missing-token"} {
		t.Run(scenario, func(t *testing.T) {
			var issuer string
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				switch r.URL.Path {
				case "/.well-known/openid-configuration":
					_ = json.NewEncoder(w).Encode(map[string]any{"issuer": issuer, "authorization_endpoint": issuer + "/authorize", "token_endpoint": issuer + "/token", "userinfo_endpoint": issuer + "/userinfo", "jwks_uri": issuer + "/keys", "id_token_signing_alg_values_supported": []string{"RS256"}})
				case "/keys":
					_ = json.NewEncoder(w).Encode(jose.JSONWebKeySet{Keys: []jose.JSONWebKey{{Key: &key.PublicKey, KeyID: "test", Algorithm: "RS256", Use: "sig"}}})
				case "/token":
					_ = r.ParseForm()
					if r.Form.Get("code") != "code" || r.Form.Get("code_verifier") != "verifier" || r.Form.Get("redirect_uri") != "http://localhost/callback" {
						http.Error(w, "PKCE or callback missing", 400)
						return
					}
					claims := map[string]any{"iss": issuer, "sub": "member", "aud": "panel", "exp": time.Now().Add(time.Minute).Unix(), "iat": time.Now().Unix(), "auth_time": time.Now().Unix(), "nonce": "nonce", "email": "Member@EXAMPLE.com", "email_verified": true}
					switch scenario {
					case "nonce":
						claims["nonce"] = "other"
					case "audience":
						claims["aud"] = "other-panel"
					case "issuer":
						claims["iss"] = "https://other.example.com"
					case "expired":
						claims["exp"] = time.Now().Add(-time.Minute).Unix()
					case "userinfo", "userinfo-subject":
						delete(claims, "email")
						delete(claims, "email_verified")
					}
					body, _ := json.Marshal(claims)
					token, _ := signer.Sign(body)
					raw, _ := token.CompactSerialize()
					if scenario == "signature" {
						raw = raw[:len(raw)-8] + "invalid!"
					}
					response := map[string]any{"access_token": "access", "token_type": "Bearer", "expires_in": 60}
					if scenario != "missing-token" {
						response["id_token"] = raw
					}
					_ = json.NewEncoder(w).Encode(response)
				case "/userinfo":
					if r.Header.Get("Authorization") != "Bearer access" {
						http.Error(w, "missing access token", 401)
						return
					}
					subject := "member"
					if scenario == "userinfo-subject" {
						subject = "other-member"
					}
					_ = json.NewEncoder(w).Encode(map[string]any{"sub": subject, "email": "Member@EXAMPLE.com", "email_verified": true})
				default:
					http.NotFound(w, r)
				}
			}))
			defer server.Close()
			issuer = server.URL
			client, err := NewOIDCClient(context.Background(), models.OIDCProvider{Issuer: issuer, ClientID: "panel", RedirectURI: "http://localhost/callback"}, "secret")
			if err != nil {
				t.Fatal(err)
			}
			claims, err := client.Exchange(context.Background(), "code", "nonce", "verifier")
			if scenario == "verified" || scenario == "userinfo" {
				if err != nil || claims.Email != "member@example.com" || !claims.EmailVerified || claims.Subject != "member" || claims.AuthTime == 0 {
					t.Fatalf("verified claims: %+v %v", claims, err)
				}
			} else if err == nil {
				t.Fatalf("accepted invalid identity: %s", scenario)
			}
		})
	}
}

func TestOIDCURLValidation(t *testing.T) {
	for _, value := range []string{"https://id.example.com/realm", "http://127.0.0.1:8085", "http://localhost/callback", "http://[::1]:8080/callback"} {
		if err := ValidateOIDCURL(value); err != nil {
			t.Fatalf("valid URL %s: %v", value, err)
		}
	}
	for _, value := range []string{"http://id.example.com", "https://user:pass@id.example.com", "https://id.example.com#fragment", "https://id.example.com?query=1", "javascript:alert(1)", "/callback"} {
		if err := ValidateOIDCURL(value); err == nil {
			t.Fatalf("accepted unsafe URL %s", value)
		}
	}
}

// Keep PKCE generation exercised with the maintained oauth2 implementation.
func TestOIDCAuthorizationCarriesPKCEAndReauthentication(t *testing.T) {
	c := &OIDCClient{config: oauth2.Config{ClientID: "panel", RedirectURL: "https://panel.example.com/callback", Endpoint: oauth2.Endpoint{AuthURL: "https://id.example.com/authorize"}, Scopes: []string{"openid", "email"}}}
	request, err := http.NewRequest("GET", c.AuthorizationURL("state", "nonce", "verifier", true), nil)
	if err != nil {
		t.Fatal(err)
	}
	q := request.URL.Query()
	if q.Get("state") != "state" || q.Get("nonce") != "nonce" || q.Get("code_challenge_method") != "S256" || q.Get("code_challenge") == "" || q.Get("prompt") != "login" || q.Get("max_age") != "0" {
		t.Fatalf("authorization parameters: %v", q)
	}
}
