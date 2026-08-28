package services

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"tunnel-manager/models"
	"tunnel-manager/store"
)

// DefaultMonitorInterval is used when a monitor has no explicit interval.
const DefaultMonitorInterval = 60

const minMonitorInterval = 30

// MonitorInterval normalizes the configured interval.
func MonitorInterval(sec int) int {
	if sec < minMonitorInterval {
		return DefaultMonitorInterval
	}
	return sec
}

// ProbeOutcome reports one target probe for API consumers.
type ProbeOutcome struct {
	TargetID  string `json:"target_id"`
	Name      string `json:"name"`
	URL       string `json:"url"`
	State     string `json:"state"`
	HTTPCode  int    `json:"http_code,omitempty"`
	LatencyMs int64  `json:"latency_ms"`
	Error     string `json:"error,omitempty"`
}

// Runner schedules periodic probes for every configured monitor.
type Runner struct {
	st       *store.Store
	hb       *HeartbeatLog
	mailer   func() *Mailer
	mu       sync.Mutex
	lastRun  map[string]time.Time
	inflight map[string]bool
}

// NewRunner wires a scheduler over the store and heartbeat log.
func NewRunner(st *store.Store, hb *HeartbeatLog) *Runner {
	return &Runner{st: st, hb: hb, lastRun: map[string]time.Time{}, inflight: map[string]bool{}}
}

// SetMailer wires the alert email provider, built lazily per notification.
func (r *Runner) SetMailer(provider func() *Mailer) {
	r.mailer = provider
}

// NewMonitorID returns a random hex identifier.
func NewMonitorID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// Start launches the scheduling loop until ctx is cancelled.
func (r *Runner) Start(ctx context.Context) {
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case now := <-ticker.C:
			r.tick(ctx, now)
		}
	}
}

func (r *Runner) tick(ctx context.Context, now time.Time) {
	for _, m := range r.st.GetConfig().Monitors {
		r.mu.Lock()
		last, seen := r.lastRun[m.ID]
		busy := r.inflight[m.ID]
		interval := time.Duration(MonitorInterval(m.IntervalSec)) * time.Second
		if busy || (seen && now.Sub(last) < interval) {
			r.mu.Unlock()
			continue
		}
		r.lastRun[m.ID] = now
		r.inflight[m.ID] = true
		r.mu.Unlock()
		go func(m models.Monitor) {
			defer func() {
				r.mu.Lock()
				r.inflight[m.ID] = false
				r.mu.Unlock()
			}()
			r.RunNow(context.WithoutCancel(ctx), m)
		}(m)
	}
}

// RunNow probes every target of one monitor concurrently and records
// the outcomes in the heartbeat log.
func (r *Runner) RunNow(ctx context.Context, m models.Monitor) []ProbeOutcome {
	if len(m.Targets) == 0 {
		return []ProbeOutcome{}
	}
	outcomes := make([]ProbeOutcome, len(m.Targets))
	sem := make(chan struct{}, 8)
	var wg sync.WaitGroup
	for i, t := range m.Targets {
		wg.Add(1)
		go func(i int, t models.MonitorTarget) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			res := ProbeTarget(ctx, t.Type, t.URL, t.Method)
			errText := ProbeErrorText(res.Err)
			outcomes[i] = ProbeOutcome{
				TargetID:  t.ID,
				Name:      t.Name,
				URL:       t.URL,
				State:     res.State,
				HTTPCode:  res.HTTPCode,
				LatencyMs: res.LatencyMs,
				Error:     errText,
			}
			r.hb.Append(m.ID, t.ID, Heartbeat{
				T: time.Now().UnixMilli(),
				S: res.State,
				M: res.LatencyMs,
				C: res.HTTPCode,
				E: errText,
			})

			// Alert on OK<->down state changes only; the previous state is
			// persisted so restarts never trigger duplicate alerts.
			previous := t.LastState
			if previous != res.State {
				_ = r.st.UpdateTargetLastState(m.ID, t.ID, res.State)
				if m.AlertEnabled && previous != "" {
					go r.notifyStateChange(m, t, outcomes[i], previous)
				}
			}
		}(i, t)
	}
	wg.Wait()
	return outcomes
}

// UptimePct returns the percentage of fully-healthy checks (ok only).
func UptimePct(hbs []Heartbeat) float64 {
	if len(hbs) == 0 {
		return 0
	}
	ok := 0
	for _, hb := range hbs {
		if hb.S == "ok" {
			ok++
		}
	}
	pct := float64(ok) / float64(len(hbs)) * 100
	return float64(int(pct*100+0.5)) / 100
}

// Downsample evenly reduces a history slice to at most n entries.
func Downsample(hbs []Heartbeat, n int) []Heartbeat {
	if n <= 0 || len(hbs) <= n {
		return hbs
	}
	step := float64(len(hbs)) / float64(n)
	out := make([]Heartbeat, 0, n)
	for i := 0; i < n; i++ {
		idx := int(float64(i) * step)
		if idx >= len(hbs) {
			idx = len(hbs) - 1
		}
		out = append(out, hbs[idx])
	}
	return out
}

// notifyStateChange sends one alert email for a target state change and
// records the delivery attempt. Failures never affect the probe loop.
func (r *Runner) notifyStateChange(m models.Monitor, t models.MonitorTarget, outcome ProbeOutcome, previous string) {
	defer func() {
		if rec := recover(); rec != nil {
			log.Printf("monitor alert panic: %v", rec)
		}
	}()

	entry := models.AlertLog{
		MonitorID:  m.ID,
		TargetID:   t.ID,
		TargetName: t.Name,
		State:      outcome.State,
		HTTPCode:   outcome.HTTPCode,
		Error:      outcome.Error,
	}

	mailer := r.mailer()
	if mailer == nil {
		entry.Detail = "SMTP 未配置，告警未发送"
		r.st.AddAlertLog(entry)
		return
	}
	recipients := alertRecipients(r.st, m)
	if len(recipients) == 0 {
		entry.Detail = "未配置收件邮箱"
		r.st.AddAlertLog(entry)
		return
	}

	stateLabel := "状态异常"
	if outcome.State == "ok" {
		stateLabel = "已恢复"
	}
	subject := fmt.Sprintf("[Tunnel Manager] %s · %s", m.Name, stateLabel)
	var body strings.Builder
	body.WriteString(fmt.Sprintf("监控项目：%s\n", m.Name))
	body.WriteString(fmt.Sprintf("探测目标：%s（%s）\n", t.Name, t.URL))
	body.WriteString(fmt.Sprintf("状态变化：%s -> %s\n", previous, outcome.State))
	if outcome.HTTPCode > 0 {
		body.WriteString(fmt.Sprintf("HTTP 状态码：%d\n", outcome.HTTPCode))
	}
	if outcome.Error != "" {
		body.WriteString(fmt.Sprintf("错误信息：%s\n", outcome.Error))
	}
	when := time.Now().Format("2006-01-02 15:04:05")
	body.WriteString(fmt.Sprintf("时间：%s\n", when))

	plain, htmlBody := AlertEmail(m.Name, t.Name, t.URL, previous, outcome.State, outcome.HTTPCode, outcome.Error, when)

	var sendErrors []string
	for _, to := range recipients {
		if err := mailer.Send(to, subject, plain, htmlBody); err != nil {
			sendErrors = append(sendErrors, to+": "+err.Error())
		}
	}
	if len(sendErrors) > 0 {
		entry.Notified = false
		entry.Detail = strings.Join(sendErrors, "; ")
		log.Printf("monitor alert mail failed: %v", entry.Detail)
	} else {
		entry.Notified = true
		entry.Detail = "已发送至 " + strings.Join(recipients, ", ")
	}
	r.st.AddAlertLog(entry)
}

// alertRecipients resolves the notification addresses: per-monitor emails
// when configured, otherwise the owner account email.
func alertRecipients(st *store.Store, m models.Monitor) []string {
	if emails := strings.TrimSpace(m.AlertEmails); emails != "" {
		out := make([]string, 0, 4)
		for _, piece := range strings.Split(emails, ",") {
			if piece = strings.TrimSpace(piece); piece != "" {
				out = append(out, piece)
			}
		}
		return out
	}
	if owner, ok := st.GetUserByID(m.UserID); ok && owner.Email != "" {
		return []string{owner.Email}
	}
	return nil
}
