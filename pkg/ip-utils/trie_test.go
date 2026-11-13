package ip_utils

import (
	"net/netip"
	"testing"
)

func TestRuleSetMatch(t *testing.T) {
	set := NewRuleSet[string]()
	prefixes, err := Prefixes("10.0.0.0/8")
	if err != nil {
		t.Fatal(err)
	}
	set.InsertPrefixes(prefixes, 0, "rule-1")

	v, ok := set.Match(netip.MustParseAddr("10.1.2.3"))
	if !ok || v != "rule-1" {
		t.Fatalf("expected rule-1, got %q ok=%v", v, ok)
	}
}

func TestRuleSetIPv6AndNoMatch(t *testing.T) {
	set := NewRuleSet[string]()
	prefixes, err := Prefixes("2001:db8::/32")
	if err != nil {
		t.Fatal(err)
	}
	set.InsertPrefixes(prefixes, 0, "v6")

	if _, ok := set.Match(netip.MustParseAddr("198.51.100.1")); ok {
		t.Fatal("unexpected IPv4 match")
	}
	v, ok := set.Match(netip.MustParseAddr("2001:db8::1"))
	if !ok || v != "v6" {
		t.Fatalf("expected v6, got %q ok=%v", v, ok)
	}
}

func TestPrefixesRange(t *testing.T) {
	prefixes, err := Prefixes("192.168.1.1-192.168.1.254")
	if err != nil {
		t.Fatal(err)
	}
	if len(prefixes) == 0 {
		t.Fatal("expected prefixes for range")
	}

	set := NewRuleSet[string]()
	set.InsertPrefixes(prefixes, 0, "range")
	if _, ok := set.Match(netip.MustParseAddr("192.168.1.200")); !ok {
		t.Fatal("expected range match")
	}
	if _, ok := set.Match(netip.MustParseAddr("192.168.2.1")); ok {
		t.Fatal("expected no match outside range")
	}
}

func TestPrefixesErrors(t *testing.T) {
	for _, value := range []string{
		"",
		"bad",
		"10.0.0.0/bad",
		"1.1.1.10-1.1.1.1",
		"1.1.1.1-::1",
		"bad-1.1.1.1",
		"1.1.1.1-bad",
	} {
		if _, err := Prefixes(value); err == nil {
			t.Fatalf("expected error for %q", value)
		}
	}
}

func TestRuleSetDenyPriority(t *testing.T) {
	deny := NewRuleSet[string]()
	allow := NewRuleSet[string]()

	denyPrefixes, _ := Prefixes("10.0.0.0/8")
	allowPrefixes, _ := Prefixes("10.0.0.0/16")

	deny.InsertPrefixes(denyPrefixes, 0, "deny")
	allow.InsertPrefixes(allowPrefixes, 0, "allow")

	addr := netip.MustParseAddr("10.0.1.1")
	if _, ok := deny.Match(addr); !ok {
		t.Fatal("deny should match first in application logic")
	}
}
