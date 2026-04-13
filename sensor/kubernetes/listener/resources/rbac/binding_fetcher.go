package rbac

import (
	"context"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/errorhelpers"
	pkgKubernetes "github.com/stackrox/rox/pkg/kubernetes"
	"github.com/stackrox/rox/pkg/retry"
	"github.com/stackrox/rox/sensor/kubernetes/client"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/dynamic"
)

const (
	fetcherRetries = 5
)

type bindingFetcher struct {
	dynClient  dynamic.Interface
	numRetries int
}

func newBindingFetcher(dynClient dynamic.Interface) *bindingFetcher {
	return &bindingFetcher{
		dynClient:  dynClient,
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
		return nil, errors.Wrap(errList.ToError(), "generating dependent binding events")
	}
	return result, nil
}

func (r *bindingFetcher) generateDependentEvent(relatedBinding namespacedBindingID, updateRoleID string, isClusterRole bool) (*central.SensorEvent, error) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var event *central.SensorEvent
	err := retry.WithRetry(func() error {
		if relatedBinding.IsClusterBinding() {
			unstructuredObj, apiErr := r.dynClient.Resource(client.ClusterRoleBindingGVR).Get(ctx, relatedBinding.name, metav1.GetOptions{})
			if apiErr != nil {
				return errors.Wrapf(apiErr, "fetching k8s API for ClusterRoleBinding %s", relatedBinding.name)
			}
			var clusterRoleBinding rbacv1.ClusterRoleBinding
			if err := runtime.DefaultUnstructuredConverter.FromUnstructured(unstructuredObj.Object, &clusterRoleBinding); err != nil {
				return errors.Wrap(err, "converting ClusterRoleBinding from unstructured")
			}
			pkgKubernetes.TrimAnnotations(&clusterRoleBinding)
			event = toBindingEvent(toRoxClusterRoleBinding(&clusterRoleBinding, updateRoleID), central.ResourceAction_UPDATE_RESOURCE)
			return nil
		}
		unstructuredObj, apiErr := r.dynClient.Resource(client.RoleBindingGVR).Namespace(relatedBinding.namespace).Get(ctx, relatedBinding.name, metav1.GetOptions{})
		if apiErr != nil {
			return errors.Wrapf(apiErr, "fetching k8s API for RoleBinding %s", relatedBinding.name)
		}
		var roleBinding rbacv1.RoleBinding
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(unstructuredObj.Object, &roleBinding); err != nil {
			return errors.Wrap(err, "converting RoleBinding from unstructured")
		}
		pkgKubernetes.TrimAnnotations(&roleBinding)
		event = toBindingEvent(toRoxRoleBinding(&roleBinding, updateRoleID, isClusterRole), central.ResourceAction_UPDATE_RESOURCE)
		return nil
	}, retry.Tries(r.numRetries), retry.WithExponentialBackoff())
	return event, err
}
