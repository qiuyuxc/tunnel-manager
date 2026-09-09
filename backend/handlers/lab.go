package handlers

import (
	"errors"
	"net/http"
	"strings"

	"tunnel-manager/auth"
	"tunnel-manager/models"
	"tunnel-manager/services"
	"tunnel-manager/store"
)

const labHuaweiSecretPurpose = "lab-huawei-secret-key"

// LabHandler exposes the experimental IP selector APIs.
type LabHandler struct {
	store         *store.Store
	runner        *services.LabIPSelectorRunner
	encryptionKey []byte
}

// NewLabHandler creates the experimental lab API handler.
func NewLabHandler(st *store.Store, runner *services.LabIPSelectorRunner, encryptionKey []byte) *LabHandler {
	return &LabHandler{store: st, runner: runner, encryptionKey: append([]byte(nil), encryptionKey...)}
}

// GetSettings returns the sanitized lab configuration.
func (h *LabHandler) GetSettings(w http.ResponseWriter, r *http.Request) {
	if !h.enabled() {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "experimental features are disabled"})
		return
	}
	settings := h.store.GetLabSettings()
	writeJSON(w, http.StatusOK, labSettingsView(settings))
}

// SaveSettings validates and persists lab configuration.
func (h *LabHandler) SaveSettings(w http.ResponseWriter, r *http.Request) {
	if !h.enabled() {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "experimental features are disabled"})
		return
	}
	var req models.SaveLabIPSelectorRequest
	if err := readAdminJSON(w, r, &req); err != nil {
		return
	}
	stored := h.store.GetLabSettings()
	settings := models.LabIPSelectorSettings{
		Host:         strings.TrimSpace(req.Host),
		SNI:          strings.TrimSpace(req.SNI),
		Path:         strings.TrimSpace(req.Path),
		Statuses:     strings.TrimSpace(req.Statuses),
		Timeout:      req.Timeout,
		Workers:      req.Workers,
		Top:          req.Top,
		IPTargets:    strings.TrimSpace(req.IPTargets),
		Schedule:     req.Schedule,
		IntervalMins: req.IntervalMins,
		UpdateDNS:    req.UpdateDNS,
		Zone:         strings.TrimSpace(req.Zone),
		ZoneID:       strings.TrimSpace(req.ZoneID),
		Record:       strings.TrimSpace(req.Record),
		TTL:          req.TTL,
		Endpoint:     strings.TrimSpace(req.Endpoint),
		AccessKey:    strings.TrimSpace(req.AccessKey),
		SecretKey:    stored.SecretKey,
	}
	if settings.SNI == "" {
		settings.SNI = settings.Host
	}
	if settings.Path == "" {
		settings.Path = "/"
	}
	if settings.Statuses == "" {
		settings.Statuses = "200"
	}
	if settings.Timeout == 0 {
		settings.Timeout = 2
	}
	if settings.Workers == 0 {
		settings.Workers = 32
	}
	if settings.Top == 0 {
		settings.Top = 10
	}
	if settings.IntervalMins == 0 {
		settings.IntervalMins = 30
	}
	if settings.TTL == 0 {
		settings.TTL = 300
	}
	if settings.Endpoint == "" {
		settings.Endpoint = "https://dns.myhuaweicloud.com"
	}
	if err := validateLabSettings(settings, req.SecretKey != "", stored.SecretKey != ""); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	if req.SecretKey != "" {
		encrypted, err := auth.EncryptSecret(h.encryptionKey, labHuaweiSecretPurpose, []byte(req.SecretKey))
		if err != nil {
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "加密华为云 Secret Key 失败"})
			return
		}
		settings.SecretKey = encrypted
	}
	if err := h.store.SetLabSettings(settings); err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "保存实验功能设置失败"})
		return
	}
	writeJSON(w, http.StatusOK, labSettingsView(settings))
}

// GetStatus returns live state and bounded history.
func (h *LabHandler) GetStatus(w http.ResponseWriter, r *http.Request) {
	if !h.enabled() {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "experimental features are disabled"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"status": h.runner.Status(),
		"runs":   h.store.GetLabRuns(20),
	})
}

// Run starts one asynchronous task.
func (h *LabHandler) Run(w http.ResponseWriter, r *http.Request) {
	if !h.enabled() {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "experimental features are disabled"})
		return
	}
	started := h.runner.Trigger()
	if !started {
		writeJSON(w, http.StatusConflict, map[string]string{"error": "task already running"})
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]bool{"started": true})
}

func (h *LabHandler) enabled() bool {
	return h.store.GetAppSettings().ExperimentalFeatures
}

func labSettingsView(settings models.LabIPSelectorSettings) models.LabIPSelectorView {
	return models.LabIPSelectorView{
		Host:         settings.Host,
		SNI:          settings.SNI,
		Path:         settings.Path,
		Statuses:     settings.Statuses,
		Timeout:      settings.Timeout,
		Workers:      settings.Workers,
		Top:          settings.Top,
		IPTargets:    settings.IPTargets,
		Schedule:     settings.Schedule,
		IntervalMins: settings.IntervalMins,
		UpdateDNS:    settings.UpdateDNS,
		Zone:         settings.Zone,
		ZoneID:       settings.ZoneID,
		Record:       settings.Record,
		TTL:          settings.TTL,
		Endpoint:     settings.Endpoint,
		AccessKey:    settings.AccessKey,
		HasSecretKey: settings.SecretKey != "",
	}
}

func validateLabSettings(settings models.LabIPSelectorSettings, newSecret, storedSecret bool) error {
	if settings.Host == "" {
		return errString("Host is required")
	}
	if settings.Path == "" || !strings.HasPrefix(settings.Path, "/") {
		return errString("Path must start with /")
	}
	if _, err := services.ParseLabStatuses(settings.Statuses); err != nil {
		return err
	}
	if _, err := services.ParseLabIPTargets(settings.IPTargets); err != nil {
		return err
	}
	if settings.Timeout < 1 || settings.Timeout > 15 {
		return errString("Timeout must be between 1 and 15 seconds")
	}
	if settings.Workers < 1 || settings.Workers > 256 {
		return errString("Workers must be between 1 and 256")
	}
	if settings.Top < 1 || settings.Top > 100 {
		return errString("Top must be between 1 and 100")
	}
	if settings.Schedule && (settings.IntervalMins < 5 || settings.IntervalMins > 1440) {
		return errString("Schedule interval must be between 5 and 1440 minutes")
	}
	if settings.TTL < 60 || settings.TTL > 86400 {
		return errString("DNS TTL must be between 60 and 86400")
	}
	if settings.Endpoint != "" && !strings.HasPrefix(settings.Endpoint, "https://") {
		return errString("Huawei Cloud endpoint must use HTTPS")
	}
	if settings.UpdateDNS {
		if settings.Zone == "" && settings.ZoneID == "" {
			return errString("DNS update requires a zone or zone ID")
		}
		if settings.Record == "" {
			return errString("DNS update requires a record")
		}
		if settings.AccessKey == "" || (!newSecret && !storedSecret) {
			return errString("DNS update requires Huawei Cloud access and secret keys")
		}
	}
	return nil
}

func errString(message string) error {
	return errors.New(message)
}
