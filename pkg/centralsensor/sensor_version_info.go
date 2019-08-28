package centralsensor

import (
	"context"
	"encoding/json"

	"github.com/grpc-ecosystem/go-grpc-middleware/util/metautils"
	"github.com/pkg/errors"
)

// SensorVersionInfo contains information received from the sensor about its version information.
// Note that, due to compatibility reasons, there is no guarantee that all the fields here will be available,
// since the sensor might be too old to know how to send it.
// The only field that is guaranteed to be present is MainVersion, since that was the only field in the struct
// at the time of creation.
// Central MUST be able to handle this.
// DO NOT REMOVE ANY FIELDS FROM THIS STRUCT, OR CHANGE ANY JSON TAGS, AS THAT COULD BREAK
// BACKWARD COMPATIBILITY.
type SensorVersionInfo struct {
	MainVersion string `json:"mainVersion"`
}

// DeriveSensorVersionInfo derives the sensor version info from the given context.
// It is capable of understanding contexts passed in from any sensor, no matter how old.
// It should never return an error, except in the event of a programming error (or a rogue sensor...).
// It returns nil for sensors that are too old to send a version info.
func DeriveSensorVersionInfo(ctx context.Context) (*SensorVersionInfo, error) {
	md := metautils.ExtractIncoming(ctx)
	return deriveSensorVersionInfo(md)
}

func deriveSensorVersionInfo(md metautils.NiceMD) (*SensorVersionInfo, error) {
	var sensorVersionInfo SensorVersionInfo
	marshalledVersionInfo := md.Get(sensorVersionInfoKey)
	if marshalledVersionInfo == "" {
		return nil, nil
	}

	if err := json.Unmarshal([]byte(marshalledVersionInfo), &sensorVersionInfo); err != nil {
		return nil, errors.Wrap(err, "unmarshaling version info")
	}

	return &sensorVersionInfo, nil
}
