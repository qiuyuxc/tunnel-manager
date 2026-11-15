// PROTOTYPE: compare one HTTP-only sample against repeated full HTTPS timings.
// Selection uses training observations; a fresh holdout measures the frozen picks.
// This standalone program never reads credentials, updates DNS, or runs the panel.
package main

import (
	"bufio"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"math/rand"
	"net"
	"net/netip"
	"os"
	"os/signal"
	"sort"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
)

type input struct {
	Host             string              `json:"host"`
	SNI              string              `json:"sni"`
	Path             string              `json:"path"`
	Statuses         string              `json:"statuses"`
	Timeout          int                 `json:"timeout_seconds"`
	Targets          string              `json:"ip_targets"`
	Current          []string            `json:"current_ips"`
	Extra            []string            `json:"extra_ips"`
	ComparisonGroups map[string][]string `json:"comparison_groups,omitempty"`
}

type observation struct {
	IP         string    `json:"ip"`
	Phase      string    `json:"phase"`
	Round      int       `json:"round"`
	At         time.Time `json:"at"`
	Status     int       `json:"status"`
	Success    bool      `json:"success"`
	TCP        float64   `json:"tcp_ms"`
	TLS        float64   `json:"tls_ms"`
	HTTP       float64   `json:"http_ms"`
	Total      float64   `json:"total_ms"`
	ErrorPhase string    `json:"error_phase,omitempty"`
	Error      string    `json:"error,omitempty"`
}

type stats struct {
	IP          string  `json:"ip"`
	Prefix      string  `json:"prefix"`
	Attempts    int     `json:"attempts"`
	Successes   int     `json:"successes"`
	MedianTotal float64 `json:"successful_median_total_ms"`
	MedianHTTP  float64 `json:"successful_median_http_ms"`
	WorstTotal  float64 `json:"worst_observed_total_ms"`
}

type evaluation struct {
	IPs         []string `json:"ips"`
	Attempts    int      `json:"attempts"`
	Successes   int      `json:"successes"`
	SuccessRate float64  `json:"success_rate"`
	Median      float64  `json:"successful_median_total_ms"`
	Worst       float64  `json:"worst_observed_total_ms"`
	PerIP       []stats  `json:"per_ip"`
}

type report struct {
	Started             time.Time             `json:"started_at"`
	Finished            time.Time             `json:"finished_at"`
	Input               input                 `json:"input"`
	Seed                int64                 `json:"seed"`
	Candidates          []string              `json:"candidates"`
	Training            []observation         `json:"training"`
	Ranked              []stats               `json:"ranked_training"`
	Old                 []string              `json:"old_selection"`
	Full                []string              `json:"full_timing_only_selection"`
	New                 []string              `json:"new_selection"`
	Holdout             []observation         `json:"holdout"`
	Evaluation          map[string]evaluation `json:"evaluation"`
	ReferenceEvaluation map[string]evaluation `json:"reference_evaluation,omitempty"`
	Error               string                `json:"error,omitempty"`
	Notes               []string              `json:"notes"`
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	inPath := flag.String("input", "", "sanitized JSON input (no credentials)")
	outPath := flag.String("out", "ip-selector-prototype-result.json", "new result file; refuses overwrite")
	perPrefix := flag.Int("per-prefix", 2, "random samples per input /24 or smaller prefix")
	trainRounds := flag.Int("train-rounds", 5, "observations per candidate used for selection")
	holdoutRounds := flag.Int("holdout-rounds", 3, "fresh observations per frozen selected IP")
	top := flag.Int("top", 3, "number of selected IPs")
	rate := flag.Int("rate", 2, "maximum new connections per second")
	workers := flag.Int("workers", 4, "maximum concurrent connections")
	gap := flag.Duration("gap", 10*time.Second, "pause between rounds")
	seed := flag.Int64("seed", 20261008, "repeatable sampling and shuffled order")
	flag.Parse()
	if *inPath == "" {
		return errors.New("-input is required")
	}
	if *perPrefix < 1 || *perPrefix > 4 || *top < 1 || *top > 5 || *trainRounds < 2 || *trainRounds > 8 || *holdoutRounds < 1 || *holdoutRounds > 5 || *rate < 1 || *rate > 10 || *workers < 1 || *workers > 8 || *gap < 0 || *gap > time.Minute {
		return errors.New("prototype limits: per-prefix 1..4, top 1..5, train 2..8, holdout 1..5, rate 1..10, workers 1..8, gap 0..60s")
	}
	data, err := os.ReadFile(*inPath)
	if err != nil {
		return err
	}
	var cfg input
	if err := json.Unmarshal(data, &cfg); err != nil {
		return err
	}
	if cfg.SNI == "" {
		cfg.SNI = cfg.Host
	}
	if cfg.Path == "" {
		cfg.Path = "/"
	}
	if cfg.Host == "" || strings.ContainsAny(cfg.Host+cfg.SNI, "\r\n /:\t") || !strings.HasPrefix(cfg.Path, "/") || strings.ContainsAny(cfg.Path, "\r\n \t") {
		return errors.New("invalid host/SNI/path")
	}
	if cfg.Timeout < 1 || cfg.Timeout > 10 {
		return errors.New("timeout_seconds must be 1..10")
	}
	accepted := map[int]bool{}
	if cfg.Statuses == "" {
		cfg.Statuses = "200"
	}
	for _, text := range strings.Split(cfg.Statuses, ",") {
		code, err := strconv.Atoi(strings.TrimSpace(text))
		if err != nil || code < 200 || code > 599 {
			return errors.New("invalid status")
		}
		accepted[code] = true
	}
	rng := rand.New(rand.NewSource(*seed))
	if len(cfg.ComparisonGroups) > 8 {
		return errors.New("prototype supports at most 8 frozen comparison groups")
	}
	for name, ips := range cfg.ComparisonGroups {
		if strings.TrimSpace(name) == "" || len(ips) == 0 || len(ips) > 32 || len(unique(ips)) != len(ips) {
			return errors.New("comparison groups need a name and 1..32 unique IPs")
		}
	}
	ips, err := sample(cfg, *perPrefix, rng)
	if err != nil {
		return err
	}
	if len(ips) < *top || len(ips) > 96 {
		return fmt.Errorf("need %d..96 candidates, got %d", *top, len(ips))
	}
	file, err := os.OpenFile(*outPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return err
	}
	defer file.Close()
	r := report{Started: time.Now().UTC(), Input: cfg, Seed: *seed, Candidates: ips, Notes: []string{
		"PROTOTYPE; local BGP vantage only. No claim of three-carrier improvement.",
		"Cold TCP/TLS; verified certificate; HTTP/1.1 GET; stop after status line; no redirect or body download.",
		"Queue wait excluded. TCP+TLS share one timeout; HTTP has a fresh timeout, matching the panel stages.",
		"Old = first-round HTTP time. Full timing only = first-round complete time. New = repeated success ratio, median complete time, worst complete time, bounded /24 diversity.",
		"Diversity is a heuristic, not proof of independent paths. No DNS changes or long-term hysteresis validation.",
		"Latest selected IPs are candidate seeds only; old measurements do not enter the scoring.",
		"Optional comparison groups are frozen in the input before measuring; all their IPs join fresh holdout rounds.",
	}}
	checkpoint := func() {
		file.Seek(0, 0)
		file.Truncate(0)
		enc := json.NewEncoder(file)
		enc.SetIndent("", "  ")
		if e := enc.Encode(r); e != nil {
			fmt.Fprintln(os.Stderr, "checkpoint:", e)
		}
		file.Sync()
	}
	defer checkpoint()
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	ctx, cancel := context.WithTimeout(ctx, 15*time.Minute)
	defer cancel()
	ticker := time.NewTicker(time.Second / time.Duration(*rate))
	defer ticker.Stop()
	request := fmt.Sprintf("GET %s HTTP/1.1\r\nHost: %s\r\nUser-Agent: TunnelManagerLab/1.0\r\nConnection: close\r\n\r\n", cfg.Path, cfg.Host)
	measure := func(targets []string, phase string, round int) []observation {
		order := append([]string(nil), targets...)
		rng.Shuffle(len(order), func(i, j int) { order[i], order[j] = order[j], order[i] })
		results := make([]observation, len(order))
		jobs := make(chan int)
		var wg sync.WaitGroup
		for w := 0; w < *workers; w++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for i := range jobs {
					select {
					case <-ctx.Done():
						return
					case <-ticker.C:
					}
					results[i] = probe(ctx, order[i], cfg.SNI, request, time.Duration(cfg.Timeout)*time.Second, accepted)
					results[i].Phase = phase
					results[i].Round = round
				}
			}()
		}
		for i := range order {
			select {
			case <-ctx.Done():
				break
			case jobs <- i:
			}
		}
		close(jobs)
		wg.Wait()
		valid := results[:0]
		success := 0
		for _, o := range results {
			if o.IP != "" {
				valid = append(valid, o)
				if o.Success {
					success++
				}
			}
		}
		fmt.Printf("%s round=%d requests=%d success=%d elapsed=%.1fs\n", phase, round, len(valid), success, time.Since(r.Started).Seconds())
		return valid
	}
	pause := func() error {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(*gap):
			return nil
		}
	}
	fmt.Printf("host=%s candidates=%d prefixes=%d train=%d holdout=%d rate=%d/s timeout=%ds\n", cfg.Host, len(ips), prefixCount(ips), *trainRounds, *holdoutRounds, *rate, cfg.Timeout)
	for round := 1; round <= *trainRounds; round++ {
		r.Training = append(r.Training, measure(ips, "training", round)...)
		checkpoint()
		if ctx.Err() != nil {
			r.Error = ctx.Err().Error()
			return ctx.Err()
		}
		if round < *trainRounds {
			if err := pause(); err != nil {
				r.Error = err.Error()
				return err
			}
		}
	}
	r.Old = oneShot(r.Training, *top, false)
	r.Full = oneShot(r.Training, *top, true)
	r.Ranked = summarize(r.Training)
	r.New = stablePick(r.Ranked, *top)
	fmt.Printf("old_http=%v\nfull_timing_only=%v\nrepeated_diverse=%v\n", r.Old, r.Full, r.New)
	checkpoint()
	union := unique(append(append(append([]string{}, r.Old...), r.Full...), r.New...))
	for _, name := range groupNames(cfg.ComparisonGroups) {
		union = unique(append(union, cfg.ComparisonGroups[name]...))
	}
	if len(union) == 0 {
		return errors.New("no successful candidates")
	}
	for round := 1; round <= *holdoutRounds; round++ {
		if err := pause(); err != nil {
			r.Error = err.Error()
			return err
		}
		r.Holdout = append(r.Holdout, measure(union, "holdout", round)...)
		checkpoint()
		if ctx.Err() != nil {
			r.Error = ctx.Err().Error()
			return ctx.Err()
		}
	}
	r.Evaluation = map[string]evaluation{"old_http": evaluate(r.Holdout, r.Old), "full_timing_only": evaluate(r.Holdout, r.Full), "repeated_diverse": evaluate(r.Holdout, r.New)}
	if len(cfg.ComparisonGroups) > 0 {
		r.ReferenceEvaluation = make(map[string]evaluation)
		for _, name := range groupNames(cfg.ComparisonGroups) {
			r.ReferenceEvaluation[name] = evaluate(r.Holdout, cfg.ComparisonGroups[name])
		}
	}
	r.Finished = time.Now().UTC()
	for _, name := range []string{"old_http", "full_timing_only", "repeated_diverse"} {
		e := r.Evaluation[name]
		fmt.Printf("%s holdout=%d/%d median=%.1fms worst=%.1fms\n", name, e.Successes, e.Attempts, e.Median, e.Worst)
	}
	for _, name := range groupNames(cfg.ComparisonGroups) {
		e := r.ReferenceEvaluation[name]
		fmt.Printf("reference %s holdout=%d/%d median=%.1fms worst=%.1fms\n", name, e.Successes, e.Attempts, e.Median, e.Worst)
	}
	fmt.Println("result:", *outPath)
	return nil
}

func public(ip netip.Addr) bool {
	return ip.Is4() && ip.IsGlobalUnicast() && !ip.IsPrivate() && !ip.IsLoopback() && !ip.IsLinkLocalUnicast() && !ip.IsUnspecified()
}

func sample(cfg input, n int, rng *rand.Rand) ([]string, error) {
	var out []string
	seeds := append(append([]string{}, cfg.Current...), cfg.Extra...)
	for _, name := range groupNames(cfg.ComparisonGroups) {
		seeds = append(seeds, cfg.ComparisonGroups[name]...)
	}
	for _, text := range seeds {
		ip, err := netip.ParseAddr(text)
		if err != nil || !public(ip) {
			return nil, fmt.Errorf("invalid public IPv4: %s", text)
		}
		out = append(out, ip.String())
	}
	seen := map[string]bool{}
	for _, text := range strings.FieldsFunc(cfg.Targets, func(r rune) bool { return strings.ContainsRune(",\n\r\t ", r) }) {
		p, err := netip.ParsePrefix(text)
		if err != nil {
			ip, e := netip.ParseAddr(text)
			if e != nil || !public(ip) {
				return nil, fmt.Errorf("invalid target: %s", text)
			}
			out = append(out, ip.String())
			continue
		}
		p = p.Masked()
		if !public(p.Addr()) || p.Bits() < 24 || p.Bits() > 32 {
			return nil, fmt.Errorf("prototype supports public IPv4 /24../32 only: %s", text)
		}
		if seen[p.String()] {
			continue
		}
		seen[p.String()] = true
		size := 1 << (32 - p.Bits())
		offsets := rng.Perm(size)
		count := 0
		for _, offset := range offsets {
			if size > 2 && (offset == 0 || offset == size-1) {
				continue
			}
			ip := p.Addr()
			for step := 0; step < offset; step++ {
				ip = ip.Next()
			}
			out = append(out, ip.String())
			count++
			if count >= n {
				break
			}
		}
	}
	return unique(out), nil
}

func groupNames(groups map[string][]string) []string {
	names := make([]string, 0, len(groups))
	for name := range groups {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func probe(ctx context.Context, ip, sni, request string, timeout time.Duration, accepted map[int]bool) (o observation) {
	o.IP = ip
	o.At = time.Now().UTC()
	start := time.Now()
	defer func() { o.Total = float64(time.Since(start)) / float64(time.Millisecond) }()
	dialCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	tcp, err := (&net.Dialer{}).DialContext(dialCtx, "tcp", net.JoinHostPort(ip, "443"))
	o.TCP = float64(time.Since(start)) / float64(time.Millisecond)
	if err != nil {
		o.ErrorPhase = "tcp"
		o.Error = err.Error()
		return
	}
	defer tcp.Close()
	stop := context.AfterFunc(ctx, func() { tcp.Close() })
	defer stop()
	tcp.SetDeadline(start.Add(timeout))
	conn := tls.Client(tcp, &tls.Config{ServerName: sni, MinVersion: tls.VersionTLS12})
	tlsStart := time.Now()
	err = conn.HandshakeContext(dialCtx)
	o.TLS = float64(time.Since(tlsStart)) / float64(time.Millisecond)
	if err != nil {
		o.ErrorPhase = "tls"
		o.Error = err.Error()
		return
	}
	httpStart := time.Now()
	conn.SetDeadline(httpStart.Add(timeout))
	_, err = io.WriteString(conn, request)
	if err == nil {
		var line string
		line, err = bufio.NewReader(io.LimitReader(conn, 4096)).ReadString('\n')
		if err == nil {
			fields := strings.Fields(line)
			if len(fields) < 2 || !strings.HasPrefix(fields[0], "HTTP/") {
				err = errors.New("invalid status line")
			} else {
				o.Status, err = strconv.Atoi(fields[1])
			}
		}
	}
	o.HTTP = float64(time.Since(httpStart)) / float64(time.Millisecond)
	if err != nil {
		o.ErrorPhase = "http"
		o.Error = err.Error()
		return
	}
	o.Success = accepted[o.Status]
	if !o.Success {
		o.ErrorPhase = "status"
		o.Error = fmt.Sprintf("unaccepted HTTP %d", o.Status)
	}
	return
}

func prefix(ip string) string { return netip.PrefixFrom(netip.MustParseAddr(ip), 24).Masked().String() }
func prefixCount(ips []string) int {
	m := map[string]bool{}
	for _, ip := range ips {
		m[prefix(ip)] = true
	}
	return len(m)
}
func unique(ips []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, ip := range ips {
		if !seen[ip] {
			seen[ip] = true
			out = append(out, ip)
		}
	}
	return out
}
func median(v []float64) float64 {
	if len(v) == 0 {
		return 0
	}
	v = append([]float64(nil), v...)
	sort.Float64s(v)
	n := len(v)
	if n%2 == 1 {
		return v[n/2]
	}
	return (v[n/2-1] + v[n/2]) / 2
}

func summarize(obs []observation) []stats {
	groups := map[string][]observation{}
	for _, o := range obs {
		groups[o.IP] = append(groups[o.IP], o)
	}
	var out []stats
	for ip, rows := range groups {
		s := stats{IP: ip, Prefix: prefix(ip), Attempts: len(rows)}
		var total, http []float64
		for _, o := range rows {
			if o.Total > s.WorstTotal {
				s.WorstTotal = o.Total
			}
			if o.Success {
				s.Successes++
				total = append(total, o.Total)
				http = append(http, o.HTTP)
			}
		}
		s.MedianTotal = median(total)
		s.MedianHTTP = median(http)
		out = append(out, s)
	}
	sort.Slice(out, func(i, j int) bool {
		a, b := out[i], out[j]
		if a.Successes*b.Attempts != b.Successes*a.Attempts {
			return a.Successes*b.Attempts > b.Successes*a.Attempts
		}
		if a.MedianTotal != b.MedianTotal {
			return a.MedianTotal < b.MedianTotal
		}
		if a.WorstTotal != b.WorstTotal {
			return a.WorstTotal < b.WorstTotal
		}
		return a.IP < b.IP
	})
	return out
}

func oneShot(obs []observation, k int, full bool) []string {
	var rows []observation
	for _, o := range obs {
		if o.Round == 1 && o.Success {
			rows = append(rows, o)
		}
	}
	sort.Slice(rows, func(i, j int) bool {
		a, b := rows[i].HTTP, rows[j].HTTP
		if full {
			a, b = rows[i].Total, rows[j].Total
		}
		if a != b {
			return a < b
		}
		return rows[i].IP < rows[j].IP
	})
	var out []string
	for _, o := range rows {
		out = append(out, o.IP)
		if len(out) == k {
			break
		}
	}
	return out
}

func stablePick(ranked []stats, k int) []string {
	if len(ranked) == 0 || ranked[0].Successes == 0 {
		return nil
	}
	best := ranked[0]
	ceiling := best.MedianTotal * 1.5
	if ceiling < best.MedianTotal+100 {
		ceiling = best.MedianTotal + 100
	}
	var selected []string
	used := map[string]bool{}
	for _, s := range ranked {
		if s.Successes*best.Attempts != best.Successes*s.Attempts || s.MedianTotal > ceiling {
			continue
		}
		if !used[s.Prefix] {
			selected = append(selected, s.IP)
			used[s.Prefix] = true
			if len(selected) == k {
				return selected
			}
		}
	}
	// Prefer a fast, stable duplicate prefix to a slow/unreliable diversity filler.
	for _, s := range ranked {
		if s.Successes == 0 {
			continue
		}
		already := false
		for _, ip := range selected {
			if ip == s.IP {
				already = true
			}
		}
		if !already {
			selected = append(selected, s.IP)
			if len(selected) == k {
				break
			}
		}
	}
	return selected
}

func evaluate(obs []observation, ips []string) evaluation {
	e := evaluation{IPs: ips}
	wanted := map[string]bool{}
	for _, ip := range ips {
		wanted[ip] = true
	}
	var rows []observation
	var times []float64
	for _, o := range obs {
		if !wanted[o.IP] {
			continue
		}
		rows = append(rows, o)
		e.Attempts++
		if o.Total > e.Worst {
			e.Worst = o.Total
		}
		if o.Success {
			e.Successes++
			times = append(times, o.Total)
		}
	}
	if e.Attempts > 0 {
		e.SuccessRate = float64(e.Successes) / float64(e.Attempts)
	}
	e.Median = median(times)
	e.PerIP = summarize(rows)
	return e
}
