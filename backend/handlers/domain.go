package handlers

import (
	"fmt"
	"net/http"

	"tunnel-manager/models"
	"tunnel-manager/services"
)

// DomainHandler handles domain binding and fallback origin requests
type DomainHandler struct {
	svc *services.DomainService
}

// NewDomainHandler creates a new DomainHandler
func NewDomainHandler(svc *services.DomainService) *DomainHandler {
	return &DomainHandler{svc: svc}
}

// BindDomain performs the full domain binding flow
func (h *DomainHandler) BindDomain(w http.ResponseWriter, r *http.Request) {
	var req models.BindRequest
	if err := readJSON(r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	mode, err := services.NormalizeBindingMode(req.Mode)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	if req.MainDomain == "" || (mode == services.BindingModePreferred && req.AuxDomain == "") {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "main_domain is required; aux_domain is also required in preferred mode"})
		return
	}

	mode, preferredCNAME, err := h.svc.ForUser(sessionUID(r)).BindDomainWithConfiguredService(mode, req.MainDomain, req.AuxDomain, req.PreferredCNAME)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"status":          "ok",
		"mode":            mode,
		"message":         fmt.Sprintf("Domain binding complete! Access: https://%s", req.MainDomain),
		"main_domain":     req.MainDomain,
		"aux_domain":      req.AuxDomain,
		"preferred_cname": preferredCNAME,
	})
}

// BindDomainsBatch binds domain pairs sequentially and returns every result.
func (h *DomainHandler) BindDomainsBatch(w http.ResponseWriter, r *http.Request) {
	var req models.BatchBindRequest
	if err := readJSON(r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	if len(req.Items) == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "items cannot be empty"})
		return
	}

	results := make([]models.BatchBindResult, 0, len(req.Items))
	for _, item := range req.Items {
		mode, modeErr := services.NormalizeBindingMode(item.Mode)
		result := models.BatchBindResult{
			Mode:           mode,
			ServiceURL:     item.ServiceURL,
			PreferredCNAME: item.PreferredCNAME,
			MainDomain:     item.MainDomain,
			AuxDomain:      item.AuxDomain,
		}
		if modeErr != nil {
			result.Message = modeErr.Error()
			results = append(results, result)
			continue
		}
		if item.ServiceURL == "" || item.MainDomain == "" || (mode == services.BindingModePreferred && item.AuxDomain == "") {
			result.Message = "service_url and main_domain are required; aux_domain is also required in preferred mode"
			results = append(results, result)
			continue
		}

		mode, preferredCNAME, err := h.svc.ForUser(sessionUID(r)).BindDomainWithMode(mode, item.MainDomain, item.AuxDomain, item.ServiceURL, item.PreferredCNAME)
		if err != nil {
			result.Message = err.Error()
		} else {
			result.Success = true
			result.PreferredCNAME = preferredCNAME
			result.Message = fmt.Sprintf("Domain binding complete! Access: https://%s", item.MainDomain)
		}
		results = append(results, result)
	}

	writeJSON(w, http.StatusOK, models.BatchBindResponse{Results: results})
}

// SetFallbackOrigin sets the fallback origin for custom hostnames
func (h *DomainHandler) SetFallbackOrigin(w http.ResponseWriter, r *http.Request) {
	var req models.FallbackRequest
	if err := readJSON(r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	if req.Domain == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "domain is required"})
		return
	}

	if err := h.svc.ForUser(sessionUID(r)).SetFallbackOrigin(req.Domain); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"status":  "ok",
		"message": fmt.Sprintf("Fallback origin set to %s", req.Domain),
	})
}
