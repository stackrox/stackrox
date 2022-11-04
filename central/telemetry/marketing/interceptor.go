package marketing

import (
	"context"
	"strings"

	mpkg "github.com/stackrox/rox/pkg/telemetry/marketing"

	erroxGRPC "github.com/stackrox/rox/pkg/errox/grpc"
	grpcError "github.com/stackrox/rox/pkg/grpc/errors"
	"github.com/stackrox/rox/pkg/grpc/requestinfo"
	"github.com/stackrox/rox/pkg/set"
	"google.golang.org/grpc"
)

func track(ctx context.Context, t mpkg.Telemeter, err error, info *grpc.UnaryServerInfo, trackedPaths set.FrozenSet[string]) {
	userAgent, path, code := getRequestDetails(ctx, err, info)

	// Track the API path and error code of some requests:
	if path != "/v1/ping" && (trackedPaths.Contains("*") || trackedPaths.Contains(path)) {
		t.TrackProps("API Call", map[string]any{
			"Path":       path,
			"Code":       code,
			"User-Agent": userAgent,
		})
	}
}

func getRequestDetails(ctx context.Context, err error, info *grpc.UnaryServerInfo) (string, string, int) {
	ri := requestinfo.FromContext(ctx)
	userAgent := strings.Join(ri.Metadata.Get("User-Agent"), ", ")

	if ri.HTTPRequest != nil && ri.HTTPRequest.URL != nil {
		return userAgent, ri.HTTPRequest.URL.Path, grpcError.ErrToHTTPStatus(err)
	}
	return userAgent, info.FullMethod, int(erroxGRPC.RoxErrorToGRPCCode(err))
}
