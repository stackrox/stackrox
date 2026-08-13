package versioncompatibility

import (
	"slices"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/pkg/version"
	"github.com/stackrox/rox/pkg/version/productstreams"
)

// AllowedSkew is the number of minor version steps for the supported
// version compatibility range (N +/- AllowedSkew).
const AllowedSkew = 3

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

var (
	once            sync.Once
	mainXY          productstreams.XYVersion
	compatibleRange []productstreams.XYVersion
	initErr         error
)

func get() (productstreams.XYVersion, []productstreams.XYVersion, error) {
	once.Do(func() {
		mainXY, compatibleRange, initErr = computeCompatibleRange()
		utils.Should(initErr)
	})
	return mainXY, compatibleRange, initErr
}

// OverrideForTesting recomputes the singleton using the current main version.
// Call after testutils.SetMainVersion.
func OverrideForTesting(t interface{ Cleanup(func()) }) {
	old := struct {
		mainXY          productstreams.XYVersion
		compatibleRange []productstreams.XYVersion
		initErr         error
	}{mainXY, compatibleRange, initErr}
	mainXY, compatibleRange, initErr = computeCompatibleRange()
	t.Cleanup(func() {
		mainXY = old.mainXY
		compatibleRange = old.compatibleRange
		initErr = old.initErr
	})
}

func computeCompatibleRange() (productstreams.XYVersion, []productstreams.XYVersion, error) {
	xy, err := productstreams.ParseXYFromVersionString(version.GetMainVersion())
	if err != nil {
		return productstreams.XYVersion{}, nil, errors.Wrapf(err, "parsing version %q", version.GetMainVersion())
	}
	versions, err := makeCompatibleVersionRange(xy, AllowedSkew)
	return xy, versions, err
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
	if cmp < 0 {
		if remote.Compare(versions[len(versions)-1]) <= 0 {
			return CompatibleAhead
		}
		return IncompatibleAhead
	}
	return Matched
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
