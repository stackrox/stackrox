package marketing

import (
	"context"
	"time"

	"github.com/stackrox/rox/pkg/set"
	mpkg "github.com/stackrox/rox/pkg/telemetry/marketing"
	"github.com/stackrox/rox/pkg/telemetry/marketing/segment"
	"google.golang.org/grpc"
)

// Enabled returns true if marketing telemetery data collection is enabled.
func Enabled() bool {
	return segment.Enabled()
}

// Init initializes the periodic telemetry data gatherer and returns an GRPC API
// call inteceptor. Returns nil if telemetry data collection is disabled.
func Init() grpc.UnaryServerInterceptor {
	if Enabled() {
		config, err := mpkg.GetInstanceConfig()
		if err != nil {
			log.Errorf("Failed to get device telemetry configuration: %v", err)
			return nil
		}

		telemeter := segment.Init(config)

		InitGatherer(telemeter, 5*time.Minute)

		trackedPaths := set.NewFrozenSet(config.APIPaths...)
		log.Info("Telemetry device ID:", config.ID)
		log.Info("API path telemetry enabled for: ", config.APIPaths)

		return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp any, err error) {
			resp, err = handler(ctx, req)
			go track(ctx, telemeter, err, info, trackedPaths)
			return
		}
	}
	return nil
}
