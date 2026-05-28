package vmhelpers

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	coreV1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	k8sruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/dynamic"
	kubevirtv1 "kubevirt.io/api/core/v1"
)

// Defaults for VM resource requests and polling/logging during KubeVirt VM/VMI wait helpers.
const (
	defaultVMMemoryRequest = "2Gi"
	defaultVMCPUCores      = uint32(3)
	vmPollInterval         = 2 * time.Second
)

// vmGVR and vmiGVR are dynamic client resources for KubeVirt VirtualMachine and VirtualMachineInstance objects.
var (
	vmGVR = schema.GroupVersionResource{
		Group:    kubevirtv1.GroupVersion.Group,
		Version:  kubevirtv1.GroupVersion.Version,
		Resource: "virtualmachines",
	}
	vmiGVR = schema.GroupVersionResource{
		Group:    kubevirtv1.GroupVersion.Group,
		Version:  kubevirtv1.GroupVersion.Version,
		Resource: "virtualmachineinstances",
	}
)

// vmFromUnstructured converts an unstructured object returned by the dynamic client
// into a typed VirtualMachine.
func vmFromUnstructured(obj *unstructured.Unstructured) (*kubevirtv1.VirtualMachine, error) {
	var vm kubevirtv1.VirtualMachine
	if err := k8sruntime.DefaultUnstructuredConverter.FromUnstructured(obj.Object, &vm); err != nil {
		return nil, fmt.Errorf("decode VirtualMachine: %w", err)
	}
	return &vm, nil
}

// vmiFromUnstructured converts an unstructured object returned by the dynamic client
// into a typed VirtualMachineInstance.
func vmiFromUnstructured(obj *unstructured.Unstructured) (*kubevirtv1.VirtualMachineInstance, error) {
	var vmi kubevirtv1.VirtualMachineInstance
	if err := k8sruntime.DefaultUnstructuredConverter.FromUnstructured(obj.Object, &vmi); err != nil {
		return nil, fmt.Errorf("decode VirtualMachineInstance: %w", err)
	}
	return &vmi, nil
}

// VMRequest describes inputs for cloud-init rendering and VirtualMachine creation.
type VMRequest struct {
	Name         string
	Namespace    string
	Image        string
	GuestUser    string
	SSHPublicKey string
}

// renderCloudInit produces cloud-init user-data YAML for the given request.
// Callers must validate that GuestUser and SSHPublicKey are non-empty.
func renderCloudInit(req VMRequest) string {
	return fmt.Sprintf(`#cloud-config
users:
  - name: %q
    sudo: "ALL=(ALL) NOPASSWD:ALL"
    ssh_authorized_keys:
      - %q
`, req.GuestUser, req.SSHPublicKey)
}

// CreateVirtualMachine submits a KubeVirt VirtualMachine with a container disk and NoCloud user-data.
func CreateVirtualMachine(ctx context.Context, client dynamic.Interface, req VMRequest) error {
	if req.Name == "" || req.Namespace == "" || req.Image == "" || req.GuestUser == "" || req.SSHPublicKey == "" {
		return errors.New("VMRequest Name, Namespace, Image, GuestUser, and SSHPublicKey are required")
	}
	userData := renderCloudInit(req)
	runStrategy := kubevirtv1.RunStrategyAlways
	autoattachVSOCK := true
	vm := &kubevirtv1.VirtualMachine{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "kubevirt.io/v1",
			Kind:       "VirtualMachine",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      req.Name,
			Namespace: req.Namespace,
		},
		Spec: kubevirtv1.VirtualMachineSpec{
			RunStrategy: &runStrategy,
			Template: &kubevirtv1.VirtualMachineInstanceTemplateSpec{
				Spec: kubevirtv1.VirtualMachineInstanceSpec{
					Domain: kubevirtv1.DomainSpec{
						CPU: &kubevirtv1.CPU{
							Cores: defaultVMCPUCores,
						},
						Resources: kubevirtv1.ResourceRequirements{
							Requests: coreV1.ResourceList{
								coreV1.ResourceMemory: resource.MustParse(defaultVMMemoryRequest),
							},
						},
						Devices: kubevirtv1.Devices{
							AutoattachVSOCK: &autoattachVSOCK,
							Disks: []kubevirtv1.Disk{
								{
									Name: "containerdisk",
									DiskDevice: kubevirtv1.DiskDevice{
										Disk: &kubevirtv1.DiskTarget{Bus: kubevirtv1.DiskBusVirtio},
									},
								},
								{
									Name: "cloudinitdisk",
									DiskDevice: kubevirtv1.DiskDevice{
										Disk: &kubevirtv1.DiskTarget{Bus: kubevirtv1.DiskBusVirtio},
									},
								},
							},
						},
					},
					Volumes: []kubevirtv1.Volume{
						{
							Name: "containerdisk",
							VolumeSource: kubevirtv1.VolumeSource{
								ContainerDisk: &kubevirtv1.ContainerDiskSource{Image: req.Image},
							},
						},
						{
							Name: "cloudinitdisk",
							VolumeSource: kubevirtv1.VolumeSource{
								CloudInitNoCloud: &kubevirtv1.CloudInitNoCloudSource{
									UserData: userData,
								},
							},
						},
					},
				},
			},
		},
	}
	u, err := k8sruntime.DefaultUnstructuredConverter.ToUnstructured(vm)
	if err != nil {
		return fmt.Errorf("convert VirtualMachine to unstructured: %w", err)
	}
	_, err = client.Resource(vmGVR).Namespace(req.Namespace).Create(ctx, &unstructured.Unstructured{Object: u}, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("create VirtualMachine: %w", err)
	}
	return nil
}

// GetVMContainerDiskImage reads the container disk image from an existing VirtualMachine.
func GetVMContainerDiskImage(ctx context.Context, client dynamic.Interface, namespace, name string) (string, error) {
	obj, err := client.Resource(vmGVR).Namespace(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return "", fmt.Errorf("get VirtualMachine %s/%s: %w", namespace, name, err)
	}
	vm, err := vmFromUnstructured(obj)
	if err != nil {
		return "", fmt.Errorf("decode VirtualMachine %s/%s: %w", namespace, name, err)
	}
	if vm.Spec.Template == nil {
		return "", fmt.Errorf("VirtualMachine %s/%s has no spec.template", namespace, name)
	}
	for _, vol := range vm.Spec.Template.Spec.Volumes {
		if vol.Name == "containerdisk" && vol.ContainerDisk != nil && vol.ContainerDisk.Image != "" {
			return vol.ContainerDisk.Image, nil
		}
	}
	return "", fmt.Errorf("VirtualMachine %s/%s has no containerdisk volume", namespace, name)
}

// DeleteVirtualMachine removes a KubeVirt VirtualMachine by name.
func DeleteVirtualMachine(ctx context.Context, client dynamic.Interface, namespace, name string) error {
	err := client.Resource(vmGVR).Namespace(namespace).Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil && !apierrors.IsNotFound(err) {
		return fmt.Errorf("delete VirtualMachine: %w", err)
	}
	return nil
}

// WaitForVirtualMachineInstanceExists polls until the VMI object is present (same name as the VM).
func WaitForVirtualMachineInstanceExists(t testing.TB, ctx context.Context, client dynamic.Interface, namespace, name string) error {
	t.Helper()
	return pollKubeVirtCondition(t, ctx, vmPollInterval, fmt.Sprintf("wait VMI %s/%s exists", namespace, name),
		func(ctx context.Context, attempt int) (bool, string, error) {
			_, err := client.Resource(vmiGVR).Namespace(namespace).Get(ctx, name, metav1.GetOptions{})
			if err == nil {
				return true, "vmi now exists", nil
			}
			if apierrors.IsNotFound(err) {
				detail, termErr := handleVMINotFound(ctx, client, namespace, name, attempt, "VMI was not created")
				return false, detail, termErr
			}
			return false, "", fmt.Errorf("attempt %d: get VMI: %w", attempt, err)
		},
	)
}

// WaitForVirtualMachineInstanceRunning polls until the VMI reports Running phase.
func WaitForVirtualMachineInstanceRunning(t testing.TB, ctx context.Context, client dynamic.Interface, namespace, name string) error {
	t.Helper()
	return pollKubeVirtCondition(t, ctx, vmPollInterval, fmt.Sprintf("wait VMI %s/%s running", namespace, name),
		func(ctx context.Context, attempt int) (bool, string, error) {
			obj, err := client.Resource(vmiGVR).Namespace(namespace).Get(ctx, name, metav1.GetOptions{})
			if err != nil {
				if apierrors.IsNotFound(err) {
					detail, termErr := handleVMINotFound(ctx, client, namespace, name, attempt, "VMI failed before becoming Running")
					return false, detail, termErr
				}
				return false, "", fmt.Errorf("attempt %d: get VMI: %w", attempt, err)
			}
			vmi, err := vmiFromUnstructured(obj)
			if err != nil {
				return false, "", fmt.Errorf("attempt %d: decode VMI: %w", attempt, err)
			}
			detail := vmiPhaseDetail(vmi)
			switch vmi.Status.Phase {
			case kubevirtv1.Running:
				return true, detail, nil
			case kubevirtv1.Failed, kubevirtv1.Succeeded:
				return false, detail, fmt.Errorf("attempt %d: VMI reached terminal phase: %s", attempt, detail)
			default:
				vmDetail, terminal, _ := virtualMachineStatusDetail(ctx, client, namespace, name)
				if terminal {
					return false, detail + " " + vmDetail, fmt.Errorf("attempt %d: unrecoverable VM error: %s", attempt, vmDetail)
				}
				if vmDetail != "" {
					detail += " " + vmDetail
				}
				return false, detail, nil
			}
		},
	)
}

// terminalPrintableStatuses contains VM printableStatus values that indicate
// unrecoverable errors where further polling is pointless.
var terminalPrintableStatuses = []string{
	"ImagePullBackOff",
	"ErrImagePull",
	"InvalidImageName",
	"RegistryUnavailable",
	"ImageInspectError",
	"ErrImageNeverPull",
	"CrashLoopBackOff",
	"ErrorUnschedulable",
}

// vmPrintableStatus inspects a VirtualMachine and returns logging detail
// plus whether printableStatus matches a terminal pattern.
func vmPrintableStatus(vm *kubevirtv1.VirtualMachine) (string, bool) {
	ps := strings.TrimSpace(string(vm.Status.PrintableStatus))
	if ps == "" {
		return "", false
	}
	for _, terminal := range terminalPrintableStatuses {
		if strings.Contains(ps, terminal) {
			return fmt.Sprintf("vm printableStatus=%q (terminal)", ps), true
		}
	}
	return fmt.Sprintf("vm printableStatus=%q", ps), false
}

// GetVMINodeName returns the Kubernetes node name that hosts the given VirtualMachineInstance.
func GetVMINodeName(ctx context.Context, client dynamic.Interface, namespace, name string) (string, error) {
	obj, err := client.Resource(vmiGVR).Namespace(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return "", fmt.Errorf("get VMI %s/%s: %w", namespace, name, err)
	}
	vmi, err := vmiFromUnstructured(obj)
	if err != nil {
		return "", fmt.Errorf("decode VMI %s/%s: %w", namespace, name, err)
	}
	if vmi.Status.NodeName == "" {
		return "", fmt.Errorf("VMI %s/%s has no status.nodeName (phase=%s)", namespace, name, vmiPhaseDetail(vmi))
	}
	return vmi.Status.NodeName, nil
}

// vmiPhaseDetail summarizes VMI phase and Ready condition fields for poll attempt logs.
func vmiPhaseDetail(vmi *kubevirtv1.VirtualMachineInstance) string {
	parts := make([]string, 0, 2)
	if vmi.Status.Phase != "" {
		parts = append(parts, fmt.Sprintf("phase=%q", vmi.Status.Phase))
	}
	for _, cond := range vmi.Status.Conditions {
		if cond.Type != kubevirtv1.VirtualMachineInstanceReady {
			continue
		}
		part := fmt.Sprintf("ready.status=%q", cond.Status)
		if reason := strings.TrimSpace(cond.Reason); reason != "" {
			part += fmt.Sprintf(" ready.reason=%q", reason)
		}
		if message := strings.TrimSpace(cond.Message); message != "" {
			part += fmt.Sprintf(" ready.message=%q", message)
		}
		parts = append(parts, part)
		break
	}
	if len(parts) == 0 {
		return "vmi status unavailable"
	}
	return strings.Join(parts, " ")
}

// WaitForVirtualMachineDeleted polls until the VirtualMachine object is gone.
func WaitForVirtualMachineDeleted(t testing.TB, ctx context.Context, client dynamic.Interface, namespace, name string) error {
	t.Helper()
	return pollKubeVirtCondition(t, ctx, vmPollInterval, fmt.Sprintf("wait VM %s/%s deleted", namespace, name),
		func(ctx context.Context, attempt int) (bool, string, error) {
			_, err := client.Resource(vmGVR).Namespace(namespace).Get(ctx, name, metav1.GetOptions{})
			if apierrors.IsNotFound(err) {
				return true, "vm no longer found", nil
			}
			if err != nil {
				return false, "", fmt.Errorf("attempt %d: get VM: %w", attempt, err)
			}
			return false, "vm still exists", nil
		},
	)
}

// --- Shared poll infrastructure ---

// pollKubeVirtCondition runs a poll loop with attempt counting, periodic logging, and
// consistent timeout error wrapping. check receives the current attempt number and
// returns (done, detail-for-logging, error). desc is used in log lines and errors.
func pollKubeVirtCondition(t testing.TB, ctx context.Context, interval time.Duration, desc string,
	check func(ctx context.Context, attempt int) (done bool, detail string, err error),
) error {
	t.Helper()
	attempts := 0
	lastDetail := ""
	err := wait.PollUntilContextCancel(ctx, interval, true, func(ctx context.Context) (bool, error) {
		attempts++
		done, detail, err := check(ctx, attempts)
		if detail != "" {
			lastDetail = detail
		}
		logWaitAttempt(t, desc, attempts, detail)
		return done, err
	})
	if err == nil {
		return nil
	}
	if lastDetail != "" {
		return fmt.Errorf("%s failed after %d poll(s): %w (last detail: %s)", desc, attempts, err, lastDetail)
	}
	return fmt.Errorf("%s failed after %d poll(s): %w", desc, attempts, err)
}

// handleVMINotFound is used inside poll callbacks when the VMI doesn't exist yet.
// It reads the parent VM status for diagnostics and returns a terminal error if the
// VM itself is in a failure state (e.g. image pull error).
func handleVMINotFound(ctx context.Context, client dynamic.Interface, namespace, name string, attempt int, terminalPrefix string) (detail string, err error) {
	vmDetail, terminalFailure, detailErr := virtualMachineStatusDetail(ctx, client, namespace, name)
	if detailErr != nil {
		return "", fmt.Errorf("attempt %d: read VM status: %w", attempt, detailErr)
	}
	detail = fmt.Sprintf("vmi not found (%s)", vmDetail)
	if terminalFailure {
		return detail, fmt.Errorf("attempt %d: %s: %s", attempt, terminalPrefix, vmDetail)
	}
	return detail, nil
}

// virtualMachineStatusDetail loads the VM and returns status text for wait-loop logging; terminalFailure stops polling.
func virtualMachineStatusDetail(ctx context.Context, client dynamic.Interface, namespace, name string) (detail string, terminalFailure bool, err error) {
	obj, err := client.Resource(vmGVR).Namespace(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			return "virtual machine object not found", false, nil
		}
		return "", false, err
	}
	vm, err := vmFromUnstructured(obj)
	if err != nil {
		return "", false, err
	}
	if detail, terminal := vmFailureConditionDetail(vm); terminal {
		return detail, true, nil
	}
	if detail, terminal := vmPrintableStatus(vm); detail != "" {
		return detail, terminal, nil
	}
	return "vm status unavailable", false, nil
}

// vmFailureConditionDetail returns a printable detail if the VM status has a True Failure condition.
func vmFailureConditionDetail(vm *kubevirtv1.VirtualMachine) (string, bool) {
	for _, cond := range vm.Status.Conditions {
		if cond.Type != kubevirtv1.VirtualMachineFailure || cond.Status != coreV1.ConditionTrue {
			continue
		}
		reason := strings.TrimSpace(cond.Reason)
		msg := strings.TrimSpace(cond.Message)
		switch {
		case reason != "" && msg != "":
			return fmt.Sprintf("failure condition reason=%q message=%q", reason, msg), true
		case reason != "":
			return fmt.Sprintf("failure condition reason=%q", reason), true
		case msg != "":
			return fmt.Sprintf("failure condition message=%q", msg), true
		default:
			return "failure condition present", true
		}
	}
	return "", false
}
