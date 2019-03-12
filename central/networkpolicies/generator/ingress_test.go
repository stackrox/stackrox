package generator

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/namespaces"
	"github.com/stackrox/rox/pkg/networkentity"
	"github.com/stretchr/testify/assert"
)

func createDeploymentNode(id, namespace string, selectorLabels map[string]string) *node {
	return &node{
		entity: networkentity.Entity{
			Type: storage.NetworkEntityInfo_DEPLOYMENT,
			ID:   id,
		},
		deployment: &storage.Deployment{
			Id:        id,
			Namespace: namespace,
			LabelSelector: &storage.LabelSelector{
				MatchLabels: selectorLabels,
			},
		},
		incoming: make(map[*node]struct{}),
		outgoing: make(map[*node]struct{}),
	}
}
func TestGenerateIngressRule_WithInternetIngress(t *testing.T) {
	t.Parallel()

	internetNode := &node{
		entity: networkentity.Entity{
			Type: storage.NetworkEntityInfo_INTERNET,
		},
	}

	deployment0 := createDeploymentNode("deployment0", "ns", map[string]string{"app": "foo"})
	deployment1 := createDeploymentNode("deployment1", "ns", nil)
	deployment1.incoming[deployment0] = struct{}{}
	deployment1.incoming[internetNode] = struct{}{}

	rule := generateIngressRule(deployment1)
	assert.Equal(t, allowAllIngress, rule)
}

func TestGenerateIngressRule_WithInternetExposure(t *testing.T) {
	t.Parallel()

	deployment0 := createDeploymentNode("deployment0", "ns", map[string]string{"app": "foo"})
	deployment1 := createDeploymentNode("deployment1", "ns", nil)

	deployment1.deployment.Ports = []*storage.PortConfig{
		{
			ContainerPort: 443,
			Exposure:      storage.PortConfig_EXTERNAL,
		},
	}
	deployment1.incoming[deployment0] = struct{}{}

	rule := generateIngressRule(deployment1)
	assert.Equal(t, allowAllIngress, rule)
}

func TestGenerateIngressRule_WithoutInternet(t *testing.T) {
	t.Parallel()

	deployment0 := createDeploymentNode("deployment0", "ns1", map[string]string{"app": "foo"})
	deployment1 := createDeploymentNode("deployment1", "ns2", map[string]string{"app": "bar"})

	tgtDeployment := createDeploymentNode("tgtDeployment", "ns1", nil)
	tgtDeployment.incoming[deployment0] = struct{}{}
	tgtDeployment.incoming[deployment1] = struct{}{}

	expectedPeers := []*storage.NetworkPolicyPeer{
		{
			PodSelector: &storage.LabelSelector{
				MatchLabels: map[string]string{"app": "foo"},
			},
		},
		{
			NamespaceSelector: &storage.LabelSelector{
				MatchLabels: map[string]string{namespaces.NamespaceNameLabel: "ns2"},
			},
			PodSelector: &storage.LabelSelector{
				MatchLabels: map[string]string{"app": "bar"},
			},
		},
	}

	rule := generateIngressRule(tgtDeployment)
	assert.ElementsMatch(t, expectedPeers, rule.From)
}
