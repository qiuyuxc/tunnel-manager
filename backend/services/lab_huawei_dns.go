package services

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"
)

type huaweiDNSClient struct {
	endpoint   string
	accessKey  string
	secretKey  string
	httpClient *http.Client
}

type huaweiRecordSet struct {
	ID      string   `json:"id"`
	Name    string   `json:"name"`
	Type    string   `json:"type"`
	TTL     int      `json:"ttl"`
	Records []string `json:"records"`
}

func newHuaweiDNSClient(endpoint, accessKey, secretKey string) *huaweiDNSClient {
	if strings.TrimSpace(endpoint) == "" {
		endpoint = "https://dns.myhuaweicloud.com"
	}
	return &huaweiDNSClient{
		endpoint:   strings.TrimRight(endpoint, "/"),
		accessKey:  accessKey,
		secretKey:  secretKey,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

func normalizeHuaweiFQDN(name string) string {
	return strings.TrimSuffix(strings.TrimSpace(name), ".") + "."
}

func (c *huaweiDNSClient) UpdateARecord(zoneID, zoneName, record string, ttl int, ips []string) error {
	if zoneID == "" && strings.TrimSpace(zoneName) == "" {
		return fmt.Errorf("a Huawei Cloud zone or zone ID is required")
	}
	if strings.TrimSpace(record) == "" {
		return fmt.Errorf("a DNS record is required")
	}
	if ttl <= 0 {
		ttl = 300
	}
	if len(ips) == 0 {
		return fmt.Errorf("at least one selected IP is required")
	}
	if zoneID == "" {
		foundZoneID, err := c.findZoneID(zoneName)
		if err != nil {
			return err
		}
		zoneID = foundZoneID
	}
	recordName := normalizeHuaweiFQDN(record)
	existing, err := c.findRecordSet(zoneID, recordName)
	if err != nil {
		return err
	}
	payload := map[string]interface{}{
		"name":    recordName,
		"type":    "A",
		"ttl":     ttl,
		"records": ips,
	}
	method := http.MethodPost
	path := fmt.Sprintf("/v2/zones/%s/recordsets", zoneID)
	if existing != nil {
		if sameStringSet(existing.Records, ips) && existing.TTL == ttl {
			return nil
		}
		method = http.MethodPut
		path += "/" + existing.ID
	}
	if _, err := c.request(method, path, nil, payload); err != nil {
		return err
	}
	return nil
}

func (c *huaweiDNSClient) findZoneID(zoneName string) (string, error) {
	query := url.Values{"name": []string{normalizeHuaweiFQDN(zoneName)}}
	body, err := c.request(http.MethodGet, "/v2/zones", query, nil)
	if err != nil {
		return "", err
	}
	var response struct {
		Zones []struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"zones"`
	}
	if err := json.Unmarshal(body, &response); err != nil {
		return "", fmt.Errorf("decode Huawei Cloud zones: %w", err)
	}
	for _, zone := range response.Zones {
		if zone.Name == normalizeHuaweiFQDN(zoneName) {
			return zone.ID, nil
		}
	}
	return "", fmt.Errorf("Huawei Cloud zone %s not found", zoneName)
}

func (c *huaweiDNSClient) findRecordSet(zoneID, recordName string) (*huaweiRecordSet, error) {
	query := url.Values{"type": []string{"A"}, "name": []string{recordName}}
	body, err := c.request(http.MethodGet, fmt.Sprintf("/v2/zones/%s/recordsets", zoneID), query, nil)
	if err != nil {
		return nil, err
	}
	var response struct {
		RecordSets []huaweiRecordSet `json:"recordsets"`
	}
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("decode Huawei Cloud recordsets: %w", err)
	}
	for _, recordSet := range response.RecordSets {
		if recordSet.Name == recordName && recordSet.Type == "A" {
			return &recordSet, nil
		}
	}
	return nil, nil
}

func (c *huaweiDNSClient) request(method, path string, query url.Values, payload interface{}) ([]byte, error) {
	var body []byte
	if payload != nil {
		encoded, err := json.Marshal(payload)
		if err != nil {
			return nil, fmt.Errorf("encode Huawei Cloud request: %w", err)
		}
		body = encoded
	}
	target := c.endpoint + path
	if len(query) > 0 {
		target += "?" + query.Encode()
	}
	request, err := http.NewRequest(method, target, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	if payload != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	response, err := c.doSigned(request, body)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	responseBody, err := io.ReadAll(io.LimitReader(response.Body, 1<<20))
	if err != nil {
		return nil, fmt.Errorf("read Huawei Cloud response: %w", err)
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return nil, fmt.Errorf("Huawei Cloud HTTP %d: %s", response.StatusCode, strings.TrimSpace(string(responseBody)))
	}
	return responseBody, nil
}

func (c *huaweiDNSClient) doSigned(request *http.Request, body []byte) (*http.Response, error) {
	sdkDate := time.Now().UTC().Format("20060102T150405Z")
	request.Header.Set("X-Sdk-Date", sdkDate)
	if request.Host == "" {
		request.Host = request.URL.Host
	}
	headers := map[string]string{
		"host":       request.Host,
		"x-sdk-date": sdkDate,
	}
	if contentType := request.Header.Get("Content-Type"); contentType != "" {
		headers["content-type"] = strings.ToLower(contentType)
	}
	names := make([]string, 0, len(headers))
	for name := range headers {
		names = append(names, name)
	}
	sort.Strings(names)
	var canonicalHeaders strings.Builder
	for _, name := range names {
		canonicalHeaders.WriteString(name)
		canonicalHeaders.WriteString(":")
		canonicalHeaders.WriteString(strings.TrimSpace(headers[name]))
		canonicalHeaders.WriteString("\n")
	}
	signedHeaders := strings.Join(names, ";")
	canonicalURI := request.URL.Path
	if !strings.HasSuffix(canonicalURI, "/") {
		canonicalURI += "/"
	}
	payloadHash := sha256.Sum256(body)
	canonicalRequest := strings.Join([]string{
		strings.ToUpper(request.Method),
		canonicalURI,
		canonicalQuery(request.URL.Query()),
		canonicalHeaders.String(),
		signedHeaders,
		hex.EncodeToString(payloadHash[:]),
	}, "\n")
	canonicalHash := sha256.Sum256([]byte(canonicalRequest))
	stringToSign := strings.Join([]string{
		"SDK-HMAC-SHA256",
		sdkDate,
		hex.EncodeToString(canonicalHash[:]),
	}, "\n")
	mac := hmac.New(sha256.New, []byte(c.secretKey))
	mac.Write([]byte(stringToSign))
	signature := hex.EncodeToString(mac.Sum(nil))
	request.Header.Set("Authorization", fmt.Sprintf(
		"SDK-HMAC-SHA256 Access=%s, SignedHeaders=%s, Signature=%s",
		c.accessKey, signedHeaders, signature,
	))
	return c.httpClient.Do(request)
}

func canonicalQuery(values url.Values) string {
	if len(values) == 0 {
		return ""
	}
	pairs := make([]string, 0, len(values))
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		items := append([]string(nil), values[key]...)
		sort.Strings(items)
		for _, value := range items {
			pairs = append(pairs, url.QueryEscape(key)+"="+url.QueryEscape(value))
		}
	}
	return strings.Join(pairs, "&")
}

func sameStringSet(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	leftCopy := append([]string(nil), left...)
	rightCopy := append([]string(nil), right...)
	sort.Strings(leftCopy)
	sort.Strings(rightCopy)
	for i := range leftCopy {
		if leftCopy[i] != rightCopy[i] {
			return false
		}
	}
	return true
}
