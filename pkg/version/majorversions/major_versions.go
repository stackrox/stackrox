package majorversions

import (
	_ "embed"
	"fmt"
	"strconv"
	"strings"

	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/utils"
	"golang.org/x/mod/semver"
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

func (v XYVersion) semverString() string {
	return fmt.Sprintf("v%d.%d.0", v.X, v.Y)
}

func parseXYVersion(s string) (XYVersion, error) {
	parts := strings.SplitN(s, ".", 2)
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

func (v XYVersion) Compare(other XYVersion) int {
	return semver.Compare(v.semverString(), other.semverString())
}

var parsedBumps []parsedBump

func init() {
	parsedBumps = mustParseBumpsData(rawData)
}

func mustParseBumpsData(data []byte) []parsedBump {
	bumps, err := parseBumpsData(data)
	utils.CrashOnError(err)
	return bumps
}

func parseBumpsData(data []byte) ([]parsedBump, error) {
	var f bumpsFile
	if err := yaml.Unmarshal(data, &f); err != nil {
		return nil, err
	}
	var seenTo set.IntSet
	var seenFrom set.IntSet
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
		if !seenTo.Add(to.X) {
			return nil, fmt.Errorf("duplicate 'to' major %d in major_version_bumps.yaml", to.X)
		}
		if !seenFrom.Add(from.X) {
			return nil, fmt.Errorf("duplicate 'from' major %d in major_version_bumps.yaml", from.X)
		}
		result = append(result, parsedBump{From: from, To: to})
	}
	return result, nil
}

// GetPreviousYStream returns the previous Y-stream version for a given major.minor.
// If minor > 0, the previous Y-stream is simply major.(minor-1).
// If minor == 0, it looks up the major version bump history to find what came before.
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
