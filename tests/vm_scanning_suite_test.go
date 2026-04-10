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
	"github.com/stackrox/rox/pkg/namespaces"
	"github.com/stackrox/rox/pkg/testutils/centralgrpc"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stackrox/rox/tests/testmetrics"
	"github.com/stackrox/rox/tests/vmhelpers"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"
	coreV1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	kubevirtv1 "kubevirt.io/api/core/v1"
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

	// persistentVMs are the two long-lived RHEL 9 / RHEL 10 guests provisioned in SetupSuite.
	persistentVMs []VMHandle
	// allVMs tracks every VM provisioned by the suite; TearDownSuite deletes each.
	allVMs []VMHandle
	// vmCreatedAt tracks when this suite created/recreated a VM (by namespace/name key).
	vmCreatedAt map[string]time.Time
	// terminalVSOCKFailure captures a hard vsock/device failure; subsequent tests are skipped.
	terminalVSOCKFailure string
}

// TestVMScanning is the suite entrypoint for VM scanning E2E tests.
func TestVMScanning(t *testing.T) {
	suite.Run(t, new(VMScanningSuite))
}

func (s *VMScanningSuite) SetupSuite() {
	s.KubernetesSuite.SetupSuite()
	t := s.T()

	s.logf("VM scanning setup: initialize test contexts")
	s.ctx, s.cleanupCtx, s.cancel = testContexts(t, "TestVMScanning", 90*time.Minute)

	s.logf("VM scanning setup: load suite configuration from environment")
	s.cfg = mustLoadVMScanConfig(t)
	s.logf("VM scanning setup: create Kubernetes clients")
	s.restCfg = getConfig(t)
	s.k8sClient = createK8sClientWithConfig(t, s.restCfg)
	s.dynamicClient = mustCreateDynamicClient(t, s.restCfg)

	s.logf("VM scanning setup: ensure compliance metrics are exposed")
	s.ensureComplianceMetricsExposed()

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
	s.logf("VM_SSH_PUBLIC_KEY_PATH=%q", resolveSSHPublicKeyPathForLog(identity, s.cfg.SSHPublicKey))
	cmdTimeout := 30 * time.Minute
	if s.cfg.ScanTimeout > 0 && s.cfg.ScanTimeout < cmdTimeout {
		cmdTimeout = s.cfg.ScanTimeout
	}
	s.virtctl = vmhelpers.Virtctl{
		Path:              s.cfg.VirtctlPath,
		IdentityFile:      identity,
		CommandTimeout:    cmdTimeout,
		Logf:              s.logf,
		HeartbeatInterval: 20 * time.Second,
	}

	s.logf("VM scanning setup: provision persistent VMs")
	s.provisionPersistentVMs()
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

	if s.cfg != nil && s.cfg.SkipCleanup {
		s.logf("teardown: VM_SCAN_SKIP_CLEANUP is set — skipping VM and namespace deletion (VMs and namespace left intact for debugging)")
		s.closeConn()
		return
	}

	deleteTimeout := s.teardownDeleteTimeout()
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

// ensureComplianceMetricsExposed guarantees that the collector compliance container
// serves Prometheus metrics on port 9091 and that a headless Service routes to it.
// When exposeMonitoring is already enabled in the deployment, both exist and this is
// a no-op. Otherwise, the method patches the collector DaemonSet to set
// ROX_METRICS_PORT=:9091 on the compliance container, creates the Service, and waits
// for the rollout to complete.
func (s *VMScanningSuite) ensureComplianceMetricsExposed() {
	ns := namespaces.StackRox
	const (
		svcName       = "compliance-metrics"
		dsName        = "collector"
		containerName = "compliance"
		metricsEnv    = "ROX_METRICS_PORT"
		metricsValue  = ":9091"
	)
	metricsPort := int32(9091)

	ctx, cancel := context.WithTimeout(s.ctx, 5*time.Minute)
	defer cancel()

	s.ensureComplianceMetricsEnv(ctx, ns, dsName, containerName, metricsEnv, metricsValue)
	s.ensureComplianceMetricsService(ctx, ns, svcName, metricsPort)
}

func (s *VMScanningSuite) ensureComplianceMetricsEnv(ctx context.Context, ns, dsName, containerName, envName, envValue string) {
	t := s.T()
	ds, err := s.k8sClient.AppsV1().DaemonSets(ns).Get(ctx, dsName, metaV1.GetOptions{})
	require.NoError(t, err, "getting DaemonSet %s/%s", ns, dsName)

	var container *coreV1.Container
	for i := range ds.Spec.Template.Spec.Containers {
		if ds.Spec.Template.Spec.Containers[i].Name == containerName {
			container = &ds.Spec.Template.Spec.Containers[i]
			break
		}
	}
	require.NotNil(t, container, "container %q not found in DaemonSet %s/%s", containerName, ns, dsName)

	for _, e := range container.Env {
		if e.Name == envName && e.Value == envValue {
			s.logf("VM scanning setup: DaemonSet %s/%s container %q already has %s=%s", ns, dsName, containerName, envName, envValue)
			return
		}
	}

	s.logf("VM scanning setup: patching DaemonSet %s/%s container %q: setting %s=%s", ns, dsName, containerName, envName, envValue)
	updated := false
	for i, e := range container.Env {
		if e.Name == envName {
			container.Env[i].Value = envValue
			container.Env[i].ValueFrom = nil
			updated = true
			break
		}
	}
	if !updated {
		container.Env = append(container.Env, coreV1.EnvVar{Name: envName, Value: envValue})
	}

	_, err = s.k8sClient.AppsV1().DaemonSets(ns).Update(ctx, ds, metaV1.UpdateOptions{})
	require.NoError(t, err, "updating DaemonSet %s/%s to set %s=%s", ns, dsName, envName, envValue)

	s.logf("VM scanning setup: waiting for DaemonSet %s/%s rollout", ns, dsName)
	err = wait.PollUntilContextCancel(ctx, 10*time.Second, false, func(pollCtx context.Context) (bool, error) {
		current, getErr := s.k8sClient.AppsV1().DaemonSets(ns).Get(pollCtx, dsName, metaV1.GetOptions{})
		if getErr != nil {
			s.logf("VM scanning setup: transient error checking DaemonSet rollout: %v", getErr)
			return false, nil
		}
		ready := current.Status.DesiredNumberScheduled > 0 &&
			current.Status.UpdatedNumberScheduled == current.Status.DesiredNumberScheduled &&
			current.Status.NumberReady == current.Status.DesiredNumberScheduled &&
			current.Status.ObservedGeneration >= current.Generation
		if !ready {
			s.logf("VM scanning setup: DaemonSet %s/%s rollout in progress (desired=%d updated=%d ready=%d)",
				ns, dsName, current.Status.DesiredNumberScheduled, current.Status.UpdatedNumberScheduled, current.Status.NumberReady)
		}
		return ready, nil
	})
	require.NoError(t, err, "waiting for DaemonSet %s/%s rollout after setting %s=%s", ns, dsName, envName, envValue)
	s.logf("VM scanning setup: DaemonSet %s/%s rollout complete", ns, dsName)
}

func (s *VMScanningSuite) ensureComplianceMetricsService(ctx context.Context, ns, svcName string, metricsPort int32) {
	t := s.T()
	desired := &coreV1.Service{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      svcName,
			Namespace: ns,
			Labels: map[string]string{
				"app.kubernetes.io/managed-by": "vm-scanning-e2e",
			},
		},
		Spec: coreV1.ServiceSpec{
			ClusterIP: "None",
			Selector:  map[string]string{"app": "collector"},
			Ports: []coreV1.ServicePort{{
				Name:       "monitoring",
				Port:       metricsPort,
				TargetPort: intstr.FromInt32(metricsPort),
				Protocol:   coreV1.ProtocolTCP,
			}},
		},
	}

	err := wait.PollUntilContextCancel(ctx, 5*time.Second, true, func(pollCtx context.Context) (bool, error) {
		existing, getErr := s.k8sClient.CoreV1().Services(ns).Get(pollCtx, svcName, metaV1.GetOptions{})
		if getErr == nil {
			if serviceExposesPort(existing, metricsPort) {
				s.logf("VM scanning setup: service %s/%s verified (port %d)", ns, svcName, metricsPort)
				return true, nil
			}
			s.logf("VM scanning setup: service %s/%s exists but missing port %d, deleting and re-creating", ns, svcName, metricsPort)
			_ = s.k8sClient.CoreV1().Services(ns).Delete(pollCtx, svcName, metaV1.DeleteOptions{})
			return false, nil
		}
		if !apierrors.IsNotFound(getErr) {
			s.logf("VM scanning setup: transient error checking service %s/%s: %v (retrying)", ns, svcName, getErr)
			return false, nil
		}

		s.logf("VM scanning setup: creating service %s/%s (compliance metrics port %d)", ns, svcName, metricsPort)
		_, createErr := s.k8sClient.CoreV1().Services(ns).Create(pollCtx, desired, metaV1.CreateOptions{})
		if createErr != nil {
			if apierrors.IsAlreadyExists(createErr) {
				return false, nil
			}
			s.logf("VM scanning setup: create service %s/%s failed: %v (retrying)", ns, svcName, createErr)
			return false, nil
		}
		return false, nil
	})
	require.NoError(t, err, "ensuring compliance-metrics service %s/%s with port %d", ns, svcName, metricsPort)
}

func serviceExposesPort(svc *coreV1.Service, port int32) bool {
	for _, p := range svc.Spec.Ports {
		if p.Port == port || p.TargetPort.IntValue() == int(port) {
			return true
		}
	}
	return false
}

func (s *VMScanningSuite) teardownDeleteTimeout() time.Duration {
	if s.cfg != nil && s.cfg.DeleteTimeout > 0 {
		return s.cfg.DeleteTimeout
	}
	return 5 * time.Minute
}

func (s *VMScanningSuite) vmProvisionTimeout() time.Duration {
	if s.cfg != nil && s.cfg.ScanTimeout > 0 {
		return s.cfg.ScanTimeout
	}
	return 20 * time.Minute
}

func (s *VMScanningSuite) virtctlForVM(vm VMHandle) vmhelpers.Virtctl {
	virt := s.virtctl
	if u := strings.TrimSpace(vm.GuestUser); u != "" {
		virt.Username = u
	}
	return virt
}

func (s *VMScanningSuite) guestStepTimeout() time.Duration {
	// Keep guest prep bounded even when suite-level timeout is large.
	const defaultTimeout = 20 * time.Minute
	if s.cfg == nil || s.cfg.ScanTimeout <= 0 {
		return defaultTimeout
	}
	if s.cfg.ScanTimeout < defaultTimeout {
		return s.cfg.ScanTimeout
	}
	return defaultTimeout
}

func (s *VMScanningSuite) guestBootGracePeriod() time.Duration {
	// New/recreated guests may report VMI Running before sshd/network are ready.
	const defaultGrace = 20 * time.Minute
	if s.cfg == nil || s.cfg.ScanTimeout <= 0 {
		return defaultGrace
	}
	if s.cfg.ScanTimeout < defaultGrace {
		return s.cfg.ScanTimeout
	}
	return defaultGrace
}

func stepElapsedSince(start time.Time) time.Duration {
	elapsed := time.Since(start).Round(time.Second)
	if elapsed <= 0 {
		return time.Second
	}
	return elapsed
}

func resolveSSHPublicKeyPathForLog(privateKeyPath, publicKeyRaw string) string {
	// Most local runs generate "<private>.pub"; prefer reporting that when present.
	if privateKeyPath != "" {
		candidate := privateKeyPath + ".pub"
		if fi, err := os.Stat(candidate); err == nil && !fi.IsDir() {
			return candidate
		}
	}

	// Fallback: the env value itself may already be a file path.
	publicKeyRaw = strings.TrimSpace(publicKeyRaw)
	if publicKeyRaw != "" {
		if fi, err := os.Stat(publicKeyRaw); err == nil && !fi.IsDir() {
			return publicKeyRaw
		}
	}
	return "<inline/unknown>"
}

func truncateGuestBootWarmupDetail(detail string) string {
	const maxLen = 240
	detail = strings.TrimSpace(detail)
	if detail == "" {
		return "<no stderr>"
	}
	if len(detail) <= maxLen {
		return detail
	}
	return detail[:maxLen] + fmt.Sprintf(" ... (truncated from %d bytes)", len(detail))
}

func (s *VMScanningSuite) waitForGuestBootWarmup(vm VMHandle) {
	const (
		pollInterval   = 5 * time.Second
		logEvery       = 5
		firstAttempt   = 1
		defaultDetails = "<no additional details>"
	)
	grace := s.guestBootGracePeriod()
	if grace <= 0 {
		return
	}

	start := time.Now()
	virt := s.virtctlForVM(vm)
	s.logf("[guest prep warm-up] START: %s/%s (up to %v before strict SSH checks)",
		vm.Namespace, vm.Name, grace)

	warmCtx, warmCancel := context.WithTimeout(s.ctx, grace)
	defer warmCancel()
	attempts := 0
	maxAttempts := int(grace/pollInterval) + 1
	err := wait.PollUntilContextCancel(warmCtx, pollInterval, true, func(ctx context.Context) (bool, error) {
		attempts++
		_, stderr, sshErr := virt.SSH(ctx, vm.Namespace, vm.Name, "true")
		if sshErr == nil {
			s.logf("[guest prep warm-up] DONE: %s/%s in %s (SSH reachable at attempt %d/%d)",
				vm.Namespace, vm.Name, stepElapsedSince(start), attempts, maxAttempts)
			return true, nil
		}
		if attempts == firstAttempt || attempts%logEvery == 0 {
			retriesLeft := maxAttempts - attempts
			if retriesLeft < 0 {
				retriesLeft = 0
			}
			detail := truncateGuestBootWarmupDetail(stderr)
			if detail == "<no stderr>" && sshErr != nil {
				detail = truncateGuestBootWarmupDetail(sshErr.Error())
			}
			if detail == "<no stderr>" {
				detail = defaultDetails
			}
			s.logf("Guest boot warm-up: %s/%s still booting (attempt %d/%d, retries left: %d): %s",
				vm.Namespace, vm.Name, attempts, maxAttempts, retriesLeft, detail)
		}
		return false, nil
	})
	if err != nil {
		s.logf("[guest prep warm-up] END: %s/%s after %s; proceeding to strict SSH checks",
			vm.Namespace, vm.Name, stepElapsedSince(start))
	}
}

func vmHandleKey(vm VMHandle) string {
	return vm.Namespace + "/" + vm.Name
}

func (s *VMScanningSuite) recordVMCreatedNow(vm VMHandle) {
	if s.vmCreatedAt == nil {
		s.vmCreatedAt = make(map[string]time.Time)
	}
	s.vmCreatedAt[vmHandleKey(vm)] = time.Now()
}

func (s *VMScanningSuite) recordVMCreationFromCluster(ctx context.Context, vm VMHandle) {
	if s.dynamicClient == nil {
		return
	}
	vmGVR := schema.GroupVersionResource{
		Group:    kubevirtv1.GroupVersion.Group,
		Version:  kubevirtv1.GroupVersion.Version,
		Resource: "virtualmachines",
	}
	obj, err := s.dynamicClient.Resource(vmGVR).Namespace(vm.Namespace).Get(ctx, vm.Name, metaV1.GetOptions{})
	if err != nil {
		s.logf("could not read creation timestamp for %s/%s; SSH broken-claim grace will use suite-local timestamps only: %v",
			vm.Namespace, vm.Name, err)
		return
	}
	createdAt := obj.GetCreationTimestamp().Time
	if createdAt.IsZero() {
		s.logf("creation timestamp missing for %s/%s; SSH broken-claim grace will use suite-local timestamps only",
			vm.Namespace, vm.Name)
		return
	}
	if s.vmCreatedAt == nil {
		s.vmCreatedAt = make(map[string]time.Time)
	}
	s.vmCreatedAt[vmHandleKey(vm)] = createdAt
}

func (s *VMScanningSuite) vmCreationAge(vm VMHandle) (time.Duration, bool) {
	if s.vmCreatedAt == nil {
		return 0, false
	}
	createdAt, ok := s.vmCreatedAt[vmHandleKey(vm)]
	if !ok {
		return 0, false
	}
	age := time.Since(createdAt)
	if age < 0 {
		return 0, true
	}
	return age, true
}

func (s *VMScanningSuite) maybeDelaySSHBrokenClaim(vm VMHandle, err error) (bool, error) {
	const progressLogEvery = 30 * time.Second
	recoverableSSHErr := errors.Is(err, vmhelpers.ErrSSHAuthenticationFailed) || errors.Is(err, vmhelpers.ErrSSHConnectivityStalled)
	if !recoverableSSHErr {
		return false, nil
	}
	age, known := s.vmCreationAge(vm)
	if !known {
		return false, nil
	}

	minAge := s.guestBootGracePeriod()
	if minAge <= 0 || age >= minAge {
		return false, nil
	}

	remaining := minAge - age
	s.logf("Delaying SSH-broken classification for %s/%s by %s (VM age %s < required %s); last error: %v",
		vm.Namespace, vm.Name, remaining.Round(time.Second), age.Round(time.Second), minAge, err)

	delayDeadline := time.Now().Add(remaining)
	timer := time.NewTimer(remaining)
	defer timer.Stop()
	progressTicker := time.NewTicker(progressLogEvery)
	defer progressTicker.Stop()
	for {
		select {
		case <-timer.C:
			s.logf("SSH-broken classification delay elapsed for %s/%s; continuing with strict checks", vm.Namespace, vm.Name)
			return true, nil
		case <-progressTicker.C:
			left := time.Until(delayDeadline)
			if left < 0 {
				left = 0
			}
			s.logf("SSH-broken classification delay still active for %s/%s (%s remaining); last error: %v",
				vm.Namespace, vm.Name, left.Round(time.Second), err)
		case <-s.ctx.Done():
			return false, fmt.Errorf("wait SSH broken-claim delay for %s/%s: %w", vm.Namespace, vm.Name, s.ctx.Err())
		}
	}
}

func (s *VMScanningSuite) prepareGuestRespectingSSHBrokenGrace(vm VMHandle) error {
	for {
		err := s.prepareGuest(vm)
		if err == nil {
			return nil
		}
		delayed, delayErr := s.maybeDelaySSHBrokenClaim(vm, err)
		if delayErr != nil {
			return delayErr
		}
		if !delayed {
			return err
		}
		s.logf("Retrying guest preparation for %s/%s after SSH broken-claim delay", vm.Namespace, vm.Name)
	}
}

func waitForNamespaceDeleted(ctx context.Context, k8s kubernetes.Interface, name string) error {
	return wait.PollUntilContextCancel(ctx, 2*time.Second, true, func(ctx context.Context) (bool, error) {
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
	t := s.T()
	ctx, cancel := context.WithTimeout(s.ctx, 2*time.Minute)
	defer cancel()

	flagsSvc := v1.NewFeatureFlagServiceClient(s.conn)
	resp, err := flagsSvc.GetFeatureFlags(ctx, &v1.Empty{})
	require.NoError(t, err, "GetFeatureFlags")

	wantEnv := vmScanVirtualMachinesFeatureEnvVar()
	for _, f := range resp.GetFeatureFlags() {
		if f.GetEnvVar() == wantEnv && f.GetEnabled() {
			return
		}
	}
	require.Failf(t, "Virtual Machines product feature is not enabled on Central",
		"expected feature flag %q to be enabled; snapshot: %s", wantEnv, formatFeatureFlagsForDiag(resp))
}

func (s *VMScanningSuite) provisionPersistentVMs() {
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

	specs := []struct {
		name      string
		image     string
		guestUser string
	}{
		{name: "vm-rhel9", image: s.cfg.ImageRHEL9, guestUser: s.cfg.GuestUserRHEL9},
		{name: "vm-rhel10", image: s.cfg.ImageRHEL10, guestUser: s.cfg.GuestUserRHEL10},
	}
	for _, sp := range specs {
		req := vmhelpers.VMRequest{
			Name:         sp.name,
			Namespace:    s.namespace,
			Image:        sp.image,
			GuestUser:    sp.guestUser,
			SSHPublicKey: s.cfg.SSHPublicKey,
		}
		s.logf("provision persistent VMs: ensuring VM exists %s/%s with image %q", s.namespace, sp.name, sp.image)
		createErr := vmhelpers.CreateVirtualMachine(ctx, s.dynamicClient, req)
		if createErr == nil {
			s.logf("provision persistent VMs: created VM %s/%s", s.namespace, sp.name)
			createdNow[sp.name] = true
		} else if apierrors.IsAlreadyExists(createErr) {
			s.logf("provision persistent VMs: VM %s/%s already exists; reusing it", s.namespace, sp.name)
			createdNow[sp.name] = false
		} else {
			require.NoError(s.T(), createErr, "EnsureVirtualMachineExists %s/%s", s.namespace, sp.name)
		}
		h := VMHandle{Name: sp.name, Namespace: s.namespace, GuestUser: sp.guestUser}
		if createdNow[sp.name] {
			s.recordVMCreatedNow(h)
		} else {
			s.recordVMCreationFromCluster(ctx, h)
		}
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

		if createdNow[vm.Name] {
			s.waitForGuestBootWarmup(*vm)
		}
	}

	s.logVMPlacement(ctx)
}

func (s *VMScanningSuite) logVMPlacement(ctx context.Context) {
	lookupCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	nodeToCollector := make(map[string]string)
	pods, err := s.k8sClient.CoreV1().Pods(namespaces.StackRox).List(lookupCtx, metaV1.ListOptions{
		LabelSelector: "app=collector",
	})
	if err != nil {
		s.logf("VM placement: could not list collector pods: %v", err)
	} else {
		for _, p := range pods.Items {
			nodeToCollector[p.Spec.NodeName] = p.Name
		}
	}

	for _, vm := range s.persistentVMs {
		node := vm.NodeName
		if node == "" {
			node = "<unknown>"
		}
		collector := nodeToCollector[vm.NodeName]
		if collector == "" {
			collector = "<none>"
		}
		s.logf("- VM: %s, Node: %s, Collector pod: %s", vm.Name, node, collector)
	}
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
		err := s.prepareGuestRespectingSSHBrokenGrace(*vm)
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
	s.recordVMCreatedNow(*vm)
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

	s.waitForGuestBootWarmup(*vm)
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
	guestUser := strings.TrimSpace(vm.GuestUser)
	if guestUser == "" {
		switch vm.Name {
		case "vm-rhel9":
			guestUser = s.cfg.GuestUserRHEL9
		case "vm-rhel10":
			guestUser = s.cfg.GuestUserRHEL10
		}
	}

	var image string
	switch vm.Name {
	case "vm-rhel9":
		image = s.cfg.ImageRHEL9
	case "vm-rhel10":
		image = s.cfg.ImageRHEL10
	default:
		return vmhelpers.VMRequest{}, fmt.Errorf("unsupported persistent VM %s/%s for recreation", vm.Namespace, vm.Name)
	}
	if image == "" {
		return vmhelpers.VMRequest{}, fmt.Errorf("missing image for persistent VM %s/%s recreation", vm.Namespace, vm.Name)
	}
	if guestUser == "" {
		return vmhelpers.VMRequest{}, fmt.Errorf("missing guest user for persistent VM %s/%s recreation", vm.Namespace, vm.Name)
	}

	return vmhelpers.VMRequest{
		Name:         vm.Name,
		Namespace:    vm.Namespace,
		Image:        image,
		GuestUser:    guestUser,
		SSHPublicKey: s.cfg.SSHPublicKey,
	}, nil
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
	defer f.Close()
	if _, err := f.WriteString(stdout); err != nil {
		s.logf("ensureCanonicalScan: could not write roxagent stdout file %q: %v", f.Name(), err)
		return ""
	}
	return f.Name()
}

// ensureCanonicalScan runs a single guest-side roxagent invocation and validates failure signals.
func (s *VMScanningSuite) ensureCanonicalScan(ctx context.Context, vm *VMHandle) (*vmhelpers.RoxagentRunResult, error) {
	if vm == nil {
		return nil, errors.New("ensureCanonicalScan: nil VM handle")
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
	present, err := vmhelpers.WaitForVMPresentInCentralWithOptions(waitCtx, s.vmClient, baseOpts, vm.Namespace, vm.Name)
	if err != nil {
		return nil, err
	}
	vm.ID = present.GetId()
	s.logf("scan wait %s/%s step 1/6 complete: id=%q", vm.Namespace, vm.Name, vm.ID)

	s.logf("scan wait %s/%s step 2/6: wait VM identity fields", vm.Namespace, vm.Name)
	if _, err := vmhelpers.WaitForVMIdentityFieldsWithOptions(waitCtx, s.vmClient, baseOpts, present.GetId(), vm.Namespace, vm.Name); err != nil {
		return nil, err
	}
	s.logf("scan wait %s/%s step 2/6 complete", vm.Namespace, vm.Name)
	s.logf("scan wait %s/%s step 3/6: wait VM running in Central", vm.Namespace, vm.Name)
	if _, err := vmhelpers.WaitForVMRunningInCentralWithOptions(waitCtx, s.vmClient, baseOpts, present.GetId()); err != nil {
		return nil, err
	}
	s.logf("scan wait %s/%s step 3/6 complete", vm.Namespace, vm.Name)
	s.logf("scan wait %s/%s step 4/6: wait non-nil scan", vm.Namespace, vm.Name)
	if _, err := vmhelpers.WaitForVMScanNonNilWithOptions(waitCtx, s.vmClient, baseOpts, present.GetId()); err != nil {
		return nil, err
	}
	s.logf("scan wait %s/%s step 4/6 complete", vm.Namespace, vm.Name)
	s.logf("scan wait %s/%s step 5/6: wait scan timestamp", vm.Namespace, vm.Name)
	if _, err := vmhelpers.WaitForVMScanTimestampWithOptions(waitCtx, s.vmClient, baseOpts, present.GetId()); err != nil {
		return nil, err
	}
	s.logf("scan wait %s/%s step 5/6 complete", vm.Namespace, vm.Name)

	conds := vmhelpers.ScanReadiness{Components: true}
	if s.cfg.RequireActivation {
		conds.AllScanned = true
	}
	s.logf("scan wait %s/%s step 6/6: wait scan ready (components=%v all-scanned=%v)",
		vm.Namespace, vm.Name, conds.Components, conds.AllScanned)
	return vmhelpers.WaitForScanReadyWithOptions(waitCtx, s.vmClient, baseOpts, present.GetId(), conds)
}

// complianceTarget returns the ScrapeTarget for the compliance container on the VM's node.
func (s *VMScanningSuite) complianceTarget(vmNodeName string) testmetrics.ScrapeTarget {
	t := testmetrics.ScrapeTarget{
		ComponentName: "compliance",
		Namespace:     namespaces.StackRox,
		LabelSelector: "app=collector",
		MetricsPort:   9091,
		MetricsPath:   "metrics",
	}
	if vmNodeName != "" {
		t.FieldSelector = "spec.nodeName=" + vmNodeName
	}
	return t
}

// sensorTarget returns the ScrapeTarget for the sensor deployment.
func (s *VMScanningSuite) sensorTarget() testmetrics.ScrapeTarget {
	return testmetrics.ScrapeTarget{
		ComponentName: "sensor",
		Namespace:     namespaces.StackRox,
		LabelSelector: "app=sensor",
		MetricsPort:   9090,
		MetricsPath:   "metrics",
	}
}

// complianceQueries returns the Query set for the compliance relay.
func complianceQueries() []testmetrics.Query {
	return []testmetrics.Query{
		{Name: vmhelpers.MetricComplianceRelayConnectionsAcceptedTotal},
		{Name: vmhelpers.MetricComplianceRelayIndexReportsReceivedTotal},
		{Name: vmhelpers.MetricComplianceRelayIndexReportsSentTotal, LabelFilter: `failed="false"`},
		{Name: vmhelpers.MetricComplianceRelayIndexReportsSentTotal, LabelFilter: `failed="true"`},
		{Name: vmhelpers.MetricComplianceRelayIndexReportsMismatchingVsockTotal},
		{Name: vmhelpers.MetricComplianceRelayIndexReportAcksReceivedTotal},
	}
}

// sensorQueries returns the Query set for the sensor VM index pipeline.
func sensorQueries() []testmetrics.Query {
	return []testmetrics.Query{
		{Name: vmhelpers.MetricSensorVMIndexReportsReceivedTotal},
		{Name: vmhelpers.MetricSensorVMIndexReportsSentTotal, LabelFilter: `status="` + vmhelpers.SensorIndexReportStatusSuccess + `"`},
		{Name: vmhelpers.MetricSensorVMIndexReportsSentTotal, LabelFilter: `status="` + vmhelpers.SensorIndexReportStatusError + `"`},
		{Name: vmhelpers.MetricSensorVMIndexReportsSentTotal, LabelFilter: `status="` + vmhelpers.SensorIndexReportStatusCentralNotReady + `"`},
		{Name: vmhelpers.MetricSensorVMIndexReportAcksReceivedTotal, LabelFilter: `action="ACK"`},
		{Name: vmhelpers.MetricSensorVMIndexReportEnqueueBlockedTotal},
	}
}

// collectStableMetrics scrapes compliance and sensor metrics until values stabilize.
// It returns two maps keyed by testmetrics.Key.
func (s *VMScanningSuite) collectStableMetrics(ctx context.Context, vmNodeName string) (compliance, sensor map[string]testmetrics.Value) {
	const (
		metricsTimeout  = 2 * time.Minute
		metricsPollWait = 10 * time.Second
		stableRounds    = 3
	)

	compTarget := s.complianceTarget(vmNodeName)
	senTarget := s.sensorTarget()
	compQ := complianceQueries()
	senQ := sensorQueries()
	transport := testmetrics.TransportProxy

	stableCfg := testmetrics.StableConfig{
		PollInterval: metricsPollWait,
		StableRounds: stableRounds,
		Logf:         s.logf,
	}

	compCtx, compCancel := context.WithTimeout(ctx, metricsTimeout)
	defer compCancel()
	compliance = testmetrics.PollUntilStable(compCtx, stableCfg, func(ctx context.Context) (map[string]testmetrics.Value, error) {
		return testmetrics.ScrapeComponent(ctx, s.k8sClient, compTarget, transport, compQ)
	})

	senCtx, senCancel := context.WithTimeout(ctx, metricsTimeout)
	defer senCancel()
	sensor = testmetrics.PollUntilStable(senCtx, stableCfg, func(ctx context.Context) (map[string]testmetrics.Value, error) {
		return testmetrics.ScrapeComponent(ctx, s.k8sClient, senTarget, transport, senQ)
	})

	return compliance, sensor
}

// assertPipelineMetrics collects stable metrics and asserts their values.
// vmNodeName must be non-empty so compliance metrics are scoped to the VM's local collector pod.
func (s *VMScanningSuite) assertPipelineMetrics(ctx context.Context, t require.TestingT, vmNodeName string) {
	require.NotEmpty(t, vmNodeName,
		"VM node name must be known before asserting pipeline metrics; "+
			"cluster-wide collector scraping is not supported because it conflates metrics from unrelated VMs")

	compTarget := s.complianceTarget(vmNodeName)
	s.logf("pipeline metrics: VM node=%q, compliance selector=%q field=%q",
		vmNodeName, compTarget.LabelSelector, compTarget.FieldSelector)

	err := testmetrics.FindServicePort(ctx, s.k8sClient, compTarget.Namespace, "app", "collector", compTarget.MetricsPort)
	require.NoError(t, err,
		"collector Service should expose compliance metrics port %d; the deployment may be missing the metrics port definition",
		compTarget.MetricsPort)

	comp, sen := s.collectStableMetrics(ctx, vmNodeName)

	get := func(src map[string]testmetrics.Value, q testmetrics.Query) testmetrics.Value {
		return src[testmetrics.Key(q)]
	}

	requirePositive := func(src map[string]testmetrics.Value, q testmetrics.Query, label string) {
		v := get(src, q)
		require.Truef(t, v.Found, "%s should be present in scraped metrics, but was not found", label)
		require.Greaterf(t, v.Val, float64(0), "%s should be > 0, but got %.0f", label, v.Val)
	}

	requireZero := func(src map[string]testmetrics.Value, q testmetrics.Query, label string) {
		v := get(src, q)
		if !v.Found {
			return
		}
		require.Equalf(t, float64(0), v.Val, "%s should be 0, but got %.0f", label, v.Val)
	}

	// Compliance relay assertions.
	cq := complianceQueries()
	requirePositive(comp, cq[0], "compliance relay connections_accepted")
	requirePositive(comp, cq[1], "compliance relay index_reports_received")
	requirePositive(comp, cq[2], "compliance relay index_reports_sent (failed=false)")
	requireZero(comp, cq[3], "compliance relay index_reports_sent (failed=true)")
	requireZero(comp, cq[4], "compliance relay vsock CID mismatches")
	requirePositive(comp, cq[5],
		"compliance relay acks_received should be > 0 when Central ACKs an index report; "+
			"known gap: Sensor does not propagate ACKs back to the compliance relay yet")

	// Sensor assertions.
	sq := sensorQueries()
	requirePositive(sen, sq[0], "sensor index_reports_received")
	requirePositive(sen, sq[1], "sensor index_reports_sent (success)")
	requireZero(sen, sq[2], "sensor index_reports_sent (error)")
	requireZero(sen, sq[3], "sensor index_reports_sent (central not ready)")
	requirePositive(sen, sq[4], "sensor index_report_acks_received (ACK)")
	requireZero(sen, sq[5], "sensor enqueue_blocked")
}

func (s *VMScanningSuite) vmDeleteTimeout() time.Duration {
	if s.cfg != nil && s.cfg.DeleteTimeout > 0 {
		return s.cfg.DeleteTimeout
	}
	return 5 * time.Minute
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
	stepTimeout := s.guestStepTimeout()
	stepNum := 0
	withStepTimeout := func(fn func(stepCtx context.Context) error) error {
		stepCtx, cancel := context.WithTimeout(s.ctx, stepTimeout)
		defer cancel()
		return fn(stepCtx)
	}
	runStep := func(stepName, errContext string, fn func(stepCtx context.Context) error) error {
		stepNum++
		s.logf("[guest preparation step %02d]: %s on %s/%s (timeout=%v)",
			stepNum, stepName, vm.Namespace, vm.Name, stepTimeout)
		if err := withStepTimeout(fn); err != nil {
			return fmt.Errorf("prepare guest %s/%s: %s: %w", vm.Namespace, vm.Name, errContext, err)
		}
		return nil
	}

	if err := runStep("Wait for SSH to become reachable", "WaitForSSHReachable", func(stepCtx context.Context) error {
		return vmhelpers.WaitForSSHReachable(s.T(), stepCtx, virt, vm.Namespace, vm.Name)
	}); err != nil {
		return err
	}
	if err := runStep("Wait for cloud-init to finish", "WaitForCloudInitFinished", func(stepCtx context.Context) error {
		return vmhelpers.WaitForCloudInitFinished(stepCtx, virt, vm.Namespace, vm.Name)
	}); err != nil {
		return err
	}
	if err := runStep("Verify sudo", "VerifySudoWorks", func(stepCtx context.Context) error {
		return vmhelpers.VerifySudoWorks(stepCtx, virt, vm.Namespace, vm.Name)
	}); err != nil {
		return err
	}
	if err := runStep("Copy roxagent binary", "CopyRoxagentBinary", func(stepCtx context.Context) error {
		return vmhelpers.CopyRoxagentBinary(stepCtx, virt, vm.Namespace, vm.Name, s.cfg.RoxagentBinaryPath)
	}); err != nil {
		return err
	}
	if err := runStep("Verify roxagent binary presence", "VerifyRoxagentBinaryPresent", func(stepCtx context.Context) error {
		return vmhelpers.VerifyRoxagentBinaryPresent(stepCtx, virt, vm.Namespace, vm.Name)
	}); err != nil {
		return err
	}
	if err := runStep("Verify roxagent executable mode", "VerifyRoxagentExecutable", func(stepCtx context.Context) error {
		return vmhelpers.VerifyRoxagentExecutable(stepCtx, virt, vm.Namespace, vm.Name)
	}); err != nil {
		return err
	}
	if err := runStep("Verify roxagent install path", "VerifyRoxagentInstallPath", func(stepCtx context.Context) error {
		return vmhelpers.VerifyRoxagentInstallPath(stepCtx, virt, vm.Namespace, vm.Name)
	}); err != nil {
		return err
	}
	var (
		activated bool
	)
	if err := runStep("Check activation status", "GetActivationStatus", func(stepCtx context.Context) error {
		var innerErr error
		activated, _, innerErr = vmhelpers.GetActivationStatus(stepCtx, virt, vm.Namespace, vm.Name)
		return innerErr
	}); err != nil {
		return err
	}
	if !activated && s.cfg.ActivationOrg != "" && s.cfg.ActivationKey != "" {
		if err := runStep("Activate system via rhc connect", "ActivateWithRHC", func(stepCtx context.Context) error {
			return vmhelpers.ActivateWithRHC(stepCtx, virt, vm.Namespace, vm.Name,
				s.cfg.ActivationOrg, s.cfg.ActivationKey, s.cfg.ActivationEndpoint)
		}); err != nil {
			return err
		}
		activated = true
	}
	if s.cfg.RequireActivation && !activated {
		return fmt.Errorf("prepare guest %s/%s: VM activation required but guest is not activated", vm.Namespace, vm.Name)
	}
	if activated {
		if err := runStep("Verify activation success", "VerifyActivationSucceeded", func(stepCtx context.Context) error {
			return vmhelpers.VerifyActivationSucceeded(stepCtx, virt, vm.Namespace, vm.Name)
		}); err != nil {
			return err
		}
	}
	if err := runStep("Populate dnf history", "PopulateDnfHistoryWithRandomPackage", func(stepCtx context.Context) error {
		return vmhelpers.PopulateDnfHistoryWithRandomPackage(stepCtx, virt, vm.Namespace, vm.Name)
	}); err != nil {
		return err
	}
	s.logf("[guest prep] COMPLETED for %s/%s in %d step(s)", vm.Namespace, vm.Name, stepNum)
	return nil
}
