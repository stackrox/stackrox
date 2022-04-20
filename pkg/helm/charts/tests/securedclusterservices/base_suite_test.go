package securedclusterservices

import (
	"io"
	"path"
	"strings"
	"testing"

	"github.com/stackrox/rox/image"
	"github.com/stackrox/rox/pkg/features"
	metaUtil "github.com/stackrox/rox/pkg/helm/charts/testutils"
	helmUtil "github.com/stackrox/rox/pkg/helm/util"
	"github.com/stackrox/rox/pkg/testutils/envisolator"
	"github.com/stretchr/testify/suite"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chartutil"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/yaml"
)

var (
	installOpts = helmUtil.Options{
		ReleaseOptions: chartutil.ReleaseOptions{
			Name:      "stackrox-secured-cluster-services",
			Namespace: "stackrox",
			Revision:  1,
			IsInstall: true,
		},
		APIVersions: append(chartutil.DefaultVersionSet, "app.k8s.io/v1beta1/Application"),
	}

	// A values YAML that sets all generatable values explicitly, and causes all
	// objects to be generated.
	allValuesExplicit = `
cluster:
  name: foo
  type: OPENSHIFT4_CLUSTER

ca:
  cert: "DUMMY CA CERTIFICATE"

imagePullSecrets:
  username: myuser
  password: mypass
  
endpoint:
  central: "central.stackrox:443"
  advertised: "central-advertised.stackrox:443"

image:
  repository:
    main: "custom-main-repo"
    collector: "custom-collector-repo"
  registry:
    main: "custom-main-registry"
    collector: "custom-collector-registry"

envVars:
- name: CUSTOM_ENV_VAR
  value: FOO

sensor:
  serviceTLS:
    cert: "DUMMY SENSOR CERT"
    key: "DUMMY SENSOR KEY"

collector:
  serviceTLS:
    cert: "DUMMY COLLECTOR CERT"
    key: "DUMMY COLLECTOR KEY"

admissionControl:
  serviceTLS:
    cert: "DUMMY ADMISSION CONTROL CERT"
    key: "DUMMY ADMISSION CONTROL KEY"

config:
  collectionMethod: KERNEL_MODULE
  admissionControl:
    listenOnCreates: true
    listenOnUpdates: true
    enforceOnCreates: true
    enforceOnUpdates: true
    scanInline: true
    disableBypass: true
    timeout: 4
  disableTaintTolerations: true
  createUpgraderServiceAccount: true
  createSecrets: true
  offlineMode: true
  slimCollector: true
  exposeMonitoring: true

enableOpenShiftMonitoring: true
scanner:
  disable: false
`
)

type baseSuite struct {
	suite.Suite
	envIsolator *envisolator.EnvIsolator
}

func TestBase(t *testing.T) {
	suite.Run(t, new(baseSuite))
}

func (s *baseSuite) SetupTest() {
	s.envIsolator = envisolator.NewEnvIsolator(s.T())
}

func (s *baseSuite) TearDownTest() {
	s.envIsolator.RestoreAll()
}

func (s *baseSuite) LoadAndRenderWithNamespace(namespace string, valStrs ...string) (*chart.Chart, map[string]string) {
	var helmVals chartutil.Values
	helmImage := image.GetDefaultImage()
	for _, valStr := range valStrs {
		extraVals, err := chartutil.ReadValues([]byte(valStr))
		s.Require().NoError(err, "failed to parse values string %s", valStr)
		chartutil.CoalesceTables(extraVals, helmVals)
		helmVals = extraVals
	}

	// Retrieve template files from box.
	tpl, err := helmImage.GetSecuredClusterServicesChartTemplate()
	s.Require().NoError(err, "error retrieving chart template")
	ch, err := tpl.InstantiateAndLoad(metaUtil.MakeMetaValuesForTest(s.T()))
	s.Require().NoError(err, "error instantiating chart")

	effectiveInstallOpts := installOpts
	if namespace != "" {
		effectiveInstallOpts.ReleaseOptions.Namespace = namespace
	}
	rendered, err := helmUtil.Render(ch, helmVals, effectiveInstallOpts)
	s.Require().NoError(err, "failed to render chart")

	for k, v := range rendered {
		rendered[k] = strings.TrimSpace(v)
	}

	return ch, rendered
}

func (s *baseSuite) LoadAndRender(valStrs ...string) (*chart.Chart, map[string]string) {
	return s.LoadAndRenderWithNamespace("", valStrs...)
}

func (s *baseSuite) ParseObjects(objYAMLs map[string]string) []unstructured.Unstructured {
	var objs []unstructured.Unstructured
	for fileName, yamlStr := range objYAMLs {
		if path.Ext(fileName) != ".yaml" {
			continue
		}
		dec := yaml.NewYAMLToJSONDecoder(strings.NewReader(yamlStr))
		for {
			var obj unstructured.Unstructured
			if err := dec.Decode(&obj.Object); err != nil {
				if err == io.EOF {
					break
				}
				s.Require().NoError(err, "could not unmarshal YAML from rendered file %s", fileName)
			}
			objs = append(objs, obj)
		}
	}
	return objs
}

func (s *baseSuite) TestAllGeneratableExplicit() {
	// Ensures that allValuesExplicit causes all templates to be rendered non-empty, including the one
	// containing generated values.

	s.envIsolator.Setenv(features.LocalImageScanning.EnvVar(), "true")

	_, rendered := s.LoadAndRender(allValuesExplicit)
	s.Require().NotEmpty(rendered)

	for k, v := range rendered {
		if path.Base(k) == "additional-ca-sensor.yaml" {
			s.Empty(v, "expected additional CAs to be empty")
		} else {
			s.NotEmptyf(v, "unexpected empty rendered YAML %s", k)
		}
	}

	// Verify custom environment variable
	expectedEnvVar := map[string]interface{}{
		"name":  "CUSTOM_ENV_VAR",
		"value": "FOO",
	}

	objs := s.ParseObjects(rendered)
	for i := range objs {
		obj := objs[i]
		if obj.GetKind() != "Deployment" && obj.GetKind() != "DaemonSet" {
			continue
		}
		containers, _, err := unstructured.NestedSlice(obj.Object, "spec", "template", "spec", "containers")
		s.NoError(err)
		s.NotEmpty(containers)
		for _, container := range containers {
			containerObj := container.(map[string]interface{})
			envVars, _, err := unstructured.NestedSlice(containerObj, "env")
			s.NoError(err)
			s.Contains(envVars, expectedEnvVar)
		}
	}
}
