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
