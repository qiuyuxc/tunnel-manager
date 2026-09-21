package handlers

import (
	"net/http"
	"testing"

	"tunnel-manager/models"
)

func TestUpdateAppSettingsAcceptsRoundTrippedFields(t *testing.T) {
	st := newTestStore(t)
	h := NewManagementHandler(st, nil)
	// The GET response now carries turnstile_has_secret; PUT must accept it
	// back verbatim instead of failing with "invalid request body".
	body := `{"registration_enabled":true,"invite_mode":"optional","default_group_id":"","email_verify_disabled":false,"turnstile_enabled":false,"turnstile_site_key":"","turnstile_has_secret":false}`
	resp := performJSON(t, h.UpdateAppSettings, http.MethodPut, "", body, "")
	if resp.Code != http.StatusOK {
		t.Fatalf("UpdateAppSettings() = %d: %s", resp.Code, resp.Body.String())
	}
	var view models.AppSettingsView
	decodeResponse(t, resp, &view)
	if !view.RegistrationEnabled || view.InviteMode != "optional" {
		t.Fatalf("view = %#v", view)
	}
}

func TestUpdateAppSettingsAcceptsItsOwnGetResponse(t *testing.T) {
	st := newTestStore(t)
	h := NewManagementHandler(st, nil)
	get := performJSON(t, h.GetAppSettings, http.MethodGet, "", "", "")
	if get.Code != http.StatusOK {
		t.Fatalf("GetAppSettings() = %d: %s", get.Code, get.Body.String())
	}
	// Clients hand the document they were given straight back, and the decoder
	// refuses unknown fields. Every field the GET emits therefore needs a
	// counterpart on the request, or saving the form is a 400 for everyone.
	put := performJSON(t, h.UpdateAppSettings, http.MethodPut, "", get.Body.String(), "")
	if put.Code != http.StatusOK {
		t.Fatalf("UpdateAppSettings(round-tripped GET) = %d: %s", put.Code, put.Body.String())
	}
}

func TestUpdateAppSettingsRejectsMismatchedPasskeyRelyingParty(t *testing.T) {
	st := newTestStore(t)
	h := NewManagementHandler(st, nil)
	// Renaming the relying party without moving the origins leaves a pair that
	// fails every passkey request; the save has to be refused, not stored.
	body := `{"passkey_rp_id":"m.veits.bond","passkey_origins":"https://cf.kukie.cn"}`
	resp := performJSON(t, h.UpdateAppSettings, http.MethodPut, "", body, "")
	if resp.Code != http.StatusBadRequest {
		t.Fatalf("mismatched passkey settings = %d: %s", resp.Code, resp.Body.String())
	}
	if stored := st.GetAppSettings(); stored.PasskeyRPID != "" || stored.PasskeyOrigins != "" {
		t.Fatalf("rejected pair was stored: %#v", stored)
	}
}

func TestUpdateAppSettingsAcceptsMatchedPasskeyRelyingParty(t *testing.T) {
	st := newTestStore(t)
	h := NewManagementHandler(st, nil)
	body := `{"passkey_rp_id":"m.veits.bond","passkey_origins":"https://m.veits.bond"}`
	resp := performJSON(t, h.UpdateAppSettings, http.MethodPut, "", body, "")
	if resp.Code != http.StatusOK {
		t.Fatalf("matched passkey settings = %d: %s", resp.Code, resp.Body.String())
	}
	if stored := st.GetAppSettings(); stored.PasskeyRPID != "m.veits.bond" {
		t.Fatalf("relying party = %q; want m.veits.bond", stored.PasskeyRPID)
	}
}

func TestUpdateAppSettingsRejectsEnableWithoutKeys(t *testing.T) {
	st := newTestStore(t)
	h := NewManagementHandler(st, nil)
	body := `{"turnstile_enabled":true,"turnstile_site_key":"","turnstile_secret":""}`
	resp := performJSON(t, h.UpdateAppSettings, http.MethodPut, "", body, "")
	if resp.Code != http.StatusBadRequest {
		t.Fatalf("enabling turnstile without keys = %d: %s", resp.Code, resp.Body.String())
	}
}
