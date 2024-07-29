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
	log.Info("Pulling feature flag configuration from central source")
	log.Info("Before")
	for _, f := range features.Flags {
		log.Info("Flag: ", f.EnvVar(), ": ", f.Enabled())
	}
	featureClient := v1.NewFeatureFlagServiceClient(cc)
	response, err := featureClient.GetFeatureFlags(ctx, &v1.Empty{})
	if err != nil {
		log.Errorf("Failed to get feature flags from central configuration source: %v", err)
		return
	}
	for _, flag := range response.GetFeatureFlags() {
		features.Flags[flag.EnvVar].Set(flag.GetEnabled(), features.FlagSource_CENTRAL)
	}
	log.Info("After")
	for _, f := range features.Flags {
		log.Info("Flag: ", f.EnvVar(), ": ", f.Enabled())
	}
	log.Info("Feature flags configured")
}
