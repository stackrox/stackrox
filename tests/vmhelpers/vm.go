package vmhelpers

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"testing"
	"text/template"
	"time"

	vmscanning "github.com/stackrox/rox/tests/testdata/vm-scanning"

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
	defaultVMCPUCores      = int64(3)
	vmPollInterval         = 2 * time.Second
	vmWaitLogEveryAttempts = 5
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

// VMRequest describes inputs for cloud-init rendering and VirtualMachine creation.
type VMRequest struct {
	Name         string
	Namespace    string
	Image        string
	GuestUser    string
	SSHPublicKey string
}

// RenderCloudInit expands the embedded cloud-init template using req.
func RenderCloudInit(req VMRequest) ([]byte, error) {
	if err := validateCloudInitRequest(req); err != nil {
		return nil, err
	}
	tmpl, err := template.New("cloud-init").Funcs(template.FuncMap{
		"yamlQuote": strconv.Quote,
	}).Parse(string(vmscanning.CloudInitUserDataTemplate))
	if err != nil {
		return nil, fmt.Errorf("parse cloud-init template: %w", err)
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, req); err != nil {
		return nil, fmt.Errorf("execute cloud-init template: %w", err)
	}
	return buf.Bytes(), nil
}

// validateCloudInitRequest ensures GuestUser and SSHPublicKey are set for cloud-init rendering.
func validateCloudInitRequest(req VMRequest) error {
	if req.GuestUser == "" {
		return errors.New("VMRequest GuestUser is required")
	}
	if req.SSHPublicKey == "" {
		return errors.New("VMRequest SSHPublicKey is required")
	}
	return nil
}

// CreateVirtualMachine submits a KubeVirt VirtualMachine with a container disk and NoCloud user-data.
func CreateVirtualMachine(ctx context.Context, client dynamic.Interface, req VMRequest) error {
	if req.Name == "" || req.Namespace == "" || req.Image == "" {
		return errors.New("VMRequest Name, Namespace, and Image are required")
	}
	userData, err := RenderCloudInit(req)
	if err != nil {
		return err
	}
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
									UserData: string(userData),
								},
							},
						},
					},
				},
			},
		},
	}
	vm.Spec.Template.Spec.Domain.Resources.Requests = coreV1.ResourceList{
		coreV1.ResourceMemory: resource.MustParse(defaultVMMemoryRequest),
	}
	u, err := k8sruntime.DefaultUnstructuredConverter.ToUnstructured(vm)
	if err != nil {
		return fmt.Errorf("convert VirtualMachine to unstructured: %w", err)
	}
	if err := unstructured.SetNestedField(u, defaultVMCPUCores, "spec", "template", "spec", "domain", "cpu", "cores"); err != nil {
		return fmt.Errorf("set vm cpu cores in manifest: %w", err)
	}
	_, err = client.Resource(vmGVR).Namespace(req.Namespace).Create(ctx, &unstructured.Unstructured{Object: u}, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("create VirtualMachine: %w", err)
	}
	return nil
}

// GetVMContainerDiskImage reads the container disk image from an existing VirtualMachine.
func GetVMContainerDiskImage(ctx context.Context, client dynamic.Interface, namespace, name string) (string, error) {
	vm, err := client.Resource(vmGVR).Namespace(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return "", fmt.Errorf("get VirtualMachine %s/%s: %w", namespace, name, err)
	}
	volumes, found, err := unstructured.NestedSlice(vm.Object, "spec", "template", "spec", "volumes")
	if err != nil || !found {
		return "", fmt.Errorf("VirtualMachine %s/%s has no spec.template.spec.volumes", namespace, name)
	}
	for _, raw := range volumes {
		vol, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		n, _ := vol["name"].(string)
		if n != "containerdisk" {
			continue
		}
		image, _, _ := unstructured.NestedString(vol, "containerDisk", "image")
		if image != "" {
			return image, nil
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

// vmFailureConditionDetail returns a printable detail if the VM status has a True Failure condition.
func vmFailureConditionDetail(vm *unstructured.Unstructured) (string, bool) {
	conds, found, err := unstructured.NestedSlice(vm.Object, "status", "conditions")
	if err != nil || !found {
		return "", false
	}
	for _, raw := range conds {
		cond, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		typ, _ := cond["type"].(string)
		if !strings.EqualFold(strings.TrimSpace(typ), "Failure") {
			continue
		}
		status, _ := cond["status"].(string)
		if !strings.EqualFold(strings.TrimSpace(status), "True") {
			continue
		}
		reason, _ := cond["reason"].(string)
		msg, _ := cond["message"].(string)
		reason = strings.TrimSpace(reason)
		msg = strings.TrimSpace(msg)
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

// virtualMachineStatusDetail loads the VM and returns status text for wait-loop logging; terminalFailure stops polling.
func virtualMachineStatusDetail(ctx context.Context, client dynamic.Interface, namespace, name string) (detail string, terminalFailure bool, err error) {
	vm, err := client.Resource(vmGVR).Namespace(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			return "virtual machine object not found", false, nil
		}
		return "", false, err
	}
	if detail, terminal := vmFailureConditionDetail(vm); terminal {
		return detail, true, nil
	}
	printableStatus, found, nestedErr := unstructured.NestedString(vm.Object, "status", "printableStatus")
	if nestedErr != nil {
		return "", false, nestedErr
	}
	if found && strings.TrimSpace(printableStatus) != "" {
		return fmt.Sprintf("vm printableStatus=%q", printableStatus), false, nil
	}
	return "vm status unavailable", false, nil
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

// vmPrintableStatusTerminal reads the VM printableStatus and returns it along
// with whether the status indicates a terminal (unrecoverable) failure.
func vmPrintableStatusTerminal(ctx context.Context, client dynamic.Interface, namespace, name string) (string, bool) {
	vm, err := client.Resource(vmGVR).Namespace(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return "", false
	}
	ps, found, _ := unstructured.NestedString(vm.Object, "status", "printableStatus")
	if !found || strings.TrimSpace(ps) == "" {
		return "", false
	}
	for _, terminal := range terminalPrintableStatuses {
		if strings.Contains(ps, terminal) {
			return fmt.Sprintf("vm printableStatus=%q (terminal)", ps), true
		}
	}
	return fmt.Sprintf("vm printableStatus=%q", ps), false
}

// vmiPhaseDetail summarizes VMI phase and Ready condition fields for poll attempt logs.
func vmiPhaseDetail(vmi *unstructured.Unstructured) string {
	parts := make([]string, 0, 2)
	phase, found, _ := unstructured.NestedString(vmi.Object, "status", "phase")
	if found && strings.TrimSpace(phase) != "" {
		parts = append(parts, fmt.Sprintf("phase=%q", phase))
	}
	conds, found, _ := unstructured.NestedSlice(vmi.Object, "status", "conditions")
	if found {
		for _, raw := range conds {
			cond, ok := raw.(map[string]any)
			if !ok {
				continue
			}
			typ, _ := cond["type"].(string)
			if !strings.EqualFold(strings.TrimSpace(typ), "Ready") {
				continue
			}
			status, _ := cond["status"].(string)
			reason, _ := cond["reason"].(string)
			message, _ := cond["message"].(string)
			part := fmt.Sprintf("ready.status=%q", strings.TrimSpace(status))
			if strings.TrimSpace(reason) != "" {
				part += fmt.Sprintf(" ready.reason=%q", strings.TrimSpace(reason))
			}
			if strings.TrimSpace(message) != "" {
				part += fmt.Sprintf(" ready.message=%q", strings.TrimSpace(message))
			}
			parts = append(parts, part)
			break
		}
	}
	if len(parts) == 0 {
		return "vmi status unavailable"
	}
	return strings.Join(parts, " ")
}

// maxPollAttempts estimates how many PollUntilContextCancel iterations fit before ctx's deadline (if known).
func maxPollAttempts(ctx context.Context, interval time.Duration) (max int, known bool) {
	deadline, ok := ctx.Deadline()
	if !ok || interval <= 0 {
		return 0, false
	}
	remaining := time.Until(deadline)
	if remaining <= 0 {
		return 1, true
	}
	// PollUntilContextCancel executes the condition immediately, then sleeps each interval.
	return int(remaining/interval) + 1, true
}

// shouldLogVMWaitAttempt limits VM wait polling noise: log the first attempt and every vmWaitLogEveryAttempts thereafter.
func shouldLogVMWaitAttempt(attempt int) bool {
	if attempt <= 1 {
		return true
	}
	return attempt%vmWaitLogEveryAttempts == 0
}

// logVMWaitAttempt logs one poll iteration when shouldLogVMWaitAttempt allows, including retries left when known.
func logVMWaitAttempt(t testing.TB, desc string, attempt int, max int, maxKnown bool, detail string) {
	t.Helper()
	if !shouldLogVMWaitAttempt(attempt) {
		return
	}
	if maxKnown {
		left := max - attempt
		if left < 0 {
			left = 0
		}
		t.Logf("%s: attempt %d/%d (retries left: %d): %s", desc, attempt, max, left, detail)
		return
	}
	t.Logf("%s: attempt %d: %s", desc, attempt, detail)
}

// WaitForVirtualMachineInstanceExists polls until the VMI object is present (same name as the VM).
func WaitForVirtualMachineInstanceExists(t testing.TB, ctx context.Context, client dynamic.Interface, namespace, name string) error {
	t.Helper()
	attempts := 0
	lastDetail := ""
	desc := fmt.Sprintf("wait VMI %s/%s exists", namespace, name)
	maxAttempts, maxKnown := maxPollAttempts(ctx, vmPollInterval)
	err := wait.PollUntilContextCancel(ctx, vmPollInterval, true, func(ctx context.Context) (bool, error) {
		attempts++
		_, err := client.Resource(vmiGVR).Namespace(namespace).Get(ctx, name, metav1.GetOptions{})
		if err == nil {
			lastDetail = fmt.Sprintf("attempt=%d vmi exists", attempts)
			logVMWaitAttempt(t, desc, attempts, maxAttempts, maxKnown, "vmi now exists")
			return true, nil
		}
		if apierrors.IsNotFound(err) {
			detail, terminalFailure, detailErr := virtualMachineStatusDetail(ctx, client, namespace, name)
			if detailErr != nil {
				return false, fmt.Errorf("attempt %d: read VM status: %w", attempts, detailErr)
			}
			lastDetail = fmt.Sprintf("attempt=%d vmi not found (%s)", attempts, detail)
			logVMWaitAttempt(t, desc, attempts, maxAttempts, maxKnown, fmt.Sprintf("vmi not found (%s)", detail))
			if terminalFailure {
				return false, fmt.Errorf("attempt %d: VMI was not created: %s", attempts, detail)
			}
			return false, nil
		}
		return false, fmt.Errorf("attempt %d: get VMI: %w", attempts, err)
	})
	if err == nil {
		return nil
	}
	if lastDetail != "" {
		return fmt.Errorf("wait for VMI %s/%s to exist failed after %d poll(s): %w (last detail: %s)", namespace, name, attempts, err, lastDetail)
	}
	return fmt.Errorf("wait for VMI %s/%s to exist failed after %d poll(s): %w", namespace, name, attempts, err)
}

// WaitForVirtualMachineInstanceRunning polls until the VMI reports Running phase.
func WaitForVirtualMachineInstanceRunning(t testing.TB, ctx context.Context, client dynamic.Interface, namespace, name string) error {
	t.Helper()
	attempts := 0
	lastDetail := ""
	desc := fmt.Sprintf("wait VMI %s/%s running", namespace, name)
	maxAttempts, maxKnown := maxPollAttempts(ctx, vmPollInterval)
	err := wait.PollUntilContextCancel(ctx, vmPollInterval, true, func(ctx context.Context) (bool, error) {
		attempts++
		vmi, err := client.Resource(vmiGVR).Namespace(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			if apierrors.IsNotFound(err) {
				detail, terminalFailure, detailErr := virtualMachineStatusDetail(ctx, client, namespace, name)
				if detailErr != nil {
					return false, fmt.Errorf("attempt %d: read VM status: %w", attempts, detailErr)
				}
				lastDetail = fmt.Sprintf("attempt=%d vmi not found (%s)", attempts, detail)
				logVMWaitAttempt(t, desc, attempts, maxAttempts, maxKnown, fmt.Sprintf("vmi not found (%s)", detail))
				if terminalFailure {
					return false, fmt.Errorf("attempt %d: VMI failed before becoming Running: %s", attempts, detail)
				}
				return false, nil
			}
			return false, fmt.Errorf("attempt %d: get VMI: %w", attempts, err)
		}
		phase, found, err := unstructured.NestedString(vmi.Object, "status", "phase")
		if err != nil {
			return false, fmt.Errorf("attempt %d: read VMI phase: %w", attempts, err)
		}
		if !found {
			lastDetail = fmt.Sprintf("attempt=%d %s", attempts, vmiPhaseDetail(vmi))
			logVMWaitAttempt(t, desc, attempts, maxAttempts, maxKnown, vmiPhaseDetail(vmi))
			return false, nil
		}
		detail := vmiPhaseDetail(vmi)
		lastDetail = fmt.Sprintf("attempt=%d %s", attempts, detail)
		if phase != string(kubevirtv1.Running) {
			vmDetail, terminal := vmPrintableStatusTerminal(ctx, client, namespace, name)
			if terminal {
				detail += " " + vmDetail
				lastDetail = fmt.Sprintf("attempt=%d %s", attempts, detail)
				logVMWaitAttempt(t, desc, attempts, maxAttempts, maxKnown, detail)
				return false, fmt.Errorf("attempt %d: unrecoverable VM error: %s", attempts, vmDetail)
			}
			if vmDetail != "" {
				detail += " " + vmDetail
				lastDetail = fmt.Sprintf("attempt=%d %s", attempts, detail)
			}
		}
		logVMWaitAttempt(t, desc, attempts, maxAttempts, maxKnown, detail)
		return phase == string(kubevirtv1.Running), nil
	})
	if err == nil {
		return nil
	}
	if lastDetail != "" {
		return fmt.Errorf("wait for VMI %s/%s to be Running failed after %d poll(s): %w (last detail: %s)", namespace, name, attempts, err, lastDetail)
	}
	return fmt.Errorf("wait for VMI %s/%s to be Running failed after %d poll(s): %w", namespace, name, attempts, err)
}

// GetVMINodeName returns the Kubernetes node name that hosts the given VirtualMachineInstance.
func GetVMINodeName(ctx context.Context, client dynamic.Interface, namespace, name string) (string, error) {
	vmi, err := client.Resource(vmiGVR).Namespace(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return "", fmt.Errorf("get VMI %s/%s: %w", namespace, name, err)
	}
	nodeName, found, err := unstructured.NestedString(vmi.Object, "status", "nodeName")
	if err != nil {
		return "", fmt.Errorf("read VMI %s/%s status.nodeName: %w", namespace, name, err)
	}
	if !found || nodeName == "" {
		return "", fmt.Errorf("VMI %s/%s has no status.nodeName (phase=%s)", namespace, name, vmiPhaseDetail(vmi))
	}
	return nodeName, nil
}

// WaitForVirtualMachineDeleted polls until the VirtualMachine object is gone.
func WaitForVirtualMachineDeleted(t testing.TB, ctx context.Context, client dynamic.Interface, namespace, name string) error {
	t.Helper()
	attempts := 0
	desc := fmt.Sprintf("wait VM %s/%s deleted", namespace, name)
	maxAttempts, maxKnown := maxPollAttempts(ctx, vmPollInterval)
	err := wait.PollUntilContextCancel(ctx, vmPollInterval, true, func(ctx context.Context) (bool, error) {
		attempts++
		_, err := client.Resource(vmGVR).Namespace(namespace).Get(ctx, name, metav1.GetOptions{})
		if apierrors.IsNotFound(err) {
			logVMWaitAttempt(t, desc, attempts, maxAttempts, maxKnown, "vm no longer found")
			return true, nil
		}
		if err != nil {
			return false, fmt.Errorf("attempt %d: get VM: %w", attempts, err)
		}
		logVMWaitAttempt(t, desc, attempts, maxAttempts, maxKnown, "vm still exists")
		return false, nil
	})
	if err == nil {
		return nil
	}
	return fmt.Errorf("wait for VM %s/%s deletion failed after %d poll(s): %w", namespace, name, attempts, err)
}
