// PROTOTYPE: sample official Cloudflare IPv4 space from the executing device,
// keep observed APAC entries, and validate several addresses per candidate /24.
// No credentials, panel integration, DNS mutation, or external probe services.
package main

import (
	"bufio"
	"context"
	"crypto/tls"
	"encoding/binary"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"math/rand"
	"net"
	"net/http"
	"net/netip"
	"os"
	"os/signal"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"syscall"
	"time"
)

var apac = strings.Fields("NRT HND KIX FUK OKA SDJ HKG MFM TPE KHH ICN GMP PUS CJU SIN BKK DMK CNX HKT KUL JHB PEN KCH BWN MNL CEB DVO CGK JKT SUB DPS JOG SGN HAN DAD PNH REP VTE RGN DAC KTM CMB MAA BOM DEL HYD BLR CCU AMD NAG JAI ISB KHI LHE SYD MEL BNE PER ADL CBR AKL CHC WLG GUM POM NAN NOU PPT")

type observation struct {
	IP      string    `json:"ip"`
	Prefix  string    `json:"prefix"`
	Phase   string    `json:"phase"`
	Round   int       `json:"round"`
	At      time.Time `json:"at"`
	LocalIP string    `json:"local_ip,omitempty"`
	Status  int       `json:"status"`
	Success bool      `json:"success"`
	Colo    string    `json:"colo,omitempty"`
	APAC    bool      `json:"apac"`
	TCP     float64   `json:"tcp_ms"`
	TLS     float64   `json:"tls_ms"`
	HTTP    float64   `json:"http_ms"`
	Total   float64   `json:"total_ms"`
	Error   string    `json:"error,omitempty"`
}

type segment struct {
	Prefix        string         `json:"prefix"`
	IPs           []string       `json:"ips"`
	Attempts      int            `json:"attempts"`
	Successes     int            `json:"successes"`
	APACSuccesses int            `json:"apac_successes"`
	Median        float64        `json:"successful_median_ms"`
	P90           float64        `json:"successful_p90_ms"`
	Worst         float64        `json:"worst_ms"`
	Colos         map[string]int `json:"colos"`
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	rangesPath := flag.String("ranges", "", "official IPv4 CIDRs or their subranges, one per line")
	seedPath := flag.String("seeds", "", "optional known IPv4 controls, one per line")
	outDir := flag.String("out", "", "existing directory for NEW output files")
	host := flag.String("host", "", "business Host and TLS SNI")
	rate := flag.Int("rate", 5, "maximum connection starts per second (1..5)")
	workers := flag.Int("workers", 8, "maximum simultaneous connections (1..8)")
	seed := flag.Int64("seed", 20261008, "deterministic sampling seed")
	flag.Parse()
	if *rangesPath == "" || *outDir == "" || *host == "" || strings.ContainsAny(*host, " /:\r\n\t") || *rate < 1 || *rate > 5 || *workers < 1 || *workers > 8 {
		return errors.New("valid -ranges, -out, -host, rate 1..5 and workers 1..8 required")
	}
	data, err := os.ReadFile(*rangesPath)
	if err != nil {
		return err
	}
	var networks []netip.Prefix
	for _, line := range strings.Fields(string(data)) {
		p, e := netip.ParsePrefix(line)
		if e != nil || !p.Addr().Is4() || p.Bits() < 12 || p.Bits() > 24 || p != p.Masked() {
			return fmt.Errorf("unsupported official range: %s", line)
		}
		networks = append(networks, p)
	}
	if len(networks) == 0 || len(networks) > 512 {
		return errors.New("need 1..512 official IPv4 ranges or subranges")
	}
	contains := func(ip netip.Addr) bool {
		for _, p := range networks {
			if p.Contains(ip) {
				return true
			}
		}
		return false
	}
	var controls []string
	if *seedPath != "" {
		b, e := os.ReadFile(*seedPath)
		if e != nil {
			return e
		}
		for _, value := range strings.Fields(string(b)) {
			ip, e := netip.ParseAddr(value)
			if e != nil || !contains(ip) {
				return fmt.Errorf("control outside official ranges: %s", value)
			}
			controls = append(controls, ip.String())
		}
	}
	if len(controls) > 16 {
		return errors.New("at most 16 control IPs")
	}
	journal, err := os.OpenFile(filepath.Join(*outDir, "observations.jsonl"), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return err
	}
	defer journal.Close()
	reportFile, err := os.OpenFile(filepath.Join(*outDir, "summary.json"), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return err
	}
	defer reportFile.Close()
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	ctx, cancel := context.WithTimeout(ctx, 25*time.Minute)
	defer cancel()
	started := time.Now().UTC()
	rng := rand.New(rand.NewSource(*seed))
	allowed := map[string]bool{}
	for _, code := range apac {
		allowed[code] = true
	}
	used := map[string]bool{}
	newIP := func(p netip.Prefix) string {
		base := binary.BigEndian.Uint32(p.Addr().AsSlice())
		for {
			value := base + uint32(1+rng.Intn(254))
			var b [4]byte
			binary.BigEndian.PutUint32(b[:], value)
			ip := netip.AddrFrom4(b).String()
			if !used[ip] {
				used[ip] = true
				return ip
			}
		}
	}
	var broad []string
	strata := 0
	for _, p := range networks {
		base := uint64(binary.BigEndian.Uint32(p.Addr().AsSlice()))
		size := uint64(1) << (32 - p.Bits())
		for offset := uint64(0); offset < size; offset += 4096 {
			blockSize := min(uint64(4096), size-offset)
			choices := rng.Perm(int(blockSize / 256))
			for _, n := range choices[:min(2, len(choices))] {
				var b [4]byte
				binary.BigEndian.PutUint32(b[:], uint32(base+offset+uint64(n*256)))
				broad = append(broad, newIP(netip.PrefixFrom(netip.AddrFrom4(b), 24)))
			}
			strata++
		}
	}
	for _, ip := range controls {
		if !used[ip] {
			used[ip] = true
			broad = append(broad, ip)
		}
	}
	if len(broad) > 1024 {
		return errors.New("sampling cap exceeded: at most 1024 broad probes")
	}
	var all []observation
	report := map[string]any{"started_at": started, "host": *host, "source": "executing Android/device network, direct sockets; system VPN routing unknown unless confirmed by user", "official_ranges": networks, "apac_allowlist": apac, "seed": *seed, "rate": *rate, "workers": *workers, "strata": strata, "broad_candidates": len(broad), "complete": false, "notes": []string{"Two sampled /24s per /20 or smaller block; NOT an exhaustive scan.", "APAC is based on observed colo codes, never IP geolocation. Unknown codes are excluded.", "3s combined TCP/TLS budget and a fresh 3s HTTP budget. HTTPS timings end at response headers.", "Short probes do not measure full-page download or prove a whole prefix works.", "Confirmation uses 3 addresses x 3 rounds; validation uses 2 NEW addresses x 3 rounds.", "No panel, DNS, proxy-environment, or network-setting changes."}}
	defer func() {
		report["finished_at"] = time.Now().UTC()
		report["request_count"] = len(all)
		if ctx.Err() != nil {
			report["error"] = ctx.Err().Error()
		}
		enc := json.NewEncoder(reportFile)
		enc.SetIndent("", "  ")
		enc.Encode(report)
		reportFile.Sync()
	}()
	ticker := time.NewTicker(time.Second / time.Duration(*rate))
	defer ticker.Stop()
	encoder := json.NewEncoder(journal)
	measure := func(ips []string, phase string, round int, trace bool) ([]observation, error) {
		order := append([]string{}, ips...)
		rng.Shuffle(len(order), func(i, j int) { order[i], order[j] = order[j], order[i] })
		jobs := make(chan string)
		results := make(chan observation)
		var wg sync.WaitGroup
		for w := 0; w < *workers; w++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for ip := range jobs {
					select {
					case <-ctx.Done():
						return
					case <-ticker.C:
					}
					o := probe(ctx, ip, *host, trace, allowed)
					o.Phase = phase
					o.Round = round
					results <- o
				}
			}()
		}
		go func() {
			defer close(jobs)
			for _, ip := range order {
				select {
				case <-ctx.Done():
					return
				case jobs <- ip:
				}
			}
		}()
		go func() { wg.Wait(); close(results) }()
		var rows []observation
		good, asian := 0, 0
		var writeErr error
		for o := range results {
			rows = append(rows, o)
			all = append(all, o)
			if e := encoder.Encode(o); e != nil && writeErr == nil {
				writeErr = e
				cancel()
			}
			if o.Success {
				good++
			}
			if o.Success && o.APAC {
				asian++
			}
			if len(rows)%50 == 0 {
				fmt.Printf("%s round=%d done=%d/%d success=%d apac=%d elapsed=%.1fs\n", phase, round, len(rows), len(order), good, asian, time.Since(started).Seconds())
			}
		}
		journal.Sync()
		fmt.Printf("%s round=%d done=%d/%d success=%d apac=%d elapsed=%.1fs\n", phase, round, len(rows), len(order), good, asian, time.Since(started).Seconds())
		if writeErr != nil {
			return rows, writeErr
		}
		return rows, ctx.Err()
	}
	pause := func() error {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(5 * time.Second):
			return nil
		}
	}
	fmt.Printf("host=%s official_ranges=%d strata=%d broad_samples=%d direct_local=true rate=%d/s workers=%d\n", *host, len(networks), strata, len(broad), *rate, *workers)
	rows, err := measure(broad, "region", 1, true)
	if err != nil {
		return err
	}
	regionCounts := map[string]int{}
	var candidates []observation
	for _, o := range rows {
		if o.Success {
			regionCounts[o.Colo]++
		}
		if o.Success && o.APAC {
			candidates = append(candidates, o)
		}
	}
	report["region_counts"] = regionCounts
	sort.Slice(candidates, func(i, j int) bool { return candidates[i].Total < candidates[j].Total })
	chosen := map[string]string{}
	var prefixes []string
	for _, o := range candidates {
		if len(prefixes) >= 32 {
			break
		}
		if _, ok := chosen[o.Prefix]; !ok {
			chosen[o.Prefix] = o.IP
			prefixes = append(prefixes, o.Prefix)
		}
	}
	controlSet := map[string]bool{}
	for _, ip := range controls {
		controlSet[ip] = true
	}
	for _, o := range candidates {
		if controlSet[o.IP] {
			if _, ok := chosen[o.Prefix]; !ok {
				chosen[o.Prefix] = o.IP
				prefixes = append(prefixes, o.Prefix)
			}
		}
	}
	if len(prefixes) == 0 {
		return errors.New("no APAC candidates observed; inspect region journal")
	}
	report["confirmation_prefixes"] = prefixes
	var confirmationIPs []string
	for _, prefix := range prefixes {
		p := netip.MustParsePrefix(prefix)
		confirmationIPs = append(confirmationIPs, chosen[prefix], newIP(p), newIP(p))
	}
	var confirmation []observation
	for round := 1; round <= 3; round++ {
		if err := pause(); err != nil {
			return err
		}
		r, e := measure(confirmationIPs, "confirmation", round, false)
		confirmation = append(confirmation, r...)
		if e != nil {
			return e
		}
	}
	ranked := summarize(confirmation)
	report["confirmation_ranked"] = ranked
	var selected []segment
	for _, s := range ranked {
		if s.Attempts == 9 && s.APACSuccesses == 9 {
			selected = append(selected, s)
			if len(selected) == 12 {
				break
			}
		}
	}
	if len(selected) == 0 {
		report["complete"] = true
		report["recommendations"] = []segment{}
		fmt.Println("No prefix passed all nine APAC business probes.")
		return nil
	}
	var validationIPs []string
	for _, s := range selected {
		p := netip.MustParsePrefix(s.Prefix)
		validationIPs = append(validationIPs, newIP(p), newIP(p))
	}
	var validation []observation
	for round := 1; round <= 3; round++ {
		if err := pause(); err != nil {
			return err
		}
		r, e := measure(validationIPs, "validation", round, false)
		validation = append(validation, r...)
		if e != nil {
			return e
		}
	}
	validationStats := summarize(validation)
	report["validation_ranked"] = validationStats
	valid := map[string]bool{}
	for _, s := range validationStats {
		if s.Attempts == 6 && s.APACSuccesses == 6 {
			valid[s.Prefix] = true
		}
	}
	var combined []observation
	for _, o := range append(confirmation, validation...) {
		if valid[o.Prefix] {
			combined = append(combined, o)
		}
	}
	recommendations := summarize(combined)
	report["recommendations"] = recommendations
	report["complete"] = true
	for _, s := range recommendations {
		fmt.Printf("candidate=%s business=%d/%d addresses=%d colo=%v median=%.1fms p90=%.1fms worst=%.1fms\n", s.Prefix, s.APACSuccesses, s.Attempts, len(s.IPs), s.Colos, s.Median, s.P90, s.Worst)
	}
	fmt.Println("summary:", filepath.Join(*outDir, "summary.json"))
	return nil
}

func probe(ctx context.Context, ip, host string, trace bool, allowed map[string]bool) (o observation) {
	o.IP = ip
	o.Prefix = netip.PrefixFrom(netip.MustParseAddr(ip), 24).Masked().String()
	o.At = time.Now().UTC()
	start := time.Now()
	defer func() {
		if !o.Success {
			o.Total = float64(time.Since(start)) / float64(time.Millisecond)
		}
	}()
	dialCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	tcp, err := (&net.Dialer{}).DialContext(dialCtx, "tcp", net.JoinHostPort(ip, "443"))
	o.TCP = float64(time.Since(start)) / float64(time.Millisecond)
	if err != nil {
		o.Error = "tcp: " + err.Error()
		return
	}
	defer tcp.Close()
	o.LocalIP = tcp.LocalAddr().(*net.TCPAddr).IP.String()
	stop := context.AfterFunc(ctx, func() { tcp.Close() })
	defer stop()
	tcp.SetDeadline(start.Add(3 * time.Second))
	conn := tls.Client(tcp, &tls.Config{ServerName: host, MinVersion: tls.VersionTLS12})
	tlsStart := time.Now()
	err = conn.HandshakeContext(dialCtx)
	o.TLS = float64(time.Since(tlsStart)) / float64(time.Millisecond)
	if err != nil {
		o.Error = "tls: " + err.Error()
		return
	}
	path := "/"
	if trace {
		path = "/cdn-cgi/trace"
	}
	httpStart := time.Now()
	conn.SetDeadline(httpStart.Add(3 * time.Second))
	_, err = fmt.Fprintf(conn, "GET %s HTTP/1.1\r\nHost: %s\r\nUser-Agent: TunnelManagerLab/1.0\r\nConnection: close\r\n\r\n", path, host)
	if err != nil {
		o.Error = "http write: " + err.Error()
		return
	}
	response, err := http.ReadResponse(bufio.NewReader(io.LimitReader(conn, 64<<10)), &http.Request{Method: "GET"})
	o.HTTP = float64(time.Since(httpStart)) / float64(time.Millisecond)
	o.Total = float64(time.Since(start)) / float64(time.Millisecond)
	if err != nil {
		o.Error = "http headers: " + err.Error()
		return
	}
	// Close TCP without draining the body: the business probe stops at headers.
	o.Status = response.StatusCode
	if trace {
		body, e := io.ReadAll(io.LimitReader(response.Body, 8192))
		if e != nil {
			o.Error = "trace body: " + e.Error()
			return
		}
		for _, line := range strings.Split(string(body), "\n") {
			if strings.HasPrefix(line, "colo=") {
				o.Colo = strings.TrimSpace(strings.TrimPrefix(line, "colo="))
			}
		}
	} else {
		ray := response.Header.Get("CF-Ray")
		if pos := strings.LastIndex(ray, "-"); pos >= 0 {
			o.Colo = ray[pos+1:]
		}
	}
	o.Success = o.Status == 200
	o.APAC = allowed[o.Colo]
	if !o.Success {
		o.Error = fmt.Sprintf("HTTP %d", o.Status)
	}
	return
}

func summarize(rows []observation) []segment {
	groups := map[string][]observation{}
	for _, o := range rows {
		groups[o.Prefix] = append(groups[o.Prefix], o)
	}
	var out []segment
	for prefix, group := range groups {
		s := segment{Prefix: prefix, Attempts: len(group), Colos: map[string]int{}}
		ips := map[string]bool{}
		var times []float64
		for _, o := range group {
			ips[o.IP] = true
			if o.Total > s.Worst {
				s.Worst = o.Total
			}
			if o.Success {
				s.Successes++
				times = append(times, o.Total)
				s.Colos[o.Colo]++
				if o.APAC {
					s.APACSuccesses++
				}
			}
		}
		for ip := range ips {
			s.IPs = append(s.IPs, ip)
		}
		sort.Strings(s.IPs)
		sort.Float64s(times)
		if n := len(times); n > 0 {
			s.Median = times[n/2]
			if n%2 == 0 {
				s.Median = (times[n/2-1] + times[n/2]) / 2
			}
			s.P90 = times[(9*n+9)/10-1]
		}
		out = append(out, s)
	}
	sort.Slice(out, func(i, j int) bool {
		a, b := out[i], out[j]
		if a.APACSuccesses*b.Attempts != b.APACSuccesses*a.Attempts {
			return a.APACSuccesses*b.Attempts > b.APACSuccesses*a.Attempts
		}
		if a.P90 != b.P90 {
			return a.P90 < b.P90
		}
		if a.Median != b.Median {
			return a.Median < b.Median
		}
		return a.Prefix < b.Prefix
	})
	return out
}
