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
