package featureclient

import (
	"context"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/logging"
	"google.golang.org/grpc"
)

var (
	log = logging.LoggerForModule()
)

func ConfigureFeaturesFromCentralSource(ctx context.Context, cc grpc.ClientConnInterface) {
	featureClient := v1.NewFeatureFlagServiceClient(cc)
	response, err := featureClient.GetFeatureFlags(ctx, &v1.Empty{})
	if err != nil {
		log.Errorf("Failed to get feature flags from central configuration source: %v", err)
		return
	}
	for _, flag := range response.GetFeatureFlags() {
		features.Flags[flag.EnvVar].Set(flag.GetEnabled(), features.FlagSource_CENTRAL)
	}
}
