package services

import "testing"

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
