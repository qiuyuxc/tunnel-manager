package services

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"tunnel-manager/models"
)

// CloudflareClient wraps Cloudflare API calls
type CloudflareClient struct {
	apiToken   string
	accountID  string
	baseURL    string
	httpClient *http.Client
	oauth      *CloudflareOAuth
	store      interface {
		ActiveCFConnection(string) (models.CFConnection, bool)
	}
	userID     string
	pinnedConn *models.CFConnection
}

// NewCloudflareClient creates a new Cloudflare API client
func NewCloudflareClient(apiToken, accountID string) *CloudflareClient {
	return &CloudflareClient{
		apiToken:   apiToken,
		accountID:  accountID,
		baseURL:    "https://api.cloudflare.com/client/v4",
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

// SetOAuth enables OAuth credentials while preserving static credentials as fallback.
func (c *CloudflareClient) SetOAuth(oauth *CloudflareOAuth) {
	c.oauth = oauth
}

// SetSessionStore wires the store used to resolve per-user connections.
func (c *CloudflareClient) SetSessionStore(st interface {
	ActiveCFConnection(string) (models.CFConnection, bool)
}) {
	c.store = st
}

// ForUser returns a shallow client bound to one account's active connection.
func (c *CloudflareClient) ForUser(userID string) *CloudflareClient {
	clone := *c
	clone.userID = userID
	clone.pinnedConn = nil
	return &clone
}

// WithConnection returns a client that uses exactly this connection, even
// before it becomes the account's active one.
func (c *CloudflareClient) WithConnection(conn models.CFConnection) *CloudflareClient {
	clone := *c
	clone.userID = conn.UserID
	clone.pinnedConn = &conn
	return &clone
}

// DefaultAccountID returns the account configured through the environment.
func (c *CloudflareClient) DefaultAccountID() string {
	return c.accountID
}

// HasStaticCredentials reports whether legacy environment credentials are usable.
func (c *CloudflareClient) HasStaticCredentials() bool {
	return c.apiToken != "" && c.accountID != ""
}

func (c *CloudflareClient) accessToken() (string, error) {
	if c.oauth != nil && c.pinnedConn != nil {
		return c.oauth.AccessTokenFor(*c.pinnedConn)
	}
	if c.userID != "" && c.store != nil {
		if conn, ok := c.store.ActiveCFConnection(c.userID); ok && conn.HasToken() {
			return c.oauth.AccessTokenFor(conn)
		}
		if c.apiToken != "" {
			return c.apiToken, nil
		}
		return "", fmt.Errorf("尚未授权 Cloudflare 账户，请先在账户页完成授权")
	}
	if c.oauth != nil && c.oauth.Connected() {
		return c.oauth.AccessToken()
	}
	if c.apiToken == "" {
		return "", fmt.Errorf("Cloudflare 未连接，请先完成 OAuth 授权")
	}
	return c.apiToken, nil
}

func (c *CloudflareClient) currentAccountID() (string, error) {
	if c.store != nil && c.userID != "" {
		if conn, ok := c.store.ActiveCFConnection(c.userID); ok && conn.AccountID != "" {
			return conn.AccountID, nil
		}
	}
	if c.oauth != nil && c.oauth.Connected() {
		if accountID := c.oauth.AccountID(); accountID != "" {
			return accountID, nil
		}
	}
	if c.accountID == "" {
		return "", fmt.Errorf("Cloudflare 账户未选择")
	}
	return c.accountID, nil
}

func (c *CloudflareClient) newRequest(method, path string, body io.Reader) (*http.Request, error) {
	token, err := c.accessToken()
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest(method, c.baseURL+path, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	return req, nil
}

func (c *CloudflareClient) do(req *http.Request, target interface{}) error {
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response failed: %w", err)
	}

	var apiResp models.CFAPIResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return fmt.Errorf("parse response failed: %w", err)
	}

	if !apiResp.Success {
		errMsg := "unknown error"
		if len(apiResp.Errors) > 0 {
			errMsg = apiResp.Errors[0].Message
		}
		return fmt.Errorf("cloudflare API error: %s", errMsg)
	}

	if target != nil {
		return json.Unmarshal(apiResp.Result, target)
	}

	return nil
}

// ListTunnels lists all Cloudflare tunnels
func (c *CloudflareClient) ListTunnels() ([]models.Tunnel, error) {
	accountID, err := c.currentAccountID()
	if err != nil {
		return nil, err
	}
	path := fmt.Sprintf("/accounts/%s/cfd_tunnel?is_deleted=false", accountID)
	req, err := c.newRequest("GET", path, nil)
	if err != nil {
		return nil, err
	}

	var tunnels []models.Tunnel
	if err := c.do(req, &tunnels); err != nil {
		return nil, err
	}
	return tunnels, nil
}

// CreateTunnel creates a remotely-managed tunnel so its ingress can be edited through this API.
func (c *CloudflareClient) CreateTunnel(name string) (*models.Tunnel, error) {
	accountID, err := c.currentAccountID()
	if err != nil {
		return nil, err
	}
	body, err := json.Marshal(map[string]string{"name": name, "config_src": "cloudflare"})
	if err != nil {
		return nil, err
	}
	path := fmt.Sprintf("/accounts/%s/cfd_tunnel", accountID)
	req, err := c.newRequest(http.MethodPost, path, strings.NewReader(string(body)))
	if err != nil {
		return nil, err
	}
	var tunnel models.Tunnel
	if err := c.do(req, &tunnel); err != nil {
		return nil, err
	}
	return &tunnel, nil
}

// DeleteTunnel removes a tunnel. Cloudflare rejects deletion while connections stay active.
func (c *CloudflareClient) DeleteTunnel(tunnelID string) error {
	accountID, err := c.currentAccountID()
	if err != nil {
		return err
	}
	path := fmt.Sprintf("/accounts/%s/cfd_tunnel/%s", accountID, tunnelID)
	req, err := c.newRequest(http.MethodDelete, path, nil)
	if err != nil {
		return err
	}
	return c.do(req, nil)
}

// GetTunnelToken returns the connector token cloudflared needs to run the tunnel.
func (c *CloudflareClient) GetTunnelToken(tunnelID string) (string, error) {
	accountID, err := c.currentAccountID()
	if err != nil {
		return "", err
	}
	path := fmt.Sprintf("/accounts/%s/cfd_tunnel/%s/token", accountID, tunnelID)
	req, err := c.newRequest(http.MethodGet, path, nil)
	if err != nil {
		return "", err
	}
	var token string
	if err := c.do(req, &token); err != nil {
		return "", err
	}
	return token, nil
}

// GetTunnelConfig fetches the current tunnel configuration
func (c *CloudflareClient) GetTunnelConfig(tunnelID string) (*models.TunnelConfigResponse, error) {
	accountID, err := c.currentAccountID()
	if err != nil {
		return nil, err
	}
	path := fmt.Sprintf("/accounts/%s/cfd_tunnel/%s/configurations", accountID, tunnelID)
	req, err := c.newRequest("GET", path, nil)
	if err != nil {
		return nil, err
	}

	var cfg models.TunnelConfigResponse
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response failed: %w", err)
	}

	if err := json.Unmarshal(body, &cfg); err != nil {
		return nil, fmt.Errorf("parse response failed: %w", err)
	}

	if !cfg.Success {
		errMsg := "unknown error"
		if len(cfg.Errors) > 0 {
			errMsg = cfg.Errors[0].Message
		}
		return nil, fmt.Errorf("Cloudflare API: %s", errMsg)
	}

	return &cfg, nil
}

// UpdateTunnelConfig updates the tunnel configuration
func (c *CloudflareClient) UpdateTunnelConfig(tunnelID string, config interface{}) error {
	accountID, err := c.currentAccountID()
	if err != nil {
		return err
	}
	path := fmt.Sprintf("/accounts/%s/cfd_tunnel/%s/configurations", accountID, tunnelID)
	body, err := json.Marshal(config)
	if err != nil {
		return err
	}

	req, err := c.newRequest("PUT", path, strings.NewReader(string(body)))
	if err != nil {
		return err
	}

	return c.do(req, nil)
}

// ListAccounts lists accounts available to the active credential.
func (c *CloudflareClient) ListAccounts() ([]models.Account, error) {
	req, err := c.newRequest("GET", "/accounts?per_page=100", nil)
	if err != nil {
		return nil, err
	}
	var accounts []models.Account
	if err := c.do(req, &accounts); err != nil {
		return nil, err
	}
	return accounts, nil
}

// GetZoneIDByHostname finds the zone ID for a given hostname
func (c *CloudflareClient) GetZoneIDByHostname(hostname string) (string, error) {
	hostname = strings.TrimSpace(strings.ToLower(strings.TrimSuffix(hostname, ".")))

	req, err := c.newRequest("GET", "/zones?status=active&per_page=1000", nil)
	if err != nil {
		return "", err
	}

	var zones []models.Zone
	if err := c.do(req, &zones); err != nil {
		return "", err
	}

	var bestMatch *models.Zone
	for _, z := range zones {
		zoneName := strings.TrimSpace(strings.ToLower(z.Name))
		if hostname == zoneName || strings.HasSuffix(hostname, "."+zoneName) {
			if bestMatch == nil || len(zoneName) > len(bestMatch.Name) {
				zCopy := z
				bestMatch = &zCopy
			}
		}
	}

	if bestMatch == nil {
		return "", fmt.Errorf("no zone found for hostname: %s", hostname)
	}
	return bestMatch.ID, nil
}

// UpsertDNSRecord creates or updates a DNS record
func (c *CloudflareClient) UpsertDNSRecord(zoneID, name, recordType, content string, proxied bool) error {
	// List existing records
	listURL := fmt.Sprintf("/zones/%s/dns_records?name=%s&type=%s", zoneID, url.QueryEscape(name), recordType)
	req, err := c.newRequest("GET", listURL, nil)
	if err != nil {
		return err
	}

	var records []models.DNSRecord
	if err := c.do(req, &records); err != nil {
		// If listing fails, try to create directly
		records = nil
	}

	payload := models.DNSRecord{
		Name:    name,
		Type:    recordType,
		Content: content,
		Proxied: proxied,
	}
	body, _ := json.Marshal(payload)

	if len(records) > 0 {
		// Update existing record
		updateURL := fmt.Sprintf("/zones/%s/dns_records/%s", zoneID, records[0].ID)
		req, err = c.newRequest("PUT", updateURL, strings.NewReader(string(body)))
	} else {
		// Create new record
		createURL := fmt.Sprintf("/zones/%s/dns_records", zoneID)
		req, err = c.newRequest("POST", createURL, strings.NewReader(string(body)))
	}

	if err != nil {
		return err
	}
	return c.do(req, nil)
}

// ListDNSRecords returns DNS records in a zone with optional type and name filters.
func (c *CloudflareClient) ListDNSRecords(zoneID, recordType, name string) ([]models.DNSRecord, error) {
	values := url.Values{}
	values.Set("per_page", "5000")
	if recordType != "" {
		values.Set("type", strings.ToUpper(recordType))
	}
	if name != "" {
		values.Set("name", name)
	}
	path := fmt.Sprintf("/zones/%s/dns_records?%s", zoneID, values.Encode())
	req, err := c.newRequest(http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	var records []models.DNSRecord
	if err := c.do(req, &records); err != nil {
		return nil, err
	}
	return records, nil
}

// CreateDNSRecord creates a DNS record and returns the Cloudflare representation.
func (c *CloudflareClient) CreateDNSRecord(zoneID string, payload models.DNSRecordRequest) (*models.DNSRecord, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	req, err := c.newRequest(http.MethodPost, fmt.Sprintf("/zones/%s/dns_records", zoneID), strings.NewReader(string(body)))
	if err != nil {
		return nil, err
	}
	var record models.DNSRecord
	if err := c.do(req, &record); err != nil {
		return nil, err
	}
	return &record, nil
}

// UpdateDNSRecord replaces an existing DNS record.
func (c *CloudflareClient) UpdateDNSRecord(zoneID, recordID string, payload models.DNSRecordRequest) (*models.DNSRecord, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	path := fmt.Sprintf("/zones/%s/dns_records/%s", zoneID, recordID)
	req, err := c.newRequest(http.MethodPut, path, strings.NewReader(string(body)))
	if err != nil {
		return nil, err
	}
	var record models.DNSRecord
	if err := c.do(req, &record); err != nil {
		return nil, err
	}
	return &record, nil
}

// DeleteDNSRecord removes a DNS record by ID.
func (c *CloudflareClient) DeleteDNSRecord(zoneID, recordID string) error {
	path := fmt.Sprintf("/zones/%s/dns_records/%s", zoneID, recordID)
	req, err := c.newRequest(http.MethodDelete, path, nil)
	if err != nil {
		return err
	}
	return c.do(req, nil)
}

// SetFallbackOrigin sets the fallback origin for custom hostnames
func (c *CloudflareClient) SetFallbackOrigin(zoneID, origin string) error {
	path := fmt.Sprintf("/zones/%s/custom_hostnames/fallback_origin", zoneID)
	payload := map[string]string{"origin": origin}
	body, _ := json.Marshal(payload)

	req, err := c.newRequest("PUT", path, strings.NewReader(string(body)))
	if err != nil {
		return err
	}
	return c.do(req, nil)
}

// UpsertCustomHostname creates or updates a SaaS custom hostname
func (c *CloudflareClient) UpsertCustomHostname(zoneID, hostname, originServer string) error {
	payload := map[string]interface{}{
		"hostname":             hostname,
		"custom_origin_server": originServer,
		"ssl": map[string]interface{}{
			"method": "http",
			"type":   "dv",
		},
	}
	body, _ := json.Marshal(payload)

	// Check if custom hostname already exists
	listURL := fmt.Sprintf("/zones/%s/custom_hostnames?hostname=%s", zoneID, url.QueryEscape(hostname))
	req, err := c.newRequest("GET", listURL, nil)
	if err != nil {
		return err
	}

	var existing []models.CustomHostname
	if err := c.do(req, &existing); err == nil && len(existing) > 0 {
		// Update existing
		updateURL := fmt.Sprintf("/zones/%s/custom_hostnames/%s", zoneID, existing[0].ID)
		req, err = c.newRequest("PATCH", updateURL, strings.NewReader(string(body)))
	} else {
		// Create new
		createURL := fmt.Sprintf("/zones/%s/custom_hostnames", zoneID)
		req, err = c.newRequest("POST", createURL, strings.NewReader(string(body)))
	}
	if err != nil {
		return err
	}
	return c.do(req, nil)
}

// ListZones lists all active zones
func (c *CloudflareClient) ListZones() ([]models.Zone, error) {
	req, err := c.newRequest("GET", "/zones?status=active&per_page=1000", nil)
	if err != nil {
		return nil, err
	}

	var zones []models.Zone
	if err := c.do(req, &zones); err != nil {
		return nil, err
	}
	return zones, nil
}
