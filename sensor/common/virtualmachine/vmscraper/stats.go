package vmscraper

import (
	"slices"
	"strings"
)

const (
	maxVersionBuckets   = 20
	unknownAgentVersion = "unknown"
	otherAgentVersion   = "other"
)

// Stats is a point-in-time snapshot of the scraper's fleet state.
type Stats struct {
	TrackedVMs    int
	VMsScanned    int
	VersionCounts map[string]int // scanned VMs only; empty AgentVersion is "unknown"
}

// Stats returns a point-in-time snapshot of fleet statistics.
// VersionCounts omits never-scraped VMs: those have no agent version to report.
func (s *VMScraper) Stats() Stats {
	s.mu.Lock()
	defer s.mu.Unlock()
	return computeStats(s.vmState)
}

func computeStats(vmState map[string]*vmState) Stats {
	st := Stats{
		TrackedVMs:    len(vmState),
		VersionCounts: make(map[string]int, min(len(vmState), maxVersionBuckets+1)),
	}

	for _, vs := range vmState {
		if vs.lastForwardedAt.IsZero() {
			continue
		}
		st.VMsScanned++
		ver := vs.lastAgentVersion
		if ver == "" {
			ver = unknownAgentVersion
		}
		st.VersionCounts[ver]++
	}

	capVersionCounts(st.VersionCounts)
	return st
}

// capVersionCounts keeps only the top-N versions by count, folding the rest
// into an "other" bucket so the map size is bounded regardless of fleet size.
func capVersionCounts(counts map[string]int) {
	if len(counts) <= maxVersionBuckets {
		return
	}

	type entry struct {
		ver   string
		count int
	}
	entries := make([]entry, 0, len(counts))
	for v, c := range counts {
		entries = append(entries, entry{v, c})
	}
	slices.SortFunc(entries, func(a, b entry) int {
		if a.count != b.count {
			return b.count - a.count // descending by count
		}
		return strings.Compare(a.ver, b.ver) // stable tiebreak
	})

	otherTotal := 0
	for _, e := range entries[maxVersionBuckets:] {
		otherTotal += e.count
		delete(counts, e.ver)
	}
	if otherTotal > 0 {
		counts[otherAgentVersion] += otherTotal
	}
}
