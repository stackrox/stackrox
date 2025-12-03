package fake

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"strings"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	kubeVirtV1 "kubevirt.io/api/core/v1"
)

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

// vmTemplatePool holds a fixed-size pool of VM/VMI templates
type vmTemplatePool struct {
	templates []*vmTemplate
}

type vmTemplate struct {
	index         int // Template index, used for deterministic UUID generation
	baseName      string
	baseNamespace string
	vsockCID      uint32
	guestOS       string
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
			index:         i,
			baseName:      fmt.Sprintf("vm-%d", i),
			baseNamespace: "default",
			vsockCID:      vsockBaseCID + uint32(i),
			guestOS:       guestOS,
		}
	}

	return pool
}

func (p *vmTemplatePool) getTemplate(idx int) *vmTemplate {
	if idx < 0 || idx >= len(p.templates) {
		return nil
	}
	return p.templates[idx]
}

func (p *vmTemplatePool) size() int {
	return len(p.templates)
}

func (t *vmTemplate) instantiate(iteration int) (*unstructured.Unstructured, *unstructured.Unstructured) {
	// Use deterministic UUID based on template index to match index report generation.
	// This ensures the VM UID in informer events matches the VM ID used in index reports.
	vmUID := types.UID(fakeVMUUID(t.index))
	vmName := fmt.Sprintf("%s-%d", t.baseName, iteration)

	vm := &kubeVirtV1.VirtualMachine{
		TypeMeta: metav1.TypeMeta{
			Kind:       "VirtualMachine",
			APIVersion: "kubevirt.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      vmName,
			Namespace: t.baseNamespace,
			UID:       vmUID,
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

	// VMI gets a unique UUID based on template index and iteration
	// Format: 00000000-0000-4000-9000-{6-digit-index}{6-digit-iteration}
	vmiUID := types.UID(fmt.Sprintf("00000000-0000-4000-9000-%06d%06d", t.index, iteration))
	vmiName := fmt.Sprintf("%s-%d-vmi", t.baseName, iteration)
	vsockCID := t.vsockCID
	vmi := &kubeVirtV1.VirtualMachineInstance{
		TypeMeta: metav1.TypeMeta{
			Kind:       "VirtualMachineInstance",
			APIVersion: "kubevirt.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      vmiName,
			Namespace: t.baseNamespace,
			UID:       vmiUID,
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

	vmObj := toUnstructuredVM(vm)
	vmiObj := toUnstructuredVMI(vmi)
	return vmObj, vmiObj
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
	unstructuredObj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(vm)
	if err != nil {
		log.Warnf("failed to convert VM %s to unstructured object: %v", vm.GetName(), err)
		return &unstructured.Unstructured{Object: map[string]interface{}{}}
	}
	return &unstructured.Unstructured{Object: unstructuredObj}
}

// toUnstructuredVMI converts a VirtualMachineInstance to unstructured.Unstructured
func toUnstructuredVMI(vmi *kubeVirtV1.VirtualMachineInstance) *unstructured.Unstructured {
	unstructuredObj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(vmi)
	if err != nil {
		log.Warnf("failed to convert VMI %s to unstructured object: %v", vmi.GetName(), err)
		return &unstructured.Unstructured{Object: map[string]interface{}{}}
	}
	obj := &unstructured.Unstructured{Object: unstructuredObj}
	normalizeVMIUnsignedFields(obj, vmi.Status)
	return obj
}

// normalizeVMIUnsignedFields rewrites known unsigned VMI status fields so DeepCopyJSONValue does not panic.
func normalizeVMIUnsignedFields(obj *unstructured.Unstructured, status kubeVirtV1.VirtualMachineInstanceStatus) {
	setNestedField(obj, jsonSafeUint64(status.RuntimeUser), "status", "runtimeUser")
	if status.VSOCKCID != nil {
		setNestedField(obj, int64(*status.VSOCKCID), "status", "vsockCID")
	}
}

func jsonSafeUint64(value uint64) int64 {
	if value > math.MaxInt64 {
		return math.MaxInt64
	}
	return int64(value)
}

type vmResourcesToBeManaged struct {
	workload VirtualMachineWorkload
	template *vmTemplate
	vm       *unstructured.Unstructured
	vmi      *unstructured.Unstructured
}

func (w *WorkloadManager) getVMResources(workload VirtualMachineWorkload, templateIdx int, templatePool *vmTemplatePool) *vmResourcesToBeManaged {
	template := templatePool.getTemplate(templateIdx)
	if template == nil {
		return nil
	}

	return &vmResourcesToBeManaged{
		workload: workload,
		template: template,
	}
}

// manageVirtualMachine manages repeated VM/VMI lifecycles for a single template.
func (w *WorkloadManager) manageVirtualMachine(ctx context.Context, resources *vmResourcesToBeManaged) {
	defer w.wg.Done()

	lifecycles := resources.workload.NumLifecycles
	for iteration := 0; lifecycles <= 0 || iteration < lifecycles; iteration++ {
		vm, vmi := resources.template.instantiate(iteration)
		resources.vm = vm
		resources.vmi = vmi

		if w.manageVirtualMachineLifecycleOnce(ctx, resources) {
			return
		}
	}
}

// manageVirtualMachineLifecycleOnce runs a single VM/VMI lifecycle.
// It returns true if the caller should stop spawning further lifecycles (e.g., context cancelled or setup failed).
func (w *WorkloadManager) manageVirtualMachineLifecycleOnce(ctx context.Context, resources *vmResourcesToBeManaged) bool {
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
		return true
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
			return true
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
			// Drop the fake tracker entries so unstructured DeepCopy payloads do not accumulate indefinitely.
			w.cleanupVMHistory(resources.vm.GetNamespace(), vmName, resources.vmi.GetName())
			return false
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
