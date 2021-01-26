package renderer

import (
	"bytes"
	"fmt"
	"strconv"
	"testing"

	"github.com/stackrox/rox/image/sensor"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/helmutil"
	"github.com/stackrox/rox/pkg/istioutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"helm.sh/helm/v3/pkg/chartutil"
)

func TestRenderSensorHelm(t *testing.T) {
	// Verify at runtime that this won't panic
	envVars := make(map[string]string)
	for _, feature := range features.Flags {
		envVars[feature.EnvVar()] = strconv.FormatBool(feature.Enabled())
	}

	cases := map[string]struct {
		admissionController                       bool
		istioVersion                              string
		expectedAdmissionControllerRendered       bool
		expectedAdmissionControllerSecretRendered bool
		expectedHasDestinationRule                bool
	}{
		"withAdmissionControllerListenOnCreates": {
			admissionController:                       true,
			istioVersion:                              "",
			expectedAdmissionControllerRendered:       true,
			expectedAdmissionControllerSecretRendered: true,
			expectedHasDestinationRule:                false,
		},
		"withoutAdmissionControllerListenOnCreates": {
			admissionController:                       false,
			istioVersion:                              "",
			expectedAdmissionControllerRendered:       true,
			expectedAdmissionControllerSecretRendered: true,
			expectedHasDestinationRule:                false,
		},
		"onIstio": {
			admissionController:                       true,
			istioVersion:                              "1.5",
			expectedAdmissionControllerRendered:       true,
			expectedAdmissionControllerSecretRendered: true,
			expectedHasDestinationRule:                true,
		},
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
				"AdmissionControlListenOnUpdates":  false,
				"AdmissionControlListenOnEvents":   true,
				"DisableBypass":                    false,
				"TimeoutSeconds":                   3,
				"ScanInline":                       true,
				"AdmissionControllerEnabled":       c.admissionController,
				"AdmissionControlEnforceOnUpdates": c.admissionController,

				"EnvVars":      envVars,
				"FeatureFlags": make(map[string]string),

				"RenderAsLegacyChart": true,
			}

			certs := &sensor.Certs{Files: map[string][]byte{
				"secrets/ca.pem":                     []byte("abc"),
				"secrets/sensor-cert.pem":            []byte("def"),
				"secrets/sensor-key.pem":             []byte("ghi"),
				"secrets/collector-cert.pem":         []byte("jkl"),
				"secrets/collector-key.pem":          []byte("mno"),
				"secrets/admission-control-cert.pem": []byte("pqr"),
				"secrets/admission-control-key.pem":  []byte("stu"),
			}}

			opts := helmutil.Options{
				ReleaseOptions: chartutil.ReleaseOptions{
					Name:      "stackrox-secured-cluster-services",
					Namespace: "stackrox",
					IsInstall: true,
				},
			}

			if c.istioVersion != "" {
				istioAPIResources, err := istioutils.GetAPIResourcesByVersion(c.istioVersion)
				require.NoError(t, err)
				opts.APIVersions = helmutil.VersionSetFromResources(istioAPIResources...)
			}

			files, err := RenderSensor(fields, certs, opts)
			require.NoError(t, err)

			admissionControllerRendered := false
			admissionControllerSecretRendered := false
			hasDestinationRule := false

			for _, file := range files {
				if file.Name == "admission-controller.yaml" {
					fmt.Println(string(file.Content))
					admissionControllerRendered = true
				}
				if file.Name == "admission-controller-secret.yaml" {
					admissionControllerSecretRendered = true
				}
				if bytes.Contains(file.Content, []byte("DestinationRule")) {
					hasDestinationRule = true
				}
			}

			assert.Equal(t, c.expectedAdmissionControllerRendered, admissionControllerRendered, "incorrect bundle rendered")
			assert.Equal(t, c.expectedAdmissionControllerSecretRendered, admissionControllerSecretRendered, "incorrect bundle rendered")
			assert.Equal(t, c.expectedHasDestinationRule, hasDestinationRule, "unexpected presence/absence of destination rule")
		})
	}
}
