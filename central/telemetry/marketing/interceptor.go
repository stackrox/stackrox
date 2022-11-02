package marketing

import (
	"context"

	mpkg "github.com/stackrox/rox/pkg/telemetry/marketing"

	grpcErrox "github.com/stackrox/rox/pkg/errox/grpc"
	grpcErr "github.com/stackrox/rox/pkg/grpc/errors"
	"github.com/stackrox/rox/pkg/grpc/requestinfo"
	"github.com/stackrox/rox/pkg/set"
	"google.golang.org/grpc"
)

func interceptor(d *mpkg.Device, t mpkg.Telemeter) grpc.UnaryServerInterceptor {

	trackedPaths := set.NewFrozenSet(d.ApiPaths...)
	log.Info("API path telemetry enabled for: ", trackedPaths.AsSlice())

	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp any, err error) {
		resp, err = handler(ctx, req)

		ri := requestinfo.FromContext(ctx)

		uarr := ri.Metadata.Get("User-Agent")
		var userAgent string
		if len(uarr) == 0 {
			userAgent = "unknown"
		} else {
			userAgent = uarr[0]
		}
		t.Track(userAgent, "User-Agent")

		var code int
		var path string
		if ri.HTTPRequest != nil && ri.HTTPRequest.URL != nil {
			path = ri.HTTPRequest.URL.Path
			code = grpcErr.ErrToHTTPStatus(err)
		} else {
			path = info.FullMethod
			code = int(grpcErrox.RoxErrorToGRPCCode(err))
		}
		log.Info("Telemetry intercepted ", path)

		if trackedPaths.Contains(path) {
			log.Info("Telemetry tracks ", path)
			t.TrackProps(userAgent, "API Call", map[string]any{
				"Path": path,
				"Code": code,
			})
		}
		return
	}
}
