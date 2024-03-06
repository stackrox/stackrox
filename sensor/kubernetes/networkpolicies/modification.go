package networkpolicies

import (
	stdErrors "errors"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/k8sutil"
	"github.com/stackrox/rox/pkg/protoconv/networkpolicy"
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

func validateModification(policies []*networkingV1.NetworkPolicy, _ map[k8sutil.NSObjRef]struct{}) error {
	var validationErrs error

	uniqueRefs := make(map[k8sutil.NSObjRef]struct{})

	for _, policy := range policies {
		if policy.Name == "" {
			validationErrs = stdErrors.Join(validationErrs, errors.New("network policy has empty name"))
			continue
		}
		if policy.Namespace == "" {
			validationErrs = stdErrors.Join(validationErrs, errors.New("network policy has empty namespace"))
			continue
		}
		ref := k8sutil.RefOf(policy)
		if _, ok := uniqueRefs[ref]; ok {
			validationErrs = stdErrors.Join(validationErrs,
				errors.Errorf("multiple network policies with name %s in namespace %s", policy.Name, policy.Namespace))
			continue
		}
		uniqueRefs[ref] = struct{}{}
	}

	return validationErrs
}
