package versioncheck

import (
	"context"
	"fmt"

	"sync/atomic"

	"github.com/stackrox/rox/pkg/clientconn"
	"github.com/stackrox/rox/pkg/version"
	"github.com/stackrox/rox/pkg/version/productstreams"
	"github.com/stackrox/rox/pkg/version/versioncompatibility"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

// WarnFunc is called when a version compatibility issue is detected.
type WarnFunc func(format string, a ...interface{})

// UnaryClientInterceptor returns a gRPC unary client interceptor that reads
// the Central version from response metadata and emits a warning if the
// versions are incompatible. The warning is emitted at most once per
// interceptor instance.
func UnaryClientInterceptor(warn WarnFunc) grpc.UnaryClientInterceptor {
	var warned atomic.Bool
	return func(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		var md metadata.MD
		opts = append(opts, grpc.Header(&md))
		err := invoker(ctx, method, req, reply, cc, opts...)
		if !warned.Load() {
			if vals := md.Get(clientconn.CentralVersionHeader); len(vals) > 0 {
				if checkAndWarn(vals[0], warn) {
					warned.Store(true)
				}
			}
		}
		return err
	}
}

func checkAndWarn(centralVersion string, warn WarnFunc) bool {
	remoteXY, err := productstreams.ParseXYFromVersionString(centralVersion)
	if err != nil {
		return false
	}
	localVersion := version.GetMainVersion()
	localXY, err := productstreams.ParseXYFromVersionString(localVersion)
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

	switch compat {
	case versioncompatibility.CompatibleBehind, versioncompatibility.CompatibleAhead:
		warn("roxctl version %s and Central version %s differ; some features may not work as expected",
			localVersion, centralVersion)
		return true
	case versioncompatibility.IncompatibleBehind, versioncompatibility.IncompatibleAhead:
		warn(incompatibleMessage(localVersion, centralVersion))
		return true
	}
	return false
}

func incompatibleMessage(localVersion, centralVersion string) string {
	versions, err := versioncompatibility.CompatibleVersions()
	if err != nil || len(versions) == 0 {
		return fmt.Sprintf("roxctl version %s is incompatible with Central version %s", localVersion, centralVersion)
	}
	return fmt.Sprintf(
		"roxctl version %s is incompatible with Central version %s; supported Central range for this roxctl is %s to %s",
		localVersion, centralVersion, versions[0], versions[len(versions)-1],
	)
}
