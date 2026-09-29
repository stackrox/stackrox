package versioncheck

import (
	"context"
	"fmt"
	"io"
	"strings"
	"sync/atomic"

	"github.com/stackrox/rox/pkg/clientconn"
	"github.com/stackrox/rox/pkg/version"
	"github.com/stackrox/rox/pkg/version/productstreams"
	"github.com/stackrox/rox/pkg/version/versioncompatibility"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

var suppressWarning atomic.Bool

// SuppressWarning if call the warnings in CentralVersionClientInterceptor will
// be suppressed
func SuppressWarning() {
	suppressWarning.Store(true)
}

// CentralVersionClientInterceptor returns a gRPC unary client interceptor that reads
// the Central version from response header and emits a warning if the
// versions of Central and roxctl are incompatible. The warning is emitted at most once per
// interceptor instance.
func CentralVersionClientInterceptor(w io.Writer) grpc.UnaryClientInterceptor {
	var checked atomic.Bool
	return func(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		var md metadata.MD
		opts = append(opts, grpc.Header(&md))
		// Response headers are populated after the RPC completes.
		err := invoker(ctx, method, req, reply, cc, opts...)
		if vals := md.Get(clientconn.CentralVersionHeader); len(vals) > 0 {
			if !checked.Swap(true) && !suppressWarning.Load() {
				checkAndWarn(vals[0], w)
			}
		}
		return err
	}
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
				"If you prefer not to upgrade Central, consider using an older roxctl version to match Central.",
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
	remoteXY, err := productstreams.ParseXYFromVersionString(centralVersion)
	if err != nil {
		return false
	}
	roxctlVersion := version.GetMainVersion()
	localXY, err := productstreams.ParseXYFromVersionString(roxctlVersion)
	if err != nil {
		return false
	}
	if localXY == remoteXY {
		return false
	}

	compat, err := versioncompatibility.ClassifyVersion(remoteXY)
	if err != nil {
		return false
	}
	if compat != versioncompatibility.IncompatibleAhead && compat != versioncompatibility.IncompatibleBehind {
		return false
	}

	versionRange, err := versioncompatibility.CompatibleVersions()
	if err != nil {
		return false
	}
	compatRange := formatVersionRange(versionRange)

	g := Guidance(compat)
	fmt.Fprintf(w, "Warning: roxctl %s and Central %s are incompatible.\n", roxctlVersion, centralVersion)
	fmt.Fprintf(w, "         %s\n", g.Summary)
	if g.Recommendation != "" {
		fmt.Fprintf(w, "         %s\n", g.Recommendation)
	}
	fmt.Fprintf(w, "         roxctl: %s | Central: %s | Compatible Centrals: %s\n",
		roxctlVersion, centralVersion, compatRange)
	return true
}

func formatVersionRange(versions []productstreams.XYVersion) string {
	strs := make([]string, len(versions))
	for i, v := range versions {
		strs[i] = v.String()
	}
	return strings.Join(strs, ", ")
}
