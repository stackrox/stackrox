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
// If v is at or past a bump's From field (v.Y >= bump.From.Y within the same
// major), the next version is the bump's To (e.g. {4,11} -> {5,0}). This
// means phantom versions past the bump point (e.g. 4.12 when the bump is
// 4.11→5.0) also jump to the next major, since they shouldn't exist and
// the bump is the ceiling for that major.
// Otherwise, the next version is simply {v.X, v.Y+1}.
func GetNextYStream(v XYVersion) XYVersion {
	for _, b := range parsedBumps {
		if v.X == b.From.X && v.Y >= b.From.Y {
			return b.To
		}
	}
	return XYVersion{X: v.X, Y: v.Y + 1}
}

// ParseXYFromVersionString extracts the major.minor (X.Y) components from
// a full version string. It accepts formats such as "4.11", "4.11.2",
// "4.11.0-rc.1", and "4.11.x-123-gabcdef1234".
func ParseXYFromVersionString(version string) (XYVersion, error) {
	before, _, _ := strings.Cut(version, "-")
	parts := strings.Split(before, ".")
	if len(parts) < 2 || len(parts) > 3 {
		return XYVersion{}, fmt.Errorf("expected major.minor[.patch] format, got %q", version)
	}
	major, err := strconv.Atoi(parts[0])
	if err != nil {
		return XYVersion{}, fmt.Errorf("invalid major %q in version %q: %w", parts[0], version, err)
	}
	minor, err := strconv.Atoi(parts[1])
	if err != nil {
		return XYVersion{}, fmt.Errorf("invalid minor %q in version %q: %w", parts[1], version, err)
	}
	return XYVersion{X: major, Y: minor}, nil
}

// GetPreviousYStream returns the previous Y-stream version for a given major.minor.
// If v is a phantom version (past a bump point in the same major), it snaps
// back to the bump point directly. For example, with bump 4.11→5.0,
// GetPreviousYStream(4.14) returns 4.11.
// If minor == 0, it crosses back via the bump history.
// Otherwise, the previous Y-stream is simply major.(minor-1).
func GetPreviousYStream(v XYVersion) (XYVersion, error) {
	for _, b := range parsedBumps {
		// Phantom version: snap back to the bump point (e.g. 4.14 → 4.11).
		if b.From.X == v.X && v.Y > b.From.Y {
			return b.From, nil
		}
		// Major boundary: cross back via bump history (e.g. 5.0 → 4.11).
		if b.To.X == v.X && v.Y == 0 {
			return b.From, nil
		}
	}
	// Normal decrement (e.g. 4.10 → 4.9).
	if v.Y > 0 {
		return XYVersion{X: v.X, Y: v.Y - 1}, nil
	}
	return XYVersion{}, fmt.Errorf("don't know the previous Y-Stream for %s", v)
}
