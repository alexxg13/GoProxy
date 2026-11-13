package ip_utils

import "net/netip"

func ExpandLoopbackPrefixes(prefixes []netip.Prefix) []netip.Prefix {
	seen := make(map[string]struct{}, len(prefixes)+2)
	out := make([]netip.Prefix, 0, len(prefixes)+2)

	add := func(p netip.Prefix) {
		key := p.String()
		if _, ok := seen[key]; ok {
			return
		}
		seen[key] = struct{}{}
		out = append(out, p.Masked())
	}

	for _, p := range prefixes {
		add(p)
		addr := p.Addr().Unmap()
		if !addr.IsLoopback() {
			continue
		}
		if addr.Is4() {
			add(netip.PrefixFrom(netip.MustParseAddr("::1"), 128))
			continue
		}
		if addr.Is6() {
			add(netip.PrefixFrom(netip.MustParseAddr("127.0.0.1"), 32))
		}
	}

	return out
}
