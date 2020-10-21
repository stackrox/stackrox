package derivelocalvalues

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stackrox/rox/image"
	"github.com/stackrox/rox/pkg/charts"
	"github.com/stackrox/rox/pkg/helmutil"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/version"
	"github.com/stretchr/testify/suite"
	"gopkg.in/yaml.v3"
	"helm.sh/helm/v3/pkg/chartutil"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

const namespace string = "stackrox"

var (
	metaValues = &charts.MetaValues{
		Versions: version.Versions{
			ChartVersion:   "50.0.60-gac5d043be8",
			MainVersion:    "3.0.50.x-60-gac5d043be8",
			ScannerVersion: "2.5.0",
		},
		MainRegistry: "docker.io/stackrox",
	}

	installOpts = helmutil.Options{
		ReleaseOptions: chartutil.ReleaseOptions{
			Name:      "stackrox-central-services",
			Namespace: namespace,
			Revision:  1,
			IsInstall: true,
		},
		APIVersions: chartutil.DefaultVersionSet,
	}
)

type baseSuite struct {
	suite.Suite
}

func TestBase(t *testing.T) {
	suite.Run(t, new(baseSuite))
}

func (s *baseSuite) TestK8sResourcesRoundTrip() {
	testDataFiles := []string{
		"defaults.yaml",
		"loadBalancer.yaml",
		"hostPath.yaml",
		"offlineMode-internalRegistry-nodePortExposure-scannerDisabled.yaml",
	}

	for _, testDataFile := range testDataFiles {
		s.DoTestK8sResourcesRoundTrip(filepath.Join("testdata/k8s-resources", testDataFile))
	}
}

func (s *baseSuite) DoTestK8sResourcesRoundTrip(testDataFile string) {
	ctx := context.Background()

	// Retrieve persisted K8s resources.
	k8sLocal, err := newLocalK8sObjectDescriptionFromPath(testDataFile)
	s.Require().NoError(err, "Failed to retrieve persisted Kubernetes resources")
	initialK8sResources := k8sLocal.getAll(ctx)

	// Remove the special secret holding generated secrets.
	if secrets := initialK8sResources["secret"]; secrets != nil {
		for name := range secrets {
			if strings.HasPrefix(name, "stackrox-generated-") {
				delete(secrets, name)
			}
		}
	}

	// Derive local values.
	k8sResourceProvider := newK8sObjectDescription(k8sLocal)
	publicValues, privateValues, err := helmValuesForCentralServices(ctx, namespace, k8sResourceProvider)
	s.Require().NoError(err, "deriving local Helm values failed")

	// And patch the resulting configuration slightly.
	publicValues["imagePullSecrets"] = map[string]interface{}{
		"allowNone": true,
	}

	// Combine public and private values into a single configuration.
	helmVals := chartutil.CoalesceTables(yamlRoundTrip(s, publicValues), yamlRoundTrip(s, privateValues))

	// Instantiate central-services Helm chart.
	tpl, err := image.GetCentralServicesChartTemplate()
	s.Require().NoError(err, "error retrieving chart template")
	ch, err := tpl.InstantiateAndLoad(metaValues)
	s.Require().NoError(err, "error instantiating chart")

	// Render Helm chart using the retrieved configuration.
	rendered, err := helmutil.Render(ch, helmVals, installOpts)
	s.Require().NoError(err, "failed to render chart")

	// Concatenate freshly rendered resource definitions.
	allYamls := ""
	for name, resource := range rendered {
		if !strings.HasSuffix(name, ".yaml") {
			continue
		}
		if strings.HasSuffix(name, "99-generated-values-secret.yaml") {
			continue
		}
		allYamls = fmt.Sprintf("%s\n---\n%s", allYamls, resource)
	}

	// Parse rendered Kubernetes resources.
	renderedK8sResources, err := k8sResourcesFromString(allYamls)
	s.Require().NoError(err, "failed to parse rendered Kubernetes resources")

	// And diff it.
	diff := diffK8sResources(initialK8sResources, renderedK8sResources)

	if diff != nil {
		// The K8s resources differ, print a pretty diff.
		fmt.Fprintln(os.Stderr, "Kubernetes resource diff:")
		prettyDiff, err := json.MarshalIndent(diff, "", "  ")
		s.Require().NoError(err, "failed to serialize unstructured diff as JSON")
		fmt.Fprintf(os.Stderr, "%s\n", prettyDiff)
	}

	s.Require().Nil(diff, "Persisted and rendered Kubernetes resources differ")
}

func yamlRoundTrip(s *baseSuite, v map[string]interface{}) map[string]interface{} {
	marshalled, err := yaml.Marshal(v)
	s.Require().NoError(err, "error converting into YAML")
	unmarshalled := make(map[string]interface{})
	err = yaml.Unmarshal(marshalled, unmarshalled)
	s.Require().NoError(err, "error unmarshalling derived Helm values")
	return unmarshalled
}

func (s *baseSuite) TestsHelmValuesRoundTrip() {
	testDataFiles := []string{
		"defaults.yaml",
		"loadBalancer.yaml",
		"hostPath.yaml",
		"offlineMode-internalRegistry-nodePortExposure-scannerDisabled.yaml",
	}

	for _, testDataFile := range testDataFiles {
		s.DoTestsHelmValuesRoundTrip(filepath.Join("testdata/helm-values", testDataFile))
	}
}

func (s *baseSuite) DoTestsHelmValuesRoundTrip(helmValuesFile string) {
	// Read and parse Helm values.
	valStr, err := ioutil.ReadFile(helmValuesFile)
	s.Require().NoError(err, "failed to read Helm values from file %q", helmValuesFile)
	helmVals, err := chartutil.ReadValues([]byte(valStr))
	s.Require().NoError(err, "failed to parse Helm values in file %q", helmValuesFile)
	// Doing the roundtrip for all Helm values simplifies the diffing later on, since there might be
	// diffs due to type mismatches for numeric types (e.g. float64 vs int64) which would vanish when
	// unmarshalling for normalization purposes.
	helmVals = yamlRoundTrip(s, helmVals)

	effectiveHelmVals := chartutil.CoalesceTables(map[string]interface{}{
		"imagePullSecrets": map[string]interface{}{
			"allowNone": true,
		},
	}, helmVals)

	// Instantiate central-services Helm chart.
	tpl, err := image.GetCentralServicesChartTemplate()
	s.Require().NoError(err, "error retrieving chart template")
	ch, err := tpl.InstantiateAndLoad(metaValues)
	s.Require().NoError(err, "error instantiating chart")

	// Render Helm chart using the retrieved configuration.
	rendered, err := helmutil.Render(ch, effectiveHelmVals, installOpts)
	s.Require().NoError(err, "failed to render chart")

	// Concatenate freshly rendered resource definitions.
	allYamls := ""
	for name, resource := range rendered {
		if !strings.HasSuffix(name, ".yaml") {
			continue
		}
		if strings.HasSuffix(name, "99-generated-values-secret.yaml") {
			continue
		}
		allYamls = fmt.Sprintf("%s\n---\n%s", allYamls, resource)
	}

	ctx := context.Background()

	// Retrieve persisted K8s resources.
	k8sLocal, err := newLocalK8sObjectDescriptionFromString(allYamls)
	s.Require().NoError(err, "Failed to retrieve rendered Kubernetes resources")

	// Derive local values.
	k8sResourceProvider := newK8sObjectDescription(k8sLocal)
	publicValues, privateValues, err := helmValuesForCentralServices(ctx, namespace, k8sResourceProvider)
	s.Require().NoError(err, "deriving local Helm values failed")

	// Combine public and private values into a single configuration.
	newlyDerivedHelmVals := chartutil.CoalesceTables(yamlRoundTrip(s, publicValues), yamlRoundTrip(s, privateValues))

	// And diff it.
	diff := diffGenericMap(helmVals, newlyDerivedHelmVals)

	if diff != nil {
		// The K8s resources differ, print a pretty diff.
		fmt.Fprintln(os.Stderr, "Helm values diff:")
		prettyDiff, err := json.MarshalIndent(diff, "", "  ")
		s.Require().NoError(err, "failed to serialize unstructured diff as JSON")
		fmt.Fprintf(os.Stderr, "%s\n", prettyDiff)
	}

	s.Require().Nil(diff, "given Helm values and newly derived Helm values differ")
}

func extractResourceIdentifiers(src map[string]map[string]unstructured.Unstructured, dst map[string]set.StringSet) {
	for kind, resources := range src {
		if dst[kind] == nil {
			dst[kind] = set.NewStringSet()
		}
		resourceSet := dst[kind]
		for name := range resources {
			resourceSet.Add(name)
		}
		dst[kind] = resourceSet
	}
}

type k8sResourceDiff struct {
	A    *unstructured.Unstructured `json:"a"`
	B    *unstructured.Unstructured `json:"b"`
	Diff map[string]interface{}     `json:"diff"`
}

func diffK8sResources(a, b map[string]map[string]unstructured.Unstructured) map[string]map[string]k8sResourceDiff {
	allResources := make(map[string]set.StringSet)
	k8sResourceDiffs := make(map[string]map[string]k8sResourceDiff)

	extractResourceIdentifiers(a, allResources)
	extractResourceIdentifiers(b, allResources)

	for kind, names := range allResources {
		for name := range names {
			var resourceA *unstructured.Unstructured
			var resourceB *unstructured.Unstructured
			var diff map[string]interface{}

			if a[kind] != nil {
				res, ok := a[kind][name]
				if ok {
					resourceA = &res
				}
			}
			if b[kind] != nil {
				res, ok := b[kind][name]
				if ok {
					resourceB = &res
				}
			}

			if resourceA != nil && resourceB != nil {
				diff = diffUnstructured(*resourceA, *resourceB)
			}

			if (resourceA == nil) != (resourceB == nil) || diff != nil {
				k8sDiff := &k8sResourceDiff{A: resourceA, B: resourceB, Diff: diff}

				if k8sResourceDiffs[kind] == nil {
					k8sResourceDiffs[kind] = make(map[string]k8sResourceDiff)
				}
				k8sResourceDiffs[kind][name] = *k8sDiff
			}
		}
	}

	if len(k8sResourceDiffs) == 0 {
		return nil
	}
	return k8sResourceDiffs
}
