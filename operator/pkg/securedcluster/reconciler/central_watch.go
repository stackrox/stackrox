package reconciler

import (
	platform "github.com/stackrox/rox/operator/apis/platform/v1alpha1"
	"github.com/stackrox/rox/operator/pkg/utils"
	controllerruntime "sigs.k8s.io/controller-runtime"
	ctrlClient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// handleSiblingSecuredClusters returns an event handler which generates reconcile requests for
// every (typically one) SecuredCluster resource which resides in the same namespace as the
// observed Central resource.
// TODO(ROX-9617): merge with handleSiblingCentrals once we have generics
func handleSiblingSecuredClusters(manager controllerruntime.Manager) handler.EventHandler {
	return handler.EnqueueRequestsFromMapFunc(func(central ctrlClient.Object) []reconcile.Request {
		securedClusterList := &platform.SecuredClusterList{}
		utils.ListSiblings(securedClusterList, central, manager.GetClient())
		var ret []reconcile.Request
		for _, c := range securedClusterList.Items {
			ret = append(ret, utils.RequestFor(&c)) // #nosec
		}
		return ret
	})
}
