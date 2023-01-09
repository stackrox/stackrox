package rbac

import (
	"context"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/errorhelpers"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type bindingFetcher struct {
	k8sAPI kubernetes.Interface
}

func newBindingFetcher(k8sAPI kubernetes.Interface) *bindingFetcher {
	return &bindingFetcher{k8sAPI}
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

func (r *bindingFetcher) generateDependentEvent(relatedBinding namespacedBindingID, updateRoleID string, isClusterRole bool) (*central.SensorEvent, error) {
	if relatedBinding.IsClusterBinding() {
		clusterBinding, err := r.k8sAPI.RbacV1().ClusterRoleBindings().Get(context.TODO(), relatedBinding.name, metav1.GetOptions{})
		if err != nil {
			return nil, errors.Wrapf(err, "fetching k8s API for ClusterRoleBinding %s", relatedBinding.name)
		}
		return toBindingEvent(toRoxClusterRoleBinding(clusterBinding, updateRoleID), central.ResourceAction_UPDATE_RESOURCE), nil
	}
	namespacedBinding, err := r.k8sAPI.RbacV1().RoleBindings(relatedBinding.namespace).Get(context.TODO(), relatedBinding.name, metav1.GetOptions{})
	if err != nil {
		return nil, errors.Wrapf(err, "fetching k8s API for ClusterRoleBinding %s", relatedBinding.name)
	}
	return toBindingEvent(toRoxRoleBinding(namespacedBinding, updateRoleID, isClusterRole), central.ResourceAction_UPDATE_RESOURCE), nil
}
