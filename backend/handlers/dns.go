package handlers

import (
	"bytes"
	"encoding/json"
	"io"
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
	if !editableDNSRecord(w, r, zoneID, recordID) {
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
	if !editableDNSRecord(w, r, zoneID, recordID) {
		return
	}
	if err := UserCF(r).DeleteDNSRecord(zoneID, recordID); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func editableDNSRecord(w http.ResponseWriter, r *http.Request, zoneID, recordID string) bool {
	record, err := UserCF(r).GetDNSRecord(zoneID, recordID)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return false
	}
	if !models.EditableDNSRecordType(record.Type) {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "unsupported DNS record type (read-only)"})
		return false
	}
	return true
}

func readDNSRecordRequest(w http.ResponseWriter, r *http.Request, zoneID string) (models.DNSRecordRequest, bool) {
	payload := models.DNSRecordRequest{TTL: 1}
	if zoneID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "zone_id is required"})
		return payload, false
	}
	defer r.Body.Close()
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 64<<10))
	var raw json.RawMessage
	if err := decoder.Decode(&raw); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return payload, false
	}
	var fields map[string]json.RawMessage
	if json.Unmarshal(raw, &fields) != nil || fields == nil || decoder.Decode(new(any)) != io.EOF {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return payload, false
	}
	for _, key := range []string{"ttl", "proxied"} {
		if value, exists := fields[key]; exists && bytes.Equal(bytes.TrimSpace(value), []byte("null")) {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": key + " must not be null"})
			return payload, false
		}
	}
	decoder = json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&payload); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request fields: " + err.Error()})
		return payload, false
	}
	if err := models.ValidateDNSRecordRequest(&payload); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return payload, false
	}
	return payload, true
}
