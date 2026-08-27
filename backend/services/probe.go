package services

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// ProbeResult summarizes one reachability check.
type ProbeResult struct {
	State     string
	HTTPCode  int
	LatencyMs int64
	Err       error
}

// ProbeHTTPClient performs health-check requests across the app.
var ProbeHTTPClient = &http.Client{Timeout: 8 * time.Second}

// probeHTTP fetches rawURL (prefixing https:// when the scheme is missing)
// with the given HTTP method and classifies the outcome.
func probeHTTP(ctx context.Context, rawURL, method string) ProbeResult {
	url := strings.TrimSpace(rawURL)
	if url == "" {
		return ProbeResult{State: "down", Err: errEmptyURL}
	}
	if !strings.Contains(url, "://") {
		url = "https://" + url
	}
	req, err := http.NewRequestWithContext(ctx, method, url, nil)
	if err != nil {
		return ProbeResult{State: "down", Err: err}
	}
	req.Header.Set("User-Agent", "tunnel-manager-healthcheck")
	start := time.Now()
	resp, err := ProbeHTTPClient.Do(req)
	if err != nil {
		return ProbeResult{State: "down", Err: err}
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 4<<10))
	return ProbeResult{
		State:     ClassifyProbe(resp.StatusCode, nil),
		HTTPCode:  resp.StatusCode,
		LatencyMs: time.Since(start).Milliseconds(),
	}
}

// ProbeURL keeps the legacy GET-only entry used by the tunnel detector.
func ProbeURL(ctx context.Context, rawURL string) ProbeResult {
	return probeHTTP(ctx, rawURL, http.MethodGet)
}

// ProbeTarget dispatches one check by target type: http (default), tcp or icmp.
func ProbeTarget(ctx context.Context, targetType, rawURL, method string) ProbeResult {
	switch targetType {
	case "tcp":
		return ProbeTCP(rawURL)
	case "icmp":
		return ProbeICMP(rawURL)
	default:
		m := strings.ToUpper(strings.TrimSpace(method))
		if m != http.MethodPost {
			m = http.MethodGet
		}
		return probeHTTP(ctx, rawURL, m)
	}
}

// ProbeTCP checks a host:port endpoint by completing the TCP handshake.
func ProbeTCP(hostPort string) ProbeResult {
	hostPort = strings.TrimSpace(hostPort)
	if hostPort == "" {
		return ProbeResult{State: "down", Err: errEmptyURL}
	}
	if _, _, err := net.SplitHostPort(hostPort); err != nil {
		hostPort = net.JoinHostPort(strings.Trim(hostPort, "[]"), "80")
	}
	start := time.Now()
	conn, err := net.DialTimeout("tcp", hostPort, 8*time.Second)
	latency := time.Since(start).Milliseconds()
	if err != nil {
		return ProbeResult{State: "down", LatencyMs: latency, Err: err}
	}
	_ = conn.Close()
	return ProbeResult{State: "ok", LatencyMs: latency}
}

// ProbeICMP shells out to the system ping binary; raw ICMP sockets are
// typically unavailable to unprivileged users.
func ProbeICMP(host string) ProbeResult {
	host = strings.TrimSpace(host)
	if host == "" {
		return ProbeResult{State: "down", Err: errEmptyURL}
	}
	host = strings.TrimPrefix(strings.TrimPrefix(host, "https://"), "http://")
	if i := strings.IndexAny(host, "/:"); i >= 0 {
		host = host[:i]
	}
	start := time.Now()
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "ping", "-c", "1", "-W", "5", host)
	out, err := cmd.Output()
	latency := time.Since(start).Milliseconds()
	if err != nil {
		msg := err.Error()
		var ee *exec.ExitError
		if errors.As(err, &ee) && len(ee.Stderr) > 0 {
			msg = strings.TrimSpace(string(ee.Stderr))
		}
		return ProbeResult{State: "down", LatencyMs: latency, Err: &probeError{msg}}
	}
	ms := latency
	for _, line := range strings.Split(string(out), "\n") {
		if i := strings.Index(line, "time="); i >= 0 {
			fields := strings.Fields(line[i+5:])
			if len(fields) > 0 {
				v := strings.TrimSuffix(fields[0], "ms")
				if f, e := strconv.ParseFloat(v, 64); e == nil {
					ms = int64(f)
				}
			}
			break
		}
	}
	return ProbeResult{State: "ok", LatencyMs: ms}
}

var errEmptyURL = &probeError{"empty target URL"}

type probeError struct{ msg string }

func (e *probeError) Error() string { return e.msg }

// ClassifyProbe maps a probe outcome to ok | warn | down:
// any response under 5xx means the edge could reach our origin,
// 5xx means reachable edge but broken origin, transport errors mean down.
func ClassifyProbe(code int, err error) string {
	switch {
	case err != nil:
		return "down"
	case code >= 500:
		return "warn"
	default:
		return "ok"
	}
}

// ProbeErrorText shortens a transport error for UI display.
func ProbeErrorText(err error) string {
	if err == nil {
		return ""
	}
	msg := err.Error()
	if i := strings.Index(msg, ": "); i >= 0 && len(msg) > 80 {
		msg = msg[i+2:]
	}
	return msg
}
