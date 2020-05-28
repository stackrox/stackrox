# Changelog
All notable changes to this project that require documentation updates will be documented in this file.

## [NEXT RELEASE]

## [44.0]
- Previously, a scan for an image that may have been retagged (e.g. using the latest tag) would return a stale scan if it had been previously scanned.
- UI: In Platform Configuration > Interactions: 1. replace "AWS ECR" with "Amazon ECR" and 2. replace "S3" (and "AWS S3" placeholder for Integration Name in New Integration pane) with "Amazon S3" (ROX-4912)
- Docker Registry Integration now doesn't require entering password every time an existing integration is tested or updated (part of ROX-4539).
- UI: Replace `Page 1 of 0` with `Page 1 of 1` for 0 results in table pagination (ROX-1072)
- Added `ExportPolicies(POST /v1/policies/export)` API which accepts a list of policy IDs and returns a list of json policies
- Added `ImportPolicies(POST /v1/policies/import)` API which accepts a json list of policies, imports them into your StackRox installation, and returns a list with success/failure details per policy
- Added UI to export a single policy from the policy details on the System Policies page
- Added UI to import a single policy from the System Policies page
- Sensor resource requests and limits have been increased to 1 core / 1GB and 2 cores / 4GB respectively.
- Added User Page in UI to show current User Permissions and Roles
- `roxctl` commands now gives users an error message when unexpected arguments are given (ROX-4709)

## [43.0]
- Detection APIs were not properly handling suppressed CVEs and they were being included in evaluation. This is now resolved.
- Previously, the Scanner deployment did not mount the additional CA secret and thus would fail to scan self-signed registries. This is resolved.
- AWS S3 and AWS ECR integrations now accept an endpoint to work with non public AWS endpoints.
- UI: Fixed the display of the Privileged field when viewing a policy in the Vulnerability Management section (ROX-4752)
- API changes/deprecations related to supporting multiple roles:
  - `GenerateToken(/v1/apitokens/generate)`: the singular `role` field in the request field is deprecated; please use
    the array field `roles`.
  - `GetAPIToken(/v1/apitokens/{id})`, `GetAPITokens(/v1/apitokens)`: the singular `role` field in the response payload
    is deprecated; please use the array field `roles`.
  - Audit logs: the singular `user.role` field in the audit message payload is deprecated; please use the singular
    `user.permissions` field for the effective permissions of the user, and the array field `user.roles` for all the
    the individual roles associated with a user.
- The Compliance container within the Collector daemonset now has a hostpath of '/', which is needed to be able to read 
  configuration files anywhere on the host. This requires the allowedHostVolumes within the stackrox-collector PSP to allow '/' to be mounted.
  For added security, the PSP has set '/' as readonly and the Collector container's docker socket mount has also been set to readonly.
  
## [42.0]
- All `/v1/` API endpoints now support pretty-printing.  Make requests with the `?pretty` path parameter to receive pretty-printed json responses.
- UI: added "Deployment Name" property in side panel for Deployment Details on Violations and Risk pages.
- UI: In the Risk view, the URL now includes any search filters applied. You can now share the link and see the same filtered view.
- UI: In the Config Management section, fixed a UI crash issue when going from a single image view within containing context, like a single cluster, down to that image's deployments. (ROX-4543)
- UI: In the Platform Configuration -> Clusters view, the text “On the latest version” has been changed to “Up to date with Central version” (ROX-4739).
- `SuppressCVEs(/v1/cves/suppress)` endpoint now only supports cve suppression/snoozing.
- `SuppressCVEs(/v1/cves/suppress)` endpoint now supports cve suppression/snoozing for specific duration.
- Added `UnsuppressCVEs(/v1/cves/unsuppress)` endpoint to support cve un-suppression/un-snoozing.
- Changed central and sensor's SecurityContextConstraint (SCC) priority to 0 for OpenShift, so that they don't supercede default SCCs.

## [41.0]
### Changed
- Updated RHEL base images from UBI7.7 to UBI8.1

## [40.0]
### Added
- Added the ability to customize the endpoints exposed by Central via a YAML-based configuration file.
- Added a Required Image Label policy type.  Policies of this type will create a violation for any deployment containing images that lack the required label.  This policy type uses a regex match on either the key or the key and the value of a label.
- Added a Disallowed Image Label policy type.  Policies of this type will create a violation for any deployment containing images with the disallowed label.  This policy type uses a regex match on either the key or the key and the value of a label.

### Changed
- Collector images shipped with versions of the StackRox platform prior to this were affected by CVE-2019-5482, CVE-2019-5481 and CVE-2019-5436. The cause was an older version of curl that was vulnerable to buffer overflow and double free vulnerabilities in the FTP handler. We have upgraded curl to a version that does not suffer from these vulnerabilties. The curl program is only used to download new collector modules from a fixed set of URLs that do not make use of FTP, therefore according to our assessment there never existed a risk of an attacker exploiting this vulnerability.
- The `-e`/`--endpoint` argument of `roxctl` now supports URLs as arguments. The path in this URLs must either be empty
  or `/` (i.e., `https://central.stackrox` and `https://central.stackrox/` are both allowed, while
  `https://central.stackrox/api` is not). If this format is used, the URL scheme determines whether or not an unecrypted
  (plaintext) connection is established; if the `--plaintext` flag is used explicitly, its value has to be compatible
  with the chosen scheme (e.g., specifying an `https://` URL along with `--plaintext` will result in an error, as will
  a `http://` URL in conjunction with `--plaintext=false`).
- Detection and image enrichment have been moved to the individual Sensor clusters. Sensor will proxy image scan requests
  through Central and then run detection to generate both runtime and deploytime alerts. These alerts are sent to Central and any
  enforcement if necessary will be executed by Sensor without a roundtrip to Central.

## [39.0]
### Added
- `roxctl central cert` can be used to download Central's TLS certificate, which is then passed to `roxctl --ca`.
- The Scanner deployment has been split into two separate deployments: Scanner and Scanner DB. The Scanner deployment is now
  controlled by a Horizontal Pod Autoscaler (HPA) that will automatically scale up the scanner as the number of requests increase.
- Added a feature to report telemetry about a StackRox installation.  This will default to off in existing installations and can be enabled through the System Configuration page.
- Added a feature to download a diagnostic bundle.  This can be accessed through the System Configuration page or through `roxctl central debug download-diagnostics`
- A new `ScannerBundle` resource type (for the purposes of StackRox RBAC) is introduced. The resource definition for this is:
    Read permission: Download the scanner bundle (with `roxctl scanner generate`)
    Write permission: N/A
- Related to above, `roxctl scanner generate` now requires users to have read permissions to the newly created `ScannerBundle` resource.
  Previously, this endpoint was accessible to any authenticated user.
- OIDC auth providers now support refresh tokens, in order to keep you logged in beyond the ID token expiration time
  configured in your identity provider (sometimes only 15 minutes or less). In order to use refresh tokens, a client
  secret must be specified in the OIDC auth provider configuration.

### Changed
- UseStartTLS field in the Email notifier configuration has been deprecated in lieu of an enum which supports several
different authentication methods
- `roxctl central generate k8s` and `roxctl central generate openshift` no longer contain prompts for the monitoring stack because
  it is now deprecated
- The scanner v2 preview is now removed
- The scanner's updater now pulls from https://definitions.stackrox.io, and not https://storage.googleapis.com/definitions.stackrox.io/ like it previously would.
- Fixed https://stack-rox.atlassian.net/browse/ROX-3985.
- Collector images shipped with versions of the StackRox platform prior to this were affected by CVE-2017-14062. The cause was an older version of libidn (parsing of internationalized domain names) that was vulnerable due to a possible buffer overflow. We have upgraded libidn to a version that no longer suffers from this vulnerability. Since libidn is only used by curl, and curl is only used to download new collector modules from a fixed set of URLs that do not make use of international domain names, according to our assessment there never existed a risk of an attacker exploiting this vulnerability.

## [38.0]
### Added
- Added a REST endpoint `/v1/group` that can be used to retrieve a single group by exact property match (cf. ROX-3928).
- Scanner version updated to 2.0.4
- Collector version updated to 3.0.2

## [37.0]
### Changed
- The "NIST 800-190" standard has been renamed to "NIST SP 800-190", for correctness.
The ID continues to be the same, so no API calls will need to be updated.
Existing data will be preserved and available on upgrade.

### Added
- Added a `roxctl sensor get-bundle <cluster-name-or-id>` command to download sensor bundles for existing
  clusters by name or ID.

## [36.0]
### Changed
- Removed the endpoints `GET /v1/complianceManagement/schedules`, `POST /v1/complianceManagement/schedules`,
  `POST /v1/complianceManagement/schedules/{schedule_id}`, and `DELETE /v1/complianceManagement/schedules/{schedule_id}`.
  These were purely experimental and did not function correctly.  They were erroneously included in the public API specification.
- All YAML files have been updated to no longer reference the deprecated `extensions/v1beta1` API group. Previously,
 we used these API versions for deployments, daemonsets and pod security policies. This should have no effect on existing
 installs, but will mean that new installs can successfully install on Kube 1.16.

## [35.0]
- Proxy configuration can now be changed at runtime by editing and applying `proxy-config-secret.yaml` in the cluster
  where central and scanner run (ROX-3348, #3994, #4127).
- The component object within the image object now contains a field "Source", which indicates how the component was identified. Components derived from package managers
  will have the type "OS" whereas components derived from language analysis will have the language as the source (e.g. PYTHON).
### Added
- Images based on the Red Hat Universal Base Image (UBI) are published in stackrox.io/main-rhel,
  stackrox.io/scanner-rhel, stackrox.io/scanner-db-rhel and collector.stackrox.io/collector-rhel repositories. These
  images are functionally equivalent to our regular images and use the same version tags.

## [34.0]
### Added
- Policy whitelists are now shown in the UI. Previously, we only showed whitelisted deployment names, and not the entire structure that was
  actually in the policy object. This means that users can now whitelist by cluster, namespace and labels using the UI.
- There now exists a `roxctl collector support-packages upload <file>` command, which can be used to upload files from
  a Collector runtime support package to Central (e.g., kernel modules, eBPF probes). Assuming that Collectors can talk
  to Sensor, and Sensor can talk to Central, Collectors can then download these files they require at runtime from
  Central, even if none of the components has access the internet. Refer to the official documentation or contact
  StackRox support for information on obtaining a Collector support package.
- The `roxctl image scan` command now has a `--force` flag, which causes Central to re-pull the data from the registry and
  the scanner.

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
