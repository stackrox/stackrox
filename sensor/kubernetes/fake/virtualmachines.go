package fake

import (
	"context"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/stackrox/rox/pkg/fixtures/vmindexreport"
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
	defaultVMLifecycleDuration = 30 * time.Minute
	defaultVMUpdateInterval    = 3 * time.Minute
	// initialReportMaxDeviation is the maximum relative deviation applied to the initial report delay.
	// With 0.2 (20%), a 30s delay will vary uniformly between 24s-36s.
	initialReportMaxDeviation = 0.2
	// reportIntervalMaxDeviation is the maximum relative deviation applied to report intervals.
	// With 0.05 (5%), a 60s interval will vary uniformly between 57s-63s.
	reportIntervalMaxDeviation = 0.05
	// lifecycleMaxDeviation is the maximum relative deviation applied to VM lifecycle duration.
	// With 0.5 (50%), a 60s lifecycle will vary uniformly between 30s-90s.
	lifecycleMaxDeviation = 0.5
	vmBaseNamespace       = "default"
)

// Centralized GVR definitions for KubeVirt resources
var (
	vmGVR = schema.GroupVersionResource{
		Group:    "kubevirt.io",
		Version:  "v1",
		Resource: "virtualmachines",
	}
	vmiGVR = schema.GroupVersionResource{
		Group:    "kubevirt.io",
		Version:  "v1",
		Resource: "virtualmachineinstances",
	}
	// defaultGuestOSPool is an arbitrary list of OS names.
	// They do not indicate any support of ACS'es roxagent for the given OS.
	// The values does not matter - we could have used any strings.
	defaultGuestOSPool = []string{"rhel7", "rhel8", "rhel9", "rhel10", "fedora", "centos", "ubuntu", "debian"}
)

func validateVMWorkload(workload VirtualMachineWorkload) (VirtualMachineWorkload, error) {
	// Skip validation and defaults if workload is disabled (poolSize=0)
	if workload.PoolSize <= 0 {
		return workload, nil
	}
	if workload.LifecycleDuration <= 0 {
		workload.LifecycleDuration = defaultVMLifecycleDuration
		return workload, fmt.Errorf("virtualMachineWorkload.lifecycleDuration not set or <= 0; defaulting to %s", defaultVMLifecycleDuration)
	}
	if workload.UpdateInterval <= 0 {
		workload.UpdateInterval = defaultVMUpdateInterval
		return workload, fmt.Errorf("virtualMachineWorkload.updateInterval not set or <= 0; defaulting to %s", defaultVMUpdateInterval)
	}

	// Sanity check timing relationships
	lowerBoundVMLifetime := time.Duration(float64(workload.LifecycleDuration) * (1 - lifecycleMaxDeviation))
	upperBoundVMLifetime := time.Duration(float64(workload.LifecycleDuration) * (1 + lifecycleMaxDeviation))
	lifecycleText := fmt.Sprintf("The VM will live for a random duration between %s and %s", lowerBoundVMLifetime, upperBoundVMLifetime)
	causeText := func(param string, value time.Duration) string {
		return fmt.Sprintf("Setting %q=%s", param, value)
	}
	actionText := func(param string) string {
		return fmt.Sprintf("Lower the value of %q or increase the 'lifecycleDuration'.", param)
	}

	// Check timing: interval should be < lowerBound for guaranteed firing
	// - interval < lowerBound: OK (fires before shortest-lived VM dies)
	// - interval in [lowerBound, upperBound]: some VMs may miss it
	// - interval > upperBound: no VM will ever see it (all die first)
	if workload.UpdateInterval > upperBoundVMLifetime {
		return workload, fmt.Errorf("%s. %s causes none of the VMs to ever receive an update. %s",
			lifecycleText, causeText("updateInterval", workload.UpdateInterval), actionText("updateInterval"))
	} else if workload.UpdateInterval > lowerBoundVMLifetime {
		return workload, fmt.Errorf("%s. %s may cause some VMs to never receive an update. %s",
			lifecycleText, causeText("updateInterval", workload.UpdateInterval), actionText("updateInterval"))
	}
	if workload.ReportInterval > 0 {
		if workload.ReportInterval > upperBoundVMLifetime {
			return workload, fmt.Errorf("%s. %s causes the workload to never send any index reports. %s",
				lifecycleText, causeText("reportInterval", workload.ReportInterval), actionText("reportInterval"))
		}
		if workload.ReportInterval > lowerBoundVMLifetime {
			return workload, fmt.Errorf("%s. %s may cause some VMs to never send any index reports. %s",
				lifecycleText, causeText("reportInterval", workload.ReportInterval), actionText("reportInterval"))
		}
		if workload.InitialReportDelay > upperBoundVMLifetime {
			return workload, fmt.Errorf("%s. %s causes the workload to never send any index reports. %s",
				lifecycleText, causeText("initialReportDelay", workload.InitialReportDelay), actionText("initialReportDelay"))
		}
		if workload.InitialReportDelay > lowerBoundVMLifetime {
			return workload, fmt.Errorf("%s. %s may cause some VMs to never send any index reports. %s",
				lifecycleText, causeText("initialReportDelay", workload.InitialReportDelay), actionText("initialReportDelay"))
		}
	}
	return workload, nil
}

func getRandomVMPair(vsockCID uint32, guestOSes []string) (*unstructured.Unstructured, *unstructured.Unstructured) {
	// Use deterministic UUID based on template index to match index report generation.
	// This ensures the VM UID in informer events matches the VM ID used in index reports.
	vmUID := types.UID(fakeVMUUID(int(vsockCID)))
	vmName := fmt.Sprintf("%s-%d", "vm", vsockCID)
	os := guestOSes[int(vsockCID)%len(guestOSes)]

	vm := &kubeVirtV1.VirtualMachine{
		TypeMeta: metav1.TypeMeta{
			Kind:       "VirtualMachine",
			APIVersion: "kubevirt.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      vmName,
			Namespace: vmBaseNamespace,
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
	// Format: 00000000-0000-4000-9000-{12-digit-vsockCID}
	vmiUID := types.UID(fmt.Sprintf("00000000-0000-4000-9000-%012d", vsockCID))
	vmiName := fmt.Sprintf("vm-%d-vmi", vsockCID)
	vmi := &kubeVirtV1.VirtualMachineInstance{
		TypeMeta: metav1.TypeMeta{
			Kind:       "VirtualMachineInstance",
			APIVersion: "kubevirt.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      vmiName,
			Namespace: vmBaseNamespace,
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
				Name: os,
			},
		},
	}

	vmObj := toUnstructuredVM(vm)
	vmiObj := toUnstructuredVMI(vmi)
	return vmObj, vmiObj
}

// randomizeAnnotationsLabels updates object's metadata while keeping base structure
func randomizeAnnotationsLabels(vm *unstructured.Unstructured) {
	vm.SetAnnotations(createRandMap(16, 3))
	vm.SetLabels(createRandMap(16, 3))
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

// manageVirtualMachine manages repeated VM/VMI lifecycles for a single template.
func (w *WorkloadManager) manageVirtualMachine(
	ctx context.Context,
	workload VirtualMachineWorkload,
	vsockCID uint32,
	reportGen *vmindexreport.Generator,
) {
	defer w.wg.Done()

	lifecycles := workload.NumLifecycles
	for iteration := 0; lifecycles <= 0 || iteration < lifecycles; iteration++ {
		if ctx.Err() != nil {
			return
		}
		vm, vmi := getRandomVMPair(vsockCID, defaultGuestOSPool)
		w.runVMLifecycle(ctx, workload, vsockCID, vm, vmi, reportGen)
	}
}

// runVMLifecycle runs a single VM/VMI lifecycle.
// It blocks until the lifecycle ends (timer fires) or context is cancelled.
// If index reports are enabled (reportGen != nil), it sends index reports while the VM is alive.
func (w *WorkloadManager) runVMLifecycle(
	ctx context.Context,
	workload VirtualMachineWorkload,
	vsockCID uint32,
	vm, vmi *unstructured.Unstructured,
	reportGen *vmindexreport.Generator,
) {
	lifecycleTimer := newTimerWithJitter(randomizedInterval(workload.LifecycleDuration, lifecycleMaxDeviation))
	defer lifecycleTimer.Stop()

	updateTicker := time.NewTicker(calculateDurationWithJitter(workload.UpdateInterval))
	defer updateTicker.Stop()

	vmClient := w.client.Dynamic().Resource(vmGVR).Namespace(vm.GetNamespace())
	vmiClient := w.client.Dynamic().Resource(vmiGVR).Namespace(vmi.GetNamespace())

	// Create initial resources
	vmName := vm.GetName()
	if _, err := vmClient.Create(ctx, vm, metav1.CreateOptions{}); err != nil {
		log.Errorf("error creating VirtualMachine: %v", err)
		return
	}

	if _, err := vmiClient.Create(ctx, vmi, metav1.CreateOptions{}); err != nil {
		log.Errorf("error creating VirtualMachineInstance: %v", err)
		// Continue even if VMI creation fails
	}

	// Start index report generation if enabled (runs while VM is alive)
	var reportCancel context.CancelFunc
	if reportGen != nil && workload.ReportInterval > 0 {
		var reportCtx context.Context
		reportCtx, reportCancel = context.WithCancel(ctx)
		go w.sendIndexReportsWhileAlive(
			reportCtx,
			reportGen,
			vsockCID,
			workload.ReportInterval,
			workload.InitialReportDelay,
		)
	}

	// Ensure report generation stops when this function exits
	defer func() {
		if reportCancel != nil {
			reportCancel()
		}
	}()

	for {
		select {
		case <-ctx.Done():
			return
		case <-lifecycleTimer.C:
			// Delete resources
			if err := vmiClient.Delete(ctx, vmi.GetName(), metav1.DeleteOptions{}); err != nil {
				log.Debugf("error deleting VirtualMachineInstance (may not exist): %v", err)
			}

			if err := vmClient.Delete(ctx, vmName, metav1.DeleteOptions{}); err != nil {
				log.Debugf("error deleting VirtualMachine (may not exist): %v", err)
			}
			// Drop the fake tracker entries so unstructured DeepCopy payloads do not accumulate indefinitely.
			w.cleanupVMHistory(vm.GetNamespace(), vmName, vmi.GetName())
			return
		case <-updateTicker.C:
			// Reset ticker with jitter for next update
			updateTicker.Reset(calculateDurationWithJitter(workload.UpdateInterval))

			// Update VM metadata
			randomizeAnnotationsLabels(vm)
			if _, err := vmClient.Update(ctx, vm, metav1.UpdateOptions{}); err != nil {
				log.Debugf("error updating VirtualMachine: %v", err)
			}

			// Update VMI metadata
			randomizeAnnotationsLabels(vmi)
			if _, err := vmiClient.Update(ctx, vmi, metav1.UpdateOptions{}); err != nil {
				log.Debugf("error updating VirtualMachineInstance: %v", err)
			}
		}
	}
}

// sendIndexReportsWhileAlive sends index reports for a VM at the configured interval.
// It waits for prerequisites (handler, store, central) before starting, then runs
// until the context is cancelled (when the VM lifecycle ends).
func (w *WorkloadManager) sendIndexReportsWhileAlive(
	ctx context.Context,
	reportGen *vmindexreport.Generator,
	vsockCID uint32,
	interval time.Duration,
	initialDelay time.Duration,
) {
	// Wait for all prerequisites before sending reports
	if !w.vmPrerequisitesReady.Wait(ctx) {
		log.Debugf("Prerequisites not ready to start sending fake index reports")
		return
	}

	log.Debugf("Starting index report generation for a VM (vsockCID=%d, interval=%s, initialDelay=%s)", vsockCID, interval, initialDelay)

	reportTicker := time.NewTicker(randomizedInterval(interval, reportIntervalMaxDeviation))
	if initialDelay > 0 {
		firstPeriod := randomizedInterval(initialDelay, initialReportMaxDeviation)
		reportTicker.Reset(firstPeriod)
	} else {
		w.sendOneIndexReport(ctx, reportGen, vsockCID)
	}
	defer reportTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Debugf("Stopping index report generation for VM with vsockCID %d (lifecycle ended)", vsockCID)
			return
		case <-reportTicker.C:
			reportTicker.Reset(randomizedInterval(interval, reportIntervalMaxDeviation))
			w.sendOneIndexReport(ctx, reportGen, vsockCID)
		}
	}
}

// sendOneIndexReport generates and sends a single index report for a VM.
func (w *WorkloadManager) sendOneIndexReport(
	ctx context.Context,
	reportGen *vmindexreport.Generator,
	vsockCID uint32,
) {
	if ctx.Err() != nil {
		return
	}

	report := reportGen.GenerateV1IndexReport(vsockCID)

	if w.vmIndexReportHandler == nil {
		log.Debugf("VM index report handler not set, skipping report for VM %d", vsockCID)
		return
	}

	if err := w.vmIndexReportHandler.Send(ctx, report); err != nil {
		// Don't log errors during shutdown
		if ctx.Err() == nil {
			log.Debugf("Failed to send index report for VM %d: %v", vsockCID, err)
		}
	}
}
