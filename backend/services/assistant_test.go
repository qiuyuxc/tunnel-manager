package services

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"strings"
	"sync/atomic"
	"testing"

	"tunnel-manager/models"
)

type aiTransport func(*http.Request) (*http.Response, error)

func (transport aiTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	return transport(request)
}

func TestAIEndpointBoundaries(t *testing.T) {
	for _, endpoint := range []string{"ftp://example.com/v1", "file:///tmp/model", "//localhost:8787/v1", "http:///v1", "http://localhost:0/v1", "https://localhost:65536/v1", "http://localhost:/v1", "http://localhost:invalid/v1", "http://localhost:-1/v1", "http://0.0.0.0/v1", "http://[::]/v1", "http://224.0.0.1/v1", "https://169.254.169.254/v1", "http://[fd00:ec2::254]/v1", "http://[fe80::1%25eth0]/v1", "https://user:pass@example.com/v1", "https://example.com/v1?secret=x", "https://example.com/v1?", "https://example.com/v1#key"} {
		if _, err := ValidateAIEndpoint(endpoint); err == nil {
			t.Errorf("accepted unsafe endpoint %s", endpoint)
		}
	}
	for _, endpoint := range []string{"https://example.com/v1/", "https://example.com:443/chat/completions", "http://example.com/v1", " http://127.0.0.1:8787/v1/ ", "http://localhost:11434/v1", "http://host.docker.internal:8787/v1", "https://localhost:8443/v1", "http://[::1]:8787/v1", "http://10.0.0.1/v1", "https://192.168.1.10:8443/v1", "http://172.17.0.1:8787/v1", "http://100.64.0.1:8787/v1", "http://[fd12::1]:8787/v1", "https://[::ffff:127.0.0.1]/v1", "https://example.com:65535/v1", "http://localhost:1/v1"} {
		if normalized, err := ValidateAIEndpoint(endpoint); err != nil {
			t.Errorf("rejected endpoint %s: %v", endpoint, err)
		} else if normalized != strings.TrimRight(strings.TrimSpace(endpoint), "/") {
			t.Errorf("incorrect normalization: %q", normalized)
		}
	}
	for _, address := range []string{"0.0.0.0", "0.0.0.1", "::", "224.0.0.1", "ff02::1", "198.18.1.1", "169.254.169.254", "::ffff:169.254.169.254", "fd00:ec2::254", "fe80::1", "::1%lo", "64:ff9b::a00:1", "2002:7f00:1::"} {
		if AllowedAIAddress(netip.MustParseAddr(address)) {
			t.Errorf("accepted address %s", address)
		}
	}
	for _, address := range []string{"1.1.1.1", "2606:4700:4700::1111", "127.0.0.1", "::1", "::ffff:127.0.0.1", "10.0.0.1", "172.17.0.1", "192.168.1.10", "100.64.0.1", "fc00::1"} {
		if !AllowedAIAddress(netip.MustParseAddr(address)) {
			t.Errorf("rejected address %s", address)
		}
	}
}

func TestAIClientSelfHostedConnections(t *testing.T) {
	for _, variant := range []string{"http", "localhost", "https"} {
		t.Run(variant, func(t *testing.T) {
			server := httptest.NewUnstartedServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
				if request.Method != http.MethodPost || request.URL.Path != "/v1/chat/completions" || request.Header.Get("Authorization") != "Bearer local-key" {
					t.Error("incorrect self-hosted request")
				}
				response.Header().Set("Content-Type", "application/json")
				io.WriteString(response, `{"choices":[{"message":{"content":"local ready"},"finish_reason":"stop"}]}`)
			}))
			if variant == "https" {
				server.StartTLS()
			} else {
				server.Start()
			}
			defer server.Close()
			client := NewAIClient()
			defer client.Client.CloseIdleConnections()
			transport := client.Client.Transport.(*http.Transport)
			if transport.Proxy != nil {
				t.Fatal("environment proxies must not receive AI credentials")
			}
			if variant == "https" {
				trusted := x509.NewCertPool()
				trusted.AddCert(server.Certificate())
				transport.TLSClientConfig = &tls.Config{RootCAs: trusted}
			}
			endpoint := server.URL + "/v1"
			if variant == "localhost" {
				endpoint = strings.Replace(endpoint, "127.0.0.1", "localhost", 1)
			}
			reply, err := client.Complete(context.Background(), models.AIConnection{Endpoint: endpoint, Model: "local-model"}, "local-key", []models.AIMessage{{Role: "user", Content: "hello"}}, nil)
			if err != nil || reply.Content != "local ready" {
				t.Fatalf("self-hosted connection failed: reply=%+v err=%v", reply, err)
			}
		})
	}
}

func TestAIClientSelfHostedSafety(t *testing.T) {
	var redirected atomic.Int32
	target := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		redirected.Add(1)
	}))
	defer target.Close()
	redirect := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		http.Redirect(response, request, target.URL, http.StatusTemporaryRedirect)
	}))
	defer redirect.Close()
	client := NewAIClient()
	defer client.Client.CloseIdleConnections()
	_, err := client.Complete(context.Background(), models.AIConnection{Endpoint: redirect.URL, Model: "local-model"}, "local-key", nil, nil)
	if err == nil || !strings.Contains(err.Error(), "307") || redirected.Load() != 0 {
		t.Fatalf("redirect was not rejected: err=%v requests=%d", err, redirected.Load())
	}
	secure := httptest.NewTLSServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Error("untrusted TLS certificate accepted")
	}))
	defer secure.Close()
	if _, err := client.Complete(context.Background(), models.AIConnection{Endpoint: secure.URL, Model: "local-model"}, "local-key", nil, nil); err == nil {
		t.Fatal("HTTPS certificate validation was disabled")
	}
	for _, address := range []string{"169.254.169.254:80", "[fd00:ec2::254]:80", "0.0.0.0:80", "[::]:80"} {
		connection, err := client.Client.Transport.(*http.Transport).DialContext(context.Background(), "tcp", address)
		if connection != nil {
			connection.Close()
		}
		if err == nil || !strings.Contains(err.Error(), "不允许的地址") {
			t.Errorf("dial did not reject unsafe resolved address %s: %v", address, err)
		}
	}
}

func TestAIClientProtocolAndErrors(t *testing.T) {
	for _, testCase := range []struct {
		name   string
		status int
		body   string
		valid  bool
	}{
		{"text", 200, `{"choices":[{"message":{"content":"ready"},"finish_reason":"stop"}]}`, true},
		{"tools", 200, `{"choices":[{"message":{"content":null,"tool_calls":[{"id":"call_1","type":"function","function":{"name":"create_tunnel","arguments":"{\"name\":\"site\"}"}}]},"finish_reason":"tool_calls"}]}`, true},
		{"error secret", 401, `secret-from-upstream`, false},
		{"redirect", 302, `secret-from-upstream`, false},
		{"empty", 200, `{"choices":[]}`, false},
		{"truncated", 200, `{"choices":[{"message":{"content":"partial"},"finish_reason":"length"}]}`, false},
		{"oversized", 200, strings.Repeat("x", (1<<20)+1), false},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			client := NewAIClient()
			client.Client.Transport = aiTransport(func(request *http.Request) (*http.Response, error) {
				if request.URL.String() != "https://example.com/v1/chat/completions" || request.Header.Get("Authorization") != "Bearer test-key" {
					t.Fatal("incorrect upstream request")
				}
				var payload map[string]any
				if err := json.NewDecoder(request.Body).Decode(&payload); err != nil || payload["model"] != "test-model" || payload["stream"] != false {
					t.Fatal("incorrect request payload")
				}
				return &http.Response{StatusCode: testCase.status, Body: io.NopCloser(strings.NewReader(testCase.body)), Header: http.Header{}}, nil
			})
			_, err := client.Complete(context.Background(), models.AIConnection{Endpoint: "https://example.com/v1", Model: "test-model"}, "test-key", []models.AIMessage{{Role: "user", Content: "hello"}}, nil)
			if (err == nil) != testCase.valid {
				t.Fatalf("valid=%v err=%v", testCase.valid, err)
			}
			if err != nil && (strings.Contains(err.Error(), "secret-from-upstream") || strings.Contains(err.Error(), "test-key")) {
				t.Fatal("upstream secret leaked")
			}
		})
	}
	client := NewAIClient()
	if client.Client.CheckRedirect(nil, nil) != http.ErrUseLastResponse {
		t.Fatal("redirects must not forward credentials")
	}
}
