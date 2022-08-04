package centralsensor

import (
	"context"
	"strings"

	"github.com/grpc-ecosystem/go-grpc-middleware/util/metautils"
	"google.golang.org/grpc/metadata"
)

const (
	// legacyCapsMetadataKey is the gRPC metadata key under which we sent supported capabilities for legacy support.
	legacyCapsMetadataKey = `Rox-Sensor-Capabilities`
)

// appendCapsInfoToContext appends information about the supported capabilities to the context.
func appendCapsInfoToContext(ctx context.Context, caps SensorCapabilitySet) context.Context {
	capsStrs := make([]string, 0, len(caps))
	for capability := range caps {
		capsStrs = append(capsStrs, string(capability))
	}
	return metadata.AppendToOutgoingContext(ctx, legacyCapsMetadataKey, strings.Join(capsStrs, ","))
}

// extractCapsFromMD retrieves the set of sensor capabilities from the metadata set.
func extractCapsFromMD(md metautils.NiceMD) SensorCapabilitySet {
	capsStr := md.Get(legacyCapsMetadataKey)

	result := NewSensorCapabilitySet()
	if capsStr != "" {
		capsStrs := strings.Split(capsStr, ",")
		for _, capsStr := range capsStrs {
			result.Add(SensorCapability(capsStr))
		}
	}

	return result
}
