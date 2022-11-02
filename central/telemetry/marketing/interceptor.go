package marketing

import (
	"context"

	mpkg "github.com/stackrox/rox/pkg/telemetry/marketing"

	"github.com/stackrox/rox/pkg/grpc/requestinfo"
	"github.com/stackrox/rox/pkg/set"
	"google.golang.org/grpc"
)

func interceptor(d *mpkg.Device, t mpkg.Telemeter) grpc.UnaryServerInterceptor {

	trackedPaths := set.NewFrozenSet(d.ApiPaths...)

	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp any, err error) {
		resp, err = handler(ctx, req)

		ri := requestinfo.FromContext(ctx)
		if ri.HTTPRequest != nil && ri.HTTPRequest.URL != nil {
			path := ri.HTTPRequest.URL.Path
			ua := ri.HTTPRequest.Headers.Get("User-Agent")
			t.Track(ua, "User-Agent")
			if trackedPaths.Contains(path) {
				log.Debug("Telemetry tracks ", path)
				t.TrackProp(ua, "API Call", "Path", path)
			}
		} else {
			log.Info("No HTTP data: ")
		}
		return
	}
}
