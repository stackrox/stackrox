package utils

import (
	"context"
	"fmt"
	"time"

	"github.com/stackrox/rox/pkg/k8sutil"
	"k8s.io/apimachinery/pkg/types"
	ctrlClient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// ListSiblings populates "list" with objects in the same namespace as "object", using "client".
// Panics on error.
func ListSiblings(ctx context.Context, list ctrlClient.ObjectList, object ctrlClient.Object, client ctrlClient.Client) {
	// We specify a timeout as an attempt to not hang indefinitely, in case the upstream context has no deadline.
	// Hopefully an informer-backed client does not block anyway.
	ctx, cancel := context.WithTimeout(ctx, 10*time.Minute)
	defer cancel() // free resources if List returns ok
	if err := client.List(ctx, list, ctrlClient.InNamespace(object.GetNamespace())); err != nil {
		// This should restart the controller process and force a reconciliation.
		// Currently, there is not much we can do as an alternative, see
		// https://kubernetes.slack.com/archives/C02MRBMN00Z/p1638785272070400?thread_ts=1638784979.069900&cid=C02MRBMN00Z
		// Ignoring the error could mean failing to create a necessary resource for as long as the
		// default time based reconcile (10 hours by default).
		// Hopefully List() call from an informer-backed client is unlikely, so this is likely one of those
		// "should never happen" situations.
		panic(fmt.Errorf("cannot retrieve %T in namespace %q when processing event from %T %q: %w", list, object.GetNamespace(), object, object.GetName(), err))
	}
}

// RequestFor returns a new Request struct referring to the "object".
func RequestFor(object k8sutil.NamespacedObject) reconcile.Request {
	return reconcile.Request{NamespacedName: types.NamespacedName{
		Namespace: object.GetNamespace(),
		Name:      object.GetName(),
	}}
}
