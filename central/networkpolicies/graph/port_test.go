package graph

import (
	"testing"

	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stretchr/testify/assert"
)

func TestNormalizeInPlace(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		input    portDescs
		expected portDescs
	}{
		{
			name: "concrete ports w/ duplicates",
			input: portDescs{
				{
					l4proto: storage.Protocol_TCP_PROTOCOL,
					port:    80,
				},
				{
					l4proto: storage.Protocol_SCTP_PROTOCOL,
					port:    1024,
				},
				{
					l4proto: storage.Protocol_UDP_PROTOCOL,
					port:    53,
				},
				{
					l4proto: storage.Protocol_TCP_PROTOCOL,
					port:    80,
				},
				{
					l4proto: storage.Protocol_TCP_PROTOCOL,
					port:    443,
				},
			},
			expected: portDescs{
				{
					l4proto: storage.Protocol_TCP_PROTOCOL,
					port:    80,
				},
				{
					l4proto: storage.Protocol_TCP_PROTOCOL,
					port:    443,
				},
				{
					l4proto: storage.Protocol_UDP_PROTOCOL,
					port:    53,
				},
				{
					l4proto: storage.Protocol_SCTP_PROTOCOL,
					port:    1024,
				},
			},
		},
		{
			name: "some ports with all TCP",
			input: portDescs{
				{
					l4proto: storage.Protocol_TCP_PROTOCOL,
					port:    80,
				},
				{
					l4proto: storage.Protocol_SCTP_PROTOCOL,
					port:    1024,
				},
				{
					l4proto: storage.Protocol_UDP_PROTOCOL,
					port:    53,
				},
				{
					l4proto: storage.Protocol_TCP_PROTOCOL,
					port:    80,
				},
				{
					l4proto: storage.Protocol_TCP_PROTOCOL,
					port:    443,
				},
				{
					l4proto: storage.Protocol_TCP_PROTOCOL,
				},
			},
			expected: portDescs{
				{
					l4proto: storage.Protocol_TCP_PROTOCOL,
				},
				{
					l4proto: storage.Protocol_UDP_PROTOCOL,
					port:    53,
				},
				{
					l4proto: storage.Protocol_SCTP_PROTOCOL,
					port:    1024,
				},
			},
		},
		{
			name: "some ports with all ports",
			input: portDescs{
				{
					l4proto: storage.Protocol_TCP_PROTOCOL,
					port:    80,
				},
				{
					l4proto: storage.Protocol_SCTP_PROTOCOL,
					port:    1024,
				},
				{
					l4proto: storage.Protocol_UDP_PROTOCOL,
					port:    53,
				},
				{
					l4proto: storage.Protocol_TCP_PROTOCOL,
					port:    80,
				},
				{
					l4proto: storage.Protocol_TCP_PROTOCOL,
					port:    443,
				},
				{
					l4proto: storage.Protocol_TCP_PROTOCOL,
				},
				{},
			},
			expected: portDescs{
				{},
			},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			pds := c.input.Clone()
			pds.normalizeInPlace()
			assert.Equal(t, c.expected, pds)
		})
	}
}

func TestIntersectNormalized(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name           string
		inputA, inputB portDescs
		expected       portDescs
	}{
		{
			name:     "any/empty",
			inputA:   portDescs{{}},
			inputB:   nil,
			expected: nil,
		},
		{
			name:     "any/any",
			inputA:   portDescs{{}},
			inputB:   portDescs{{}},
			expected: portDescs{{}},
		},
		{
			name:     "any/some with wildcard",
			inputA:   portDescs{{}},
			inputB:   portDescs{{l4proto: storage.Protocol_TCP_PROTOCOL, port: 443}, {l4proto: storage.Protocol_UDP_PROTOCOL}},
			expected: portDescs{{l4proto: storage.Protocol_TCP_PROTOCOL, port: 443}, {l4proto: storage.Protocol_UDP_PROTOCOL}},
		},
		{
			name:     "any/some without wildcard",
			inputA:   portDescs{{}},
			inputB:   portDescs{{l4proto: storage.Protocol_TCP_PROTOCOL, port: 443}, {l4proto: storage.Protocol_UDP_PROTOCOL, port: 53}},
			expected: portDescs{{l4proto: storage.Protocol_TCP_PROTOCOL, port: 443}, {l4proto: storage.Protocol_UDP_PROTOCOL, port: 53}},
		},
		{
			name:     "some with wildcard/some without wildcard",
			inputA:   portDescs{{l4proto: storage.Protocol_TCP_PROTOCOL, port: 443}, {l4proto: storage.Protocol_TCP_PROTOCOL, port: 8080}, {l4proto: storage.Protocol_UDP_PROTOCOL}},
			inputB:   portDescs{{l4proto: storage.Protocol_TCP_PROTOCOL, port: 443}, {l4proto: storage.Protocol_UDP_PROTOCOL, port: 53}},
			expected: portDescs{{l4proto: storage.Protocol_TCP_PROTOCOL, port: 443}, {l4proto: storage.Protocol_UDP_PROTOCOL, port: 53}},
		},
		{
			name:     "some with wildcard/some with wildcard",
			inputA:   portDescs{{l4proto: storage.Protocol_TCP_PROTOCOL, port: 443}, {l4proto: storage.Protocol_TCP_PROTOCOL, port: 8080}, {l4proto: storage.Protocol_UDP_PROTOCOL}},
			inputB:   portDescs{{l4proto: storage.Protocol_TCP_PROTOCOL}, {l4proto: storage.Protocol_UDP_PROTOCOL, port: 53}},
			expected: portDescs{{l4proto: storage.Protocol_TCP_PROTOCOL, port: 443}, {l4proto: storage.Protocol_TCP_PROTOCOL, port: 8080}, {l4proto: storage.Protocol_UDP_PROTOCOL, port: 53}},
		},
		{
			name:     "some without wildcard/some without wildcard",
			inputA:   portDescs{{l4proto: storage.Protocol_TCP_PROTOCOL, port: 443}, {l4proto: storage.Protocol_TCP_PROTOCOL, port: 8080}, {l4proto: storage.Protocol_UDP_PROTOCOL, port: 1024}, {l4proto: storage.Protocol_UDP_PROTOCOL, port: 27015}},
			inputB:   portDescs{{l4proto: storage.Protocol_TCP_PROTOCOL, port: 80}, {l4proto: storage.Protocol_TCP_PROTOCOL, port: 8080}, {l4proto: storage.Protocol_UDP_PROTOCOL, port: 53}, {l4proto: storage.Protocol_UDP_PROTOCOL, port: 1024}},
			expected: portDescs{{l4proto: storage.Protocol_TCP_PROTOCOL, port: 8080}, {l4proto: storage.Protocol_UDP_PROTOCOL, port: 1024}},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			a := c.inputA.Clone()
			b := c.inputB.Clone()
			a.normalizeInPlace()
			b.normalizeInPlace()

			intersection := intersectNormalized(a, b)
			assert.Equal(t, c.expected, intersection, "incorrect intersectNormalized(a, b) result")

			revIntersection := intersectNormalized(b, a)
			assert.Equal(t, c.expected, revIntersection, "incorrect intersectNormalized(b, a) result")
		})
	}
}
