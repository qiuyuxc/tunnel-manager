package services

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"strconv"
	"strings"
	"time"

	"tunnel-manager/models"
)

type AIToolCall struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	Function struct {
		Name      string `json:"name"`
		Arguments string `json:"arguments"`
	} `json:"function"`
}

type AIReply struct {
	Content   string       `json:"content"`
	ToolCalls []AIToolCall `json:"tool_calls"`
}

type AIClient struct{ Client *http.Client }

func AllowedAIAddress(address netip.Addr) bool {
	address = address.Unmap()
	if address.Zone() != "" {
		return false
	}
	if address.IsLoopback() {
		return true
	}
	if !address.IsGlobalUnicast() || address.IsLinkLocalUnicast() {
		return false
	}
	if address.Is6() && !address.IsPrivate() && !netip.MustParsePrefix("2000::/3").Contains(address) {
		return false
	}
	for _, blocked := range []string{"0.0.0.0/8", "192.0.0.0/24", "192.0.2.0/24", "198.18.0.0/15", "198.51.100.0/24", "203.0.113.0/24", "240.0.0.0/4", "2001::/23", "2002::/16", "64:ff9b::/96", "fd00:ec2::254/128"} {
		if netip.MustParsePrefix(blocked).Contains(address) {
			return false
		}
	}
	return true
}

func ValidateAIEndpoint(endpoint string) (string, error) {
	parsed, err := url.Parse(strings.TrimSpace(endpoint))
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Hostname() == "" || parsed.User != nil || parsed.RawQuery != "" || parsed.ForceQuery || parsed.Fragment != "" || len(endpoint) > 2048 {
		return "", errors.New("API 端点须为 HTTP 或 HTTPS 地址，不可包含密码、查询参数或片段")
	}
	if strings.HasSuffix(parsed.Host, ":") {
		return "", errors.New("API 端点端口须为 1–65535")
	}
	if parsed.Port() != "" {
		port, err := strconv.Atoi(parsed.Port())
		if err != nil || port < 1 || port > 65535 {
			return "", errors.New("API 端点端口须为 1–65535")
		}
	}
	if address, err := netip.ParseAddr(parsed.Hostname()); err == nil && !AllowedAIAddress(address) {
		return "", errors.New("API 端点不可指向未指定、链路本地、组播或保留地址")
	}
	return strings.TrimRight(parsed.String(), "/"), nil
}

func NewAIClient() *AIClient {
	transport := &http.Transport{
		TLSHandshakeTimeout:   10 * time.Second,
		ResponseHeaderTimeout: 70 * time.Second,
		IdleConnTimeout:       30 * time.Second,
		MaxIdleConns:          16,
		DialContext: func(ctx context.Context, network, address string) (net.Conn, error) {
			host, port, err := net.SplitHostPort(address)
			if err != nil {
				return nil, err
			}
			addresses, err := net.DefaultResolver.LookupNetIP(ctx, "ip", host)
			if err != nil || len(addresses) == 0 {
				return nil, errors.New("API 端点解析失败")
			}
			for _, resolved := range addresses {
				if !AllowedAIAddress(resolved) {
					return nil, errors.New("API 端点解析到不允许的地址")
				}
			}
			dialContext, cancel := context.WithTimeout(ctx, 10*time.Second)
			defer cancel()
			dialer := net.Dialer{Timeout: 10 * time.Second}
			var dialErr error
			for _, resolved := range addresses {
				connection, err := dialer.DialContext(dialContext, network, net.JoinHostPort(resolved.String(), port))
				if err == nil {
					return connection, nil
				}
				dialErr = err
			}
			return nil, dialErr
		},
	}
	return &AIClient{Client: &http.Client{Timeout: 75 * time.Second, Transport: transport, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}}
}

func (client *AIClient) Complete(ctx context.Context, connection models.AIConnection, key string, messages []models.AIMessage, tools []map[string]any) (AIReply, error) {
	endpoint, err := ValidateAIEndpoint(connection.Endpoint)
	if err != nil {
		return AIReply{}, err
	}
	if !strings.HasSuffix(endpoint, "/chat/completions") {
		endpoint += "/chat/completions"
	}
	request := map[string]any{"model": connection.Model, "messages": messages, "stream": false}
	if len(tools) > 0 {
		request["tools"] = tools
		request["tool_choice"] = "auto"
	}
	body, err := json.Marshal(request)
	if err != nil {
		return AIReply{}, errors.New("构造 AI 请求失败")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return AIReply{}, errors.New("构造 AI 请求失败")
	}
	req.Header.Set("Authorization", "Bearer "+key)
	req.Header.Set("Content-Type", "application/json")
	response, err := client.Client.Do(req)
	if err != nil {
		return AIReply{}, errors.New("AI 连接失败或超时，请检查端点与网络后重试")
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return AIReply{}, fmt.Errorf("AI 服务返回 HTTP %d，请检查密钥、模型和额度", response.StatusCode)
	}
	data, err := io.ReadAll(io.LimitReader(response.Body, (1<<20)+1))
	if err != nil || len(data) > 1<<20 {
		return AIReply{}, errors.New("AI 响应过大或读取失败")
	}
	var result struct {
		Choices []struct {
			Message      AIReply `json:"message"`
			FinishReason string  `json:"finish_reason"`
		} `json:"choices"`
	}
	if json.Unmarshal(data, &result) != nil || len(result.Choices) == 0 {
		return AIReply{}, errors.New("AI 响应格式无效，需兼容 Chat Completions")
	}
	reply := result.Choices[0].Message
	if result.Choices[0].FinishReason == "length" || len(reply.ToolCalls) > 8 || len(reply.Content) > 32000 || (strings.TrimSpace(reply.Content) == "" && len(reply.ToolCalls) == 0) {
		return AIReply{}, errors.New("AI 输出为空、过长或任务过多，请缩小本次请求范围")
	}
	return reply, nil
}
