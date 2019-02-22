package networkpolicies

import (
	"fmt"

	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/k8sutil"
	networkingV1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/json"
	networkingV1Client "k8s.io/client-go/kubernetes/typed/networking/v1"
)

const (
	applyIDLabelKey         = `network-policies.stackrox.io/apply-id`
	disablePolicyLabelKey   = `network-policies.stackrox.io/disable`
	disablePolicyLabelValue = `nomatch`
)

type rollbackAction interface {
	Execute(client networkingV1Client.NetworkingV1Interface) error
}

type applyTx struct {
	id               string
	networkingClient networkingV1Client.NetworkingV1Interface
	timestamp        string

	rollbackActions []rollbackAction
}

func (t *applyTx) Do(newOrUpdated []*networkingV1.NetworkPolicy, toDelete map[k8sutil.NSObjRef]struct{}) error {
	for _, policy := range newOrUpdated {
		ref := k8sutil.RefOf(policy)
		if _, shouldReplace := toDelete[ref]; shouldReplace {
			if err := t.replaceNetworkPolicy(policy); err != nil {
				return err
			}
			delete(toDelete, ref)
		} else {
			if err := t.createNetworkPolicy(policy); err != nil {
				return err
			}
		}
	}

	for deleteRef := range toDelete {
		if err := t.deleteNetworkPolicy(deleteRef.Namespace, deleteRef.Name); err != nil {
			return err
		}
	}

	return nil
}

func (t *applyTx) oldNetworkPolicyJSONAnnotationKey() string {
	return fmt.Sprintf("previous-json.network-policies.stackrox.io/%s", t.id)
}

func (t *applyTx) applyTimestampAnnotationKey() string {
	return fmt.Sprintf("apply-timestamp.network-policies.stackrox.io/%s", t.id)
}

func (t *applyTx) annotateAndLabel(np *networkingV1.NetworkPolicy) {
	if np.Annotations == nil {
		np.Annotations = make(map[string]string)
	}
	np.Annotations[t.applyTimestampAnnotationKey()] = t.timestamp

	if np.Labels == nil {
		np.Labels = make(map[string]string)
	}
	np.Labels[applyIDLabelKey] = t.id
}

type deletePolicy struct {
	name, namespace string
}

func (a *deletePolicy) Execute(client networkingV1Client.NetworkingV1Interface) error {
	return client.NetworkPolicies(a.namespace).Delete(a.name, &metav1.DeleteOptions{})
}

type restorePolicy struct {
	oldPolicy *networkingV1.NetworkPolicy
}

func (a *restorePolicy) Execute(client networkingV1Client.NetworkingV1Interface) error {
	_, err := client.NetworkPolicies(a.oldPolicy.Namespace).Update(a.oldPolicy)
	return err
}

func (t *applyTx) Rollback() error {
	var errList errorhelpers.ErrorList
	for i := len(t.rollbackActions) - 1; i >= 0; i-- {
		errList.AddError(t.rollbackActions[i].Execute(t.networkingClient))
	}
	return errList.ToError()
}

func (t *applyTx) createNetworkPolicy(policy *networkingV1.NetworkPolicy) error {
	nsClient := t.networkingClient.NetworkPolicies(policy.Namespace)
	_, err := nsClient.Create(policy)
	if err != nil {
		return err
	}
	t.rollbackActions = append(t.rollbackActions, &deletePolicy{
		namespace: policy.Namespace,
		name:      policy.Name,
	})
	return nil
}

func (t *applyTx) replaceNetworkPolicy(policy *networkingV1.NetworkPolicy) error {
	nsClient := t.networkingClient.NetworkPolicies(policy.Namespace)

	old, err := nsClient.Get(policy.Name, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("retrieving network policy: %v", err)
	}

	oldJSON, err := json.Marshal(old)
	if err != nil {
		return fmt.Errorf("marshalling old network policy: %v", err)
	}

	t.annotateAndLabel(policy)
	policy.Annotations[t.oldNetworkPolicyJSONAnnotationKey()] = string(oldJSON)

	updated, err := nsClient.Update(policy)
	if err != nil {
		return fmt.Errorf("updating network policy: %v", err)
	}

	old.ResourceVersion = updated.ResourceVersion
	t.rollbackActions = append(t.rollbackActions, &restorePolicy{
		oldPolicy: old,
	})
	return nil
}

func (t *applyTx) deleteNetworkPolicy(namespace, name string) error {
	nsClient := t.networkingClient.NetworkPolicies(namespace)

	existing, err := nsClient.Get(name, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("retrieving network policy: %v", err)
	}

	disabled := existing.DeepCopy()
	t.annotateAndLabel(disabled)
	disabled.Spec.PodSelector.MatchLabels[disablePolicyLabelKey] = disablePolicyLabelValue

	updated, err := nsClient.Update(disabled)
	if err != nil {
		return fmt.Errorf("updating network policy: %v", err)
	}
	existing.ResourceVersion = updated.ResourceVersion
	t.rollbackActions = append(t.rollbackActions, &restorePolicy{
		oldPolicy: existing,
	})
	return nil
}
