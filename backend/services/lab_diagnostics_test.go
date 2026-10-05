package services

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"tunnel-manager/models"
)

func TestLabRunRetainsRejectedDiagnostics(t *testing.T) {
	for _, count := range []int{2, 105} {
		runner := newVerifiedLabRunner(t)
		settings := runner.store.GetLabSettings()
		runner.scan = func(context.Context, models.LabIPSelectorSettings, ...func(LabIPSelectorProgress)) ([]models.LabIPScanResult, []models.LabIPScanResult, []string, []models.LabIPSegmentResult, error) {
			results := make([]models.LabIPScanResult, count)
			for index := range results {
				results[index] = models.LabIPScanResult{IP: "43.175.131.30", Status: 403, LatencyMS: 12}
			}
			results[1] = models.LabIPScanResult{IP: "192.0.2.2", Error: "TLS handshake timeout"}
			return results, nil, nil, nil, errors.New("no IP returned an accepted status")
		}
		run, err := runner.run(context.Background(), settings, runner.store.AdminUserID())
		if err == nil || run.Success || run.DNSUpdated || run.Matched != 0 || len(run.Results) != 0 {
			t.Fatalf("diagnostics changed rejection or DNS behavior: %+v %v", run, err)
		}
		encoded, err := json.Marshal(runner.store.GetLabRuns(1)[0])
		if err != nil {
			t.Fatal(err)
		}
		var saved map[string]any
		if err := json.Unmarshal(encoded, &saved); err != nil {
			t.Fatal(err)
		}
		diagnostics, found := saved["diagnostics"].([]any)
		if !found || len(diagnostics) != min(count, 100) || saved["rejected"] != float64(count) {
			t.Fatalf("lost or unbounded rejection diagnostics: %s", encoded)
		}
		first := diagnostics[0].(map[string]any)
		if first["status"] != float64(403) || diagnostics[1].(map[string]any)["error"] != "TLS handshake timeout" {
			t.Fatalf("lost failure evidence: %v", diagnostics)
		}
		probe := saved["probe"].(map[string]any)
		if probe["host"] != settings.Host || probe["statuses"] != "200" || probe["path"] != settings.Path {
			t.Fatalf("missing historical probe context: %v", probe)
		}
	}
}

func TestLabDiagnosticsKeepAcceptedIPsSelected(t *testing.T) {
	runner := newVerifiedLabRunner(t)
	settings := runner.store.GetLabSettings()
	runner.scan = func(context.Context, models.LabIPSelectorSettings, ...func(LabIPSelectorProgress)) ([]models.LabIPScanResult, []models.LabIPScanResult, []string, []models.LabIPSegmentResult, error) {
		accepted := models.LabIPScanResult{IP: "1.1.1.1", Status: 200, LatencyMS: 10}
		rejected := models.LabIPScanResult{IP: "43.175.131.30", Status: 403, LatencyMS: 5}
		return []models.LabIPScanResult{accepted, rejected}, []models.LabIPScanResult{accepted}, []string{accepted.IP}, nil, nil
	}
	run, err := runner.run(context.Background(), settings, runner.store.AdminUserID())
	if err != nil || !run.Success || run.Matched != 1 || run.Rejected != 1 || len(run.Results) != 1 || len(run.Diagnostics) != 1 || run.Results[0].IP != "1.1.1.1" || run.Diagnostics[0].Status != 403 {
		t.Fatalf("diagnostics changed selection or lost evidence: %+v %v", run, err)
	}
}
