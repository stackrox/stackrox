package centralsensor

import (
	"context"
	"encoding/json"

	"github.com/pkg/errors"
	"google.golang.org/grpc/metadata"
)

const (
	// legacySensorVersionInfoKey is the key in the gRPC metadata under which sensor sends central its version info.
	legacySensorVersionInfoKey = "rox-sensor-version-info"
)

// sensorVersionInfo is still encoded into outgoing Sensor metadata for older Centrals.
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
