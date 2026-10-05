package handlers

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"tunnel-manager/models"
	"tunnel-manager/services"
)

func TestLabWritesRequireRealAdministrator(t *testing.T) {
	state := newTestStore(t)
	settings := state.GetAppSettings()
	settings.ExperimentalFeatures = true
	if err := state.SetAppSettings(settings); err != nil {
		t.Fatal(err)
	}
	handler := NewLabHandler(state, services.NewLabIPSelectorRunner(state, nil), nil)
	for name, action := range map[string]http.HandlerFunc{"save": handler.SaveSettings, "run": handler.Run, "stop": handler.Stop, "challenge": handler.Challenge, "verify": handler.Verify, "dns": handler.ProvisionDNS, "dns_availability": handler.DNSAvailability} {
		for _, user := range []models.SessionUser{{}, {Role: models.RoleAdmin}, {ID: state.AdminUserID(), Role: models.RoleUser}} {
			request := withUser(httptest.NewRequest(http.MethodPost, "/api/lab/ip-selector", strings.NewReader(`{}`)), user)
			response := httptest.NewRecorder()
			action(response, request)
			if response.Code != http.StatusForbidden {
				t.Fatalf("%s accepted invalid session: %d", name, response.Code)
			}
		}
	}
	settings.ExperimentalFeatures = false
	if err := state.SetAppSettings(settings); err != nil {
		t.Fatal(err)
	}
	for name, action := range map[string]http.HandlerFunc{"settings": handler.GetSettings, "status": handler.GetStatus, "save": handler.SaveSettings, "run": handler.Run, "stop": handler.Stop, "challenge": handler.Challenge, "verify": handler.Verify, "dns": handler.ProvisionDNS, "dns_availability": handler.DNSAvailability} {
		response := performJSON(t, action, http.MethodPost, "/api/lab/ip-selector", `{}`, "")
		if response.Code != http.StatusNotFound {
			t.Fatalf("disabled %s: %d", name, response.Code)
		}
	}
}

func TestLabDNSWithoutBoundAccountRetainsManualFlow(t *testing.T) {
	state := newTestStore(t)
	settings := state.GetAppSettings()
	settings.ExperimentalFeatures = true
	if err := state.SetAppSettings(settings); err != nil {
		t.Fatal(err)
	}
	runner := services.NewLabIPSelectorRunner(state, nil)
	handler := NewLabHandler(state, runner, nil)
	owner := models.SessionUser{ID: state.AdminUserID(), Role: models.RoleAdmin}
	request := withUser(httptest.NewRequest(http.MethodGet, "/api/lab/ip-selector/ownership/dns", nil), owner)
	response := httptest.NewRecorder()
	handler.DNSAvailability(response, request)
	var availability models.LabDNSAvailability
	decodeResponse(t, response, &availability)
	if response.Code != http.StatusOK || availability.Available || availability.Message == "" {
		t.Fatalf("unexpected manual fallback: %s", response.Body.String())
	}
	request = withUser(httptest.NewRequest(http.MethodPost, "/api/lab/ip-selector/ownership/dns", strings.NewReader(`{"domain":"example.com"}`)), owner)
	response = httptest.NewRecorder()
	handler.ProvisionDNS(response, request)
	if response.Code != http.StatusBadRequest || state.GetLabSecurity().Verification.Token != "" {
		t.Fatal("unbound request created an authorization")
	}
}

func TestLabSettingsEncryptSecretAndPreserveIndependentVerification(t *testing.T) {
	state := newTestStore(t)
	settings := state.GetAppSettings()
	settings.ExperimentalFeatures = true
	if err := state.SetAppSettings(settings); err != nil {
		t.Fatal(err)
	}
	key := bytes.Repeat([]byte{42}, 32)
	runner := services.NewLabIPSelectorRunner(state, key)
	handler := NewLabHandler(state, runner, key)
	owner := models.SessionUser{ID: state.AdminUserID(), Role: models.RoleAdmin}
	save := func(body string) *httptest.ResponseRecorder {
		request := withUser(httptest.NewRequest(http.MethodPut, "/api/lab/ip-selector", strings.NewReader(body)), owner)
		response := httptest.NewRecorder()
		handler.SaveSettings(response, request)
		return response
	}
	response := save(`{"host":"PROBE.example.com.","ip_targets":"1.1.1.1","secret_key":"private-huawei-secret","schedule":true}`)
	if response.Code != 200 {
		t.Fatal(response.Body.String())
	}
	if strings.Contains(response.Body.String(), "private-huawei-secret") || strings.Contains(response.Body.String(), `"secret_key"`) {
		t.Fatal("secret leaked in response")
	}
	stored := state.GetLabSettings()
	if stored.Host != "probe.example.com" || stored.Schedule || stored.SecretKey == "" || stored.SecretKey == "private-huawei-secret" {
		t.Fatal("unsafe saved settings")
	}
	response = save(`{"host":"probe.example.com","ip_targets":"1.1.1.1","endpoint":"https://other.example.com"}`)
	if response.Code != 400 || state.GetLabSettings() != stored {
		t.Fatal("endpoint switch reused existing credentials")
	}
	if _, err := runner.NewChallenge("example.com", owner.ID); err != nil {
		t.Fatal(err)
	}
	challenge := state.GetLabSecurity().Verification
	response = save(`{"host":"changed.example.com","ip_targets":"1.1.1.1"}`)
	if response.Code != 200 || state.GetLabSecurity().Verification != challenge || state.GetLabSettings().Schedule {
		t.Fatal("Host change discarded independent TXT or enabled an unverified schedule")
	}
}

func TestLabAPIHiddenWhileExperimentalFeaturesDisabled(t *testing.T) {
	st := newTestStore(t)
	h := NewLabHandler(st, services.NewLabIPSelectorRunner(st, nil), nil)

	resp := performJSON(t, h.GetSettings, http.MethodGet, "/api/lab/ip-selector", "", "")
	if resp.Code != http.StatusNotFound {
		t.Fatalf("GetSettings() code = %d: %s", resp.Code, resp.Body.String())
	}
	resp = performJSON(t, h.Run, http.MethodPost, "/api/lab/ip-selector/run", "", "")
	if resp.Code != http.StatusNotFound {
		t.Fatalf("Run() code = %d: %s", resp.Code, resp.Body.String())
	}
}

func TestLabSettingsSaveRejectsInvalidTargets(t *testing.T) {
	st := newTestStore(t)
	settings := st.GetAppSettings()
	settings.ExperimentalFeatures = true
	if err := st.SetAppSettings(settings); err != nil {
		t.Fatal(err)
	}
	h := NewLabHandler(st, services.NewLabIPSelectorRunner(st, nil), nil)
	body := `{"host":"cdn.example.com","path":"/healthz","statuses":"200","timeout":2,"workers":4,"top":3,"ip_targets":"example.com"}`
	req := withUser(httptest.NewRequest(http.MethodPut, "/api/lab/ip-selector", strings.NewReader(body)), models.SessionUser{ID: st.AdminUserID(), Role: models.RoleAdmin})
	resp := httptest.NewRecorder()
	h.SaveSettings(resp, req)
	if resp.Code != http.StatusBadRequest {
		t.Fatalf("SaveSettings() code = %d: %s", resp.Code, resp.Body.String())
	}
	var result map[string]interface{}
	decodeResponse(t, resp, &result)
	if result["error"] == "" {
		t.Fatalf("SaveSettings() response missing error: %#v", result)
	}
}
