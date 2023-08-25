package gatherers

import (
	"context"
	"testing"

	"github.com/stackrox/rox/sensor/kubernetes/listener/resources"
	"github.com/stretchr/testify/suite"
	v1 "k8s.io/api/core/v1"
	k8sMetaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestClusterGatherer(t *testing.T) {
	suite.Run(t, new(ClusterGathererTestSuite))
}

type ClusterGathererTestSuite struct {
	suite.Suite
}

// There is not much business logic in the Cluster gatherer so this only tests that cluster gathering has all the right
// parts and doesn't panic
func (s *ClusterGathererTestSuite) TestGatherCluster() {
	node := &v1.Node{
		ObjectMeta: k8sMetaV1.ObjectMeta{
			Name: "NodeName",
		},
	}
	namespace := &v1.Namespace{
		ObjectMeta: k8sMetaV1.ObjectMeta{
			Name: "NamespaceName",
		},
	}
	gatherer := NewClusterGatherer(fake.NewSimpleClientset(node, namespace), resources.InitializeStore().Deployments())
	cluster := gatherer.Gather(context.Background())
	s.NotNil(cluster)
	s.Len(cluster.Nodes, 1)
	s.Len(cluster.Namespaces, 1)
}
