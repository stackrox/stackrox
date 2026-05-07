//go:build test_e2e || test_e2e_vm

package tests

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync/atomic"
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
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	kubefake "k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/rest"
	k8stesting "k8s.io/client-go/testing"
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

const (
	vmImagePullSecretName = "vm-image-pull-secret" //nolint:gosec // G101: not a credential, just the k8s Secret resource name

	defaultServiceAccountWaitTimeout  = 10 * time.Second
	defaultServiceAccountPollInterval = 100 * time.Millisecond
)

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
		existingSecret, getErr := s.k8sClient.CoreV1().Secrets(s.namespace).Get(ctx, vmImagePullSecretName, metaV1.GetOptions{})
		require.NoError(s.T(), getErr, "get existing image pull secret %q in namespace %q", vmImagePullSecretName, s.namespace)
		existingSecret.Type = coreV1.SecretTypeDockerConfigJson
		if existingSecret.Data == nil {
			existingSecret.Data = make(map[string][]byte)
		}
		existingSecret.Data[coreV1.DockerConfigJsonKey] = dockerCfg
		_, err = s.k8sClient.CoreV1().Secrets(s.namespace).Update(ctx, existingSecret, metaV1.UpdateOptions{})
	}
	require.NoError(s.T(), err, "ensure image pull secret %q in namespace %q", vmImagePullSecretName, s.namespace)

	sa, err := s.waitForDefaultServiceAccount(ctx)
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

func (s *VMScanningSuite) waitForDefaultServiceAccount(ctx context.Context) (*coreV1.ServiceAccount, error) {
	waitCtx, cancel := context.WithTimeout(ctx, defaultServiceAccountWaitTimeout)
	defer cancel()

	var serviceAccount *coreV1.ServiceAccount
	err := wait.PollUntilContextCancel(waitCtx, defaultServiceAccountPollInterval, true, func(ctx context.Context) (bool, error) {
		sa, err := s.k8sClient.CoreV1().ServiceAccounts(s.namespace).Get(ctx, "default", metaV1.GetOptions{})
		if apierrors.IsNotFound(err) {
			return false, nil
		}
		if err != nil {
			return false, err
		}
		serviceAccount = sa
		return true, nil
	})
	if err != nil {
		return nil, err
	}
	return serviceAccount, nil
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

func (s *VMScanningSuite) vmDeleteTimeout() time.Duration {
	if s.cfg != nil && s.cfg.DeleteTimeout > 0 {
		return s.cfg.DeleteTimeout
	}
	return defaultVMDeleteTimeout
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

func writeDockerConfigFile(t *testing.T, content string) string {
	t.Helper()
	path := t.TempDir() + "/config.json"
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))
	return path
}

func newVMScanningSuiteForPullSecretTest(t *testing.T, client *kubefake.Clientset, namespace, secretPath string) *VMScanningSuite {
	t.Helper()
	s := &VMScanningSuite{
		cfg: &vmScanConfig{
			ImagePullSecretPath: secretPath,
		},
		k8sClient: client,
		namespace: namespace,
	}
	s.SetT(t)
	return s
}

func TestEnsureImagePullSecret_UpdatesExistingSecretUsingFetchedResourceVersion(t *testing.T) {
	t.Parallel()

	const namespace = "vm-scan-test"
	secretPath := writeDockerConfigFile(t, `{"auths":{"quay.io":{"auth":"new"}}}`)

	client := kubefake.NewSimpleClientset(
		&coreV1.Secret{
			ObjectMeta: metaV1.ObjectMeta{
				Name:            vmImagePullSecretName,
				Namespace:       namespace,
				ResourceVersion: "7",
			},
			Type: coreV1.SecretTypeDockerConfigJson,
			Data: map[string][]byte{
				coreV1.DockerConfigJsonKey: []byte(`{"auths":{"quay.io":{"auth":"old"}}}`),
			},
		},
		&coreV1.ServiceAccount{
			ObjectMeta: metaV1.ObjectMeta{
				Name:      "default",
				Namespace: namespace,
			},
		},
	)
	client.PrependReactor("update", "secrets", func(action k8stesting.Action) (bool, runtime.Object, error) {
		updateAction := action.(k8stesting.UpdateAction)
		secret := updateAction.GetObject().(*coreV1.Secret)
		if secret.ResourceVersion == "" {
			return true, nil, apierrors.NewInvalid(
				coreV1.SchemeGroupVersion.WithKind("Secret").GroupKind(),
				secret.Name,
				field.ErrorList{field.Required(field.NewPath("metadata", "resourceVersion"), "must be set for an update")},
			)
		}
		return false, nil, nil
	})

	s := newVMScanningSuiteForPullSecretTest(t, client, namespace, secretPath)

	s.ensureImagePullSecret(t.Context())

	secret, err := client.CoreV1().Secrets(namespace).Get(t.Context(), vmImagePullSecretName, metaV1.GetOptions{})
	require.NoError(t, err)
	require.Equal(t, "7", secret.ResourceVersion)
	require.Equal(t, `{"auths":{"quay.io":{"auth":"new"}}}`, string(secret.Data[coreV1.DockerConfigJsonKey]))

	sa, err := client.CoreV1().ServiceAccounts(namespace).Get(t.Context(), "default", metaV1.GetOptions{})
	require.NoError(t, err)
	require.Contains(t, sa.ImagePullSecrets, coreV1.LocalObjectReference{Name: vmImagePullSecretName})
}

func TestEnsureImagePullSecret_WaitsForDefaultServiceAccountToAppear(t *testing.T) {
	t.Parallel()

	const namespace = "vm-scan-test"
	secretPath := writeDockerConfigFile(t, `{"auths":{"quay.io":{"auth":"new"}}}`)
	client := kubefake.NewSimpleClientset()

	var getAttempts atomic.Int32
	client.PrependReactor("get", "serviceaccounts", func(action k8stesting.Action) (bool, runtime.Object, error) {
		getAction := action.(k8stesting.GetAction)
		if getAction.GetName() != "default" || getAction.GetNamespace() != namespace {
			return false, nil, nil
		}

		if getAttempts.Add(1) == 1 {
			require.NoError(t, client.Tracker().Add(&coreV1.ServiceAccount{
				ObjectMeta: metaV1.ObjectMeta{
					Name:      "default",
					Namespace: namespace,
				},
			}))
			return true, nil, apierrors.NewNotFound(coreV1.Resource("serviceaccounts"), "default")
		}
		return false, nil, nil
	})

	s := newVMScanningSuiteForPullSecretTest(t, client, namespace, secretPath)

	s.ensureImagePullSecret(t.Context())

	require.GreaterOrEqual(t, getAttempts.Load(), int32(2))
	sa, err := client.CoreV1().ServiceAccounts(namespace).Get(t.Context(), "default", metaV1.GetOptions{})
	require.NoError(t, err)
	require.Contains(t, sa.ImagePullSecrets, coreV1.LocalObjectReference{Name: vmImagePullSecretName})
}
