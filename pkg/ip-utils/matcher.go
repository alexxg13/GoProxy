package ip_utils

import (
	"fmt"
	"net/netip"
	"strings"
)

type Matcher interface {
	Match(addr netip.Addr) bool
	Description() string
}

type singleMatcher struct {
	addr netip.Addr
}

func (m singleMatcher) Match(addr netip.Addr) bool {
	return m.addr.Compare(addr.Unmap()) == 0
}

func (m singleMatcher) Description() string {
	return m.addr.String()
}

type prefixMatcher struct {
	prefix netip.Prefix
}

func (m prefixMatcher) Match(addr netip.Addr) bool {
	return m.prefix.Contains(addr.Unmap())
}

func (m prefixMatcher) Description() string {
	return m.prefix.String()
}

type rangeMatcher struct {
	from netip.Addr
	to   netip.Addr
}

func (m rangeMatcher) Match(addr netip.Addr) bool {
	unmapped := addr.Unmap()
	return m.from.Compare(unmapped) <= 0 && m.to.Compare(unmapped) >= 0
}

func (m rangeMatcher) Description() string {
	return fmt.Sprintf("%s-%s", m.from, m.to)
}

func Parse(value string) (Matcher, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil, fmt.Errorf("empty matcher")
	}

	if strings.Contains(trimmed, "/") {
		prefix, err := netip.ParsePrefix(trimmed)
		if err != nil {
			return nil, fmt.Errorf("parse CIDR %q: %w", trimmed, err)
		}
		return prefixMatcher{prefix: prefix.Masked()}, nil
	}

	if strings.Contains(trimmed, "-") {
		parts := strings.SplitN(trimmed, "-", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid IP range %q", trimmed)
		}

		from, err := netip.ParseAddr(strings.TrimSpace(parts[0]))
		if err != nil {
			return nil, fmt.Errorf("parse range start %q: %w", parts[0], err)
		}

		to, err := netip.ParseAddr(strings.TrimSpace(parts[1]))
		if err != nil {
			return nil, fmt.Errorf("parse range end %q: %w", parts[1], err)
		}

		from = from.Unmap()
		to = to.Unmap()
		if from.BitLen() != to.BitLen() {
			return nil, fmt.Errorf("range endpoints must use the same address family")
		}
		if from.Compare(to) > 0 {
			return nil, fmt.Errorf("range start must be <= range end")
		}

		return rangeMatcher{from: from, to: to}, nil
	}

	addr, err := netip.ParseAddr(trimmed)
	if err != nil {
		return nil, fmt.Errorf("parse IP %q: %w", trimmed, err)
	}

	return singleMatcher{addr: addr.Unmap()}, nil
}
