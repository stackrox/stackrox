package env

import (
	"fmt"
	"os"
	"strings"
)

const (
	sensorEndpointEnvVar           = "ROX_SENSOR_ENDPOINT"
	legacyAdvertisedSensorEndpoint = "ROX_ADVERTISED_ENDPOINT"
	defaultSensorEndpoint          = "sensor.stackrox.svc:443"
)

// SensorEndpointSetting returns the effective in-cluster Sensor host:port endpoint by checking:
//  1. ROX_SENSOR_ENDPOINT (canonical)
//  2. ROX_ADVERTISED_ENDPOINT (legacy fallback)
//  3. sensor.{POD_NAMESPACE}.svc:443 (runtime derivation)
//  4. sensor.stackrox.svc:443 (hard default)
func SensorEndpointSetting() string {
	if endpoint := sensorEndpointFromEnv(sensorEndpointEnvVar); endpoint != "" {
		return endpoint
	}
	if endpoint := sensorEndpointFromEnv(legacyAdvertisedSensorEndpoint); endpoint != "" {
		return endpoint
	}
	if ns := Namespace.Setting(); ns != "" {
		return fmt.Sprintf("sensor.%s.svc:443", ns)
	}
	return defaultSensorEndpoint
}

func sensorEndpointFromEnv(envVar string) string {
	val, ok := os.LookupEnv(envVar)
	if !ok {
		return ""
	}
	val = strings.TrimSpace(val)
	if val == "" {
		return ""
	}
	return stripSchemePrefix(val)
}

func stripSchemePrefix(val string) string {
	for _, prefix := range []string{"https://", "http://"} {
		prev := val
		val = strings.TrimPrefix(val, prefix)
		if prev != val {
			break
		}
	}
	return val
}
