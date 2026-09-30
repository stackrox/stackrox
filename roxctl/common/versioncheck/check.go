package versioncheck

import (
	"context"
	"fmt"
	"io"
	"strings"
	"sync/atomic"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/clientconn"
	"github.com/stackrox/rox/pkg/sliceutils"
	"github.com/stackrox/rox/pkg/version"
	"github.com/stackrox/rox/pkg/version/productstreams"
	"github.com/stackrox/rox/pkg/version/versioncompatibility"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

var suppressVersionMismatchWarning atomic.Bool

// SuppressVersionMismatchWarning if call the warnings in CentralVersionClientInterceptor will
// be suppressed
func SuppressVersionMismatchWarning() {
	suppressVersionMismatchWarning.Store(true)
}

// CentralVersionClientInterceptor returns a gRPC unary client interceptor that reads
// the Central version from response header and emits a warning if the
// versions of Central and roxctl are incompatible. The warning is emitted at most once per
// interceptor instance.
func CentralVersionClientInterceptor(w io.Writer) grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		var md metadata.MD
		opts = append(opts, grpc.Header(&md))
		// Response headers are populated after the RPC completes.
		err := invoker(ctx, method, req, reply, cc, opts...)
		if vals := md.Get(clientconn.CentralVersionHeader); len(vals) > 0 {
			if !suppressVersionMismatchWarning.Swap(true) {
				checkAndWarn(vals[0], w)
			}
		}
		return err
	}
}

// VersionResult holds structured version and compatibility information
// for the running roxctl and a given Central.
type VersionResult struct {
	CentralVersion            string   `json:"CentralVersion"`
	RoxctlVersion             string   `json:"RoxctlVersion"`
	CompatibleCentralVersions []string `json:"CompatibleCentralVersions"`
	Compatibility             string   `json:"Compatibility"`
	DisplayName               string   `json:"-"`
	Guidance                  string   `json:"Guidance"`
	Summary                   string   `json:"-"`
	Recommendation            string   `json:"-"`

	compatibility versioncompatibility.Compatibility
}

// ClassifyCentralVersion classifies the given Central version against
// the running roxctl version and returns structured version info.
func ClassifyCentralVersion(centralVersion string) (*VersionResult, error) {
	roxctlVersion := version.GetMainVersion()
	_, err := productstreams.ParseXYFromVersionString(roxctlVersion)
	if err != nil {
		return nil, errors.Wrapf(err, "parsing roxctl version %q", roxctlVersion)
	}
	centralXY, err := productstreams.ParseXYFromVersionString(centralVersion)
	if err != nil {
		return nil, errors.Wrapf(err, "parsing Central version %q", centralVersion)
	}

	compat, err := versioncompatibility.ClassifyVersion(centralXY)
	if err != nil {
		return nil, errors.Wrap(err, "classifying Central version")
	}

	compatVersions, err := versioncompatibility.CompatibleVersions()
	if err != nil {
		return nil, errors.Wrap(err, "getting compatible versions")
	}

	g := Guidance(compat)

	return &VersionResult{
		CentralVersion:            centralVersion,
		RoxctlVersion:             roxctlVersion,
		CompatibleCentralVersions: sliceutils.StringSlice[productstreams.XYVersion](compatVersions...),
		Compatibility:             compat.String(),
		DisplayName:               compat.DisplayName(),
		Guidance:                  g.String(),
		Summary:                   g.Summary,
		Recommendation:            g.Recommendation,
		compatibility:             compat,
	}, nil
}

// VersionGuidance holds structured guidance about version compatibility.
type VersionGuidance struct {
	Summary        string
	Recommendation string
}

func (g VersionGuidance) String() string {
	if g.Recommendation == "" {
		return g.Summary
	}
	return g.Summary + "\n" + g.Recommendation
}

// Guidance returns structured guidance for the given compatibility classification,
// describing the version relationship and recommended actions.
func Guidance(c versioncompatibility.Compatibility) VersionGuidance {
	switch c {
	case versioncompatibility.Matched:
		return VersionGuidance{
			Summary: "roxctl version is matched with Central.",
		}
	case versioncompatibility.CompatibleAhead:
		return VersionGuidance{
			Summary:        "Central version is compatible with roxctl but is ahead of roxctl.",
			Recommendation: "No immediate action is required. Use newer roxctl version to match Central for optimal functionality.",
		}
	case versioncompatibility.CompatibleBehind:
		return VersionGuidance{
			Summary: "Central version is compatible with roxctl but is behind roxctl.",
			Recommendation: "No immediate action is required. It is recommended to plan a Central upgrade. " +
				"If you prefer not to upgrade Central, consider using an older roxctl version to match the Central's version.",
		}
	case versioncompatibility.IncompatibleAhead:
		return VersionGuidance{
			Summary:        "Central version is outside the compatible version range and is ahead of roxctl.",
			Recommendation: "Use newer roxctl version to match Central, or at minimum to within the compatible version range.",
		}
	case versioncompatibility.IncompatibleBehind:
		return VersionGuidance{
			Summary:        "Central version is outside the compatible version range and is behind roxctl.",
			Recommendation: "Plan a Central upgrade or use older roxctl version to be within the compatible version range.",
		}
	default:
		return VersionGuidance{}
	}
}

func checkAndWarn(centralVersion string, w io.Writer) bool {
	result, err := ClassifyCentralVersion(centralVersion)
	if err != nil {
		return false
	}
	if result.compatibility != versioncompatibility.IncompatibleAhead && result.compatibility != versioncompatibility.IncompatibleBehind {
		return false
	}

	fmt.Fprintf(w, "Warning: roxctl %s and Central %s versions are outside the supported version skew range. Correct functioning is not guaranteed.\n", result.RoxctlVersion, centralVersion)
	fmt.Fprintf(w, "         %s\n", result.Summary)
	if result.Recommendation != "" {
		fmt.Fprintf(w, "         %s\n", result.Recommendation)
	}
	fmt.Fprintf(w, "         roxctl: %s | Central: %s | Compatible Centrals: %s\n",
		result.RoxctlVersion, centralVersion, strings.Join(result.CompatibleCentralVersions, ", "))
	return true
}
