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

// UnaryClientInterceptor returns a gRPC unary client interceptor that reads
// the Central version from response metadata and emits a warning if the
// versions are incompatible. The warning is emitted at most once per
// interceptor instance.
func UnaryClientInterceptor(w io.Writer) grpc.UnaryClientInterceptor {
	var warned atomic.Bool
	return func(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		getHeader := headerSource(&opts)
		err := invoker(ctx, method, req, reply, cc, opts...)
		if !warned.Load() {
			if vals := getHeader().Get(clientconn.CentralVersionHeader); len(vals) > 0 {
				if checkAndWarn(vals[0], w) {
					warned.Store(true)
				}
			}
		}
		return err
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

	versionRange, err := versioncompatibility.CompatibleVersions()
	if err != nil {
		return false
	}
	compatRange := formatVersionRange(versionRange)

	switch compat {
	case versioncompatibility.IncompatibleAhead:
		fmt.Fprintf(w, "Warning: Your roxctl %s is too old for this Central %s. "+
			"Correct functioning is not guaranteed. "+
			"Use roxctl version matching the Central version or at least such that the Central version is within the roxctl compatibility range.\n",
			roxctlVersion, centralVersion)
		fmt.Fprintf(w, "         roxctl: %s | Central: %s | Compatible Centrals: %s\n",
			roxctlVersion, centralVersion, compatRange)
		return true
	case versioncompatibility.IncompatibleBehind:
		fmt.Fprintf(w, "Warning: Your roxctl %s is too new for this Central %s. "+
			"Correct functioning is not guaranteed. "+
			"Use roxctl version matching the Central version or at least such that the Central version is within the roxctl compatibility range.\n",
			roxctlVersion, centralVersion)
		fmt.Fprintf(w, "         roxctl: %s | Central: %s | Compatible Centrals: %s\n",
			roxctlVersion, centralVersion, compatRange)
		return true
	}
	return false
}

// headerSource returns a function that, when called after the RPC completes,
// yields the response metadata. If opts already contains a grpc.Header option
// the existing pointer is reused; otherwise a new one is appended.
func headerSource(opts *[]grpc.CallOption) func() metadata.MD {
	for _, o := range *opts {
		if h, ok := o.(grpc.HeaderCallOption); ok {
			return func() metadata.MD { return *h.HeaderAddr }
		}
	}
	var md metadata.MD
	*opts = append(*opts, grpc.Header(&md))
	return func() metadata.MD { return md }
}

func formatVersionRange(versions []productstreams.XYVersion) string {
	strs := make([]string, len(versions))
	for i, v := range versions {
		strs[i] = v.String()
	}
	return strings.Join(strs, ", ")
}
