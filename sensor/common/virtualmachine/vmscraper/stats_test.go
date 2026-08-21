package vmscraper

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestVMScraper_Stats(t *testing.T) {
	tests := map[string]struct {
		setup        func(s *VMScraper)
		wantTracked  int
		wantScanned  int
		wantVersions map[string]int
	}{
		"should return zeros when no VMs are tracked": {
			setup:        func(_ *VMScraper) {},
			wantTracked:  0,
			wantScanned:  0,
			wantVersions: map[string]int{},
		},
		"should count tracked and scanned VMs correctly": {
			setup: func(s *VMScraper) {
				now := s.now()
				s.vmState["ns1/vm-a"] = &vmState{
					lastForwardedAt:  now,
					lastAgentVersion: "v1.0.0",
				}
				s.vmState["ns1/vm-b"] = &vmState{
					lastAgentVersion: "v1.0.0",
				}
				s.vmState["ns2/vm-c"] = &vmState{
					lastForwardedAt:  now,
					lastAgentVersion: "v2.0.0",
				}
			},
			wantTracked: 3,
			wantScanned: 2,
			wantVersions: map[string]int{
				"v1.0.0": 1,
				"v2.0.0": 1,
			},
		},
		"should omit unscanned VMs from the version histogram": {
			setup: func(s *VMScraper) {
				s.vmState["ns1/vm-a"] = &vmState{}
				s.vmState["ns1/vm-b"] = &vmState{
					lastAgentVersion: "v1.0.0",
					lastForwardedAt:  s.now(),
				}
			},
			wantTracked: 2,
			wantScanned: 1,
			wantVersions: map[string]int{
				"v1.0.0": 1,
			},
		},
		"should bucket empty agent version of a scanned VM as unknown": {
			setup: func(s *VMScraper) {
				s.vmState["ns1/vm-a"] = &vmState{
					lastForwardedAt: s.now(),
				}
				s.vmState["ns1/vm-b"] = &vmState{
					lastAgentVersion: "v1.0.0",
					lastForwardedAt:  s.now(),
				}
			},
			wantTracked: 2,
			wantScanned: 2,
			wantVersions: map[string]int{
				unknownAgentVersion: 1,
				"v1.0.0":            1,
			},
		},
		"should cap version map to top N and fold remainder into other": {
			setup: func(s *VMScraper) {
				for i := range maxVersionBuckets + 5 {
					ver := fmt.Sprintf("v0.0.%d", i)
					s.vmState[fmt.Sprintf("ns/vm-%d", i)] = &vmState{
						lastAgentVersion: ver,
						lastForwardedAt:  s.now(),
					}
				}
			},
			wantTracked: maxVersionBuckets + 5,
			wantScanned: maxVersionBuckets + 5,
			wantVersions: func() map[string]int {
				m := make(map[string]int, maxVersionBuckets+1)
				for i := range maxVersionBuckets + 5 {
					m[fmt.Sprintf("v0.0.%d", i)] = 1
				}
				// After capping, we expect maxVersionBuckets entries + 1 "other" entry.
				// Since all counts are 1, any maxVersionBuckets are kept; 5 go to "other".
				// The exact set kept is deterministic only if we sort; the test checks
				// sum(values) == tracked and len(m) <= maxVersionBuckets+1.
				return nil // We'll assert structurally below instead
			}(),
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			s, _ := newTestScraper(&mockStore{}, &mockSender{}, &mockDialer{}, &mockProtocolClient{})
			tc.setup(s)
			stats := s.Stats()

			assert.Equal(t, tc.wantTracked, stats.TrackedVMs)
			assert.Equal(t, tc.wantScanned, stats.VMsScanned)

			if tc.wantVersions != nil {
				assert.Equal(t, tc.wantVersions, stats.VersionCounts)
			} else {
				// Structural check for the top-N cap test case
				assert.LessOrEqual(t, len(stats.VersionCounts), maxVersionBuckets+1,
					"version map should not exceed maxVersionBuckets + 1 (other)")
				total := 0
				for _, c := range stats.VersionCounts {
					total += c
				}
				assert.Equal(t, tc.wantScanned, total,
					"sum of version counts should equal scanned VMs")
			}
		})
	}
}
