package renderer

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"sort"
	"strconv"
	"testing"

	"github.com/stackrox/rox/image/sensor"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/helm/charts"
	helmUtil "github.com/stackrox/rox/pkg/helm/util"
	"github.com/stackrox/rox/pkg/images/defaults"
	"github.com/stackrox/rox/pkg/istioutils"
	"github.com/stackrox/rox/pkg/version/testutils"
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

		ScannerSlimImageRemote: "scanner",
		ScannerImageTag:        "3.0.11-slim",

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

		ChartRepo: defaults.ChartRepo{URL: "https://mock.stackrox.io/mock-charts", IconURL: "https://mock.icon/ic.png"},

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

func TestRenderSensorTLSSensorOnly_NoErrorOnMissingImageData(t *testing.T) {
	fields := getDefaultMetaValues(t)
	fields.CertsOnly = true
	// (ROX-16212) Should not fail when meta-values don't set ImageTag (e.g. when running with Operator installation)
	// ImageTag isn't used to render TLS secrets, therefore it shouldn't result in RenderSensorTLSSecretsOnly returning an error
	fields.ImageTag = ""
	renderedManifests, err := RenderSensorTLSSecretsOnly(*fields, certs)
	require.NoError(t, err)

	// Image tag should not be seen in any of the yaml files
	rawYamlString := string(renderedManifests)
	assert.Contains(t, rawYamlString, "sensor-tls")
	assert.Contains(t, rawYamlString, "collector-tls")
	assert.Contains(t, rawYamlString, "admission-control-tls")
	assert.NotContains(t, rawYamlString, "should-never-see-this")
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
		}
		require.NoError(t, err)

		secret := &corev1.Secret{}
		err = runtime.DefaultUnstructuredConverter.FromUnstructured(spec, secret)
		require.NoError(t, err)

		encounteredSecretNames = append(encounteredSecretNames, secret.Name)
	}

	sort.Strings(encounteredSecretNames)
	assert.Equal(t, expectedSecrets, encounteredSecretNames)
}
