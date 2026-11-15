package services

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/netip"
	"time"
)

var ErrProbeAddressDenied = errors.New("monitor target address is not permitted")

// privateProbeKey is set only for monitors currently owned by an administrator.
type privateProbeKey struct{}

func privateProbeContext(ctx context.Context) context.Context {
	return context.WithValue(ctx, privateProbeKey{}, true)
}

var specialProbeNetworks = []netip.Prefix{
	netip.MustParsePrefix("0.0.0.0/8"), netip.MustParsePrefix("100.64.0.0/10"),
	netip.MustParsePrefix("192.0.0.0/24"), netip.MustParsePrefix("192.0.2.0/24"),
	netip.MustParsePrefix("198.18.0.0/15"), netip.MustParsePrefix("198.51.100.0/24"),
	netip.MustParsePrefix("203.0.113.0/24"), netip.MustParsePrefix("240.0.0.0/4"),
	netip.MustParsePrefix("2001:db8::/32"), netip.MustParsePrefix("64:ff9b::/96"),
	netip.MustParsePrefix("64:ff9b:1::/48"), netip.MustParsePrefix("2002::/16"),
	netip.MustParsePrefix("2001::/32"),
	netip.MustParsePrefix("fd00:ec2::254/128"),
	netip.MustParsePrefix("168.63.129.16/32"),
}

func probeAddressAllowed(ctx context.Context, ip netip.Addr) bool {
	ip = ip.Unmap()
	// Metadata/link-local and multicast are never monitoring targets.
	if !ip.IsValid() || ip.IsUnspecified() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsMulticast() {
		return false
	}
	for _, prefix := range specialProbeNetworks {
		if prefix.Contains(ip) {
			return false
		}
	}
	if ip.IsPrivate() || ip.IsLoopback() {
		allowed, _ := ctx.Value(privateProbeKey{}).(bool)
		return allowed
	}
	// Native public IPv6 lives in 2000::/3. Reject deprecated site-local and
	// translation-compatible ranges which IsGlobalUnicast alone can accept.
	if ip.Is6() && !netip.MustParsePrefix("2000::/3").Contains(ip) {
		return false
	}
	if !ip.IsGlobalUnicast() {
		return false
	}
	return true
}

// resolveProbeHost checks every answer and returns numeric addresses. Dialing
// those exact answers avoids a second DNS lookup and DNS rebinding.
func resolveProbeHost(ctx context.Context, host string) ([]netip.Addr, error) {
	addresses, err := net.DefaultResolver.LookupNetIP(ctx, "ip", host)
	if err != nil {
		return nil, err
	}
	if len(addresses) == 0 {
		return nil, ErrProbeAddressDenied
	}
	for _, ip := range addresses {
		if !probeAddressAllowed(ctx, ip) {
			return nil, ErrProbeAddressDenied
		}
	}
	return addresses, nil
}

func dialProbe(ctx context.Context, network, address string) (net.Conn, error) {
	host, port, err := net.SplitHostPort(address)
	if err != nil {
		return nil, err
	}
	ips, err := resolveProbeHost(ctx, host)
	if err != nil {
		return nil, err
	}
	var lastErr error
	for _, ip := range ips {
		conn, err := (&net.Dialer{Timeout: 8 * time.Second}).DialContext(ctx, network, net.JoinHostPort(ip.String(), port))
		if err == nil {
			return conn, nil
		}
		lastErr = err
	}
	return nil, lastErr
}

func newProbeHTTPClient() *http.Client {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.Proxy = nil
	transport.DialContext = dialProbe
	transport.DisableKeepAlives = true
	return &http.Client{Timeout: 8 * time.Second, Transport: transport}
}

// Shared across scheduled and manual probes, accounts, and protocols.
var probeSlots = make(chan struct{}, 32)

func acquireProbe(ctx context.Context) bool {
	select {
	case probeSlots <- struct{}{}:
		return true
	case <-ctx.Done():
		return false
	}
}
