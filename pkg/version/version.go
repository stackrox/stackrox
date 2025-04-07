package version

import (
	"fmt"
	"runtime"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/buildinfo"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/pkg/version/internal"
)

var (
	// ErrInvalidNumberOfComponents represents a version that has too few or
	// too many components for the given use case.
	ErrInvalidNumberOfComponents = errors.New("invalid number of components")
)

// GetMainVersion returns the tag of Rox.
func GetMainVersion() string {
	return internal.MainVersion
}

// getCollectorVersion returns the current Collector tag.
func getCollectorVersion() string {
	if env.CollectorVersion.Setting() != "" {
		return env.CollectorVersion.Setting()
	}
	return internal.CollectorVersion
}

// Versions represents a collection of various pieces of version information.
type Versions struct {
	// CollectorVersion is exported for compatibility with users that depend on `roxctl version --json` output.
	// Please do not depend on it. Rely on internal.CollectorVersion if you need the value from the COLLECTOR_VERSION file,
	// or rely on defaults.ImageFlavor if you need a default collector image tag.
	CollectorVersion string `json:"CollectorVersion"`
	GitCommit        string `json:"GitCommit"`
	GoVersion        string `json:"GoVersion"`
	MainVersion      string `json:"MainVersion"`
	Platform         string `json:"Platform"`
	// ScannerVersion is exported for compatibility with users that depend on `roxctl version --json` output.
	// Please do not depend on it. Rely on internal.ScannerVersion if you need the value from the SCANNER_VERSION file,
	// or rely on defaults.ImageFlavor if you need a default collector image tag.
	ScannerVersion string `json:"ScannerVersion"`
	ChartVersion   string `json:"ChartVersion"`
	// The Database versioning needs to be added by the caller due to scoping issues of config availabilty
	Database              string `json:"Database,omitempty"`
	DatabaseServerVersion string `json:"DatabaseServerVersion,omitempty"`
}

// GetAllVersionsDevelopment returns all of the various pieces of version information for development builds of the product.
func GetAllVersionsDevelopment() Versions {
	return Versions{
		CollectorVersion: getCollectorVersion(),
		GitCommit:        internal.GitShortSha,
		GoVersion:        runtime.Version(),
		MainVersion:      GetMainVersion(),
		Platform:         runtime.GOOS + "/" + runtime.GOARCH,
		ScannerVersion:   internal.ScannerVersion,
		ChartVersion:     GetChartVersion(),
	}
}

// GetAllVersionsUnified returns all of the various pieces of version information.
// Unified versions means that collector and scanner versions as shown in image tags are the same as main image version/tag.
// Unified versions are effective for the release images.
// Unified versions were introduced in the release 3.68.
func GetAllVersionsUnified() Versions {
	v := GetAllVersionsDevelopment()
	v.CollectorVersion = GetMainVersion()
	v.ScannerVersion = GetMainVersion()
	return v
}

// parsedMainVersion contains a parsed StackRox Main Version (see https://stack-rox.atlassian.net/wiki/spaces/StackRox/pages/673808422/Product+Versioning+yes+again).
type ParsedMainVersion struct {
	MarketingMajor int
	MarketingMinor *int
	EngRelease     int
	PatchLevel     string // A string, since the current scheme allows versions like "3.0.49.x-1-ga0897a21ee" where patch level is "x".
	PatchSuffix    string // Everything after the (dash-separated) `PatchLevel`.
}

type XYVersion struct {
	X int
	Y int
}

func ParseXYVersion(versionString string) (XYVersion, error) {
	parsedVersion, err := parseVersion(versionString)
	if err != nil {
		return XYVersion{}, errors.Errorf("parsing version string %q: %v", versionString, err)
	}
	xyVersion := XYVersion{X: parsedVersion.MarketingMajor, Y: parsedVersion.EngRelease} // Historical reasons.
	return xyVersion, nil
}

func MustParseXYVersion(versionString string) XYVersion {
	parsedVersion, err := ParseXYVersion(versionString)
	if err != nil {
		panic(err)
	}
	return parsedVersion
}

func GetMainXYVersion() XYVersion {
	xyVersion, err := ParseXYVersion(internal.MainVersion)
	if err != nil {
		panic(err)
	}
	return xyVersion
}

func (a XYVersion) LessOrEqual(b XYVersion) bool {
	if a.X != b.X {
		return a.X <= b.X
	}
	return a.Y <= b.Y
}

func (a XYVersion) Less(b XYVersion) bool {
	return !b.LessOrEqual(a)
}

func (v XYVersion) Serialize() string {
	return fmt.Sprintf("%v.%v", v.X, v.Y)
}

func parseMainVersion(mainVersion string) (ParsedMainVersion, error) {
	parts := strings.SplitN(mainVersion, "-", 2)

	components := strings.SplitN(parts[0], ".", 4)

	nComponents := len(components)
	if nComponents < 3 || nComponents > 4 {
		return ParsedMainVersion{}, fmt.Errorf("%w (expected 3 or 4, got %d)", ErrInvalidNumberOfComponents, nComponents)
	}

	marketingMajor, err := strconv.Atoi(components[0])
	if err != nil {
		return ParsedMainVersion{}, errors.Wrapf(err, "invalid marketing major version (%q)", components[0])
	}

	var marketingMinorOpt *int
	engReleaseOfs := 1
	if len(components) == 4 {
		// It's highly unlikely we're going to ever use non-SemVer product versions that include four components.
		// However, there's a lot of test code that was written when this was the way of versioning. Therefore this
		// parsing still exists.
		// TODO: clean up all versioning and test code that deals with "marketing minor".
		marketingMinor, err := strconv.Atoi(components[1])
		if err != nil {
			return ParsedMainVersion{}, errors.Wrapf(err, "invalid marketing minor major version (%q)", components[1])
		}
		engReleaseOfs = 2
		marketingMinorOpt = &marketingMinor
	}

	engRelease, err := strconv.Atoi(components[engReleaseOfs])
	if err != nil {
		return ParsedMainVersion{}, errors.Wrapf(err, "invalid eng release version (%q)", components[engReleaseOfs])
	}

	patchLevel := components[engReleaseOfs+1]

	if patchLevel == "" {
		// Main Version scheme requires the "patch" component to be non-empty.
		return ParsedMainVersion{}, errors.New("empty patch component")
	}

	patchSuffix := ""
	if len(parts) == 2 {
		patchSuffix = parts[1]
	}

	parsedVersion := ParsedMainVersion{
		MarketingMajor: marketingMajor,
		MarketingMinor: marketingMinorOpt,
		EngRelease:     engRelease,
		PatchLevel:     patchLevel,
		PatchSuffix:    patchSuffix,
	}

	return parsedVersion, nil
}

// GetChartVersion derives a Chart Version string from the provided Main Version string.
func GetChartVersion() string {
	chartVersion, err := deriveChartVersion(GetMainVersion())
	utils.Should(err)
	return chartVersion
}

// deriveChartVersion derives a Chart Version string from the provided Main Version string.
func deriveChartVersion(mainVersion string) (string, error) {
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

	if parsedMainVersion.MarketingMajor != 3 && parsedMainVersion.MarketingMinor != nil {
		return "", errors.Errorf(
			"unexpected main version %s: minor marketing version component is not supported after the product version 3",
			mainVersion)
	}

	// In 3[.0].y.z era Y/Minor versions were used as Helm Major (Main Major, 3, was ignored). Main Minor versions got
	// up to 74 and occupied Helm chart versions 74.something.something. Because of that, we have to assign even bigger
	// Major Helm chart version for release 4.0.0 and later. Otherwise, if we simply take Main Major (e.g. 4) and assign
	// it to Helm Major, it will be recognized as old according to SemVer (4.0.0<74.0.0). Therefore, we pad Main Major
	// with two trailing zeroes making such chart appear newer than charts from 3[.0].y.z era, as it should.
	chartMajor := parsedMainVersion.MarketingMajor * 100

	chartMinor := parsedMainVersion.EngRelease
	chartPatch := patchLevelInteger

	chartSuffix := ""
	if parsedMainVersion.PatchSuffix != "" {
		chartSuffix = "-" + parsedMainVersion.PatchSuffix
	}

	chartVersion := fmt.Sprintf("%d.%d.%d%s", chartMajor, chartMinor, chartPatch, chartSuffix)
	return chartVersion, nil
}

// IsReleaseVersion tells whether the binary is built for a release.
func IsReleaseVersion() bool {
	return buildinfo.ReleaseBuild && !buildinfo.TestBuild &&
		GetMainVersion() != "" &&
		!strings.Contains(GetMainVersion(), "-")
}

// IsPriorToScannerV4 returns true if version represents a version of ACS from prior to the
// introduction of Scanner V4. Will return an error if cannot determine result.
func IsPriorToScannerV4(version string) (bool, error) {
	parsed, err := parseVersion(version)
	if err != nil {
		return false, err
	}

	x := parsed.MarketingMajor
	y := parsed.EngRelease
	z := parsed.PatchLevel

	if x < 4 {
		return true, nil
	}

	if x > 4 {
		return false, nil
	}

	// x == 4
	// Scanner V4 was introduced in 4.4
	return y < 3 || (y == 3 && (z == "" || z != "x")), nil
}

// Variants breaks a version into a series of version strings starting with
// the most specific to the least specific, stopping at X.Y. It uses
// parseVersion to validate the format.
func Variants(version string) ([]string, error) {
	_, err := parseVersion(version)
	if err != nil {
		return nil, err
	}

	var res []string
	for i := len(version); i != -1; i = strings.LastIndex(version, "-") {
		res = append(res, version[:i])
		version = version[:i]
	}

	for i := strings.LastIndex(version, "."); i != -1 && strings.Count(version, ".") > 1; i = strings.LastIndex(version, ".") {
		res = append(res, version[:i])
		version = version[:i]
	}

	return res, nil
}

// parseVersion mimics parseMainVersion but allows for versions to be in a
// format that would otherwise not be a valid main version (such as X.Y).
func parseVersion(version string) (ParsedMainVersion, error) {
	parsed, err := parseMainVersion(version)
	if err != nil && !errors.Is(err, ErrInvalidNumberOfComponents) {
		return ParsedMainVersion{}, err
	}

	if err == nil {
		return parsed, nil
	}

	before, _, _ := strings.Cut(version, "-")
	parts := strings.Split(before, ".")
	if len(parts) != 2 {
		return ParsedMainVersion{}, fmt.Errorf("%w (expected 2-4, got %d)", ErrInvalidNumberOfComponents, len(parts))
	}

	x, err := strconv.Atoi(parts[0])
	if err != nil {
		return ParsedMainVersion{}, err
	}

	y, err := strconv.Atoi(parts[1])
	if err != nil {
		return ParsedMainVersion{}, err
	}

	return ParsedMainVersion{MarketingMajor: x, EngRelease: y}, nil
}
