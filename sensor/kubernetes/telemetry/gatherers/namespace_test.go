package gatherers

import (
	"context"
	"testing"

	"github.com/stackrox/rox/pkg/telemetry/data"
	"github.com/stackrox/rox/sensor/kubernetes/listener/resources"
	"github.com/stretchr/testify/suite"
	v1 "k8s.io/api/core/v1"
	k8sMetaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestNamespaceGatherer(t *testing.T) {
	suite.Run(t, new(NamespaceGathererTestSuite))
}

type NamespaceGathererTestSuite struct {
	suite.Suite
}

// There is not much business logic in the namespace gatherer so this only tests that unknown namespace names are
// removed
func (s *NamespaceGathererTestSuite) TestGatherNamespaces() {
	knownName := "stackrox"
	unknownName := "Joseph Rules"
	knownNamespace := &v1.Namespace{
		ObjectMeta: k8sMetaV1.ObjectMeta{
			Name: knownName,
		},
	}
	unknownNamespace := &v1.Namespace{
		ObjectMeta: k8sMetaV1.ObjectMeta{
			Name: unknownName,
		},
	}
	gatherer := newNamespaceGatherer(fake.NewSimpleClientset(knownNamespace, unknownNamespace), resources.InitializeStore().Deployments())
	namespaces, err := gatherer.Gather(context.Background())
	s.Empty(err)
	s.Len(namespaces, 2)
	nameMap := make(map[string]*data.NamespaceInfo, 2)
	for _, namespace := range namespaces {
		nameMap[namespace.Name] = namespace
	}
	s.Len(nameMap, 2)
	s.Contains(nameMap, knownName)
	s.NotContains(nameMap, unknownName)
}
