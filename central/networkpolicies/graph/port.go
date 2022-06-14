package graph

import (
	"sort"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
)

var (
	protoToL4Proto = map[storage.Protocol]storage.L4Protocol{
		storage.Protocol_UNSET_PROTOCOL: storage.L4Protocol_L4_PROTOCOL_ANY,
		storage.Protocol_TCP_PROTOCOL:   storage.L4Protocol_L4_PROTOCOL_TCP,
		storage.Protocol_UDP_PROTOCOL:   storage.L4Protocol_L4_PROTOCOL_UDP,
		storage.Protocol_SCTP_PROTOCOL:  storage.L4Protocol_L4_PROTOCOL_SCTP,
	}
)

// portDesc describes a port with corresponding L4 protocol.
// The zero value (an l4proto of UNSET_PROTOCOL in combination with a port value of 0)
// represents "all ports, all protocols", while a set l4proto with a port value of 0 means
// "all tcp/udp/... ports".
type portDesc struct {
	l4proto storage.Protocol
	port    int32
}

func (p *portDesc) isValid() bool {
	if p.l4proto == storage.Protocol_UNSET_PROTOCOL && p.port != 0 {
		return false
	}
	return p.port >= 0 && p.port <= 65535
}

func (p *portDesc) matches(other *portDesc) bool {
	if p.isAllPorts() {
		return true
	}
	if p.l4proto != other.l4proto {
		return false
	}
	if p.port == 0 {
		return true
	}
	return p.port == other.port
}

func (p *portDesc) isLessThan(other *portDesc) bool {
	if p.l4proto != other.l4proto {
		return p.l4proto < other.l4proto
	}
	return p.port < other.port
}

func (p *portDesc) isAllPorts() bool {
	return *p == portDesc{}
}

type portDescs []portDesc

func (p portDescs) Less(i, j int) bool {
	if p[i].l4proto != p[j].l4proto {
		return p[i].l4proto < p[j].l4proto
	}
	return p[i].port < p[j].port
}

func (p portDescs) Swap(i, j int) {
	p[i], p[j] = p[j], p[i]
}

func (p portDescs) Len() int {
	return len(p)
}

func (p portDescs) Clone() portDescs {
	if p == nil {
		return nil
	}

	clone := make(portDescs, len(p))
	copy(clone, p)
	return clone
}

// normalizeInPlaces orders the given port descriptors in a deterministic order, eliminates
// duplicates, and removes elements that are subsumed by more permissive port specification
// (e.g., if all ports are allowed, then the result is simply an "all ports" entry; if all TCP
// ports are allowed, then the result no longer contains any non-zero TCP port entries).
func (p *portDescs) normalizeInPlace() {
	sort.Sort(*p)
	filtered := (*p)[:0]

	var prev *portDesc
	var skipProto *storage.Protocol
	for i, curr := range *p {
		if !curr.isValid() {
			continue
		}
		if prev != nil && *prev == curr {
			continue
		}

		if skipProto != nil && *skipProto == curr.l4proto {
			continue
		}
		skipProto = nil

		filtered = append(filtered, curr)
		if curr.l4proto == storage.Protocol_UNSET_PROTOCOL {
			// in conjunction with isValid(), this can only mean "all ports, all protocols"
			break
		}
		prev = &(*p)[i]
		if prev.port == 0 {
			skipProto = &prev.l4proto
		}
	}

	*p = filtered
}

func (p portDescs) ToProto() []*v1.NetworkEdgeProperties {
	if len(p) == 0 {
		return nil
	}

	props := make([]*v1.NetworkEdgeProperties, 0, len(p))
	for _, port := range p {
		props = append(props, &v1.NetworkEdgeProperties{
			Protocol: protoToL4Proto[port.l4proto],
			Port:     uint32(port.port),
		})
	}

	return props
}

func intersectNormalized(a, b portDescs) portDescs {
	if len(a) == 0 || len(b) == 0 {
		return nil
	}

	if len(a) == 1 && a[0].isAllPorts() {
		return b
	}
	if len(b) == 1 && b[0].isAllPorts() {
		return a
	}

	var result portDescs

	i, j := 0, 0
	for i < len(a) && j < len(b) {
		if a[i] == b[j] {
			result = append(result, a[i])
			i++
			j++
		} else if a[i].matches(&b[j]) {
			result = append(result, b[j])
			j++
		} else if b[j].matches(&a[i]) {
			result = append(result, a[i])
			i++
		} else if a[i].isLessThan(&b[j]) {
			i++
		} else {
			j++
		}
	}

	return result
}
