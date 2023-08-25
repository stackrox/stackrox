package networkpolicies

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/k8sutil"
	"github.com/stackrox/rox/pkg/kubernetes"
	"github.com/stackrox/rox/pkg/protoconv/networkpolicy"
	networkingV1 "k8s.io/api/networking/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/json"
	networkingV1Client "k8s.io/client-go/kubernetes/typed/networking/v1"
)

const (
	applyIDLabelKey         = `network-policies.stackrox.io/apply-id`
	disablePolicyLabelKey   = `network-policies.stackrox.io/disable`
	disablePolicyLabelValue = `nomatch`

	previousJSONAnnotationKey   = `network-policies.stackrox.io/previous-json`
	applyTimestampAnnotationKey = `network-policies.stackrox.io/apply-timestamp`

	deletedAnnotationKey   = `network-policies.stackrox.io/deleted`
	deletedAnnotationValue = `This network policy was deleted via StackRox. To avoid the risk of data loss, it has been preserved but ` +
		`rendered ineffective by means of a special pod selector. Remove the "` + disablePolicyLabelKey + `" pod ` +
		`selector and rename the policy to restore it.`
	originalNameAnnotationKey = `network-policies.stackrox.io/original-name`

	yamlSep = "---\n"

	maxConflictRetries = 5
)

type rollbackAction interface {
	Execute(ctx context.Context, client networkingV1Client.NetworkingV1Interface) error
	Record(mod *storage.NetworkPolicyModification)
}

type applyTx struct {
	id               string
	networkingClient networkingV1Client.NetworkingV1Interface
	timestamp        string

	rollbackActions []rollbackAction
}

func (t *applyTx) Do(ctx context.Context, newOrUpdated []*networkingV1.NetworkPolicy, toDelete map[k8sutil.NSObjRef]struct{}) error {
	for _, policy := range newOrUpdated {
		ref := k8sutil.RefOf(policy)
		if _, shouldReplace := toDelete[ref]; shouldReplace {
			if err := t.replaceNetworkPolicy(ctx, policy); err != nil {
				return err
			}
			delete(toDelete, ref)
		} else {
			if err := t.createNetworkPolicy(ctx, policy); err != nil {
				return err
			}
		}
	}

	for deleteRef := range toDelete {
		if err := t.deleteNetworkPolicy(ctx, deleteRef.Namespace, deleteRef.Name); err != nil {
			return err
		}
	}

	return nil
}

func (t *applyTx) annotateAndLabel(np *networkingV1.NetworkPolicy) {
	if np.Annotations == nil {
		np.Annotations = make(map[string]string)
	}
	np.Annotations[applyTimestampAnnotationKey] = t.timestamp

	if np.Labels == nil {
		np.Labels = make(map[string]string)
	}
	np.Labels[applyIDLabelKey] = t.id
}

type norecordAction struct {
	rollbackAction
}

func (a norecordAction) Record(_ *storage.NetworkPolicyModification) {}

type deletePolicy struct {
	name, namespace string
}

func (a *deletePolicy) Execute(ctx context.Context, client networkingV1Client.NetworkingV1Interface) error {
	return client.NetworkPolicies(a.namespace).Delete(ctx, a.name, kubernetes.DeleteBackgroundOption)
}

func (a *deletePolicy) Record(mod *storage.NetworkPolicyModification) {
	mod.ToDelete = append(mod.ToDelete, &storage.NetworkPolicyReference{
		Namespace: a.namespace,
		Name:      a.name,
	})
}

type restorePolicy struct {
	oldPolicy  *networkingV1.NetworkPolicy
	wasDeleted bool
}

func (a *restorePolicy) Execute(ctx context.Context, client networkingV1Client.NetworkingV1Interface) error {
	_, err := client.NetworkPolicies(a.oldPolicy.Namespace).Update(ctx, a.oldPolicy, metav1.UpdateOptions{})
	return err
}

func (a *restorePolicy) Record(mod *storage.NetworkPolicyModification) {
	if mod.ApplyYaml != "" {
		mod.ApplyYaml += yamlSep
	}
	yaml, err := networkpolicy.KubernetesNetworkPolicyWrap{NetworkPolicy: a.oldPolicy}.ToYaml()
	if err != nil {
		// This makes the YAML malformed, but it is still better than failing to provide any value here. If something
		// goes wrong, having the user look into the returned YAML seems the right thing to do.
		yaml = fmt.Sprintf("ERROR serializing networkpolicy YAML: %v\n", err)
	}
	mod.ApplyYaml += yaml

	if !a.wasDeleted {
		mod.ToDelete = append(mod.ToDelete, &storage.NetworkPolicyReference{
			Namespace: a.oldPolicy.Namespace,
			Name:      a.oldPolicy.Name,
		})
	}
}

func (t *applyTx) Rollback(ctx context.Context) error {
	var errList errorhelpers.ErrorList
	for i := len(t.rollbackActions) - 1; i >= 0; i-- {
		errList.AddError(t.rollbackActions[i].Execute(ctx, t.networkingClient))
	}
	return errList.ToError()
}

func (t *applyTx) createNetworkPolicy(ctx context.Context, policy *networkingV1.NetworkPolicy) error {
	nsClient := t.networkingClient.NetworkPolicies(policy.Namespace)

	if policy.ResourceVersion != "" {
		policy = policy.DeepCopy()
		policy.ResourceVersion = ""
	}

	_, err := nsClient.Create(ctx, policy, metav1.CreateOptions{})
	if err != nil {
		return err
	}
	t.rollbackActions = append(t.rollbackActions, &deletePolicy{
		namespace: policy.Namespace,
		name:      policy.Name,
	})
	return nil
}

func (t *applyTx) replaceNetworkPolicy(ctx context.Context, policy *networkingV1.NetworkPolicy) error {
	nsClient := t.networkingClient.NetworkPolicies(policy.Namespace)

	for retryCount := 0; retryCount < maxConflictRetries; retryCount++ {
		old, err := nsClient.Get(ctx, policy.Name, metav1.GetOptions{})

		if err != nil {
			if k8sErrors.IsNotFound(err) {
				// The policy has possibly been deleted. Either way, doesn't matter for us.
				return t.createNetworkPolicy(ctx, policy)
			}
			return errors.Wrap(err, "retrieving network policy")
		}

		// Do not serialize the `previous JSON` annotation when JSON-encoding.
		oldStripped := old.DeepCopy()
		delete(oldStripped.Annotations, previousJSONAnnotationKey)
		oldJSON, err := json.Marshal(oldStripped)
		if err != nil {
			return errors.Wrap(err, "marshalling old network policy")
		}

		t.annotateAndLabel(policy)
		policy.Annotations[previousJSONAnnotationKey] = string(oldJSON)

		updated, err := nsClient.Update(ctx, policy, metav1.UpdateOptions{})
		if err != nil {
			if k8sErrors.IsConflict(err) {
				log.Errorf("Encountered conflict when trying to update network policy %s/%s: %v. Retrying (attempt %d of %d)...", old.GetNamespace(), old.GetName(), err, retryCount+1, maxConflictRetries)
				continue
			}
			return errors.Wrap(err, "updating network policy")
		}

		// For rollback, update the resource version of the original network policy to the updated one, so we don't
		// accidentally overwrite a concurrent modification.
		old.ResourceVersion = updated.ResourceVersion
		t.rollbackActions = append(t.rollbackActions, &restorePolicy{
			oldPolicy: old,
		})

		return nil
	}

	return fmt.Errorf("trying to update network policy %s/%s: giving up after %d conflicts", policy.GetNamespace(), policy.GetName(), maxConflictRetries)
}

func (t *applyTx) deleteNetworkPolicy(ctx context.Context, namespace, name string) error {
	nsClient := t.networkingClient.NetworkPolicies(namespace)

	existing, err := nsClient.Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return errors.Wrap(err, "retrieving network policy")
	}

	deleted := existing.DeepCopy()
	// no need to worry about max length, name will automatically be truncated.
	deleted.GenerateName = fmt.Sprintf("deleted-%s-", deleted.Name)
	t.annotateAndLabel(deleted)
	deleted.Annotations[deletedAnnotationKey] = deletedAnnotationValue
	deleted.Annotations[originalNameAnnotationKey] = existing.Name
	if deleted.Spec.PodSelector.MatchLabels == nil {
		deleted.Spec.PodSelector.MatchLabels = make(map[string]string)
	}
	deleted.Spec.PodSelector.MatchLabels[disablePolicyLabelKey] = disablePolicyLabelValue
	deleted.Name = ""
	deleted.ResourceVersion = ""

	deleted, err = nsClient.Create(ctx, deleted, metav1.CreateOptions{})
	if err != nil {
		return errors.Wrap(err, "creating backup network policy")
	}
	t.rollbackActions = append(t.rollbackActions, norecordAction{rollbackAction: &deletePolicy{
		namespace: deleted.Namespace,
		name:      deleted.Name,
	}})

	err = nsClient.Delete(ctx, existing.Name, kubernetes.DeleteBackgroundOption)
	if err != nil {
		return errors.Wrapf(err, "deleting network policy %s/%s", existing.Namespace, existing.Name)
	}
	t.rollbackActions = append(t.rollbackActions, &restorePolicy{
		oldPolicy:  existing,
		wasDeleted: true,
	})

	return nil
}

func (t *applyTx) UndoModification() *storage.NetworkPolicyModification {
	mod := &storage.NetworkPolicyModification{}
	for i := len(t.rollbackActions) - 1; i >= 0; i-- {
		t.rollbackActions[i].Record(mod)
	}
	return mod
}
