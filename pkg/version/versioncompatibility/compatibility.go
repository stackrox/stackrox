package versioncompatibility

import (
	"slices"

	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/version/productstreams"
)

var log = logging.LoggerForModule()

// DefaultSkew is the number of minor version steps for the supported
// version compatibility range (N +/- DefaultSkew).
const DefaultSkew = 3

// Compatibility represents the version compatibility classification between
// two components (e.g., Central and Sensor, or roxctl and Central).
// Values map 1:1 to storage.SensorVersionCompatibility proto enum values.
type Compatibility int

const (
	Unknown            Compatibility = iota
	Matched                          // Same X.Y version.
	CompatibleBehind                 // Remote is older but within range.
	CompatibleAhead                  // Remote is newer but within range.
	IncompatibleBehind               // Remote is too old.
	IncompatibleAhead                // Remote is too new.
)

func (c Compatibility) String() string {
	switch c {
	case Unknown:
		return "UNKNOWN"
	case Matched:
		return "MATCHED"
	case CompatibleBehind:
		return "COMPATIBLE_BEHIND"
	case CompatibleAhead:
		return "COMPATIBLE_AHEAD"
	case IncompatibleBehind:
		return "INCOMPATIBLE_BEHIND"
	case IncompatibleAhead:
		return "INCOMPATIBLE_AHEAD"
	default:
		return "UNKNOWN"
	}
}

// CompatibleVersions returns the compatible version range for self using
// the default skew tolerance (DefaultSkew).
func CompatibleVersions(self productstreams.XYVersion) []productstreams.XYVersion {
	return CompatibleVersionRange(self, DefaultSkew)
}

// ClassifyVersion classifies remote relative to self using the default
// skew tolerance (DefaultSkew).
func ClassifyVersion(self, remote productstreams.XYVersion) Compatibility {
	return Classify(self, remote, DefaultSkew)
}

// CompatibleVersionRange computes the range of X.Y versions compatible with
// self, spanning n minor versions in each direction. It correctly crosses
// major version boundaries using the bump history embedded in the
// majorversions package.
//
// Returns the ordered list of compatible versions from oldest to newest,
// including self. The list contains 2*n+1 elements when all steps succeed,
// or fewer if the backward walk cannot go far enough.
func CompatibleVersionRange(self productstreams.XYVersion, n int) []productstreams.XYVersion {
	if n < 0 {
		n = 0
	}
	backward := make([]productstreams.XYVersion, 0, n)
	cur := self
	for range n {
		prev, err := productstreams.GetPreviousYStream(cur)
		if err != nil {
			log.Errorf("Failed to compute previous Y-stream for %s: %v", cur, err)
			break
		}
		backward = append(backward, prev)
		cur = prev
	}
	slices.Reverse(backward)

	forward := make([]productstreams.XYVersion, 0, n)
	cur = self
	for range n {
		next := productstreams.GetNextYStream(cur)
		forward = append(forward, next)
		cur = next
	}

	result := make([]productstreams.XYVersion, 0, len(backward)+1+len(forward))
	result = append(result, backward...)
	result = append(result, self)
	result = append(result, forward...)
	return result
}

// Classify determines the compatibility of remote relative to self with
// a custom skew tolerance n.
func Classify(self, remote productstreams.XYVersion, n int) Compatibility {
	if n < 0 {
		n = 0
	}
	cmp := self.Compare(remote)
	if cmp == 0 {
		return Matched
	}

	versions := CompatibleVersionRange(self, n)
	if slices.Contains(versions, remote) {
		if cmp > 0 {
			return CompatibleBehind
		}
		return CompatibleAhead
	}

	// Remote is not in the known compatible set. Fall back to naive
	// distance (uses bumps only to cross majors, linear Y within)
	// to handle phantom versions past a bump point that was delayed
	// or cancelled.
	if dist := self.NaiveDistance(remote); dist >= 0 && dist <= n {
		if cmp > 0 {
			return CompatibleBehind
		}
		return CompatibleAhead
	}

	if cmp > 0 {
		return IncompatibleBehind
	}
	return IncompatibleAhead
}
