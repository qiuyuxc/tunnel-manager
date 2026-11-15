package handlers

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/fxamacker/cbor/v2"
	"github.com/go-webauthn/webauthn/webauthn"
	"tunnel-manager/auth"
	"tunnel-manager/models"
	"tunnel-manager/services"
)

func signedTestPasskey(t *testing.T, admin *AdminHandler) *ecdsa.PrivateKey {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	publicKey, err := cbor.Marshal(map[int]any{1: 2, 3: -7, -1: 1, -2: key.X.FillBytes(make([]byte, 32)), -3: key.Y.FillBytes(make([]byte, 32))})
	if err != nil {
		t.Fatal(err)
	}
	credential := &webauthn.Credential{ID: []byte{1, 2, 3, 4}, PublicKey: publicKey, AttestationType: "none"}
	encoded, err := services.EncodePasskeyCredential(credential)
	if err != nil {
		t.Fatal(err)
	}
	if err := admin.store.AddPasskey(models.Passkey{ID: "AQIDBA", UserID: admin.store.AdminUserID(), Credential: encoded}); err != nil {
		t.Fatal(err)
	}
	return key
}

func passkeyFactorAssertion(t *testing.T, key *ecdsa.PrivateKey, challenge, userID string) json.RawMessage {
	t.Helper()
	rpHash := sha256.Sum256([]byte("panel.example.com"))
	authenticator := append(rpHash[:], 0x05, 0, 0, 0, 1)
	client := []byte(fmt.Sprintf(`{"type":"webauthn.get","challenge":"%s","origin":"https://panel.example.com"}`, challenge))
	clientHash := sha256.Sum256(client)
	signed := append(append([]byte{}, authenticator...), clientHash[:]...)
	digest := sha256.Sum256(signed)
	signature, err := ecdsa.SignASN1(rand.Reader, key, digest[:])
	if err != nil {
		t.Fatal(err)
	}
	encode := base64.RawURLEncoding.EncodeToString
	raw, err := json.Marshal(map[string]any{"id": "AQIDBA", "rawId": "AQIDBA", "type": "public-key", "response": map[string]string{
		"clientDataJSON": encode(client), "authenticatorData": encode(authenticator), "signature": encode(signature), "userHandle": encode([]byte(userID)),
	}})
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func TestPasskeyFactorCompletionAndCancellation(t *testing.T) {
	for _, scenario := range []string{"finish", "cancel", "different-challenge", "anonymous-finish", "last-begin-budget"} {
		t.Run(scenario, func(t *testing.T) {
			admin, secret, _ := newEnabledAdminHandler(t)
			key := signedTestPasskey(t, admin)
			if scenario == "last-begin-budget" {
				settings := admin.store.GetAppSettings()
				settings.RateLimitPerIP = 1
				if err := admin.store.SetAppSettings(settings); err != nil {
					t.Fatal(err)
				}
				admin.SetThrottle(NewThrottle(admin.store))
			}
			token := beginChallenge(t, admin)
			h := NewPasskeyHandler(admin.store, services.NewPasskeyService(admin.store), admin)
			begin := performJSON(t, h.TwoFactorBegin, "POST", "https://panel.example.com/api/admin/login/2fa/passkey/begin", `{"challenge_token":"`+token+`"}`, "")
			if begin.Code != http.StatusOK {
				t.Fatalf("begin: %d %s", begin.Code, begin.Body.String())
			}
			if admin.challenges[token].inUse {
				t.Fatal("Begin kept login challenge locked")
			}
			var options struct {
				Ceremony  string `json:"ceremony_token"`
				PublicKey struct {
					Challenge string `json:"challenge"`
				} `json:"public_key"`
			}
			decodeResponse(t, begin, &options)
			if scenario == "cancel" {
				factor := auth.GenerateTOTPCode(secret, admin.now().Unix()/30)
				response := performJSON(t, admin.LoginTwoFactor, "POST", "/api/admin/login/2fa", `{"challenge_token":"`+token+`","code":"`+factor+`"}`, "")
				var login models.LoginResponse
				decodeResponse(t, response, &login)
				if response.Code != 200 || !admin.ValidateToken(login.Token) {
					t.Fatalf("TOTP fallback: %d %s", response.Code, response.Body.String())
				}
				return
			}
			targetToken := token
			if scenario == "different-challenge" {
				targetToken = beginChallenge(t, admin)
			}
			credential := passkeyFactorAssertion(t, key, options.PublicKey.Challenge, admin.store.AdminUserID())
			raw, _ := json.Marshal(map[string]any{"challenge_token": targetToken, "ceremony_token": options.Ceremony, "credential": credential})
			finishHandler := h.TwoFactorFinish
			if scenario == "anonymous-finish" {
				finishHandler = h.LoginFinish
			}
			finish := performJSON(t, finishHandler, "POST", "https://panel.example.com/api/admin/login/2fa/passkey/finish", string(raw), "")
			var login models.LoginResponse
			decodeResponse(t, finish, &login)
			if scenario != "finish" && scenario != "last-begin-budget" {
				if finish.Code != 401 || admin.ValidateToken(login.Token) {
					t.Fatalf("binding bypass: %d", finish.Code)
				}
				return
			}
			if finish.Code != 200 || !admin.ValidateToken(login.Token) {
				t.Fatalf("valid assertion: %d %s", finish.Code, finish.Body.String())
			}
			replay := performJSON(t, h.TwoFactorFinish, "POST", "https://panel.example.com/api/admin/login/2fa/passkey/finish", string(raw), "")
			if replay.Code != 401 {
				t.Fatalf("replay: %d", replay.Code)
			}
		})
	}
}
