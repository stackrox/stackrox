package renderer

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"sort"
	"strconv"
	"testing"

	"github.com/stackrox/stackrox/image/sensor"
	"github.com/stackrox/stackrox/pkg/features"
	"github.com/stackrox/stackrox/pkg/helm/charts"
	helmUtil "github.com/stackrox/stackrox/pkg/helm/util"
	"github.com/stackrox/stackrox/pkg/images/defaults"
	"github.com/stackrox/stackrox/pkg/istioutils"
	"github.com/stackrox/stackrox/pkg/version/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
	"helm.sh/helm/v3/pkg/chartutil"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

var certs = &sensor.Certs{
	Files: map[string][]byte{
		"secrets/ca.pem":                     []byte("abc"),
		"secrets/sensor-cert.pem":            []byte("def"),
		"secrets/sensor-key.pem":             []byte("ghi"),
		"secrets/collector-cert.pem":         []byte("jkl"),
		"secrets/collector-key.pem":          []byte("mno"),
		"secrets/admission-control-cert.pem": []byte("pqr"),
		"secrets/admission-control-key.pem":  []byte("stu"),
	},
}

func getDefaultMetaValues(t *testing.T) *charts.MetaValues {
	return &charts.MetaValues{
		MainRegistry: "stackrox.io",
		ImageRemote:  "main",
		ImageTag:     "3.0.41.x-92-g9e8a347ffe",

		PublicEndpoint: "central.stackrox:443",

		ClusterName: "remote",

		AdvertisedEndpoint: "sensor.stackrox:443",

		CollectorRegistry:        "collector.stackrox.io",
		CollectorFullImageRemote: "collector",
		CollectorSlimImageRemote: "collector",
		CollectorFullImageTag:    "3.0.11-latest",
		CollectorSlimImageTag:    "3.0.11-slim",

		CollectionMethod: "EBPF",

		ClusterType: "KUBERNETES_CLUSTER",

		TolerationsEnabled: false,

		CreateUpgraderSA: true,

		AdmissionController:              false,
		AdmissionControlListenOnUpdates:  false,
		AdmissionControlListenOnEvents:   true,
		DisableBypass:                    false,
		TimeoutSeconds:                   3,
		ScanInline:                       true,
		AdmissionControllerEnabled:       false,
		AdmissionControlEnforceOnUpdates: false,

		EnvVars:      nil,
		FeatureFlags: make(map[string]interface{}),

		Versions: testutils.GetExampleVersion(t),

		ChartRepo: defaults.ChartRepo{URL: "https://mock.stackrox.io/mock-charts"},

		KubectlOutput: true,
	}
}
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
			fields := getDefaultMetaValues(t)
			fields.AdmissionController = c.admissionController
			fields.AdmissionControllerEnabled = c.admissionController
			fields.AdmissionControlEnforceOnUpdates = c.admissionController
			fields.EnvVars = envVars
			opts := helmUtil.Options{
				ReleaseOptions: chartutil.ReleaseOptions{
					Name:      "stackrox-secured-cluster-services",
					Namespace: "stackrox",
					IsInstall: true,
				},
			}

			if c.istioVersion != "" {
				istioAPIResources, err := istioutils.GetAPIResourcesByVersion(c.istioVersion)
				require.NoError(t, err)
				opts.APIVersions = helmUtil.VersionSetFromResources(istioAPIResources...)
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

func TestRenderSensorTLSSecretsOnly(t *testing.T) {
	fields := getDefaultMetaValues(t)
	fields.CertsOnly = true

	manifestsBytes, err := RenderSensorTLSSecretsOnly(*fields, certs)
	require.NoError(t, err)
	d := yaml.NewDecoder(bytes.NewReader(manifestsBytes))

	expectedSecrets := []string{"admission-control-tls", "collector-tls", "sensor-tls"}
	var encounteredSecretNames []string
	for {
		spec := make(map[string]interface{})
		err := d.Decode(spec)
		if errors.Is(err, io.EOF) {
			break
		} else {
			require.NoError(t, err)
		}

		secret := &corev1.Secret{}
		err = runtime.DefaultUnstructuredConverter.FromUnstructured(spec, secret)
		require.NoError(t, err)

		encounteredSecretNames = append(encounteredSecretNames, secret.Name)
	}

	sort.Strings(encounteredSecretNames)
	assert.Equal(t, expectedSecrets, encounteredSecretNames)
}
