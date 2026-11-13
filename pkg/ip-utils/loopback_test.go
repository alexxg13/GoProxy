package ip_utils

import (
	"net/netip"
	"testing"
)

func TestExpandLoopbackPrefixes(t *testing.T) {
	prefixes, err := Prefixes("127.0.0.1")
	if err != nil {
		t.Fatal(err)
	}
	expanded := ExpandLoopbackPrefixes(prefixes)

	v6 := netip.MustParseAddr("::1")
	matched := false
	for _, p := range expanded {
		if p.Contains(v6) {
			matched = true
			break
		}
	}
	if !matched {
		t.Fatal("127.0.0.1 allow rule should also match ::1")
	}
}

func TestExpandLoopbackPrefixesIPv6AndNonLoopback(t *testing.T) {
	prefixes, err := Prefixes("::1")
	if err != nil {
		t.Fatal(err)
	}
	expanded := ExpandLoopbackPrefixes(prefixes)
	v4 := netip.MustParseAddr("127.0.0.1")
	matched := false
	for _, p := range expanded {
		if p.Contains(v4) {
			matched = true
			break
		}
	}
	if !matched {
		t.Fatal("::1 allow rule should also match 127.0.0.1")
	}

	prefixes, err = Prefixes("10.0.0.1")
	if err != nil {
		t.Fatal(err)
	}
	expanded = ExpandLoopbackPrefixes(prefixes)
	if len(expanded) != 1 {
		t.Fatalf("non-loopback prefix should not expand, got %d", len(expanded))
	}
}
