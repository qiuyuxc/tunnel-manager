package services

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"golang.org/x/net/publicsuffix"
	"tunnel-manager/models"
)

type LabDNSProvider interface {
	FindZone(context.Context, string) (models.Zone, error)
	EnsureTXT(context.Context, models.Zone, string, string) (bool, error)
}

type labCloudflareDNS struct {
	baseURL   string
	token     string
	accountID string
	client    *http.Client
}

func (client *CloudflareClient) LabDNSAvailability(ownerID string) models.LabDNSAvailability {
	unavailable := models.LabDNSAvailability{Message: "当前账户未绑定可用的 Cloudflare 账户，请手动添加 TXT，或先在账户设置中绑定并选择连接。"}
	if client == nil || ownerID == "" || client.userID != ownerID || client.store == nil {
		return unavailable
	}
	if connection, found := client.store.ActiveCFConnection(ownerID); found && connection.HasToken() {
		if connection.UserID != ownerID || connection.AccountID == "" || client.oauth == nil {
			return unavailable
		}
		name := connection.AccountName
		if name == "" {
			name = connection.Label
		}
		return models.LabDNSAvailability{Available: true, AccountName: name, Message: "使用当前绑定的 Cloudflare 账户；域名不在该账户或没有 DNS 写入权限时，请手动添加。"}
	}
	if client.adminUser() && client.HasStaticCredentials() {
		return models.LabDNSAvailability{Available: true, AccountName: "项目配置的 Cloudflare 账户", Message: "使用项目配置的 Cloudflare 凭据；仅可操作该账户有权限管理的域名。"}
	}
	return unavailable
}

func (client *CloudflareClient) LabDNSForUser(ownerID string) (LabDNSProvider, error) {
	availability := client.LabDNSAvailability(ownerID)
	if !availability.Available {
		return nil, errors.New(availability.Message)
	}
	token, accountID := client.apiToken, client.accountID
	if connection, found := client.store.ActiveCFConnection(ownerID); found && connection.HasToken() {
		if connection.UserID != ownerID || connection.AccountID == "" || client.oauth == nil {
			return nil, errors.New("Cloudflare 连接已变化，请刷新后重试或手动添加 TXT")
		}
		var err error
		token, err = client.oauth.AccessTokenFor(connection)
		if err != nil {
			return nil, errors.New("当前 Cloudflare 授权不可用，请重新授权或手动添加 TXT；不会自动切换其他账户")
		}
		accountID = connection.AccountID
	} else if !client.adminUser() || !client.HasStaticCredentials() {
		return nil, errors.New("Cloudflare 连接已解除，请手动添加 TXT")
	}
	httpClient := *client.httpClient
	httpClient.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	return &labCloudflareDNS{baseURL: client.baseURL, token: token, accountID: accountID, client: &httpClient}, nil
}

func (client *labCloudflareDNS) request(ctx context.Context, method, path string, payload, result interface{}) (int, error) {
	var body []byte
	if payload != nil {
		var err error
		body, err = json.Marshal(payload)
		if err != nil {
			return 0, err
		}
	}
	request, err := http.NewRequestWithContext(ctx, method, client.baseURL+path, bytes.NewReader(body))
	if err != nil {
		return 0, err
	}
	request.Header.Set("Authorization", "Bearer "+client.token)
	request.Header.Set("Content-Type", "application/json")
	response, err := client.client.Do(request)
	if err != nil {
		return 0, fmt.Errorf("Cloudflare DNS 请求失败：%w", err)
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return 0, fmt.Errorf("Cloudflare DNS 返回 HTTP %d，请检查当前账户的域名和 DNS 权限，或手动添加 TXT", response.StatusCode)
	}
	content, err := io.ReadAll(io.LimitReader(response.Body, (1<<20)+1))
	if err != nil || len(content) > 1<<20 {
		return 0, errors.New("Cloudflare DNS 响应无法读取或超过大小限制")
	}
	var envelope struct {
		Success bool            `json:"success"`
		Result  json.RawMessage `json:"result"`
		Info    struct {
			TotalPages int `json:"total_pages"`
		} `json:"result_info"`
	}
	if err := json.Unmarshal(content, &envelope); err != nil || !envelope.Success {
		return 0, errors.New("Cloudflare DNS 操作失败，请检查授权或手动添加 TXT")
	}
	if result != nil {
		if err := json.Unmarshal(envelope.Result, result); err != nil {
			return 0, errors.New("Cloudflare DNS 返回的数据格式不正确")
		}
	}
	return envelope.Info.TotalPages, nil
}

func (client *labCloudflareDNS) FindZone(ctx context.Context, domain string) (models.Zone, error) {
	domain, err := NormalizeLabHostname(domain)
	if err != nil {
		return models.Zone{}, err
	}
	root, err := publicsuffix.EffectiveTLDPlusOne(domain)
	if err != nil {
		return models.Zone{}, err
	}
	for candidate := domain; ; {
		query := url.Values{"name": {candidate}, "account.id": {client.accountID}, "status": {"active"}, "per_page": {"50"}}
		var zones []struct {
			models.Zone
			Account models.Account `json:"account"`
			Status  string         `json:"status"`
		}
		if _, err := client.request(ctx, http.MethodGet, "/zones?"+query.Encode(), nil, &zones); err != nil {
			return models.Zone{}, err
		}
		for _, zone := range zones {
			if zone.ID != "" && zone.Account.ID == client.accountID && zone.Status == "active" && strings.EqualFold(strings.TrimSuffix(zone.Name, "."), candidate) {
				return models.Zone{ID: zone.ID, Name: candidate}, nil
			}
		}
		if candidate == root {
			break
		}
		_, candidate, _ = strings.Cut(candidate, ".")
	}
	return models.Zone{}, errors.New("当前绑定的 Cloudflare 账户中没有可管理的该域名，请切换已绑定账户或手动添加 TXT")
}

func (client *labCloudflareDNS) EnsureTXT(ctx context.Context, zone models.Zone, name, token string) (bool, error) {
	if zone.ID == "" || !labDomainContains(zone.Name, name) || !strings.HasPrefix(name, "_tunnel-manager-verify.") || !strings.HasPrefix(token, "tunnel-manager=") {
		return false, errors.New("TXT 验证记录不属于已选区域")
	}
	path := "/zones/" + url.PathEscape(zone.ID) + "/dns_records"
	for page := 1; ; page++ {
		query := url.Values{"type": {"TXT"}, "name": {name}, "per_page": {"100"}, "page": {strconv.Itoa(page)}}
		var records []models.DNSRecord
		pages, err := client.request(ctx, http.MethodGet, path+"?"+query.Encode(), nil, &records)
		if err != nil {
			return false, err
		}
		for _, record := range records {
			if record.Type == "TXT" && strings.EqualFold(strings.TrimSuffix(record.Name, "."), name) && (record.Content == token || record.Content == strconv.Quote(token)) {
				return false, nil
			}
		}
		if page >= pages {
			break
		}
		if page >= 20 {
			return false, errors.New("同名 TXT 记录过多，请手动添加验证记录")
		}
	}
	payload := models.DNSRecordRequest{Type: "TXT", Name: name, Content: token, TTL: 60, Proxied: false}
	_, err := client.request(ctx, http.MethodPost, path, payload, nil)
	return err == nil, err
}
