package handlers

import (
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"tunnel-manager/models"
	"tunnel-manager/services"
)

type DNSHandler struct{ cf *services.CloudflareClient }

func NewDNSHandler(cf *services.CloudflareClient) *DNSHandler { return &DNSHandler{cf: cf} }

func (h *DNSHandler) List(w http.ResponseWriter, r *http.Request) {
	zoneID := strings.TrimSpace(chi.URLParam(r, "zoneID"))
	if zoneID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "zone_id is required"})
		return
	}
	records, err := UserCF(r).ListDNSRecords(zoneID, r.URL.Query().Get("type"), r.URL.Query().Get("name"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	if records == nil {
		records = []models.DNSRecord{}
	}
	writeJSON(w, http.StatusOK, records)
}

func (h *DNSHandler) Create(w http.ResponseWriter, r *http.Request) {
	zoneID := strings.TrimSpace(chi.URLParam(r, "zoneID"))
	payload, ok := readDNSRecordRequest(w, r, zoneID)
	if !ok {
		return
	}
	record, err := UserCF(r).CreateDNSRecord(zoneID, payload)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusCreated, record)
}

func (h *DNSHandler) Update(w http.ResponseWriter, r *http.Request) {
	zoneID := strings.TrimSpace(chi.URLParam(r, "zoneID"))
	recordID := strings.TrimSpace(chi.URLParam(r, "recordID"))
	if recordID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "record_id is required"})
		return
	}
	payload, ok := readDNSRecordRequest(w, r, zoneID)
	if !ok {
		return
	}
	record, err := UserCF(r).UpdateDNSRecord(zoneID, recordID, payload)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, record)
}

func (h *DNSHandler) Delete(w http.ResponseWriter, r *http.Request) {
	zoneID := strings.TrimSpace(chi.URLParam(r, "zoneID"))
	recordID := strings.TrimSpace(chi.URLParam(r, "recordID"))
	if zoneID == "" || recordID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "zone_id and record_id are required"})
		return
	}
	if err := UserCF(r).DeleteDNSRecord(zoneID, recordID); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func readDNSRecordRequest(w http.ResponseWriter, r *http.Request, zoneID string) (models.DNSRecordRequest, bool) {
	var payload models.DNSRecordRequest
	if zoneID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "zone_id is required"})
		return payload, false
	}
	if err := readJSON(r, &payload); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return payload, false
	}
	payload.Name = strings.TrimSuffix(strings.TrimSpace(payload.Name), ".")
	payload.Type = strings.ToUpper(strings.TrimSpace(payload.Type))
	payload.Content = strings.TrimSpace(payload.Content)
	if payload.Name == "" || payload.Content == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "name and content are required"})
		return payload, false
	}
	allowed := map[string]bool{"A": true, "AAAA": true, "CNAME": true, "TXT": true, "MX": true}
	if !allowed[payload.Type] {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "unsupported DNS record type"})
		return payload, false
	}
	if payload.TTL == 0 {
		payload.TTL = 1
	}
	if payload.TTL != 1 && (payload.TTL < 60 || payload.TTL > 86400) {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "ttl must be automatic or between 60 and 86400 seconds"})
		return payload, false
	}
	if payload.Type == "MX" {
		if payload.Priority == nil || *payload.Priority < 0 || *payload.Priority > 65535 {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "MX priority must be between 0 and 65535"})
			return payload, false
		}
		payload.Proxied = false
	} else {
		payload.Priority = nil
		if payload.Type == "TXT" {
			payload.Proxied = false
		}
	}
	return payload, true
}
