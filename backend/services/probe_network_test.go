package services

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"strings"
	"sync/atomic"
	"testing"
)

func TestProbeAddressPolicy(t *testing.T) {
	for _, address := range []string{"127.0.0.1", "::1", "10.0.0.1", "172.16.0.1", "192.168.1.1", "::ffff:127.0.0.1", "fc00::1", "169.254.169.254", "fe80::1", "0.0.0.0", "100.100.100.200", "224.0.0.1", "64:ff9b::7f00:1", "2002:7f00:1::", "fec0::1", "::2", "::ffff:0:7f00:1"} {
		if probeAddressAllowed(context.Background(), netip.MustParseAddr(address)) {
			t.Errorf("allowed %s", address)
		}
	}
	for _, address := range []string{"127.0.0.1", "10.0.0.1", "::1", "fc00::1"} {
		if !probeAddressAllowed(privateProbeContext(context.Background()), netip.MustParseAddr(address)) {
			t.Errorf("admin cannot probe %s", address)
		}
	}
	for _, address := range []string{"169.254.169.254", "fe80::1", "100.100.100.200", "fd00:ec2::254", "168.63.129.16"} {
		if probeAddressAllowed(privateProbeContext(context.Background()), netip.MustParseAddr(address)) {
			t.Errorf("admin metadata bypass: %s", address)
		}
	}
	for _, address := range []string{"1.1.1.1", "2606:4700:4700::1111"} {
		if !probeAddressAllowed(context.Background(), netip.MustParseAddr(address)) {
			t.Errorf("blocked public %s", address)
		}
	}
}

func TestProbeProtocolsBlockPrivateAddresses(t *testing.T) {
	var hits atomic.Int32
	internal := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { hits.Add(1); w.WriteHeader(204) }))
	defer internal.Close()
	for _, spec := range []struct{ kind, target string }{
		{"http", internal.URL}, {"http", strings.Replace(internal.URL, "127.0.0.1", "localhost", 1)},
		{"tcp", strings.TrimPrefix(internal.URL, "http://")}, {"icmp", "localhost"},
	} {
		result := ProbeTarget(context.Background(), spec.kind, spec.target, "POST")
		if !errors.Is(result.Err, ErrProbeAddressDenied) {
			t.Errorf("%s: %v", spec.kind, result.Err)
		}
	}
	if hits.Load() != 0 {
		t.Fatal("private endpoint reached")
	}
	result := ProbeTarget(privateProbeContext(context.Background()), "http", internal.URL, "POST")
	if result.Err != nil || hits.Load() != 1 {
		t.Fatalf("authorized private probe failed: %+v", result)
	}
}

func TestProbeRedirectChecksActualDestination(t *testing.T) {
	var hits atomic.Int32
	metadata := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { hits.Add(1) }))
	defer metadata.Close()
	// Serve the first response with a mock transport, then let the real hardened
	// transport handle the redirected request, with no public network access.
	transport := newProbeHTTPClient().Transport
	client := &http.Client{Transport: probeRedirectTransport{next: transport, location: metadata.URL}}
	previous := ProbeHTTPClient
	ProbeHTTPClient = client
	defer func() { ProbeHTTPClient = previous }()
	result := ProbeURL(context.Background(), "http://public.example/start")
	if !errors.Is(result.Err, ErrProbeAddressDenied) || hits.Load() != 0 {
		t.Fatalf("redirect bypass: %v hits=%d", result.Err, hits.Load())
	}
}

type probeRedirectTransport struct {
	next     http.RoundTripper
	location string
}

func (t probeRedirectTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	if r.URL.Host == "public.example" {
		return &http.Response{StatusCode: 302, Header: http.Header{"Location": []string{t.location}}, Body: http.NoBody, Request: r}, nil
	}
	return t.next.RoundTrip(r)
}

func TestDialProbeRejectsPrivateDNS(t *testing.T) {
	// The dial guard checks DNS answers before attempting a TCP connection.
	_, err := dialProbe(context.Background(), "tcp", net.JoinHostPort("localhost", "80"))
	if !errors.Is(err, ErrProbeAddressDenied) {
		t.Fatalf("dial guard: %v", err)
	}
}
