package networkpolicy

import (
	roxV1 "github.com/stackrox/rox/generated/api/v1"
	k8sV1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/runtime/serializer/json"
	"k8s.io/client-go/discovery"
)

// YamlWrap wraps a string json formatted yaml and provides functions to decode it into stackrox or k8s network policies.
type YamlWrap struct {
	yaml string
}

// ToKubernetesNetworkPolicy outputs the k8s NetworkPolicy proto described by the input yaml.
func (y YamlWrap) ToKubernetesNetworkPolicy() (*k8sV1.NetworkPolicy, error) {
	decoder := json.NewYAMLSerializer(json.DefaultMetaFactory, nil, &discovery.UnstructuredObjectTyper{})

	networkPolicy := new(k8sV1.NetworkPolicy)
	_, _, err := decoder.Decode([]byte(y.yaml), nil, networkPolicy)
	if err != nil {
		return nil, err
	}
	return networkPolicy, nil
}

// ToRoxNetworkPolicy outputs the stackrox NetworkPolicy proto described by the input yaml.
func (y YamlWrap) ToRoxNetworkPolicy() (*roxV1.NetworkPolicy, error) {
	networkPolicy, err := y.ToKubernetesNetworkPolicy()
	return KubernetesNetworkPolicyWrap{networkPolicy}.ToRoxNetworkPolicy(), err
}
