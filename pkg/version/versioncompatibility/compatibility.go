package versioncompatibility

import (
	"slices"

	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/version"
	"github.com/stackrox/rox/pkg/version/productstreams"
)

var (
	log = logging.LoggerForModule()

	once            sync.Once
	selfXY          productstreams.XYVersion
	compatibleRange []productstreams.XYVersion
	initErr         error
)

func initialize() (productstreams.XYVersion, []productstreams.XYVersion, error) {
	once.Do(func() {
		selfXY, initErr = productstreams.ParseXYFromVersionString(version.GetMainVersion())
		if initErr != nil {
			return
		}
		compatibleRange = CompatibleVersionRange(selfXY, DefaultSkew)
	})
	return selfXY, compatibleRange, initErr
}

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

// CompatibleVersions returns the cached compatible version range for the
// running binary's version, computed once on first call.
func CompatibleVersions() ([]productstreams.XYVersion, error) {
	_, versions, err := initialize()
	return versions, err
}

// ClassifyVersion classifies remote relative to the running binary's
// version, using the cached compatible range.
func ClassifyVersion(remote productstreams.XYVersion) (Compatibility, error) {
	self, versions, err := initialize()
	if err != nil {
		return Unknown, err
	}
	return classify(self, versions, remote), nil
}

// CompatibleVersionRange computes the range of X.Y versions compatible with
// self, spanning n minor versions in each direction. It correctly crosses
// major version boundaries using the bump history.
//
// Returns the ordered list of compatible versions from oldest to newest,
// including self. The list contains 2*n+1 elements when all steps succeed,
// or fewer if the backward walk cannot go far enough.
func CompatibleVersionRange(self productstreams.XYVersion, n int) []productstreams.XYVersion {
	if n < 0 {
		panic("CompatibleVersionRange: n must be non-negative")
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

func classify(self productstreams.XYVersion, versions []productstreams.XYVersion, remote productstreams.XYVersion) Compatibility {
	cmp := self.Compare(remote)
	if cmp == 0 {
		return Matched
	}

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
