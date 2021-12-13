package renderer

import (
	"fmt"
	"testing"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/testutils/envisolator"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	flavorUtils "github.com/stackrox/rox/pkg/images/testutils"
)

func getBaseConfig(t *testing.T) Config {
	return Config{
		ClusterType: storage.ClusterType_KUBERNETES_CLUSTER,
		K8sConfig: &K8sConfig{
			CommonConfig: CommonConfig{
				MainImage:    "stackrox/main:2.2.11.0-57-g392c0f5bed-dirty",
				ScannerImage: "stackrox.io/scanner:0.4.2",
			},
		},
		Flavor: flavorUtils.TestFlavor(t),
	}
}

func TestRender(t *testing.T) {
	suite.Run(t, new(renderSuite))
}

type renderSuite struct {
	suite.Suite
	envIsolator *envisolator.EnvIsolator
}

func (suite *renderSuite) SetupSuite() {
	suite.envIsolator = envisolator.NewEnvIsolator(suite.T())
	suite.envIsolator.Setenv("TEST_VERSIONS", "true")
}

func (suite *renderSuite) TeardownSuite() {
	suite.envIsolator.RestoreAll()
}

func (suite *renderSuite) testWithHostPath(t *testing.T, c Config) {
	c.HostPath = &HostPathPersistence{
		HostPath: "/var/lib/stackrox",
	}
	_, err := Render(c)
	assert.NoError(t, err)

	c.HostPath = &HostPathPersistence{
		HostPath:          "/var/lib/stackrox",
		NodeSelectorKey:   "key",
		NodeSelectorValue: "value",
	}
	_, err = Render(c)
	assert.NoError(t, err)
}

func (suite *renderSuite) testWithPV(t *testing.T, c Config) {
	c.External = &ExternalPersistence{
		Name: "name",
	}
	_, err := Render(c)
	assert.NoError(t, err)

	c.External = &ExternalPersistence{
		Name:         "name",
		StorageClass: "storageClass",
	}
	_, err = Render(c)
	assert.NoError(t, err)
}

func (suite *renderSuite) testWithLoadBalancers(t *testing.T, c Config) {
	c.K8sConfig.LoadBalancerType = v1.LoadBalancerType_NODE_PORT
	_, err := Render(c)
	assert.NoError(t, err)

	c.K8sConfig.LoadBalancerType = v1.LoadBalancerType_LOAD_BALANCER
	_, err = Render(c)
	assert.NoError(t, err)
}

func (suite *renderSuite) TestRenderMultiple() {
	for _, orch := range []storage.ClusterType{storage.ClusterType_KUBERNETES_CLUSTER, storage.ClusterType_OPENSHIFT_CLUSTER, storage.ClusterType_OPENSHIFT4_CLUSTER} {
		for _, format := range []v1.DeploymentFormat{v1.DeploymentFormat_KUBECTL, v1.DeploymentFormat_HELM} {
			suite.T().Run(fmt.Sprintf("%s-%s", orch, format), func(t *testing.T) {
				conf := getBaseConfig(t)
				conf.ClusterType = orch
				conf.K8sConfig.DeploymentFormat = format

				suite.testWithHostPath(t, conf)
				suite.testWithPV(t, conf)
				suite.testWithLoadBalancers(t, conf)
			})
		}
	}
}

func (suite *renderSuite) TestRenderWithBadImage() {
	conf := getBaseConfig(suite.T())
	conf.K8sConfig.ScannerImage = "invalid-image#!@$"
	_, err := Render(conf)
	suite.Error(err)
}
