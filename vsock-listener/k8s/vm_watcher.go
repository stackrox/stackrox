package k8s

// VirtualMachineInstance Client Implementation
//
// This implementation uses the Kubernetes dynamic client rather than KubeVirt's
// generated typed client. This represents a classic software engineering trade-off:
//
// DYNAMIC CLIENT (this implementation):
// ✅ Fewer dependency conflicts when updating Kubernetes/KubeVirt versions
// ✅ Works across Kubernetes version upgrades without regenerating clients
// ✅ Uses standard k8s.io/client-go libraries already in the project
// ❌ More boilerplate code for type conversion (unstructured → typed)
// ❌ Harder to understand (runtime type conversion instead of compile-time types)
// ❌ Type conversion can fail at runtime vs compile-time safety
//
// TYPED CLIENT (kubevirt.io/client-go - not used):
// ✅ Clean, readable code with full compile-time type safety
// ✅ No runtime type conversion boilerplate
// ❌ Complex dependency tree prone to version conflicts
// ❌ May break during Kubernetes upgrades
// ❌ Requires regenerating clients when CRD schemas change
//
// For production environments, dependency stability usually outweighs code clarity
// because one person writes the boilerplate once, but many people suffer from
// dependency conflicts over time as the Kubernetes ecosystem evolves.

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/logging"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/tools/cache"
	kubevirtv1 "kubevirt.io/api/core/v1"
)

var (
	log = logging.LoggerForModule()
)

// VMInfo represents VM information needed for VSOCK communication
type VMInfo struct {
	Name      string
	Namespace string
	UID       string
	VSOCKCID  uint32
	Labels    map[string]string
}

var (
	// VirtualMachineInstance GVR for dynamic client
	vmiGVR = schema.GroupVersionResource{
		Group:    "kubevirt.io",
		Version:  "v1",
		Resource: "virtualmachineinstances",
	}
)

// VMWatcher watches VirtualMachineInstance resources and tracks VSOCK-enabled VMs
type VMWatcher struct {
	ctx           context.Context
	cancel        context.CancelFunc
	dynamicClient dynamic.Interface
	informer      cache.SharedIndexInformer
	stopper       concurrency.Stopper

	// VM tracking
	vmsMutex sync.RWMutex
	vms      map[uint32]*VMInfo // CID -> VMInfo
	cidToUID map[uint32]string  // CID -> UID for fast lookup
	uidToCID map[string]uint32  // UID -> CID for fast delete lookup
}

// NewVMWatcher creates a new VM watcher
func NewVMWatcher(ctx context.Context, dynamicClient dynamic.Interface) (*VMWatcher, error) {
	ctx, cancel := context.WithCancel(ctx)

	// Create informer for VirtualMachineInstance using dynamic client
	informer := cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
				return dynamicClient.Resource(vmiGVR).List(ctx, options)
			},
			WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
				return dynamicClient.Resource(vmiGVR).Watch(ctx, options)
			},
		},
		&unstructured.Unstructured{},
		time.Minute*10, // Resync every 10 minutes
		cache.Indexers{},
	)

	watcher := &VMWatcher{
		ctx:           ctx,
		cancel:        cancel,
		dynamicClient: dynamicClient,
		informer:      informer,
		stopper:       concurrency.NewStopper(),
		vms:           make(map[uint32]*VMInfo),
		cidToUID:      make(map[uint32]string),
		uidToCID:      make(map[string]uint32),
	}

	// Add event handlers
	informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj any) {
			if vmi, err := watcher.convertToVMI(obj); err != nil {
				log.Errorf("Failed to convert object to VMI in Add handler: %v", err)
			} else {
				watcher.handleVMIAdd(vmi)
			}
		},
		UpdateFunc: func(oldObj, newObj any) {
			if vmi, err := watcher.convertToVMI(newObj); err != nil {
				log.Errorf("Failed to convert object to VMI in Update handler: %v", err)
			} else {
				watcher.handleVMIUpdate(vmi)
			}
		},
		DeleteFunc: func(obj any) {
			if vmi, err := watcher.convertToVMI(obj); err != nil {
				log.Errorf("Failed to convert object to VMI in Delete handler: %v", err)
			} else {
				watcher.handleVMIDelete(vmi)
			}
		},
	})

	return watcher, nil
}

// convertToVMI converts an unstructured object to a VirtualMachineInstance
//
// This is the "boilerplate" cost of using dynamic client - we have to convert
// from unstructured.Unstructured to the typed kubevirtv1.VirtualMachineInstance
// at runtime. A typed client would give us the typed object directly at compile time.
func (w *VMWatcher) convertToVMI(obj any) (*kubevirtv1.VirtualMachineInstance, error) {
	// Handle tombstone objects (objects that were deleted)
	if tombstone, ok := obj.(cache.DeletedFinalStateUnknown); ok {
		obj = tombstone.Obj
	}

	// Convert unstructured to VMI - this is where runtime type safety happens
	unstructuredObj, ok := obj.(*unstructured.Unstructured)
	if !ok {
		return nil, fmt.Errorf("expected *unstructured.Unstructured, got %T", obj)
	}

	var vmi kubevirtv1.VirtualMachineInstance
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(unstructuredObj.Object, &vmi); err != nil {
		return nil, fmt.Errorf("failed to convert unstructured to VMI: %w", err)
	}

	return &vmi, nil
}

// Start starts the VM watcher
func (w *VMWatcher) Start() error {
	log.Info("Starting VM watcher...")

	// Start informer
	go w.informer.Run(w.ctx.Done())

	// Wait for cache sync
	if !cache.WaitForCacheSync(w.ctx.Done(), w.informer.HasSynced) {
		return errors.New("failed to sync VM informer cache")
	}

	log.Info("VM watcher started and cache synced")
	return nil
}

// Stop stops the VM watcher
func (w *VMWatcher) Stop() error {
	log.Info("Stopping VM watcher...")
	w.cancel()
	w.stopper.Client().Stop()
	return nil
}

// GetVMByCID returns VM information for a given VSOCK CID
func (w *VMWatcher) GetVMByCID(cid uint32) (*VMInfo, bool) {
	w.vmsMutex.RLock()
	defer w.vmsMutex.RUnlock()

	vm, exists := w.vms[cid]
	if !exists {
		return nil, false
	}

	// Return a copy to avoid race conditions
	return &VMInfo{
		Name:      vm.Name,
		Namespace: vm.Namespace,
		UID:       vm.UID,
		VSOCKCID:  vm.VSOCKCID,
		Labels:    copyLabels(vm.Labels),
	}, true
}

// ListVMs returns all tracked VMs
func (w *VMWatcher) ListVMs() []*VMInfo {
	w.vmsMutex.RLock()
	defer w.vmsMutex.RUnlock()

	vms := make([]*VMInfo, 0, len(w.vms))
	for _, vm := range w.vms {
		vms = append(vms, &VMInfo{
			Name:      vm.Name,
			Namespace: vm.Namespace,
			UID:       vm.UID,
			VSOCKCID:  vm.VSOCKCID,
			Labels:    copyLabels(vm.Labels),
		})
	}

	return vms
}

// handleVMIAdd handles VM addition events
func (w *VMWatcher) handleVMIAdd(vmi *kubevirtv1.VirtualMachineInstance) {
	if !isVSockEnabled(vmi) {
		return
	}

	if vmi.Status.VSOCKCID == nil {
		return
	}

	cid := *vmi.Status.VSOCKCID
	vmInfo := &VMInfo{
		Name:      vmi.Name,
		Namespace: vmi.Namespace,
		UID:       string(vmi.UID),
		VSOCKCID:  cid,
		Labels:    copyLabels(vmi.Labels),
	}

	w.vmsMutex.Lock()
	w.vms[cid] = vmInfo
	w.cidToUID[cid] = string(vmi.UID)
	w.uidToCID[string(vmi.UID)] = cid
	w.vmsMutex.Unlock()

	log.Infof("Tracking VM %s/%s with VSOCK CID %d", vmi.Namespace, vmi.Name, cid)
}

// handleVMIUpdate handles VM update events
func (w *VMWatcher) handleVMIUpdate(vmi *kubevirtv1.VirtualMachineInstance) {
	if !isVSockEnabled(vmi) {
		// VM no longer has VSOCK enabled, remove it if we were tracking it
		w.vmsMutex.Lock()
		uid := string(vmi.UID)
		if cid, exists := w.uidToCID[uid]; exists {
			delete(w.vms, cid)
			delete(w.cidToUID, cid)
			delete(w.uidToCID, uid)
			log.Infof("Stopped tracking VM %s/%s (VSOCK disabled)", vmi.Namespace, vmi.Name)
		}
		w.vmsMutex.Unlock()
		return
	}

	if vmi.Status.VSOCKCID == nil {
		return
	}

	cid := *vmi.Status.VSOCKCID
	vmInfo := &VMInfo{
		Name:      vmi.Name,
		Namespace: vmi.Namespace,
		UID:       string(vmi.UID),
		VSOCKCID:  cid,
		Labels:    copyLabels(vmi.Labels),
	}

	w.vmsMutex.Lock()
	w.vms[cid] = vmInfo
	w.cidToUID[cid] = string(vmi.UID)
	w.uidToCID[string(vmi.UID)] = cid
	w.vmsMutex.Unlock()

	log.Debugf("Updated VM %s/%s with VSOCK CID %d", vmi.Namespace, vmi.Name, cid)
}

// handleVMIDelete handles VM deletion events
func (w *VMWatcher) handleVMIDelete(vmi *kubevirtv1.VirtualMachineInstance) {
	w.vmsMutex.Lock()
	defer w.vmsMutex.Unlock()

	// Use UID to look up in case VSOCK CID was removed from status before deletion
	uid := string(vmi.UID)
	if cid, exists := w.uidToCID[uid]; exists {
		delete(w.vms, cid)
		delete(w.cidToUID, cid)
		delete(w.uidToCID, uid)
		log.Infof("Stopped tracking VM %s/%s (deleted)", vmi.Namespace, vmi.Name)
	}
}

// isVSockEnabled checks if VSOCK is enabled for a VM
func isVSockEnabled(vmi *kubevirtv1.VirtualMachineInstance) bool {
	if vmi.Spec.Domain.Devices.AutoattachVSOCK == nil {
		return false
	}
	return *vmi.Spec.Domain.Devices.AutoattachVSOCK
}

// copyLabels creates a copy of labels map
func copyLabels(labels map[string]string) map[string]string {
	if labels == nil {
		return nil
	}

	copy := make(map[string]string, len(labels))
	for k, v := range labels {
		copy[k] = v
	}
	return copy
}
