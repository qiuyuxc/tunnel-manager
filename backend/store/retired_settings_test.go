package store

import (
	"encoding/json"
	"testing"

	"tunnel-manager/db"
)

func TestStoreRestoresLabSettingsWithoutUnverifiedSchedule(t *testing.T) {
	st := testStore(t)
	handle, err := db.Open(st.dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer handle.Close()

	legacyRecords := map[string]string{
		"lab_ip_selector":      `{"host":"probe.example.com","workers":256,"schedule":true,"update_dns":true,"secret_key":"legacy-encrypted-secret"}`,
		"lab_ip_selector_runs": `[{"id":"legacy-run","selected_ips":["1.1.1.1"]}]`,
	}
	if st.GetAppSettings().ExperimentalFeatures || st.GetLabSettings().Schedule {
		t.Fatal("fresh install enables experimental scans")
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
	lab := reloaded.GetLabSettings()
	if lab.Host != "probe.example.com" || lab.Schedule || !lab.UpdateDNS || lab.Workers != 32 || lab.SecretKey != "legacy-encrypted-secret" {
		t.Fatalf("unsafe or lossy lab migration: %#v", lab)
	}
	runs := reloaded.GetLabRuns(20)
	if len(runs) != 1 || runs[0].ID != "legacy-run" || len(runs[0].SelectedIPs) != 1 || runs[0].SelectedIPs[0] != "1.1.1.1" {
		t.Fatalf("lost legacy history: %#v", runs)
	}
	if reloaded.GetLabSecurity().Verification.VerifiedAt != 0 {
		t.Fatal("migration fabricated domain authorization")
	}
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
	if persisted["experimental_features_enabled"] != true {
		t.Fatal("existing experimental preference was lost")
	}
	if persisted["registration_enabled"] != false || persisted["invite_mode"] != "optional" {
		t.Fatalf("application settings did not persist correctly: %#v", persisted)
	}
	again := NewStore(st.dsn)
	if again.GetLabSettings() != lab || len(again.GetLabRuns(20)) != 1 {
		t.Fatal("restored lab settings/history did not survive a save and reload")
	}
}
