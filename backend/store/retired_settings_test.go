package store

import (
	"encoding/json"
	"testing"

	"tunnel-manager/db"
)

func TestStoreIgnoresRetiredIPSelectorSettings(t *testing.T) {
	st := testStore(t)
	handle, err := db.Open(st.dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer handle.Close()

	legacyRecords := map[string]string{
		"lab_ip_selector":      `{"schedule":true,"update_dns":true,"secret_key":"legacy-encrypted-secret"}`,
		"lab_ip_selector_runs": `[{"id":"legacy-run","selected":["192.0.2.1"]}]`,
	}
	for key := range legacyRecords {
		if _, found, err := loadSetting(handle, key); err != nil || found {
			t.Fatalf("fresh store contains retired setting %q: found=%v, err=%v", key, found, err)
		}
	}

	transaction, err := handle.Begin()
	if err != nil {
		t.Fatal(err)
	}
	defer transaction.Rollback()
	if err := upsertSetting(transaction, "app", `{"registration_enabled":true,"invite_mode":"optional","experimental_features_enabled":true}`); err != nil {
		t.Fatal(err)
	}
	for key, value := range legacyRecords {
		if err := upsertSetting(transaction, key, value); err != nil {
			t.Fatal(err)
		}
	}
	if err := transaction.Commit(); err != nil {
		t.Fatal(err)
	}

	reloaded := NewStore(st.dsn)
	settings := reloaded.GetAppSettings()
	if !reloaded.Installed() || !settings.RegistrationEnabled || settings.InviteMode != "optional" {
		t.Fatalf("legacy settings disrupted existing configuration: %#v", settings)
	}
	settings.RegistrationEnabled = false
	if err := reloaded.SetAppSettings(settings); err != nil {
		t.Fatal(err)
	}

	storedApp, found, err := loadSetting(handle, "app")
	if err != nil || !found {
		t.Fatalf("load app settings: found=%v, err=%v", found, err)
	}
	var persisted map[string]interface{}
	if err := json.Unmarshal([]byte(storedApp), &persisted); err != nil {
		t.Fatal(err)
	}
	if _, found := persisted["experimental_features_enabled"]; found {
		t.Fatal("retired feature flag is still persisted")
	}
	if persisted["registration_enabled"] != false || persisted["invite_mode"] != "optional" {
		t.Fatalf("application settings did not persist correctly: %#v", persisted)
	}
	for key, expected := range legacyRecords {
		actual, found, err := loadSetting(handle, key)
		if err != nil || !found || actual != expected {
			t.Fatalf("retired record %q was modified: value=%q, found=%v, err=%v", key, actual, found, err)
		}
	}
}
