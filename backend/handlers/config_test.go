package handlers

import (
	"net/http"
	"testing"

	"tunnel-manager/models"
)

func TestGetSiteSettingsIsPublicSafeResponse(t *testing.T) {
	st := newTestStore(t)
	if err := st.SetSiteSettings("My Panel", "Operations", "/icon.png", false); err != nil {
		t.Fatal(err)
	}
	h := NewConfigHandler(st)
	resp := performJSON(t, h.GetSiteSettings, http.MethodGet, "/api/site", "", "")
	if resp.Code != http.StatusOK {
		t.Fatalf("GetSiteSettings() code = %d: %s", resp.Code, resp.Body.String())
	}
	var result map[string]interface{}
	decodeResponse(t, resp, &result)
	if len(result) != 4 || result["name"] != "My Panel" || result["description"] != "Operations" || result["icon"] != "/icon.png" || result["landing_enabled"] != false {
		t.Fatalf("GetSiteSettings() response = %#v", result)
	}
}

func TestConfigurationResponsesOmitRetiredFeatureFlag(t *testing.T) {
	st := newTestStore(t)
	configHandler := NewConfigHandler(st)
	managementHandler := NewManagementHandler(st, nil)
	for name, handler := range map[string]http.HandlerFunc{
		"config": configHandler.GetConfig,
		"site":   configHandler.GetSiteSettings,
		"admin":  managementHandler.GetAppSettings,
	} {
		t.Run(name, func(t *testing.T) {
			response := performJSON(t, handler, http.MethodGet, "", "", "")
			if response.Code != http.StatusOK {
				t.Fatalf("configuration response = %d: %s", response.Code, response.Body.String())
			}
			var result map[string]interface{}
			decodeResponse(t, response, &result)
			if _, found := result["experimental_features_enabled"]; found {
				t.Fatal("configuration response exposes the retired feature flag")
			}
		})
	}
}

func TestNormalizeCNAMEPresets(t *testing.T) {
	items, err := normalizeCNAMEPresets([]models.CNAMEPreset{})
	if err == nil || items != nil {
		t.Fatal("normalizeCNAMEPresets accepted an empty list")
	}
}
