package securedclusterservices

import (
	"io"
	"path"
	"strings"
	"testing"

	"github.com/stackrox/rox/image"
	"github.com/stackrox/rox/pkg/helmutil"
	"github.com/stackrox/rox/pkg/version"
	"github.com/stretchr/testify/suite"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chartutil"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/yaml"
)

var (
	metaValues = map[string]interface{}{
		"Versions": version.Versions{
			ChartVersion:     "1.0.0",
			MainVersion:      "3.0.49.0",
			CollectorVersion: "1.2.3",
		},
		"MainRegistry":      "stackrox.io", // TODO: custom?
		"CollectorRegistry": "stackrox.io",
		"RenderMode":        "",
	}

	installOpts = helmutil.Options{
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
  type: OPENSHIFT_CLUSTER

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

config:
  collectionMethod: KERNEL_MODULE
  admissionControl:
    createService: true
    listenOnUpdates: true
    enableService: true
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
`
)

type baseSuite struct {
	suite.Suite
}

func TestBase(t *testing.T) {
	suite.Run(t, new(baseSuite))
}

func (s *baseSuite) LoadAndRenderWithNamespace(namespace string, valStrs ...string) (*chart.Chart, map[string]string) {
	var helmVals chartutil.Values
	for _, valStr := range valStrs {
		extraVals, err := chartutil.ReadValues([]byte(valStr))
		s.Require().NoError(err, "failed to parse values string %s", valStr)
		chartutil.CoalesceTables(extraVals, helmVals)
		helmVals = extraVals
	}

	// Retrieve template files from box.
	tpl, err := image.GetSecuredClusterServicesChartTemplate()
	s.Require().NoError(err, "error retrieving chart template")
	ch, err := tpl.InstantiateAndLoad(metaValues)
	s.Require().NoError(err, "error instantiating chart")

	effectiveInstallOpts := installOpts
	if namespace != "" {
		effectiveInstallOpts.ReleaseOptions.Namespace = namespace
	}
	rendered, err := helmutil.Render(ch, helmVals, effectiveInstallOpts)
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

	_, rendered := s.LoadAndRender(allValuesExplicit)
	s.Require().NotEmpty(rendered)

	for k, v := range rendered {
		if path.Base(k) == "additional-ca-sensor.yaml" {
			s.Empty(v, "expected additional CAs to be empty")
		} else {
			s.NotEmptyf(v, "unexpected empty rendered YAML %s", k)
		}
	}
}
