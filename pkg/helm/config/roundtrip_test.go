package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ghodss/yaml"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/image"
	"github.com/stackrox/rox/pkg/helm/charts"
	helmUtil "github.com/stackrox/rox/pkg/helm/util"
	flavorUtils "github.com/stackrox/rox/pkg/images/defaults/testutils"
	"github.com/stackrox/rox/pkg/jsonutil"
	"github.com/stackrox/rox/pkg/k8sutil"
	"github.com/stackrox/rox/pkg/maputil"
	"github.com/stretchr/testify/suite"
	"helm.sh/helm/v3/pkg/chartutil"
)

var (
	installOpts = helmUtil.Options{
		ReleaseOptions: chartutil.ReleaseOptions{
			Name:      "stackrox-secured-cluster-services",
			Namespace: "stackrox",
			Revision:  1,
			IsInstall: true,
		},
		APIVersions: chartutil.DefaultVersionSet,
	}
)

type helmConfigSuite struct {
	suite.Suite
}

func TestBase(t *testing.T) {
	suite.Run(t, new(helmConfigSuite))
}

func (h *helmConfigSuite) TestHelmConfigRoundTrip() {
	testDataFiles := []string{
		"simple.yaml",
	}

	for _, testDataFile := range testDataFiles {
		h.DoTestHelmConfigRoundTrip(filepath.Join("testdata/helm-chart-configurations", testDataFile))
	}
}

type HelmClusterConfig struct {
	ClusterName   string                 `json:"clusterName"`
	ClusterConfig map[string]interface{} `json:"clusterConfig"`
}

func (h *helmConfigSuite) toClusterConfig(helmCfg chartutil.Values) (*storage.CompleteClusterConfig, error) {
	helmImage := image.GetDefaultImage()
	// Instantiate central-services Helm chart.
	tpl, err := helmImage.GetSecuredClusterServicesChartTemplate()
	if err != nil {
		return nil, errors.Wrap(err, "retrieving chart template")
	}
	ch, err := tpl.InstantiateAndLoad(charts.GetMetaValuesForFlavor(flavorUtils.MakeImageFlavorForTest(h.T())))
	if err != nil {
		return nil, errors.Wrap(err, "instantiating chart")
	}
	rendered, err := helmUtil.Render(ch, helmCfg, installOpts)
	if err != nil {
		return nil, errors.Wrap(err, "rendering chart")
	}

	var clusterCfgResource map[string]interface{}
	for name, resource := range rendered {
		if strings.HasSuffix(name, "cluster-config.yaml") {
			u, err := k8sutil.UnstructuredFromYAML(resource)
			if err != nil {
				return nil, errors.Wrap(err, "extracting K8s resource from rendered chart")
			}
			clusterCfgResource = u.Object

		}
	}
	if clusterCfgResource == nil {
		return nil, errors.New("Secret 'helm-cluster-config' not found in rendered chart")
	}

	clusterCfgData, ok := clusterCfgResource["stringData"].(map[string]interface{})
	if !ok {
		return nil, errors.New("field 'stringData' not found in secret 'helm-cluster-config'")
	}
	clusterCfgContent, ok := clusterCfgData["config.yaml"].(string)
	if !ok {
		return nil, errors.New(`path 'stringData."config.yaml"' not found secret 'helm-cluster-config' or wrong type`)
	}

	var helmClusterCfg HelmClusterConfig
	err = yaml.Unmarshal([]byte(clusterCfgContent), &helmClusterCfg)
	if err != nil {
		return nil, errors.Wrap(err, "YAML unmarshalling")
	}

	clusterCfgJSON, err := json.Marshal(helmClusterCfg.ClusterConfig)
	if err != nil {
		return nil, errors.Wrap(err, "converting YAML to JSON")
	}

	var clusterCfg storage.CompleteClusterConfig

	err = jsonutil.JSONBytesToProto(clusterCfgJSON, &clusterCfg)
	if err != nil {
		return nil, errors.Wrap(err, "JSON unmarshalling")
	}

	return &clusterCfg, nil
}

func (h *helmConfigSuite) DoTestHelmConfigRoundTrip(helmValuesFile string) {
	// Read and parse Helm values.
	valBytes, err := os.ReadFile(helmValuesFile)
	h.Require().NoError(err, "failed to read Helm values from file %q", helmValuesFile)
	helmCfg, err := chartutil.ReadValues(valBytes)
	h.Require().NoError(err, "failed to parse Helm configuration in file %q", helmValuesFile)
	clusterName := helmCfg["clusterName"].(string)

	helmCfgOverwrites := map[string]interface{}{
		"imagePullSecrets": map[string]interface{}{
			"allowNone": true,
		},
		"createSecrets": false,
		"ca": map[string]interface{}{
			"cert": "DUMMY CA CERTIFICATE",
		},
	}

	enrichedHelmCfg := chartutil.CoalesceTables(helmCfgOverwrites, helmCfg)

	// Convert Helm config into a `CompleteClusterConfig`.
	clusterCfg, err := h.toClusterConfig(enrichedHelmCfg)
	h.Require().NoError(err, "transforming Helm configuration to cluster configuration")

	// Create a `Cluster` proto from  the `CompleteClusterConfig`.
	cluster := initClusterFromCompleteClusterConfig(clusterCfg)
	cluster.Name = clusterName

	// Derive a new Helm config from the `Cluster` proto.
	derivedHelmCfg, err := FromCluster(cluster, flavorUtils.MakeImageFlavorForTest(h.T()))
	h.Require().NoError(err, "deriving Helm config for cluster")

	diff := maputil.DiffGenericMap(helmCfg, derivedHelmCfg)
	if diff != nil {
		fmt.Fprintln(os.Stderr, "Helm config diff:")
		prettyDiff, err := json.MarshalIndent(diff, "", "  ")
		h.Require().NoError(err, "failed to serialize unstructured diff as JSON")
		fmt.Fprintf(os.Stderr, "%s\n", prettyDiff)
	}

	h.Require().Nil(diff, "Original Helm configuration and derived Helm configuration differ")
}

// This should probably be moved to a better place since this could be useful for Central when a new Cluster needs
// to be created, given a `CompleteClusterConfig`.
func initClusterFromCompleteClusterConfig(cfg *storage.CompleteClusterConfig) *storage.Cluster {
	cluster := storage.Cluster{
		Type:                       cfg.GetStaticConfig().GetType(),
		MainImage:                  cfg.GetStaticConfig().GetMainImage(),
		CollectorImage:             cfg.GetStaticConfig().GetCollectorImage(),
		CentralApiEndpoint:         cfg.GetStaticConfig().GetCentralApiEndpoint(),
		CollectionMethod:           cfg.GetStaticConfig().GetCollectionMethod(),
		AdmissionController:        cfg.GetStaticConfig().GetAdmissionController(),
		AdmissionControllerUpdates: cfg.GetStaticConfig().GetAdmissionControllerUpdates(),
		AdmissionControllerEvents:  cfg.GetStaticConfig().GetAdmissionControllerEvents(),
		DynamicConfig:              cfg.GetDynamicConfig(),
		TolerationsConfig:          cfg.GetStaticConfig().GetTolerationsConfig(),
		SlimCollector:              cfg.GetStaticConfig().GetSlimCollector(),
		HelmConfig:                 cfg,
		ManagedBy:                  storage.ManagerType_MANAGER_TYPE_HELM_CHART,
	}
	return &cluster
}
