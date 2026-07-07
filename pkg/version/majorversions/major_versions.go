package majorversions

import (
	_ "embed"
	"fmt"
	"strconv"
	"strings"

	"github.com/stackrox/rox/pkg/utils"
	"gopkg.in/yaml.v3"
)

//go:embed major_version_bumps.yaml
var rawData []byte

type bump struct {
	From string `yaml:"from"`
	To   string `yaml:"to"`
}

type bumpsFile struct {
	Bumps []bump `yaml:"bumps"`
}

type parsedBump struct {
	FromMajor int
	FromMinor int
	ToMajor   int
	ToMinor   int
}

var parsedBumps []parsedBump

func init() {
	var f bumpsFile
	utils.CrashOnError(yaml.Unmarshal(rawData, &f))
	for _, b := range f.Bumps {
		pb, err := parseBump(b)
		utils.CrashOnError(err)
		parsedBumps = append(parsedBumps, pb)
	}
}

func parseBump(b bump) (parsedBump, error) {
	fromMajor, fromMinor, err := parseMajorMinor(b.From)
	if err != nil {
		return parsedBump{}, fmt.Errorf("invalid 'from' value %q: %w", b.From, err)
	}
	toMajor, toMinor, err := parseMajorMinor(b.To)
	if err != nil {
		return parsedBump{}, fmt.Errorf("invalid 'to' value %q: %w", b.To, err)
	}
	return parsedBump{
		FromMajor: fromMajor,
		FromMinor: fromMinor,
		ToMajor:   toMajor,
		ToMinor:   toMinor,
	}, nil
}

func parseMajorMinor(s string) (int, int, error) {
	parts := strings.SplitN(s, ".", 2)
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("expected major.minor format, got %q", s)
	}
	major, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, 0, fmt.Errorf("invalid major %q: %w", parts[0], err)
	}
	minor, err := strconv.Atoi(parts[1])
	if err != nil {
		return 0, 0, fmt.Errorf("invalid minor %q: %w", parts[1], err)
	}
	return major, minor, nil
}

// GetPreviousYStream returns the previous Y-stream version for a given major.minor.
// If minor > 0, the previous Y-stream is simply major.(minor-1).
// If minor == 0, it looks up the major version bump history to find what came before.
func GetPreviousYStream(major, minor int) (prevMajor, prevMinor int, err error) {
	if minor > 0 {
		return major, minor - 1, nil
	}
	for _, b := range parsedBumps {
		if b.ToMajor == major && b.ToMinor == 0 {
			return b.FromMajor, b.FromMinor, nil
		}
	}
	return 0, 0, fmt.Errorf("don't know the previous Y-Stream for %d.%d", major, minor)
}
