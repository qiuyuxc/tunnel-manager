package services

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sync"
	"time"

	"tunnel-manager/auth"
	"tunnel-manager/models"
	"tunnel-manager/store"
)

const labHuaweiSecretPurpose = "lab-huawei-secret-key"

// LabIPSelectorRunner schedules and executes experimental IP selector jobs.
type LabIPSelectorRunner struct {
	store             *store.Store
	encryptionKey     []byte
	mu                sync.Mutex
	running           bool
	nextRun           time.Time
	scheduledInterval time.Duration
	lastRun           *models.LabIPSelectorRun
	progress          LabIPSelectorProgress
	progressPhase     string
}

// LabIPSelectorStatus describes the current task state.
type LabIPSelectorStatus struct {
	Running  bool                     `json:"running"`
	NextRun  *time.Time               `json:"next_run,omitempty"`
	LastRun  *models.LabIPSelectorRun `json:"last_run,omitempty"`
	Progress LabIPSelectorProgress    `json:"progress"`
	Phase    string                   `json:"phase"`
}

// NewLabIPSelectorRunner creates the background runner.
func NewLabIPSelectorRunner(st *store.Store, encryptionKey []byte) *LabIPSelectorRunner {
	return &LabIPSelectorRunner{store: st, encryptionKey: append([]byte(nil), encryptionKey...)}
}

// Start runs the lightweight scheduler until the context is canceled.
func (r *LabIPSelectorRunner) Start(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			settings := r.store.GetLabSettings()
			interval := time.Duration(settings.IntervalMins) * time.Minute
			if !r.store.GetAppSettings().ExperimentalFeatures || !settings.Schedule || interval < 5*time.Minute {
				r.mu.Lock()
				r.nextRun = time.Time{}
				r.scheduledInterval = 0
				r.mu.Unlock()
				continue
			}
			r.mu.Lock()
			if r.scheduledInterval != interval {
				r.scheduledInterval = interval
				r.nextRun = time.Now().Add(interval)
				if runs := r.store.GetLabRuns(1); len(runs) > 0 {
					next := time.Unix(runs[0].FinishedAt, 0).Add(interval)
					if next.After(time.Now()) {
						r.nextRun = next
					}
				}
			}
			due := !r.running && !time.Now().Before(r.nextRun)
			r.mu.Unlock()
			if due {
				r.Trigger()
			}
		}
	}
}

// Trigger starts one asynchronous run. It returns false if another run is active.
func (r *LabIPSelectorRunner) Trigger() bool {
	if !r.beginRun() {
		return false
	}
	go func() {
		_, _ = r.run(context.Background())
	}()
	return true
}

func (r *LabIPSelectorRunner) beginRun() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.running {
		return false
	}
	r.running = true
	r.progress = LabIPSelectorProgress{}
	r.progressPhase = "preparing"
	return true
}

// Run executes one synchronous scan and optional DNS update.
func (r *LabIPSelectorRunner) Run(ctx context.Context) (models.LabIPSelectorRun, error) {
	if !r.beginRun() {
		return models.LabIPSelectorRun{}, fmt.Errorf("another IP selector run is active")
	}
	return r.run(ctx)
}

func (r *LabIPSelectorRunner) run(ctx context.Context) (models.LabIPSelectorRun, error) {
	defer func() {
		r.mu.Lock()
		r.running = false
		settings := r.store.GetLabSettings()
		if settings.Schedule && time.Duration(settings.IntervalMins)*time.Minute >= 5*time.Minute {
			r.scheduledInterval = time.Duration(settings.IntervalMins) * time.Minute
			r.nextRun = time.Now().Add(r.scheduledInterval)
		} else {
			r.nextRun = time.Time{}
		}
		r.mu.Unlock()
	}()

	run := models.LabIPSelectorRun{ID: newLabRunID(), StartedAt: time.Now().Unix()}
	var runErr error
	defer func() {
		run.FinishedAt = time.Now().Unix()
		if runErr != nil {
			run.Error = runErr.Error()
		}
		if appendErr := r.store.AppendLabRun(run); appendErr != nil && runErr == nil {
			run.Error = appendErr.Error()
		}
		r.mu.Lock()
		runCopy := run
		r.lastRun = &runCopy
		if runErr != nil {
			r.progressPhase = ""
		}
		r.mu.Unlock()
	}()

	if !r.store.GetAppSettings().ExperimentalFeatures {
		runErr = fmt.Errorf("experimental features are disabled")
		return run, runErr
	}
	settings := r.store.GetLabSettings()
	targets, err := ParseLabIPTargets(settings.IPTargets)
	if err != nil {
		runErr = err
		return run, runErr
	}
	r.mu.Lock()
	r.progress = LabIPSelectorProgress{Total: len(targets)}
	r.progressPhase = "scanning"
	r.mu.Unlock()

	allResults, matches, selected, segmentResults, err := ScanLabIPs(ctx, settings, func(progress LabIPSelectorProgress) {
		r.mu.Lock()
		r.progress = progress
		r.mu.Unlock()
	})
	run.Segments = segmentResults
	if err != nil {
		runErr = err
		return run, runErr
	}
	run.Scanned = len(allResults)
	run.Matched = len(matches)
	run.SelectedIPs = selected
	top := settings.Top
	if top < 1 {
		top = 10
	}
	if top > len(matches) {
		top = len(matches)
	}
	run.Results = matches[:top]
	if len(selected) == 0 {
		runErr = fmt.Errorf("no selected IPs")
		return run, runErr
	}
	if settings.UpdateDNS {
		r.mu.Lock()
		r.progressPhase = "updating_dns"
		r.mu.Unlock()
		if settings.AccessKey == "" || settings.SecretKey == "" {
			runErr = fmt.Errorf("Huawei Cloud access key and secret key are required")
			return run, runErr
		}
		secret, err := auth.DecryptSecret(r.encryptionKey, labHuaweiSecretPurpose, settings.SecretKey)
		if err != nil {
			runErr = fmt.Errorf("decrypt Huawei Cloud secret: %w", err)
			return run, runErr
		}
		client := newHuaweiDNSClient(settings.Endpoint, settings.AccessKey, string(secret))
		if err := client.UpdateARecord(settings.ZoneID, settings.Zone, settings.Record, settings.TTL, selected); err != nil {
			runErr = err
			return run, runErr
		}
		run.DNSUpdated = true
	}
	r.mu.Lock()
	r.progressPhase = "completed"
	r.mu.Unlock()
	run.Success = true
	return run, nil
}

// Status returns the current runner state.
func (r *LabIPSelectorRunner) Status() LabIPSelectorStatus {
	r.mu.Lock()
	defer r.mu.Unlock()
	status := LabIPSelectorStatus{
		Running:  r.running,
		Progress: r.progress,
		Phase:    r.progressPhase,
	}
	if !r.nextRun.IsZero() {
		nextRun := r.nextRun
		status.NextRun = &nextRun
	}
	if r.lastRun != nil {
		lastRun := *r.lastRun
		status.LastRun = &lastRun
	}
	return status
}

func newLabRunID() string {
	buf := make([]byte, 8)
	if _, err := rand.Read(buf); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(buf)
}
