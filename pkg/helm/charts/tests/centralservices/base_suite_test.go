package centralservices

import (
	_ "embed"
	"io"
	"path"
	"strings"
	"testing"

	"github.com/stackrox/rox/image"
	metaUtil "github.com/stackrox/rox/pkg/helm/charts/testutils"
	helmUtil "github.com/stackrox/rox/pkg/helm/util"
	"github.com/stackrox/rox/pkg/set"
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
	//go:embed "testdata/all-values-explicit.yaml"
	allValuesExplicit string
	//go:embed "testdata/autogenerate-all.yaml"
	autogenerateAll string
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
	// We are in the process to remove these files. The support is limited to
	// upgrade process only. Exclude them for now.
	// TODO(ROX-16253): Remove PVC
	excludes := set.NewFrozenStringSet("01-central-11-pvc.yaml", "00-storage-class.yaml")

	for k, v := range rendered {
		if excludes.Contains(path.Base(k)) {
			s.Empty(v, "expected generated values file %s to be empty when specifying all generatable values", k)
			continue
		}
		s.NotEmptyf(v, "unexpected empty rendered YAML %s", k)
	}
}

func (s *baseSuite) TestAllGeneratableExplicit() {
	// Ensures that allValuesExplicit causes all templates to be rendered non-empty, including the one
	// containing generated values.

	_, rendered := s.LoadAndRender(allValuesExplicit)
	s.Require().NotEmpty(rendered)

	// We are in the process to remove these files. The support is limited to
	// upgrade process only. Exclude them for now.
	excludes := set.NewFrozenStringSet("01-central-11-pvc.yaml", "00-storage-class.yaml", "99-generated-values-secret.yaml")

	for k, v := range rendered {
		if excludes.Contains(path.Base(k)) {
			s.Empty(v, "expected generated values file %s to be empty when specifying all generatable values", k)
			continue
		}
		s.NotEmptyf(v, "unexpected empty rendered YAML %s", k)
	}
}

func (s *baseSuite) TestHelmVersionRequirements() {
	helmImage := image.GetDefaultImage()
	helmVals, err := chartutil.ReadValues([]byte(autogenerateAll))
	s.Require().NoError(err, "failed to parse values string %s", autogenerateAll)

	confAllowUnsupporteVersion := "allowUnsupportedHelmVersion: true"
	helmValsAllowingUnsupportedVersion, err := chartutil.ReadValues([]byte(confAllowUnsupporteVersion))
	s.Require().NoError(err, "failed to parse values string %s", confAllowUnsupporteVersion)
	helmValsAllowingUnsupportedVersion = chartutil.CoalesceTables(helmValsAllowingUnsupportedVersion, helmVals)

	// Retrieve template files from box.
	tpl, err := helmImage.GetCentralServicesChartTemplate()
	s.Require().NoError(err, "error retrieving chart template")
	ch, err := tpl.InstantiateAndLoad(metaUtil.MakeMetaValuesForTest(s.T()))
	s.Require().NoError(err, "error instantiating chart")

	effectiveInstallOpts := installOpts
	effectiveInstallOpts.HelmVersion = "v3.8.99"

	_, err = helmUtil.Render(ch, helmVals, effectiveInstallOpts)
	s.Require().Error(err, "successfully rendered chart")

	_, err = helmUtil.Render(ch, helmValsAllowingUnsupportedVersion, effectiveInstallOpts)
	s.Require().NoError(err, "failed to render chart while allowing unsupported Helm versions")

	effectiveInstallOpts.HelmVersion = "v3.9.0"
	_, err = helmUtil.Render(ch, helmVals, effectiveInstallOpts)
	s.Require().NoError(err, "failed to render chart")
}
