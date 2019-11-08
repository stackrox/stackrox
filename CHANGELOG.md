# Changelog
All notable changes to this project that require documentation updates will be documented in this file.

## [34.0]
### Fixed
- Policy whitelists are now shown in the UI. Previously, we only showed whitelisted deployment names, and not the entire structure that was
actually in the policy object. This means that users can now whitelist by cluster, namespace and labels using the UI.

## [33.0]
### Changed
- Both the `runAsUser` and `fsGroup` for the central deployment are now 4000.
  This required changes in the the pod security policy, and the OpenShift Security Context Contraint (scc) objects.
  If you are upgrading from a previous version, please refer to the upgrade instructions on how to apply these changes
  to your existing deployment, pod security policy and OpenShift scc objects.
- CVEs with a CVSS score of 0 will now be displayed as "Pending" in the UI because it indicates that a CVE
  is still being analyzed or the CVE has been disputed. The API will continue to return a CVSS score of 0.
- Scopes now include support for Regex on the namespace and label fields including both Policy Scope and Whitelist Scope.
  The supported Regex syntax can be found here: https://github.com/google/re2/wiki/Syntax.
- The `validated` field in an auth provider is deprecated and will be removed in 3 releases. Please use the `active` field instead.
- RHSA vulnerabilities will now be displayed with the highest CVSS score from the CVEs it references. The referenced CVEs will
  now also be available. (ROX-3519, ROX-3550; d36f2ccf)
- `GetRisk(/v1/risks/{subjectType}/{subjectID})` endpoint is removed. For obtaining deployment risk, use `GetDeploymentWithRisk(/v1/deploymentswithrisk/{id})`. (8844549b)

## [32.0]
### Changed
- The port used for prometheus metrics can now be customized with the environment variable `ROX_METRICS_PORT`. Supported
  options include `disabled`, `:port-num` (will bind to wildcard address) and `host_or_addr:port`. IPv6 address literals
  are supported with brackets, like so: `[2001:db8::1234]:9090`. The default setting is still `:9090`. (ROX-3209)
- The `roxctl sensor generate` and `roxctl scanner generate` subcommands now accept an optional `--output-dir <dir>` flag
  that can be used to extract the bundle files to a custom directory. (ROX-2529)
- The `roxctl central debug dump` subcommand now accepts an optional `--output-dir <dir>` flag
  that can be used to specify a custom directory for the debug zip file.
- The format of collector tags changed from `<version>` to `<version>-latest`. This tag references a *mutable* image in
  canonical upstream repository (`collector.stackrox.io/collector`) that will get updated whenever kernel modules/eBPF
  probes for new Linux kernel versions become available. This decreases the need to rely on module downloads via
  the internet. If you configure StackRox to pull collector images from your private registry, you need to configure a
  periodic mirroring to take advantage of this effect.

## [31.0]
### Changed
- `roxctl` can now talk to Central instances exposed behind a non-gRPC-capable proxy (e.g., AWS ELB/ALB). To support
  this, requests go through an ephemeral client-side reverse proxy. If you observe any issues with `roxctl` that you
  suspect might be due to this change, pass the `--direct-grpc` flag to resort to the old connection behavior.
- `roxctl` can now talk to Central instances exposed via plaintext (either directly, or via a plaintext proxy talking to
  a plaintext or TLS-enabled backend). While we advise against this, this behavior can be enabled via the `--plaintext`
  flag in conjunction with the `--insecure` flag.
- `roxctl` now has a `--tolerations` flag that is true by default, and can be set to false to disable tolerations for
  tainted nodes from being added into `sensor.yaml`. If the flag is set to true, collectors will be deployed to and run on
  all nodes of the cluster.
- Changes to default TLS cert and `htpasswd` secrets (`central-default-tls-cert` and `central-htpasswd`) are now picked
  up automatically, without needing to restart Central. Note that Kubernetes secret changes may take up to a minute to
  get propagated to the pod.

## [30.0]
### Changed
- `TriggerRun(/v1/complianceManagement/runs)` endpoint is removed. All clients should use `TriggerRuns(/v1/compliancemanagement/runs)` to start a compliance run.
- The EmitTimestamp field that was unset in the ProcessIndicator resource has been removed
- Link field is removed from the violation message

## [28.0]
### Changed
- The Prometheus scrape endpoint has been moved from localhost:9090 to :9090 so users can use their own Prometheus installations and pull StackRox metrics.
- UpdatedAt in the deployment object has been corrected to Created

## [27.0]
### Changed
- Reprocessing of deployments and images has been moved to an interval of 4 hours
- Improved user experience for `roxctl central db restore`:
  - Resuming restores is now supported, either after connection interruptions (automatic) or
    after terminating `roxctl` (manual). In the latter case, resuming is performed automatically
    by invoking `roxctl` with the same database export file.
  - The `--timeout` flag now specifies the relative time (from the start of the `roxctl` invocation) after which
    `roxctl` will no longer automatically try to resume restore operations. This does not affect the
    restore operation on the server side, it can still be resumed by restarting `roxctl` with the same parameters.
  - Restore operations cannot be resumed across restarts of the Central pod. If a restore
    operation is interrupted, it must be resumed within 24 hours (and before Central restarts), otherwise it will be canceled.
  - `roxctl central db restore status` can be used to inspect the status of the ongoing restore process,
    if any.
  - `roxctl central db restore cancel` can be used to cancel any ongoing restore process.
  - The `--file` flag is deprecated (but still valid). The preferred invocation now is
    `roxctl central db restore <file>` instead of `roxctl central db restore --file <file>`. If for
    any reason the name of the database export file is `status` or `cancel`, insert an `--` in front
    of the file name, e.g., `roxctl central db restore -- status`.

### Added
- `roxctl central db backup` now supports an optional `--output` argument to specify the output location to write the backup to.

## [25.0]
### Added
- `roxctl sensor generate openshift` can be used to generate sensor bundles for OpenShift clusters from
  the command line.
### Changed
- Removed _DebugMetrics_ resource.  
  Only users with _Admin_ role can access `/debug` endpoint.  
  _Note: This is also applicable with authorization plugin for scoped access control enabled._
- Due to the addition of the `roxctl sensor generate openshift` command, the `--admission-controller`
  flags that are exclusive to Kubernetes (non-OpenShift, `k8s`) clusters must be specified *after* the
  `k8s` command.  
  For example, `roxctl sensor generate --admission-controller=true k8s` is no longer a
  legal invocation; use `roxctl sensor generate k8s --admission-controller=true` instead.


## [24.0]
### Changed
- Queries against time fields involving a duration have now flipped directionality to a more intuitive way.
  Previously, searching `Image Creation Time: >3h` would show all images created _after_ 3 hours before the current time;
  now, it shows all images created more than three hours ago -- that is, _before_ the moment in time 3 hours before the current time.
- Removed the `/v1/deployments/metadata/multipliers` API.  User defined risk multipliers will no longer be taken into account.


## [23.0]
### Added
- Installer prompt to configure the size of the external volume for central.
### Changed
- Prometheus endpoint changed from https://localhost:8443 to http://localhost:9090.
- Scanner is now given certificates, and Central<->Scanner communication secured via mTLS.
- Central CPU Request changed from 1 core to 1.5 cores
- Central Memory Request changed from 2Gi to 4Gi
- Sensor CPU Request changed from .2 cores to .5 cores
- Sensor Memory Request changes from 250Mi to 500Mi
- Sensor CPU Limit changed from .5 cores to 1 core


## [22.0]
### Changed
- Default size of central's PV changed from 10Gi to 100Gi.
