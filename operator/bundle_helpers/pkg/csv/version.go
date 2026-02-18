package csv

import (
	"fmt"
	"regexp"
	"slices"
	"strconv"
)

// XyzVersion represents a semantic version with major.minor.patch components
type XyzVersion struct {
	X int // Major version
	Y int // Minor version
	Z int // Patch version
}

var versionRegex = regexp.MustCompile(`^v?(\d+)\.(\d+)\.(x|\d+)(-.+)?$`)

// ParseXyzVersion parses a version string into XyzVersion
// Supports formats: "3.74.0", "3.74.0-123", "3.74.x-nightly-20230224"
func ParseXyzVersion(versionStr string) (XyzVersion, error) {
	matches := versionRegex.FindStringSubmatch(versionStr)
	if matches == nil {
		return XyzVersion{}, fmt.Errorf("invalid version format: %s", versionStr)
	}

	// Atoi errors are safely ignored here because the regex ensures matches[1] and matches[2]
	// contain only digits (\d+), making conversion to int guaranteed to succeed.
	x, _ := strconv.Atoi(matches[1])
	y, _ := strconv.Atoi(matches[2])

	z := 0
	if matches[3] != "x" {
		// matches[3] is either "x" or digits (\d+), so Atoi is safe here too
		z, _ = strconv.Atoi(matches[3])
	}

	return XyzVersion{X: x, Y: y, Z: z}, nil
}

// String returns the version as "x.y.z"
func (v XyzVersion) String() string {
	return fmt.Sprintf("%d.%d.%d", v.X, v.Y, v.Z)
}

// Compare returns -1 if v < other, 0 if equal, 1 if v > other
func (v XyzVersion) Compare(other XyzVersion) int {
	if v.X != other.X {
		if v.X < other.X {
			return -1
		}
		return 1
	}
	if v.Y != other.Y {
		if v.Y < other.Y {
			return -1
		}
		return 1
	}
	if v.Z != other.Z {
		if v.Z < other.Z {
			return -1
		}
		return 1
	}
	return 0
}

// GetPreviousYStream returns the previous Y-Stream version
// Y-Stream versions have patch number = 0 (e.g., 3.73.0, 3.74.0, 4.0.0)
// This implements the logic from scripts/get-previous-y-stream.sh
func GetPreviousYStream(versionStr string) (string, error) {
	v, err := ParseXyzVersion(versionStr)
	if err != nil {
		return "", err
	}

	if v.Y > 0 {
		// If minor version > 0, previous Y-Stream is one minor less
		return fmt.Sprintf("%d.%d.0", v.X, v.Y-1), nil
	}

	// For major version bumps, maintain hardcoded mapping
	switch v.X {
	case 4:
		return "3.74.0", nil
	case 1:
		// 0.0.0 was never released, but used for trunk builds
		return "0.0.0", nil
	default:
		return "", fmt.Errorf("don't know the previous Y-Stream for %d.%d", v.X, v.Y)
	}
}

// initialReplaceFor calculates the initial replacement version based on current and previous Y-stream versions
func initialReplaceFor(current, previousXyz XyzVersion) XyzVersion {
	if current.Z == 0 {
		// New minor release replaces previous minor (e.g., 4.2.0 replaces 4.1.0)
		return previousXyz
	}
	// Patch replaces previous patch (e.g., 4.2.2 replaces 4.2.1)
	return XyzVersion{X: current.X, Y: current.Y, Z: current.Z - 1}
}

// adjustForUnreleased adjusts the initial replacement if it matches an unreleased version
func adjustForUnreleased(initialReplace XyzVersion, unreleased string) (XyzVersion, error) {
	if unreleased == "" || initialReplace.String() != unreleased {
		return initialReplace, nil
	}

	prev, err := GetPreviousYStream(initialReplace.String())
	if err != nil {
		return XyzVersion{}, err
	}
	return ParseXyzVersion(prev)
}

// advancePastSkips advances the replacement version past any skipped versions
func advancePastSkips(initialReplace, currentXyz XyzVersion, skips []XyzVersion) XyzVersion {
	current := initialReplace
	for {
		// Check safety BEFORE incrementing
		next := XyzVersion{X: current.X, Y: current.Y, Z: current.Z + 1}
		if next.Y != initialReplace.Y || next.Compare(currentXyz) >= 0 {
			break
		}

		if !slices.Contains(skips, current) {
			break
		}

		current = next
	}

	// Exception: if we're releasing immediate patch to broken version, still replace it
	if current.Compare(currentXyz) >= 0 {
		return initialReplace
	}
	return current
}

// CalculateReplacedVersion determines which version this release replaces
// Handles Y-Stream vs patch releases, version skips, and unreleased versions.
func CalculateReplacedVersion(current, first, previousYStream string, skips []XyzVersion, unreleased string) (XyzVersion, error) {
	currentXyz, err := ParseXyzVersion(current)
	if err != nil {
		return XyzVersion{}, err
	}

	firstXyz, err := ParseXyzVersion(first)
	if err != nil {
		return XyzVersion{}, err
	}

	previousXyz, err := ParseXyzVersion(previousYStream)
	if err != nil {
		return XyzVersion{}, err
	}

	// First version or earlier gets no replace
	if currentXyz.Compare(firstXyz) <= 0 {
		return XyzVersion{}, nil
	}

	// Determine initial replace candidate
	initialReplace := initialReplaceFor(currentXyz, previousXyz)

	// If this version is not yet released, try previous one
	// E.g. 4.5 branch was cut and the 4.6.x tag created, but the 4.5 release process is still in progress
	initialReplace, err = adjustForUnreleased(initialReplace, unreleased)
	if err != nil {
		return XyzVersion{}, err
	}

	// Skip over broken versions in the skips list
	currentReplace := advancePastSkips(initialReplace, currentXyz, skips)

	return currentReplace, nil
}
