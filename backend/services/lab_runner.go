package services

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	"tunnel-manager/auth"
	"tunnel-manager/models"
	"tunnel-manager/store"
)

const labHuaweiSecretPurpose = "lab-huawei-secret-key"

var ErrLabRunning = errors.New("另一个优选任务正在执行")

type LabIPSelectorRunner struct {
	store             *store.Store
	encryptionKey     []byte
	resolver          LabTXTResolver
	control           sync.Mutex
	mu                sync.Mutex
	running           bool
	cancel            context.CancelFunc
	nextRun           time.Time
	scheduledInterval time.Duration
	lastRun           *models.LabIPSelectorRun
	progress          LabIPSelectorProgress
	progressPhase     string
	lastError         string
	scan              func(context.Context, models.LabIPSelectorSettings, ...func(LabIPSelectorProgress)) ([]models.LabIPScanResult, []models.LabIPScanResult, []string, []models.LabIPSegmentResult, error)
}

type LabIPSelectorStatus struct {
	Running           bool                         `json:"running"`
	NextRun           *time.Time                   `json:"next_run,omitempty"`
	LastRun           *models.LabIPSelectorRun     `json:"last_run,omitempty"`
	Progress          LabIPSelectorProgress        `json:"progress"`
	Phase             string                       `json:"phase"`
	Verification      models.LabDomainVerification `json:"verification"`
	BudgetDay         string                       `json:"budget_day"`
	BudgetUsed        int                          `json:"budget_used"`
	BudgetLimit       int                          `json:"budget_limit"`
	RequestsPerSecond int                          `json:"requests_per_second"`
	MaxWorkers        int                          `json:"max_workers"`
	SuggestedDomain   string                       `json:"suggested_domain"`
	LastError         string                       `json:"last_error"`
	NextAllowedAt     int64                        `json:"next_allowed_at"`
}

func NewLabIPSelectorRunner(st *store.Store, encryptionKey []byte, resolvers ...LabTXTResolver) *LabIPSelectorRunner {
	var resolver LabTXTResolver = net.DefaultResolver
	if len(resolvers) > 0 && resolvers[0] != nil {
		resolver = resolvers[0]
	}
	return &LabIPSelectorRunner{store: st, encryptionKey: append([]byte(nil), encryptionKey...), resolver: resolver, scan: ScanLabIPs}
}

func (runner *LabIPSelectorRunner) Start(ctx context.Context) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			runner.cancelRun()
			return
		case <-ticker.C:
			settings := runner.store.GetLabSettings()
			verification := runner.store.GetLabSecurity().Verification
			interval := time.Duration(settings.IntervalMins) * time.Minute
			if !runner.store.GetAppSettings().ExperimentalFeatures || !settings.Schedule || verification.VerifiedAt == 0 || interval < 5*time.Minute {
				runner.mu.Lock()
				runner.nextRun = time.Time{}
				runner.scheduledInterval = 0
				runner.mu.Unlock()
				continue
			}
			runner.mu.Lock()
			if runner.scheduledInterval != interval {
				runner.scheduledInterval = interval
				runner.nextRun = time.Now().Add(interval)
			}
			due := !runner.running && !time.Now().Before(runner.nextRun)
			if due {
				runner.nextRun = time.Now().Add(interval)
			}
			runner.mu.Unlock()
			if due {
				_ = runner.trigger(ctx, verification.OwnerID, true)
			}
		}
	}
}

func (runner *LabIPSelectorRunner) ownerAllowed(ownerID string) bool {
	user, found := runner.store.GetUserByID(ownerID)
	return ownerID != "" && found && user.Role == models.RoleAdmin && user.Status == models.UserActive
}

func (runner *LabIPSelectorRunner) SaveSettings(settings models.LabIPSelectorSettings, ownerID string) error {
	if runner.ownerAllowed(ownerID) {
		runner.cancelRun()
	}
	runner.control.Lock()
	defer runner.control.Unlock()
	if !runner.ownerAllowed(ownerID) {
		return errors.New("请使用有效的管理员账号操作")
	}
	if err := NormalizeLabProbe(&settings); err != nil {
		return err
	}
	security := runner.store.GetLabSecurity()
	authorized := validateLabVerification(security.Verification, ownerID) == nil
	runner.cancelRun()
	if !authorized {
		settings.Schedule = false
		if err := runner.store.UpdateLabSecurity(func(state *models.LabSecurityState) error {
			if state.Verification.OwnerID != ownerID {
				state.Verification = models.LabDomainVerification{}
			}
			return nil
		}); err != nil {
			return err
		}
	}
	return runner.store.SetLabSettings(settings)
}

func (runner *LabIPSelectorRunner) NewChallenge(domain, ownerID string) (models.LabDomainVerification, error) {
	if runner.ownerAllowed(ownerID) {
		runner.cancelRun()
	}
	runner.control.Lock()
	defer runner.control.Unlock()
	return runner.newChallengeLocked(domain, ownerID)
}

func (runner *LabIPSelectorRunner) newChallengeLocked(domain, ownerID string) (models.LabDomainVerification, error) {
	var empty models.LabDomainVerification
	if !runner.ownerAllowed(ownerID) {
		return empty, errors.New("请使用有效的管理员账号验证域名")
	}
	domain, err := LabVerificationDomain(domain)
	if err != nil {
		return empty, err
	}
	nonce, err := newLabToken()
	if err != nil {
		return empty, err
	}
	instance, err := newLabToken()
	if err != nil {
		return empty, err
	}
	runner.cancelRun()
	if err := runner.store.SuspendLab(); err != nil {
		return empty, err
	}
	var challenge models.LabDomainVerification
	err = runner.store.UpdateLabSecurity(func(state *models.LabSecurityState) error {
		if state.InstanceID == "" {
			state.InstanceID = instance
		}
		challenge = models.LabDomainVerification{
			Domain:     domain,
			RecordName: "_tunnel-manager-verify." + domain,
			Token:      "tunnel-manager=" + state.InstanceID + "." + nonce,
			OwnerID:    ownerID, IssuedAt: time.Now().Unix(),
		}
		state.Verification = challenge
		return nil
	})
	return challenge, err
}

func (runner *LabIPSelectorRunner) Verify(ctx context.Context, ownerID string) (models.LabDomainVerification, error) {
	if runner.ownerAllowed(ownerID) {
		runner.cancelRun()
	}
	runner.control.Lock()
	defer runner.control.Unlock()
	return runner.verifyLocked(ctx, ownerID, true)
}

func (runner *LabIPSelectorRunner) verifyLocked(ctx context.Context, ownerID string, allowPending bool) (models.LabDomainVerification, error) {
	security := runner.store.GetLabSecurity()
	verification := security.Verification
	if !runner.ownerAllowed(ownerID) {
		return verification, errors.New("验证所属管理员已失效")
	}
	check := verification
	if allowPending && check.VerifiedAt == 0 {
		if check.IssuedAt == 0 || time.Now().Unix()-check.IssuedAt > 86400 {
			return verification, errors.New("待验证记录已超过 24 小时，请重新生成")
		}
		check.VerifiedAt = 1
	}
	if security.InstanceID == "" || !strings.HasPrefix(check.Token, "tunnel-manager="+security.InstanceID+".") {
		return verification, errors.New("请先生成本实例的 DNS TXT 验证记录")
	}
	if err := validateLabVerification(check, ownerID); err != nil {
		return verification, err
	}
	verification.CheckedAt = time.Now().Unix()
	lookupErr := lookupLabVerification(ctx, runner.resolver, verification)
	verification.LastError = ""
	if lookupErr != nil {
		verification.VerifiedAt = 0
		verification.LastError = lookupErr.Error()
		runner.cancelRun()
		if err := runner.store.SuspendLab(); err != nil {
			return verification, err
		}
	} else {
		verification.VerifiedAt = verification.CheckedAt
	}
	err := runner.store.UpdateLabSecurity(func(state *models.LabSecurityState) error {
		state.Verification = verification
		return nil
	})
	if err != nil {
		return verification, err
	}
	return verification, lookupErr
}

func (runner *LabIPSelectorRunner) prepare(ctx context.Context, ownerID string) (models.LabIPSelectorSettings, error) {
	settings := runner.store.GetLabSettings()
	appSettings := runner.store.GetAppSettings()
	settings.Limits = appSettings.LabLimits()
	if !appSettings.ExperimentalFeatures {
		return settings, errors.New("实验性功能已关闭")
	}
	if err := NormalizeLabProbe(&settings); err != nil {
		return settings, err
	}
	if settings.Workers < 1 || settings.Workers > models.LabWorkerCeiling || settings.Timeout < 1 || settings.Timeout > 15 || settings.Top < 1 || settings.Top > 100 {
		return settings, errors.New("探测参数无效，请重新保存配置")
	}
	if _, err := ParseLabStatuses(settings.Statuses); err != nil {
		return settings, err
	}
	targets, err := ParseLabIPTargets(settings.IPTargets)
	if err != nil {
		return settings, err
	}
	if err := ValidateLabPublicTargets(targets); err != nil {
		return settings, err
	}
	if _, err := runner.verifyLocked(ctx, ownerID, false); err != nil {
		return settings, err
	}
	now := time.Now().UTC()
	err = runner.store.UpdateLabSecurity(func(state *models.LabSecurityState) error {
		if now.Unix()-state.LastStartedAt < 60 {
			return errors.New("任务启动间隔至少 60 秒，请稍后重试")
		}
		day := now.Format("2006-01-02")
		if state.BudgetDay != day {
			state.BudgetDay, state.BudgetUsed = day, 0
		}
		if len(targets) > settings.Limits.DailyRequests-state.BudgetUsed {
			return errors.New("本日扫描预算不足，请减少候选 IP 或等待 UTC 次日")
		}
		state.BudgetUsed += len(targets)
		state.LastStartedAt = now.Unix()
		return nil
	})
	return settings, err
}

func (runner *LabIPSelectorRunner) Trigger(ctx context.Context, ownerID string) error {
	return runner.trigger(ctx, ownerID, false)
}

func (runner *LabIPSelectorRunner) trigger(ctx context.Context, ownerID string, scheduled bool) error {
	runner.control.Lock()
	defer runner.control.Unlock()
	if scheduled && !runner.store.GetLabSettings().Schedule {
		return errors.New("定时任务已停用")
	}
	runner.mu.Lock()
	busy := runner.running
	runner.mu.Unlock()
	if busy {
		return ErrLabRunning
	}
	settings, err := runner.prepare(ctx, ownerID)
	if err != nil {
		runner.mu.Lock()
		runner.lastError = err.Error()
		runner.mu.Unlock()
		return err
	}
	runContext := runner.beginRun(context.Background())
	go func() { _, _ = runner.run(runContext, settings, ownerID) }()
	return nil
}

func (runner *LabIPSelectorRunner) beginRun(parent context.Context) context.Context {
	deadline := time.Now().Add(2 * time.Hour)
	midnight := time.Now().UTC().Truncate(24 * time.Hour).Add(24 * time.Hour)
	if midnight.Before(deadline) {
		deadline = midnight
	}
	ctx, cancel := context.WithDeadline(parent, deadline)
	runner.mu.Lock()
	runner.running, runner.cancel = true, cancel
	runner.progress = LabIPSelectorProgress{}
	runner.progressPhase = "preparing"
	runner.lastError = ""
	runner.mu.Unlock()
	return ctx
}

func (runner *LabIPSelectorRunner) cancelRun() {
	runner.mu.Lock()
	defer runner.mu.Unlock()
	if runner.cancel != nil {
		runner.cancel()
	}
}

func (runner *LabIPSelectorRunner) Stop() error {
	runner.cancelRun()
	runner.control.Lock()
	defer runner.control.Unlock()
	runner.cancelRun()
	return runner.store.SuspendLab()
}

func (runner *LabIPSelectorRunner) run(ctx context.Context, settings models.LabIPSelectorSettings, ownerID string) (run models.LabIPSelectorRun, runErr error) {
	ctx, cancelCurrentRun := context.WithCancel(ctx)
	defer cancelCurrentRun()
	identifier, err := newLabToken()
	if err != nil {
		identifier = fmt.Sprintf("%d", time.Now().UnixNano())
	}
	run = models.LabIPSelectorRun{ID: identifier, StartedAt: time.Now().Unix()}
	finished := make(chan struct{})
	go func() {
		ticker := time.NewTicker(250 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-finished:
				return
			case <-ctx.Done():
				return
			case <-ticker.C:
				if !runner.store.GetAppSettings().ExperimentalFeatures || !runner.ownerAllowed(ownerID) {
					cancelCurrentRun()
					return
				}
			}
		}
	}()
	defer func() {
		close(finished)
		run.FinishedAt = time.Now().Unix()
		if runErr != nil {
			run.Error = runErr.Error()
		}
		if err := runner.store.AppendLabRun(run); err != nil && runErr == nil {
			runErr, run.Error = err, err.Error()
		}
		runner.mu.Lock()
		if runner.cancel != nil {
			runner.cancel()
		}
		runner.cancel, runner.running = nil, false
		copyRun := run
		runner.lastRun = &copyRun
		if runErr != nil {
			runner.progressPhase = "failed"
		}
		runner.nextRun = time.Now().Add(time.Duration(settings.IntervalMins) * time.Minute)
		runner.mu.Unlock()
	}()
	all, matches, selected, segments, err := runner.scan(ctx, settings, func(progress LabIPSelectorProgress) {
		runner.mu.Lock()
		runner.progress, runner.progressPhase = progress, "scanning"
		runner.mu.Unlock()
	})
	run.Scanned, run.Matched, run.SelectedIPs, run.Segments = len(all), len(matches), selected, segments
	run.Probe = &models.LabProbeSnapshot{Host: settings.Host, SNI: settings.SNI, Path: settings.Path, Statuses: settings.Statuses}
	matchedIPs := make(map[string]bool, len(matches))
	for _, result := range matches {
		matchedIPs[result.IP] = true
	}
	for _, result := range all {
		if matchedIPs[result.IP] && result.Error == "" {
			continue
		}
		run.Rejected++
		if len(run.Diagnostics) < 100 {
			run.Diagnostics = append(run.Diagnostics, result)
		}
	}
	if err != nil {
		return run, err
	}
	if err := ctx.Err(); err != nil {
		return run, err
	}
	if len(selected) == 0 {
		return run, errors.New("没有通过 HTTP 验证的 IP，保留现有 DNS")
	}
	top := settings.Top
	if top > len(matches) {
		top = len(matches)
	}
	run.Results = matches[:top]
	if settings.UpdateDNS {
		runner.control.Lock()
		defer runner.control.Unlock()
		savedSettings := settings
		savedSettings.Limits = models.LabLimits{}
		if ctx.Err() != nil || !runner.store.GetAppSettings().ExperimentalFeatures || runner.store.GetLabSettings() != savedSettings {
			return run, errors.New("任务已停止或配置已变化，未更新 DNS")
		}
		if _, err := runner.verifyLocked(ctx, ownerID, false); err != nil {
			return run, err
		}
		if settings.AccessKey == "" || settings.SecretKey == "" {
			return run, errors.New("缺少华为云 DNS 凭证")
		}
		secret, err := auth.DecryptSecret(runner.encryptionKey, labHuaweiSecretPurpose, settings.SecretKey)
		if err != nil {
			return run, errors.New("无法解密华为云密钥，请保留原应用加密密钥或重新配置")
		}
		runner.mu.Lock()
		runner.progressPhase = "updating_dns"
		runner.mu.Unlock()
		client := newHuaweiDNSClient(settings.Endpoint, settings.AccessKey, string(secret))
		client.context = ctx
		if err := client.UpdateARecord(settings.ZoneID, settings.Zone, settings.Record, settings.TTL, selected); err != nil {
			return run, err
		}
		run.DNSUpdated = true
	}
	runner.mu.Lock()
	runner.progressPhase = "completed"
	runner.mu.Unlock()
	run.Success = true
	return run, nil
}

func (runner *LabIPSelectorRunner) Status() LabIPSelectorStatus {
	security := runner.store.GetLabSecurity()
	limits := runner.store.GetAppSettings().LabLimits()
	day := time.Now().UTC().Format("2006-01-02")
	if security.BudgetDay != day {
		security.BudgetUsed = 0
	}
	runner.mu.Lock()
	defer runner.mu.Unlock()
	status := LabIPSelectorStatus{
		Running: runner.running, Progress: runner.progress, Phase: runner.progressPhase,
		Verification: security.Verification, BudgetDay: day, BudgetUsed: security.BudgetUsed,
		BudgetLimit: limits.DailyRequests, RequestsPerSecond: limits.RequestsPerSecond,
		MaxWorkers: limits.MaxWorkers, SuggestedDomain: security.Verification.Domain, LastError: runner.lastError,
		NextAllowedAt: security.LastStartedAt + 60,
	}
	if !runner.nextRun.IsZero() {
		next := runner.nextRun
		status.NextRun = &next
	}
	if runner.lastRun != nil {
		last := *runner.lastRun
		status.LastRun = &last
	}
	return status
}
