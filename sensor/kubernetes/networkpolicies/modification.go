package networkpolicies

import (
	"github.com/pkg/errors"
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/errorhelpers"
	"github.com/stackrox/stackrox/pkg/k8sutil"
	"github.com/stackrox/stackrox/pkg/protoconv/networkpolicy"
	networkingV1 "k8s.io/api/networking/v1"
)

func parseModification(mod *storage.NetworkPolicyModification) ([]*networkingV1.NetworkPolicy, map[k8sutil.NSObjRef]struct{}, error) {
	toDelete := make(map[k8sutil.NSObjRef]struct{})

	for _, toDeleteProto := range mod.GetToDelete() {
		toDelete[k8sutil.RefOf(toDeleteProto)] = struct{}{}
	}

	policies, err := networkpolicy.YamlWrap{Yaml: mod.GetApplyYaml()}.ToKubernetesNetworkPolicies()
	if err != nil {
		return nil, nil, errors.Wrap(err, "parsing YAML")
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
