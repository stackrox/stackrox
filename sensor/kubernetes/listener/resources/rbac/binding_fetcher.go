package rbac

import (
	"context"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/errorhelpers"
	pkgKubernetes "github.com/stackrox/rox/pkg/kubernetes"
	"github.com/stackrox/rox/pkg/retry"
	v1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

const (
	fetcherRetries = 5
)

type bindingFetcher struct {
	k8sAPI     kubernetes.Interface
	numRetries int
}

func newBindingFetcher(k8sAPI kubernetes.Interface) *bindingFetcher {
	return &bindingFetcher{
		k8sAPI:     k8sAPI,
		numRetries: fetcherRetries,
	}
}

func (r *bindingFetcher) generateManyDependentEvents(bindings []namespacedBindingID, updateRoleID string, isClusterRole bool) ([]*central.SensorEvent, error) {
	errList := errorhelpers.NewErrorList("generating dependent binding events")
	var result []*central.SensorEvent
	for _, b := range bindings {
		if newEvent, err := r.generateDependentEvent(b, updateRoleID, isClusterRole); err != nil {
			errList.AddError(err)
		} else {
			result = append(result, newEvent)
		}
	}

	if !errList.Empty() {
		return nil, errList.ToError()
	}
	return result, nil
}

// generateDependentEvent generates a fake update event for a RoleBinding or a ClusterRoleBinding from a Role or ClusterRole
// that received an update. `relatedBinding` is the metadata reference to the binding that needs to be updated with a new `updateRoleID`.
// Rather than storing all Bindings observed by sensor, we've decided to try something different: fetch Binding data from the K8s API
// when needed. This should only be called on CREATE/DELETE events from Roles. Because any legitimate Role updates won't affect rox bindings.
// Rox bindings need to have a RoleID reference to roles, which can't be changed with an Update event. Therefore the two scenario where this
// functionality is needed is:
// 1) Binding was created first and has RoleID == "" and a Role event creates a Role that matches Binding's roleRef.
// 2) Binding already has a RoleID and matching Role receives a delete event.
//
// This behavior is required in order to disable the re-sync of RoleBindings. This wasn't needed previously because every minute
// role bindings were updated by re-sync. The update events on RoleBindings would pick up any created/removed roles and update
// RoleID accordingly.
func (r *bindingFetcher) generateDependentEvent(relatedBinding namespacedBindingID, updateRoleID string, isClusterRole bool) (*central.SensorEvent, error) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if relatedBinding.IsClusterBinding() {
		var clusterRoleBinding *v1.ClusterRoleBinding
		if err := retry.WithRetry(func() error {
			var apiErr error
			clusterRoleBinding, apiErr = r.k8sAPI.RbacV1().ClusterRoleBindings().Get(ctx, relatedBinding.name, metav1.GetOptions{})
			if apiErr != nil {
				return errors.Wrapf(apiErr, "fetching k8s API for ClusterRoleBinding %s", relatedBinding.name)
			}
			return nil
		}, retry.Tries(r.numRetries), retry.WithExponentialBackoff()); err != nil {
			return nil, err
		}
		pkgKubernetes.TrimAnnotations(clusterRoleBinding)
		return toBindingEvent(toRoxClusterRoleBinding(clusterRoleBinding, updateRoleID), central.ResourceAction_UPDATE_RESOURCE), nil
	}
	var roleBinding *v1.RoleBinding
	if err := retry.WithRetry(func() error {
		var apiErr error
		roleBinding, apiErr = r.k8sAPI.RbacV1().RoleBindings(relatedBinding.namespace).Get(ctx, relatedBinding.name, metav1.GetOptions{})
		if apiErr != nil {
			return errors.Wrapf(apiErr, "fetching k8s API for RoleBinding %s", relatedBinding.name)
		}
		return nil
	}, retry.Tries(r.numRetries), retry.WithExponentialBackoff()); err != nil {
		return nil, err
	}
	pkgKubernetes.TrimAnnotations(roleBinding)
	return toBindingEvent(toRoxRoleBinding(roleBinding, updateRoleID, isClusterRole), central.ResourceAction_UPDATE_RESOURCE), nil
}
