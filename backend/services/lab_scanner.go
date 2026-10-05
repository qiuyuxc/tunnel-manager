package services

import (
	"bufio"
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"tunnel-manager/models"
)

const maxLabScanTargets = 65536

// LabIPSegment is one user-entered target and its expanded unique IPs.
type LabIPSegment struct {
	Target string
	IPs    []string
}

// ParseLabIPSegments expands single IPs, CIDRs, and start-end ranges while
// preserving the original target text for per-segment reporting.
func ParseLabIPSegments(input string) ([]LabIPSegment, error) {
	fields := strings.FieldsFunc(input, func(r rune) bool {
		return r == ',' || r == '\n' || r == '\r' || r == '\t' || r == ' '
	})
	segments := make([]LabIPSegment, 0, len(fields))
	seen := make(map[string]struct{})
	total := 0
	for _, field := range fields {
		segment := LabIPSegment{Target: field, IPs: make([]string, 0)}
		for _, ip := range expandLabTarget(field) {
			if net.ParseIP(ip) == nil {
				return nil, fmt.Errorf("invalid IP target %q", field)
			}
			if _, duplicate := seen[ip]; duplicate {
				continue
			}
			seen[ip] = struct{}{}
			segment.IPs = append(segment.IPs, ip)
			total++
			if total > maxLabScanTargets {
				return nil, fmt.Errorf("IP targets exceed the %d limit", maxLabScanTargets)
			}
		}
		if len(segment.IPs) > 0 {
			segments = append(segments, segment)
		}
	}
	if len(segments) == 0 {
		return nil, errors.New("at least one IP target is required")
	}
	return segments, nil
}

// ParseLabIPTargets expands and de-duplicates all IP targets.
func ParseLabIPTargets(input string) ([]string, error) {
	segments, err := ParseLabIPSegments(input)
	if err != nil {
		return nil, err
	}
	targets := make([]string, 0)
	for _, segment := range segments {
		targets = append(targets, segment.IPs...)
	}
	return targets, nil
}

func expandLabTarget(field string) []string {
	if strings.Contains(field, "/") {
		_, network, err := net.ParseCIDR(field)
		if err != nil {
			return []string{field}
		}
		ones, bits := network.Mask.Size()
		if bits-ones > 16 {
			return []string{field}
		}
		count := 1 << (bits - ones)
		if count > maxLabScanTargets+1 {
			return []string{field}
		}
		addresses := make([]string, 0, count)
		for ip := network.IP.Mask(network.Mask); network.Contains(ip); incrementIP(ip) {
			addresses = append(addresses, ip.String())
			if len(addresses) > maxLabScanTargets {
				break
			}
		}
		return addresses
	}
	if startText, endText, found := strings.Cut(field, "-"); found {
		start := net.ParseIP(startText)
		end := net.ParseIP(endText)
		if start != nil && end != nil && start.To4() != nil && end.To4() != nil {
			if compareIPs(start, end) > 0 {
				return []string{field}
			}
			addresses := []string{}
			for ip := start.To4(); ip != nil && compareIPs(ip, end.To4()) <= 0; incrementIP(ip) {
				addresses = append(addresses, ip.String())
				if len(addresses) > maxLabScanTargets || compareIPs(ip, end) == 0 {
					break
				}
			}
			return addresses
		}
	}
	if address := net.ParseIP(field); address != nil {
		return []string{address.String()}
	}
	return []string{field}
}

func incrementIP(ip net.IP) {
	for i := len(ip) - 1; i >= 0; i-- {
		ip[i]++
		if ip[i] != 0 {
			return
		}
	}
}

func compareIPs(left, right net.IP) int {
	leftBytes, rightBytes := left.To16(), right.To16()
	for i := range leftBytes {
		if leftBytes[i] < rightBytes[i] {
			return -1
		}
		if leftBytes[i] > rightBytes[i] {
			return 1
		}
	}
	return 0
}

// ParseLabStatuses parses the accepted HTTP status list.
func ParseLabStatuses(input string) ([]int, error) {
	if strings.TrimSpace(input) == "" {
		input = "200"
	}
	parts := strings.Split(input, ",")
	statuses := make([]int, 0, len(parts))
	seen := make(map[int]struct{}, len(parts))
	for _, part := range parts {
		status, err := strconv.Atoi(strings.TrimSpace(part))
		if err != nil || status < 200 || status > 599 {
			return nil, fmt.Errorf("invalid HTTP status %q", strings.TrimSpace(part))
		}
		if _, duplicate := seen[status]; duplicate {
			continue
		}
		seen[status] = struct{}{}
		statuses = append(statuses, status)
	}
	return statuses, nil
}

// LabIPSelectorProgress reports scanner progress to callers and the UI.
type LabIPSelectorProgress struct {
	Scanned int `json:"scanned"`
	Matched int `json:"matched"`
	Total   int `json:"total"`
	Percent int `json:"percent"`
}

// ScanLabIPs probes every target with the configured Host and TLS SNI. The
// optional callback receives throttled live progress updates.
func ScanLabIPs(ctx context.Context, settings models.LabIPSelectorSettings, onProgress ...func(LabIPSelectorProgress)) ([]models.LabIPScanResult, []models.LabIPScanResult, []string, []models.LabIPSegmentResult, error) {
	return scanLabIPs(ctx, settings, scanLabIP, onProgress...)
}

func scanLabIPs(ctx context.Context, settings models.LabIPSelectorSettings, probe func(context.Context, string, string, string, time.Duration, <-chan time.Time) models.LabIPScanResult, onProgress ...func(LabIPSelectorProgress)) ([]models.LabIPScanResult, []models.LabIPScanResult, []string, []models.LabIPSegmentResult, error) {
	if err := NormalizeLabProbe(&settings); err != nil {
		return nil, nil, nil, nil, err
	}
	var progressCallback func(LabIPSelectorProgress)
	if len(onProgress) > 0 {
		progressCallback = onProgress[0]
	}
	segments, err := ParseLabIPSegments(settings.IPTargets)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	targets := make([]string, 0)
	targetSegments := make([]int, 0)
	for segmentIndex, segment := range segments {
		for _, ip := range segment.IPs {
			targets = append(targets, ip)
			targetSegments = append(targetSegments, segmentIndex)
		}
	}
	statuses, err := ParseLabStatuses(settings.Statuses)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	if err := ValidateLabPublicTargets(targets); err != nil {
		return nil, nil, nil, nil, err
	}
	if strings.TrimSpace(settings.Host) == "" {
		return nil, nil, nil, nil, errors.New("Host is required")
	}
	sni := strings.TrimSpace(settings.SNI)
	if sni == "" {
		sni = strings.TrimSpace(settings.Host)
	}
	path := strings.TrimSpace(settings.Path)
	if path == "" {
		path = "/"
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	request := fmt.Sprintf("GET %s HTTP/1.1\r\nHost: %s\r\nUser-Agent: TunnelManagerLab/1.0\r\nConnection: close\r\n\r\n", path, settings.Host)
	workers := settings.Workers
	limits := settings.Limits.Normalized()
	if workers < 1 {
		workers = models.LabMaxWorkers
	}
	if workers > limits.MaxWorkers {
		workers = limits.MaxWorkers
	}
	timeout := time.Duration(settings.Timeout) * time.Second
	if timeout <= 0 {
		timeout = 2 * time.Second
	}

	accepted := make(map[int]struct{}, len(statuses))
	for _, status := range statuses {
		accepted[status] = struct{}{}
	}
	type scanJob struct {
		index   int
		segment int
		ip      string
	}
	jobs := make(chan scanJob)
	requestTicker := time.NewTicker(time.Second / time.Duration(limits.RequestsPerSecond))
	defer requestTicker.Stop()
	results := make([]models.LabIPScanResult, len(targets))
	var wg sync.WaitGroup
	var progressMu sync.Mutex
	progressScanned := 0
	progressMatched := 0
	reportProgress := func(status int, force bool) {
		if progressCallback == nil {
			return
		}
		progressMu.Lock()
		progressScanned++
		if _, ok := accepted[status]; ok {
			progressMatched++
		}
		progress := LabIPSelectorProgress{
			Scanned: progressScanned,
			Matched: progressMatched,
			Total:   len(targets),
			Percent: progressScanned * 100 / len(targets),
		}
		shouldReport := force || progressScanned%20 == 0
		progressMu.Unlock()
		if shouldReport {
			progressCallback(progress)
		}
	}
	if progressCallback != nil {
		progressCallback(LabIPSelectorProgress{Total: len(targets)})
	}
	for worker := 0; worker < workers; worker++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for job := range jobs {
				result := probe(ctx, job.ip, request, sni, timeout, requestTicker.C)
				results[job.index] = result
				reportProgress(result.Status, false)
				select {
				case <-ctx.Done():
					return
				default:
				}
			}
		}()
	}
	for index, target := range targets {
		select {
		case jobs <- scanJob{index: index, segment: targetSegments[index], ip: target}:
		case <-ctx.Done():
			close(jobs)
			wg.Wait()
			return nil, nil, nil, nil, ctx.Err()
		}
	}
	close(jobs)
	wg.Wait()
	if err := ctx.Err(); err != nil {
		return nil, nil, nil, nil, err
	}
	if progressCallback != nil {
		progressMu.Lock()
		progress := LabIPSelectorProgress{
			Scanned: progressScanned,
			Matched: progressMatched,
			Total:   len(targets),
			Percent: progressScanned * 100 / len(targets),
		}
		progressMu.Unlock()
		progressCallback(progress)
	}

	segmentResults := make([]models.LabIPSegmentResult, len(segments))
	for index, segment := range segments {
		segmentResults[index] = models.LabIPSegmentResult{Target: segment.Target}
	}
	matches := make([]models.LabIPScanResult, 0)
	for index, result := range results {
		segmentResult := &segmentResults[targetSegments[index]]
		segmentResult.Scanned++
		if _, ok := accepted[result.Status]; ok && result.Error == "" {
			segmentResult.Matched++
			matches = append(matches, result)
		}
	}
	if len(matches) == 0 {
		return results, nil, nil, segmentResults, errors.New("no IP returned an accepted status")
	}
	sort.Slice(matches, func(i, j int) bool {
		if matches[i].LatencyMS != matches[j].LatencyMS {
			return matches[i].LatencyMS < matches[j].LatencyMS
		}
		return compareIPs(net.ParseIP(matches[i].IP), net.ParseIP(matches[j].IP)) < 0
	})
	top := settings.Top
	if top < 1 {
		top = 10
	}
	if top > len(matches) {
		top = len(matches)
	}
	if top > 100 {
		top = 100
	}
	selected := make([]string, top)
	for i, match := range matches[:top] {
		selected[i] = match.IP
	}
	return results, matches, selected, segmentResults, nil
}

func scanLabIP(ctx context.Context, ip, request, sni string, timeout time.Duration, gate <-chan time.Time) models.LabIPScanResult {
	return scanLabIPWithDialer(ctx, ip, request, sni, timeout, gate, func(ctx context.Context, dialer *tls.Dialer, address string) (net.Conn, error) {
		return dialer.DialContext(ctx, "tcp", address)
	})
}

func scanLabIPWithDialer(ctx context.Context, ip, request, sni string, timeout time.Duration, gate <-chan time.Time, dial func(context.Context, *tls.Dialer, string) (net.Conn, error)) models.LabIPScanResult {
	start := time.Now()
	result := models.LabIPScanResult{IP: ip}
	dialer := &tls.Dialer{NetDialer: &net.Dialer{Timeout: timeout}, Config: &tls.Config{ServerName: sni, MinVersion: tls.VersionTLS12}}
	connection, err := dial(ctx, dialer, net.JoinHostPort(ip, "443"))
	if err != nil {
		result.LatencyMS = time.Since(start).Milliseconds()
		result.Error = err.Error()
		return result
	}
	defer connection.Close()
	stopClosing := context.AfterFunc(ctx, func() { _ = connection.Close() })
	defer stopClosing()
	select {
	case <-ctx.Done():
		result.Error = ctx.Err().Error()
		return result
	case <-gate:
	}
	start = time.Now()
	if err := connection.SetDeadline(time.Now().Add(timeout)); err != nil {
		result.LatencyMS = time.Since(start).Milliseconds()
		result.Error = err.Error()
		return result
	}
	if _, err := connection.Write([]byte(request)); err != nil {
		result.LatencyMS = time.Since(start).Milliseconds()
		result.Error = err.Error()
		return result
	}
	status, err := readLabStatus(connection)
	result.LatencyMS = time.Since(start).Milliseconds()
	if err != nil {
		result.Error = err.Error()
		return result
	}
	result.Status = status
	select {
	case <-ctx.Done():
		result.Error = ctx.Err().Error()
	default:
	}
	return result
}

func readLabStatus(reader io.Reader) (int, error) {
	line, err := bufio.NewReader(io.LimitReader(reader, 4096)).ReadString('\n')
	if err != nil {
		return 0, err
	}
	fields := strings.Fields(line)
	if len(fields) < 2 || (fields[0] != "HTTP/1.1" && fields[0] != "HTTP/1.0") || !strings.HasSuffix(line, "\r\n") || len(fields[1]) != 3 {
		return 0, errors.New("invalid HTTP response")
	}
	status, err := strconv.Atoi(fields[1])
	if err != nil || status < http.StatusOK || status > 599 {
		return 0, errors.New("invalid HTTP status")
	}
	return status, nil
}
