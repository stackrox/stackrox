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

	// If self is a phantom version (past a bump point), snap the backward
	// walk to the bump point. The bump point is included in the backward
	// list and phantom intermediates between it and self are skipped.
	// For example, with bump 4.11→5.0 and self=4.14, n=3:
	//   backward starts at 4.11: [4.11, 4.10, 4.9]
	//   result: [4.9, 4.10, 4.11, 4.14, 5.0, 5.1, 5.2]
	backwardStart := self
	backward := make([]productstreams.XYVersion, 0, n)
	if n > 0 {
		if bp, ok := productstreams.GetBumpPointFor(self); ok {
			backwardStart = bp
			backward = append(backward, bp)
		}
	}

	cur := backwardStart
	for len(backward) < n {
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

	// If remote fits between two consecutive versions in the compatible
	// range (e.g. phantom 4.12 between 4.11 and 5.0), it is compatible.
	for i := 0; i < len(versions)-1; i++ {
		if versions[i].Compare(remote) < 0 && remote.Compare(versions[i+1]) < 0 {
			if cmp > 0 {
				return CompatibleBehind
			}
			return CompatibleAhead
		}
	}

	if cmp > 0 {
		return IncompatibleBehind
	}
	return IncompatibleAhead
}
