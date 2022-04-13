package gatherers

import (
	"context"
	"testing"

	"github.com/stackrox/stackrox/pkg/telemetry/data"
	"github.com/stretchr/testify/suite"
	v1 "k8s.io/api/core/v1"
	k8sMetaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/fake"
)

func TestNodeGatherer(t *testing.T) {
	suite.Run(t, new(NodeGathererTestSuite))
}

type NodeGathererTestSuite struct {
	suite.Suite
}

// As of the time I wrote this test the only business logic in the Node gatherer is interpreting the conditions, so this
// is all I've tested
func (s *NodeGathererTestSuite) TestGatherNodes() {
	noConditionsID := types.UID("NoConditions")
	noConditions := &v1.Node{
		ObjectMeta: k8sMetaV1.ObjectMeta{
			UID:  noConditionsID,
			Name: "name1",
		},
		Status: v1.NodeStatus{
			Conditions: []v1.NodeCondition{
				{
					Type:   v1.NodeReady,
					Status: v1.ConditionTrue,
				},
			},
		},
	}
	conditionsID := types.UID("YesConditions")
	conditions := &v1.Node{
		ObjectMeta: k8sMetaV1.ObjectMeta{
			UID:  conditionsID,
			Name: "name2",
		},
		Status: v1.NodeStatus{
			Conditions: []v1.NodeCondition{
				{
					Type:   v1.NodeDiskPressure,
					Status: v1.ConditionTrue,
				},
			},
		},
	}
	gatherer := newNodeGatherer(fake.NewSimpleClientset(noConditions, conditions))
	gathered, err := gatherer.Gather(context.Background())
	s.NoError(err)
	s.Len(gathered, 2)
	idMap := make(map[string]*data.NodeInfo, 2)
	for _, node := range gathered {
		idMap[node.ID] = node
	}
	s.Contains(idMap, string(noConditionsID))
	s.Contains(idMap, string(conditionsID))

	noConditionsNode := idMap[string(noConditionsID)]
	s.Empty(noConditionsNode.AdverseConditions)
	conditionsNode := idMap[string(conditionsID)]
	s.Len(conditionsNode.AdverseConditions, 1)
}
