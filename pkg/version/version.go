package version

import (
	"fmt"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/buildinfo"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/pkg/version/internal"
)

// GetMainVersion returns the tag of Rox.
func GetMainVersion() string {
	return internal.MainVersion
}

// GetCollectorVersion returns the current Collector tag.
func GetCollectorVersion() string {
	if env.CollectorVersion.Setting() != "" {
		return env.CollectorVersion.Setting()
	}
	return internal.CollectorVersion
}

// GetScannerVersion returns the current Scanner tag.
func GetScannerVersion() string {
	return internal.ScannerVersion
}

// Versions represents a collection of various pieces of version information.
type Versions struct {
	BuildDate        time.Time `json:"BuildDate"`
	CollectorVersion string    `json:"CollectorVersion"`
	GitCommit        string    `json:"GitCommit"`
	GoVersion        string    `json:"GoVersion"`
	MainVersion      string    `json:"MainVersion"`
	Platform         string    `json:"Platform"`
	ScannerVersion   string    `json:"ScannerVersion"`
	ChartVersion     string    `json:"ChartVersion"`
}

// GetAllVersions returns all of the various pieces of version information.
func GetAllVersions() Versions {
	v := Versions{
		BuildDate:        buildinfo.BuildTimestamp(),
		CollectorVersion: GetCollectorVersion(),
		GitCommit:        internal.GitShortSha,
		GoVersion:        runtime.Version(),
		MainVersion:      GetMainVersion(),
		Platform:         runtime.GOOS + "/" + runtime.GOARCH,
		ScannerVersion:   GetScannerVersion(),
		ChartVersion:     GetChartVersion(),
	}
	return v
}

// parsedMainVersion contains a parsed StackRox Main Version (see https://stack-rox.atlassian.net/wiki/spaces/StackRox/pages/673808422/Product+Versioning+yes+again).
type parsedMainVersion struct {
	MarketingMajor int
	MarketingMinor int
	EngRelease     int
	PatchLevel     string // A string, since the current scheme allows versions like "3.0.49.x-1-ga0897a21ee" where patch level is "x".
	PatchSuffix    string // Everything after the (dash-separated) `PatchLevel`.
}

func parseMainVersion(mainVersion string) (parsedMainVersion, error) {
	components := strings.SplitN(mainVersion, ".", 4)

	nComponents := len(components)
	if nComponents != 4 {
		return parsedMainVersion{}, errors.Errorf("invalid number of components (expected 4, got %d)", nComponents)
	}

	marketingMajor, err := strconv.Atoi(components[0])
	if err != nil {
		return parsedMainVersion{}, errors.Wrapf(err, "invalid marketing major version (%q)", components[0])
	}

	marketingMinor, err := strconv.Atoi(components[1])
	if err != nil {
		return parsedMainVersion{}, errors.Wrapf(err, "invalid marketing minor major version (%q)", components[1])
	}

	engRelease, err := strconv.Atoi(components[2])
	if err != nil {
		return parsedMainVersion{}, errors.Wrapf(err, "invalid eng release version (%q)", components[2])
	}

	patchComponents := strings.SplitN(components[3], "-", 2)
	nPatchComponents := len(patchComponents)

	if nPatchComponents == 0 {
		// Main Version scheme requires the "patch" component to be non-empty.
		return parsedMainVersion{}, errors.New("empty patch component")
	}

	patchLevel := patchComponents[0]
	patchSuffix := ""
	if nPatchComponents == 2 {
		patchSuffix = patchComponents[1]
	}

	parsedVersion := parsedMainVersion{
		MarketingMajor: marketingMajor,
		MarketingMinor: marketingMinor,
		EngRelease:     engRelease,
		PatchLevel:     patchLevel,
		PatchSuffix:    patchSuffix,
	}

	return parsedVersion, nil
}

// GetChartVersion derives a Chart Version string from the provided Main Version string.
func GetChartVersion() string {
	return DeriveChartVersion(GetMainVersion())
}

func doDeriveChartVersion(mainVersion string) (string, error) {
	parsedMainVersion, err := parseMainVersion(mainVersion)
	if err != nil {
		return "", errors.Wrapf(err, "failed to parse main version %q", mainVersion)
	}

	// For SemVer validity, the first component of the patch level should contain a number or an "x", which we map to 0.
	patchLevelInteger := 0
	if parsedMainVersion.PatchLevel != "x" {
		patchLevelInteger, err = strconv.Atoi(parsedMainVersion.PatchLevel)
		if err != nil {
			return "", errors.Wrap(err, "patch level expected to contain a number")
		}
	}

	// We need to make sure that the patch suffix will begin with a number for obtaining a valid SemVer 2 version string.
	patchSuffixWithInitialNumber := parsedMainVersion.PatchSuffix
	if patchSuffixWithInitialNumber == "" {
		// For release versions.
		patchSuffixWithInitialNumber = "0"
	} else if c := patchSuffixWithInitialNumber[0]; !(c >= '0' && c <= '9') {
		// Prefix with "0-".
		patchSuffixWithInitialNumber = fmt.Sprintf("0-%s", patchSuffixWithInitialNumber)
	}

	chartVersion := fmt.Sprintf("%d.%d.%s", parsedMainVersion.EngRelease, patchLevelInteger, patchSuffixWithInitialNumber)
	return chartVersion, nil
}

// DeriveChartVersion derives a Chart Version string from the provided Main Version string.
func DeriveChartVersion(mainVersion string) string {
	chartVersion, err := doDeriveChartVersion(mainVersion)
	utils.Should(err)
	return chartVersion
}
