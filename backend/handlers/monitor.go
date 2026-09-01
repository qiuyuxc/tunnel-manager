package handlers

import (
	"net/http"
	"strings"
	"sync"
	"time"

	"tunnel-manager/services"
	"tunnel-manager/store"
)

// ServiceProbe is the result of checking one routed hostname.
type ServiceProbe struct {
	Hostname  string `json:"hostname"`
	Service   string `json:"service"`
	State     string `json:"state"` // ok | warn | down
	HTTPCode  int    `json:"http_code,omitempty"`
	LatencyMs int64  `json:"latency_ms"`
	Error     string `json:"error,omitempty"`
}

// ServicesHealthResponse reports reachability of every routed hostname on the selected tunnel.
type ServicesHealthResponse struct {
	TunnelID   string         `json:"tunnel_id,omitempty"`
	TunnelName string         `json:"tunnel_name,omitempty"`
	CheckedAt  string         `json:"checked_at"`
	Services   []ServiceProbe `json:"services"`
}

// MonitorHandler probes ingress routes of the selected tunnel through Cloudflare's edge.
type MonitorHandler struct {
	cf *services.CloudflareClient
	st *store.Store
}

func NewMonitorHandler(cf *services.CloudflareClient, st *store.Store) *MonitorHandler {
	return &MonitorHandler{cf: cf, st: st}
}

const monitorConcurrency = 8

// resolveUserID returns the account whose preferences apply to this request.
// API-key access (no individual account) falls back to the administrator.
func (h *MonitorHandler) resolveUserID(r *http.Request) string {
	if user := SessionUser(r); user != nil && !user.IsAPIKey() {
		return user.ID
	}
	return h.st.AdminUserID()
}

// ServiceStatus handles GET /api/monitor/services.
func (h *MonitorHandler) ServiceStatus(w http.ResponseWriter, r *http.Request) {
	prefs := h.st.GetUserPrefs(h.resolveUserID(r))
	if prefs.TunnelID == "" {
		writeJSON(w, http.StatusOK, ServicesHealthResponse{Services: []ServiceProbe{}, CheckedAt: nowRFC3339()})
		return
	}
	detail, err := UserCF(r).GetTunnelConfig(prefs.TunnelID)
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
		return
	}

	type target struct {
		index   int
		host    string
		service string
	}
	var targets []target
	for i, rule := range detail.Result.Config.Ingress {
		host := strings.TrimSpace(rule.Hostname)
		svc := strings.TrimSpace(rule.Service)
		if host == "" || svc == "" {
			continue // catch-all http_status rule
		}
		if !strings.HasPrefix(svc, "http://") && !strings.HasPrefix(svc, "https://") {
			continue // unix sockets and synthetic responses cannot be probed
		}
		targets = append(targets, target{index: i, host: host, service: svc})
	}

	results := make([]ServiceProbe, len(targets))
	sem := make(chan struct{}, monitorConcurrency)
	var wg sync.WaitGroup
	for _, t := range targets {
		wg.Add(1)
		go func(t target) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			res := services.ProbeURL(r.Context(), "https://"+t.host+"/")
			results[t.index] = ServiceProbe{
				Hostname:  t.host,
				Service:   t.service,
				State:     res.State,
				HTTPCode:  res.HTTPCode,
				LatencyMs: res.LatencyMs,
				Error:     services.ProbeErrorText(res.Err),
			}
		}(t)
	}
	wg.Wait()

	writeJSON(w, http.StatusOK, ServicesHealthResponse{
		TunnelID:   prefs.TunnelID,
		TunnelName: prefs.TunnelName,
		CheckedAt:  nowRFC3339(),
		Services:   results,
	})
}

func nowRFC3339() string { return time.Now().Format(time.RFC3339) }
