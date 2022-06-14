package networkpolicy

import (
	"bytes"
	"io"

	"github.com/stackrox/stackrox/generated/storage"
	k8sV1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/util/yaml"
)

// YamlWrap wraps a string json formatted yaml and provides functions to decode it into stackrox or k8s network policies.
// YAMLs may contain configs for many policies in a single string.
type YamlWrap struct {
	Yaml string
}

// ToKubernetesNetworkPolicies outputs the k8s NetworkPolicy protos described by the input yaml.
func (y YamlWrap) ToKubernetesNetworkPolicies() (k8sPolicies []*k8sV1.NetworkPolicy, err error) {
	decoder := yaml.NewYAMLToJSONDecoder(bytes.NewReader([]byte(y.Yaml)))
	for err == nil {
		k8sPolicy := new(k8sV1.NetworkPolicy)
		err = decoder.Decode(k8sPolicy)
		if err == nil {
			k8sPolicies = append(k8sPolicies, k8sPolicy)
		}
	}
	if err == io.EOF {
		err = nil
	}
	return
}

// ToRoxNetworkPolicies outputs the stackrox NetworkPolicy protos described by the input yaml.
func (y YamlWrap) ToRoxNetworkPolicies() (roxPolicies []*storage.NetworkPolicy, err error) {
	var k8sPolicies []*k8sV1.NetworkPolicy
	k8sPolicies, err = y.ToKubernetesNetworkPolicies()
	if err != nil {
		return
	}
	for _, k8SPolicy := range k8sPolicies {
		roxPolicy := KubernetesNetworkPolicyWrap{k8SPolicy}.ToRoxNetworkPolicy()
		roxPolicies = append(roxPolicies, roxPolicy)
	}
	return
}
