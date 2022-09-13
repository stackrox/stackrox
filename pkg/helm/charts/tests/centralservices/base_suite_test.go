package centralservices

import (
	"io"
	"path"
	"strings"
	"testing"

	"github.com/stackrox/rox/image"
	metaUtil "github.com/stackrox/rox/pkg/helm/charts/testutils"
	helmUtil "github.com/stackrox/rox/pkg/helm/util"
	"github.com/stretchr/testify/suite"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chartutil"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/yaml"
)

var (
	installOpts = helmUtil.Options{
		ReleaseOptions: chartutil.ReleaseOptions{
			Name:      "stackrox-central-services",
			Namespace: "stackrox",
			Revision:  1,
			IsInstall: true,
		},
		APIVersions: append(chartutil.DefaultVersionSet, "app.k8s.io/v1beta1/Application"),
	}

	// A values YAML that sets all generatable values explicitly, and causes all
	// objects to be generated.
	allValuesExplicit = `
licenseKey: "my license key"
env:
  platform: gke
  openshift: 4
  istio: true
  proxyConfig: "proxy config"
imagePullSecrets:
  username: myuser
  password: mypass
ca:
  cert: "ca cert pem"
  key: "ca key pem"
additionalCAs:
  ca.crt: |
    Extra CA certificate
central:
  adminPassword:
    htpasswd: "htpasswd file"
  jwtSigner:
    key: "jwt signing key"
  serviceTLS:
    cert: "central tls cert pem"
    key: "central tls key pem"
  defaultTLS:
    cert: "central default tls cert pem"
    key: "central default tls key pem"
  exposure:
    loadBalancer:
      enabled: true
  dbPassword:
    value: "central db password"
  dbServiceTLS:
    cert: "central db cert"
    key: "central db key"
  db:
    enabled: true
    password:
      value: "password"
    serviceTLS:
      cert: "central db tls cert pem"
      key: "central db tls key pem"
scanner:
  dbPassword:
    value: "db password"
  serviceTLS:
    cert: "scanner tls cert pem"
    key: "scanner tls key pem"
  dbServiceTLS:
    cert: "scanner-db tls cert pem"
    key: "scanner-db tls key pem"
enableOpenShiftMonitoring: true
system:
    enablePodSecurityPolicies: true
`
	autogenerateAll = `
licenseKey: "my license key"
additionalCAs:
  ca.crt: |
    Extra CA certificate
env:
  platform: gke
  openshift: 4
  istio: true
  proxyConfig: "proxy config"
imagePullSecrets:
  username: myuser
  password: mypass
central:
  defaultTLS:
    cert: "central default tls cert pem"
    key: "central default tls key pem"
  exposure:
    loadBalancer:
      enabled: true
  db:
    enabled: true
enableOpenShiftMonitoring: true
system:
    enablePodSecurityPolicies: true
`
)

type baseSuite struct {
	suite.Suite
}

func TestBase(t *testing.T) {
	suite.Run(t, new(baseSuite))
}

func (s *baseSuite) LoadAndRenderWithNamespace(namespace string, valStrs ...string) (*chart.Chart, map[string]string) {
	helmVals := make(chartutil.Values)
	helmImage := image.GetDefaultImage()
	for _, valStr := range valStrs {
		extraVals, err := chartutil.ReadValues([]byte(valStr))
		s.Require().NoError(err, "failed to parse values string %s", valStr)
		chartutil.CoalesceTables(extraVals, helmVals)
		helmVals = extraVals
	}

	// Retrieve template files from box.
	tpl, err := helmImage.GetCentralServicesChartTemplate()
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

func (s *baseSuite) TestAllGeneratableGenerated() {
	_, rendered := s.LoadAndRender(autogenerateAll)
	s.Require().NotEmpty(rendered)

	for k, v := range rendered {
		s.NotEmptyf(v, "unexpected empty rendered YAML %s", k)
	}
}

func (s *baseSuite) TestAllGeneratableExplicit() {
	// Ensures that allValuesExplicit causes all templates to be rendered non-empty, including the one
	// containing generated values.

	_, rendered := s.LoadAndRender(allValuesExplicit)
	s.Require().NotEmpty(rendered)

	for k, v := range rendered {
		if path.Base(k) == "99-generated-values-secret.yaml" {
			s.Empty(v, "expected generated values file to be empty when specifying all generatable values")
		} else {
			s.NotEmptyf(v, "unexpected empty rendered YAML %s", k)
		}
	}
}
