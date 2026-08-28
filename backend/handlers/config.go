package handlers

import (
	"fmt"
	"net/http"
	"strings"

	"tunnel-manager/models"
	"tunnel-manager/store"
)

// ConfigHandler handles configuration-related requests
type ConfigHandler struct {
	store *store.Store
}

// NewConfigHandler creates a new ConfigHandler
func NewConfigHandler(s *store.Store) *ConfigHandler {
	return &ConfigHandler{store: s}
}

// resolveUserID returns the account whose preferences apply to this request.
// API-key access (no individual account) falls back to the administrator.
func (h *ConfigHandler) resolveUserID(r *http.Request) string {
	if user := SessionUser(r); user != nil && !user.IsAPIKey() {
		return user.ID
	}
	return h.store.AdminUserID()
}

// GetConfig returns the current configuration (sanitized): global branding and
// presets merged with the requesting user's selections.
func (h *ConfigHandler) GetConfig(w http.ResponseWriter, r *http.Request) {
	cfg := h.store.GetConfig()
	prefs := h.store.GetUserPrefs(h.resolveUserID(r))
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"tunnel_id":        prefs.TunnelID,
		"tunnel_name":      prefs.TunnelName,
		"service_url":      prefs.ServiceURL,
		"preferred_cname":  cfg.PreferredCNAME,
		"cname_presets":    cfg.CNAMEPresets,
		"site_name":        cfg.SiteName,
		"site_description": cfg.SiteDescription,
		"site_icon":        cfg.SiteIcon,
		"landing_enabled":  cfg.LandingEnabled,
	})
}

// GetSiteSettings returns public-facing site branding without authentication.
func (h *ConfigHandler) GetSiteSettings(w http.ResponseWriter, r *http.Request) {
	cfg := h.store.GetConfig()
	writeJSON(w, http.StatusOK, models.SiteSettings{
		Name:           cfg.SiteName,
		Description:    cfg.SiteDescription,
		Icon:           cfg.SiteIcon,
		LandingEnabled: cfg.LandingEnabled,
	})
}

// SetTunnelSelection sets the active tunnel and its display name.
func (h *ConfigHandler) SetTunnelSelection(w http.ResponseWriter, r *http.Request) {
	var req models.SetTunnelRequest
	if err := readJSON(r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	if req.ID == "" {
		req.ID = req.Value
	}
	req.ID = strings.TrimSpace(req.ID)
	req.Name = strings.TrimSpace(req.Name)
	if err := h.store.SetUserTunnelSelection(h.resolveUserID(r), req.ID, req.Name); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to save tunnel selection"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "tunnel_id": req.ID, "tunnel_name": req.Name})
}

// SetSiteSettings updates public-facing site branding.
func (h *ConfigHandler) SetSiteSettings(w http.ResponseWriter, r *http.Request) {
	var req models.SetSiteSettingsRequest
	if err := readJSON(r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	req.Name = strings.TrimSpace(req.Name)
	req.Description = strings.TrimSpace(req.Description)
	req.Icon = strings.TrimSpace(req.Icon)
	if err := validateSiteSettings(req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	if err := h.store.SetSiteSettings(req.Name, req.Description, req.Icon, req.LandingEnabled); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to save site settings"})
		return
	}
	writeJSON(w, http.StatusOK, models.SiteSettings{Name: req.Name, Description: req.Description, Icon: req.Icon, LandingEnabled: req.LandingEnabled})
}

// SetCNAMEPresets replaces the reusable preferred CNAME options.
func (h *ConfigHandler) SetCNAMEPresets(w http.ResponseWriter, r *http.Request) {
	var req models.SetCNAMEPresetsRequest
	if err := readJSON(r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	items, err := normalizeCNAMEPresets(req.Items)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	if err := h.store.SetCNAMEPresets(items); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to save CNAME presets"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"status": "ok", "cname_presets": items})
}

func validateSiteSettings(req models.SetSiteSettingsRequest) error {
	if req.Name == "" {
		return fmt.Errorf("site name is required")
	}
	if len([]rune(req.Name)) > 60 {
		return fmt.Errorf("site name is too long")
	}
	if len([]rune(req.Description)) > 160 {
		return fmt.Errorf("site description is too long")
	}
	if len(req.Icon) > 768*1024 {
		return fmt.Errorf("site icon is too large")
	}
	if req.Icon != "" && !strings.HasPrefix(req.Icon, "/") && !strings.HasPrefix(req.Icon, "https://") && !strings.HasPrefix(req.Icon, "http://") && !strings.HasPrefix(req.Icon, "data:image/") {
		return fmt.Errorf("site icon must be an image URL or uploaded image")
	}
	return nil
}

func normalizeCNAMEPresets(items []models.CNAMEPreset) ([]models.CNAMEPreset, error) {
	if len(items) == 0 {
		return nil, fmt.Errorf("at least one CNAME preset is required")
	}
	if len(items) > 20 {
		return nil, fmt.Errorf("at most 20 CNAME presets are allowed")
	}
	normalized := make([]models.CNAMEPreset, 0, len(items))
	seen := make(map[string]struct{}, len(items))
	for _, item := range items {
		item.Name = strings.TrimSpace(item.Name)
		item.Value = strings.TrimSuffix(strings.ToLower(strings.TrimSpace(item.Value)), ".")
		if item.Name == "" || item.Value == "" {
			return nil, fmt.Errorf("preset name and CNAME are required")
		}
		if len([]rune(item.Name)) > 40 || len(item.Value) > 253 || strings.ContainsAny(item.Value, " /:@") || !strings.Contains(item.Value, ".") {
			return nil, fmt.Errorf("invalid CNAME preset: %s", item.Name)
		}
		if _, exists := seen[item.Value]; exists {
			return nil, fmt.Errorf("duplicate CNAME: %s", item.Value)
		}
		seen[item.Value] = struct{}{}
		normalized = append(normalized, item)
	}
	return normalized, nil
}

// SetServiceURL sets the forwarding service URL
func (h *ConfigHandler) SetServiceURL(w http.ResponseWriter, r *http.Request) {
	var req models.SetValueRequest
	if err := readJSON(r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	if req.Value == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "value is required"})
		return
	}
	if err := h.store.SetUserServiceURL(h.resolveUserID(r), req.Value); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to save service URL"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "service_url": req.Value})
}

// SetPreferredCNAME sets the global preferred CNAME
func (h *ConfigHandler) SetPreferredCNAME(w http.ResponseWriter, r *http.Request) {
	var req models.SetValueRequest
	if err := readJSON(r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	if req.Value == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "value is required"})
		return
	}
	h.store.SetPreferredCNAME(req.Value)
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "preferred_cname": req.Value})
}
