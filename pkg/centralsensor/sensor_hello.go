package centralsensor

import (
	"context"

	"github.com/grpc-ecosystem/go-grpc-middleware/util/metautils"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/sliceutils"
)

const (
	// SensorHelloMetadataKey is the key to indicate by both sensor and central that *sensor*, not *central*, will be
	// the first one to send a message on the stream. Sensor must not assume it can safely start by sending a message
	// unless it has received the header metadata from sensor, with the metadata value for that key set to true.
	SensorHelloMetadataKey = `Rox-Sensor-Hello`
)

// AppendSensorHelloInfoToOutgoingMetadata takes information from the given SensorHello message, and uses it to populate
// legacy sensor info outgoing metadata in the given context. It does *not* indicate that the client wants to send
// a SensorHello message.
func AppendSensorHelloInfoToOutgoingMetadata(ctx context.Context, hello *central.SensorHello) (context.Context, error) {
	ctx = appendCapsInfoToContext(ctx, set.NewSet(sliceutils.
		FromStringSlice[SensorCapability](hello.GetCapabilities()...)...))
	return appendSensorVersionInfoToContext(ctx, hello.GetSensorVersion())
}

// DeriveSensorHelloFromIncomingMetadata derives a SensorHello message from incoming sensor metadata in a legacy
// fashion (i.e., without an explicit message exchange).
// Note: Even when this function returns an error, it will still return a partially populated SensorHello message.
func DeriveSensorHelloFromIncomingMetadata(md metautils.NiceMD) (*central.SensorHello, error) {
	sensorHello := &central.SensorHello{}

	versionInfo, versionErr := deriveSensorVersionInfo(md)
	if versionInfo != nil {
		sensorHello.SensorVersion = versionInfo.MainVersion
	}

	sensorHello.Capabilities = sliceutils.StringSlice(extractCapsFromMD(md).AsSlice()...)
	return sensorHello, versionErr
}
