package handlers

import (
	"net/http"
	"path/filepath"
	"testing"

	"tunnel-manager/models"
	"tunnel-manager/store"
)

func TestGetSiteSettingsIsPublicSafeResponse(t *testing.T) {
	st := store.NewStore(filepath.Join(t.TempDir(), "config.json"))
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

func TestNormalizeCNAMEPresets(t *testing.T) {
	items, err := normalizeCNAMEPresets([]models.CNAMEPreset{})
	if err == nil || items != nil {
		t.Fatal("normalizeCNAMEPresets accepted an empty list")
	}
}
