package handlers

import (
	"fmt"
	"net/http"
	"testing"

	"tunnel-manager/models"
)

func TestAppSettingsLabLimitsRoundTripAndPartialSave(t *testing.T) {
	state := newTestStore(t)
	handler := NewManagementHandler(state, nil)
	for _, scenario := range []struct {
		body string
		want models.LabLimits
	}{
		{`{}`, models.LabLimits{DailyRequests: 100000, RequestsPerSecond: 10, MaxWorkers: 32}},
		{`{"lab_daily_request_limit":250000,"lab_requests_per_second":80,"lab_max_workers":64}`, models.LabLimits{DailyRequests: 250000, RequestsPerSecond: 80, MaxWorkers: 64}},
		{`{"rate_limit_per_account":9}`, models.LabLimits{DailyRequests: 250000, RequestsPerSecond: 80, MaxWorkers: 64}},
		{`{"lab_max_workers":8}`, models.LabLimits{DailyRequests: 250000, RequestsPerSecond: 80, MaxWorkers: 8}},
	} {
		response := performJSON(t, handler.UpdateAppSettings, http.MethodPut, "/api/admin/settings", scenario.body, "")
		if response.Code != http.StatusOK {
			t.Fatal(response.Body.String())
		}
		var view models.AppSettingsView
		decodeResponse(t, response, &view)
		got := models.LabLimits{DailyRequests: view.LabDailyRequestLimit, RequestsPerSecond: view.LabRequestsPerSecond, MaxWorkers: view.LabMaxWorkers}
		if got != scenario.want || state.GetAppSettings().LabLimits() != scenario.want {
			t.Fatalf("limits = %+v, want %+v", got, scenario.want)
		}
	}
}

func TestAppSettingsRejectInvalidLabLimitsAtomically(t *testing.T) {
	state := newTestStore(t)
	handler := NewManagementHandler(state, nil)
	before := state.GetAppSettings()
	for field, maximum := range map[string]int{
		"lab_daily_request_limit": models.LabDailyRequestCeiling,
		"lab_requests_per_second": models.LabRequestRateCeiling,
		"lab_max_workers":         models.LabWorkerCeiling,
	} {
		for _, value := range []string{"0", "-1", "1.5", `"10"`, fmt.Sprint(maximum + 1)} {
			body := fmt.Sprintf(`{"experimental_features_enabled":true,%q:%s}`, field, value)
			response := performJSON(t, handler.UpdateAppSettings, http.MethodPut, "", body, "")
			if response.Code != http.StatusBadRequest || state.GetAppSettings() != before {
				t.Fatalf("accepted or partially stored %s: %d", body, response.Code)
			}
		}
	}
}

func TestValidateLabSettingsUsesConfiguredWorkerLimit(t *testing.T) {
	settings := models.LabIPSelectorSettings{Host: "probe.example.com", Path: "/", Statuses: "200", IPTargets: "1.1.1.1", Timeout: 2, Workers: 64, Top: 1, TTL: 300}
	limits := models.LabLimits{MaxWorkers: 64}
	if err := validateLabSettings(settings, limits, false, false); err != nil {
		t.Fatal(err)
	}
	limits.MaxWorkers = 8
	if err := validateLabSettings(settings, limits, false, false); err == nil {
		t.Fatal("accepted workers above administrator limit")
	}
}
