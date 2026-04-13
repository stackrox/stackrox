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
	"github.com/stackrox/rox/sensor/kubernetes/client"
	networkingV1 "k8s.io/api/networking/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/json"
	"k8s.io/client-go/dynamic"
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
	Execute(ctx context.Context, dynClient dynamic.Interface) error
	Record(mod *storage.NetworkPolicyModification)
}

type applyTx struct {
	id        string
	dynClient dynamic.Interface
	timestamp string

	rollbackActions []rollbackAction
}

func npClient(dynClient dynamic.Interface, ns string) dynamic.ResourceInterface {
	return dynClient.Resource(client.NetworkPolicyGVR).Namespace(ns)
}

func toUnstructuredNP(np *networkingV1.NetworkPolicy) (*unstructured.Unstructured, error) {
	obj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(np)
	if err != nil {
		return nil, err
	}
	return &unstructured.Unstructured{Object: obj}, nil
}

func fromUnstructuredNP(u *unstructured.Unstructured) (*networkingV1.NetworkPolicy, error) {
	var np networkingV1.NetworkPolicy
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(u.Object, &np); err != nil {
		return nil, err
	}
	return &np, nil
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

func (a *deletePolicy) Execute(ctx context.Context, dynClient dynamic.Interface) error {
	err := npClient(dynClient, a.namespace).Delete(ctx, a.name, kubernetes.DeleteBackgroundOption)
	if err != nil {
		return errors.Wrapf(err, "deleting network policy %s/%s", a.namespace, a.name)
	}
	return nil
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

func (a *restorePolicy) Execute(ctx context.Context, dynClient dynamic.Interface) error {
	u, err := toUnstructuredNP(a.oldPolicy)
	if err != nil {
		return errors.Wrap(err, "converting network policy to unstructured for restore")
	}
	_, err = npClient(dynClient, a.oldPolicy.Namespace).Update(ctx, u, metav1.UpdateOptions{})
	if err != nil {
		return errors.Wrap(err, "restoring network policy")
	}
	return nil
}

func (a *restorePolicy) Record(mod *storage.NetworkPolicyModification) {
	if mod.GetApplyYaml() != "" {
		mod.ApplyYaml += yamlSep
	}
	yaml, err := networkpolicy.KubernetesNetworkPolicyWrap{NetworkPolicy: a.oldPolicy}.ToYaml()
	if err != nil {
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
		errList.AddError(t.rollbackActions[i].Execute(ctx, t.dynClient))
	}
	if err := errList.ToError(); err != nil {
		return errors.Wrap(err, "reverting network policy modifications")
	}
	return nil
}

func (t *applyTx) createNetworkPolicy(ctx context.Context, policy *networkingV1.NetworkPolicy) error {
	c := npClient(t.dynClient, policy.Namespace)

	if policy.ResourceVersion != "" {
		policy = policy.DeepCopy()
		policy.ResourceVersion = ""
	}

	u, err := toUnstructuredNP(policy)
	if err != nil {
		return errors.Wrap(err, "converting network policy")
	}

	_, err = c.Create(ctx, u, metav1.CreateOptions{})
	if err != nil {
		return errors.Wrapf(err, "creating network policy %s/%s", policy.Namespace, policy.Name)
	}
	t.rollbackActions = append(t.rollbackActions, &deletePolicy{
		namespace: policy.Namespace,
		name:      policy.Name,
	})
	return nil
}

func (t *applyTx) replaceNetworkPolicy(ctx context.Context, policy *networkingV1.NetworkPolicy) error {
	c := npClient(t.dynClient, policy.Namespace)

	for retryCount := 0; retryCount < maxConflictRetries; retryCount++ {
		oldU, err := c.Get(ctx, policy.Name, metav1.GetOptions{})

		if err != nil {
			if k8sErrors.IsNotFound(err) {
				return t.createNetworkPolicy(ctx, policy)
			}
			return errors.Wrap(err, "retrieving network policy")
		}

		old, err := fromUnstructuredNP(oldU)
		if err != nil {
			return errors.Wrap(err, "converting old network policy")
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

		u, err := toUnstructuredNP(policy)
		if err != nil {
			return errors.Wrap(err, "converting policy to unstructured")
		}

		updatedU, err := c.Update(ctx, u, metav1.UpdateOptions{})
		if err != nil {
			if k8sErrors.IsConflict(err) {
				log.Errorf("Encountered conflict when trying to update network policy %s/%s: %v. Retrying (attempt %d of %d)...", old.GetNamespace(), old.GetName(), err, retryCount+1, maxConflictRetries)
				continue
			}
			return errors.Wrap(err, "updating network policy")
		}

		// For rollback, update the resource version of the original network policy to the updated one.
		old.ResourceVersion = updatedU.GetResourceVersion()
		t.rollbackActions = append(t.rollbackActions, &restorePolicy{
			oldPolicy: old,
		})

		return nil
	}

	return fmt.Errorf("trying to update network policy %s/%s: giving up after %d conflicts", policy.GetNamespace(), policy.GetName(), maxConflictRetries)
}

func (t *applyTx) deleteNetworkPolicy(ctx context.Context, namespace, name string) error {
	c := npClient(t.dynClient, namespace)

	existingU, err := c.Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return errors.Wrap(err, "retrieving network policy")
	}

	existing, err := fromUnstructuredNP(existingU)
	if err != nil {
		return errors.Wrap(err, "converting existing network policy")
	}

	deleted := existing.DeepCopy()
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

	deletedU, err := toUnstructuredNP(deleted)
	if err != nil {
		return errors.Wrap(err, "converting deleted network policy")
	}

	createdU, err := c.Create(ctx, deletedU, metav1.CreateOptions{})
	if err != nil {
		return errors.Wrap(err, "creating backup network policy")
	}
	t.rollbackActions = append(t.rollbackActions, norecordAction{rollbackAction: &deletePolicy{
		namespace: createdU.GetNamespace(),
		name:      createdU.GetName(),
	}})

	err = c.Delete(ctx, existing.Name, kubernetes.DeleteBackgroundOption)
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
