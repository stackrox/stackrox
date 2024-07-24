package telemetry

import (
	"context"
	"fmt"

	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/telemetry/phonehome"
)

// Gather a `Feature "<name>"` user property for each feature flag.
var Gather phonehome.GatherFunc = func(ctx context.Context) (map[string]any, error) {
	props := make(map[string]any)

	for _, flag := range features.Flags {
		props[fmt.Sprintf("Feature %s", flag.EnvVar())] = flag.Enabled()
	}

	return props, nil
}
