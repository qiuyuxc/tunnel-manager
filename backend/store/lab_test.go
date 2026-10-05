package store

import (
	"errors"
	"testing"

	"tunnel-manager/models"
)

func TestLabStatePersistsAndRollsBack(t *testing.T) {
	state := testStore(t)
	settings := models.LabIPSelectorSettings{Host: "probe.example.com", Schedule: true}
	if err := state.SetLabSettings(settings); err != nil {
		t.Fatal(err)
	}
	security := models.LabSecurityState{InstanceID: "instance", BudgetDay: "2026-10-03", BudgetUsed: 12, Verification: models.LabDomainVerification{VerifiedAt: 123}}
	if err := state.UpdateLabSecurity(func(value *models.LabSecurityState) error { *value = security; return nil }); err != nil {
		t.Fatal(err)
	}
	reloaded := NewStore(state.dsn)
	if reloaded.GetLabSecurity() != security || reloaded.GetLabSettings() != settings {
		t.Fatal("lab state lost on reload")
	}
	if err := state.UpdateLabSecurity(func(value *models.LabSecurityState) error { value.BudgetUsed++; return errors.New("reject") }); err == nil || state.GetLabSecurity() != security {
		t.Fatal("callback failure did not roll back")
	}
	breakStore(t, state)
	if err := state.UpdateLabSecurity(func(value *models.LabSecurityState) error { value.BudgetUsed++; return nil }); err == nil || state.GetLabSecurity() != security {
		t.Fatal("write failure did not roll back security")
	}
	if err := state.SuspendLab(); err == nil || state.GetLabSettings() != settings {
		t.Fatal("write failure did not roll back settings")
	}
	if err := state.SetLabSettings(models.LabIPSelectorSettings{Host: "changed.example.com"}); err == nil || state.GetLabSettings() != settings {
		t.Fatal("write failure did not roll back new settings")
	}
	if err := state.AppendLabRun(models.LabIPSelectorRun{ID: "failed-save"}); err == nil || len(state.GetLabRuns(20)) != 0 {
		t.Fatal("write failure did not roll back history")
	}
}

func TestLabLimitsAndExpandedWorkersSurviveRestart(t *testing.T) {
	state := testStore(t)
	app := state.GetAppSettings()
	app.LabDailyRequestLimit, app.LabRequestsPerSecond, app.LabMaxWorkers = 500000, 80, 64
	if err := state.SetAppSettings(app); err != nil {
		t.Fatal(err)
	}
	if err := state.SetLabSettings(models.LabIPSelectorSettings{Workers: 64}); err != nil {
		t.Fatal(err)
	}
	reloaded := NewStore(state.dsn)
	if reloaded.GetAppSettings().LabLimits() != app.LabLimits() || reloaded.GetLabSettings().Workers != 64 {
		t.Fatal("restart reset configured limits or reduced workers to old ceiling")
	}
}

func TestLabStopSurvivesRestartAndHistoryIsBounded(t *testing.T) {
	state := testStore(t)
	if err := state.UpdateLabSecurity(func(value *models.LabSecurityState) error {
		value.Verification.VerifiedAt = 123
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if err := state.SetLabSettings(models.LabIPSelectorSettings{Schedule: true}); err != nil {
		t.Fatal(err)
	}
	if err := state.SuspendLab(); err != nil {
		t.Fatal(err)
	}
	for index := 0; index < maxLabRuns+2; index++ {
		if err := state.AppendLabRun(models.LabIPSelectorRun{StartedAt: int64(index)}); err != nil {
			t.Fatal(err)
		}
	}
	reloaded := NewStore(state.dsn)
	if reloaded.GetLabSettings().Schedule {
		t.Fatal("restart re-enabled a stopped schedule")
	}
	runs := reloaded.GetLabRuns(0)
	if len(runs) != maxLabRuns || runs[0].StartedAt != maxLabRuns+1 || runs[maxLabRuns-1].StartedAt != 2 {
		t.Fatalf("incorrect retained history: %#v", runs)
	}
}
