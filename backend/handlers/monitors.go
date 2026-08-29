package handlers

import (
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"

	"tunnel-manager/models"
	"tunnel-manager/services"
	"tunnel-manager/store"
)

// barsPerTarget caps how many bars each target renders on pages.
const barsPerTarget = 48

var slugRe = regexp.MustCompile("^[a-z0-9_-]{1,32}$")

// domainRe matches a dotted hostname: labels of a-z 0-9 and inner hyphens,
// at least two of them, so a bare label cannot be claimed.
var domainRe = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]*[a-z0-9])?(\.[a-z0-9]([a-z0-9-]*[a-z0-9])?)+$`)

type statusDomainProvisioner interface {
	ProvisionStatusDomain(userID, panelHost, hostname, mode, auxDomain, preferredCNAME string) error
}

// MonitorsHandler serves monitor CRUD and the public status API.
type MonitorsHandler struct {
	updateMu sync.Mutex
	st       *store.Store
	hb       *services.HeartbeatLog
	runner   *services.Runner
	domains  statusDomainProvisioner
}

// NewMonitorsHandler wires the handler with its collaborators.
func NewMonitorsHandler(st *store.Store, hb *services.HeartbeatLog, runner *services.Runner, domains statusDomainProvisioner) *MonitorsHandler {
	return &MonitorsHandler{st: st, hb: hb, runner: runner, domains: domains}
}

func newAPIToken() string { return services.NewMonitorID() }

type targetStatus struct {
	ID          string       `json:"id"`
	Name        string       `json:"name"`
	URL         string       `json:"url"`
	Type        string       `json:"type,omitempty"`
	Method      string       `json:"method,omitempty"`
	LinkEnabled bool         `json:"link_enabled,omitempty"`
	State       string       `json:"state,omitempty"`
	LatencyMs   int64        `json:"latency_ms,omitempty"`
	HTTPCode    int          `json:"http_code,omitempty"`
	Error       string       `json:"error,omitempty"`
	Uptime24h   float64      `json:"uptime_24h"`
	Bars        []monitorBar `json:"bars,omitempty"`
}

type monitorBar struct {
	T int64  `json:"t"`
	S string `json:"s"`
	M int64  `json:"ms,omitempty"`
	C int    `json:"c,omitempty"`
}

type monitorView struct {
	ID                   string         `json:"id"`
	Name                 string         `json:"name"`
	IntervalSec          int            `json:"interval_sec"`
	PublishEnabled       bool           `json:"publish_enabled"`
	PublicToken          string         `json:"public_token,omitempty"`
	PublicTitle          string         `json:"public_title,omitempty"`
	PublicIcon           string         `json:"public_icon,omitempty"`
	PublicSlug           string         `json:"public_slug,omitempty"`
	PublicDomain         string         `json:"public_domain,omitempty"`
	PublicDomainMode     string         `json:"public_domain_mode"`
	PublicAuxDomain      string         `json:"public_aux_domain,omitempty"`
	PublicPreferredCNAME string         `json:"public_preferred_cname,omitempty"`
	DomainWarning        string         `json:"domain_warning,omitempty"`
	PublicTheme          string         `json:"public_theme,omitempty"`
	Announcement         string         `json:"announcement,omitempty"`
	AlertEnabled         bool           `json:"alert_enabled"`
	AlertEmails          string         `json:"alert_emails,omitempty"`
	CreatedAt            int64          `json:"created_at,omitempty"`
	Targets              []targetStatus `json:"targets"`
}

func (h *MonitorsHandler) enrich(m models.Monitor, withBars bool) monitorView {
	view := monitorView{
		ID:                   m.ID,
		Name:                 m.Name,
		IntervalSec:          services.MonitorInterval(m.IntervalSec),
		PublishEnabled:       m.PublishEnabled,
		PublicToken:          m.PublicToken,
		PublicTitle:          m.PublicTitle,
		Announcement:         m.Announcement,
		AlertEnabled:         m.AlertEnabled,
		AlertEmails:          m.AlertEmails,
		PublicIcon:           m.PublicIcon,
		PublicSlug:           m.PublicSlug,
		PublicDomain:         m.PublicDomain,
		PublicDomainMode:     m.PublicDomainMode,
		PublicAuxDomain:      m.PublicAuxDomain,
		PublicPreferredCNAME: m.PublicPreferredCNAME,
		PublicTheme:          m.PublicTheme,
		CreatedAt:            m.CreatedAt,
		Targets:              make([]targetStatus, 0, len(m.Targets)),
	}
	dayAgo := time.Now().Add(-24 * time.Hour).UnixMilli()
	for _, t := range m.Targets {
		list := h.hb.Recent(m.ID, t.ID, 0)
		st := targetStatus{ID: t.ID, Name: t.Name, URL: t.URL, Type: t.Type, Method: t.Method, LinkEnabled: t.LinkEnabled}
		if len(list) > 0 {
			last := list[len(list)-1]
			st.State = last.S
			st.LatencyMs = last.M
			st.HTTPCode = last.C
			st.Error = publicErrorText(last.E)
			st.Uptime24h = services.UptimePct(filterSince(list, dayAgo))
			if withBars {
				raw := services.Downsample(list, barsPerTarget)
				bars := make([]monitorBar, 0, len(raw))
				for _, b := range raw {
					bars = append(bars, monitorBar{T: b.T, S: b.S, M: b.M, C: b.C})
				}
				st.Bars = bars
			}
		}
		view.Targets = append(view.Targets, st)
	}
	return view
}

func filterSince(list []services.Heartbeat, sinceMS int64) []services.Heartbeat {
	out := make([]services.Heartbeat, 0, len(list))
	for _, hb := range list {
		if hb.T >= sinceMS {
			out = append(out, hb)
		}
	}
	return out
}

func (h *MonitorsHandler) lookup(id string) (models.Monitor, bool) {
	for _, m := range h.st.GetConfig().Monitors {
		if m.ID == id {
			return m, true
		}
	}
	return models.Monitor{}, false
}

// visibleTo reports whether the requesting session may see this monitor.
// Administrators see every project; other users only their own.
func (h *MonitorsHandler) visibleTo(r *http.Request, m models.Monitor) bool {
	user := SessionUser(r)
	if user == nil {
		return false
	}
	if user.IsAdmin() {
		return true
	}
	return m.UserID == user.ID
}

// lookupVisible fetches the monitor only when the session may access it.
func (h *MonitorsHandler) lookupVisible(r *http.Request, id string) (models.Monitor, bool) {
	m, ok := h.lookup(id)
	if !ok || !h.visibleTo(r, m) {
		return models.Monitor{}, false
	}
	return m, true
}

type createReq struct {
	Name           string `json:"name"`
	IntervalSec    int    `json:"interval_sec"`
	PublishEnabled bool   `json:"publish_enabled"`
	AlertEnabled   bool   `json:"alert_enabled"`
	AlertEmails    string `json:"alert_emails"`
}

// Create handles POST /api/monitors.
func (h *MonitorsHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req createReq
	if err := readJSON(r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "name is required"})
		return
	}
	ownerID := ""
	if user := SessionUser(r); user != nil {
		ownerID = user.ID
		if ownerID == "" && user.IsAdmin() {
			ownerID = h.st.AdminUserID()
		}
	}
	m := models.Monitor{
		ID:               services.NewMonitorID(),
		UserID:           ownerID,
		Name:             name,
		IntervalSec:      req.IntervalSec,
		PublishEnabled:   req.PublishEnabled,
		AlertEnabled:     req.AlertEnabled,
		AlertEmails:      strings.TrimSpace(req.AlertEmails),
		PublicToken:      newAPIToken(),
		PublicDomainMode: services.BindingModePreferred,
		CreatedAt:        time.Now().Unix(),
	}
	if err := h.st.AddMonitor(m); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, h.enrich(m, false))
}

// List handles GET /api/monitors.
func (h *MonitorsHandler) List(w http.ResponseWriter, r *http.Request) {
	all := h.st.GetConfig().Monitors
	out := make([]monitorView, 0, len(all))
	for _, m := range all {
		if !h.visibleTo(r, m) {
			continue
		}
		out = append(out, h.enrich(m, true))
	}
	writeJSON(w, http.StatusOK, out)
}

// Get handles GET /api/monitors/{monitorID}.
func (h *MonitorsHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "monitorID")
	m, ok := h.lookupVisible(r, id)
	if !ok {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "monitor not found"})
		return
	}
	writeJSON(w, http.StatusOK, h.enrich(m, true))
}

type updateReq struct {
	Name                 *string `json:"name"`
	IntervalSec          *int    `json:"interval_sec"`
	PublishEnabled       *bool   `json:"publish_enabled"`
	RegenerateToken      bool    `json:"regenerate_token"`
	PublicTitle          *string `json:"public_title"`
	PublicIcon           *string `json:"public_icon"`
	PublicTheme          *string `json:"public_theme"`
	PublicSlug           *string `json:"public_slug"`
	PublicDomain         *string `json:"public_domain"`
	PublicDomainMode     *string `json:"public_domain_mode"`
	PublicAuxDomain      *string `json:"public_aux_domain"`
	PublicPreferredCNAME *string `json:"public_preferred_cname"`
	Announcement         *string `json:"announcement"`
	AlertEnabled         *bool   `json:"alert_enabled"`
	AlertEmails          *string `json:"alert_emails"`
}

// Update handles PUT /api/monitors/{monitorID}.
func (h *MonitorsHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "monitorID")
	current, ok := h.lookupVisible(r, id)
	h.updateMu.Lock()
	defer h.updateMu.Unlock()

	if !ok {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "monitor not found"})
		return
	}
	var req updateReq
	if err := readJSON(r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	if req.Name != nil {
		v := strings.TrimSpace(*req.Name)
		if v == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "name cannot be empty"})
			return
		}
		current.Name = v
	}
	if req.IntervalSec != nil {
		current.IntervalSec = *req.IntervalSec
	}
	if req.PublishEnabled != nil {
		current.PublishEnabled = *req.PublishEnabled
	}
	if req.AlertEnabled != nil {
		current.AlertEnabled = *req.AlertEnabled
	}
	if req.AlertEmails != nil {
		current.AlertEmails = strings.TrimSpace(*req.AlertEmails)
	}
	if req.RegenerateToken {
		current.PublicToken = newAPIToken()
	}
	if req.PublicTitle != nil {
		v := strings.TrimSpace(*req.PublicTitle)
		if len([]rune(v)) > 120 {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "public_title max 120 chars"})
			return
		}
		current.PublicTitle = v
	}
	if req.Announcement != nil {
		v := strings.TrimSpace(*req.Announcement)
		if len([]rune(v)) > 200 {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "announcement max 200 chars"})
			return
		}
		current.Announcement = v
	}
	if req.PublicIcon != nil {
		v := strings.TrimSpace(*req.PublicIcon)
		if v == "" {
			current.PublicIcon = ""
		} else {
			bad := len([]rune(v)) > 300 || strings.ContainsAny(v, "\r\n") || !(strings.HasPrefix(v, "http://") || strings.HasPrefix(v, "https://") || strings.HasPrefix(v, "/uploads/") || strings.HasPrefix(v, "data:image/"))
			if bad {
				writeJSON(w, http.StatusBadRequest, map[string]string{"error": "public_icon must be an image URL or short emoji"})
				return
			}
			current.PublicIcon = v
		}
	}
	if req.PublicTheme != nil {
		tv := strings.TrimSpace(*req.PublicTheme)
		switch tv {
		case "auto", "light", "dark":
			tv = ""
		case "warm_dark":
			tv = "warm"
		case "", "blue", "warm":
		default:
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid public_theme"})
			return
		}
		current.PublicTheme = tv
	}
	if req.PublicSlug != nil {
		sv := strings.ToLower(strings.TrimSpace(*req.PublicSlug))
		if sv == "" {
			current.PublicSlug = ""
		} else {
			if len(sv) > 32 || !slugRe.MatchString(sv) {
				writeJSON(w, http.StatusBadRequest, map[string]string{"error": "slug must be 1-32 chars of a-z 0-9 - _"})
				return
			}
			for _, other := range h.st.GetConfig().Monitors {
				if other.ID != current.ID && other.PublicSlug == sv {
					writeJSON(w, http.StatusBadRequest, map[string]string{"error": "slug already in use"})
					return
				}
			}
			current.PublicSlug = sv
		}
	}
	if req.PublicDomain != nil {
		dv := hostWithoutPort(*req.PublicDomain)
		if dv == "" {
			current.PublicDomain = ""
		} else {
			if len(dv) > 253 || !domainRe.MatchString(dv) {
				writeJSON(w, http.StatusBadRequest, map[string]string{"error": "自定义域名格式不正确，请填写完整主机名，如 status.example.com"})
				return
			}
			panelHost := hostWithoutPort(h.st.GetConfig().PanelHost)
			requestHost := hostWithoutPort(r.Host)
			if dv == panelHost || dv == requestHost {
				writeJSON(w, http.StatusBadRequest, map[string]string{"error": "不能使用面板自身域名"})
				return
			}
			for _, other := range h.st.GetConfig().Monitors {
				if other.ID != current.ID && strings.EqualFold(other.PublicDomain, dv) {
					writeJSON(w, http.StatusBadRequest, map[string]string{"error": "该域名已被其他状态页占用"})
					return
				}
			}
			current.PublicDomain = dv
		}
	}
	if req.PublicDomainMode != nil {
		mode, err := services.NormalizeBindingMode(*req.PublicDomainMode)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "自定义域名接入方式不正确"})
			return
		}
		current.PublicDomainMode = mode
	}
	if req.PublicAuxDomain != nil {
		auxDomain := hostWithoutPort(*req.PublicAuxDomain)
		if auxDomain != "" && (len(auxDomain) > 253 || !domainRe.MatchString(auxDomain)) {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "辅助回源域名格式不正确，请填写完整主机名"})
			return
		}
		current.PublicAuxDomain = auxDomain
	}
	if req.PublicPreferredCNAME != nil {
		preferredCNAME := hostWithoutPort(*req.PublicPreferredCNAME)
		if preferredCNAME != "" && (len(preferredCNAME) > 253 || !domainRe.MatchString(preferredCNAME)) {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "优选 CNAME 格式不正确，请填写完整主机名"})
			return
		}
		current.PublicPreferredCNAME = preferredCNAME
	}
	domainConfigTouched := req.PublicDomain != nil || req.PublicDomainMode != nil ||
		req.PublicAuxDomain != nil || req.PublicPreferredCNAME != nil
	if domainConfigTouched && current.PublicDomain != "" && current.PublicDomainMode == services.BindingModePreferred {
		if current.PublicAuxDomain == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "优选模式需要填写辅助回源域名"})
			return
		}
		if strings.EqualFold(current.PublicDomain, current.PublicAuxDomain) {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "访问域名和辅助回源域名不能相同"})
			return
		}
		if current.PublicPreferredCNAME != "" && strings.EqualFold(current.PublicDomain, current.PublicPreferredCNAME) {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "优选 CNAME 不能与访问域名相同"})
			return
		}
	}
	if err := h.st.MutateMonitor(id, func(dst *models.Monitor) bool {
		saved := *dst
		*dst = current
		_ = saved
		return true
	}); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	fresh, _ := h.lookup(id)
	view := h.enrich(fresh, false)
	if domainConfigTouched && fresh.PublicDomain != "" {
		if h.domains == nil {
			view.DomainWarning = "自动配置服务不可用"
		} else if err := h.domains.ProvisionStatusDomain(
			fresh.UserID, h.st.GetConfig().PanelHost, fresh.PublicDomain, fresh.PublicDomainMode, fresh.PublicAuxDomain, fresh.PublicPreferredCNAME,
		); err != nil {
			view.DomainWarning = err.Error()
		}
	}
	writeJSON(w, http.StatusOK, view)
}

// Delete handles DELETE /api/monitors/{monitorID}.
func (h *MonitorsHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "monitorID")
	if _, ok := h.lookupVisible(r, id); !ok {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "monitor not found"})
		return
	}
	if err := h.st.RemoveMonitor(id); err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "monitor not found"})
		return
	}
	h.hb.Forget(id)
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

type addTargetReq struct {
	Name   string `json:"name"`
	URL    string `json:"url"`
	Type   string `json:"type"`
	Method string `json:"method"`
	Link   bool   `json:"link_enabled"`
}

// probeSpec normalizes an add/edit target payload into validated fields.
func probeSpec(rawName, rawURL, rawType, rawMethod string) (name, url, pType, pMethod string, httpErr bool, errMsg string) {
	name = strings.TrimSpace(rawName)
	url = strings.TrimSpace(rawURL)
	if url == "" {
		return "", "", "", "", true, "url is required"
	}
	if name == "" {
		return "", "", "", "", true, "name is required"
	}
	pType = strings.ToLower(strings.TrimSpace(rawType))
	switch pType {
	case "", "http":
		pType = "http"
		if strings.Contains(url, "://") && !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
			return "", "", "", "", true, "only http(s) targets are supported"
		}
	case "tcp":
		url = strings.TrimPrefix(url, "tcp://")
	case "icmp":
		url = strings.TrimPrefix(strings.TrimPrefix(url, "https://"), "http://")
		if i := strings.IndexAny(url, "/:"); i >= 0 {
			url = url[:i]
		}
	default:
		return "", "", "", "", true, "type must be http, tcp or icmp"
	}
	pMethod = ""
	if pType == "http" {
		pMethod = strings.ToUpper(strings.TrimSpace(rawMethod))
		if pMethod != "POST" {
			pMethod = "GET"
		}
	}
	return name, url, pType, pMethod, false, ""
}

// AddTarget handles POST /api/monitors/{monitorID}/targets.
func (h *MonitorsHandler) AddTarget(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "monitorID")
	if _, ok := h.lookupVisible(r, id); !ok {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "monitor not found"})
		return
	}
	var req addTargetReq
	if err := readJSON(r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	name, rawURL, pType, pMethod, bad, msg := probeSpec(req.Name, req.URL, req.Type, req.Method)
	if bad {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": msg})
		return
	}
	t := models.MonitorTarget{
		ID:          services.NewMonitorID(),
		Name:        name,
		URL:         rawURL,
		Type:        pType,
		Method:      pMethod,
		LinkEnabled: req.Link,
		CreatedAt:   time.Now().Unix(),
	}
	added := false
	err := h.st.MutateMonitor(id, func(m *models.Monitor) bool {
		m.Targets = append(m.Targets, t)
		added = true
		return true
	})
	if err != nil || !added {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "monitor not found"})
		return
	}
	fresh, _ := h.lookup(id)
	writeJSON(w, http.StatusOK, h.enrich(fresh, true))
}

// EditTarget handles PUT /api/monitors/{monitorID}/targets/{targetID}.
func (h *MonitorsHandler) EditTarget(w http.ResponseWriter, r *http.Request) {
	monitorID := chi.URLParam(r, "monitorID")
	targetID := chi.URLParam(r, "targetID")
	if _, ok := h.lookupVisible(r, monitorID); !ok {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "monitor not found"})
		return
	}
	var req addTargetReq
	if err := readJSON(r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	name, rawURL, pType, pMethod, bad, msg := probeSpec(req.Name, req.URL, req.Type, req.Method)
	if bad {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": msg})
		return
	}
	probeChanged := false
	err := h.st.MutateMonitor(monitorID, func(m *models.Monitor) bool {
		for i := range m.Targets {
			t := &m.Targets[i]
			if t.ID != targetID {
				continue
			}
			probeChanged = t.URL != rawURL || t.Type != pType || t.Method != pMethod
			t.Name, t.URL, t.Type, t.Method, t.LinkEnabled = name, rawURL, pType, pMethod, req.Link
			return true
		}
		return false
	})
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	found := false
	if m, ok := h.lookup(monitorID); ok {
		for _, t := range m.Targets {
			if t.ID == targetID {
				found = true
			}
		}
	}
	if !found {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "target not found"})
		return
	}
	if probeChanged {
		h.hb.ForgetTarget(monitorID, targetID)
	}
	fresh, _ := h.lookup(monitorID)
	writeJSON(w, http.StatusOK, h.enrich(fresh, true))
}

// RemoveTarget handles DELETE /api/monitors/{monitorID}/targets/{targetID}.
func (h *MonitorsHandler) RemoveTarget(w http.ResponseWriter, r *http.Request) {
	monitorID := chi.URLParam(r, "monitorID")
	targetID := chi.URLParam(r, "targetID")
	if _, ok := h.lookupVisible(r, monitorID); !ok {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "monitor not found"})
		return
	}
	removed := false
	err := h.st.MutateMonitor(monitorID, func(m *models.Monitor) bool {
		kept := make([]models.MonitorTarget, 0, len(m.Targets))
		for _, t := range m.Targets {
			if t.ID == targetID {
				removed = true
				continue
			}
			kept = append(kept, t)
		}
		if !removed {
			return true
		}
		m.Targets = kept
		return true
	})
	if err != nil || !removed {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "target not found"})
		return
	}
	h.hb.ForgetTarget(monitorID, targetID)
	fresh, _ := h.lookup(monitorID)
	writeJSON(w, http.StatusOK, h.enrich(fresh, true))
}

// CheckNow handles POST /api/monitors/{monitorID}/check.
func (h *MonitorsHandler) CheckNow(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "monitorID")
	m, ok := h.lookupVisible(r, id)
	if !ok {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "monitor not found"})
		return
	}
	outcomes := h.runner.RunNow(r.Context(), m)
	okCount := 0
	for _, o := range outcomes {
		if o.State == "ok" {
			okCount++
		}
	}
	fresh, _ := h.lookup(id)
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"status":  "ok",
		"checked": len(outcomes),
		"ok":      okCount,
		"monitor": h.enrich(fresh, true),
	})
}

type bucketStat struct {
	Hour   int64   `json:"hour"`
	AvgMs  float64 `json:"avg_ms"`
	PeakMs int64   `json:"peak_ms"`
	Total  int     `json:"total"`
	Warn   int     `json:"warn"`
	Down   int     `json:"down"`
}

// Overview handles GET /api/monitors/overview with cross-monitor stats.
func (h *MonitorsHandler) Overview(w http.ResponseWriter, r *http.Request) {
	dayAgo := time.Now().Add(-24 * time.Hour).UnixMilli()
	all := h.st.GetConfig().Monitors
	visible := make([]models.Monitor, 0, len(all))
	for _, m := range all {
		if h.visibleTo(r, m) {
			visible = append(visible, m)
		}
	}
	all = visible
	targets := 0
	var okN, warnN, downN int
	var latSum, peak int64
	latCnt := 0
	type acc struct {
		hour       int64
		sum, peak  int64
		total      int
		warn, down int
	}
	bmap := map[int64]*acc{}
	for _, m := range all {
		for _, t := range m.Targets {
			targets++
			list := h.hb.Recent(m.ID, t.ID, dayAgo)
			if len(list) == 0 {
				downN++
				continue
			}
			switch list[len(list)-1].S {
			case "ok":
				okN++
			case "warn":
				warnN++
			default:
				downN++
			}
			for _, hb := range list {
				latSum += hb.M
				latCnt++
				if hb.M > peak {
					peak = hb.M
				}
				hr := hb.T - hb.T%3600000
				bk := bmap[hr]
				if bk == nil {
					bk = &acc{hour: hr / 1000}
					bmap[hr] = bk
				}
				bk.total++
				bk.sum += hb.M
				if hb.M > bk.peak {
					bk.peak = hb.M
				}
				switch hb.S {
				case "down":
					bk.down++
				case "warn":
					bk.warn++
				}
			}
		}
	}
	buckets := make([]bucketStat, 0, len(bmap))
	for _, bk := range bmap {
		st := bucketStat{Hour: bk.hour, PeakMs: bk.peak, Total: bk.total, Warn: bk.warn, Down: bk.down}
		if st.Total > 0 {
			st.AvgMs = float64(bk.sum) / float64(st.Total)
		}
		buckets = append(buckets, st)
	}
	for i := 1; i < len(buckets); i++ {
		for j := i; j > 0 && buckets[j].Hour < buckets[j-1].Hour; j-- {
			buckets[j], buckets[j-1] = buckets[j-1], buckets[j]
		}
	}
	avg := int64(0)
	if latCnt > 0 {
		avg = latSum / int64(latCnt)
	}
	var good, totalChecks int
	for _, m := range all {
		for _, t := range m.Targets {
			for _, hb := range filterSince(h.hb.Recent(m.ID, t.ID, 0), dayAgo) {
				totalChecks++
				if hb.S == "ok" {
					good++
				}
			}
		}
	}
	pct := 0.0
	if totalChecks > 0 {
		pct = float64(int(float64(good)/float64(totalChecks)*10000+0.5)) / 100
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"targets":         targets,
		"ok":              okN,
		"warn":            warnN,
		"down":            downN,
		"uptime_24h":      pct,
		"avg_latency_ms":  avg,
		"peak_latency_ms": peak,
		"buckets":         buckets,
	})
}

type publicStatus struct {
	Name         string         `json:"name"`
	UpdatedAt    int64          `json:"updated_at"`
	IntervalSec  int            `json:"interval_sec"`
	PublicTitle  string         `json:"public_title,omitempty"`
	PublicIcon   string         `json:"public_icon,omitempty"`
	PublicSlug   string         `json:"public_slug,omitempty"`
	PublicTheme  string         `json:"public_theme,omitempty"`
	Announcement string         `json:"announcement,omitempty"`
	Targets      []publicTarget `json:"targets"`
}

type publicTarget struct {
	Name      string      `json:"name"`
	State     string      `json:"state"`
	LatencyMs int64       `json:"latency_ms"`
	HTTPCode  int         `json:"http_code,omitempty"`
	Error     string      `json:"error,omitempty"`
	Uptime24h float64     `json:"uptime_24h"`
	Link      string      `json:"link,omitempty"`
	Bars      []publicBar `json:"bars"`
}

type publicBar struct {
	T int64  `json:"t"`
	S string `json:"s"`
	M int64  `json:"ms,omitempty"`
	C int    `json:"c,omitempty"`
}

// AlertLogs handles GET /api/monitors/{monitorID}/alerts and returns the
// newest alert delivery records of one owned monitor.
func (h *MonitorsHandler) AlertLogs(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "monitorID")
	if _, ok := h.lookupVisible(r, id); !ok {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "monitor not found"})
		return
	}
	writeJSON(w, http.StatusOK, h.st.ListAlertLogs(id, 100))
}

// PublicStatus serves the token-or-slug scoped public payload.
func (h *MonitorsHandler) PublicStatus(w http.ResponseWriter, r *http.Request) {
	token := chi.URLParam(r, "token")
	m, found := h.st.FindMonitorByToken(token)
	if !found {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
		return
	}
	payload := publicStatus{
		Name:         m.Name,
		IntervalSec:  services.MonitorInterval(m.IntervalSec),
		PublicTitle:  m.PublicTitle,
		PublicIcon:   m.PublicIcon,
		PublicSlug:   m.PublicSlug,
		PublicTheme:  m.PublicTheme,
		Announcement: m.Announcement,
		Targets:      []publicTarget{},
	}
	dayAgo := time.Now().Add(-24 * time.Hour).UnixMilli()
	for _, t := range m.Targets {
		list := h.hb.Recent(m.ID, t.ID, 0)
		pt := publicTarget{Name: t.Name, State: "unknown", Bars: []publicBar{}}
		pt.Link = linkURL(t)
		if len(list) > 0 {
			last := list[len(list)-1]
			pt.State = last.S
			pt.LatencyMs = last.M
			pt.HTTPCode = last.C
			pt.Error = publicErrorText(last.E)
			pt.Uptime24h = services.UptimePct(filterSince(list, dayAgo))
			raw := services.Downsample(list, barsPerTarget)
			pt.Bars = make([]publicBar, len(raw))
			for i, b := range raw {
				pt.Bars[i] = publicBar{T: b.T, S: b.S, M: b.M, C: b.C}
			}
			if last.T > payload.UpdatedAt {
				payload.UpdatedAt = last.T
			}
		}
		payload.Targets = append(payload.Targets, pt)
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, http.StatusOK, payload)
}

// linkURL returns the browsable address for link-enabled http targets.
func linkURL(t models.MonitorTarget) string {
	if !t.LinkEnabled || t.Type == "tcp" || t.Type == "icmp" {
		return ""
	}
	u := strings.TrimSpace(t.URL)
	if u == "" {
		return ""
	}
	if !strings.Contains(u, "://") {
		u = "https://" + u
	}
	return u
}

// publicErrorText shortens transport failures for the unauthenticated page.
func publicErrorText(e string) string {
	e = strings.TrimSpace(e)
	if e == "" {
		return ""
	}
	if i := strings.LastIndex(e, ":"); i > 0 {
		candidate := strings.TrimSpace(e[:i])
		if candidate != "" {
			e = candidate
		}
	}
	return e
}
