package complianceoperator

import (
	"time"

	"github.com/ComplianceAsCode/compliance-operator/pkg/apis/compliance/v1alpha1"
	"github.com/stackrox/rox/pkg/complianceoperator"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/sensor/common/pubsub"
	"github.com/stackrox/rox/sensor/kubernetes/eventpipeline/component"
	nodeRoleStore "github.com/stackrox/rox/sensor/kubernetes/listener/resources/noderole/store"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/dynamic"
)

const reconcileDebounce = 5 * time.Second

// NodeRoleReconciler subscribes to NodeRoleChangeEvent notifications and
// reconciles the Roles field on all StackRox-managed ScanSettings so they
// match the MachineConfigPools present in the cluster.
type NodeRoleReconciler struct {
	client        dynamic.Interface
	coInfo        StatusInfo
	nodeRoleStore *nodeRoleStore.Store
	stopSignal    *concurrency.Signal
	trigger       chan struct{}
}

// NewNodeRoleReconciler creates a new NodeRoleReconciler.
func NewNodeRoleReconciler(
	client dynamic.Interface,
	coInfo StatusInfo,
	store *nodeRoleStore.Store,
	stopSignal *concurrency.Signal,
) *NodeRoleReconciler {
	r := &NodeRoleReconciler{
		client:        client,
		coInfo:        coInfo,
		nodeRoleStore: store,
		stopSignal:    stopSignal,
		trigger:       make(chan struct{}, 1),
	}
	go r.run()
	return r
}

// Register subscribes this reconciler to node-role change events on the
// given PubSubDispatcher.
func (r *NodeRoleReconciler) Register(dispatcher component.PubSubDispatcher) error {
	return dispatcher.RegisterConsumerToLane(
		pubsub.NodeRoleReconcilerConsumer,
		pubsub.NodeRoleChangeTopic,
		pubsub.NodeRoleChangeLane,
		r.onNodeRoleChange,
	)
}

func (r *NodeRoleReconciler) onNodeRoleChange(_ pubsub.Event) error {
	// Non-blocking send: if a trigger is already pending, this is a no-op.
	select {
	case r.trigger <- struct{}{}:
	default:
	}
	return nil
}

func (r *NodeRoleReconciler) run() {
	for {
		select {
		case <-r.trigger:
			r.debounceAndReconcile()
		case <-r.stopSignal.Done():
			return
		}
	}
}

// debounceAndReconcile waits for the debounce window to expire without new
// triggers before executing the reconciliation. Rapid-fire events (e.g. initial
// sync) collapse into a single reconciliation.
func (r *NodeRoleReconciler) debounceAndReconcile() {
	timer := time.NewTimer(reconcileDebounce)
	defer timer.Stop()
	for {
		select {
		case <-r.trigger:
			if !timer.Stop() {
				<-timer.C
			}
			timer.Reset(reconcileDebounce)
		case <-timer.C:
			r.reconcile()
			return
		case <-r.stopSignal.Done():
			return
		}
	}
}

func (r *NodeRoleReconciler) reconcile() {
	roles := r.nodeRoleStore.GetRoles()
	if len(roles) == 0 {
		roles = []string{masterRole, workerRole}
	}

	ns := r.coInfo.GetNamespace()
	if ns == "" {
		log.Debug("Compliance operator namespace unknown, skipping ScanSetting reconciliation")
		return
	}

	log.Infof("Node roles changed, reconciling ScanSettings in namespace %q with roles %v", ns, roles)

	ctx := concurrency.AsContext(r.stopSignal)
	res := r.client.Resource(complianceoperator.ScanSetting.GroupVersionResource()).Namespace(ns)
	list, err := res.List(ctx, v1.ListOptions{
		LabelSelector: labels.SelectorFromSet(stackroxLabels).String(),
	})
	if err != nil {
		log.Errorf("Failed to list ScanSettings in namespace %q: %v", ns, err)
		return
	}

	for i := range list.Items {
		item := &list.Items[i]
		var ss v1alpha1.ScanSetting
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(item.Object, &ss); err != nil {
			log.Errorf("Failed to convert ScanSetting %q: %v", item.GetName(), err)
			continue
		}

		ss.Roles = roles
		updated, err := runtimeObjToUnstructured(&ss)
		if err != nil {
			log.Errorf("Failed to convert ScanSetting %q back to unstructured: %v", ss.Name, err)
			continue
		}

		if _, err := res.Update(ctx, updated, v1.UpdateOptions{}); err != nil {
			log.Errorf("Failed to update ScanSetting %q roles: %v", ss.Name, err)
			continue
		}
		log.Infof("Updated ScanSetting %q roles to %v", ss.Name, roles)
	}
}
