package ip_utils

import (
	"fmt"
	"net/netip"
	"strings"
)

type candidate[T any] struct {
	order int
	value T
}

type trieNode[T any] struct {
	zero      *trieNode[T]
	one       *trieNode[T]
	candidate *candidate[T]
}

type RuleSet[T any] struct {
	v4 *trieNode[T]
	v6 *trieNode[T]
}

func NewRuleSet[T any]() *RuleSet[T] {
	return &RuleSet[T]{
		v4: &trieNode[T]{},
		v6: &trieNode[T]{},
	}
}

func (s *RuleSet[T]) InsertPrefixes(prefixes []netip.Prefix, order int, value T) {
	for _, prefix := range prefixes {
		addr := prefix.Addr().Unmap()
		root := s.root(addr)
		if root == nil {
			continue
		}

		node := root
		for bit := 0; bit < prefix.Bits(); bit++ {
			if bitAt(addr, bit) == 0 {
				if node.zero == nil {
					node.zero = &trieNode[T]{}
				}
				node = node.zero
				continue
			}

			if node.one == nil {
				node.one = &trieNode[T]{}
			}
			node = node.one
		}

		if node.candidate == nil || order < node.candidate.order {
			node.candidate = &candidate[T]{
				order: order,
				value: value,
			}
		}
	}
}

func (s *RuleSet[T]) Match(addr netip.Addr) (T, bool) {
	var zero T

	addr = addr.Unmap()
	root := s.root(addr)
	if root == nil {
		return zero, false
	}

	node := root
	var best *candidate[T]
	if node.candidate != nil {
		best = node.candidate
	}

	for bit := 0; bit < addr.BitLen(); bit++ {
		if bitAt(addr, bit) == 0 {
			node = node.zero
		} else {
			node = node.one
		}
		if node == nil {
			break
		}
		if node.candidate != nil && (best == nil || node.candidate.order < best.order) {
			best = node.candidate
		}
	}

	if best == nil {
		return zero, false
	}

	return best.value, true
}

func (s *RuleSet[T]) root(addr netip.Addr) *trieNode[T] {
	switch addr.BitLen() {
	case 32:
		return s.v4
	case 128:
		return s.v6
	default:
		return nil
	}
}

func Prefixes(value string) ([]netip.Prefix, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil, fmt.Errorf("empty matcher")
	}

	if strings.Contains(trimmed, "/") {
		prefix, err := netip.ParsePrefix(trimmed)
		if err != nil {
			return nil, fmt.Errorf("parse CIDR %q: %w", trimmed, err)
		}
		return []netip.Prefix{prefix.Masked()}, nil
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

		return rangeToPrefixes(from.Unmap(), to.Unmap())
	}

	addr, err := netip.ParseAddr(trimmed)
	if err != nil {
		return nil, fmt.Errorf("parse IP %q: %w", trimmed, err)
	}

	addr = addr.Unmap()
	return []netip.Prefix{netip.PrefixFrom(addr, addr.BitLen())}, nil
}

func rangeToPrefixes(from netip.Addr, to netip.Addr) ([]netip.Prefix, error) {
	if from.BitLen() != to.BitLen() {
		return nil, fmt.Errorf("range endpoints must use the same address family")
	}
	if from.Compare(to) > 0 {
		return nil, fmt.Errorf("range start must be <= range end")
	}

	prefixes := make([]netip.Prefix, 0, 8)
	current := from
	for current.Compare(to) <= 0 {
		prefix := largestPrefixWithinRange(current, to)
		prefixes = append(prefixes, prefix)
		last := lastAddrOfPrefix(prefix)
		if last.Compare(to) >= 0 {
			break
		}
		current = nextAddr(last)
	}

	return prefixes, nil
}

func largestPrefixWithinRange(start netip.Addr, end netip.Addr) netip.Prefix {
	bits := start.BitLen()
	best := netip.PrefixFrom(start, bits)

	for prefixBits := bits - 1; prefixBits >= 0; prefixBits-- {
		prefix := netip.PrefixFrom(start, prefixBits).Masked()
		if prefix.Addr().Compare(start) != 0 {
			break
		}
		if lastAddrOfPrefix(prefix).Compare(end) > 0 {
			continue
		}
		best = prefix
	}

	return best
}

func lastAddrOfPrefix(prefix netip.Prefix) netip.Addr {
	addr := prefix.Addr().Unmap()
	bytes := addrBytes(addr)
	bits := prefix.Bits()

	for bit := bits; bit < addr.BitLen(); bit++ {
		byteIndex := bit / 8
		bitIndex := 7 - (bit % 8)
		bytes[byteIndex] |= 1 << bitIndex
	}

	return bytesToAddr(addr.BitLen(), bytes)
}

func nextAddr(addr netip.Addr) netip.Addr {
	bytes := addrBytes(addr.Unmap())
	for idx := len(bytes) - 1; idx >= 0; idx-- {
		bytes[idx]++
		if bytes[idx] != 0 {
			break
		}
	}
	return bytesToAddr(addr.BitLen(), bytes)
}

func addrBytes(addr netip.Addr) []byte {
	if addr.BitLen() == 32 {
		value := addr.As4()
		return append([]byte(nil), value[:]...)
	}
	value := addr.As16()
	return append([]byte(nil), value[:]...)
}

func bytesToAddr(bitLen int, bytes []byte) netip.Addr {
	if bitLen == 32 {
		var value [4]byte
		copy(value[:], bytes)
		return netip.AddrFrom4(value)
	}
	var value [16]byte
	copy(value[:], bytes)
	return netip.AddrFrom16(value)
}

func bitAt(addr netip.Addr, bit int) int {
	bytes := addrBytes(addr.Unmap())
	byteIndex := bit / 8
	bitIndex := 7 - (bit % 8)
	if bytes[byteIndex]&(1<<bitIndex) == 0 {
		return 0
	}
	return 1
}
