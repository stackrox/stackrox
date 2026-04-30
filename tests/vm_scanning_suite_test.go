//go:build test_e2e

package tests

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	v1 "github.com/stackrox/rox/generated/api/v1"
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
	"google.golang.org/protobuf/types/known/timestamppb"
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
	// run after SSH is confirmed working (cloud-init wait, sudo check,
	// activation, roxagent install).
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

	// namespacePollInterval is the polling cadence for waiting on namespace
	// deletion during teardown.
	namespacePollInterval = 2 * time.Second
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
}

// VMScanningSuite exercises OpenShift VM scanning end-to-end (KubeVirt guests, roxagent, Central).
type VMScanningSuite struct {
	KubernetesSuite

	ctx        context.Context
	cleanupCtx context.Context
	cancel     func()

	cfg           *vmScanConfig
	restCfg       *rest.Config
	k8sClient     kubernetes.Interface
	dynamicClient dynamic.Interface
	namespace     string

	conn     *grpc.ClientConn
	vmClient v2.VirtualMachineServiceClient

	virtctl vmhelpers.Virtctl

	// vmSpecs is the provisioning blueprint for each persistent VM.
	vmSpecs []vmSpec
	// persistentVMs are the long-lived guests provisioned in SetupSuite.
	persistentVMs []VMHandle
	// allVMs tracks every VM provisioned by the suite; TearDownSuite deletes each.
	allVMs []VMHandle
	// terminalVSOCKFailure captures a hard vsock/device failure; subsequent tests are skipped.
	terminalVSOCKFailure string
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

	s.logf("VM scanning setup: verify central/sensor connectivity and feature gates")
	s.mustWaitForHealthyCentralSensorConnection()
	s.mustVerifyVirtualMachinesFeatureEnabled()
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
		KnownHostsFile:    vmhelpers.CreateKnownHostsFile(t),
		Logf:              s.logf,
		HeartbeatInterval: defaultVirtctlHeartbeatInterval,
	}

	s.vmSpecs = s.cfg.vmSpecs()
	s.logf("VM scanning setup: provision persistent VMs (%d specs)", len(s.vmSpecs))
	s.provisionPersistentVMs(s.vmSpecs)
	s.logf("VM scanning setup: prepare guests (ssh/cloud-init/roxagent/activation)")
	s.preparePersistentGuests()
	s.logf("VM scanning setup: complete")
}

func (s *VMScanningSuite) BeforeTest(_, testName string) {
	if s.terminalVSOCKFailure == "" {
		return
	}
	s.T().Skipf("skipping %s due to prior terminal vsock failure: %s", testName, s.terminalVSOCKFailure)
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

	deleteTimeout := s.vmDeleteTimeout()
	if s.dynamicClient != nil {
		for _, vm := range s.allVMs {
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
	} else if len(s.allVMs) > 0 {
		s.logf("teardown: skipping VM cleanup (%d handle(s)): dynamic client is nil", len(s.allVMs))
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
	if u := strings.TrimSpace(vm.GuestUser); u != "" {
		s.virtctl.Username = u
	}
	return s.virtctl
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
	return wait.PollUntilContextCancel(ctx, namespacePollInterval, true, func(ctx context.Context) (bool, error) {
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

func (s *VMScanningSuite) mustVerifyVirtualMachinesFeatureEnabled() {
	ctx, cancel := context.WithTimeout(s.ctx, featureGateVerifyTimeout)
	defer cancel()

	wantEnv := features.VirtualMachines.EnvVar()
	ns := namespaces.StackRox

	// Verify the feature flag env var is set on all components that need it.
	s.mustVerifyContainerEnvVar(ctx, "deployment", "central", "central", ns, wantEnv)
	s.mustVerifyContainerEnvVar(ctx, "deployment", sensorDeployment, sensorContainer, ns, wantEnv)
	s.mustVerifyContainerEnvVar(ctx, "daemonset", "collector", "compliance", ns, wantEnv)
}

// mustVerifyContainerEnvVar asserts that the named container within a Deployment or DaemonSet
// has the given environment variable set to a truthy value ("true", "1", etc.).
// This catches deployment misconfigurations where a feature flag reaches Central but not
// the workload containers that also need it.
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
					"%s %s/%s container %q has %s=%q which is not truthy; "+
						"the VSOCK relay will not start without this flag",
					kind, ns, name, containerName, envName, e.Value)
				return
			}
		}
		require.Failf(t, fmt.Sprintf("%s %s/%s container %q is missing env var %s", kind, ns, name, containerName, envName),
			"the feature flag must be set on all components that need it (Central, Sensor, compliance); "+
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

func (s *VMScanningSuite) provisionPersistentVMs(specs []vmSpec) {
	ctx := s.ctx
	createdNow := make(map[string]bool)

	s.logf("provision persistent VMs: creating namespace %q", s.namespace)
	_, err := s.k8sClient.CoreV1().Namespaces().Create(ctx, &coreV1.Namespace{
		ObjectMeta: metaV1.ObjectMeta{Name: s.namespace},
	}, metaV1.CreateOptions{})
	if err != nil && !apierrors.IsAlreadyExists(err) {
		require.NoError(s.T(), err, "create test namespace %q", s.namespace)
	}
	if apierrors.IsAlreadyExists(err) {
		s.logf("provision persistent VMs: namespace %q already exists; reusing it", s.namespace)
	}

	s.ensureImagePullSecret(ctx)

	for _, sp := range specs {
		req := vmhelpers.VMRequest{
			Name:         sp.Name,
			Namespace:    s.namespace,
			Image:        sp.Image,
			GuestUser:    sp.GuestUser,
			SSHPublicKey: s.cfg.SSHPublicKey,
		}
		s.logf("provision persistent VMs: ensuring VM exists %s/%s with image %q", s.namespace, sp.Name, sp.Image)
		createErr := vmhelpers.CreateVirtualMachine(ctx, s.dynamicClient, req)
		if createErr == nil {
			s.logf("provision persistent VMs: created VM %s/%s", s.namespace, sp.Name)
			createdNow[sp.Name] = true
		} else if apierrors.IsAlreadyExists(createErr) {
			currentImage, imgErr := vmhelpers.GetVMContainerDiskImage(ctx, s.dynamicClient, s.namespace, sp.Name)
			if imgErr != nil {
				s.logf("provision persistent VMs: could not read image for existing VM %s/%s: %v; recreating", s.namespace, sp.Name, imgErr)
			}
			if imgErr != nil || currentImage != sp.Image {
				if imgErr == nil {
					s.logf("provision persistent VMs: VM %s/%s has image %q but want %q; deleting and recreating",
						s.namespace, sp.Name, currentImage, sp.Image)
				}
				delCtx, delCancel := context.WithTimeout(ctx, s.vmDeleteTimeout())
				require.NoError(s.T(), vmhelpers.DeleteVirtualMachine(delCtx, s.dynamicClient, s.namespace, sp.Name),
					"DeleteVirtualMachine %s/%s for image mismatch", s.namespace, sp.Name)
				require.NoError(s.T(), vmhelpers.WaitForVirtualMachineDeleted(s.T(), delCtx, s.dynamicClient, s.namespace, sp.Name),
					"WaitForVirtualMachineDeleted %s/%s for image mismatch", s.namespace, sp.Name)
				delCancel()
				require.NoError(s.T(), vmhelpers.CreateVirtualMachine(ctx, s.dynamicClient, req),
					"CreateVirtualMachine %s/%s after image mismatch delete", s.namespace, sp.Name)
				s.logf("provision persistent VMs: recreated VM %s/%s with correct image", s.namespace, sp.Name)
				createdNow[sp.Name] = true
			} else {
				s.logf("provision persistent VMs: VM %s/%s already exists with correct image; reusing it", s.namespace, sp.Name)
				createdNow[sp.Name] = false
			}
		} else {
			require.NoError(s.T(), createErr, "EnsureVirtualMachineExists %s/%s", s.namespace, sp.Name)
		}
		h := VMHandle{Name: sp.Name, Namespace: s.namespace, GuestUser: sp.GuestUser}
		s.persistentVMs = append(s.persistentVMs, h)
		s.allVMs = append(s.allVMs, h)
	}
	for i := range s.persistentVMs {
		vm := &s.persistentVMs[i]
		vmCtx, vmCancel := context.WithTimeout(ctx, s.vmProvisionTimeout())
		s.logf("provision persistent VMs: waiting for VMI object %s/%s (timeout=%v)", vm.Namespace, vm.Name, s.vmProvisionTimeout())
		require.NoError(s.T(), vmhelpers.WaitForVirtualMachineInstanceExists(s.T(), vmCtx, s.dynamicClient, vm.Namespace, vm.Name),
			"WaitForVirtualMachineInstanceExists %s/%s", vm.Namespace, vm.Name)
		s.logf("provision persistent VMs: waiting for VMI Running %s/%s (timeout=%v)", vm.Namespace, vm.Name, s.vmProvisionTimeout())
		require.NoError(s.T(), vmhelpers.WaitForVirtualMachineInstanceRunning(s.T(), vmCtx, s.dynamicClient, vm.Namespace, vm.Name),
			"WaitForVirtualMachineInstanceRunning %s/%s", vm.Namespace, vm.Name)
		vmCancel()

		nodeName, err := vmhelpers.GetVMINodeName(ctx, s.dynamicClient, vm.Namespace, vm.Name)
		if err != nil {
			s.logf("provision persistent VMs: could not determine node for %s/%s: %v", vm.Namespace, vm.Name, err)
		} else {
			vm.NodeName = nodeName
			s.allVMs[i].NodeName = nodeName
		}

	}

	s.logf("VM placement:\n%s", s.vmPlacementSummary(ctx))
}

const vmImagePullSecretName = "vm-image-pull-secret" //nolint:gosec // G101: not a credential, just the k8s Secret resource name

func (s *VMScanningSuite) ensureImagePullSecret(ctx context.Context) {
	if s.cfg.ImagePullSecretPath == "" {
		return
	}
	s.logf("provision persistent VMs: creating image pull secret from %q", s.cfg.ImagePullSecretPath)
	dockerCfg, err := os.ReadFile(s.cfg.ImagePullSecretPath)
	require.NoError(s.T(), err, "read image pull secret file %q", s.cfg.ImagePullSecretPath)

	secret := &coreV1.Secret{
		ObjectMeta: metaV1.ObjectMeta{Name: vmImagePullSecretName},
		Type:       coreV1.SecretTypeDockerConfigJson,
		Data:       map[string][]byte{coreV1.DockerConfigJsonKey: dockerCfg},
	}
	_, err = s.k8sClient.CoreV1().Secrets(s.namespace).Create(ctx, secret, metaV1.CreateOptions{})
	if apierrors.IsAlreadyExists(err) {
		_, err = s.k8sClient.CoreV1().Secrets(s.namespace).Update(ctx, secret, metaV1.UpdateOptions{})
	}
	require.NoError(s.T(), err, "ensure image pull secret %q in namespace %q", vmImagePullSecretName, s.namespace)

	sa, err := s.k8sClient.CoreV1().ServiceAccounts(s.namespace).Get(ctx, "default", metaV1.GetOptions{})
	require.NoError(s.T(), err, "get default service account in namespace %q", s.namespace)
	hasRef := false
	for _, ref := range sa.ImagePullSecrets {
		if ref.Name == vmImagePullSecretName {
			hasRef = true
			break
		}
	}
	if !hasRef {
		sa.ImagePullSecrets = append(sa.ImagePullSecrets, coreV1.LocalObjectReference{Name: vmImagePullSecretName})
		_, err = s.k8sClient.CoreV1().ServiceAccounts(s.namespace).Update(ctx, sa, metaV1.UpdateOptions{})
		require.NoError(s.T(), err, "link image pull secret to default service account in namespace %q", s.namespace)
	}
	s.logf("provision persistent VMs: image pull secret %q ready in namespace %q", vmImagePullSecretName, s.namespace)
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
	for _, vm := range s.persistentVMs {
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

func (s *VMScanningSuite) preparePersistentGuests() {
	t := s.T()
	for i := range s.persistentVMs {
		require.NoError(t, s.preparePersistentGuestWithRecovery(&s.persistentVMs[i]))
	}
}

func (s *VMScanningSuite) preparePersistentGuestWithRecovery(vm *VMHandle) error {
	const maxRecoveries = 2
	for recoveryAttempt := 0; recoveryAttempt <= maxRecoveries; recoveryAttempt++ {
		err := s.prepareGuest(*vm)
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
		if recreateErr := s.recreatePersistentVM(vm); recreateErr != nil {
			return fmt.Errorf("recreate persistent VM %s/%s after recoverable SSH failure: %w", vm.Namespace, vm.Name, recreateErr)
		}
	}
	return nil
}

func (s *VMScanningSuite) recreatePersistentVM(vm *VMHandle) error {
	req, err := s.vmRequestForExistingPersistentVM(*vm)
	if err != nil {
		return err
	}

	delCtx, delCancel := context.WithTimeout(s.ctx, s.vmDeleteTimeout())
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
	s.syncVMHandleToAllVMs(*vm)
	return nil
}

// syncVMHandleToAllVMs propagates field updates from a VMHandle to the matching entry in allVMs.
func (s *VMScanningSuite) syncVMHandleToAllVMs(vm VMHandle) {
	for i := range s.allVMs {
		if s.allVMs[i].Name == vm.Name && s.allVMs[i].Namespace == vm.Namespace {
			s.allVMs[i] = vm
			return
		}
	}
}

func (s *VMScanningSuite) vmRequestForExistingPersistentVM(vm VMHandle) (vmhelpers.VMRequest, error) {
	for _, sp := range s.vmSpecs {
		if sp.Name == vm.Name {
			guestUser := strings.TrimSpace(vm.GuestUser)
			if guestUser == "" {
				guestUser = sp.GuestUser
			}
			return vmhelpers.VMRequest{
				Name:         vm.Name,
				Namespace:    vm.Namespace,
				Image:        sp.Image,
				GuestUser:    guestUser,
				SSHPublicKey: s.cfg.SSHPublicKey,
			}, nil
		}
	}
	return vmhelpers.VMRequest{}, fmt.Errorf("no spec found for persistent VM %s/%s", vm.Namespace, vm.Name)
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

const maxRoxagentStderrInError = 4096

func formatRoxagentStderrForError(stderr string) string {
	s := strings.TrimSpace(stderr)
	if len(s) > maxRoxagentStderrInError {
		return s[:maxRoxagentStderrInError] + fmt.Sprintf(" ... (truncated from %d bytes)", len(s))
	}
	return s
}

// roxagentStderrCrashLinePrefixes are lowercase; each stderr line is trimmed and lowercased before HasPrefix.
var roxagentStderrCrashLinePrefixes = []string{
	"panic:",
	"fatal error:",
	"runtime error:",
}

// validateRoxagentSuccessStderr allows empty or benign stderr but fails only on lines that start with
// unambiguous Go/runtime crash signatures (after trim + lowercase), avoiding substring false positives.
func validateRoxagentSuccessStderr(stderr string) error {
	if strings.TrimSpace(stderr) == "" {
		return nil
	}
	for _, line := range strings.Split(stderr, "\n") {
		ln := strings.TrimSpace(strings.ToLower(line))
		for _, prefix := range roxagentStderrCrashLinePrefixes {
			if strings.HasPrefix(ln, prefix) {
				return fmt.Errorf("ensureCanonicalScan: roxagent stderr indicates process/runtime failure (matched line prefix %q): %s", prefix, formatRoxagentStderrForError(stderr))
			}
		}
	}
	return nil
}

func (s *VMScanningSuite) persistRoxagentStdout(vm *VMHandle, stdout string) string {
	if vm == nil || strings.TrimSpace(stdout) == "" {
		return ""
	}
	f, err := os.CreateTemp(s.T().TempDir(), fmt.Sprintf("roxagent-%s-%s-*.stdout", vm.Namespace, vm.Name))
	if err != nil {
		s.logf("ensureCanonicalScan: could not persist roxagent stdout for %s/%s: %v", vm.Namespace, vm.Name, err)
		return ""
	}
	defer func() { _ = f.Close() }()
	if _, err := f.WriteString(stdout); err != nil {
		s.logf("ensureCanonicalScan: could not write roxagent stdout file %q: %v", f.Name(), err)
		return ""
	}
	return f.Name()
}

// waitForScannerV4Initialized blocks until the Scanner V4 matcher has finished
// loading its vulnerability database. It polls Central's GetVulnDefinitionsInfo API
// (which internally calls GetMatcherMetadata on the matcher) until a non-zero
// last-updated timestamp is returned, meaning the vuln store is ready.
//
// Called once (guarded by scannerV4Checked) right before the first roxagent invocation
// so that the matcher initialization happens in parallel with VM boot and SSH readiness.
func (s *VMScanningSuite) waitForScannerV4Initialized() error {
	s.logf("Scanner V4: waiting for matcher deployment to be K8s-ready")
	s.waitUntilK8sDeploymentReady(s.ctx, namespaces.StackRox, "scanner-v4-matcher")

	s.logf("Scanner V4: polling Central for matcher vuln DB initialization")
	healthClient := v1.NewIntegrationHealthServiceClient(s.conn)

	ctx, cancel := context.WithTimeout(s.ctx, 20*time.Minute)
	defer cancel()

	return wait.PollUntilContextCancel(ctx, 15*time.Second, true, func(ctx context.Context) (bool, error) {
		info, err := healthClient.GetVulnDefinitionsInfo(ctx, &v1.VulnDefinitionsInfoRequest{
			Component: v1.VulnDefinitionsInfoRequest_SCANNER_V4,
		})
		if err != nil {
			s.logf("Scanner V4: matcher not yet initialized: %v", err)
			return false, nil
		}
		ts := info.GetLastUpdatedTimestamp().AsTime()
		if ts.IsZero() {
			s.logf("Scanner V4: vuln definitions timestamp is zero, still loading")
			return false, nil
		}
		s.logf("Scanner V4: matcher initialized (vuln defs last updated: %v)", ts)
		return true, nil
	})
}

// ensureCanonicalScan runs a single guest-side roxagent invocation and validates failure signals.
// It verifies the ROX_VIRTUAL_MACHINES feature flag is enabled before triggering the scan.
func (s *VMScanningSuite) ensureCanonicalScan(ctx context.Context, vm *VMHandle) (*vmhelpers.RoxagentRunResult, error) {
	if vm == nil {
		return nil, errors.New("ensureCanonicalScan: nil VM handle")
	}
	s.mustVerifyVirtualMachinesFeatureEnabled()
	if !s.scannerV4Checked {
		if err := s.waitForScannerV4Initialized(); err != nil {
			return nil, fmt.Errorf("Scanner V4 matcher did not initialize within timeout: %w", err)
		}
		s.scannerV4Checked = true
	}
	virt := s.virtctlForVM(*vm)
	cfg := vmhelpers.RoxagentRunConfig{
		Repo2CPEPrimaryURL:      s.cfg.Repo2CPEPrimaryURL,
		Repo2CPEFallbackURL:     s.cfg.Repo2CPEFallbackURL,
		Repo2CPEPrimaryAttempts: s.cfg.Repo2CPEPrimaryAttempts,
	}
	res, err := vmhelpers.RunRoxagentOnce(ctx, virt, vm.Namespace, vm.Name, cfg)
	if err != nil {
		if vmhelpers.IsTerminalVSOCKUnavailableError(err) {
			s.terminalVSOCKFailure = fmt.Sprintf("%s/%s: %v", vm.Namespace, vm.Name, err)
			s.logf("terminal vsock failure detected; skipping remaining suite tests: %s", s.terminalVSOCKFailure)
			return nil, fmt.Errorf("ensureCanonicalScan: terminal vsock failure for %s/%s; subsequent suite tests will be skipped: %w",
				vm.Namespace, vm.Name, err)
		}
		return nil, err
	}
	if res == nil {
		return nil, errors.New("ensureCanonicalScan: nil result from RunRoxagentOnce")
	}
	stdoutPath := s.persistRoxagentStdout(vm, res.Stdout)
	if stdoutPath != "" {
		s.logf("ensureCanonicalScan: roxagent stdout saved to %q (%d bytes)", stdoutPath, len(res.Stdout))
		if !vmhelpers.VerboseOutputLooksLikeReport(res.Stdout) {
			s.logf("ensureCanonicalScan: roxagent stdout on %s/%s does not match known report shapes; continuing (stdout_file=%q)",
				vm.Namespace, vm.Name, stdoutPath)
		}
	} else {
		s.logf("ensureCanonicalScan: roxagent stdout empty on %s/%s (non-verbose mode)", vm.Namespace, vm.Name)
	}
	if err := validateRoxagentSuccessStderr(res.Stderr); err != nil {
		return nil, err
	}
	return res, nil
}

// waitForScan polls Central in order until scan data is visible.
func (s *VMScanningSuite) waitForScan(ctx context.Context, vm *VMHandle) (*v2.VirtualMachine, error) {
	if vm == nil {
		return nil, errors.New("waitForScan: nil VM handle")
	}
	s.logf("scan wait %s/%s: start (timeout=%v poll=%v)", vm.Namespace, vm.Name, s.cfg.ScanTimeout, s.cfg.ScanPollInterval)
	waitCtx, cancel := context.WithTimeout(ctx, s.cfg.ScanTimeout)
	defer cancel()

	baseOpts := vmhelpers.WaitOptions{
		Timeout:      s.cfg.ScanTimeout,
		PollInterval: s.cfg.ScanPollInterval,
		Logf:         s.logf,
	}

	s.logf("scan wait %s/%s step 1/6: wait VM present in Central", vm.Namespace, vm.Name)
	present, err := vmhelpers.WaitForVMPresentInCentral(waitCtx, s.vmClient, baseOpts, vm.Namespace, vm.Name)
	if err != nil {
		return nil, err
	}
	vm.ID = present.GetId()
	s.logf("scan wait %s/%s step 1/6 complete: id=%q", vm.Namespace, vm.Name, vm.ID)

	s.logf("scan wait %s/%s step 2/6: wait VM identity fields", vm.Namespace, vm.Name)
	if _, err := vmhelpers.WaitForVMIdentityFields(waitCtx, s.vmClient, baseOpts, present.GetId(), vm.Namespace, vm.Name); err != nil {
		return nil, err
	}
	s.logf("scan wait %s/%s step 2/6 complete", vm.Namespace, vm.Name)
	s.logf("scan wait %s/%s step 3/6: wait VM running in Central", vm.Namespace, vm.Name)
	if _, err := vmhelpers.WaitForVMRunningInCentral(waitCtx, s.vmClient, baseOpts, present.GetId()); err != nil {
		return nil, err
	}
	s.logf("scan wait %s/%s step 3/6 complete", vm.Namespace, vm.Name)
	s.logf("scan wait %s/%s step 4/6: wait non-nil scan", vm.Namespace, vm.Name)
	if _, err := vmhelpers.WaitForVMScanNonNil(waitCtx, s.vmClient, baseOpts, present.GetId()); err != nil {
		return nil, err
	}
	s.logf("scan wait %s/%s step 4/6 complete", vm.Namespace, vm.Name)
	s.logf("scan wait %s/%s step 5/6: wait scan timestamp", vm.Namespace, vm.Name)
	if _, err := vmhelpers.WaitForVMScanTimestamp(waitCtx, s.vmClient, baseOpts, present.GetId()); err != nil {
		return nil, err
	}
	s.logf("scan wait %s/%s step 5/6 complete", vm.Namespace, vm.Name)

	conds := vmhelpers.ScanReadiness{Components: true, AllScanned: true}
	s.logf("scan wait %s/%s step 6/6: wait scan ready (components=%v all-scanned=%v)",
		vm.Namespace, vm.Name, conds.Components, conds.AllScanned)
	return vmhelpers.WaitForScanReady(waitCtx, s.vmClient, baseOpts, present.GetId(), conds)
}

func (s *VMScanningSuite) vmDeleteTimeout() time.Duration {
	if s.cfg != nil && s.cfg.DeleteTimeout > 0 {
		return s.cfg.DeleteTimeout
	}
	return defaultVMDeleteTimeout
}

func (s *VMScanningSuite) mustGetScanTimestamp(id string) *timestamppb.Timestamp {
	t := s.T()
	t.Helper()
	vm := s.mustGetVM(id)
	require.NotNil(t, vm.GetScan(), "mustGetScanTimestamp: GetVirtualMachine id=%q returned nil scan", id)
	ts := vm.GetScan().GetScanTime()
	require.NotNil(t, ts, "mustGetScanTimestamp: GetVirtualMachine id=%q scan_time is nil", id)
	return ts
}

func (s *VMScanningSuite) prepareGuest(vm VMHandle) error {
	virt := s.virtctlForVM(vm)
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
	if err := runStep("Copy roxagent binary", "CopyRoxagentBinary", stepTimeout, func(stepCtx context.Context) error {
		return vmhelpers.CopyRoxagentBinary(stepCtx, virt, vm.Namespace, vm.Name, s.cfg.RoxagentBinaryPath)
	}); err != nil {
		return err
	}
	if err := runStep("Verify roxagent binary presence", "VerifyRoxagentBinaryPresent", stepTimeout, func(stepCtx context.Context) error {
		return vmhelpers.VerifyRoxagentBinaryPresent(stepCtx, virt, vm.Namespace, vm.Name)
	}); err != nil {
		return err
	}
	if err := runStep("Verify roxagent executable mode", "VerifyRoxagentExecutable", stepTimeout, func(stepCtx context.Context) error {
		return vmhelpers.VerifyRoxagentExecutable(stepCtx, virt, vm.Namespace, vm.Name)
	}); err != nil {
		return err
	}
	if err := runStep("Verify roxagent install path", "VerifyRoxagentInstallPath", stepTimeout, func(stepCtx context.Context) error {
		return vmhelpers.VerifyRoxagentInstallPath(stepCtx, virt, vm.Namespace, vm.Name)
	}); err != nil {
		return err
	}
	var (
		activated   bool
		activStatus string
	)
	if err := runStep("Check activation status", "GetActivationStatus", stepTimeout, func(stepCtx context.Context) error {
		var innerErr error
		activated, activStatus, innerErr = vmhelpers.GetActivationStatus(stepCtx, virt, vm.Namespace, vm.Name)
		return innerErr
	}); err != nil {
		return err
	}
	s.logf("[guest prep] activation status for %s/%s: activated=%v detail=%q", vm.Namespace, vm.Name, activated, activStatus)
	s.logf("[guest prep] COMPLETED for %s/%s in %d step(s)", vm.Namespace, vm.Name, stepNum)
	return nil
}
