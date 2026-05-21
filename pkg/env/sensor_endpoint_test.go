package env

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSensorEndpointSetting(t *testing.T) {
	cases := map[string]struct {
		sensorEndpoint     string
		advertisedEndpoint string
		namespace          string
		setSensor          bool
		setAdvertised      bool
		setNamespace       bool
		expected           string
	}{
		"canonical endpoint": {
			sensorEndpoint: "sensor.custom.svc:8443",
			setSensor:      true,
			expected:       "sensor.custom.svc:8443",
		},
		"canonical strips https prefix": {
			sensorEndpoint: "https://sensor.custom.svc:8443",
			setSensor:      true,
			expected:       "sensor.custom.svc:8443",
		},
		"legacy advertised endpoint only": {
			advertisedEndpoint: "sensor.legacy.svc:443",
			setAdvertised:      true,
			expected:           "sensor.legacy.svc:443",
		},
		"canonical wins over legacy": {
			sensorEndpoint:     "sensor.canonical.svc:443",
			advertisedEndpoint: "sensor.legacy.svc:443",
			setSensor:          true,
			setAdvertised:      true,
			expected:           "sensor.canonical.svc:443",
		},
		"derive from namespace when unset": {
			namespace:    "rhacs",
			setNamespace: true,
			expected:     "sensor.rhacs.svc:443",
		},
		"empty canonical falls through to legacy": {
			sensorEndpoint:     "",
			advertisedEndpoint: "sensor.legacy.svc:443",
			setSensor:          true,
			setAdvertised:      true,
			expected:           "sensor.legacy.svc:443",
		},
		"namespace default stackrox when unset": {
			expected: defaultSensorEndpoint,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Setenv(sensorEndpointEnvVar, "")
			t.Setenv(legacyAdvertisedSensorEndpoint, "")
			t.Setenv(Namespace.EnvVar(), "")

			if tc.setSensor {
				t.Setenv(sensorEndpointEnvVar, tc.sensorEndpoint)
			}
			if tc.setAdvertised {
				t.Setenv(legacyAdvertisedSensorEndpoint, tc.advertisedEndpoint)
			}
			if tc.setNamespace {
				t.Setenv(Namespace.EnvVar(), tc.namespace)
			}

			assert.Equal(t, tc.expected, SensorEndpointSetting())
		})
	}
}
