package main

import (
	"fmt"
	"regexp"
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

// CalculateReplacedVersion determines which version this release replaces
// This is complex logic that handles Y-Stream vs patch releases, version skips, and unreleased versions
func CalculateReplacedVersion(current, first, previousYStream string, skips []XyzVersion, unreleased string) (*XyzVersion, error) {
	currentXyz, err := ParseXyzVersion(current)
	if err != nil {
		return nil, err
	}

	firstXyz, err := ParseXyzVersion(first)
	if err != nil {
		return nil, err
	}

	previousXyz, err := ParseXyzVersion(previousYStream)
	if err != nil {
		return nil, err
	}

	// First version or earlier gets no replace
	if currentXyz.Compare(firstXyz) <= 0 {
		return nil, nil
	}

	// Determine initial replace candidate
	var initialReplace XyzVersion
	if currentXyz.Z == 0 {
		// New minor release replaces previous minor (e.g., 4.2.0 replaces 4.1.0)
		initialReplace = previousXyz
	} else {
		// Patch replaces previous patch (e.g., 4.2.2 replaces 4.2.1)
		initialReplace = XyzVersion{X: currentXyz.X, Y: currentXyz.Y, Z: currentXyz.Z - 1}
	}

	// If initial replace is unreleased, try previous one
	if unreleased != "" && initialReplace.String() == unreleased {
		prev, err := GetPreviousYStream(initialReplace.String())
		if err != nil {
			return nil, err
		}
		initialReplace, err = ParseXyzVersion(prev)
		if err != nil {
			return nil, err
		}
	}

	currentReplace := initialReplace

	// Skip over broken versions in the skips list
	skipMap := make(map[string]bool)
	for _, skip := range skips {
		skipMap[skip.String()] = true
	}

	for skipMap[currentReplace.String()] {
		// Try next patch
		currentReplace = XyzVersion{X: currentReplace.X, Y: currentReplace.Y, Z: currentReplace.Z + 1}
		// Safety: stop if we've reached or exceeded current version, or crossed into next minor release
		if currentReplace.Y != initialReplace.Y || currentReplace.Compare(currentXyz) >= 0 {
			break
		}
	}

	// Exception: if we're releasing immediate patch to broken version, still replace it
	// E.g., 4.1.0 is broken and in skips, 4.1.1 still replaces 4.1.0
	// This works because 4.1.1 will have skipRange allowing upgrade from 4.0.0
	if currentReplace.Compare(currentXyz) >= 0 {
		currentReplace = initialReplace
	}

	return &currentReplace, nil
}
