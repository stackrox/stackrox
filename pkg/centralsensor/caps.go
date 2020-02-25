package centralsensor

import (
	"context"
	"strings"

	"github.com/grpc-ecosystem/go-grpc-middleware/util/metautils"
	"google.golang.org/grpc/metadata"
)

// SensorCapability identifies a capability exposed by sensor.
type SensorCapability string

//go:generate genny -in=../set/generic.go -out=gen-caps-set.go -pkg centralsensor gen "KeyType=SensorCapability"

const (
	// CapsMetadataKey is the gRPC metadata key under which we store the supported capabilities.
	CapsMetadataKey = `Rox-Sensor-Capabilities`
)

// AppendCapsInfoToContext appends information about the supported capabilities to the context.
func AppendCapsInfoToContext(ctx context.Context, caps SensorCapabilitySet) context.Context {
	capsStrs := make([]string, 0, len(caps))
	for capability := range caps {
		capsStrs = append(capsStrs, string(capability))
	}
	return metadata.AppendToOutgoingContext(ctx, CapsMetadataKey, strings.Join(capsStrs, ","))
}

// ExtractCapsFromContext retrieves the set of sensor capabilities from the incoming context.
func ExtractCapsFromContext(ctx context.Context) SensorCapabilitySet {
	md := metautils.ExtractIncoming(ctx)
	capsStr := md.Get(CapsMetadataKey)

	result := NewSensorCapabilitySet()
	if capsStr != "" {
		capsStrs := strings.Split(capsStr, ",")
		for _, capsStr := range capsStrs {
			result.Add(SensorCapability(capsStr))
		}
	}

	return result
}
