package centralsensor

import (
	"context"
	"encoding/json"

	metautils "github.com/grpc-ecosystem/go-grpc-middleware/v2/metadata"
	"github.com/pkg/errors"
	"google.golang.org/grpc/metadata"
)

const (
	// legacySensorVersionInfoKey is the key in the gRPC metadata under which sensor sends central its version info.
	legacySensorVersionInfoKey = "rox-sensor-version-info"
)

// sensorVersionInfo contains information received from the sensor about its version information.
// Note that, due to compatibility reasons, there is no guarantee that all the fields here will be available,
// since the sensor might be too old to know how to send it.
// The only field that is guaranteed to be present is MainVersion, since that was the only field in the struct
// at the time of creation.
// Central MUST be able to handle this.
// DO NOT REMOVE ANY FIELDS FROM THIS STRUCT, OR CHANGE ANY JSON TAGS, AS THAT COULD BREAK
// BACKWARD COMPATIBILITY.
type sensorVersionInfo struct {
	MainVersion string `json:"mainVersion"`
}

func appendSensorVersionInfoToContext(ctx context.Context, version string) (context.Context, error) {
	if version == "" {
		return ctx, nil
	}

	versionInfo := sensorVersionInfo{MainVersion: version}
	marshalled, err := json.Marshal(versionInfo)
	if err != nil {
		return nil, errors.Wrap(err, "couldn't marshal version info")
	}
	return metadata.AppendToOutgoingContext(ctx,
		legacySensorVersionInfoKey, string(marshalled),
	), nil
}

func deriveSensorVersionInfo(md metautils.MD) (*sensorVersionInfo, error) {
	var sensorVersionInfo sensorVersionInfo
	marshalledVersionInfo := md.Get(legacySensorVersionInfoKey)
	if marshalledVersionInfo == "" {
		return nil, nil
	}

	if err := json.Unmarshal([]byte(marshalledVersionInfo), &sensorVersionInfo); err != nil {
		return nil, errors.Wrap(err, "unmarshaling version info")
	}

	return &sensorVersionInfo, nil
}
