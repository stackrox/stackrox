package version

import (
	"fmt"

	"github.com/stackrox/rox/generated/storage"
)

const (
	supportedMinorVersionSkew = 3
)

func compatibleRange(ver string) (min, max string) {
	parsed, err := parseVersion(ver)
	if err != nil {
		return "", ""
	}
	minMinor := parsed.EngRelease - supportedMinorVersionSkew + 1
	if minMinor < 0 {
		minMinor = 0
	}
	return fmt.Sprintf("%d.%d", parsed.MarketingMajor, minMinor),
		fmt.Sprintf("%d.%d", parsed.MarketingMajor, parsed.EngRelease)
}

// MinCompatibleSensorVersion returns the oldest sensor version that
// the current Central build is compatible with.
func MinCompatibleSensorVersion() string {
	min, _ := compatibleRange(GetMainVersion())
	return min
}

// MaxCompatibleSensorVersion returns the newest sensor version that
// the current Central build knows about.
func MaxCompatibleSensorVersion() string {
	_, max := compatibleRange(GetMainVersion())
	return max
}

// ComputeVersionSkew determines the version skew status between Central and a Sensor.
// It compares only the X.Y (major.minor) components, ignoring patch versions.
func ComputeVersionSkew(centralVersion, sensorVersion string) *storage.VersionSkew {
	result := &storage.VersionSkew{}

	if sensorVersion == "" || centralVersion == "" {
		return result
	}

	centralParsed, err := parseVersion(centralVersion)
	if err != nil {
		return result
	}

	sensorParsed, err := parseVersion(sensorVersion)
	if err != nil {
		return result
	}

	minMinor := centralParsed.EngRelease - supportedMinorVersionSkew + 1
	if minMinor < 0 {
		minMinor = 0
	}

	result.MinCompatibleSensorVersion = fmt.Sprintf("%d.%d", centralParsed.MarketingMajor, minMinor)
	result.MaxCompatibleSensorVersion = fmt.Sprintf("%d.%d", centralParsed.MarketingMajor, centralParsed.EngRelease)

	if centralParsed.MarketingMajor == sensorParsed.MarketingMajor &&
		centralParsed.EngRelease == sensorParsed.EngRelease {
		result.Status = storage.VersionSkewStatus_VERSION_SKEW_STATUS_MATCHING
		return result
	}

	if sensorParsed.MarketingMajor > centralParsed.MarketingMajor ||
		(sensorParsed.MarketingMajor == centralParsed.MarketingMajor &&
			sensorParsed.EngRelease > centralParsed.EngRelease) {
		result.Status = storage.VersionSkewStatus_VERSION_SKEW_STATUS_INCOMPATIBLE
		result.Reason = storage.VersionSkewReason_VERSION_SKEW_REASON_SENSOR_AHEAD
		return result
	}

	if sensorParsed.MarketingMajor == centralParsed.MarketingMajor &&
		sensorParsed.EngRelease >= minMinor {
		result.Status = storage.VersionSkewStatus_VERSION_SKEW_STATUS_COMPATIBLE
		return result
	}

	result.Status = storage.VersionSkewStatus_VERSION_SKEW_STATUS_INCOMPATIBLE
	result.Reason = storage.VersionSkewReason_VERSION_SKEW_REASON_SENSOR_TOO_OLD
	return result
}
