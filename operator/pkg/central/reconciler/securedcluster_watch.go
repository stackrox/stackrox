package reconciler

import (
	"context"
	"fmt"
	"time"

	platform "github.com/stackrox/rox/operator/apis/platform/v1alpha1"
	"k8s.io/apimachinery/pkg/types"
	controllerruntime "sigs.k8s.io/controller-runtime"
	ctrlClient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// handleSiblingCentrals returns an event handler which generates reconcile requests for
// every (typically one) Central resource which resides in the same namespace as the
// observed SecuredCluster resource.
func handleSiblingCentrals(manager controllerruntime.Manager) handler.EventHandler {
	return handler.EnqueueRequestsFromMapFunc(func(securedCluster ctrlClient.Object) []reconcile.Request {
		var ret []reconcile.Request
		for _, c := range listSiblingCentrals(securedCluster, manager.GetClient()) {
			ret = append(ret, requestFor(c))
		}
		return ret
	})
}

func listSiblingCentrals(securedCluster ctrlClient.Object, client ctrlClient.Client) []platform.Central {
	// Unfortunately the EventHandler API does not provide a context, so we do our best
	// not to hang indefinitely. Hopefully an informer-backed client does not block anyway.
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel() // free resources if List returns ok
	list := &platform.CentralList{}
	if err := client.List(ctx, list, ctrlClient.InNamespace(securedCluster.GetNamespace())); err != nil {
		// This should restart the controller process and force a reconciliation.
		// Currently, there is not much we can do as an alternative, see
		// https://kubernetes.slack.com/archives/C02MRBMN00Z/p1638785272070400?thread_ts=1638784979.069900&cid=C02MRBMN00Z
		// Ignoring the error could mean failing to create an init bundle for a new SecuredCluster for as long as the
		// default time based reconcile (10 hours by default).
		// Hopefully List() call from an informer-backed client is unlikely, so this is likely one of those
		// "should never happen" situations.
		panic(fmt.Errorf("cannot list Centrals in namespace %q when processing event from SecuredCluster %q: %w", securedCluster.GetNamespace(), securedCluster.GetName(), err))
	}
	return list.Items
}

func requestFor(central platform.Central) reconcile.Request {
	return reconcile.Request{NamespacedName: types.NamespacedName{
		Namespace: central.GetNamespace(),
		Name:      central.GetName(),
	}}
}

// createAndDeleteOnly is a Predicate which triggers reconciliations only on creation and deletion events.
// This is because only appearance and disappearance of a SecuredCluster resource can influence whether
// an init bundle should be created by the Central controller.
// We define our own type to avoid the default-true behaviour of the Funcs predicate in case
// new methods are added to the Predicate interface in the future.
type createAndDeleteOnly struct{}

var _ predicate.Predicate = createAndDeleteOnly{}

func (c createAndDeleteOnly) Create(_ event.CreateEvent) bool {
	return true
}

func (c createAndDeleteOnly) Delete(_ event.DeleteEvent) bool {
	return true
}

func (c createAndDeleteOnly) Update(_ event.UpdateEvent) bool {
	return false
}

func (c createAndDeleteOnly) Generic(_ event.GenericEvent) bool {
	return false
}
