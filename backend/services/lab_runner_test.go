package services

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"tunnel-manager/auth"
	"tunnel-manager/models"
)

func newVerifiedLabRunner(t *testing.T) *LabIPSelectorRunner {
	t.Helper()
	state := newTestStore(t)
	app := state.GetAppSettings()
	app.ExperimentalFeatures = true
	if err := state.SetAppSettings(app); err != nil {
		t.Fatal(err)
	}
	settings := models.LabIPSelectorSettings{Host: "probe.example.com", SNI: "probe.example.com", Path: "/ip-check.txt", Statuses: "200", Timeout: 1, Workers: 2, Top: 1, IPTargets: "1.1.1.1", IntervalMins: 5, TTL: 300}
	resolver := labResolverFunc(func(ctx context.Context, name string) ([]string, error) {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		return []string{state.GetLabSecurity().Verification.Token}, nil
	})
	runner := NewLabIPSelectorRunner(state, nil, resolver)
	owner := state.AdminUserID()
	if err := runner.SaveSettings(settings, owner); err != nil {
		t.Fatal(err)
	}
	if _, err := runner.NewChallenge("example.com", owner); err != nil {
		t.Fatal(err)
	}
	if _, err := runner.Verify(context.Background(), owner); err != nil {
		t.Fatal(err)
	}
	return runner
}

func waitLabStopped(t *testing.T, runner *LabIPSelectorRunner) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for runner.Status().Running {
		if time.Now().After(deadline) {
			t.Fatal("scan did not stop")
		}
		time.Sleep(time.Millisecond)
	}
}

func TestLabOwnershipLifecycle(t *testing.T) {
	runner := newVerifiedLabRunner(t)
	owner := runner.store.AdminUserID()
	if _, err := runner.Verify(context.Background(), "other-admin"); err == nil {
		t.Fatal("accepted foreign owner")
	}
	settings := runner.store.GetLabSettings()
	settings.Schedule = true
	if err := runner.SaveSettings(settings, owner); err != nil {
		t.Fatal(err)
	}
	runner.resolver = labResolverFunc(func(context.Context, string) ([]string, error) { return nil, errors.New("DNS unavailable") })
	if _, err := runner.prepare(context.Background(), owner); err == nil {
		t.Fatal("ran after DNS failure")
	}
	if runner.store.GetLabSettings().Schedule || runner.Status().Verification.VerifiedAt != 0 || runner.Status().BudgetUsed != 0 {
		t.Fatal("DNS failure did not suspend authorization/schedule before reserving budget")
	}
	settings.Host, settings.SNI = "other.example.com", "other.example.com"
	if err := runner.SaveSettings(settings, owner); err != nil {
		t.Fatal(err)
	}
	if runner.Status().Verification.Token == "" || runner.Status().Verification.VerifiedAt != 0 || runner.store.GetLabSettings().Schedule {
		t.Fatal("Host change discarded pending TXT or restored failed verification")
	}
}

func TestLabChallengeBindingAndExpiry(t *testing.T) {
	runner := newVerifiedLabRunner(t)
	owner := runner.store.AdminUserID()
	initial := runner.Status().Verification.Token
	challenge, err := runner.NewChallenge("example.com", owner)
	if err != nil || challenge.Token == initial || challenge.VerifiedAt != 0 {
		t.Fatalf("challenge not rotated: %v", err)
	}
	if err := runner.store.UpdateLabSecurity(func(state *models.LabSecurityState) error {
		state.Verification.IssuedAt = time.Now().Add(-25 * time.Hour).Unix()
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := runner.Verify(context.Background(), owner); err == nil {
		t.Fatal("accepted expired challenge")
	}
	if err := runner.store.UpdateLabSecurity(func(state *models.LabSecurityState) error {
		state.Verification.IssuedAt = time.Now().Unix()
		state.InstanceID = "different-instance"
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := runner.Verify(context.Background(), owner); err == nil {
		t.Fatal("accepted another instance's token")
	}
}

func TestLabBudgetCooldownAndRollover(t *testing.T) {
	runner := newVerifiedLabRunner(t)
	owner := runner.store.AdminUserID()
	if _, err := runner.prepare(context.Background(), owner); err != nil {
		t.Fatal(err)
	}
	if runner.Status().BudgetUsed != 1 {
		t.Fatal("missing budget reservation")
	}
	if _, err := runner.prepare(context.Background(), owner); err == nil {
		t.Fatal("bypassed cooldown")
	}
	if err := runner.store.UpdateLabSecurity(func(state *models.LabSecurityState) error {
		state.LastStartedAt = 0
		state.BudgetUsed = models.LabDailyRequestLimit
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := runner.prepare(context.Background(), owner); err == nil {
		t.Fatal("exceeded daily limit")
	}
	if err := runner.store.UpdateLabSecurity(func(state *models.LabSecurityState) error {
		state.BudgetDay = time.Now().UTC().Add(-24 * time.Hour).Format("2006-01-02")
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := runner.prepare(context.Background(), owner); err != nil {
		t.Fatal(err)
	}
	if runner.Status().BudgetUsed != 1 {
		t.Fatal("daily rollover failed")
	}
}

func TestLabDisabledScheduleCannotStartFromStaleTick(t *testing.T) {
	runner := newVerifiedLabRunner(t)
	if err := runner.trigger(context.Background(), runner.store.AdminUserID(), true); err == nil {
		t.Fatal("stale scheduler tick started a disabled task")
	}
	if runner.Status().Running || runner.Status().BudgetUsed != 0 {
		t.Fatal("disabled scheduler consumed resources")
	}
}

func TestLabScheduledRunUsesTheVerifiedRunner(t *testing.T) {
	runner := newVerifiedLabRunner(t)
	settings := runner.store.GetLabSettings()
	settings.Schedule = true
	if err := runner.SaveSettings(settings, runner.store.AdminUserID()); err != nil {
		t.Fatal(err)
	}
	started := make(chan struct{}, 1)
	runner.scan = func(context.Context, models.LabIPSelectorSettings, ...func(LabIPSelectorProgress)) ([]models.LabIPScanResult, []models.LabIPScanResult, []string, []models.LabIPSegmentResult, error) {
		started <- struct{}{}
		matches := []models.LabIPScanResult{{IP: "1.1.1.1", Status: 200}}
		return matches, matches, []string{"1.1.1.1"}, nil, nil
	}
	runner.scheduledInterval = 5 * time.Minute
	runner.nextRun = time.Now().Add(-time.Minute)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan struct{})
	go func() { runner.Start(ctx); close(done) }()
	defer func() { cancel(); <-done; _ = runner.Stop(); waitLabStopped(t, runner) }()
	select {
	case <-started:
	case <-time.After(3 * time.Second):
		t.Fatal("scheduler did not execute")
	}
	waitLabStopped(t, runner)
	if !runner.Status().LastRun.Success || runner.Status().BudgetUsed != 1 || len(runner.store.GetLabRuns(20)) != 1 {
		t.Fatal("scheduled run did not persist its result and budget")
	}
	if err := runner.Stop(); err != nil {
		t.Fatal(err)
	}
	if runner.store.GetLabSettings().Schedule {
		t.Fatal("stop retained schedule")
	}
}

func TestLabRevokedAdministratorCannotRun(t *testing.T) {
	runner := newVerifiedLabRunner(t)
	if err := runner.store.CreateUser(models.User{ID: "backup-admin", Username: "backup", Role: models.RoleAdmin, Status: models.UserActive}); err != nil {
		t.Fatal(err)
	}
	owner := runner.store.AdminUserID()
	if err := runner.store.SetUserStatus(owner, models.UserDisabled); err != nil {
		t.Fatal(err)
	}
	if err := runner.Trigger(context.Background(), owner); err == nil || runner.Status().BudgetUsed != 0 {
		t.Fatal("revoked administrator started scanning")
	}
}

func TestLabRunningTasksCancelAndRemainExclusive(t *testing.T) {
	for _, operation := range []string{"stop", "disable", "settings", "owner"} {
		t.Run(operation, func(t *testing.T) {
			runner := newVerifiedLabRunner(t)
			owner := runner.store.AdminUserID()
			started := make(chan struct{})
			var calls atomic.Int32
			runner.scan = func(ctx context.Context, settings models.LabIPSelectorSettings, callbacks ...func(LabIPSelectorProgress)) ([]models.LabIPScanResult, []models.LabIPScanResult, []string, []models.LabIPSegmentResult, error) {
				calls.Add(1)
				close(started)
				<-ctx.Done()
				return nil, nil, nil, nil, ctx.Err()
			}
			t.Cleanup(func() { _ = runner.Stop(); waitLabStopped(t, runner) })
			if err := runner.Trigger(context.Background(), owner); err != nil {
				t.Fatal(err)
			}
			<-started
			if err := runner.Trigger(context.Background(), owner); !errors.Is(err, ErrLabRunning) {
				t.Fatalf("concurrent scan: %v", err)
			}
			switch operation {
			case "stop":
				if err := runner.Stop(); err != nil {
					t.Fatal(err)
				}
			case "disable":
				app := runner.store.GetAppSettings()
				app.ExperimentalFeatures = false
				if err := runner.store.SetAppSettings(app); err != nil {
					t.Fatal(err)
				}
			case "settings":
				settings := runner.store.GetLabSettings()
				settings.Path = "/other"
				if err := runner.SaveSettings(settings, owner); err != nil {
					t.Fatal(err)
				}
			case "owner":
				if err := runner.store.CreateUser(models.User{ID: "backup-admin", Username: "backup", Role: models.RoleAdmin, Status: models.UserActive}); err != nil {
					t.Fatal(err)
				}
				if err := runner.store.SetUserStatus(owner, models.UserDisabled); err != nil {
					t.Fatal(err)
				}
			}
			waitLabStopped(t, runner)
			if calls.Load() != 1 || runner.Status().LastRun.Success || runner.Status().LastRun.DNSUpdated {
				t.Fatal("cancelled scan succeeded")
			}
		})
	}
}

func TestLabFailureNeverUpdatesDNS(t *testing.T) {
	for _, failure := range []string{"empty", "scan", "cancel", "authorization", "credentials", "api", "configuration"} {
		t.Run(failure, func(t *testing.T) {
			runner := newVerifiedLabRunner(t)
			var requests, writes atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
				requests.Add(1)
				if request.Method != http.MethodGet {
					writes.Add(1)
				}
				http.Error(response, "DNS unavailable", http.StatusServiceUnavailable)
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
			if failure == "credentials" {
				settings.SecretKey = "invalid-encrypted-key"
			}
			if err := runner.store.SetLabSettings(settings); err != nil {
				t.Fatal(err)
			}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			runner.scan = func(context.Context, models.LabIPSelectorSettings, ...func(LabIPSelectorProgress)) ([]models.LabIPScanResult, []models.LabIPScanResult, []string, []models.LabIPSegmentResult, error) {
				if failure == "scan" {
					return nil, nil, nil, nil, errors.New("scan failed")
				}
				if failure == "empty" {
					return nil, nil, nil, nil, nil
				}
				if failure == "cancel" {
					cancel()
				}
				if failure == "authorization" {
					runner.resolver = labResolverFunc(func(context.Context, string) ([]string, error) { return nil, nil })
				}
				if failure == "configuration" {
					updated := settings
					updated.Path = "/new-probe.txt"
					if err := runner.store.SetLabSettings(updated); err != nil {
						t.Fatal(err)
					}
				}
				matches := []models.LabIPScanResult{{IP: "1.1.1.1", Status: 200}}
				return matches, matches, []string{"1.1.1.1"}, nil, nil
			}
			run, err := runner.run(ctx, settings, runner.store.AdminUserID())
			if err == nil || run.Success || run.DNSUpdated {
				t.Fatalf("unsafe completion: %#v %v", run, err)
			}
			if writes.Load() != 0 || (failure != "api" && requests.Load() != 0) {
				t.Fatalf("failed run accessed DNS: %d requests, %d writes", requests.Load(), writes.Load())
			}
		})
	}
}

func TestLabRunDeadlineAndCancellation(t *testing.T) {
	runner := newVerifiedLabRunner(t)
	started := time.Now()
	ctx := runner.beginRun(context.Background())
	defer runner.cancelRun()
	deadline, found := ctx.Deadline()
	midnight := started.UTC().Truncate(24 * time.Hour).Add(24 * time.Hour)
	if !found || deadline.After(started.Add(2*time.Hour+time.Second)) || deadline.After(midnight) {
		t.Fatalf("run can exceed runtime/day boundary: %v", deadline)
	}
	runner.cancelRun()
	if !errors.Is(ctx.Err(), context.Canceled) {
		t.Fatal("run context did not cancel")
	}
}

func TestLabAuthorizationChangesCancelDNSBeforeWaitingForControl(t *testing.T) {
	for _, operation := range []string{"settings", "challenge", "verification"} {
		t.Run(operation, func(t *testing.T) {
			runner := newVerifiedLabRunner(t)
			owner := runner.store.AdminUserID()
			var txtAvailable atomic.Bool
			txtAvailable.Store(true)
			runner.resolver = labResolverFunc(func(context.Context, string) ([]string, error) {
				if txtAvailable.Load() {
					return []string{runner.store.GetLabSecurity().Verification.Token}, nil
				}
				return nil, nil
			})
			lookupStarted := make(chan struct{})
			lookupCancelled := make(chan struct{})
			releaseLookup := make(chan struct{})
			var writes atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
				if request.Method == http.MethodGet {
					close(lookupStarted)
					select {
					case <-request.Context().Done():
						close(lookupCancelled)
					case <-releaseLookup:
					}
					fmt.Fprint(response, `{"recordsets":[]}`)
					return
				}
				writes.Add(1)
				fmt.Fprint(response, `{}`)
			}))
			t.Cleanup(func() {
				close(releaseLookup)
				_ = runner.Stop()
				waitLabStopped(t, runner)
				server.Close()
			})
			runner.encryptionKey = make([]byte, 32)
			secret, err := auth.EncryptSecret(runner.encryptionKey, labHuaweiSecretPurpose, []byte("test-secret"))
			if err != nil {
				t.Fatal(err)
			}
			settings := runner.store.GetLabSettings()
			settings.UpdateDNS, settings.Endpoint = true, server.URL
			settings.AccessKey, settings.SecretKey = "test-access", secret
			settings.ZoneID, settings.Record = "test-zone", "edge.example.com"
			if err := runner.SaveSettings(settings, owner); err != nil {
				t.Fatal(err)
			}
			runner.scan = func(context.Context, models.LabIPSelectorSettings, ...func(LabIPSelectorProgress)) ([]models.LabIPScanResult, []models.LabIPScanResult, []string, []models.LabIPSegmentResult, error) {
				matches := []models.LabIPScanResult{{IP: "1.1.1.1", Status: 200}}
				return matches, matches, []string{"1.1.1.1"}, nil, nil
			}
			if err := runner.Trigger(context.Background(), owner); err != nil {
				t.Fatal(err)
			}
			select {
			case <-lookupStarted:
			case <-time.After(3 * time.Second):
				t.Fatal("DNS lookup did not start")
			}
			mutationDone := make(chan error, 1)
			go func() {
				if operation == "verification" {
					txtAvailable.Store(false)
					_, err := runner.Verify(context.Background(), owner)
					if err == nil {
						mutationDone <- errors.New("removed TXT record was accepted")
						return
					}
					mutationDone <- nil
					return
				}
				if operation == "challenge" {
					_, err := runner.NewChallenge("example.com", owner)
					mutationDone <- err
					return
				}
				settings.Path = "/new-probe.txt"
				mutationDone <- runner.SaveSettings(settings, owner)
			}()
			select {
			case <-lookupCancelled:
			case <-time.After(time.Second):
				t.Fatal("configuration mutation waited for DNS instead of cancelling it")
			}
			if err := <-mutationDone; err != nil {
				t.Fatal(err)
			}
			waitLabStopped(t, runner)
			if writes.Load() != 0 || runner.Status().LastRun.DNSUpdated || runner.Status().LastRun.Success {
				t.Fatal("configuration mutation allowed a DNS write")
			}
		})
	}
}
