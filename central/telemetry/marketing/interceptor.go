package marketing

import (
	"context"

	mpkg "github.com/stackrox/rox/pkg/telemetry/marketing"

	erroxGRPC "github.com/stackrox/rox/pkg/errox/grpc"
	grpcError "github.com/stackrox/rox/pkg/grpc/errors"
	"github.com/stackrox/rox/pkg/grpc/requestinfo"
	"github.com/stackrox/rox/pkg/set"
	"google.golang.org/grpc"
)

func track(ctx context.Context, t mpkg.Telemeter, err error, info *grpc.UnaryServerInfo, trackedPaths set.FrozenSet[string]) {
	ri := requestinfo.FromContext(ctx)

	userAgent := "unknown"
	if agents := ri.Metadata.Get("User-Agent"); len(agents) != 0 {
		userAgent = agents[0]
	}

	// Track the user agent of every request:
	t.Track(userAgent, "User-Agent")

	var code int
	var path string
	if ri.HTTPRequest != nil && ri.HTTPRequest.URL != nil {
		path = ri.HTTPRequest.URL.Path
		code = grpcError.ErrToHTTPStatus(err)
	} else {
		path = info.FullMethod
		code = int(erroxGRPC.RoxErrorToGRPCCode(err))
	}

	// Track the API path and error code of some requests:
	if trackedPaths.Contains(path) {
		t.TrackProps(userAgent, "API Call", map[string]any{
			"Path": path,
			"Code": code,
		})
	}
}
