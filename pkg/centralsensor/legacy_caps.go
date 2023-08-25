package centralsensor

import (
	"context"
	"strings"

	"github.com/grpc-ecosystem/go-grpc-middleware/util/metautils"
	"github.com/stackrox/rox/pkg/set"
	"google.golang.org/grpc/metadata"
)

const (
	// legacyCapsMetadataKey is the gRPC metadata key under which we sent supported capabilities for legacy support.
	legacyCapsMetadataKey = `Rox-Sensor-Capabilities`
)

// appendCapsInfoToContext appends information about the supported capabilities to the context.
func appendCapsInfoToContext(ctx context.Context, caps set.Set[SensorCapability]) context.Context {
	capsStrs := make([]string, 0, len(caps))
	for capability := range caps {
		capsStrs = append(capsStrs, string(capability))
	}
	return metadata.AppendToOutgoingContext(ctx, legacyCapsMetadataKey, strings.Join(capsStrs, ","))
}

// extractCapsFromMD retrieves the set of sensor capabilities from the metadata set.
func extractCapsFromMD(md metautils.NiceMD) set.Set[SensorCapability] {
	capsStr := md.Get(legacyCapsMetadataKey)

	result := set.NewSet[SensorCapability]()
	if capsStr != "" {
		capsStrs := strings.Split(capsStr, ",")
		for _, capsStr := range capsStrs {
			result.Add(SensorCapability(capsStr))
		}
	}

	return result
}
