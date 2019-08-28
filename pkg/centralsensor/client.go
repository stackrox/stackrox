package centralsensor

import (
	"context"
	"encoding/json"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/version"
	"google.golang.org/grpc/metadata"
)

// AppendSensorVersionInfoToContext appends the sensor version info to the given context.
func AppendSensorVersionInfoToContext(ctx context.Context) (context.Context, error) {
	return appendSensorVersionInfoToContext(ctx, version.GetMainVersion())
}

func appendSensorVersionInfoToContext(ctx context.Context, version string) (context.Context, error) {
	versionInfo := SensorVersionInfo{MainVersion: version}
	marshalled, err := json.Marshal(versionInfo)
	if err != nil {
		return nil, errors.Wrap(err, "couldn't marshal version info")
	}
	return metadata.AppendToOutgoingContext(ctx,
		sensorVersionInfoKey, string(marshalled),
	), nil
}
