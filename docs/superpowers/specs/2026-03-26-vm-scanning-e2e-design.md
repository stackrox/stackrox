# VM Scanning End-to-End Test Design

## Goal

Design a low-flake Go end-to-end test suite for the VM scanning feature that exercises the full product path from a KubeVirt guest VM running `roxagent` through Compliance, Sensor, Central, and the v2 `VirtualMachineService` API.

## Scope

This design covers:

- Full pipeline E2E on OpenShift with OCP Virtualization (CNV)
- Go-based tests under `tests/`
- RHEL 9 and RHEL 10 guest VMs in the same suite run
- Step-level validation for provisioning, guest preparation, scan execution, and Central visibility
- Dedicated CI coverage on the latest stable OCP version as the initial rollout

This design does not cover:

- Synthetic Sensor-to-Central integration tests
- Scanner v4 enrichment correctness assertions for specific CVEs
- Feature flag disabled behavior in the E2E suite
- Full OCP version matrix in the first rollout

## Key Decisions

- Use Go E2E tests instead of `qa-tests-backend` Groovy/Spock tests.
- Exercise the full VM scanning path, not only Central-side processing.
- Run one suite that provisions two VMs side by side: one RHEL 9 VM and one RHEL 10 VM.
- Use independent test functions with shared helpers instead of one monolithic scenario runner.
- Keep tests serial in the first release to reduce flake risk.
- Use `virtctl ssh` and `virtctl scp` for guest interaction.
- Install `roxagent` before activation. Activate the guest after installation and before the canonical scan whose results the tests assert on. Conduct `dnf install` with a random package on the VM to populate the caches and dnf history database. 
- Assert on Central first. If the expected scan does not arrive in time, escalate to Compliance and Sensor metrics/log diagnostics.

## Product Context

The feature under test has the following path:

1. A KubeVirt VM runs `roxagent` inside the guest.
2. `roxagent` builds a VM index report and sends it to the host over vsock.
3. Compliance's VM relay accepts the vsock connection, validates the report, and forwards it to Sensor.
4. Sensor maps vsock CID to the VM known from KubeVirt inventory and forwards the VM index report to Central.
5. Central enriches and stores the resulting VM scan and exposes it through the v2 `VirtualMachineService` API.

The E2E tests should verify that this full path works with real VMs and real guest-side setup.

## Test Location and File Layout

The suite should live under `tests/` and follow the existing Go E2E conventions:

```text
tests/
├── vm_scanning_suite_test.go
├── vm_scanning_test.go
├── vm_scanning_utils_test.go
└── vmhelpers/
    ├── vm.go
    ├── guest.go
    ├── roxagent.go
    ├── virtctl.go
    ├── central.go
    ├── observability.go
    └── wait.go
```

### File responsibilities

- `tests/vm_scanning_suite_test.go`
  - suite definition
  - `SetupSuite` and `TeardownSuite`
  - shared namespace, clients, and VM handles
- `tests/vm_scanning_test.go`
  - scenario tests
  - product-facing assertions
- `tests/vm_scanning_utils_test.go`
  - test-only assertion helpers
  - failure-report formatting
- `tests/vmhelpers/vm.go`
  - VM creation, VMI readiness, deletion
- `tests/vmhelpers/guest.go`
  - SSH reachability, command execution, activation checks
- `tests/vmhelpers/roxagent.go`
  - copy binary, verify install, trigger scans
- `tests/vmhelpers/virtctl.go`
  - wrapper around `virtctl ssh` and `virtctl scp`
- `tests/vmhelpers/central.go`
  - `VirtualMachineService` polling helpers
- `tests/vmhelpers/observability.go`
  - optional escalation helpers for Compliance/Sensor metrics and logs
- `tests/vmhelpers/wait.go`
  - generic polling utilities with timeouts and structured diagnostics

## Suite Architecture

The suite should use `testify/suite` and the existing `test_e2e` build tag pattern used in `tests/`.

### Cluster ownership boundary

The Go E2E suite assumes that the target OCP cluster already exists and is reachable through `kubeconfig` when the test process starts. Cluster creation and cluster-wide preconfiguration are out of scope for the test binary itself.

The CI job may still create and prepare that cluster as part of job orchestration. In that model, the CI layer is responsible for:

- creating or acquiring the OCP cluster
- exposing `kubeconfig` to the test environment
- enabling CNV/KubeVirt and the OCP VSOCK feature gate
- deploying StackRox
- preparing required secrets, VM images, `virtctl`, activation inputs, and repo-to-CPE fallback data

Both existing infra-backed clusters and job-created clusters are valid for this design, as long as they satisfy the suite's preconditions before the Go tests begin.

### Shared suite setup

`SetupSuite` should:

1. Connect to Central using `pkg/testutils/centralgrpc`.
2. Create a `v2.VirtualMachineServiceClient`.
3. Build Kubernetes clients for cluster operations.
4. Create a dedicated namespace for the suite.
5. Verify that the `VirtualMachines` feature is enabled.
6. Verify that the OCP cluster has the VSOCK feature gate enabled and that the VM runtime path needed by `roxagent` is available.
7. Provision two VMs:
   - one RHEL 9 VM
   - one RHEL 10 VM
8. Bring each VM through guest preparation:
   - wait for SSH
   - verify `sudo`
   - copy `roxagent`
   - verify install
   - activate with `rhc` if needed and activation credentials are available
   - verify activation status

`SetupSuite` should not require Central scan data yet. It should stop after both VMs are ready for the first canonical scan so that the happy-path test owns the first end-to-end scan assertions.

`TeardownSuite` should:

1. Delete all test-created VMs.
2. Delete the suite namespace.
3. Close the Central gRPC connection.

### Shared suite state

The suite should keep references to:

- Central gRPC connection
- `VirtualMachineServiceClient`
- Kubernetes clients
- suite namespace
- a handle for the RHEL 9 VM
- a handle for the RHEL 10 VM

### Execution model

- One suite run per CI job
- Two guest VMs in the same run
- No `t.Parallel()` in the initial rollout
- The happy-path assertions over the two persistent VMs also serve as the initial multi-VM coverage
- Test methods must not rely on execution order. Any scenario that needs prior scan state must establish that state within the test itself.

## Guest Preparation and Scan Flow

The guest preparation flow should be step-oriented. Each step must assert success before proceeding to the next one.

### Provisioning checkpoints

For each VM:

1. Create the `VirtualMachine` object, including cloud-init user data that injects the SSH public key the suite will use for guest access.
2. Wait for the corresponding `VirtualMachineInstance` to appear.
3. Wait for the VMI to reach `Running`.
4. Wait for SSH reachability.
5. Verify cloud-init completion by running `cloud-init status --wait` over SSH.

The suite assumes cloud-init-capable guest images because cloud-init is required to inject SSH access. If `cloud-init status --wait` is unavailable or fails, provisioning should fail fast as an image or prerequisite problem.

### Install and activation checkpoints

For each VM:

1. Verify `sudo` works non-interactively.
2. Copy `roxagent` into the guest.
3. Verify the binary is present.
4. Verify the binary is executable.
5. Verify the expected install path is correct.
6. Check activation state.
7. If activation credentials are configured and the guest is not activated:
   - run `rhc` activation
   - verify activation succeeded

`roxagent` may run on a non-activated system, but results can be incomplete. Therefore, installation must not depend on activation, but the scan used for primary assertions should be performed after activation.

### Canonical scan checkpoints

For each VM:

1. Run `roxagent --verbose` in single-shot mode.
2. Capture stdout and stderr from the guest command.
3. Verify the guest command exits successfully.
4. Assert on the verbose output before polling Central. At minimum, the output must show that `roxagent` produced report data rather than exiting quietly or failing before report generation.
5. Use a bounded repo-to-CPE lookup strategy when invoking `roxagent`:
   - first try the preferred remote `--repo-cpe-url` for a limited number of attempts
   - if those attempts fail due to connectivity, timeout, or fetch errors, rerun with `--repo-cpe-url` pointing at a local fallback copy
6. Treat fallback to the local copy as a supported degraded mode, not a test failure, but record in the test output that fallback was used.
7. Poll Central for the VM and its scan.
8. If Central does not reflect the scan after repeated attempts, escalate to Compliance and Sensor observability checks.

Transient VMs used by scenario tests must reuse the same step-level helpers as the persistent VMs. The difference is only where the flow stops:

- lifecycle transient VM: full flow through canonical scan, then deletion assertions

## Progressive Debugging and Anti-Flake Strategy

The suite should prefer a fast, optimistic path for successful runs and only pay the cost of deeper tracing when Central does not reflect the expected scan in time.

### Fast path

For the first several attempts, poll only Central state:

- VM visible in `ListVirtualMachines`
- expected VM identity fields present
- `scan` non-nil
- `scan_time` set
- one or more components reported
- all reported components fully scanned

These checks must still be implemented as a composition of single-condition polling helpers. For example:

- `WaitForVMPresentInCentral`
- `WaitForVMRunningInCentral`
- `WaitForVMScanNonNil`
- `WaitForVMScanTimestamp`
- `WaitForVMComponentsReported`
- `WaitForAllVMComponentsScanned`

Any higher-level helper such as `WaitForVMScanInCentralWithEscalation` is an orchestration helper that calls these single-condition waits in order and owns the escalation logic. It must not replace step-level failure reporting.

The component checks must distinguish two different Central states:

- `WaitForVMComponentsReported`: the VM scan contains `N > 0` components, regardless of whether they have been fully scanned yet
- `WaitForAllVMComponentsScanned`: all reported components are fully scanned, meaning no returned `ScanComponent` carries the `UNSCANNED` note

This distinction is important because a VM may already have inventory data in Central while still having zero fully scanned components.

### Escalation path

If Central still does not show the expected scan after a bounded number of retries, switch to a deeper diagnostic path:

- collect Compliance metrics
- collect Sensor metrics
- collect targeted Compliance logs
- collect targeted Sensor logs
- continue polling Central while attaching the richer diagnostics to the eventual failure output

This keeps the common case lightweight while still making stalled reports debuggable.

### Metrics and logs used in escalation

On escalation, inspect metrics and logs that show the message advancing through the pipeline.

Compliance metrics of interest:

- `rox_compliance_virtual_machine_relay_connections_accepted_total`
- `rox_compliance_virtual_machine_relay_index_reports_received_total`
- `rox_compliance_virtual_machine_relay_index_reports_sent_total{failed="false"}`
- `rox_compliance_virtual_machine_relay_index_reports_mismatching_vsock_cid_total`

Sensor metrics of interest:

- `rox_sensor_virtual_machine_index_reports_received_total`
- `rox_sensor_virtual_machine_index_reports_sent_total{status="success"}`
- `rox_sensor_virtual_machine_index_report_acks_received_total{action="ACK"}`
- `rox_sensor_virtual_machine_index_reports_sent_total{status="central not ready"}`
- `rox_sensor_virtual_machine_index_reports_sent_total{status="error"}`
- `rox_sensor_virtual_machine_index_report_enqueue_blocked_total`

Central metrics of interest:

- `rox_central_resource_processed_count{Operation="Sync",Resource="VirtualMachineIndex"}`

Relevant log markers include messages for:

- relay send to Sensor
- Sensor receipt of the VM report
- Sensor handling of the vsock CID
- Sensor ACK receipt from Central
- Central receipt of the VM index report
- Central successful enrichment and storage of the VM scan
- Central VM index rate-limiter rejection marker `vm_index_reports_rate_limiter`

Metrics should be the primary signal during escalation. Logs should be used as a correlation aid and to improve failure reports.

The source of truth for metric names and label sets is:

- `compliance/virtualmachines/relay/metrics/metrics.go`
- `sensor/common/virtualmachine/metrics/metrics.go`
- `central/metrics/central.go`

Metric collection code in the test helpers should centralize these selectors so later instrumentation changes are localized.

The escalation path must also verify that Central did not reject the VM index report due to rate limiting. In this code path, rate limiting is surfaced through:

- a Central log marker named `vm_index_reports_rate_limiter`
- a VM index NACK with reason `central rate limit exceeded`

So on escalation the helpers should assert both:

- the Central rate-limiter log marker does not appear for the VM scan attempt being investigated
- the Sensor-side NACK path does not report the reason `central rate limit exceeded`

Central rate limiting happens before the VM index pipeline runs, so absence of the `rox_central_resource_processed_count{Operation="Sync",Resource="VirtualMachineIndex"}` increment combined with the rate-limit marker or NACK reason indicates the report was rejected before normal Central processing.

The escalation path must not depend on a cluster Prometheus deployment. `observability.go` should:

- discover the active Compliance and Sensor pods through Kubernetes
- fetch logs directly from those pods using Kubernetes pod log APIs or equivalent `oc logs` behavior
- scrape `/metrics` directly from those pods by creating short-lived local port-forwards during escalation

Pod selectors, namespaces, and metric endpoints should be centralized in one place in the helper package.

### Failure-report goal

At final failure, the suite should be able to say where progress stopped:

- guest execution failed
- relay did not accept the connection
- relay did not send to Sensor
- Sensor did not receive or forward the report
- Central did not acknowledge the message
- Central did not expose the VM
- Central exposed the VM but not the scan

## Helper API Design

Avoid coarse helpers that hide multiple phases. Prefer helpers that match one observable step each.

Examples of helper responsibilities:

- `CreateVirtualMachine`
- `WaitForVirtualMachineInstanceReady`
- `WaitForSSHReachable`
- `WaitForCloudInitFinished`
- `CopyRoxagentBinary`
- `VerifyRoxagentBinaryPresent`
- `VerifyRoxagentExecutable`
- `GetActivationStatus`
- `ActivateWithRHC`
- `VerifyActivationSucceeded`
- `RunRoxagentOnce`
- `WaitForVMPresentInCentral`
- `WaitForVMRunningInCentral`
- `WaitForVMComponentsReported`
- `WaitForAllVMComponentsScanned`
- `WaitForVMScanInCentral`
- `WaitForVMScanInCentralWithEscalation`
- `DeleteVirtualMachine`
- `WaitForVirtualMachineDeleted`

Each polling helper must have:

- one condition only
- explicit timeout
- explicit poll interval
- targeted diagnostics on failure

No helper should silently perform multiple waits that make flakes hard to localize.

## Scenarios

The first rollout should cover the following scenarios.

### 1. Happy path

For both the RHEL 9 and RHEL 10 VMs:

- run the first canonical `roxagent` scan after setup is complete
- assert both VMs appear through `ListVirtualMachines`
- assert the expected identity fields
- assert `state == RUNNING`
- assert `scan` exists
- assert `scan_time` exists
- assert `operating_system` is populated
- assert `N > 0` components are reported
- assert all reported components are fully scanned, meaning none of them carry the `UNSCANNED` note
- assert `GetVirtualMachine` returns the expected data

### 2. VM lifecycle

Create a third short-lived VM and verify:

- run the same guest-preparation and canonical scan flow used for the persistent VMs
- VM appears in Central
- VM gains scan data
- after deletion, the VM is removed from Kubernetes
- `ListVirtualMachines` no longer includes the VM
- `GetVirtualMachine` for the deleted VM eventually returns `NotFound`

Deletion checks must use an explicit bounded wait with the same poll interval as other waits and a dedicated delete timeout.

### 3. Rescan

For one of the persistent VMs:

- establish baseline scan state within the test if needed
- record the current `scan_time`
- run `roxagent` again
- wait for a newer `scan_time`
- verify the scan remains attached to the same VM

### 4. Multiple VMs

This is covered implicitly by the suite design because the RHEL 9 and RHEL 10 VMs are both provisioned, scanned, and asserted independently in the same run.

### Out of scope for this suite

- feature flag disabled behavior
- VMs that do not have `roxagent` installed or running
- specific vulnerability correctness assertions for known packages and CVEs

Feature flag disabled behavior should be covered later by an integration-oriented test that can safely restart or reconfigure Central without destabilizing the E2E environment.

## CI Design

The initial rollout should use a dedicated OpenShift CI lane rather than modifying all existing OCP jobs immediately.

### Initial CI shape

- one dedicated VM scanning E2E job
- start with the latest supported OCP as the first covered version
- two guest images in the same suite run:
  - RHEL 9
  - RHEL 10
- the CI job may either provision a fresh OCP cluster or reuse an existing infra-backed cluster, but in both cases it must complete all cluster-wide preconfiguration before invoking the Go suite
- cluster bring-up must enable the OCP VSOCK feature gate before the test suite starts
- serial execution
- enough diagnostic collection to debug failures from artifacts alone

This reduces the risk of introducing flakes into existing jobs while the suite stabilizes.

### Future CI direction

After the suite is stable:

- make KubeVirt enablement available on all OCP-based E2E clusters
- extend OCP version coverage
- decide whether to split smoke and extended VM scenarios if runtime becomes too high

## Configuration and Secrets

The suite should be fully parameterized through environment variables or CI-injected config.

Core inputs:

- `VM_IMAGE_RHEL9`
- `VM_IMAGE_RHEL10`
- `VM_GUEST_USER_RHEL9`
- `VM_GUEST_USER_RHEL10`
- `VIRTCTL_PATH`
- `ROXAGENT_BINARY_PATH`
- `ROXAGENT_REPO2CPE_PRIMARY_URL`
- `ROXAGENT_REPO2CPE_FALLBACK_URL`
- `ROXAGENT_REPO2CPE_PRIMARY_ATTEMPTS`
- `VM_SSH_PRIVATE_KEY`
- `VM_SSH_PUBLIC_KEY`
- `VM_SCAN_NAMESPACE_PREFIX`
- `VM_SCAN_TIMEOUT`
- `VM_SCAN_POLL_INTERVAL`
- `VM_SCAN_ESCALATION_ATTEMPT`
- `VM_DELETE_TIMEOUT`
- `VM_SCAN_REQUIRE_ACTIVATION`

Activation-related inputs:

- `RHEL_ACTIVATION_ORG`
- `RHEL_ACTIVATION_KEY`
- optional activation endpoint override if required by the environment

The suite should support both:

- pre-activated images
- images that must be activated during the test

If the CI lane requires activation for the primary assertions and activation credentials are missing, the suite should fail fast with a prerequisite error instead of running a degraded happy-path test.

The initial dedicated CI lane should set `VM_SCAN_REQUIRE_ACTIVATION=true`. Rich happy-path assertions such as populated operating system fields and non-empty scan contents are only valid when the guest is known to be pre-activated or has been activated successfully during the test. If a future lane sets `VM_SCAN_REQUIRE_ACTIVATION=false`, its assertions must be reduced to delivery and state checks rather than completeness checks.

Parameter meanings:

- `VM_SCAN_TIMEOUT`: total timeout for one scan-visibility wait in Central, counted per VM and per scan attempt
- `VM_SCAN_POLL_INTERVAL`: poll interval used by Central wait helpers
- `VM_SCAN_ESCALATION_ATTEMPT`: poll attempt number within a single `WaitForVMScanInCentralWithEscalation` invocation at which Compliance and Sensor diagnostics begin; resets for each VM wait
- `VM_DELETE_TIMEOUT`: total timeout for post-deletion Central and Kubernetes disappearance checks
- `VM_SCAN_REQUIRE_ACTIVATION`: boolean flag for CI lanes where the primary scan assertions must only run after successful activation; when `true`, missing activation credentials are a hard prerequisite failure
- `ROXAGENT_BINARY_PATH`: absolute path to the host-side `roxagent` binary that will be copied into guest VMs
- `ROXAGENT_REPO2CPE_PRIMARY_URL`: preferred remote URL for repo-to-CPE mapping lookups
- `ROXAGENT_REPO2CPE_FALLBACK_URL`: local fallback URL or other implementation-validated local source for the repo-to-CPE mapping used when remote fetches fail
- `ROXAGENT_REPO2CPE_PRIMARY_ATTEMPTS`: bounded number of attempts to use the preferred remote mapping source before falling back to the local copy

The VM manifest generation helpers must use `VM_SSH_PUBLIC_KEY` when constructing the cloud-init user data so the suite can reach the guest with the matching private key.

## Reliability Principles

To minimize flakes:

- keep tests serial in the first rollout
- keep at most three VMs alive at once: the two persistent suite VMs plus one transient VM for lifecycle scenarios
- require transient VM cleanup to complete before the next transient scenario begins
- perform one assertion per wait condition
- prefer product-state checks over log scraping
- use logs and metrics for escalation, not as the default success path
- stop at the first failed phase and emit focused diagnostics
- do not require daemon mode for the initial design
- use explicit single-shot `roxagent` execution for deterministic rescans
- avoid making the suite depend on live Internet access for repo-to-CPE mapping; prefer bounded remote attempts followed by a deterministic local fallback

## Recommended Next Step

After this spec is reviewed and approved, write a detailed implementation plan that:

- maps exact files to create or modify
- breaks the work into small TDD-style tasks
- includes concrete commands for local verification and CI wiring
