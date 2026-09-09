package handlers

import (
	"net/http"
	"path/filepath"
	"testing"

	"tunnel-manager/services"
	"tunnel-manager/store"
)

func TestLabAPIHiddenWhileExperimentalFeaturesDisabled(t *testing.T) {
	st := store.NewStore(filepath.Join(t.TempDir(), "config.json"))
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
	st := store.NewStore(filepath.Join(t.TempDir(), "config.json"))
	settings := st.GetAppSettings()
	settings.ExperimentalFeatures = true
	if err := st.SetAppSettings(settings); err != nil {
		t.Fatal(err)
	}
	h := NewLabHandler(st, services.NewLabIPSelectorRunner(st, nil), nil)
	body := `{"host":"cdn.example.com","path":"/healthz","statuses":"200","timeout":2,"workers":4,"top":3,"ip_targets":"example.com"}`
	resp := performJSON(t, h.SaveSettings, http.MethodPut, "/api/lab/ip-selector", body, "")
	if resp.Code != http.StatusBadRequest {
		t.Fatalf("SaveSettings() code = %d: %s", resp.Code, resp.Body.String())
	}
	var result map[string]interface{}
	decodeResponse(t, resp, &result)
	if result["error"] == "" {
		t.Fatalf("SaveSettings() response missing error: %#v", result)
	}
}
