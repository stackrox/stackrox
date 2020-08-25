package renderer

import (
	"bytes"
	"strconv"
	"testing"

	"github.com/stackrox/rox/image/sensor"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/helmutil"
	"github.com/stackrox/rox/pkg/istioutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRenderSensorHelm(t *testing.T) {
	// Verify at runtime that this won't panic
	envVars := make(map[string]string)
	for _, feature := range features.Flags {
		envVars[feature.EnvVar()] = strconv.FormatBool(feature.Enabled())
	}

	cases := map[string]struct {
		admissionController bool
		istioVersion        string
	}{
		"withAdmissionController":    {admissionController: true},
		"withoutAdmissionController": {admissionController: false},
		"onIstio":                    {istioVersion: "1.5"},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {

			fields := map[string]interface{}{
				"ImageRegistry": "stackrox.io",
				"ImageRemote":   "main",
				"ImageTag":      "3.0.41.x-92-g9e8a347ffe",

				"PublicEndpoint": "central.stackrox:443",

				"ClusterName": "remote",

				"AdvertisedEndpoint": "sensor.stackrox:443",

				"CollectorRegistry":    "collector.stackrox.io",
				"CollectorImageRemote": "collector",
				"CollectorImageTag":    "3.0.11-latest",

				"CollectionMethod": "EBPF",

				"ClusterType": "KUBERNETES_CLUSTER",

				"TolerationsEnabled": false,

				"CreateUpgraderSA": true,

				"AdmissionController":              c.admissionController,
				"AdmissionControlListenOnUpdates":  c.admissionController,
				"DisableBypass":                    false,
				"TimeoutSeconds":                   3,
				"ScanInline":                       true,
				"AdmissionControllerEnabled":       c.admissionController,
				"AdmissionControlEnforceOnUpdates": c.admissionController,

				"EnvVars": envVars,
			}

			certs := &sensor.Certs{Files: map[string][]byte{
				"ca.pem":                     []byte("abc"),
				"sensor-cert.pem":            []byte("def"),
				"sensor-key.pem":             []byte("ghi"),
				"collector-cert.pem":         []byte("jkl"),
				"collector-key.pem":          []byte("mno"),
				"admission-control-cert.pem": []byte("pqr"),
				"admission-control-key.pem":  []byte("stu"),
			}}

			var opts helmutil.Options

			if c.istioVersion != "" {
				istioAPIResources, err := istioutils.GetAPIResourcesByVersion(c.istioVersion)
				require.NoError(t, err)
				opts.APIVersions = helmutil.VersionSetFromResources(istioAPIResources...)
			}

			files, err := RenderSensor(fields, certs, opts)

			admissionControllerRendered := false
			admissionControllerSecretRendered := false
			hasDestinationRule := false

			for _, file := range files {
				if file.Name == "admission-controller.yaml" {
					admissionControllerRendered = true
				}
				if file.Name == "admission-controller-secret.yaml" {
					admissionControllerSecretRendered = true
				}
				if bytes.Contains(file.Content, []byte("DestinationRule")) {
					hasDestinationRule = true
				}
			}

			require.NoError(t, err)

			assert.Equal(t, c.admissionController, admissionControllerRendered, "incorrect bundle rendered")
			assert.Equal(t, c.admissionController, admissionControllerSecretRendered, "incorrect bundle rendered")
			assert.Equal(t, c.istioVersion != "", hasDestinationRule, "unexpected presence/absence of destination rule")
		})
	}
}
