package networkpolicies

import (
	"fmt"

	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/k8sutil"
	"github.com/stackrox/rox/pkg/protoconv/networkpolicy"
	networkingV1 "k8s.io/api/networking/v1"
)

func parseModification(mod *v1.NetworkPolicyModification) ([]*networkingV1.NetworkPolicy, map[k8sutil.NSObjRef]struct{}, error) {
	toDelete := make(map[k8sutil.NSObjRef]struct{})

	for _, toDeleteProto := range mod.GetToDelete() {
		toDelete[k8sutil.RefOf(toDeleteProto)] = struct{}{}
	}

	policies, err := networkpolicy.YamlWrap{Yaml: mod.GetApplyYaml()}.ToKubernetesNetworkPolicies()
	if err != nil {
		return nil, nil, fmt.Errorf("parsing YAML: %v", err)
	}

	return policies, toDelete, nil
}

func validateModification(policies []*networkingV1.NetworkPolicy, toDelete map[k8sutil.NSObjRef]struct{}) error {
	var errList errorhelpers.ErrorList

	uniqueRefs := make(map[k8sutil.NSObjRef]struct{})

	for _, policy := range policies {
		if policy.Name == "" {
			errList.AddString("network policy has empty name")
			continue
		}
		if policy.Namespace == "" {
			errList.AddString("network policy has empty namespace")
			continue
		}
		ref := k8sutil.RefOf(policy)
		if _, ok := uniqueRefs[ref]; ok {
			errList.AddStringf("multiple network policies with name %s in namespace %s", policy.Name, policy.Namespace)
			continue
		}
		uniqueRefs[ref] = struct{}{}
	}

	return errList.ToError()
}
