package networkpolicy

import (
	"bytes"

	roxV1 "github.com/stackrox/rox/generated/api/v1"
	k8sV1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/util/yaml"
)

// YamlWrap wraps a string json formatted yaml and provides functions to decode it into stackrox or k8s network policies.
type YamlWrap struct {
	yaml string
}

// ToKubernetesNetworkPolicy outputs the k8s NetworkPolicy proto described by the input yaml.
func (y YamlWrap) ToKubernetesNetworkPolicy() (*k8sV1.NetworkPolicy, error) {
	var k8sNp k8sV1.NetworkPolicy
	err := yaml.NewYAMLToJSONDecoder(bytes.NewReader([]byte(y.yaml))).Decode(&k8sNp)
	return &k8sNp, err
}

// ToRoxNetworkPolicy outputs the stackrox NetworkPolicy proto described by the input yaml.
func (y YamlWrap) ToRoxNetworkPolicy() (*roxV1.NetworkPolicy, error) {
	networkPolicy, err := y.ToKubernetesNetworkPolicy()
	return KubernetesNetworkPolicyWrap{networkPolicy}.ToRoxNetworkPolicy(), err
}
