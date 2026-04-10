# VM Scanning E2E Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a Go-based OpenShift VM-scanning E2E lane that provisions RHEL 9 and RHEL 10 KubeVirt VMs, installs and runs `roxagent`, validates the report path through Compliance, Sensor, and Central, and wires the suite into a dedicated OCP CI job.

**Architecture:** Add a focused `tests/vmhelpers` helper package for VM lifecycle, guest operations, Central polling, and escalation diagnostics. Keep product assertions in `tests/vm_scanning_*_test.go`, and wire the suite into CI with a dedicated `tests/e2e/run-vm-scanning.sh` script plus `.openshift-ci` / `scripts/ci/jobs` runner glue.

**Tech Stack:** Go 1.25, `testify/suite`, `k8s.io/client-go`, `kubevirt.io/api`, generated `api/v2` gRPC clients, `virtctl`, Bash, OpenShift CI Python runners.

---

## File Structure

### New files

- `tests/vm_scanning_suite_test.go`
  - `//go:build test_e2e`
  - top-level `TestVMScanning`
  - suite lifecycle, config loading, shared VM handles, failure artifact collection
- `tests/vm_scanning_test.go`
  - `//go:build test_e2e`
  - `TestHappyPath`, `TestRescan`, `TestVMLifecycle`
- `tests/vm_scanning_utils_test.go`
  - `//go:build test_e2e`
  - suite-local assertions, env/config validation tests, artifact formatting helpers
- `tests/vmhelpers/vm.go`
  - VM manifest rendering, create/delete calls, VMI readiness
- `tests/vmhelpers/vm_test.go`
  - cloud-init injection, VM manifest rendering, VMI readiness predicate tests
- `tests/vmhelpers/virtctl.go`
  - `virtctl ssh` / `virtctl scp` command wrappers with timeout and stderr capture
- `tests/vmhelpers/virtctl_test.go`
  - command-line rendering and argument validation tests
- `tests/vmhelpers/guest.go`
  - SSH reachability, cloud-init completion, activation checks, `sudo` checks, dnf-history priming
- `tests/vmhelpers/guest_test.go`
  - guest command parsing and activation-state tests
- `tests/vmhelpers/roxagent.go`
  - copy binary, run `roxagent --verbose`, repo-to-CPE primary/fallback strategy, verbose-output assertions
- `tests/vmhelpers/roxagent_test.go`
  - fallback decision logic and verbose-output predicate tests
- `tests/vmhelpers/central.go`
  - `VirtualMachineService` polling helpers and scan/component-state predicates
- `tests/vmhelpers/central_test.go`
  - `UNSCANNED` note handling, `N > 0 components` and `all components scanned` predicate tests
- `tests/vmhelpers/observability.go`
  - on-demand pod log collection, metrics scraping via port-forward, Central/Sensor/Compliance escalation summary
- `tests/vmhelpers/observability_test.go`
  - metric-name / log-marker parsing and rate-limit detection tests
- `tests/vmhelpers/wait.go`
  - generic single-condition polling helpers with structured errors
- `tests/testdata/vm-scanning/cloud-init.yaml.tmpl`
  - cloud-init user data for SSH key injection
- `tests/e2e/vm-scanning-lib.sh`
  - OCP cluster preflight and VM-scanning-specific shell helpers
- `tests/e2e/run-vm-scanning.sh`
  - dedicated E2E entrypoint for the new job
- `scripts/ci/jobs/ocp_vm_scanning_e2e_tests.py`
  - dedicated OCP VM-scanning CI runner

### Modified files

- `tests/Makefile`
  - add `vm-scanning-tests` target with JUnit output
- `.openshift-ci/ci_tests.py`
  - add a `VMScanningE2e` test class that runs `tests/e2e/run-vm-scanning.sh`

### Existing files to follow

- `tests/common.go`
  - shared kubeconfig loading, retryable k8s client setup, timestamped logging, log collection
- `tests/delegated_scanning_test.go`
  - suite structure, artifact capture, context/timeouts, Central/Sensor health checks
- `tests/delegated_scanning_test_utils.go`
  - OCP-specific helper style
- `pkg/testutils/centralgrpc/connect_to_central.go`
  - Central connection pattern
- `tests/e2e/lib.sh`
  - deployment orchestration and result collection
- `scripts/ci/jobs/ocp_nongroovy_e2e_tests.py`
  - current OCP non-groovy job style

## Task 1: Create the VM-scanning suite skeleton

**Files:**
- Create: `tests/vm_scanning_suite_test.go`
- Create: `tests/vm_scanning_test.go`
- Create: `tests/vm_scanning_utils_test.go`
- Modify: `tests/Makefile`

- [ ] **Step 1: Write the failing config tests**

```go
func TestLoadVMScanConfig_MissingRequired(t *testing.T) {
    t.Setenv("VM_IMAGE_RHEL9", "")
    _, err := loadVMScanConfig()
    require.ErrorContains(t, err, "VM_IMAGE_RHEL9")
}

func TestLoadVMScanConfig_RequiresActivationInputsWhenEnabled(t *testing.T) {
    t.Setenv("VM_SCAN_REQUIRE_ACTIVATION", "true")
    t.Setenv("RHEL_ACTIVATION_ORG", "")
    _, err := loadVMScanConfig()
    require.ErrorContains(t, err, "RHEL_ACTIVATION_ORG")
}
```

- [ ] **Step 2: Run the tests to verify they fail**

Run: `go test -tags test_e2e ./tests -run 'TestLoadVMScanConfig' -v`

Expected: FAIL with missing `loadVMScanConfig`.

- [ ] **Step 3: Add the build-tagged suite files, config loader, and focused Make target**

```go
//go:build test_e2e

type vmScanConfig struct {
    ImageRHEL9              string
    ImageRHEL10             string
    GuestUserRHEL9          string
    GuestUserRHEL10         string
    VirtctlPath             string
    RoxagentBinaryPath      string
    Repo2CPEPrimaryURL      string // from ROXAGENT_REPO2CPE_PRIMARY_URL
    Repo2CPEFallbackURL     string // from ROXAGENT_REPO2CPE_FALLBACK_URL
    Repo2CPEPrimaryAttempts int    // from ROXAGENT_REPO2CPE_PRIMARY_ATTEMPTS
    SSHPrivateKey           string
    SSHPublicKey            string
    NamespacePrefix         string
    ScanTimeout             time.Duration
    ScanPollInterval        time.Duration
    ScanEscalationAttempt   int
    DeleteTimeout           time.Duration
    RequireActivation       bool
    ActivationOrg           string
    ActivationKey           string
    ActivationEndpoint      string
}
```

```makefile
.PHONY: vm-scanning-tests
vm-scanning-tests:
	@echo "+ $@"
	@GOTAGS=$(GOTAGS),test,test_e2e $(TOPLEVEL)/scripts/go-test.sh -cover $(TESTFLAGS) -v -run 'TestVMScanning|TestLoadVMScanConfig' $(shell go list -e ./... | grep -v generated | grep -v vendor) 2>&1 | tee test.log
	@$(MAKE) report JUNIT_OUT=vm-scanning-tests-results
```

- [ ] **Step 4: Run the focused tests again**

Run: `go test -tags test_e2e ./tests -run 'TestLoadVMScanConfig' -v`

Expected: PASS

- [ ] **Step 5: Verify the Make target layout**

Run: `make -C tests TESTFLAGS='-race -p 1 -timeout 90m' vm-scanning-tests`

Expected: writes `tests/test.log` and `tests/vm-scanning-tests-results/report.xml`; test bodies may still fail.

- [ ] **Step 6: Commit**

```bash
git add tests/Makefile tests/vm_scanning_suite_test.go tests/vm_scanning_test.go tests/vm_scanning_utils_test.go
git commit -m "test(vm-scanning): add suite skeleton and config contract"
```

## Task 2: Build VM manifest and `virtctl` helpers

**Files:**
- Create: `tests/testdata/vm-scanning/cloud-init.yaml.tmpl`
- Create: `tests/vmhelpers/vm.go`
- Create: `tests/vmhelpers/vm_test.go`
- Create: `tests/vmhelpers/virtctl.go`
- Create: `tests/vmhelpers/virtctl_test.go`

- [ ] **Step 1: Write failing helper tests for cloud-init and `virtctl`**

```go
func TestRenderCloudInit_InjectsAuthorizedKey(t *testing.T) {
    data, err := RenderCloudInit(VMRequest{
        GuestUser:    "cloud-user",
        SSHPublicKey: "ssh-rsa AAAATEST",
    })
    require.NoError(t, err)
    require.Contains(t, data, "ssh_authorized_keys:")
    require.Contains(t, data, "ssh-rsa AAAATEST")
}

func TestSSHCommandArgs_UsesIdentityAndNamespace(t *testing.T) {
    args := buildVirtctlSSHArgs("/usr/bin/virtctl", "stackrox", "vm-rhel9", "/tmp/id_rsa", "sudo", "true")
    require.Equal(t, []string{
        "/usr/bin/virtctl", "ssh", "--namespace", "stackrox", "--identity-file", "/tmp/id_rsa", "vm-rhel9", "--", "sudo", "true",
    }, args)
}
```

- [ ] **Step 2: Run the helper tests to verify they fail**

Run: `go test ./tests/vmhelpers -run 'TestRenderCloudInit|TestSSHCommandArgs' -v`

Expected: FAIL with missing helper symbols.

- [ ] **Step 3: Implement VM manifest rendering and `virtctl` wrappers**

```go
type VMRequest struct {
    Name         string
    Namespace    string
    Image        string
    GuestUser    string
    SSHPublicKey string
}

func RenderCloudInit(req VMRequest) ([]byte, error)
func CreateVirtualMachine(ctx context.Context, client dynamic.Interface, req VMRequest) error
func DeleteVirtualMachine(ctx context.Context, client dynamic.Interface, namespace, name string) error
func WaitForVirtualMachineInstanceExists(ctx context.Context, client dynamic.Interface, namespace, name string) error
func WaitForVirtualMachineInstanceRunning(ctx context.Context, client dynamic.Interface, namespace, name string) error
func WaitForVirtualMachineDeleted(ctx context.Context, client dynamic.Interface, namespace, name string) error
func buildVirtctlSSHArgs(virtctlPath, namespace, vm, identityFile string, command ...string) []string

type Virtctl struct {
    Path           string
    IdentityFile   string
    CommandTimeout time.Duration
}

func (v Virtctl) SSH(ctx context.Context, namespace, vm string, command ...string) (stdout string, stderr string, err error)
func (v Virtctl) SCPTo(ctx context.Context, namespace, vm, src, dst string) (stderr string, err error)
```

- [ ] **Step 4: Run the helper tests again**

Run: `go test ./tests/vmhelpers -run 'TestRenderCloudInit|TestSSHCommandArgs' -v`

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add tests/testdata/vm-scanning/cloud-init.yaml.tmpl tests/vmhelpers/vm.go tests/vmhelpers/vm_test.go tests/vmhelpers/virtctl.go tests/vmhelpers/virtctl_test.go
git commit -m "test(vm-scanning): add VM manifest and virtctl helpers"
```

## Task 3: Add guest preparation and `roxagent` execution helpers

**Files:**
- Create: `tests/vmhelpers/guest.go`
- Create: `tests/vmhelpers/guest_test.go`
- Create: `tests/vmhelpers/roxagent.go`
- Create: `tests/vmhelpers/roxagent_test.go`

- [ ] **Step 1: Write failing tests for activation, dnf priming, and repo-to-CPE fallback**

```go
func TestChooseRepo2CPESource_FallsBackAfterPrimaryAttempts(t *testing.T) {
    src := chooseRepo2CPESource(3, 3, "https://remote/repo2cpe.json", "file:///var/lib/rox/repo2cpe.json")
    require.Equal(t, "file:///var/lib/rox/repo2cpe.json", src)
}

func TestVerboseOutputLooksLikeReport_FalseForQuietOutput(t *testing.T) {
    require.False(t, VerboseOutputLooksLikeReport("done"))
}

func TestRoxagentInstallHelpers_BinaryPresentAndExecutableChecks(t *testing.T) {
    require.Equal(t,
        "test -x /usr/local/bin/roxagent",
        buildExecutableCheckCommand("/usr/local/bin/roxagent"),
    )
    require.Equal(t,
        "test -f /usr/local/bin/roxagent",
        buildPresenceCheckCommand("/usr/local/bin/roxagent"),
    )
}
```

- [ ] **Step 2: Run the helper tests to verify they fail**

Run: `go test ./tests/vmhelpers -run 'TestChooseRepo2CPESource|TestVerboseOutputLooksLikeReport' -v`

Expected: FAIL with missing helper symbols.

- [ ] **Step 3: Implement guest-step helpers and `roxagent --verbose` execution**

```go
func WaitForSSHReachable(ctx context.Context, virt Virtctl, namespace, vm string) error
func WaitForCloudInitFinished(ctx context.Context, virt Virtctl, namespace, vm string) error
func VerifySudoWorks(ctx context.Context, virt Virtctl, namespace, vm string) error
func CopyRoxagentBinary(ctx context.Context, virt Virtctl, namespace, vm, hostBinaryPath string) error
func VerifyRoxagentBinaryPresent(ctx context.Context, virt Virtctl, namespace, vm string) error
func VerifyRoxagentExecutable(ctx context.Context, virt Virtctl, namespace, vm string) error
func VerifyRoxagentInstallPath(ctx context.Context, virt Virtctl, namespace, vm string) error
func GetActivationStatus(ctx context.Context, virt Virtctl, namespace, vm string) (activated bool, details string, err error)
func ActivateWithRHC(ctx context.Context, virt Virtctl, namespace, vm, org, activationKey, activationEndpoint string) error
func VerifyActivationSucceeded(ctx context.Context, virt Virtctl, namespace, vm string) error
func PopulateDnfHistoryWithRandomPackage(ctx context.Context, virt Virtctl, namespace, vm string) error

type RoxagentRunResult struct {
    Stdout       string
    Stderr       string
    UsedFallback bool
}

func RunRoxagentOnce(ctx context.Context, virt Virtctl, namespace, vm string, cfg RoxagentRunConfig) (*RoxagentRunResult, error)
```

- [ ] **Step 4: Run the helper tests again**

Run: `go test ./tests/vmhelpers -run 'TestChooseRepo2CPESource|TestVerboseOutputLooksLikeReport' -v`

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add tests/vmhelpers/guest.go tests/vmhelpers/guest_test.go tests/vmhelpers/roxagent.go tests/vmhelpers/roxagent_test.go
git commit -m "test(vm-scanning): add guest prep and roxagent helpers"
```

## Task 4: Add Central polling and escalation diagnostics helpers

**Files:**
- Create: `tests/vmhelpers/central.go`
- Create: `tests/vmhelpers/central_test.go`
- Create: `tests/vmhelpers/observability.go`
- Create: `tests/vmhelpers/observability_test.go`
- Create: `tests/vmhelpers/wait.go`

- [ ] **Step 1: Write failing tests for component-state predicates and rate-limit detection**

```go
func TestHasReportedComponents_TrueForNComponents(t *testing.T) {
    vm := &v2.VirtualMachine{
        Scan: &v2.VirtualMachineScan{
            Components: []*v2.ScanComponent{{Name: "pkg-a"}},
        },
    }
    require.True(t, hasReportedComponents(vm))
}

func TestDetectRateLimitedNACK_TrueForCentralReason(t *testing.T) {
    require.True(t, isRateLimitedNACKReason("central rate limit exceeded"))
}

func TestContainsCentralRateLimiterMarker(t *testing.T) {
    require.True(t, containsCentralRateLimiterMarker("vm_index_reports_rate_limiter"))
}
```

- [ ] **Step 2: Run the helper tests to verify they fail**

Run: `go test ./tests/vmhelpers -run 'TestHasReportedComponents|TestDetectRateLimitedNACK|TestContainsCentralRateLimiterMarker' -v`

Expected: FAIL with missing helper symbols.

- [ ] **Step 3: Implement single-condition waits and escalation snapshot helpers**

```go
func WaitForVMPresentInCentral(ctx context.Context, client v2.VirtualMachineServiceClient, namespace, name string) (*v2.VirtualMachine, error) // implemented via ListVirtualMachines
func WaitForVMIdentityFields(ctx context.Context, client v2.VirtualMachineServiceClient, id, expectedNamespace, expectedName string) (*v2.VirtualMachine, error)
func WaitForVMRunningInCentral(ctx context.Context, client v2.VirtualMachineServiceClient, id string) (*v2.VirtualMachine, error)
func WaitForVMScanNonNil(ctx context.Context, client v2.VirtualMachineServiceClient, id string) (*v2.VirtualMachine, error)
func WaitForVMScanTimestamp(ctx context.Context, client v2.VirtualMachineServiceClient, id string) (*v2.VirtualMachine, error)
func WaitForVMComponentsReported(ctx context.Context, client v2.VirtualMachineServiceClient, id string) (*v2.VirtualMachine, error)
func WaitForAllVMComponentsScanned(ctx context.Context, client v2.VirtualMachineServiceClient, id string) (*v2.VirtualMachine, error)
```

The observability helpers must assert both rate-limit conditions from the spec:

- no Central `vm_index_reports_rate_limiter` log marker for the investigated attempt
- no Sensor NACK reason equal to `central rate limit exceeded`

They must also collect and correlate the full escalation signal set from the design:

- Compliance metrics for accepted connections, reports received, successful sends, and mismatching vsock CID
- Sensor metrics for received reports, successful sends, ACKs, `rox_sensor_virtual_machine_index_reports_sent_total{status="central not ready"}`, `rox_sensor_virtual_machine_index_reports_sent_total{status="error"}`, and enqueue blocking
- Central metrics for `rox_central_resource_processed_count{Operation="Sync",Resource="VirtualMachineIndex"}`
- pod-level Compliance, Sensor, and Central logs needed to say where progress stopped

Escalation output must map these signals into the failure taxonomy required by the spec (guest run failed, relay accept failed, relay send failed, Sensor receive/forward failed, Central ACK missing, Central VM/scan visibility missing).

Metric selectors/labels must be sourced from `compliance/virtualmachines/relay/metrics/metrics.go`, `sensor/common/virtualmachine/metrics/metrics.go`, and `central/metrics/central.go`.

Add at least one parser/selector-focused unit test that fails when required metric names or labels drift from those source files.

`VM_SCAN_ESCALATION_ATTEMPT` must be interpreted as the attempt index inside a single scan wait and reset for each VM wait / each new `waitForScanWithEscalation` invocation.

- [ ] **Step 4: Run the helper tests again**

Run: `go test ./tests/vmhelpers -run 'TestHasReportedComponents|TestDetectRateLimitedNACK|TestContainsCentralRateLimiterMarker' -v`

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add tests/vmhelpers/central.go tests/vmhelpers/central_test.go tests/vmhelpers/observability.go tests/vmhelpers/observability_test.go tests/vmhelpers/wait.go
git commit -m "test(vm-scanning): add central polling and escalation helpers"
```

## Task 5: Implement suite preflight and persistent VM provisioning

**Files:**
- Modify: `tests/vm_scanning_suite_test.go`
- Modify: `tests/vm_scanning_utils_test.go`

- [ ] **Step 1: Implement `SetupSuite` preflight and persistent VM provisioning**

```go
func (s *VMScanningSuite) SetupSuite() {
    s.KubernetesSuite.SetupSuite()
    s.ctx, s.cleanupCtx, s.cancel = testContexts(s.T(), "TestVMScanning", 90*time.Minute)
    s.cfg = mustLoadVMScanConfig(s.T())
    s.restCfg = getConfig(s.T())
    s.k8sClient = createK8sClientWithConfig(s.T(), s.restCfg)
    s.dynamicClient = mustCreateDynamicClient(s.T(), s.restCfg)
    s.namespace = fmt.Sprintf("%s-%s", s.cfg.NamespacePrefix, uuid.NewV4().String()[:8])
    s.conn = centralgrpc.GRPCConnectionToCentral(s.T())
    s.vmClient = v2.NewVirtualMachineServiceClient(s.conn)
    s.mustWaitForHealthyCentralSensorConnection()
    s.mustVerifyVirtualMachinesFeatureEnabled()
    s.mustVerifyClusterVSOCKReady() // must verify both OCP VSOCK feature gate and required VM runtime path for roxagent
    s.provisionPersistentVMs()
    s.preparePersistentGuests()
}
```

`provisionPersistentVMs` must create exactly two persistent guests in one run:
- one using `VM_IMAGE_RHEL9` with `VM_GUEST_USER_RHEL9`
- one using `VM_IMAGE_RHEL10` with `VM_GUEST_USER_RHEL10`

`mustVerifyClusterVSOCKReady` must fail with explicit diagnostics if either condition is missing:

- OCP VSOCK feature gate is not enabled
- the VM runtime path required by `roxagent` is not present/usable

- [ ] **Step 2: Run the suite entrypoint to verify preflight reaches the first remaining runtime gap**

Run: `go test -tags test_e2e ./tests -run 'TestVMScanning' -v`

Expected: reaches VM provisioning / feature checks without config-validation failures.

- [ ] **Step 3: Implement shared guest preparation after provisioning**

```go
func (s *VMScanningSuite) preparePersistentGuests() {
    for _, vm := range s.persistentVMs {
        require.NoError(s.T(), vmhelpers.WaitForSSHReachable(s.ctx, s.virtctl, vm.Namespace, vm.Name))
        require.NoError(s.T(), vmhelpers.WaitForCloudInitFinished(s.ctx, s.virtctl, vm.Namespace, vm.Name))
        require.NoError(s.T(), vmhelpers.VerifySudoWorks(s.ctx, s.virtctl, vm.Namespace, vm.Name))
        require.NoError(s.T(), vmhelpers.CopyRoxagentBinary(s.ctx, s.virtctl, vm.Namespace, vm.Name, s.cfg.RoxagentBinaryPath))
        require.NoError(s.T(), vmhelpers.VerifyRoxagentBinaryPresent(s.ctx, s.virtctl, vm.Namespace, vm.Name))
        require.NoError(s.T(), vmhelpers.VerifyRoxagentExecutable(s.ctx, s.virtctl, vm.Namespace, vm.Name))
        require.NoError(s.T(), vmhelpers.VerifyRoxagentInstallPath(s.ctx, s.virtctl, vm.Namespace, vm.Name))
        activated, _, err := vmhelpers.GetActivationStatus(s.ctx, s.virtctl, vm.Namespace, vm.Name)
        require.NoError(s.T(), err)
        if !activated && s.cfg.ActivationOrg != "" && s.cfg.ActivationKey != "" {
            require.NoError(s.T(), vmhelpers.ActivateWithRHC(s.ctx, s.virtctl, vm.Namespace, vm.Name, s.cfg.ActivationOrg, s.cfg.ActivationKey, s.cfg.ActivationEndpoint))
            activated = true
        }
        if s.cfg.RequireActivation {
            require.True(s.T(), activated, "VM activation required but guest is not activated")
        }
        if activated {
            require.NoError(s.T(), vmhelpers.VerifyActivationSucceeded(s.ctx, s.virtctl, vm.Namespace, vm.Name))
        }
        require.NoError(s.T(), vmhelpers.PopulateDnfHistoryWithRandomPackage(s.ctx, s.virtctl, vm.Namespace, vm.Name))
    }
}
```

- [ ] **Step 4: Re-run the suite entrypoint**

Run: `go test -tags test_e2e ./tests -run 'TestVMScanning' -v`

Expected: persistent VMs are ready for the first canonical scan; later assertions may still fail.

- [ ] **Step 5: Implement `TeardownSuite` cleanup**

```go
func (s *VMScanningSuite) TearDownSuite() {
    defer s.cancel()
    for _, vm := range s.allVMs {
        _ = vmhelpers.DeleteVirtualMachine(s.cleanupCtx, s.dynamicClient, vm.Namespace, vm.Name)
    }
    _ = s.k8sClient.CoreV1().Namespaces().Delete(s.cleanupCtx, s.namespace, metav1.DeleteOptions{})
    if s.conn != nil {
        _ = s.conn.Close()
    }
}
```

- [ ] **Step 6: Commit**

```bash
git add tests/vm_scanning_suite_test.go tests/vm_scanning_utils_test.go
git commit -m "test(vm-scanning): add suite preflight and persistent guest setup"
```

## Task 6: Implement the happy-path suite flow

**Files:**
- Modify: `tests/vm_scanning_test.go`
- Modify: `tests/vm_scanning_suite_test.go`

- [ ] **Step 1: Write the failing happy-path suite test**

```go
func (s *VMScanningSuite) TestHappyPath() {
    for _, vm := range s.persistentVMs {
        result, err := s.ensureCanonicalScan(s.ctx, vm)
        require.NoError(s.T(), err)
        require.NotEmpty(s.T(), result.Stdout)
        require.True(s.T(), vmhelpers.VerboseOutputLooksLikeReport(result.Stdout))
        if result.UsedFallback {
            s.T().Logf("repo2cpe fallback used for %s/%s", vm.Namespace, vm.Name)
        }

        final, err := s.waitForScanWithEscalation(s.ctx, vm)
        require.NoError(s.T(), err)
        listed := s.mustListVMByNamespaceAndName(vm.Namespace, vm.Name)
        require.Equal(s.T(), listed.GetId(), final.GetId())
        require.Equal(s.T(), vm.Name, final.GetName())
        require.Equal(s.T(), vm.Namespace, final.GetNamespace())
        require.NotEmpty(s.T(), final.GetClusterId())
        require.NotEmpty(s.T(), final.GetClusterName())
        require.Equal(s.T(), v2.VirtualMachine_RUNNING, final.GetState())
        require.NotNil(s.T(), final.GetScan())
        require.NotNil(s.T(), final.GetScan().GetScanTime())
        if s.cfg.RequireActivation {
            require.NotEmpty(s.T(), final.GetScan().GetOperatingSystem())
            for _, component := range final.GetScan().GetComponents() {
                require.NotContains(s.T(), component.GetNotes(), v2.ScanComponent_UNSCANNED)
            }
            require.NotZero(s.T(), len(final.GetScan().GetComponents()))
        }

        fetched := s.mustGetVM(final.GetId())
        require.Equal(s.T(), final.GetId(), fetched.GetId())
    }
}
```

- [ ] **Step 2: Run the happy-path suite to verify it fails at the first missing runtime step**

Run: `go test -tags test_e2e ./tests -run 'TestVMScanning/TestHappyPath' -v`

Expected: FAIL with a concrete missing runtime step such as missing `ensureCanonicalScan` or `mustListVMByNamespaceAndName`.

- [ ] **Step 3: Implement the happy-path assertions and Central lookups**

```go
func (s *VMScanningSuite) mustListVMByNamespaceAndName(namespace, name string) *v2.VirtualMachine
func (s *VMScanningSuite) mustGetVM(id string) *v2.VirtualMachine
func (s *VMScanningSuite) ensureCanonicalScan(ctx context.Context, vm VMHandle) (*vmhelpers.RoxagentRunResult, error)
func (s *VMScanningSuite) waitForScanWithEscalation(ctx context.Context, vm VMHandle) (*v2.VirtualMachine, error)
```

`ensureCanonicalScan` must perform only the guest-side single-shot `roxagent --verbose` execution, assert exit code success, capture/assert on stdout and stderr, and record fallback usage.

Implementation note: keep `ensureCanonicalScan` as a thin suite wrapper around `vmhelpers.RunRoxagentOnce` so guest-run behavior remains reusable across happy path, lifecycle, and rescan scenarios.

`waitForScanWithEscalation` is the single orchestration point for Central polling after the guest-side run. It must compose the single-condition helpers from Task 4 in order (`WaitForVMPresentInCentral`, `WaitForVMIdentityFields`, `WaitForVMRunningInCentral`, `WaitForVMScanNonNil`, `WaitForVMScanTimestamp`, `WaitForVMComponentsReported`, and, when `VM_SCAN_REQUIRE_ACTIVATION=true`, `WaitForAllVMComponentsScanned`), start metrics/log escalation at `VM_SCAN_ESCALATION_ATTEMPT`, and keep polling Central until success or `VM_SCAN_TIMEOUT` even after escalation begins.

When `VM_SCAN_REQUIRE_ACTIVATION=false`, `waitForScanWithEscalation` must still enforce delivery/state checks (present, running, scan non-nil, scan timestamp, components reported) and only relax completeness checks that depend on activation (for example, full-scan/OS completeness assertions).

- [ ] **Step 4: Run the happy-path suite again**

Run: `go test -tags test_e2e ./tests -run 'TestVMScanning/TestHappyPath' -v`

Expected: PASS on a prepared OCP/CNV cluster, or fail with phase-localized diagnostics.

- [ ] **Step 5: Commit**

```bash
git add tests/vm_scanning_suite_test.go tests/vm_scanning_test.go
git commit -m "test(vm-scanning): implement happy-path suite flow"
```

## Task 7: Implement rescan and lifecycle scenarios

**Files:**
- Modify: `tests/vm_scanning_test.go`
- Modify: `tests/vm_scanning_suite_test.go`

- [ ] **Step 1: Write the failing rescan and lifecycle tests**

```go
func (s *VMScanningSuite) TestRescan() {
    vm := s.mustPersistentVM("rhel9")
    _, err := s.ensureCanonicalScan(s.ctx, vm)
    require.NoError(s.T(), err)
    beforeVM, err := s.waitForScanWithEscalation(s.ctx, vm)
    require.NoError(s.T(), err)
    before := beforeVM.GetScan().GetScanTime()
    vm.ID = beforeVM.GetId()
    _, err = s.ensureCanonicalScan(s.ctx, vm)
    require.NoError(s.T(), err)
    afterVM, err := s.waitForScanWithEscalation(s.ctx, vm)
    require.NoError(s.T(), err)
    after := afterVM.GetScan().GetScanTime()
    require.True(s.T(), after.AsTime().After(before.AsTime()))
    fetched := s.mustGetVM(afterVM.GetId())
    require.Equal(s.T(), beforeVM.GetId(), fetched.GetId())
}

func (s *VMScanningSuite) TestVMLifecycle() {
    transient := s.createTransientVM("lifecycle")
    require.NoError(s.T(), s.prepareGuest(transient))
    _, err := s.ensureCanonicalScan(s.ctx, transient)
    require.NoError(s.T(), err)
    _, err = s.waitForScanWithEscalation(s.ctx, transient)
    require.NoError(s.T(), err)
    require.NoError(s.T(), s.deleteAndWaitForRemoval(transient))
    s.mustAssertVMLifecycleGoneFromCentral(transient)
}
```

- [ ] **Step 2: Run each scenario to verify the initial failures**

Run: `go test -tags test_e2e ./tests -run 'TestVMScanning/TestRescan' -v`

Expected: FAIL until the baseline-timestamp and rerun helpers are wired.

Run: `go test -tags test_e2e ./tests -run 'TestVMScanning/TestVMLifecycle' -v`

Expected: FAIL until transient-VM creation/deletion and Central disappearance checks are wired.

- [ ] **Step 3: Implement the scenario-specific helpers**

```go
func (s *VMScanningSuite) mustGetScanTimestamp(id string) *timestamppb.Timestamp
func (s *VMScanningSuite) mustPersistentVM(name string) VMHandle
func (s *VMScanningSuite) createTransientVM(prefix string) VMHandle
func (s *VMScanningSuite) prepareGuest(vm VMHandle) error
func (s *VMScanningSuite) deleteAndWaitForRemoval(vm VMHandle) error
func (s *VMScanningSuite) mustAssertVMLifecycleGoneFromCentral(vm VMHandle)
```

`mustAssertVMLifecycleGoneFromCentral` must use bounded waits with `VM_DELETE_TIMEOUT` and verify both:

- `ListVirtualMachines` no longer returns the deleted VM
- `GetVirtualMachine` eventually returns `NotFound`

`deleteAndWaitForRemoval` must independently verify Kubernetes-side removal before the Central disappearance assertions run, and both Kubernetes and Central disappearance polls must reuse `VM_SCAN_POLL_INTERVAL` with `VM_DELETE_TIMEOUT`.

`prepareGuest` must be the same step-level guest-preparation chain used for the persistent VMs: SSH reachability, cloud-init completion, sudo, `roxagent` copy/verification, activation-status handling, and dnf-history priming.

`createTransientVM` must register the new VM in `s.allVMs` so `TeardownSuite` can clean it up even when scenario cleanup fails mid-test.

- [ ] **Step 4: Run the scenarios again**

Run: `go test -tags test_e2e ./tests -run 'TestVMScanning/TestRescan' -v`

Expected: PASS

Run: `go test -tags test_e2e ./tests -run 'TestVMScanning/TestVMLifecycle' -v`

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add tests/vm_scanning_test.go tests/vm_scanning_suite_test.go
git commit -m "test(vm-scanning): add rescan and lifecycle scenarios"
```

## Task 8: Add the dedicated shell entrypoint and cluster preflight

**Files:**
- Create: `tests/e2e/vm-scanning-lib.sh`
- Create: `tests/e2e/run-vm-scanning.sh`

- [ ] **Step 1: Write the shell scripts with syntax-visible TODO stubs**

```bash
ensure_vm_scanning_cluster_prereqs() {
    require_environment "KUBECONFIG"
    require_environment "VM_IMAGE_RHEL9"
    require_environment "VM_IMAGE_RHEL10"
    require_environment "VM_GUEST_USER_RHEL9"
    require_environment "VM_GUEST_USER_RHEL10"
    require_environment "VIRTCTL_PATH"
    require_environment "VM_SSH_PRIVATE_KEY"
    require_environment "VM_SSH_PUBLIC_KEY"
    require_environment "ROXAGENT_BINARY_PATH"
    require_environment "ROXAGENT_REPO2CPE_PRIMARY_URL"
    require_environment "ROXAGENT_REPO2CPE_FALLBACK_URL"
    require_environment "ROXAGENT_REPO2CPE_PRIMARY_ATTEMPTS"
    require_environment "VM_SCAN_NAMESPACE_PREFIX"
    require_environment "VM_SCAN_TIMEOUT"
    require_environment "VM_SCAN_POLL_INTERVAL"
    require_environment "VM_SCAN_ESCALATION_ATTEMPT"
    require_environment "VM_DELETE_TIMEOUT"
    require_environment "VM_SCAN_REQUIRE_ACTIVATION"
}
```

If `VM_SCAN_REQUIRE_ACTIVATION=true`, the shell preflight must also require `RHEL_ACTIVATION_ORG` and `RHEL_ACTIVATION_KEY` so the lane fails fast before deployment rather than deep inside the Go suite.

If set, `RHEL_ACTIVATION_ENDPOINT` must be passed through from CI/shell into the Go config, but it remains optional in preflight.

`WaitForCloudInitFinished` must run `cloud-init status --wait` and fail fast if the command is missing or returns failure.

- [ ] **Step 2: Verify shell syntax first**

Run: `bash -n tests/e2e/vm-scanning-lib.sh tests/e2e/run-vm-scanning.sh`

Expected: PASS

- [ ] **Step 3: Implement the dedicated E2E runner**

```bash
set -euo pipefail

test_vm_scanning_e2e() {
    local output_dir="${1:-vm-scanning-tests-results}"
    ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")"/../.. && pwd)"
    # shellcheck source=../../scripts/lib.sh
    source "$ROOT/scripts/lib.sh"
    # shellcheck source=../../scripts/ci/sensor-wait.sh
    source "$ROOT/scripts/ci/sensor-wait.sh"
    # shellcheck source=../../tests/scripts/setup-certs.sh
    source "$ROOT/tests/scripts/setup-certs.sh"
    # shellcheck source=../../tests/e2e/lib.sh
    source "$ROOT/tests/e2e/lib.sh"
    # shellcheck source=../../tests/e2e/vm-scanning-lib.sh
    source "$ROOT/tests/e2e/vm-scanning-lib.sh"
    export_test_environment
    setup_deployment_env false false
    ensure_vm_scanning_cluster_prereqs
    remove_existing_stackrox_resources
    setup_default_TLS_certs
    deploy_stackrox
    cd "$ROOT"
    go test ./tests/vmhelpers -v
    make -C tests TESTFLAGS="-race -p 1 -timeout 90m" vm-scanning-tests
    store_test_results "tests/vm-scanning-tests-results" "$output_dir"
}

test_vm_scanning_e2e "${1:-vm-scanning-tests-results}"
```

`store_test_results` source path must match the Make target output path (`tests/vm-scanning-tests-results`), so artifact collection is deterministic.

- [ ] **Step 4: Re-run shell syntax and a dry-run smoke**

Run: `bash -n tests/e2e/vm-scanning-lib.sh tests/e2e/run-vm-scanning.sh`

Expected: PASS

Run: `tests/e2e/run-vm-scanning.sh /tmp/vm-scanning-test-logs`

Expected: on a prepared cluster, deploy StackRox and run only `TestVMScanning`.

- [ ] **Step 5: Commit**

```bash
git add tests/e2e/vm-scanning-lib.sh tests/e2e/run-vm-scanning.sh
git commit -m "ci(vm-scanning): add dedicated e2e runner script"
```

## Task 9: Wire the suite into OpenShift CI

**Files:**
- Modify: `.openshift-ci/ci_tests.py`
- Create: `scripts/ci/jobs/ocp_vm_scanning_e2e_tests.py`

- [ ] **Step 1: Add the CI runner class**

```python
class VMScanningE2e(BaseTest):
    TEST_TIMEOUT = 2 * 60 * 60
    TEST_OUTPUT_DIR = "/tmp/vm-scanning-test-logs"

    def run(self):
        self.run_with_graceful_kill(
            ["tests/e2e/run-vm-scanning.sh", self.TEST_OUTPUT_DIR],
            self.TEST_TIMEOUT,
            output_dir=self.TEST_OUTPUT_DIR,
        )
```

- [ ] **Step 2: Verify the Python files still parse**

Run: `python3 -m py_compile .openshift-ci/ci_tests.py scripts/ci/jobs/ocp_vm_scanning_e2e_tests.py`

Expected: PASS

- [ ] **Step 3: Implement the dedicated OCP job**

```python
import os
from runners import ClusterTestRunner
from clusters import AutomationFlavorsCluster
from pre_tests import PreSystemTests
from ci_tests import VMScanningE2e
from post_tests import PostClusterTest, FinalPost

def assert_cnv_vsock_ready():
    # Fail fast if CNV/KubeVirt or VSOCK prerequisites are missing.
    ...

os.environ["DEPLOY_STACKROX_VIA_OPERATOR"] = "true"
os.environ["ORCHESTRATOR_FLAVOR"] = "openshift"
os.environ["SENSOR_SCANNER_SUPPORT"] = "true"
os.environ["ROX_DEPLOY_SENSOR_WITH_CRS"] = "true"
os.environ["SENSOR_HELM_MANAGED"] = "true"
os.environ["VM_SCAN_REQUIRE_ACTIVATION"] = "true"

assert_cnv_vsock_ready()

ClusterTestRunner(
    cluster=AutomationFlavorsCluster(),
    pre_test=PreSystemTests(),
    test=VMScanningE2e(),
    post_test=PostClusterTest(
        check_stackrox_logs=False,
    ),
    final_post=FinalPost(),
).run()
```

The new job should use the same `AutomationFlavorsCluster()` style as the existing OCP nongroovy job and call the new `VMScanningE2e` test class. It must also guarantee (or explicitly preconfigure/verify) CNV/KubeVirt plus VSOCK feature-gate readiness before invoking the Go suite.

Make this guarantee explicit in implementation (for example: documented cluster flavor contract plus pre-test verification that fails fast if CNV/VSOCK prerequisites are missing).

This pre-test verification must be an explicit runnable step in the job (not only documentation), and it must fail before `VMScanningE2e()` executes when CNV/VSOCK prerequisites are absent.

Include a concrete runnable pre-test in the job implementation (e.g., `assert_cnv_vsock_ready()`), and invoke it before `ClusterTestRunner(...).run()`.

- [ ] **Step 4: Re-run Python parsing and, if infra is available, invoke the job locally**

Run: `python3 -m py_compile .openshift-ci/ci_tests.py scripts/ci/jobs/ocp_vm_scanning_e2e_tests.py`

Expected: PASS

Run: `PYTHONPATH=.openshift-ci python3 scripts/ci/jobs/ocp_vm_scanning_e2e_tests.py`

Expected: starts the dedicated OCP VM-scanning lane against the provided cluster environment.

- [ ] **Step 5: Open and link the `openshift/release` PR**

The new runner script is not enough by itself. Before depending on this lane, add the matching job definition in the `openshift/release` repository so CI dispatch can resolve the new `ocp_vm_scanning_e2e_tests.py` entrypoint. That follow-up must include wiring of the full VM-scanning env/secrets contract (VM images/users, SSH keys, repo2cpe URLs/attempts, scan timeout knobs, activation inputs) and is a required completion criterion for CI wiring. Do not mark the CI lane done until both stackrox and openshift/release sides are merged.

- [ ] **Step 6: Verify both sides are tracked in task status**

Record links/IDs for:
- stackrox PR containing this lane
- openshift/release PR registering and configuring the job

- [ ] **Step 7: Commit**

```bash
git add .openshift-ci/ci_tests.py scripts/ci/jobs/ocp_vm_scanning_e2e_tests.py
git commit -m "ci(vm-scanning): add openshift vm-scanning job"
```

## Task 10: Final verification sweep

**Files:**
- Verify: `tests/vm_scanning_suite_test.go`
- Verify: `tests/vm_scanning_test.go`
- Verify: `tests/vmhelpers/*.go`
- Verify: `tests/e2e/run-vm-scanning.sh`
- Verify: `.openshift-ci/ci_tests.py`
- Verify: `scripts/ci/jobs/ocp_vm_scanning_e2e_tests.py`

- [ ] **Step 1: Run helper package unit tests**

Run: `go test ./tests/vmhelpers -v`

Expected: PASS

- [ ] **Step 2: Run the focused VM-scanning suite**

Run: `go test -tags test_e2e ./tests -run 'TestVMScanning' -v`

Expected: PASS on a prepared cluster.

- [ ] **Step 3: Run the Make target that CI will use**

Run: `make -C tests TESTFLAGS='-race -p 1 -timeout 90m' vm-scanning-tests`

Expected: PASS and emit `tests/vm-scanning-tests-results/report.xml`

- [ ] **Step 4: Verify shell and Python glue**

Run: `bash -n tests/e2e/vm-scanning-lib.sh tests/e2e/run-vm-scanning.sh && python3 -m py_compile .openshift-ci/ci_tests.py scripts/ci/jobs/ocp_vm_scanning_e2e_tests.py`

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add tests/vmhelpers tests/vm_scanning_suite_test.go tests/vm_scanning_test.go tests/vm_scanning_utils_test.go tests/e2e/vm-scanning-lib.sh tests/e2e/run-vm-scanning.sh tests/Makefile .openshift-ci/ci_tests.py scripts/ci/jobs/ocp_vm_scanning_e2e_tests.py
git commit -m "test(vm-scanning): complete go e2e suite and ci wiring"
```

If there are no new changes after task-level commits, skip creating an extra final commit.

## Notes for the Implementer

- Use `@use-modern-go` and target the Go version declared in `go.mod` (currently Go 1.25.0).
- Reuse existing `tests/common.go` helpers where package visibility allows; do not duplicate kubeconfig, logging, or artifact logic without a reason.
- Keep the default execution path Central-first. Only fetch logs and scrape metrics after the configured escalation attempt.
- Do not add no-agent VM scenarios in this change set; they are explicitly out of scope.
- Prefer deterministic predicates over sleeps. Every wait should have one condition and a structured failure message.
- For `VM_SCAN_REQUIRE_ACTIVATION=false` lanes, apply the reduced-assertion behavior from the Configuration section as the normative contract.
- If additional transient-VM scenarios are added later, each one must complete transient cleanup before the next transient scenario begins.
