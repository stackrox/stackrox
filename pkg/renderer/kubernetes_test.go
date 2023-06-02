package renderer

import (
	"fmt"
	"testing"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	helmUtil "github.com/stackrox/rox/pkg/helm/util"
	"github.com/stackrox/rox/pkg/images/defaults"
	flavorUtils "github.com/stackrox/rox/pkg/images/defaults/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"helm.sh/helm/v3/pkg/chartutil"
)

func getBaseConfig() Config {
	return Config{
		ClusterType: storage.ClusterType_KUBERNETES_CLUSTER,
		K8sConfig: &K8sConfig{
			CommonConfig: CommonConfig{
				MainImage:    "stackrox/main:2.2.11.0-57-g392c0f5bed-dirty",
				ScannerImage: "stackrox.io/scanner:0.4.2",
			},
		},
	}
}

func TestRender(t *testing.T) {
	suite.Run(t, new(renderSuite))
}

type renderSuite struct {
	suite.Suite
	testFlavor defaults.ImageFlavor
}

func (suite *renderSuite) SetupSuite() {
	suite.T().Setenv("TEST_VERSIONS", "true")
	suite.testFlavor = flavorUtils.MakeImageFlavorForTest(suite.T())
}

func (suite *renderSuite) testWithHostPath(t *testing.T, c Config) {
	c.HostPath = &HostPathPersistence{
		Central: &HostPathPersistenceInstance{
			HostPath: "/var/lib/stackrox",
		},
	}
	_, err := Render(c, suite.testFlavor)
	assert.NoError(t, err)

	c.HostPath = &HostPathPersistence{
		Central: &HostPathPersistenceInstance{
			HostPath:          "/var/lib/stackrox",
			NodeSelectorKey:   "key",
			NodeSelectorValue: "value",
		},
	}
	_, err = Render(c, suite.testFlavor)
	assert.NoError(t, err)
}

func (suite *renderSuite) testWithPV(t *testing.T, c Config) {
	c.External = &ExternalPersistence{
		Central: &ExternalPersistenceInstance{
			Name: "name",
		},
	}
	_, err := Render(c, suite.testFlavor)
	assert.NoError(t, err)

	c.External = &ExternalPersistence{
		Central: &ExternalPersistenceInstance{
			Name:         "name",
			StorageClass: "storageClass",
		},
	}
	_, err = Render(c, suite.testFlavor)
	assert.NoError(t, err)
}

func (suite *renderSuite) testWithLoadBalancers(t *testing.T, c Config) {
	c.K8sConfig.LoadBalancerType = v1.LoadBalancerType_NODE_PORT
	_, err := Render(c, suite.testFlavor)
	assert.NoError(t, err)

	c.K8sConfig.LoadBalancerType = v1.LoadBalancerType_LOAD_BALANCER
	_, err = Render(c, suite.testFlavor)
	assert.NoError(t, err)
}

func (suite *renderSuite) TestRenderMultiple() {
	upgradeOpts := helmUtil.Options{
		ReleaseOptions: chartutil.ReleaseOptions{
			Name:      "stackrox-secured-cluster-services",
			Namespace: "stackrox",
			IsUpgrade: true,
		},
	}
	for _, orch := range []storage.ClusterType{storage.ClusterType_KUBERNETES_CLUSTER, storage.ClusterType_OPENSHIFT_CLUSTER, storage.ClusterType_OPENSHIFT4_CLUSTER} {
		for _, format := range []v1.DeploymentFormat{v1.DeploymentFormat_KUBECTL, v1.DeploymentFormat_HELM} {
			suite.T().Run(fmt.Sprintf("%s-%s", orch, format), func(t *testing.T) {
				conf := getBaseConfig()
				conf.ClusterType = orch
				conf.K8sConfig.DeploymentFormat = format
				// We do not need these tests anymore because this is not used in upgrade.
				// But still keep it to see if it helps to find other issues.
				conf.RenderOpts = &upgradeOpts

				suite.testWithHostPath(t, conf)
				suite.testWithPV(t, conf)
				suite.testWithLoadBalancers(t, conf)
			})
		}
	}
}

func (suite *renderSuite) TestRenderWithBadImage() {
	conf := getBaseConfig()
	conf.K8sConfig.ScannerImage = "invalid-image#!@$"
	_, err := Render(conf, suite.testFlavor)
	suite.Error(err)
}
