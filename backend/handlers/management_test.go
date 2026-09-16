package handlers

import (
	"net/http"
	"path/filepath"
	"testing"

	"tunnel-manager/models"
	"tunnel-manager/store"
)

func TestUpdateAppSettingsAcceptsRoundTrippedFields(t *testing.T) {
	st := store.NewStore(filepath.Join(t.TempDir(), "config.json"))
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
	st := store.NewStore(filepath.Join(t.TempDir(), "config.json"))
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

func TestUpdateAppSettingsRejectsEnableWithoutKeys(t *testing.T) {
	st := store.NewStore(filepath.Join(t.TempDir(), "config.json"))
	h := NewManagementHandler(st, nil)
	body := `{"turnstile_enabled":true,"turnstile_site_key":"","turnstile_secret":""}`
	resp := performJSON(t, h.UpdateAppSettings, http.MethodPut, "", body, "")
	if resp.Code != http.StatusBadRequest {
		t.Fatalf("enabling turnstile without keys = %d: %s", resp.Code, resp.Body.String())
	}
}
