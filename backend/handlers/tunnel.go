package handlers

import (
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"tunnel-manager/models"
	"tunnel-manager/services"
	"tunnel-manager/store"
)

// TunnelHandler handles tunnel-related requests
type TunnelHandler struct {
	cf    *services.CloudflareClient
	store *store.Store
}

// NewTunnelHandler creates a new TunnelHandler
func NewTunnelHandler(cf *services.CloudflareClient, s *store.Store) *TunnelHandler {
	return &TunnelHandler{cf: cf, store: s}
}

// ListTunnels returns all Cloudflare tunnels
func (h *TunnelHandler) ListTunnels(w http.ResponseWriter, r *http.Request) {
	tunnels, err := h.cf.ListTunnels()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, tunnels)
}

// CreateTunnel creates a tunnel and returns the connector token for cloudflared.
func (h *TunnelHandler) CreateTunnel(w http.ResponseWriter, r *http.Request) {
	var req models.CreateTunnelRequest
	if err := readJSON(r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "隧道名称不能为空"})
		return
	}

	tunnel, err := h.cf.CreateTunnel(req.Name)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	response := models.CreateTunnelResponse{ID: tunnel.ID, Name: tunnel.Name}
	token, err := h.cf.GetTunnelToken(tunnel.ID)
	if err != nil {
		response.Warning = fmt.Sprintf("隧道已创建，但获取连接令牌失败：%s", err.Error())
	} else {
		response.Token = token
		response.RunCommand = "cloudflared tunnel run --token " + token
	}
	writeJSON(w, http.StatusCreated, response)
}

// DeleteTunnel removes a tunnel and clears it from the stored selection.
func (h *TunnelHandler) DeleteTunnel(w http.ResponseWriter, r *http.Request) {
	tunnelID := chi.URLParam(r, "tunnelID")
	if tunnelID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "tunnel_id is required"})
		return
	}

	if err := h.cf.DeleteTunnel(tunnelID); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	if err := h.store.ClearTunnelSelectionIfUsed(tunnelID); err != nil {
		log.Printf("clear deleted tunnel selection: %v", err)
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "message": "tunnel deleted"})
}

// GetTunnelDetail returns tunnel details including ingress rules
func (h *TunnelHandler) GetTunnelDetail(w http.ResponseWriter, r *http.Request) {
	tunnelID := chi.URLParam(r, "tunnelID")

	tunnels, err := h.cf.ListTunnels()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	var tunnelName, tunnelStatus string
	for _, t := range tunnels {
		if t.ID == tunnelID {
			tunnelName = t.Name
			tunnelStatus = t.Status
			break
		}
	}
	if tunnelName == "" {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "tunnel not found"})
		return
	}

	cfg, err := h.cf.GetTunnelConfig(tunnelID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"id":      tunnelID,
		"name":    tunnelName,
		"status":  tunnelStatus,
		"ingress": cfg.Result.Config.Ingress,
	})
}

// ListZones returns all Cloudflare zones
func (h *TunnelHandler) ListZones(w http.ResponseWriter, r *http.Request) {
	zones, err := h.cf.ListZones()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, zones)
}

// IngressRuleRequest is the request body for adding/updating an ingress rule
type IngressRuleRequest struct {
	Hostname string `json:"hostname"`
	Service  string `json:"service"`
}

// AddIngressRule adds a new ingress rule to the tunnel
func (h *TunnelHandler) AddIngressRule(w http.ResponseWriter, r *http.Request) {
	tunnelID := chi.URLParam(r, "tunnelID")

	var req IngressRuleRequest
	if err := readJSON(r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	if req.Hostname == "" || req.Service == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "hostname and service are required"})
		return
	}

	cfg, err := h.cf.GetTunnelConfig(tunnelID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("get tunnel config: %s", err.Error())})
		return
	}

	ingress := cfg.Result.Config.Ingress
	// Check duplicate
	for _, rule := range ingress {
		if rule.Hostname == req.Hostname {
			writeJSON(w, http.StatusConflict, map[string]string{"error": "hostname already exists, use PUT to update"})
			return
		}
	}

	// Insert before catch-all (last rule)
	if len(ingress) > 0 {
		catchall := ingress[len(ingress)-1]
		ingress = ingress[:len(ingress)-1]
		ingress = append(ingress, models.IngressRule{Hostname: req.Hostname, Service: req.Service})
		ingress = append(ingress, catchall)
	} else {
		ingress = []models.IngressRule{
			{Hostname: req.Hostname, Service: req.Service},
			{Service: "http_status:404"},
		}
	}
	cfg.Result.Config.Ingress = ingress

	if err := h.cf.UpdateTunnelConfig(tunnelID, map[string]interface{}{"config": cfg.Result.Config}); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("update tunnel config: %s", err.Error())})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "message": "route added"})
}

// UpdateIngressRule updates an existing ingress rule by hostname
func (h *TunnelHandler) UpdateIngressRule(w http.ResponseWriter, r *http.Request) {
	tunnelID := chi.URLParam(r, "tunnelID")

	var req struct {
		OldHostname string `json:"old_hostname"`
		Hostname    string `json:"hostname"`
		Service     string `json:"service"`
	}
	if err := readJSON(r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	if req.OldHostname == "" || req.Hostname == "" || req.Service == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "old_hostname, hostname and service are required"})
		return
	}

	cfg, err := h.cf.GetTunnelConfig(tunnelID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("get tunnel config: %s", err.Error())})
		return
	}

	found := false
	for i, rule := range cfg.Result.Config.Ingress {
		if rule.Hostname == req.OldHostname {
			cfg.Result.Config.Ingress[i] = models.IngressRule{Hostname: req.Hostname, Service: req.Service}
			found = true
			break
		}
	}
	if !found {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "route not found"})
		return
	}

	if err := h.cf.UpdateTunnelConfig(tunnelID, map[string]interface{}{"config": cfg.Result.Config}); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("update tunnel config: %s", err.Error())})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "message": "route updated"})
}

// DeleteIngressRule removes a route by hostname and optionally its DNS record.
func (h *TunnelHandler) DeleteIngressRule(w http.ResponseWriter, r *http.Request) {
	tunnelID := chi.URLParam(r, "tunnelID")

	var req models.DeleteIngressRequest
	if err := readJSON(r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	req.Hostname = strings.TrimSuffix(strings.TrimSpace(req.Hostname), ".")
	if req.Hostname == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "hostname is required"})
		return
	}

	cfg, err := h.cf.GetTunnelConfig(tunnelID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("get tunnel config: %s", err.Error())})
		return
	}

	remaining := make([]models.IngressRule, 0, len(cfg.Result.Config.Ingress))
	found := false
	for _, rule := range cfg.Result.Config.Ingress {
		if rule.Hostname == req.Hostname {
			found = true
			continue
		}
		remaining = append(remaining, rule)
	}
	if !found {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "route not found"})
		return
	}
	// Cloudflare requires the ingress list to end with a rule that has no hostname.
	if len(remaining) == 0 {
		remaining = []models.IngressRule{{Service: "http_status:404"}}
	}
	cfg.Result.Config.Ingress = remaining

	if err := h.cf.UpdateTunnelConfig(tunnelID, map[string]interface{}{"config": cfg.Result.Config}); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("update tunnel config: %s", err.Error())})
		return
	}

	response := map[string]interface{}{"status": "ok", "message": "route deleted"}
	if req.DeleteDNS {
		// The route is already gone, so a DNS failure is reported as a warning rather than an error.
		deleted, err := h.deleteHostnameDNS(req.Hostname)
		response["dns_deleted"] = deleted
		if err != nil {
			response["dns_warning"] = err.Error()
		}
	}
	writeJSON(w, http.StatusOK, response)
}

// deleteHostnameDNS removes address records that resolve exactly the given hostname.
func (h *TunnelHandler) deleteHostnameDNS(hostname string) (int, error) {
	zoneID, err := h.cf.GetZoneIDByHostname(hostname)
	if err != nil {
		return 0, err
	}
	records, err := h.cf.ListDNSRecords(zoneID, "", hostname)
	if err != nil {
		return 0, err
	}
	deleted := 0
	for _, record := range records {
		switch record.Type {
		case "CNAME", "A", "AAAA":
		default:
			continue
		}
		if err := h.cf.DeleteDNSRecord(zoneID, record.ID); err != nil {
			return deleted, err
		}
		deleted++
	}
	return deleted, nil
}
