package reconciler

import (
	platform "github.com/stackrox/stackrox/operator/apis/platform/v1alpha1"
	"github.com/stackrox/stackrox/operator/pkg/utils"
	controllerruntime "sigs.k8s.io/controller-runtime"
	ctrlClient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// handleSiblingCentrals returns an event handler which generates reconcile requests for
// every (typically one) Central resource which resides in the same namespace as the
// observed SecuredCluster resource.
// TODO(ROX-9617): merge with handleSiblingSecuredClusters once we have generics
func handleSiblingCentrals(manager controllerruntime.Manager) handler.EventHandler {
	return handler.EnqueueRequestsFromMapFunc(func(securedCluster ctrlClient.Object) []reconcile.Request {
		centralList := &platform.CentralList{}
		utils.ListSiblings(centralList, securedCluster, manager.GetClient())
		var ret []reconcile.Request
		for _, c := range centralList.Items {
			ret = append(ret, utils.RequestFor(&c)) // #nosec
		}
		return ret
	})
}
