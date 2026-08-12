package versioncompatibility

import (
	"slices"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/pkg/version"
	"github.com/stackrox/rox/pkg/version/productstreams"
)

var (
	once            sync.Once
	selfXY          productstreams.XYVersion
	compatibleRange []productstreams.XYVersion
	initErr         error
)

func get() (productstreams.XYVersion, []productstreams.XYVersion, error) {
	once.Do(func() {
		utils.Should(computeCompatibleRange())
	})
	return selfXY, compatibleRange, initErr
}

// OverrideForTesting recomputes the singleton using the current main version.
// Call after testutils.SetMainVersion.
func OverrideForTesting(t interface{ Cleanup(func()) }) {
	old := struct {
		selfXY          productstreams.XYVersion
		compatibleRange []productstreams.XYVersion
		initErr         error
	}{selfXY, compatibleRange, initErr}
	initErr = computeCompatibleRange()
	t.Cleanup(func() {
		selfXY = old.selfXY
		compatibleRange = old.compatibleRange
		initErr = old.initErr
	})
}

func computeCompatibleRange() error {
	selfXY, initErr = productstreams.ParseXYFromVersionString(version.GetMainVersion())
	if initErr != nil {
		return errors.Wrapf(initErr, "failed to parse version %q", version.GetMainVersion())
	}
	compatibleRange, initErr = makeCompatibleVersionRange(selfXY, DefaultSkew)
	return initErr
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
	_, versions, err := get()
	return versions, err
}

// ClassifyVersion classifies remote relative to the running binary's
// version, using the cached compatible range.
func ClassifyVersion(remote productstreams.XYVersion) (Compatibility, error) {
	self, versions, err := get()
	if err != nil {
		return Unknown, err
	}
	return classify(self, versions, remote), nil
}

// makeCompatibleVersionRange computes the range of X.Y versions compatible
// with self, spanning n minor versions in each direction. It correctly
// crosses major version boundaries using the bump history.
//
// Returns the ordered list of compatible versions from oldest to newest,
// including self. The list contains 2*n+1 elements.
func makeCompatibleVersionRange(self productstreams.XYVersion, n int) ([]productstreams.XYVersion, error) {
	if n < 0 {
		panic("makeCompatibleVersionRange: n must be non-negative")
	}

	backward := make([]productstreams.XYVersion, 0, n)
	cur := self
	for range n {
		prev, err := productstreams.GetPreviousYStream(cur)
		if err != nil {
			return nil, err
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
	return result, nil
}

func classify(self productstreams.XYVersion, versions []productstreams.XYVersion, remote productstreams.XYVersion) Compatibility {
	cmp := self.Compare(remote)
	if cmp == 0 {
		return Matched
	}
	// Remote is compatible if it falls within the bounds of the compatible
	// range. This naturally handles phantom versions (e.g. 4.12 between
	// 4.11 and 5.0) without explicit gap-checking.
	if cmp > 0 {
		if remote.Compare(versions[0]) >= 0 {
			return CompatibleBehind
		}
		return IncompatibleBehind
	}
	if remote.Compare(versions[len(versions)-1]) <= 0 {
		return CompatibleAhead
	}
	return IncompatibleAhead
}
