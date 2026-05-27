package ip_utils

import (
	"net/netip"
	"testing"
)

func TestParseSingleIP(t *testing.T) {
	m, err := Parse("192.168.1.100")
	if err != nil {
		t.Fatal(err)
	}
	if m.Description() != "192.168.1.100" {
		t.Fatalf("unexpected description: %s", m.Description())
	}
	addr := netip.MustParseAddr("192.168.1.100")
	if !m.Match(addr) {
		t.Fatal("expected match")
	}
	if m.Match(netip.MustParseAddr("192.168.1.101")) {
		t.Fatal("expected no match")
	}
}

func TestParseCIDR(t *testing.T) {
	m, err := Parse("10.0.0.0/8")
	if err != nil {
		t.Fatal(err)
	}
	if m.Description() != "10.0.0.0/8" {
		t.Fatalf("unexpected description: %s", m.Description())
	}
	if !m.Match(netip.MustParseAddr("10.1.2.3")) {
		t.Fatal("expected match in subnet")
	}
}

func TestParseRange(t *testing.T) {
	m, err := Parse("192.168.1.1-192.168.1.10")
	if err != nil {
		t.Fatal(err)
	}
	if m.Description() != "192.168.1.1-192.168.1.10" {
		t.Fatalf("unexpected description: %s", m.Description())
	}
	if !m.Match(netip.MustParseAddr("192.168.1.5")) {
		t.Fatal("expected match in range")
	}
	if m.Match(netip.MustParseAddr("192.168.1.11")) {
		t.Fatal("expected no match outside range")
	}
}

func TestParseErrors(t *testing.T) {
	for _, value := range []string{
		"",
		"not-ip",
		"10.0.0.0/bad",
		"1.1.1.10-1.1.1.1",
		"1.1.1.1-::1",
		"bad-1.1.1.1",
		"1.1.1.1-bad",
	} {
		if _, err := Parse(value); err == nil {
			t.Fatalf("expected error for %q", value)
		}
	}
}
