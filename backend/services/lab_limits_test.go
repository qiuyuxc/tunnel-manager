package services

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"tunnel-manager/auth"
	"tunnel-manager/models"
)

func TestLabLimitChangesDoNotInvalidateActiveDNSUpdate(t *testing.T) {
	runner := newVerifiedLabRunner(t)
	var writes atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if request.Method == http.MethodGet {
			fmt.Fprint(response, `{"recordsets":[]}`)
			return
		}
		writes.Add(1)
		fmt.Fprint(response, `{}`)
	}))
	defer server.Close()
	runner.encryptionKey = make([]byte, 32)
	secret, err := auth.EncryptSecret(runner.encryptionKey, labHuaweiSecretPurpose, []byte("test-secret"))
	if err != nil {
		t.Fatal(err)
	}
	settings := runner.store.GetLabSettings()
	settings.UpdateDNS = true
	settings.Endpoint, settings.AccessKey, settings.SecretKey = server.URL, "test-access", secret
	settings.ZoneID, settings.Record = "test-zone", "edge.example.com"
	if err := runner.store.SetLabSettings(settings); err != nil {
		t.Fatal(err)
	}
	prepared, err := runner.prepare(context.Background(), runner.store.AdminUserID())
	if err != nil {
		t.Fatal(err)
	}
	runner.scan = func(ctx context.Context, current models.LabIPSelectorSettings, callbacks ...func(LabIPSelectorProgress)) ([]models.LabIPScanResult, []models.LabIPScanResult, []string, []models.LabIPSegmentResult, error) {
		app := runner.store.GetAppSettings()
		app.LabMaxWorkers, app.LabRequestsPerSecond = 1, 1
		if err := runner.store.SetAppSettings(app); err != nil {
			t.Fatal(err)
		}
		if current.Limits != prepared.Limits {
			t.Fatal("active run lost its limit snapshot")
		}
		matches := []models.LabIPScanResult{{IP: "1.1.1.1", Status: 200}}
		return matches, matches, []string{"1.1.1.1"}, nil, nil
	}
	run, err := runner.run(context.Background(), prepared, runner.store.AdminUserID())
	if err != nil || !run.DNSUpdated || writes.Load() != 1 {
		t.Fatalf("limit snapshot blocked valid DNS update: %+v, %v", run, err)
	}
}

func TestLabConfiguredBudgetAndPerRunSnapshot(t *testing.T) {
	runner := newVerifiedLabRunner(t)
	owner := runner.store.AdminUserID()
	app := runner.store.GetAppSettings()
	app.LabDailyRequestLimit, app.LabRequestsPerSecond, app.LabMaxWorkers = 200000, 40, 64
	if err := runner.store.SetAppSettings(app); err != nil {
		t.Fatal(err)
	}
	if err := runner.store.UpdateLabSecurity(func(state *models.LabSecurityState) error {
		state.BudgetDay = time.Now().UTC().Format("2006-01-02")
		state.BudgetUsed = 100000
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	prepared, err := runner.prepare(context.Background(), owner)
	if err != nil || prepared.Limits != app.LabLimits() || runner.Status().BudgetUsed != 100001 {
		t.Fatalf("configured budget/snapshot not used: %+v, %v", prepared.Limits, err)
	}
	app.LabDailyRequestLimit, app.LabRequestsPerSecond, app.LabMaxWorkers = 1, 2, 1
	if err := runner.store.SetAppSettings(app); err != nil {
		t.Fatal(err)
	}
	if err := runner.store.UpdateLabSecurity(func(state *models.LabSecurityState) error {
		state.LastStartedAt = 0
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := runner.prepare(context.Background(), owner); err == nil {
		t.Fatal("lowering limit reset spent budget")
	}
	status := runner.Status()
	if status.BudgetUsed != 100001 || status.BudgetLimit != 1 || status.RequestsPerSecond != 2 || status.MaxWorkers != 1 || prepared.Limits.MaxWorkers != 64 {
		t.Fatalf("unexpected budget/status/snapshot: %+v", status)
	}
	if err := runner.store.UpdateLabSecurity(func(state *models.LabSecurityState) error {
		state.BudgetDay = time.Now().UTC().Add(-24 * time.Hour).Format("2006-01-02")
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	next, err := runner.prepare(context.Background(), owner)
	if err != nil || next.Limits != app.LabLimits() || runner.Status().BudgetUsed != 1 {
		t.Fatalf("next day/run did not use new limits: %+v, %v", next.Limits, err)
	}
}

func TestLabScannerConfiguredWorkerCeiling(t *testing.T) {
	for _, maximum := range []int{2, 40} {
		t.Run(fmt.Sprint(maximum), func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()
			entered := make(chan struct{}, 64)
			release := make(chan struct{})
			defer close(release)
			settings := models.LabIPSelectorSettings{Host: "probe.example.com", Workers: 64, Top: 1, IPTargets: "1.1.1.1-1.1.1.64", Limits: models.LabLimits{MaxWorkers: maximum}}
			finished := make(chan error, 1)
			go func() {
				_, _, _, _, err := scanLabIPs(ctx, settings, func(ctx context.Context, address, request, sni string, timeout time.Duration, gate <-chan time.Time) models.LabIPScanResult {
					entered <- struct{}{}
					select {
					case <-ctx.Done():
					case <-release:
					}
					return models.LabIPScanResult{IP: address, Status: 200}
				})
				finished <- err
			}()
			for count := 0; count < maximum; count++ {
				select {
				case <-entered:
				case <-ctx.Done():
					t.Fatal("scanner did not use configured worker count")
				}
			}
			select {
			case <-entered:
				t.Fatal("scanner exceeded configured worker ceiling")
			case <-time.After(25 * time.Millisecond):
			}
			cancel()
			<-finished
		})
	}
}

func TestLabScannerConfiguredSharedRequestRate(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	ticks := make(chan time.Time, 2)
	settings := models.LabIPSelectorSettings{Host: "probe.example.com", Workers: 2, Top: 1, IPTargets: "1.1.1.1-1.1.1.2", Limits: models.LabLimits{RequestsPerSecond: 4}}
	_, _, _, _, err := scanLabIPs(ctx, settings, func(ctx context.Context, address, request, sni string, timeout time.Duration, gate <-chan time.Time) models.LabIPScanResult {
		select {
		case tick := <-gate:
			ticks <- tick
		case <-ctx.Done():
		}
		return models.LabIPScanResult{IP: address, Status: 200}
	})
	if err != nil || len(ticks) != 2 {
		t.Fatalf("missing paced requests: %v", err)
	}
	first, second := <-ticks, <-ticks
	if second.Sub(first) < 200*time.Millisecond {
		t.Fatal("scanner ignored configured shared request rate")
	}
}
