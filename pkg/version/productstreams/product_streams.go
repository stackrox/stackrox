package productstreams

import (
	"cmp"
	_ "embed"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/utils"
	"gopkg.in/yaml.v3"
)

//go:embed major_version_bumps.yaml
var rawData []byte

// XYVersion represents a major.minor version number.
type XYVersion struct {
	X int
	Y int
}

func (v XYVersion) String() string {
	return fmt.Sprintf("%d.%d", v.X, v.Y)
}

func parseXYVersion(s string) (XYVersion, error) {
	parts := strings.SplitN(s, ".", 3)
	if len(parts) != 2 {
		return XYVersion{}, fmt.Errorf("expected major.minor format, got %q", s)
	}
	major, err := strconv.Atoi(parts[0])
	if err != nil {
		return XYVersion{}, fmt.Errorf("invalid major %q: %w", parts[0], err)
	}
	minor, err := strconv.Atoi(parts[1])
	if err != nil {
		return XYVersion{}, fmt.Errorf("invalid minor %q: %w", parts[1], err)
	}
	return XYVersion{X: major, Y: minor}, nil
}

type bump struct {
	From string `yaml:"from"`
	To   string `yaml:"to"`
}

type bumpsFile struct {
	Bumps []bump `yaml:"bumps"`
}

type parsedBump struct {
	From XYVersion
	To   XYVersion
}

// Compare returns -1 if v < other, 0 if equal, 1 if v > other.
func (v XYVersion) Compare(other XYVersion) int {
	if c := cmp.Compare(v.X, other.X); c != 0 {
		return c
	}
	return cmp.Compare(v.Y, other.Y)
}

var parsedBumps []parsedBump

func init() {
	parsedBumps = mustParseBumpsData(rawData)
}

func mustParseBumpsData(data []byte) []parsedBump {
	bumps, err := parseBumpsData(data)
	utils.CrashOnError(errors.Wrap(err, "invalid content of major_version_bumps.yaml, please fix the file and rebuild"))
	return bumps
}

func parseBumpsData(data []byte) ([]parsedBump, error) {
	var f bumpsFile
	if err := yaml.Unmarshal(data, &f); err != nil {
		return nil, err
	}
	var result []parsedBump
	for _, b := range f.Bumps {
		from, err := parseXYVersion(b.From)
		if err != nil {
			return nil, fmt.Errorf("invalid 'from' value %q: %w", b.From, err)
		}
		to, err := parseXYVersion(b.To)
		if err != nil {
			return nil, fmt.Errorf("invalid 'to' value %q: %w", b.To, err)
		}
		if to.Y != 0 {
			return nil, fmt.Errorf("'to' value %q must have minor version 0", b.To)
		}
		if from.Compare(to) >= 0 {
			return nil, fmt.Errorf("'from' %s must be less than 'to' %s", from, to)
		}
		result = append(result, parsedBump{From: from, To: to})
	}
	slices.SortFunc(result, func(a, b parsedBump) int {
		return a.From.Compare(b.From)
	})
	for i := 1; i < len(result); i++ {
		if result[i].From.Compare(result[i-1].To) < 0 {
			return nil, fmt.Errorf("overlapping ranges: %s->%s and %s->%s",
				result[i-1].From, result[i-1].To, result[i].From, result[i].To)
		}
	}
	return result, nil
}

// NaiveDistance returns the distance between v and other using bumps only
// to cross major boundaries and linear Y arithmetic within the target
// major. This handles phantom versions (past a bump point that was
// delayed or cancelled) by ignoring bumps within the target major.
// Returns -1 if the target major cannot be reached.
func (v XYVersion) NaiveDistance(other XYVersion) int {
	lo, hi := v, other
	if other.Compare(v) < 0 {
		lo, hi = other, v
	}

	dist := 0
	cur := lo

	for cur.X < hi.X {
		bump, ok := findBumpFrom(cur.X)
		if !ok || bump.From.Y < cur.Y {
			return -1
		}
		dist += bump.From.Y - cur.Y + 1
		cur = bump.To
	}

	d := hi.Y - cur.Y
	if d < 0 {
		d = -d
	}
	return dist + d
}

func findBumpFrom(major int) (parsedBump, bool) {
	for _, b := range parsedBumps {
		if b.From.X == major {
			return b, true
		}
	}
	return parsedBump{}, false
}

// OverrideBumpsForTesting replaces the parsed bump data with the given YAML
// for the duration of the test, restoring the original data on cleanup.
func OverrideBumpsForTesting(t interface {
	Cleanup(func())
}, data []byte) {
	old := parsedBumps
	parsedBumps = mustParseBumpsData(data)
	t.Cleanup(func() { parsedBumps = old })
}

// GetNextYStream returns the next Y-stream version for a given major.minor.
// If v matches a bump's From field, the next version is the bump's To
// (e.g., {4,11} -> {5,0}). Otherwise, it is simply {v.X, v.Y+1}.
func GetNextYStream(v XYVersion) XYVersion {
	for _, b := range parsedBumps {
		if b.From == v {
			return b.To
		}
	}
	return XYVersion{X: v.X, Y: v.Y + 1}
}

// ParseXYFromVersionString extracts the major.minor (X.Y) components from
// a full version string. It accepts formats such as "4.11", "4.11.2",
// "4.11.0-rc.1", "4.11.x-123-gabcdef1234", and the legacy 4-component
// format "3.0.61.1" (where parts[1] is the marketing minor, skipped).
func ParseXYFromVersionString(version string) (XYVersion, error) {
	before, _, _ := strings.Cut(version, "-")
	parts := strings.Split(before, ".")
	switch {
	case len(parts) == 4:
		major, err := strconv.Atoi(parts[0])
		if err != nil {
			return XYVersion{}, fmt.Errorf("invalid major %q in version %q: %w", parts[0], version, err)
		}
		minor, err := strconv.Atoi(parts[2])
		if err != nil {
			return XYVersion{}, fmt.Errorf("invalid minor %q in version %q: %w", parts[2], version, err)
		}
		return XYVersion{X: major, Y: minor}, nil
	case len(parts) >= 2:
		major, err := strconv.Atoi(parts[0])
		if err != nil {
			return XYVersion{}, fmt.Errorf("invalid major %q in version %q: %w", parts[0], version, err)
		}
		minor, err := strconv.Atoi(parts[1])
		if err != nil {
			return XYVersion{}, fmt.Errorf("invalid minor %q in version %q: %w", parts[1], version, err)
		}
		return XYVersion{X: major, Y: minor}, nil
	default:
		return XYVersion{}, fmt.Errorf("expected at least major.minor format, got %q", version)
	}
}

// GetPreviousYStream returns the previous Y-stream version for a given major.minor.
// If minor > 0, the previous Y-stream is simply major.(minor-1).
// If minor == 0, it looks up the major version bump history from major_version_bumps.yaml.
// By definition, major version bumps always target X.0 (never X.N with N>0).
func GetPreviousYStream(v XYVersion) (XYVersion, error) {
	if v.Y > 0 {
		return XYVersion{X: v.X, Y: v.Y - 1}, nil
	}
	for _, b := range parsedBumps {
		if b.To.X == v.X {
			return b.From, nil
		}
	}
	return XYVersion{}, fmt.Errorf("don't know the previous Y-Stream for %s", v)
}
