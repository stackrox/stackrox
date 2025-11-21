package fake

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"strings"
	"time"

	"github.com/stackrox/rox/pkg/sync"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	kubeVirtV1 "kubevirt.io/api/core/v1"
)

// setJSONSafeVSOCKCID normalizes the vsock CID so the fake dynamic client can deep-copy the object.
// client-go's DeepCopyJSONValue only supports JSON-compatible scalars; unsigned ints (uint32/uint64)
// cause a panic. Converting to int64 before enqueueing keeps the fake informer pipeline happy.
func setJSONSafeVSOCKCID(obj *unstructured.Unstructured, vsockCID uint32) {
	setNestedField(obj, int64(vsockCID), "status", "vsockCID")
}

func setNestedField(obj *unstructured.Unstructured, value interface{}, fields ...string) {
	if err := unstructured.SetNestedField(obj.Object, value, fields...); err != nil {
		log.Warnf("failed to set nested field %s: %v", strings.Join(fields, "."), err)
	}
}

const (
	defaultVMLifecycleDuration = 30 * time.Second
	defaultVMUpdateInterval    = 10 * time.Second
)

func validateVMWorkload(workload VirtualMachineWorkload) VirtualMachineWorkload {
	if workload.LifecycleDuration <= 0 {
		log.Warnf("virtualMachineWorkload.lifecycleDuration not set or <= 0; defaulting to %s", defaultVMLifecycleDuration)
		workload.LifecycleDuration = defaultVMLifecycleDuration
	}
	if workload.UpdateInterval <= 0 {
		log.Warnf("virtualMachineWorkload.updateInterval not set or <= 0; defaulting to %s", defaultVMUpdateInterval)
		workload.UpdateInterval = defaultVMUpdateInterval
	}
	return workload
}

func sanitizeJSONNumbers(value interface{}) interface{} {
	switch v := value.(type) {
	case map[string]interface{}:
		for key, elem := range v {
			v[key] = sanitizeJSONNumbers(elem)
		}
		return v
	case []interface{}:
		for i, elem := range v {
			v[i] = sanitizeJSONNumbers(elem)
		}
		return v
	case uint:
		return int64(v)
	case uint8:
		return int64(v)
	case uint16:
		return int64(v)
	case uint32:
		return int64(v)
	case uint64:
		if v > uint64(math.MaxInt64) {
			return int64(math.MaxInt64)
		}
		return int64(v)
	case uintptr:
		val := uint64(v)
		if val > uint64(math.MaxInt64) {
			return int64(math.MaxInt64)
		}
		return int64(val)
	default:
		return value
	}
}

// vmTemplatePool holds a fixed-size pool of VM/VMI templates
type vmTemplatePool struct {
	templates []*vmTemplate
	lock      sync.RWMutex
}

type vmTemplate struct {
	// Base fields that remain constant
	baseName      string
	baseNamespace string
	baseUID       string
	vsockCID      uint32
	guestOS       string

	// Counter for generating unique variations
	variationCounter int
}

func newVMTemplatePool(poolSize int, guestOSPool []string, vsockBaseCID uint32) *vmTemplatePool {
	if poolSize <= 0 {
		poolSize = 10 // default pool size
	}

	pool := &vmTemplatePool{
		templates: make([]*vmTemplate, poolSize),
	}

	for i := 0; i < poolSize; i++ {
		guestOS := guestOSPool[rand.Intn(len(guestOSPool))]
		pool.templates[i] = &vmTemplate{
			baseName:      fmt.Sprintf("vm-%d", i),
			baseNamespace: "default",
			baseUID:       string(newUUID()),
			vsockCID:      vsockBaseCID + uint32(i),
			guestOS:       guestOS,
		}
	}

	return pool
}

func (p *vmTemplatePool) getTemplate(idx int) *vmTemplate {
	p.lock.RLock()
	defer p.lock.RUnlock()
	if idx < 0 || idx >= len(p.templates) {
		return nil
	}
	return p.templates[idx]
}

func (p *vmTemplatePool) size() int {
	p.lock.RLock()
	defer p.lock.RUnlock()
	return len(p.templates)
}

// createVMObject creates a VirtualMachine CRD object with randomized metadata
func (t *vmTemplate) createVMObject() *unstructured.Unstructured {
	t.variationCounter++

	// Randomize metadata to make each instance look unique
	vmUID := string(newUUID())
	vmName := fmt.Sprintf("%s-%d", t.baseName, t.variationCounter)

	vm := &kubeVirtV1.VirtualMachine{
		TypeMeta: metav1.TypeMeta{
			Kind:       "VirtualMachine",
			APIVersion: "kubevirt.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      vmName,
			Namespace: t.baseNamespace,
			UID:       types.UID(vmUID),
			CreationTimestamp: metav1.Time{
				Time: time.Now(),
			},
			Labels:      createRandMap(16, 3),
			Annotations: createRandMap(16, 3),
		},
		Status: kubeVirtV1.VirtualMachineStatus{
			PrintableStatus: kubeVirtV1.VirtualMachineStatusRunning,
		},
	}

	return toUnstructuredVM(vm)
}

// createVMIObject creates a VirtualMachineInstance CRD object with randomized metadata
func (t *vmTemplate) createVMIObject(vmUID types.UID, vmName string) *unstructured.Unstructured {
	t.variationCounter++

	vmiUID := string(newUUID())
	vmiName := fmt.Sprintf("%s-%d", t.baseName, t.variationCounter)

	vsockCID := t.vsockCID
	vmi := &kubeVirtV1.VirtualMachineInstance{
		TypeMeta: metav1.TypeMeta{
			Kind:       "VirtualMachineInstance",
			APIVersion: "kubevirt.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      vmiName,
			Namespace: t.baseNamespace,
			UID:       types.UID(vmiUID),
			CreationTimestamp: metav1.Time{
				Time: time.Now(),
			},
			Labels:      createRandMap(16, 3),
			Annotations: createRandMap(16, 3),
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: "kubevirt.io/v1",
					Kind:       "VirtualMachine",
					Name:       vmName,
					UID:        vmUID,
				},
			},
		},
		Status: kubeVirtV1.VirtualMachineInstanceStatus{
			Phase:    kubeVirtV1.Running,
			VSOCKCID: &vsockCID,
			GuestOSInfo: kubeVirtV1.VirtualMachineInstanceGuestOSInfo{
				Name: t.guestOS,
			},
		},
	}

	obj := toUnstructuredVMI(vmi)
	setJSONSafeVSOCKCID(obj, t.vsockCID)
	return obj
}

// updateVMObject updates VM metadata while keeping base structure
func (t *vmTemplate) updateVMObject(vm *unstructured.Unstructured) {
	vm.SetAnnotations(createRandMap(16, 3))
	vm.SetLabels(createRandMap(16, 3))

	// Randomly toggle running status
	if rand.Float32() < 0.3 {
		status, _, _ := unstructured.NestedString(vm.Object, "status", "printableStatus")
		if status == string(kubeVirtV1.VirtualMachineStatusRunning) {
			setNestedField(vm, string(kubeVirtV1.VirtualMachineStatusStopped), "status", "printableStatus")
		} else {
			setNestedField(vm, string(kubeVirtV1.VirtualMachineStatusRunning), "status", "printableStatus")
		}
	}
}

// updateVMIObject updates VMI metadata while keeping base structure
func (t *vmTemplate) updateVMIObject(vmi *unstructured.Unstructured) {
	vmi.SetAnnotations(createRandMap(16, 3))
	vmi.SetLabels(createRandMap(16, 3))

	// Randomly toggle phase
	if rand.Float32() < 0.3 {
		phase, _, _ := unstructured.NestedString(vmi.Object, "status", "phase")
		if phase == string(kubeVirtV1.Running) {
			setNestedField(vmi, string(kubeVirtV1.Scheduled), "status", "phase")
		} else {
			setNestedField(vmi, string(kubeVirtV1.Running), "status", "phase")
		}
	}
}

// toUnstructuredVM converts a VirtualMachine to unstructured.Unstructured
func toUnstructuredVM(vm *kubeVirtV1.VirtualMachine) *unstructured.Unstructured {
	unstructuredObj, _ := runtime.DefaultUnstructuredConverter.ToUnstructured(vm)
	sanitized := sanitizeJSONNumbers(unstructuredObj).(map[string]interface{})
	return &unstructured.Unstructured{Object: sanitized}
}

// toUnstructuredVMI converts a VirtualMachineInstance to unstructured.Unstructured
func toUnstructuredVMI(vmi *kubeVirtV1.VirtualMachineInstance) *unstructured.Unstructured {
	unstructuredObj, _ := runtime.DefaultUnstructuredConverter.ToUnstructured(vmi)
	sanitized := sanitizeJSONNumbers(unstructuredObj).(map[string]interface{})
	return &unstructured.Unstructured{Object: sanitized}
}

type vmResourcesToBeManaged struct {
	workload VirtualMachineWorkload
	template *vmTemplate
	vm       *unstructured.Unstructured
	vmi      *unstructured.Unstructured
}

func (w *WorkloadManager) getVMResources(workload VirtualMachineWorkload, templateIdx int, templatePool *vmTemplatePool) *vmResourcesToBeManaged {
	workload = validateVMWorkload(workload)

	template := templatePool.getTemplate(templateIdx)
	if template == nil {
		return nil
	}

	vm := template.createVMObject()
	vmi := template.createVMIObject(types.UID(vm.GetUID()), vm.GetName())

	return &vmResourcesToBeManaged{
		workload: workload,
		template: template,
		vm:       vm,
		vmi:      vmi,
	}
}

// manageVirtualMachine manages a single VM/VMI lifecycle
func (w *WorkloadManager) manageVirtualMachine(ctx context.Context, resources *vmResourcesToBeManaged) {
	defer w.wg.Done()

	// NumLifecycles+1 handles initial startup
	for count := 0; resources.workload.NumLifecycles == 0 || count < resources.workload.NumLifecycles+1; count++ {
		w.manageVirtualMachineLifecycle(ctx, resources)

		select {
		case <-ctx.Done():
			return
		default:
		}

		// Recreate resources with new UIDs/metadata
		vm := resources.template.createVMObject()
		vmi := resources.template.createVMIObject(types.UID(vm.GetUID()), vm.GetName())
		resources.vm = vm
		resources.vmi = vmi
	}
}

func (w *WorkloadManager) manageVirtualMachineLifecycle(ctx context.Context, resources *vmResourcesToBeManaged) {
	timer := newTimerWithJitter(resources.workload.LifecycleDuration/2 + time.Duration(rand.Int63n(int64(resources.workload.LifecycleDuration))))
	defer timer.Stop()

	updateNextUpdate := calculateDurationWithJitter(resources.workload.UpdateInterval)

	vmGVR := schema.GroupVersionResource{
		Group:    "kubevirt.io",
		Version:  "v1",
		Resource: "virtualmachines",
	}
	vmiGVR := schema.GroupVersionResource{
		Group:    "kubevirt.io",
		Version:  "v1",
		Resource: "virtualmachineinstances",
	}

	vmClient := w.client.Dynamic().Resource(vmGVR).Namespace(resources.vm.GetNamespace())
	vmiClient := w.client.Dynamic().Resource(vmiGVR).Namespace(resources.vmi.GetNamespace())

	// Create initial resources
	vmUID := resources.vm.GetUID()
	vmName := resources.vm.GetName()
	if _, err := vmClient.Create(ctx, resources.vm, metav1.CreateOptions{}); err != nil {
		log.Errorf("error creating VirtualMachine: %v", err)
		return
	}
	w.writeID(virtualMachinePrefix, vmUID)

	if _, err := vmiClient.Create(ctx, resources.vmi, metav1.CreateOptions{}); err != nil {
		log.Errorf("error creating VirtualMachineInstance: %v", err)
		// Continue even if VMI creation fails
	} else {
		w.writeID(vmiPrefix, resources.vmi.GetUID())
	}

	for {
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
			// Delete resources
			if err := vmiClient.Delete(ctx, resources.vmi.GetName(), metav1.DeleteOptions{}); err != nil {
				log.Debugf("error deleting VirtualMachineInstance (may not exist): %v", err)
			} else {
				w.deleteID(vmiPrefix, resources.vmi.GetUID())
			}

			if err := vmClient.Delete(ctx, vmName, metav1.DeleteOptions{}); err != nil {
				log.Debugf("error deleting VirtualMachine (may not exist): %v", err)
			} else {
				w.deleteID(virtualMachinePrefix, vmUID)
			}
			return
		case <-time.After(updateNextUpdate):
			updateNextUpdate = calculateDurationWithJitter(resources.workload.UpdateInterval)

			// Update VM metadata
			resources.template.updateVMObject(resources.vm)
			if _, err := vmClient.Update(ctx, resources.vm, metav1.UpdateOptions{}); err != nil {
				log.Debugf("error updating VirtualMachine: %v", err)
			}

			// Update VMI metadata
			resources.template.updateVMIObject(resources.vmi)
			if _, err := vmiClient.Update(ctx, resources.vmi, metav1.UpdateOptions{}); err != nil {
				log.Debugf("error updating VirtualMachineInstance: %v", err)
			}
		}
	}
}
