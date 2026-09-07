package vmscraper

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

const (
	testAgentVersion470  = "4.7.0-12-gabcdef0123" // typical ldflags / git describe build
	testAgentVersion480  = "4.8.0-3-gfedcba9876"
	testAgentDevelopment = "development" // local default before ldflags injection
)

// testGitDescribeBuild returns git-describe-shaped version strings; commitNum is
// zero-padded so lexical order matches numeric order in the cap tie-break test.
func testGitDescribeBuild(commitNum int) string {
	return fmt.Sprintf("4.7.0-%02d-g01234567%02d", commitNum, commitNum)
}

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
					lastAgentVersion: testAgentVersion470,
				}
				s.vmState["ns1/vm-b"] = &vmState{
					lastAgentVersion: testAgentVersion470,
				}
				s.vmState["ns2/vm-c"] = &vmState{
					lastForwardedAt:  now,
					lastAgentVersion: testAgentVersion480,
				}
			},
			wantTracked: 3,
			wantScanned: 2,
			wantVersions: map[string]int{
				testAgentVersion470: 1,
				testAgentVersion480: 1,
			},
		},
		"should bucket empty agent version of a scanned VM as unknown": {
			setup: func(s *VMScraper) {
				s.vmState["ns1/vm-a"] = &vmState{
					lastForwardedAt: s.now(),
				}
				s.vmState["ns1/vm-b"] = &vmState{
					lastAgentVersion: testAgentDevelopment,
					lastForwardedAt:  s.now(),
				}
			},
			wantTracked: 2,
			wantScanned: 2,
			wantVersions: map[string]int{
				unknownAgentVersion:  1,
				testAgentDevelopment: 1,
			},
		},
		"should cap version map to top N and fold remainder into other": {
			setup: func(s *VMScraper) {
				for i := range maxVersionBuckets + 5 {
					s.vmState[fmt.Sprintf("ns/vm-%d", i)] = &vmState{
						lastAgentVersion: testGitDescribeBuild(i),
						lastForwardedAt:  s.now(),
					}
				}
			},
			wantTracked: maxVersionBuckets + 5,
			wantScanned: maxVersionBuckets + 5,
			wantVersions: func() map[string]int {
				m := make(map[string]int, maxVersionBuckets+1)
				for i := range maxVersionBuckets {
					m[testGitDescribeBuild(i)] = 1
				}
				m[otherAgentVersion] = 5
				return m
			}(),
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			s, _ := newTestScraper(t, &mockStore{}, &mockDialer{}, &mockProtocolClient{})
			tc.setup(s)
			stats := s.Stats()

			assert.Equal(t, tc.wantTracked, stats.TrackedVMs)
			assert.Equal(t, tc.wantScanned, stats.VMsScanned)

			assert.Equal(t, tc.wantVersions, stats.VersionCounts)
		})
	}
}
