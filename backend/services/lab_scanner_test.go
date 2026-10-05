package services

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"io"
	"log"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"tunnel-manager/models"
)

func TestLabHTTPFilteringRankingAndSharedRequestGate(t *testing.T) {
	settings := models.LabIPSelectorSettings{Host: "probe.example.com", SNI: "tls.example.com", Path: "/ip-check.txt", Statuses: "200", Workers: 4, Top: 2, Timeout: 1, IPTargets: "1.1.1.1-1.1.1.4"}
	probe := func(ctx context.Context, address, request, sni string, timeout time.Duration, gate <-chan time.Time) models.LabIPScanResult {
		if !strings.HasPrefix(request, "GET /ip-check.txt HTTP/1.1\r\nHost: probe.example.com\r\n") || sni != settings.SNI {
			t.Error("incorrect direct-IP probe Host/SNI/path")
		}
		select {
		case <-ctx.Done():
			return models.LabIPScanResult{IP: address, Error: ctx.Err().Error()}
		case <-gate:
		}
		statuses := map[string]int{"1.1.1.1": 200, "1.1.1.2": 403, "1.1.1.3": 418, "1.1.1.4": 200}
		latency := int64(20)
		if address == "1.1.1.4" {
			latency = 5
		}
		return models.LabIPScanResult{IP: address, Status: statuses[address], LatencyMS: latency}
	}
	started := time.Now()
	all, matches, selected, segments, err := scanLabIPs(context.Background(), settings, probe)
	if err != nil || len(all) != 4 || len(matches) != 2 || len(selected) != 2 || selected[0] != "1.1.1.4" || len(segments) != 1 || segments[0].Matched != 2 {
		t.Fatalf("incorrect filtering/ranking/segments: %v %v %v %v %v", all, matches, selected, segments, err)
	}
	if time.Since(started) < 350*time.Millisecond {
		t.Fatal("workers bypassed shared request pacing")
	}
}

func TestLabHTTPStatusValidation(t *testing.T) {
	for _, line := range []string{"200 OK\r\n", "INVALID 200 OK\r\n", "HTTP/1.1 200", "HTTP/1.1 101 Switching Protocols\r\n", "HTTP/1.1 600 Invalid\r\n", strings.Repeat("a", 5000)} {
		if _, err := readLabStatus(strings.NewReader(line)); err == nil {
			t.Errorf("accepted invalid response %q", line)
		}
	}
	for _, line := range []string{"HTTP/1.1 200 OK\r\n", "HTTP/1.0 204 No Content\r\n", "HTTP/1.1 403 Forbidden\r\n", "HTTP/1.1 418 Teapot\r\n", "HTTP/1.1 302 Found\r\nLocation: https://other.example.com\r\n"} {
		if _, err := readLabStatus(strings.NewReader(line)); err != nil {
			t.Errorf("invalid parsing: %v", err)
		}
	}
}

func TestLabTLSProbeUsesTargetHostSNIAndDoesNotFollowRedirects(t *testing.T) {
	for _, mode := range []string{"success", "redirect", "stream", "wrong-sni", "untrusted", "cancel"} {
		t.Run(mode, func(t *testing.T) {
			var requests atomic.Int32
			helloNames := make(chan string, 1)
			server := httptest.NewUnstartedServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
				requests.Add(1)
				if request.Host != "probe.example.com" || request.URL.RequestURI() != "/ip-check.txt?test=1" || request.Method != http.MethodGet {
					t.Errorf("unexpected probe request: %s %s %s", request.Method, request.Host, request.URL)
				}
				if mode == "redirect" {
					response.Header().Set("Location", "https://must-not-be-followed.example.com/")
					response.WriteHeader(http.StatusFound)
					return
				}
				response.WriteHeader(http.StatusOK)
				if mode == "stream" {
					response.(http.Flusher).Flush()
					<-request.Context().Done()
				}
			}))
			server.Config.ErrorLog = log.New(io.Discard, "", 0)
			server.TLS = &tls.Config{GetConfigForClient: func(hello *tls.ClientHelloInfo) (*tls.Config, error) {
				helloNames <- hello.ServerName
				return nil, nil
			}}
			server.StartTLS()
			defer server.Close()
			roots := x509.NewCertPool()
			if mode != "untrusted" {
				roots.AddCert(server.Certificate())
			}
			sni := server.Certificate().DNSNames[0]
			if mode == "wrong-sni" {
				sni = "wrong-probe.invalid"
				if server.Certificate().VerifyHostname(sni) == nil {
					t.Fatal("mismatched SNI fixture is covered by the test certificate")
				}
			}
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()
			dial := func(ctx context.Context, dialer *tls.Dialer, address string) (net.Conn, error) {
				if address != "1.1.1.1:443" || dialer.Config.ServerName != sni || dialer.Config.InsecureSkipVerify || dialer.Config.MinVersion != tls.VersionTLS12 {
					t.Error("probe changed its target or TLS verification policy")
				}
				dialer.Config.RootCAs = roots
				connection, err := dialer.DialContext(ctx, "tcp", server.Listener.Addr().String())
				if mode == "cancel" {
					cancel()
				}
				return connection, err
			}
			gate := make(chan time.Time, 1)
			if mode != "cancel" {
				gate <- time.Now()
			}
			result := scanLabIPWithDialer(ctx, "1.1.1.1", "GET /ip-check.txt?test=1 HTTP/1.1\r\nHost: probe.example.com\r\nConnection: close\r\n\r\n", sni, time.Second, gate, dial)
			select {
			case hello := <-helloNames:
				if hello != sni {
					t.Fatalf("TLS SNI = %q, want %q", hello, sni)
				}
			case <-time.After(time.Second):
				t.Fatal("TLS ClientHello was not received")
			}
			if mode == "wrong-sni" || mode == "untrusted" || mode == "cancel" {
				if result.Error == "" || result.Status != 0 || requests.Load() != 0 {
					t.Fatalf("unsafe probe: %#v, requests: %d", result, requests.Load())
				}
				return
			}
			status := http.StatusOK
			if mode == "redirect" {
				status = http.StatusFound
			}
			if result.Error != "" || result.Status != status || requests.Load() != 1 {
				t.Fatalf("unexpected probe: %#v, requests: %d", result, requests.Load())
			}
		})
	}
}

func TestLabScanWithoutProgressCallback(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, _, _, _, err := ScanLabIPs(ctx, models.LabIPSelectorSettings{Host: "probe.example.com", IPTargets: "1.1.1.1", Statuses: "200"})
	if err == nil {
		t.Fatal("cancelled scan succeeded")
	}
}

func TestLabTargetsCanonicalizeAndRejectReversedRanges(t *testing.T) {
	addresses, err := ParseLabIPTargets("2606:4700::1 2606:4700:0:0:0:0:0:1")
	if err != nil || len(addresses) != 1 {
		t.Fatalf("duplicate IPv6 addresses: %v, %v", addresses, err)
	}
	if _, err := ParseLabIPTargets("1.1.1.2-1.1.1.1 1.1.1.3"); err == nil {
		t.Fatal("silently accepted an invalid range")
	}
}

func TestParseLabIPTargets(t *testing.T) {
	targets, err := ParseLabIPTargets("192.0.2.1 192.0.2.1,192.0.2.10-192.0.2.12\n198.51.100.0/30")
	if err != nil {
		t.Fatalf("ParseLabIPTargets() error = %v", err)
	}
	want := []string{
		"192.0.2.1", "192.0.2.10", "192.0.2.11", "192.0.2.12",
		"198.51.100.0", "198.51.100.1", "198.51.100.2", "198.51.100.3",
	}
	if len(targets) != len(want) {
		t.Fatalf("targets = %#v, want %#v", targets, want)
	}
	for i := range want {
		if targets[i] != want[i] {
			t.Fatalf("targets[%d] = %q, want %q", i, targets[i], want[i])
		}
	}
}

func TestParseLabIPTargetsRejectsInvalidInput(t *testing.T) {
	if _, err := ParseLabIPTargets("example.com"); err == nil {
		t.Fatal("ParseLabIPTargets() accepted a hostname")
	}
}

func TestParseLabStatuses(t *testing.T) {
	statuses, err := ParseLabStatuses("200, 204,200")
	if err != nil {
		t.Fatalf("ParseLabStatuses() error = %v", err)
	}
	if len(statuses) != 2 || statuses[0] != 200 || statuses[1] != 204 {
		t.Fatalf("statuses = %#v, want [200 204]", statuses)
	}
	if _, err := ParseLabStatuses("600"); err == nil {
		t.Fatal("ParseLabStatuses() accepted 600")
	}
}

func TestParseLabIPSegmentsPreservesInputTargets(t *testing.T) {
	segments, err := ParseLabIPSegments("192.0.2.10-192.0.2.12\n198.51.100.0/30")
	if err != nil {
		t.Fatalf("ParseLabIPSegments() error = %v", err)
	}
	if len(segments) != 2 {
		t.Fatalf("segments = %#v, want 2 segments", segments)
	}
	if segments[0].Target != "192.0.2.10-192.0.2.12" || len(segments[0].IPs) != 3 {
		t.Fatalf("first segment = %#v, want range target with 3 IPs", segments[0])
	}
	if segments[1].Target != "198.51.100.0/30" || len(segments[1].IPs) != 4 {
		t.Fatalf("second segment = %#v, want CIDR target with 4 IPs", segments[1])
	}
}
