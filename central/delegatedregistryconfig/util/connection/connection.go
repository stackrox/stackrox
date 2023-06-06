package connection

import (
	"github.com/stackrox/rox/central/sensor/service/connection"
	"github.com/stackrox/rox/pkg/centralsensor"
)

// ValidForDelegation returns true if the connection is valid for
// delegating scan requests and syncing related resources (such as
// image integrations).
func ValidForDelegation(conn connection.SensorConnection) bool {
	return conn != nil && conn.HasCapability(centralsensor.DelegatedRegistryCap)
}
