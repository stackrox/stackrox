package version

import (
	_ "embed"
	"fmt"

	"gopkg.in/yaml.v3"
)

//go:embed major_bumps.yaml
var majorBumpsYAML []byte

// MajorBump records a major version transition: the last minor release
// before the bump and the first release of the new major.
type MajorBump struct {
	From XY `yaml:"from"`
	To   XY `yaml:"to"`
}

// XY is a major.minor version pair (e.g. 4.11).
type XY struct {
	Major int
	Minor int
}

func (v XY) String() string {
	return fmt.Sprintf("%d.%d", v.Major, v.Minor)
}

func (v XY) less(other XY) bool {
	if v.Major != other.Major {
		return v.Major < other.Major
	}
	return v.Minor < other.Minor
}

func (v *XY) UnmarshalYAML(node *yaml.Node) error {
	var s string
	if err := node.Decode(&s); err != nil {
		return err
	}
	parsed, err := ParseXY(s)
	if err != nil {
		return err
	}
	*v = parsed
	return nil
}

// ParseXY extracts major.minor from a version string.
// Accepts formats like "4.11", "4.11.2", "4.11.0-rc.1", "4.11.x-123-gabcdef1234".
func ParseXY(version string) (XY, error) {
	parsed, err := parseVersion(version)
	if err != nil {
		return XY{}, fmt.Errorf("parsing version %q: %w", version, err)
	}
	return XY{Major: parsed.MarketingMajor, Minor: parsed.EngRelease}, nil
}

type bumpsFile struct {
	Bumps []MajorBump `yaml:"bumps"`
}

// EmbeddedMajorBumps returns the major version bumps compiled into this binary.
func EmbeddedMajorBumps() ([]MajorBump, error) {
	return parseMajorBumps(majorBumpsYAML)
}

func parseMajorBumps(data []byte) ([]MajorBump, error) {
	var f bumpsFile
	if err := yaml.Unmarshal(data, &f); err != nil {
		return nil, fmt.Errorf("parsing major bumps YAML: %w", err)
	}
	return f.Bumps, nil
}

// MergeBumps returns the deduplicated union of two bump lists.
func MergeBumps(a, b []MajorBump) []MajorBump {
	seen := make(map[XY]bool, len(a)+len(b))
	var merged []MajorBump
	for _, bump := range append(a, b...) {
		if !seen[bump.From] {
			seen[bump.From] = true
			merged = append(merged, bump)
		}
	}
	return merged
}

// SkewStatus describes the result of a version skew check.
type SkewStatus int

const (
	SkewOK            SkewStatus = iota // within allowed range
	SkewWarning                         // outside allowed range
	SkewIndeterminate                   // couldn't compare (dev builds, etc.)
)

// SkewResult holds the outcome of a version skew check.
type SkewResult struct {
	Status   SkewStatus
	Distance int
	Message  string
}

// CheckSkew checks whether roxctlVersion is within maxSkew minor releases
// of centralVersion, accounting for major version bumps.
func CheckSkew(roxctlVersion, centralVersion string, maxSkew int, bumps []MajorBump) SkewResult {
	if roxctlVersion == centralVersion {
		return SkewResult{Status: SkewOK, Distance: 0, Message: "versions match"}
	}

	roxXY, err := ParseXY(roxctlVersion)
	if err != nil {
		return SkewResult{
			Status:  SkewIndeterminate,
			Message: fmt.Sprintf("cannot parse roxctl version %q: %v", roxctlVersion, err),
		}
	}

	cenXY, err := ParseXY(centralVersion)
	if err != nil {
		return SkewResult{
			Status:  SkewIndeterminate,
			Message: fmt.Sprintf("cannot parse Central version %q: %v", centralVersion, err),
		}
	}

	if roxXY == cenXY {
		return SkewResult{Status: SkewOK, Distance: 0, Message: "versions match (same X.Y)"}
	}

	dist, err := MinorDistance(roxXY, cenXY, bumps)
	if err != nil {
		return SkewResult{
			Status:  SkewIndeterminate,
			Message: fmt.Sprintf("cannot compute version distance: %v", err),
		}
	}

	if dist <= maxSkew {
		return SkewResult{
			Status:   SkewOK,
			Distance: dist,
			Message:  fmt.Sprintf("roxctl %s and Central %s are %d minor version(s) apart (limit: %d)", roxXY, cenXY, dist, maxSkew),
		}
	}

	direction := "newer"
	if roxXY.less(cenXY) {
		direction = "older"
	}
	return SkewResult{
		Status:   SkewWarning,
		Distance: dist,
		Message: fmt.Sprintf(
			"roxctl %s is %d minor version(s) %s than Central %s (supported range: %d)",
			roxXY, dist, direction, cenXY, maxSkew,
		),
	}
}

// MinorDistance computes the absolute distance in minor releases between
// two versions, correctly walking through major version bumps.
//
// For example, with bump 4.11→5.0:
//
//	MinorDistance(4.10, 5.1) = 3   (4.10 → 4.11 → 5.0 → 5.1)
func MinorDistance(a, b XY, bumps []MajorBump) (int, error) {
	if a == b {
		return 0, nil
	}
	lo, hi := a, b
	if b.less(a) {
		lo, hi = b, a
	}

	bumpFrom := make(map[int]MajorBump, len(bumps))
	for _, bump := range bumps {
		bumpFrom[bump.From.Major] = bump
	}

	dist := 0
	cur := lo
	for cur != hi {
		if cur.Major == hi.Major {
			dist += hi.Minor - cur.Minor
			cur = hi
			continue
		}

		bump, ok := bumpFrom[cur.Major]
		if !ok {
			return 0, fmt.Errorf("cannot traverse from %s to %s: no known major bump from major version %d", lo, hi, cur.Major)
		}
		dist += bump.From.Minor - cur.Minor + 1
		cur = bump.To
	}
	return dist, nil
}
