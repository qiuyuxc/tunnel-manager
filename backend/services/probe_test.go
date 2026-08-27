package services

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestClassifyProbe(t *testing.T) {
	cases := []struct {
		name string
		code int
		err  error
		want string
	}{
		{"ok", 200, nil, "ok"},
		{"redirect counts as reachable", 301, nil, "ok"},
		{"client error still reached origin", 404, nil, "ok"},
		{"server error", 502, nil, "warn"},
		{"timeout", 0, context.DeadlineExceeded, "down"},
		{"network failure", 0, errors.New("dial tcp: connection refused"), "down"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := ClassifyProbe(tc.code, tc.err); got != tc.want {
				t.Fatalf("ClassifyProbe(%d, %v) = %q, want %q", tc.code, tc.err, got, tc.want)
			}
		})
	}
}

func TestDownsample(t *testing.T) {
	hbs := make([]Heartbeat, 0, 250)
	base := time.UnixMilli(0)
	for i := 0; i < 250; i++ {
		hbs = append(hbs, Heartbeat{T: base.Add(time.Duration(i) * time.Minute).UnixMilli(), S: "ok"})
	}
	got := Downsample(hbs, 40)
	if len(got) != 40 {
		t.Fatalf("len = %d, want 40", len(got))
	}
	for i := 1; i < len(got); i++ {
		if got[i].T < got[i-1].T {
			t.Fatalf("order broken at %d", i)
		}
	}
	if len(Downsample(hbs, 0)) != len(hbs) || len(Downsample(hbs[:5], 40)) != 5 {
		t.Fatalf("edge cases should pass through")
	}
}

func TestUptimePct(t *testing.T) {
	mk := func(states ...string) []Heartbeat {
		out := make([]Heartbeat, 0, len(states))
		for _, s := range states {
			out = append(out, Heartbeat{S: s})
		}
		return out
	}
	if got := UptimePct(nil); got != 0 {
		t.Fatalf("empty = %v", got)
	}
	if got := UptimePct(mk("ok", "ok", "warn", "down")); got != 50 {
		t.Fatalf("50%% case = %v", got)
	}
}
