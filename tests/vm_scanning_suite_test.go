//go:build test_e2e_vm

package tests

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	v2 "github.com/stackrox/rox/generated/api/v2"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/namespaces"
	"github.com/stackrox/rox/pkg/testutils/centralgrpc"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stackrox/rox/tests/vmhelpers"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"google.golang.org/grpc"
	coreV1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

const (
	// suiteTimeout bounds the entire TestVMScanning run, including provisioning,
	// guest preparation, assertions, and teardown.
	suiteTimeout = 90 * time.Minute

	// defaultVMProvisionTimeout is the per-VM ceiling for cloud-init, boot, and
	// virtctl SSH readiness when no override is set via VM_SCAN_TIMEOUT.
	defaultVMProvisionTimeout = 20 * time.Minute

	// defaultSSHFirstContactTimeout is the generous timeout for the initial SSH
	// reachability probe on a freshly-booted VM. sshd may take minutes to
	// start even after the VMI reports Running.
	defaultSSHFirstContactTimeout = 20 * time.Minute

	// defaultGuestStepTimeout caps individual guest-preparation steps that
	// run after SSH is confirmed working (cloud-init wait and sudo check).
	defaultGuestStepTimeout = 10 * time.Minute

	// defaultVirtctlCommandTimeout is the maximum wall-clock time for a single
	// virtctl SSH or SCP invocation.
	defaultVirtctlCommandTimeout = 30 * time.Minute

	// defaultVirtctlHeartbeatInterval controls how often a no-op command is sent
	// over idle SSH connections to prevent cloud/network timeouts.
	defaultVirtctlHeartbeatInterval = 20 * time.Second

	// defaultVMDeleteTimeout is the per-VM ceiling for graceful deletion during
	// TearDownSuite.
	defaultVMDeleteTimeout = 5 * time.Minute

	// featureGateVerifyTimeout bounds the env-var check that confirms the
	// VirtualMachines feature flag is set on Central, Sensor, and Compliance.
	featureGateVerifyTimeout = 2 * time.Minute

	// vmPlacementLookupTimeout bounds the best-effort collector-pod lookup used
	// for diagnostic VM-to-node-to-collector placement logging.
	vmPlacementLookupTimeout = 60 * time.Second

	// k8sResourcePollInterval is the polling cadence for waiting on k8s
	// resources (namespace deletion, service account readiness, etc.).
	k8sResourcePollInterval = 2 * time.Second
)

// VMHandle tracks a KubeVirt VM used by the suite (persistent or transient).
type VMHandle struct {
	Name      string
	Namespace string
	GuestUser string
	// ID is the Central VirtualMachine id once known (populated in later tasks).
	ID string
	// NodeName is the Kubernetes node hosting the VirtualMachineInstance (populated after VMI is Running).
	NodeName string
	// SkipReason, when set, skips this VM's subtests so other VMs still run.
	SkipReason string
}

// VMScanningSuite exercises OpenShift VM scanning end-to-end (KubeVirt guests, roxagent, Central).
type VMScanningSuite struct {
	KubernetesSuite

	ctx        context.Context
	cleanupCtx context.Context
	cancel     func()

	cfg           *vmhelpers.VMScanConfig
	restCfg       *rest.Config
	k8sClient     kubernetes.Interface
	dynamicClient dynamic.Interface
	namespace     string

	conn       *grpc.ClientConn
	vmClient   v2.VirtualMachineServiceClient
	vmV2Client v2.VirtualMachineV2ServiceClient
	// enhancedVMModel follows Central's ROX_VIRTUAL_MACHINES_ENHANCED_DATA_MODEL.
	// That flag selects VirtualMachineV2Service vs VirtualMachineService.
	enhancedVMModel bool

	virtctl vmhelpers.Virtctl

	// vmSpecs is the provisioning blueprint for each VM.
	vmSpecs []vmhelpers.VMSpec
	// vms tracks every VM provisioned by the suite; TearDownSuite deletes each.
	vms []VMHandle
	// scannerV4Checked is set after the one-time Scanner V4 matcher initialization check.
	scannerV4Checked bool
}

// TestVMScanning is the suite entrypoint for VM scanning E2E tests.
func TestVMScanning(t *testing.T) {
	suite.Run(t, new(VMScanningSuite))
}

func (s *VMScanningSuite) SetupSuite() {
	s.KubernetesSuite.SetupSuite()
	t := s.T()

	s.logf("VM scanning setup: initialize test contexts")
	s.ctx, s.cleanupCtx, s.cancel = testContexts(t, "TestVMScanning", suiteTimeout)

	s.logf("VM scanning setup: load suite configuration from environment")
	s.cfg = mustLoadVMScanConfig(t)
	s.logf("VM scanning setup: create Kubernetes clients")
	s.restCfg = getConfig(t)
	s.k8sClient = createK8sClientWithConfig(t, s.restCfg)
	s.dynamicClient = mustCreateDynamicClient(t, s.restCfg)
	s.logf("VM scanning setup: verify cluster KVM readiness")
	mustVerifyClusterKVMReady(t, s.ctx, s.k8sClient)

	s.logf("VM scanning setup: ensure compliance metrics are exposed")
	s.ensureComplianceMetricsExposed()

	// VM_SCAN_NAMESPACE pins the test namespace for local development and re-runs:
	// VMs survive across invocations so you can iterate on later test stages without
	// waiting for full VM provisioning each time.
	if fixedNamespace := strings.TrimSpace(os.Getenv("VM_SCAN_NAMESPACE")); fixedNamespace != "" {
		s.namespace = fixedNamespace
		s.logf("VM scanning setup: using fixed namespace from VM_SCAN_NAMESPACE=%q", s.namespace)
	} else {
		s.namespace = fmt.Sprintf("%s-%s", s.cfg.NamespacePrefix, uuid.NewV4().String()[:8])
	}

	s.logf("VM scanning setup: connect to Central gRPC")
	s.conn = centralgrpc.GRPCConnectionToCentral(t)
	s.vmClient = v2.NewVirtualMachineServiceClient(s.conn)
	s.vmV2Client = v2.NewVirtualMachineV2ServiceClient(s.conn)

	s.logf("VM scanning setup: verify central/sensor connectivity and feature gates")
	s.mustWaitForHealthyCentralSensorConnection()
	s.mustVerifyVirtualMachinesFeatureEnabled()
	s.resolveVMAPI()
	s.logf("VM scanning setup: verify cluster VSOCK readiness")
	mustVerifyClusterVSOCKReady(t, s.ctx, s.k8sClient, s.dynamicClient)

	s.logf("VM scanning setup: resolve SSH identity and configure virtctl")
	identity := mustResolveSSHIdentityFile(t, s.cfg)
	s.logf("VM_SSH_PRIVATE_KEY_PATH=%q", identity)
	cmdTimeout := defaultVirtctlCommandTimeout
	if s.cfg.ScanTimeout > 0 && s.cfg.ScanTimeout < cmdTimeout {
		cmdTimeout = s.cfg.ScanTimeout
	}
	s.virtctl = vmhelpers.Virtctl{
		Path:              s.cfg.VirtctlPath,
		IdentityFile:      identity,
		CommandTimeout:    cmdTimeout,
		KnownHostsFile:    filepath.Join(t.TempDir(), "known_hosts"),
		Logf:              s.logf,
		HeartbeatInterval: defaultVirtctlHeartbeatInterval,
	}

	s.vmSpecs = s.cfg.VMSpecs()
	s.logf("VM scanning setup: provision VMs (%d specs)", len(s.vmSpecs))
	s.provisionVMs(s.vmSpecs)
	s.logf("VM scanning setup: prepare guests (ssh/cloud-init/sudo readiness)")
	s.prepareGuests()
	s.logf("VM scanning setup: complete")
}

func (s *VMScanningSuite) TearDownSuite() {
	if s.cancel != nil {
		defer s.cancel()
	}

	// When VM_SCAN_SKIP_CLEANUP is set, leave VMs and the namespace intact so a
	// developer can SSH into the guests or inspect cluster state after a failure
	// without having to re-provision from scratch.
	if s.cfg != nil && s.cfg.SkipCleanup {
		s.logf("teardown: VM_SCAN_SKIP_CLEANUP is set — skipping VM and namespace deletion (VMs and namespace left intact for debugging)")
		s.closeConn()
		return
	}

	deleteTimeout := s.resourceDeleteTimeout()
	if s.dynamicClient != nil {
		for _, vm := range s.vms {
			vmCtx, vmCancel := context.WithTimeout(s.cleanupCtx, deleteTimeout)
			if err := vmhelpers.DeleteVirtualMachine(vmCtx, s.dynamicClient, vm.Namespace, vm.Name); err != nil {
				if vmhelpers.IsAuthenticationExpired(err) {
					s.logf("teardown: STOPPING — %v", vmhelpers.ErrAuthenticationExpired)
					vmCancel()
					s.closeConn()
					return
				}
				s.logf("teardown: DeleteVirtualMachine %s/%s failed: %v", vm.Namespace, vm.Name, err)
			}
			if err := vmhelpers.WaitForVirtualMachineDeleted(s.T(), vmCtx, s.dynamicClient, vm.Namespace, vm.Name); err != nil {
				if vmhelpers.IsAuthenticationExpired(err) {
					s.logf("teardown: STOPPING — %v", vmhelpers.ErrAuthenticationExpired)
					vmCancel()
					s.closeConn()
					return
				}
				s.logf("teardown: WaitForVirtualMachineDeleted %s/%s timed out or failed: %v", vm.Namespace, vm.Name, err)
			}
			vmCancel()
		}
	} else if len(s.vms) > 0 {
		s.logf("teardown: skipping VM cleanup (%d handle(s)): dynamic client is nil", len(s.vms))
	}

	if s.k8sClient != nil && s.namespace != "" {
		nsCtx, nsCancel := context.WithTimeout(s.cleanupCtx, deleteTimeout)
		err := s.k8sClient.CoreV1().Namespaces().Delete(nsCtx, s.namespace, metaV1.DeleteOptions{})
		if err != nil && !apierrors.IsNotFound(err) {
			if vmhelpers.IsAuthenticationExpired(err) {
				s.logf("teardown: STOPPING — %v", vmhelpers.ErrAuthenticationExpired)
				nsCancel()
				s.closeConn()
				return
			}
			s.logf("teardown: Namespace.Delete %q failed: %v", s.namespace, err)
		}
		if waitErr := waitForNamespaceDeleted(nsCtx, s.k8sClient, s.namespace); waitErr != nil {
			s.logf("teardown: wait for namespace %q to be removed failed: %v", s.namespace, waitErr)
		}
		nsCancel()
	}

	s.closeConn()
}

func (s *VMScanningSuite) closeConn() {
	if s.conn != nil {
		if err := s.conn.Close(); err != nil {
			s.logf("teardown: gRPC conn.Close failed: %v", err)
		}
	}
}

func (s *VMScanningSuite) vmProvisionTimeout() time.Duration {
	if s.cfg != nil && s.cfg.ScanTimeout > 0 {
		return s.cfg.ScanTimeout
	}
	return defaultVMProvisionTimeout
}

func (s *VMScanningSuite) virtctlForVM(vm VMHandle) vmhelpers.Virtctl {
	v := s.virtctl
	if u := strings.TrimSpace(vm.GuestUser); u != "" {
		v.Username = u
	}
	return v
}

func (s *VMScanningSuite) sshFirstContactTimeout() time.Duration {
	if s.cfg == nil || s.cfg.ScanTimeout <= 0 {
		return defaultSSHFirstContactTimeout
	}
	if s.cfg.ScanTimeout < defaultSSHFirstContactTimeout {
		return s.cfg.ScanTimeout
	}
	return defaultSSHFirstContactTimeout
}

func (s *VMScanningSuite) guestStepTimeout() time.Duration {
	if s.cfg == nil || s.cfg.ScanTimeout <= 0 {
		return defaultGuestStepTimeout
	}
	if s.cfg.ScanTimeout < defaultGuestStepTimeout {
		return s.cfg.ScanTimeout
	}
	return defaultGuestStepTimeout
}

func waitForNamespaceDeleted(ctx context.Context, k8s kubernetes.Interface, name string) error {
	return wait.PollUntilContextCancel(ctx, k8sResourcePollInterval, true, func(ctx context.Context) (bool, error) {
		_, err := k8s.CoreV1().Namespaces().Get(ctx, name, metaV1.GetOptions{})
		if apierrors.IsNotFound(err) {
			return true, nil
		}
		if err != nil {
			return false, err
		}
		return false, nil
	})
}

func (s *VMScanningSuite) mustWaitForHealthyCentralSensorConnection() {
	s.waitUntilK8sDeploymentReady(s.ctx, namespaces.StackRox, sensorDeployment)
	waitUntilCentralSensorConnectionIs(s.T(), s.ctx, storage.ClusterHealthStatus_HEALTHY)
}

// resolveVMAPI reads ROX_VIRTUAL_MACHINES_ENHANCED_DATA_MODEL from Central and
// checks that the matching gRPC service is registered.
func (s *VMScanningSuite) resolveVMAPI() {
	t := s.T()
	t.Helper()

	flag := features.VirtualMachinesEnhancedDataModel
	s.enhancedVMModel = s.centralFeatureEnabled(flag)
	s.logf("VM scanning setup: Central %s=%v", flag.EnvVar(), s.enhancedVMModel)

	if s.enhancedVMModel {
		_, err := s.vmV2Client.ListVMs(s.ctx, &v2.ListVMsRequest{})
		require.NoError(t, err, "VirtualMachineV2Service must be registered when %s is enabled", flag.EnvVar())
		return
	}
	_, err := s.vmClient.ListVirtualMachines(s.ctx, &v2.ListVirtualMachinesRequest{})
	require.NoError(t, err, "VirtualMachineService must be registered when %s is disabled", flag.EnvVar())
}

// centralFeatureEnabled reports whether flag is on in the Central deployment,
// matching features.Enabled: explicit true/false, otherwise the flag default.
func (s *VMScanningSuite) centralFeatureEnabled(flag features.FeatureFlag) bool {
	t := s.T()
	t.Helper()

	obj, err := s.k8sClient.AppsV1().Deployments(namespaces.StackRox).Get(s.ctx, "central", metaV1.GetOptions{})
	require.NoError(t, err, "get Deployment %s/central", namespaces.StackRox)
	for _, c := range obj.Spec.Template.Spec.Containers {
		if c.Name != "central" {
			continue
		}
		for _, e := range c.Env {
			if e.Name != flag.EnvVar() {
				continue
			}
			switch strings.ToLower(strings.TrimSpace(e.Value)) {
			case "false":
				return false
			case "true":
				return true
			default:
				return flag.Default()
			}
		}
		return flag.Default()
	}
	require.FailNowf(t, "container central not found in Deployment stackrox/central",
		"available containers: %s", formatContainerNames(obj.Spec.Template.Spec.Containers))
	return false
}

func (s *VMScanningSuite) mustVerifyVirtualMachinesFeatureEnabled() {
	ctx, cancel := context.WithTimeout(s.ctx, featureGateVerifyTimeout)
	defer cancel()

	wantEnv := features.VirtualMachines.EnvVar()
	ns := namespaces.StackRox

	// Verify the feature flag env var is set on all components that need it.
	s.mustVerifyContainerEnvVar(ctx, "deployment", "central", "central", ns, wantEnv)
	s.mustVerifyContainerEnvVar(ctx, "deployment", sensorDeployment, sensorContainer, ns, wantEnv)
	s.mustVerifySensorVSOCKRBAC(ctx)
}

// mustVerifySensorVSOCKRBAC asserts Sensor can get KubeVirt VMI vsock subresources.
// Pull-mode scraping fails without this; the Helm chart creates the binding when
// virtualMachines.enabled follows ROX_VIRTUAL_MACHINES (on by default).
func (s *VMScanningSuite) mustVerifySensorVSOCKRBAC(ctx context.Context) {
	t := s.T()
	t.Helper()

	binding, err := s.k8sClient.RbacV1().ClusterRoleBindings().Get(ctx, "stackrox:vsock-access-binding", metaV1.GetOptions{})
	require.NoError(t, err, "get ClusterRoleBinding stackrox:vsock-access-binding; "+
		"Sensor cannot scrape guest agents over vsock without this RBAC "+
		"(check that Helm virtualMachines.enabled / ROX_VIRTUAL_MACHINES is on)")
	require.Equal(t, "ClusterRole", binding.RoleRef.Kind)
	require.Equal(t, "stackrox:vsock-access", binding.RoleRef.Name)

	foundSensorSA := false
	for _, sub := range binding.Subjects {
		if sub.Kind == "ServiceAccount" && sub.Name == "sensor" && sub.Namespace == namespaces.StackRox {
			foundSensorSA = true
			break
		}
	}
	require.True(t, foundSensorSA, "ClusterRoleBinding stackrox:vsock-access-binding must bind stackrox/sensor")
}

// mustVerifyContainerEnvVar asserts that the named container within a Deployment or DaemonSet
// has the given environment variable set to a truthy value ("true", "1", etc.).
// This catches deployment misconfigurations where a feature flag reaches one component
// but not another that also needs it.
func (s *VMScanningSuite) mustVerifyContainerEnvVar(ctx context.Context, kind, name, containerName, ns, envName string) {
	t := s.T()
	t.Helper()

	var containers []coreV1.Container
	switch kind {
	case "deployment":
		obj, err := s.k8sClient.AppsV1().Deployments(ns).Get(ctx, name, metaV1.GetOptions{})
		require.NoError(t, err, "get Deployment %s/%s", ns, name)
		containers = obj.Spec.Template.Spec.Containers
	case "daemonset":
		obj, err := s.k8sClient.AppsV1().DaemonSets(ns).Get(ctx, name, metaV1.GetOptions{})
		require.NoError(t, err, "get DaemonSet %s/%s", ns, name)
		containers = obj.Spec.Template.Spec.Containers
	default:
		require.Failf(t, "unsupported kind", "%q", kind)
	}

	for _, c := range containers {
		if c.Name != containerName {
			continue
		}
		for _, e := range c.Env {
			if e.Name == envName {
				val := strings.ToLower(strings.TrimSpace(e.Value))
				require.Truef(t, val == "true" || val == "1",
					"%s %s/%s container %q has %s=%q which is not truthy",
					kind, ns, name, containerName, envName, e.Value)
				return
			}
		}
		require.Failf(t, fmt.Sprintf("%s %s/%s container %q is missing env var %s", kind, ns, name, containerName, envName),
			"the feature flag must be set on Central and Sensor for pull-mode VM scanning; "+
				"present env vars: %s", formatContainerEnvNames(c.Env))
	}
	require.Failf(t, fmt.Sprintf("container %q not found in %s %s/%s", containerName, kind, ns, name),
		"available containers: %s", formatContainerNames(containers))
}

func formatContainerEnvNames(envs []coreV1.EnvVar) string {
	names := make([]string, len(envs))
	for i, e := range envs {
		names[i] = e.Name
	}
	return strings.Join(names, ", ")
}

func formatContainerNames(containers []coreV1.Container) string {
	names := make([]string, len(containers))
	for i, c := range containers {
		names[i] = c.Name
	}
	return strings.Join(names, ", ")
}

func (s *VMScanningSuite) provisionVMs(specs []vmhelpers.VMSpec) {
	ctx := s.ctx

	s.logf("provision VMs: creating namespace %q", s.namespace)
	_, err := s.k8sClient.CoreV1().Namespaces().Create(ctx, &coreV1.Namespace{
		ObjectMeta: metaV1.ObjectMeta{Name: s.namespace},
	}, metaV1.CreateOptions{})
	if err != nil && !apierrors.IsAlreadyExists(err) {
		require.NoError(s.T(), err, "create test namespace %q", s.namespace)
	}
	if apierrors.IsAlreadyExists(err) {
		s.logf("provision VMs: namespace %q already exists; reusing it", s.namespace)
	}

	require.NoError(s.T(), vmhelpers.EnsureImagePullSecret(ctx, s.k8sClient, s.logf, s.namespace, vmhelpers.ImagePullSecretName, s.cfg.ImagePullSecretPath),
		"ensure image pull secret ready in namespace %q", s.namespace)

	for _, sp := range specs {
		req := s.vmSpecToRequest(sp)
		s.logf("provision VMs: ensuring VM exists %s/%s with image %q", s.namespace, sp.Name, sp.Image)
		createErr := vmhelpers.CreateVirtualMachine(ctx, s.dynamicClient, req)
		if createErr == nil {
			s.logf("provision VMs: created VM %s/%s", s.namespace, sp.Name)
		} else if apierrors.IsAlreadyExists(createErr) {
			currentImage, imgErr := vmhelpers.GetVMContainerDiskImage(ctx, s.dynamicClient, s.namespace, sp.Name)
			if imgErr != nil {
				s.logf("provision VMs: could not read image for existing VM %s/%s: %v; recreating", s.namespace, sp.Name, imgErr)
			}
			if imgErr != nil || currentImage != sp.Image {
				if imgErr == nil {
					s.logf("provision VMs: VM %s/%s has image %q but want %q; deleting and recreating",
						s.namespace, sp.Name, currentImage, sp.Image)
				}
				delCtx, delCancel := context.WithTimeout(ctx, s.resourceDeleteTimeout())
				defer delCancel()
				require.NoError(s.T(), vmhelpers.DeleteVirtualMachine(delCtx, s.dynamicClient, s.namespace, sp.Name),
					"DeleteVirtualMachine %s/%s for image mismatch", s.namespace, sp.Name)
				require.NoError(s.T(), vmhelpers.WaitForVirtualMachineDeleted(s.T(), delCtx, s.dynamicClient, s.namespace, sp.Name),
					"WaitForVirtualMachineDeleted %s/%s for image mismatch", s.namespace, sp.Name)
				require.NoError(s.T(), vmhelpers.CreateVirtualMachine(ctx, s.dynamicClient, req),
					"CreateVirtualMachine %s/%s after image mismatch delete", s.namespace, sp.Name)
				s.logf("provision VMs: recreated VM %s/%s with correct image", s.namespace, sp.Name)
			} else {
				s.logf("provision VMs: VM %s/%s already exists with correct image; reusing it", s.namespace, sp.Name)
			}
		} else {
			require.NoError(s.T(), createErr, "EnsureVirtualMachineExists %s/%s", s.namespace, sp.Name)
		}
		s.vms = append(s.vms, VMHandle{Name: sp.Name, Namespace: s.namespace, GuestUser: sp.GuestUser})
	}
	for i := range s.vms {
		vm := &s.vms[i]
		vmCtx, vmCancel := context.WithTimeout(ctx, s.vmProvisionTimeout())
		s.logf("provision VMs: waiting for VMI object %s/%s (timeout=%v)", vm.Namespace, vm.Name, s.vmProvisionTimeout())
		require.NoError(s.T(), vmhelpers.WaitForVirtualMachineInstanceExists(s.T(), vmCtx, s.dynamicClient, vm.Namespace, vm.Name),
			"WaitForVirtualMachineInstanceExists %s/%s", vm.Namespace, vm.Name)
		s.logf("provision VMs: waiting for VMI Running %s/%s (timeout=%v)", vm.Namespace, vm.Name, s.vmProvisionTimeout())
		require.NoError(s.T(), vmhelpers.WaitForVirtualMachineInstanceRunning(s.T(), vmCtx, s.dynamicClient, vm.Namespace, vm.Name),
			"WaitForVirtualMachineInstanceRunning %s/%s", vm.Namespace, vm.Name)
		vmCancel()

		nodeName, err := vmhelpers.GetVMINodeName(ctx, s.dynamicClient, vm.Namespace, vm.Name)
		require.NoError(s.T(), err, "GetVMINodeName %s/%s", vm.Namespace, vm.Name)
		vm.NodeName = nodeName
	}

	s.logf("VM placement:\n%s", s.vmPlacementSummary(ctx))
}

// vmPlacementSummary returns a diagnostic table mapping each persistent VM to
// the node it landed on and the collector pod running on that node, helping
// debug VM-to-collector routing issues.
func (s *VMScanningSuite) vmPlacementSummary(ctx context.Context) string {
	lookupCtx, cancel := context.WithTimeout(ctx, vmPlacementLookupTimeout)
	defer cancel()

	nodeToCollector := make(map[string]string)
	pods, err := s.k8sClient.CoreV1().Pods(namespaces.StackRox).List(lookupCtx, metaV1.ListOptions{
		LabelSelector: "app=collector",
	})
	if err != nil {
		return fmt.Sprintf("(could not list collector pods: %v)", err)
	}
	for _, p := range pods.Items {
		nodeToCollector[p.Spec.NodeName] = p.Name
	}

	var b strings.Builder
	for _, vm := range s.vms {
		node := vm.NodeName
		if node == "" {
			node = "<unknown>"
		}
		collector := nodeToCollector[vm.NodeName]
		if collector == "" {
			collector = "<none>"
		}
		fmt.Fprintf(&b, "  VM: %s, Node: %s, Collector pod: %s\n", vm.Name, node, collector)
	}
	return strings.TrimRight(b.String(), "\n")
}

func (s *VMScanningSuite) prepareGuests() {
	t := s.T()
	for i := range s.vms {
		require.NoError(t, s.prepareGuestWithRecovery(&s.vms[i]))
	}
}

func (s *VMScanningSuite) prepareGuestWithRecovery(vm *VMHandle) error {
	const maxRecoveries = 2
	for recoveryAttempt := 0; recoveryAttempt <= maxRecoveries; recoveryAttempt++ {
		err := s.prepareGuest(vm)
		if err == nil {
			return nil
		}
		recoverableSSHErr := errors.Is(err, vmhelpers.ErrSSHAuthenticationFailed) || errors.Is(err, vmhelpers.ErrSSHConnectivityStalled)
		if !recoverableSSHErr {
			return err
		}
		if recoveryAttempt == maxRecoveries {
			return fmt.Errorf("prepare guest %s/%s failed after %d recovery attempt(s): %w",
				vm.Namespace, vm.Name, maxRecoveries, err)
		}
		s.logf("SSH became unhealthy for %s/%s, recreating VM and retrying guest preparation (%d/%d): %v",
			vm.Namespace, vm.Name, recoveryAttempt+1, maxRecoveries, err)
		if recreateErr := s.recreateVM(vm); recreateErr != nil {
			return fmt.Errorf("recreate VM %s/%s after recoverable SSH failure: %w", vm.Namespace, vm.Name, recreateErr)
		}
	}
	return nil
}

func (s *VMScanningSuite) recreateVM(vm *VMHandle) error {
	req, err := s.vmRequestForVM(*vm)
	if err != nil {
		return err
	}

	delCtx, delCancel := context.WithTimeout(s.ctx, s.resourceDeleteTimeout())
	defer delCancel()
	if err := vmhelpers.DeleteVirtualMachine(delCtx, s.dynamicClient, vm.Namespace, vm.Name); err != nil {
		return fmt.Errorf("DeleteVirtualMachine: %w", err)
	}
	if err := vmhelpers.WaitForVirtualMachineDeleted(s.T(), delCtx, s.dynamicClient, vm.Namespace, vm.Name); err != nil {
		return fmt.Errorf("WaitForVirtualMachineDeleted: %w", err)
	}

	createCtx, createCancel := context.WithTimeout(s.ctx, s.vmProvisionTimeout())
	defer createCancel()
	if err := vmhelpers.CreateVirtualMachine(createCtx, s.dynamicClient, req); err != nil {
		return fmt.Errorf("CreateVirtualMachine: %w", err)
	}
	if err := vmhelpers.WaitForVirtualMachineInstanceExists(s.T(), createCtx, s.dynamicClient, vm.Namespace, vm.Name); err != nil {
		return fmt.Errorf("WaitForVirtualMachineInstanceExists: %w", err)
	}
	if err := vmhelpers.WaitForVirtualMachineInstanceRunning(s.T(), createCtx, s.dynamicClient, vm.Namespace, vm.Name); err != nil {
		return fmt.Errorf("WaitForVirtualMachineInstanceRunning: %w", err)
	}

	nodeName, nodeErr := vmhelpers.GetVMINodeName(s.ctx, s.dynamicClient, vm.Namespace, vm.Name)
	if nodeErr != nil {
		s.logf("recreate VM: could not determine node for %s/%s: %v", vm.Namespace, vm.Name, nodeErr)
		vm.NodeName = ""
	} else {
		s.logf("recreate VM: %s/%s now on node %s (was %s)", vm.Namespace, vm.Name, nodeName, vm.NodeName)
		vm.NodeName = nodeName
	}
	return nil
}

func (s *VMScanningSuite) vmRequestForVM(vm VMHandle) (vmhelpers.VMRequest, error) {
	for _, sp := range s.vmSpecs {
		if sp.Name == vm.Name {
			req := s.vmSpecToRequest(sp)
			if u := strings.TrimSpace(vm.GuestUser); u != "" {
				req.GuestUser = u
			}
			req.Namespace = vm.Namespace
			return req, nil
		}
	}
	return vmhelpers.VMRequest{}, fmt.Errorf("no spec found for VM %s/%s", vm.Namespace, vm.Name)
}

// vmSpecToRequest converts a VMSpec into a VMRequest using suite-level defaults.
func (s *VMScanningSuite) vmSpecToRequest(sp vmhelpers.VMSpec) vmhelpers.VMRequest {
	return vmhelpers.VMRequest{
		Name:         sp.Name,
		Namespace:    s.namespace,
		Image:        sp.Image,
		GuestUser:    sp.GuestUser,
		SSHPublicKey: s.cfg.SSHPublicKey,
	}
}

func (s *VMScanningSuite) skipUnlessLegacyVMAPI(t *testing.T) {
	t.Helper()
	if s.enhancedVMModel {
		t.Skip("VirtualMachineService is not used when ROX_VIRTUAL_MACHINES_ENHANCED_DATA_MODEL is enabled")
	}
}

func (s *VMScanningSuite) skipUnlessV2VMAPI(t *testing.T) {
	t.Helper()
	if !s.enhancedVMModel {
		t.Skip("VirtualMachineV2Service is not used when ROX_VIRTUAL_MACHINES_ENHANCED_DATA_MODEL is disabled")
	}
}

func (s *VMScanningSuite) mustListVMByNamespaceAndName(namespace, name string) *v2.VirtualMachine {
	t := s.T()
	t.Helper()
	vm, err := vmhelpers.ListVMByNamespaceName(s.ctx, s.vmClient, namespace, name)
	require.NoError(t, err)
	require.NotNil(t, vm, "ListVirtualMachines: no VM for namespace=%q name=%q", namespace, name)
	return vm
}

func (s *VMScanningSuite) mustGetVM(id string) *v2.VirtualMachine {
	t := s.T()
	t.Helper()
	resp, err := s.vmClient.GetVirtualMachine(s.ctx, &v2.GetVirtualMachineRequest{Id: id})
	require.NoError(t, err)
	require.NotNil(t, resp)
	return resp
}

func (s *VMScanningSuite) mustListV2VMByNamespaceAndName(namespace, name string) *v2.VMListItem {
	t := s.T()
	t.Helper()
	vm, err := vmhelpers.ListV2VMByNamespaceName(s.ctx, s.vmV2Client, namespace, name)
	require.NoError(t, err)
	require.NotNil(t, vm, "ListVMs: no VM for namespace=%q name=%q", namespace, name)
	return vm
}

func (s *VMScanningSuite) mustGetVMV2(id string) *v2.VMDetail {
	t := s.T()
	t.Helper()
	resp, err := s.vmV2Client.GetVM(s.ctx, &v2.GetVMRequest{Id: id})
	require.NoError(t, err)
	require.NotNil(t, resp)
	return resp
}

// waitForScannerV4Initialized blocks until the Scanner V4 matcher deployment
// is K8s-ready. When SCANNER_V4_MATCHER_READINESS=vulnerability (the default
// in CI via lib.sh), the K8s readiness probe already gates on the vuln DB
// being loaded, so no additional API polling is needed.
//
// Idempotent: subsequent calls return nil immediately after the first success.
func (s *VMScanningSuite) waitForScannerV4Initialized() error {
	if s.scannerV4Checked {
		return nil
	}
	s.logf("Scanner V4: waiting for matcher deployment to be K8s-ready")
	s.waitUntilK8sDeploymentReady(s.ctx, namespaces.StackRox, "scanner-v4-matcher")
	s.logf("Scanner V4: matcher deployment is ready")
	s.scannerV4Checked = true
	return nil
}

// ensureRoxagentServing starts Quadlet roxagent.service on the guest if needed
// and waits until it is active (VSOCK listener ready). Sensor scrapes afterward.
func (s *VMScanningSuite) ensureRoxagentServing(ctx context.Context, vm *VMHandle) error {
	if vm == nil {
		return errors.New("ensureRoxagentServing: nil VM handle")
	}
	s.mustVerifyVirtualMachinesFeatureEnabled()
	if err := s.waitForScannerV4Initialized(); err != nil {
		return fmt.Errorf("Scanner V4 matcher did not initialize within timeout: %w", err)
	}
	virt := s.virtctlForVM(*vm)
	return vmhelpers.EnsureRoxagentServing(ctx, virt, vm.Namespace, vm.Name)
}

// centralScanSnapshot is the scan-ready view from the VM API selected by Central's flag.
type centralScanSnapshot struct {
	ID     string
	Legacy *v2.VirtualMachine
	Detail *v2.VMDetail
}

// waitForScan polls the VM API selected by ROX_VIRTUAL_MACHINES_ENHANCED_DATA_MODEL.
func (s *VMScanningSuite) waitForScan(ctx context.Context, vm *VMHandle) (*centralScanSnapshot, error) {
	if vm == nil {
		return nil, errors.New("waitForScan: nil VM handle")
	}
	s.logf("scan wait %s/%s: start (timeout=%v poll=%v enhancedVMModel=%v)",
		vm.Namespace, vm.Name, s.cfg.ScanTimeout, s.cfg.ScanPollInterval, s.enhancedVMModel)
	waitCtx, cancel := context.WithTimeout(ctx, s.cfg.ScanTimeout)
	defer cancel()

	if s.enhancedVMModel {
		detail, err := s.waitForV2Scan(waitCtx, vm)
		if err != nil {
			return nil, err
		}
		vm.ID = detail.GetId()
		return &centralScanSnapshot{ID: vm.ID, Detail: detail}, nil
	}
	legacy, err := s.waitForLegacyScan(waitCtx, vm)
	if err != nil {
		return nil, err
	}
	vm.ID = legacy.GetId()
	return &centralScanSnapshot{ID: vm.ID, Legacy: legacy}, nil
}

func (s *VMScanningSuite) waitForLegacyScan(waitCtx context.Context, vm *VMHandle) (*v2.VirtualMachine, error) {
	baseOpts := vmhelpers.WaitOptions{
		Timeout:      s.cfg.ScanTimeout,
		PollInterval: s.cfg.ScanPollInterval,
		Logf:         s.logf,
	}

	present, err := vmhelpers.WaitForVMPresentInCentral(waitCtx, s.vmClient, baseOpts, vm.Namespace, vm.Name)
	if err != nil {
		return nil, err
	}
	vm.ID = present.GetId()

	s.logf("%s/%s: VM appeared in Central via VirtualMachineService (id=%q), waiting for namespace/name fields", vm.Namespace, vm.Name, vm.ID)
	if _, err := vmhelpers.WaitForVMIdentityFields(waitCtx, s.vmClient, baseOpts, present.GetId(), vm.Namespace, vm.Name); err != nil {
		return nil, err
	}
	s.logf("%s/%s: waiting for Central to report VM as Running", vm.Namespace, vm.Name)
	if _, err := vmhelpers.WaitForVMRunningInCentral(waitCtx, s.vmClient, baseOpts, present.GetId()); err != nil {
		return nil, err
	}
	s.logf("%s/%s: waiting for roxagent scan payload to arrive in Central", vm.Namespace, vm.Name)
	if _, err := vmhelpers.WaitForVMScanNonNil(waitCtx, s.vmClient, baseOpts, present.GetId()); err != nil {
		return nil, err
	}
	s.logf("%s/%s: waiting for Scanner to assign a scan timestamp", vm.Namespace, vm.Name)
	if _, err := vmhelpers.WaitForVMScanTimestamp(waitCtx, s.vmClient, baseOpts, present.GetId()); err != nil {
		return nil, err
	}
	s.logf("%s/%s: waiting for all components to be vulnerability-matched (no UNSCANNED)", vm.Namespace, vm.Name)
	return vmhelpers.WaitForScanReady(waitCtx, s.vmClient, baseOpts, present.GetId())
}

func (s *VMScanningSuite) waitForV2Scan(waitCtx context.Context, vm *VMHandle) (*v2.VMDetail, error) {
	baseOpts := vmhelpers.WaitOptions{
		Timeout:      s.cfg.ScanTimeout,
		PollInterval: s.cfg.ScanPollInterval,
		Logf:         s.logf,
	}

	present, err := vmhelpers.WaitForV2VMPresentInCentral(waitCtx, s.vmV2Client, baseOpts, vm.Namespace, vm.Name)
	if err != nil {
		return nil, err
	}
	vm.ID = present.GetId()

	s.logf("%s/%s: VM appeared in Central via VirtualMachineV2Service (id=%q), waiting for namespace/name fields", vm.Namespace, vm.Name, vm.ID)
	if _, err := vmhelpers.WaitForV2VMIdentityFields(waitCtx, s.vmV2Client, baseOpts, present.GetId(), vm.Namespace, vm.Name); err != nil {
		return nil, err
	}
	s.logf("%s/%s: waiting for Central to report VM as VM_STATE_RUNNING", vm.Namespace, vm.Name)
	if _, err := vmhelpers.WaitForV2VMRunningInCentral(waitCtx, s.vmV2Client, baseOpts, present.GetId()); err != nil {
		return nil, err
	}
	s.logf("%s/%s: waiting for latest_scan to arrive in Central", vm.Namespace, vm.Name)
	if _, err := vmhelpers.WaitForV2VMLatestScan(waitCtx, s.vmV2Client, baseOpts, present.GetId()); err != nil {
		return nil, err
	}
	s.logf("%s/%s: waiting for all v2 components to be vulnerability-matched (no NOT_SCANNED)", vm.Namespace, vm.Name)
	return vmhelpers.WaitForV2ScanReady(waitCtx, s.vmV2Client, baseOpts, present.GetId())
}

func (s *VMScanningSuite) resourceDeleteTimeout() time.Duration {
	if s.cfg != nil && s.cfg.DeleteTimeout > 0 {
		return s.cfg.DeleteTimeout
	}
	return defaultVMDeleteTimeout
}

func (s *VMScanningSuite) prepareGuest(vm *VMHandle) error {
	virt := s.virtctlForVM(*vm)
	stepNum := 0
	runStep := func(stepName, errContext string, timeout time.Duration, fn func(stepCtx context.Context) error) error {
		stepNum++
		s.logf("[guest preparation step %02d]: %s on %s/%s (timeout=%v)",
			stepNum, stepName, vm.Namespace, vm.Name, timeout)
		stepCtx, cancel := context.WithTimeout(s.ctx, timeout)
		defer cancel()
		if err := fn(stepCtx); err != nil {
			return fmt.Errorf("prepare guest %s/%s: %s: %w", vm.Namespace, vm.Name, errContext, err)
		}
		return nil
	}

	sshTimeout := s.sshFirstContactTimeout()
	stepTimeout := s.guestStepTimeout()

	if err := runStep("Wait for SSH to become reachable", "WaitForSSHReachable", sshTimeout, func(stepCtx context.Context) error {
		return vmhelpers.WaitForSSHReachableWithPolicy(s.T(), stepCtx, virt, vm.Namespace, vm.Name, vmhelpers.FirstContactSSHPolicy)
	}); err != nil {
		return err
	}
	if err := runStep("Wait for cloud-init to finish", "WaitForCloudInitFinished", stepTimeout, func(stepCtx context.Context) error {
		return vmhelpers.WaitForCloudInitFinished(stepCtx, virt, vm.Namespace, vm.Name)
	}); err != nil {
		return err
	}
	if err := runStep("Verify sudo", "VerifySudoWorks", stepTimeout, func(stepCtx context.Context) error {
		return vmhelpers.VerifySudoWorks(stepCtx, virt, vm.Namespace, vm.Name)
	}); err != nil {
		return err
	}
	if err := runStep("Install roxagent Quadlet", "InstallRoxagentQuadlet", max(stepTimeout, 15*time.Minute), func(stepCtx context.Context) error {
		return vmhelpers.InstallRoxagentQuadlet(stepCtx, virt, vm.Namespace, vm.Name, s.cfg.RoxagentImage, s.cfg.Repo2CPEURL, s.cfg.PodmanAuthFilePath)
	}); err != nil {
		if errors.Is(err, vmhelpers.ErrPodmanNotFound) {
			vm.SkipReason = err.Error()
			s.logf("[guest prep] Quadlet install skipped on %s/%s; VM subtest will be skipped: %v",
				vm.Namespace, vm.Name, err)
			return nil
		}
		return err
	}
	s.logf("[guest prep] COMPLETED for %s/%s in %d step(s)", vm.Namespace, vm.Name, stepNum)
	return nil
}
