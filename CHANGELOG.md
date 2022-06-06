# Changelog
Entries in this file should be limited to:
-  Any changes that introduce a deprecation in functionality, OR
-  Obscure side-effects that are not obviously apparent based on the JIRA associated with the changes.
Please avoid adding duplicate information across this changelog and JIRA/doc input pages.

## [NEXT RELEASE]

- ROX-9358: The diagnostic bundle includes notifiers, auth providers and auth provider groups, access control roles with attached permission set and access scope, and system configuration. Users with `DebugLogs` permission will be able to read listed entities from a generated diagnostic bundle regardless of their respective permissions.
- ROX-10819: The documentation for API v1/notifiers ("GetNotifiers") previously stated that the request could be filtered by name or type. This is incorrect as this API never allowed filtering. The documentation has been fixed to reflect that.
- ROX-9614: Add `file` query parameter to Central's `/api/extensions/scannerdefinitions`, allowing retrieval of individual files (not directories) from Scanner's Definition bundle using their full path within the archive. Add `sensorEndpoint` to Scanner's configmap, so Scanner in slim mode knows how to reach Sensor from its cluster.
- ROX-9928: Policy "OpenShift: Advanced Cluster Security Central Admin Secret Accessed" renamed to "OpenShift: Central Admin Secret Accessed"
- ROX-8277: changed UserAgent Header for all requests from stackrox operator to kubernetes API server to show appropriate version of the operator, for example: `rhacs-operator/v3.70.0 opensource (linux/amd64)`
- `ids` field in `/v1/cves/suppress` and `/v1/cves/unsuppress` API payload will be renamed to `cves` in 72.0 release.
- `cves.ids` field of `storage.VulnerabilityRequest` object, which is in the response of `VulnerabilityRequestService` endpoints, will be renamed to `cves.cves` in 72.0 release.
- ROX-8520: Permissions for permission sets will be grouped for simplification. As a result, the following permissions will be deprecated in favor of a new permission:
  - New permission `Access` will deprecate the permissions `AuthPlugin, AuthProvider, Group, Licenses, Role, User`.
  - New permission `DeploymentExtension` will deprecate the permissions `Indicator, NetworkBaseline, ProcessWhitelist, Risk`.
  - New permission `Integration` will deprecate the permissions `APIToken, BackupPlugins, ImageIntegration, Notifier, SignatureIntegration`.
  Each deprecated permission will be removed in a future release.
- /v1/telemetry and /v1/licenses endpoints, and related CLI functionality, are now deprecated and will be removed in 2 releases.
  - These endpoints are deprecated as license files are not required to run the platform
- `firstNodeOccurrence` field of `storage.Node` object, which is in the response of Node endpoints, has been removed.
- `vulns` fields of `storage.Node` object, which is in the response payload of `v1/nodes` is deprecated and will be removed in 72.0.

## [70.0]

- The default Admission Controller "fail open" timeout has been changed from 3 seconds to 20 seconds in Helm templates.
- The maximum Admission Controller "fail open" timeout has been set at 25 seconds in Helm template verification performed by the Operator.
  - This change is *not* backwards compatible; if an existing Custom Resource sets the value to > 25 seconds, then it will fail validation in case operator is downgraded. This change is accepted because the operator is still in v1alpha1 and subject to change.
- The admission webhook timeout is now set to the admission controller timeout plus 2 seconds.
- The "Process Ancestor" search term has been deprecated.
- Central will now respond with a 421 Misdirected Request status code to requests where the ServerName sent via TLS SNI
  does not match the `:authority` (`Host`) header. This feature can be turned off by setting the environment variable
  `ROX_ALLOW_MISDIRECTED_REQUESTS=true`.
- Registry integrations for ECR are now auto-generated if the cluster's cloud provider is AWS, and the nodes' Instance IAM Role has policies granting access to ECR.  Customers can turn this feature off by disabling the EC2 instance metadata service in their nodes.
- A new default policy added to detect Spring Cloud Function RCE vulnerability (CVE-2022-22963) and Spring Framework Spring4Shell RCE vulnerability (CVE-2022-22965).
- Fixed permissions checks in the UI that prevented users with certain limited permissions from creating report configurations.
- ROX-8957: A new default policy added to detect missing ingress NetworkPolicy associated with deployments. The policy is disabled by default.
  - Two new policy criteria were added to alert on missing ingress or egress NetworkPolicy associations. 
- ROX-8789: Change operator catalog format from deprecated SQLite database format to new file-based format.
- ROX-8331: Increase the front-end limit on rendered nodes in the Network Graph from 1100 to 2000
- ROX-9792: Introduced central limit of 2000 nodes in a Network Graph to avoid out-of-memory crashes
- ROX-9946: Fixed default permissions for the default Vuln Reporter role to exclude the modify permission on notifiers, since it is not needed for report creation.
- Added AllowPrivilegeEscalation as a new policy criteria.
- ROX-10038: Removed limit of 10 inclusions and 10 exclusions from policy form
- ROX-10090: Made the username and password optional on the Artifactory integration form
- ROX-10217: Remove format validation from the URL field of the generic webhook integration form
- ROX-9435: Updated dryrun API to generate preview violations for disabled policies
- Support for security policies that do not have a policyVersion or have versions prior to 1.1 will be removed. If you have externally stored older policies, they cannot be imported.
- ROX-10021: RHCOS node support is dropped until major improvements are made in ROX-8944.
  - The UI shows the node scanning notes in the same manner as image scanning notes.
- ROX-10097: Updated the base for the docs image from `nginx-118:1-46` to `nginx-120:latest`.
- ROX-10666: `FROM` option will be deprecated from `Disallowed Dockerfile line` policy field and removed in a future release. Any policies containing `Disallowed dockerfile line` policy field with `FROM` option must be updated to remove those policy sections. For more information, please refer "Known Issues" section in Red-Hat ACS 3.69 release notes.
- ROX-10270: The `RenamePolicyCategory` and `DeletePolicyCategory` methods in the
`v1/policycategories` endpoint have been deprecated, and will be removed in future releases. 
  - For questions about this change, please contact the Red Hat support team at support@redhat.com.
- ROX-10018: The policy `OpenShift: Kubeadmin Secret Accessed` will no longer trigger if the request was from the default OpenShift `oauth-apiserver-sa` service account, because this is an expected access pattern for the OpenShift apiserver.
- Violation tags and process tags are deprecated, and will be removed in version 3.72.0.
- Users who do not want to include the RBAC factor in risk calculation can set
  the "ROX_INCLUDE_RBAC_IN_RISK" environment variable to "false" in the Central deployment spec.
- Kubernetes' PodSecurityPolicy API is deprecated which is why installation of PodSecurityPolicies will be disabled with version 3.71.0.

## [69.1]

- A version of Scanner and ScannerDB will be installed in each OpenShift cluster to support images stored in the OpenShift Internal Image Registry.
  - The images are "slimmed" down versions of Scanner and ScannerDB
    - scanner-slim and scanner-db-slim
  - They require the same resources as the normal Scanner and ScannerDB.

## [69.0]

- `collector` image with `-slim` in the image tag is no longer published (`collector-slim` with suffix in the image name will continue to be published).
- `collector-rhel`, `main-rhel`, `scanner-rhel`, and `scanner-db-rhel` images are not published any more. These images were identical to non-rhel ones since version 3.66.
- Increased default Scanner memory limit from 3000 MiB to 4GiB.
- API changes/deprecations:
  - `GetKernelSupportAvailable (GET /v1/clusters-env/kernel-support-available)` is deprecated, use `GetClusterDefaultValues (GET /v1/cluster-defaults)` instead.
  - The following features have been deprecated and will be removed in version 3.71.0:
    - The external authorization plugin for scoped access control will be removed. Please use the existing in-product scoped access control.
    - The Anchore, Tenable, and Docker Trusted Registry integrations will be removed. Please use the ACS Scanner instead as it is more widely supported.
    - Alert and process comments will be removed.
  - `CreateRole` and `UpdateRole` in `/v1/roles/`: `role.access_scope_id` empty value is deprecated, will be set to the unrestricted access scope ID (`io.stackrox.authz.accessscope.unrestricted`) during the adoption period.
  - API endpoint `/api/helm/cluster/add` was deleted as not being used in the product.
- Improved accuracy of active component and vulnerability and presented it with higher confidence.
  - Analyzed dependencies between OS components and detected derived active components.
  - Added `Active` state to list of components and list of vulnerabilities under Vulnerability Management within the scope of a specific deployment.
  - Added `Inactive` state: the component or vulnerability was not run in the specific deployment.
  - Added image scope so that the Active State can be determined in the scope of a deployment for a specific image.
- The default gRPC port in Scanner's config map is changed to 8443, as that is what Scanner has actually been defaulting to this whole time.
  - Note: Scanner had been ignoring the default `httpsPort` and `grpcPort` in its config map, as Scanner expected `HTTPSPort` and `GRPCPort` (and `MetricsPort`, if ever specified).
- Scanner now supports Alpine 3.15.
- Scanner now identifies busybox as a base OS.
  - It does *not* find vulnerabilities nor packages, though. It solely identifies busybox as a base OS.
- CVEs in Ubuntu images will no longer link to http://people.ubuntu.com/~ubuntu-security/cve/<CVE>. Now it links to https://ubuntu.com/security/<CVE>.
- Setting ROX_DISABLE_AUTOGENERATED_REGISTRIES environment variable to true will ignore all new registry integrations from Sensors
- Vulnerability snoozing and un-snoozing will not impact image and component risk. Furthermore, it will not impact `Image Vulnerabilities` risk factor for deployments.
- In 3.70, support for security policies that do not have a policyVersion will be removed. Therefore, if you have externally stored older policies (without policyVersion or version prior to 1.1), you must convert them to use policyVersion 1.1. To do this, import the old policies into RHACS and then export them again. You can check the policyVersion field for your stored policies to identify if they need conversion.
- Vulnerability Risk Assessment: Deferral update requests that are in pending state can now be canceled.

## [68.0]

- AWS ECR integration supports AssumeRole authentication.
- The default policy to detect Log4Shell vulnerability has been updated to also detect CVE-2021-45046 and the remediation has been updated to reflect the latest guidance by the Apache Logging security team.
- Prior to this release, CVEs could be snoozed using global write access on `Images`. Starting this release, requests to snooze CVEs can be created only using `VulnerabilityManagementRequests` global write access and requests can be approved only using `VulnerabilityManagementApprovals` global write access. Roles with write access on `Images`, created prior to this release, are provided with both the newly added permissions. We recommend updating the roles to only include the least amount of resources required for each role. All new roles must be explicitly supplied with `VulnerabilityManagementRequests` and/or `VulnerabilityManagementApprovals` permissions in order to use CVE snoozing functionality.
- Editing the cluster configuration in the UI is now disabled for Helm-based installations.
- For `roxctl helm output` and `roxctl central generate` added a new flag `--image-defaults` that allows selecting the default registry from which container images will be taken for deploying central and scanner.
- For `roxctl helm output` deprecated flag `--rhacs` in favor of `--image-defaults=rhacs` (using `--rhacs` with `--image-defaults` results in an error).
- Default behavior of `roxctl helm output` results now in using container images from `registry.redhat.io` instead of `stackrox.io`.
- By default, notifications will be sent for every runtime policy violation instead of only the first encountered violation. If this is undesired, setting an environment variable `NOTIFY_EVERY_RUNTIME_EVENT` to `false` will restore the previous behavior. Please note that the environment variable will be removed in a future release, so please notify the ACS team if you have a valid use case.
- Certain ACS images were moved to new repositories:
  - main: from `registry.redhat.io/rh-acs/main` to `registry.redhat.io/advanced-cluster-security/rhacs-main-rhel8`
  - collector: from `registry.redhat.io/rh-acs/collector` (with `-latest` tag) to `registry.redhat.io/advanced-cluster-security/rhacs-collector-rhel8`
  - collector (slim): from `registry.redhat.io/rh-acs/collector` (with `-slim` tag) to `registry.redhat.io/advanced-cluster-security/rhacs-collector-slim-rhel8`
  - scanner: from `registry.redhat.io/rh-acs/scanner` to `registry.redhat.io/advanced-cluster-security/rhacs-scanner-rhel8`
  - scanner-db: from `registry.redhat.io/rh-acs/scanner-db` to `registry.redhat.io/advanced-cluster-security/rhacs-scanner-db-rhel8`
- Tags of `scanner`, `scanner-db`, and `collector` (including slim variant) images are now identical to the tag of `main` image (same as product version) for the released images. For example, a scanner image for ACS 3.68.0 is now identified as following `registry.redhat.io/advanced-cluster-security/rhacs-scanner-rhel8:3.68.0` and `stackrox.io/scanner:3.68.0`. Please make sure you follow this versioning scheme when upgrading manually. This scheme will be used for all future releases.
- Collector Slim image name and tag have changed. Now the `-slim` is not part of the image tag but part of the image name. This means that Collector Slim image for the release 3.68.0 is identified as `registry.redhat.io/advanced-cluster-security/rhacs-collector-slim-rhel8:3.68.0` and `collector.stackrox.io/collector-slim:3.68.0`.

## [67.2]

- A new default policy to detect Log4Shell vulnerability (CVE-2021-44228) has been added.

## [67.0]

- When the environment variable `ROX_NETWORK_ACCESS_LOG` for Central is enabled, the logs will now contain the request URI and `X-Forwarded-For` header values.
  Note: The network access logging feature was introduced in 51.0 and when enabled will cause noisy logging, and hence should be turned on
  only for the purpose of debugging network connectivity issues.
- Scanner container image `uid:gid` changed to `65534:65534` (user nobody).
- A new default Role called `Scope Manager` has been introduced, to be used to provide users the minimal set of
  privileges required to create and modify access scopes for the purpose of configuring access control or use in vulnerability reporting.
- The Compliance Operator integration now supports TailoredProfiles.
- Presence of `microdnf` (presence in the image and process execution) is treated as violation of policies `Red Hat Package Manager in Image` and `Red Hat Package Manager Execution` respectively.
- Central is now the only source for Scanner vulnerability updates.
  - Central, instead of Scanner, now queries definitions.stackrox.io in online-mode (determined based on `ROX_OFFLINE_MODE`).
  - `ROX_SCANNER_VULN_UPDATE_INTERVAL` determines the frequency Central should query definitions.stackrox.io, in online-mode. It is defaulted to 5 minutes.
  - Scanner's ConfigMap still has an `updater.interval` field for its own updating frequency, but it no longer has `updater.fetchFromCentral`.
- Users may upload Scanner vulnerability dumps even when we are not in "offline-mode".
  - If we are in online-mode, this vuln dump is used over the Scanner's requested one if it is more recent.
  - K8s and Istio vulns manually uploaded in online-mode are ignored. This is just for Scanner definitions.
- Roxctl's `image scan | image check | deployment check` commands received a usability overhaul.
  This includes introducing output format's `table, csv, json` for each command.
  Note: the `csv` and `json` output formats contain **breaking changes**, the old formats are kept as default but marked as deprecated.
  Ensure that you switch to the new formats in a timely manner.
- In policy exclusions, the deployment name can now be a regex. Earlier, it was an exact string match.

- Behaviour change: The built-in `None` role is no longer taken into account when determining the roles for a user. Therefore, users with only the `None`
  role will be logged out and not be able to log in, as a valid user must have some role assigned. Logout and login prevention are materialized with HTTP
  status 401 `Unauthorized` and error message reporting the lack of valid role.

## [66.0]

- Default system policies `DockerHub NGINX 1.10`, `Shellshock: Multiple CVEs`, and `Heartbleed: CVE-2014-0160` have been deprecated.
- Default system policy deletion is prohibited in fresh installations of 65 or greater. If the initial installation
  was done in a version lower than 65, then default policies can be deleted even after an upgrade to 65 or greater.
- `Analyst` permission set and corresponding role will no longer have `DebugLogs` permission. The only default role with this permission will be `Admin` role.
- The "Mount Docker Socket" policy has been renamed to "Mount Container Runtime Socket" and will now also detect if a deployment
  mounts the CRI-O socket for both Kubernetes and OpenShift.
- The policy "Docker CIS 4.4: Ensure images are scanned and rebuilt to include security patches" is now disabled by default
- Alpine-based images are now deprecated and all images will be based on UBI. main-rhel will continue to be pushed for consistency.
- Added `central.tolerations`, `scanner.tolerations` and `scanner.dbTolerations` to the `stackrox-central-services` Helm chart
- Added `sensor.tolerations` and `admission-control.tolerations` to the `stackrox-secured-cluster-services` Helm chart
- Operator now supports `tolerations`  for `Central` and `SecuredCluster`
- Operator now supports disabling the admin password generation by setting Central's option `adminPasswordGenerationDisabled` to `true`.
- Roxctl now supports shell completion for bash, zsh, fish and powershell
- Added `roxctl central debug authz-trace` command. It streams built-in authorizer traces for all incoming requests.
- Operator defaults changed for `SecuredCluster` fields `spec.admissionControl.listenOnCreates` and `spec.admissionControl.listenOnUpdates`
  from `false` to `true`. This should not affect these settings in existing `SecuredCluster` resource instances
  where the previous default had already been applied at instance creation (this typically happens when creating the resource from the OpenShift console).
  In some circumstances (for example if the instance was created without a `spec.admissionControl` section from the CLI),
  the default might not have been applied: a symptom of this is that the fields are not shown when printing the object.
  In these cases this update will change the behaviour of admission controller.
- Scanner no longer supports Oracle Linux
- Added component `Active` state to individual component and list of components under Vulnerability Management within the scope of a specific deployment. The Active state can be:
  - `Undetermined`: the component is not detected to be run in the specific deployment.
  - `Active`: the component was run in the specific deployment.

## [65.0]
- Starting 65.0, default system policies' criteria fields are read-only. This applies to all default system policies
  included in fresh install of 65.0 and later, and new default system policies added since 65.0. Policy criteria fields
  for user-defined policies, created through 'New' and 'Clone' operation, will continue to be editable.
- Newly added MITRE ATT&CK policy section is read-only for default system policies. MITRE ATT&CK section for user-defined
  policies, created through 'New' and 'Clone' operation, will continue to be editable.
- Alert titles for the PagerDuty, Slack, Microsoft Teams, JIRA and email notifiers now contain the cluster and policy names
  in addition to the deployment or image name if it exists.
- PagerDuty alerts for violations now include the full alert JSON as a custom detail.
- Message attribute keys for audit log based violation messages shortened to be more readable
- Cluster internal endpoints set to `*.svc` to be respected by OpenShift's cluster wide `noProxy` configuration
  - `sensor.stackrox` changed to `sensor.stackrox.svc`
  - `central.stackrox` changed to `central.stackrox.svc`
  - `scanner.stackrox` changed to `scanner.stackrox.svc`
  - `scanner-db.stackrox` changed to `scanner-db.stackrox.svc`
- Increased Operator memory requests from 80 MiB to 200 MiB and memory limits from 300 MiB to 1 GiB. The latter is to prevent operator restarts due to OOM on certain deployments.
- Customer advisory: Default system policies `DockerHub NGINX 1.10`, `Shellshock: Multiple CVEs`, and `Heartbleed: CVE-2014-0160` will be deprecated starting release `66.0`.

## [64.1]

- Cluster internal endpoints set to `*.svc` to be respected by OpenShift's cluster wide `noProxy` configuration
  - `sensor.stackrox` changed to `sensor.stackrox.svc`
  - `central.stackrox` changed to `central.stackrox.svc`
  - `scanner.stackrox` changed to `scanner.stackrox.svc`
  - `scanner-db.stackrox` changed to `scanner-db.stackrox.svc`
- Increased Operator memory requests from 80 MiB to 200 MiB and memory limits from 300 MiB to 1 GiB. The latter is to prevent operator restarts due to OOM on certain deployments.

## [64.0]

- Support for BadgerDB is being completely removed. Users running a version less than 48.0 will need to upgrade to 63.0
  prior to upgrading to 64.0. All backups taken prior to version 48.0 cannot be restored to 64.0 and newer.
- The `/v1/namespaces` endpoint now accepts pagination query parameters.
- Message attribute keys for audit log based violations changed to use capital case instead of lowercase in API response.
- On OpenShift, the names of all `SecurityContextConstraint` (SCC) resources are now prefixed with `stackrox-`.

## [63.1]

- Cluster internal endpoints set to `*.svc` to be respected by OpenShift's cluster wide `noProxy` configuration
  - `sensor.stackrox` changed to `sensor.stackrox.svc`
  - `central.stackrox` changed to `central.stackrox.svc`
  - `scanner.stackrox` changed to `scanner.stackrox.svc`
  - `scanner-db.stackrox` changed to `scanner-db.stackrox.svc`
- Increased Operator memory requests from 80 MiB to 200 MiB and memory limits from 300 MiB to 1 GiB. The latter is to prevent operator restarts due to OOM on certain deployments.

## [63.0]

- Clusters now can have labels.
- Role is now a combination of a permission set and an optional access scope.
- API changes/deprecations:
  - `AuthService(/v1/auth/status)`: `user_info.permissions.name` and `user_info.permissions.global_access` are
    deprecated, use `user_info.roles` instead.
  - `CreateRole(POST /v1/roles/{name})`, `UpdateRole(PUT /v1/roles/{name})`: specifying `resource_to_access` is
    disallowed, `permission_set_id` must be provided instead.
  - `GetRoles(GET /v1/roles)`, `GetRole(GET /v1/roles/{name})`: `resource_to_access` is never set, use
    `permission_set_id` instead.
  - In the GraphQL API, `Role { resourceToAccess: [Label!]! }` is deprecated, use
    `PermissionSet { resourceToAccess: [Label!]! }` instead.
  - In the GraphQL API, `Role { globalAccess: Access! }` is deprecated with no replacement intended.
- The operator now sets dynamic admission control settings (`enforceOnCreates`, `enforceOnUpdates`)
  based on `spec.admissionControl.listenOn*` in the `SecuredCluster` resource.

## [62.2]

- Cluster internal endpoints set to `*.svc` to be respected by OpenShift's cluster wide `noProxy` configuration
  - `sensor.stackrox` changed to `sensor.stackrox.svc`
  - `central.stackrox` changed to `central.stackrox.svc`
  - `scanner.stackrox` changed to `scanner.stackrox.svc`
  - `scanner-db.stackrox` changed to `scanner-db.stackrox.svc`
- Increased Operator memory requests from 80 MiB to 200 MiB and memory limits from 300 MiB to 1 GiB. The latter is to prevent operator restarts due to OOM on certain deployments.

## [62.1]

- Fixed RHSA-2021:2569, RHSA-2021:2574, RHSA-2021:2575, RHSA-2021:2717, RHBA-2021:2581 in RHEL images.

## [62.0]

- Scanner now supports alpine:edge and alpine:3.14.
- Scan results for alpine 3.2 - 3.7 were marked as stale before.
  It has since become clear that there are still updates to the secdb for these versions,
  so they are no longer marked stale.
- The `ROX_ALERT_RENOTIF_DEBOUNCE_DURATION` can be set to a duration (see https://golang.org/pkg/time/#ParseDuration
  for supported syntax), and if set, then duplicate notifications for deploy-time alerts for the same deployment-policy
  pair will not be sent if the previous alert was resolved more recently than the debounce duration.
- Scanner now supports alpine:edge.

## [61.0]

- `globalAccess` field in roles is no longer supported
- Policy matching on all fields has been made case-insensitive. For example, if you set "Volume Type" to "hostpath",
  that will match volumes that are "HostPath".
- Added the ability to make policies based on `Severity` (ROX-6639)
  - Added new default policy (disabled by default) for a `High` alert for fixable
    CVEs with severity at least Important (includes Important and Critical).
- roxctl image scan --format {csv,pretty} are now sorted by layer and severity
  instead of layer and CVSS.
- Image risk is now calculated using a score assigned to the Severity Rating,
  opposed to using the CVSS score. Severity Rating is a more accurate measure of
  a vulnerability's risk. (ROX-7133)

## [60.0]

- CVE Severity levels are now mapped to their respective Red Hat security ratings (https://access.redhat.com/security/updates/classification)
- StackRox Scanner passes Red Hat Scanner Certification
  - Images based on RHEL base images created after June 2020 will be scanned in a certified manner.
    - These images will say `rhel` as the OS instead of `centos`.
    - Language-related files like JAR (Java), egg-info (Python) will only be scanned if they are not provided by RPM.
      To determine if a file is provided by RPM, run `rpm -q --whatprovides <absolute filepath>` in the image.
  - Older RHEL-based images will be scanned the traditional way.
    - These images will continue to say `centos` is the base OS.
- StackRox Scanner now officially supports ubuntu:21.04 images

## [59.0]

- Added `GET /v1/centralhealth/upgradestatus` endpoint to support upgrade rollback.
- Scanner no longer supports RHEL/CentOS 5.
- Default value for `--json-fail-on-policy-violations` flag of `roxctl image check` changed
  from `false` to `true`.

## [58.1]

- A few CVSS3.1 scores for applicable vulnerabilities were miscalculated, but it has since been fixed.
- Fixed CVE-2021-20305, RHSA-2021:1206 in RHEL scanner images
- Fixed Java package scanning when the package has the word "agent"

## [58.0]

- The product no longer requires a license to run. Several license-related functionalities and flags
  have been removed from the product and related tooling, as well as from the Helm charts.
- Components now have `Fixed By` field that indicates the version that will fixes all the fixable vulnerabilities in the component.
  - Note:
    - It is supported only when StackRox Scanner is used.
    - It is not namespaced to distro.
- Added upgrade rollback function. By default, users may rollback to their previous version if upgrade fails before Central has started.
  After services started, users must explicitly specify the version they are rolling back to in central config `maintenance.forceRollbackVersion`.
- Added a `central.exposeMonitoring` option to the Central Services Helm chart, which, when set to `true`, allows exposing a `/metrics`
  endpoint on port 9090.


## [57.0]

- The published time for CVEs in RHEL and CentOS images is now populated correctly.
- Secured clusters deployed via Helm with `helmManaged` set to `false` can now be used with cluster init
  bundles, creating a new cluster within StackRox on-the-fly. Previously, `helmManaged=false` only worked
  with certificates that were specific to an existing cluster.
- `roxctl central generate openshift` and `roxctl sensor generate openshift` now accept an
  `--openshift-version` flag, which can be set to the major version (`3` or `4`) of the OpenShift platform
  to deploy on. By default, deployment files are generated in a compatibility mode that works on OpenShift
  3.11 as well as 4.x. When deploying to a cluster running a recent OpenShift version, set this flag to `4`
  in order to take advantage of features only supported on OpenShift 4.x.


## [56.0]
- Page titles now reflect the URL location of the user within the app in the browser tab and history.
- SAML authentication providers:
  - When using the "Dynamic configuration" option, the `IdP Metadata URL` can now specify a
    scheme of `https+insecure://` to instruct StackRox to skip TLS validation when fetching
    the metadata. It is **strongly** advised to limit the use of this to testing environments.
  - When using the "Static configuration" option, the `IdP Certificate(s) (PEM)` option now
    supports specifying multiple PEM-encoded certificates.
- When creating a new Role, Namespace and Node have been added to the default minimal access specification.
- Admission Control health status is now available as part of Cluster Health in System Health, and in the
in the Platform Configuration -> Clusters View.

- `roxctl image check` now has a `--json-fail-on-policy-violations` flag. Its current default value
   is `false` which preserves the legacy behavior of `--json` flag: the command does *not*
   exit with an error code, even if policy violations are present.

   This default value of `false` is also now deprecated and will change in three releases.
- New default policies:
  - Added default policies for Docker CIS checks
    - 4.1
    - 4.4
    - 4.7
    - 5.1
    - 5.7
    - 5.9
    - 5.15
    - 5.16
    - 5.19
    - 5.20
    - 5.21
- Splunk alert events send to HEC will no longer include policy description, remediation and rationale
 in order to allow for more violations underneath the HEC limit.
- The ROX_NETWORK_DETECTION_BASELINE_VIOLATION feature flag is now on by default: a deployment with network flows that
are outside of its network baseline can now raise violations
- New roxctl option for roxctl image check: --categories.  Specifying a comma separated list of categories will only run policies with categories in the specified list.

## [55.0]
- The `/v1/metadata` endpoint redacts version information from unauthenticated users.
- API changes/deprecations:
  - `/db/backup` is deprecated; please use `/api/extensions/backup` instead.
  - In the GraphQL API, `ProcessActivityEvent { whitelisted: Boolean! }` is deprecated, use
    `ProcessActivityEvent { inBaseline: Boolean! }` instead.
  - In the GraphQL schema, the type name `Policy { whitelists: [Whitelist]! }` changes to
    `Policy { whitelists: [Exclusion]! }` preserving the existing structure and field names.
  - In the GraphQL API, `Policy { whitelists: [Whitelist]! }` is deprecated, use
    `Policy { exclusions: [Whitelist]! }` instead.
  - `PolicyService(/v1/policies/*)`: in all affected responses, `Policy.whitelists` is now always empty, use
    `Policy.exclusions` instead. This is because the current policy version has been updated to "1.1" which deprecates
    the `Policy.whitelists` field. All previous policy versions are still accepted as input.
  - Deprecated `includeCertificates` flag in `/v1/externalbackups/*`. Certificates are included in central
    backups by default for both new and existing backup configs.
- Admission controller service will be deployed by default in new k8s and Openshift clusters.
The validating webhook configuration for exec and port forward events is not supported on and hence
will not be deployed on OpenShift clusters.
- `roxctl image check` now has a `--send-notifications` flag, which will send notifications for
  build time alerts to the notifiers configured in each violated policy.
- `roxctl central db backup` is deprecated; please use `roxctl central backup` instead.
- The following  roxctl flags have been deprecated for the command `sensor generate`:
  - `--create-admission-controller` (replaced by `--admission-controller-listen-on-creates`)
  - `--admission-controller-enabled` (replaced by `--admission-controller-enforce-on-creates`)
- Added retry flags to `roxctl image scan`, `roxctl image check`, and `roxctl deployment check`:
  - Introduced two new flags, `--retries` and `--retry-delay`, that change how the commands deal with errors
  - `--retries 3 --retry-delay 2` will retry the command three times on failure with two seconds delay between retries
  - As the default value for `retries` is 0, the behaviour of the commands is unchanged if the flag is not used
- Added a new flag `--admission-controller-listen-on-events` to `roxctl sensor generate k8s` and
`roxctl sensor generate openshift`, that controls the deployment of the admission controller webhook which
listens on Kubernetes events like exec and portforward. Default value is `true` for `roxctl sensor generate k8s`
and false for `roxctl sensor generate openshift`.

## [54.0]
- Added option to backup certificates for central.
- API changes/deprecations:
  - `ProcessWhitelistService(/v1/processwhitelists/*)`: all `processwhitelists/*` endpoints are deprecated, use
    `processbaselines/*` instead.
  - `ResolveAlert(/v1/alerts/{id}/resolve)`: `whitelist` is deprecated, use `add_to_baseline` instead.
  - In the `ListDeploymentsWithProcessInfo(/v1/deploymentswithprocessinfo)` response, `deployments.whitelist_statuses`
    is deprecated, use `deployments.baseline_statuses` instead.
  - `ROX_WHITELIST_GENERATION_DURATION` environment variable is deprecated, use `ROX_BASELINE_GENERATION_DURATION`
    instead.

## [53.0]
- [Security Advisory] Scanner was not validating Central client certificates allowing for intra-cluster unauthenticated users
  to initiate or get scans. This only affects environments without NetworkPolicy enforcement.

## [52.0]
- Added ContainerName as one of the policy criteria
- Added support for ubuntu:20.10 in Scanner.
- Added support for distroless images in Scanner.

## [51.1]
- UI: fix a browser crash when a port's exposure type is UNSET in the Deployment Details of a Risk side panel (ROX-5864)

## [51.0]
- UI: remove "phantom" turndown triangle on Network Flows table rows that have only one bidirectional connection on the same port and protocol
- UI: fix pagination in Vuln Mmgt so that filtering a list by searching will reset the page number to 1 (ROX-5751)
- A new environment variable for Central ROX_NETWORK_ACCESS_LOG, defaulted to false, is available.
When set to true, each network request to Central (via API, UI) is logged in the Central logs.
Note: When turned on, this environment variable will cause noisy logging, and hence should be turned on only for the
purpose of debugging network connectivity issues. Once network connectivity is established, we should advise
to immediately set this to false to stop logging.
- Added Namespace as one of the policy criteria
- UI: Display full height of Vulnerability Management side panel in Safari (ROX-5771)
- Added a `--force-http1` option to `roxctl` that will cause HTTP/2 to be avoided for all outgoing requests.
  This can be used in case of connectivity issues which are suspected to be due to an ingress or proxy.
- UI: Fix bug where some policy criteria values, with equal signs, are parsed incorrectly (ROX-5767)

## [50.0]
- UI: Do not display incomplete process status when Sensor Upgrade is up to date (ROX-5579)
- The minimum number of replicas for the Scanner Horizontal Pod Autoscaler has been set to 2 for better availability.
- The ROX_CONTINUE_UNKNOWN_OS feature flag is on by default in Scanner
  - Scans done by StackRox Scanner on images whose OS cannot be determined will no longer fail if the image also has feature components. Instead, they will continue and give partial scan results.
    - An example is the `fedora:32` image
- The default resource limit for Central has been changed to 4 cores. Please see the resource sizing guidelines in the help documentation for
  finer-grained settings.
- A new policy criteria on "Service Account" has been added which runs policy evaluation against the deployment's service account name.
- Use Red Hat CVSS scores instead of NVD for `rhel` and `centos` based images scanned by StackRox Scanner.
  - CVSS3 is used if it exists otherwise CVSS2 is used.
- Added support for .NET Core runtime CVEs (data from NVD).
  - This affects images with .NET Core and/or ASP.NET Core runtime(s) installed
- UI: Update the Network Graph when a different cluster is selected (ROX-5662)
- Support sub-CVEs for RHEAs and RHBAs as well as RHSAs for rhel/centos-based images.
  - Though it is not specified, it is possible RHEAs and RHBAs to have associated CVEs.
- The default policy "Required Label: Email" has been deprecated starting release 50.0.

## [49.0]
- OIDC authentication providers: added support for two rarely-needed configuration options:
  - The `Issuer` can now be prefixed with `https+insecure://` to instruct StackRox to skip TLS validation
    when talking to the provider endpoints. It is **strongly** advised to limit the use of this to testing
    environments.
  - The `Issuer` can now contain a querystring (`?key1=value1&key2=value2`), which will be appended as-is
    to the authorization endpoint. This can be used to customize the provider's login screen, e.g., by
    optimizing the GSuite login screen to a specific hosted domain via the
    [`hd` parameter](https://developers.google.com/identity/protocols/oauth2/openid-connect#hd-param),
    or to pre-select an authentication method in PingFederate via the
    [`pfidpadapterid` parameter](https://docs.pingidentity.com/bundle/pingfederate-93/page/nfr1564003024683.html).
- In `GetImage(/v1/images/{id})` response, the `vulns` field `discoveredAt` is replaced by `firstSystemOccurrence` starting release 49.0. This field represents the first time the CVE was ever discovered in the system.
- In `GetImage(/v1/images/{id})` response, a new field `firstImageOccurrence` is added to `vulns` which represents the first time a CVE was discovered in respective image.
- The default for the `--create-upgrader-sa` flag has changed to `true` in both the `roxctl sensor generate` and the
  `roxctl sensor get-bundle` commands. In order to restore the old behavior, you need to explicitly specify
  `--create-upgrader-sa=false`.
- UI: Hovering over a node in the Network Graph will show that node's listening ports (ROX-5469)
- Fixed an issue on the API docs page where the left menu panel no longer scrolled independently of the main content.
- UI: Added `Scanner` to the image single page in Vuln Mgmt (ROX-5289)
- In `v1/clusters` response, `status.lastContact` has been deprecated, hence use `healthStatus.lastContact` instead.
- UI: Disable the Next button when required fields are empty in the Cluster form (ROX-5519)
- `roxctl` can now be instructed to generate YAML files with support for Istio-enabled clusters, via the
  `--istio-support=<istio version>` flag. Istio versions in the range of 1.0-1.7 are supported. The flag is available
   for the commands `roxctl central generate`, `roxctl scanner generate`, `roxctl sensor generate`, and
   `roxctl sensor get-bundle`. The interactive installer (`roxctl central generate interactive`) will also prompt for
   this configuration option.
- Support for enforcing policies on DeploymentConfig resources in Openshift.
- The following deprecated roxctl flags have been removed for the command `sensor generate`:
  - `--admission-controller` (replaced by `--create-admission-controller`)
  - `--image` (replaced by `--main-image-repository`)
  - `--collector-image` (replaced by `--collector-image-repository`
  - `--runtime` (`--collection-method` is to be used instead)
  - `--monitoring-endpoint`

## [48.0]
- UI: Hovering over a namespace edge in the Network Graph will show the ports and protocols for it's connections (ROX-5228).
- UI: Hovering over a namespace edge in the Network Graph will show a summary of the directionality of it's connections (ROX-5215)
- UI: Hovering over a node edge in the Network Graph will show the ports and protocols for it's connection (ROX-5227)
- UI: Platform Configuration > Clusters  (ROX-5317)
  - add 'Cloud Provider' column
  - remove 'Current Sensor version' column
  - replace 'Upgrade status' column with 'Sensor Upgrade' and add tooltip which displays 'Sensor version' and 'Central version'
  - display cells in 'Sensor Upgrade' columns with same style as adjacent new Cluster Health columns
- UI: Added a toggle in the Network Policy Simulator in Network Graph to exclude ports and protocols (ROX-5248).
- UI: Platform Configuration > Clusters: Make CertificateExpiration look similar to recently improved Sensor Upgrade style and future cluster health style (ROX-5398)
  - Red X icon at left of phrase for less than 7 days (for example, in 59 minutes, in 7 hours, in 6 days on Friday)
  - Yellow triangle icon at left of phrase for less than 30 days (for example, in 29 days on 7/31/2020)
  - Green check icon at left of other phrases (for example, in 1 month on 7/31/2020, in 2 months)
- UI and strings from API: Replace term 'whitelist' with 'excluded scope' in policy context, and 'baseline' in process context (ROX-5315, ROX-5316)
- UI: Deployment Details in the Violations Side Panel now shows full deployment data if available. If not, a message will appear explaining that the deployment no longer exists.
- UI: When selecting a deployment in the Network Graph, the Network Flows table will now show some additional information: traffic, deployment name,
  namespace name, ports, protocols, connection type (ROX-5219)
- In `v1/clusters` response, `healthStatus.lastContact` field is added that represents last time sensor health was probed (aka last sensor contact). `status.lastContact` will be deprecated starting release 49.0, hence use `healthStatus.lastContact` instead.
- When attempting to scan an image, we now send back error messages under any of the following conditions:
  - no registries integrated
  - no matching registries found
  - no scanners integrated
- In `GetImage(/v1/images/{id})` response, the `vulns` field `discoveredAt` will be replaced by `firstSystemOccurrence` starting release 49.0. This field represents the first time the CVE was ever discovered in the system.

## [47.0]
- Configuration Management tables (except for Controls and Policies) are now paginated through the API, rather than loading all rows into the browser, for better performance in large environments (ROX-5067).
- Added a global flag `--token-file` to roxctl causing an API token to be read from the specified file (ROX-2319).
- Added strict validation for env var policies such that policies with non-raw sources must not specify expected values (ROX-5208). This change introduces a breaking adjustment to the `/v1.PolicyService/PostPolicy` RPC, with existing REST clients remaining unaffected.
- Emit warning if the default value for flag `--create-updater-sa` is used in roxctl (ROX-5264).
- New parameter `default` for flag `--collection-method`.
- UI: Omit Cluster column from Deployments list when entity context includes Namespace (ROX-5207)
- The help output of `roxctl` commands mentions implicit defaults for optional flags.
- UI: Fix a regression, where CSVs for a Compliance standard, or for a Cluster viewed in Compliance, were not scoped to the particular filter (ROX-5179)
- The following command line flags of `roxctl` have been deprecated:
  - Flag `--image` for `roxctl sensor generate`. Use `--main-image-repository` instead.
  - Flag `--collector-image` for `roxctl sensor generate`. Use `--collector-image-repository` instead.
  - Flag `--admission-controller` for `roxctl sensor generate k8s`. Use `--create-admission-controller` instead.

  The old flags are currently still supported but they are scheduled for removal in a future version of `roxctl`.

- UI: Added arrows to indicate directionality (ingress/egress) for Network Graph connections between deployments (ROX-5211).
- UI: Added `Image OS` to the image list and image single page in Vuln Mgmt (ROX-4083).
- Added the ability to make policies based on `Image OS` (ROX-4083).
- roxctl image scan and /v1/image/<image id> no longer return snoozed CVEs as a part of their output. The `include-snoozed` command line parameter
  and the `includeSnoozed` query parameter respectively can be used to include all CVEs.
- The 'namespace.metadata.stackrox.io/id' label is now removed in order to better support Terraform cluster management.
- UI: Hovering over a deployment in the Network Graph will show the ports and protocols for it's ingress/egress network flows (ROX-5226).
- Adding the annotation `auto-upgrade.stackrox.io/preserve-resources=true` on the `sensor` deployment and the `collector` daemonset
  will cause the auto-upgrader to preserve any overridden resource requests and limits whenever an upgrade is performed.

## [46.0]
- Added the following REST APIs:
  - PATCH `/v1/notifiers/{id}` modifies a given notifier, with optional stored credential reconciliation.
  - POST `/v1/notifiers/test/updated` checks if the given notifier is correctly configured, with optional stored credential reconciliation.
  - PATCH `/v1/scopedaccessctrl/config/{id}` modifies a given scoped access control plugin, with optional stored credential reconciliation.
  - POST `/v1/scopedaccessctrl/test/updated` checks if the given scoped access control plugin is correctly configured, with optional stored credential reconciliation
  - PATCH `/v1/externalbackups/{id}` modifies a given external backup, with optional stored credential reconciliation.
  - POST `/v1/externalbackups/test/updated` checks if the given external backup is correctly configured, with optional stored credential reconciliation.
- UI: Reset page to 1 when sort fields change (ROX-4267)
- Add a tcp prefix to the spec.Ports.name for the scanner-db service. Istio uses this name for protocol detection.
- Customer advisory: The default policy "Required Label: Email" will be deprecated starting release 48.0
- RocksDB is set as the default DB and completely replaces BadgerDB and replaces a majority of BoltDB. This should make Central significantly more performant.
  Users may see slowness during startup on initial upgrade as the data is being migrated.
- Added UI to show cluster credential expiry in the cluster page (ROX-5034).
- UI: The deployment event timeline should now visibly group events that would otherwise overlap. The grouped events will show a number in the top right that
  indicates how many events were grouped. Clicking on the icon will show an interactive tooltip that displays information for each event in a scrollable manner (ROX-5190).
- UI: Under Vulnerability Management, update "Deployment Count" column on policy entity list pages to show failing deployments count instead of all applicable deployments count (ROX-5176).
- StackRox now supports the Garden Linux operating system. Previous, collector pods would enter a crash loop when deployed on
  Garden Linux nodes.

## [45.0]
- Default policies that have been excluded for the kube-system namespace, have now been additionally excluded for the istio-system namespace.
- Default integration added for public Microsoft Container Registry
- Heads up advisory on `roxctl sensor generate k8s` command option changes slated for release in 47.0:
  1. `admission-controller` option will be renamed to `create-admission-controller`
  2. The default for `create-upgrader-sa` will change to `true`
  3. The default for `collection-method` will change to `KERNEL_MODULE`
  4. Deprecated option `runtime` will be removed
  6. `image` option  will be renamed to `main-image-repository`
  7. `collector-image` option will be renamed to `collector-image-repository`
  8. `monitoring-endpoint` option, which has already been deprecated, will be removed
- Add CVE Type to CVE list and overview pages (ROX-4482)
- UI: Open API Reference in current Web UI browser tab instead of a new tab and replace Help Center popup menu with two half-height links in left navigation for API Reference and Help Center (ROX-2200)
- UI: Move Images link on VM dashboard out of Applications menu, and into tile like Policies and CVEs link (ROX-5052)
- UI: Add Disable TLS Certificate Validation (Insecure) toggle for JFrog Artifactory registry in Platform Configuration > Integrations > New Integration (ROX-5031)
- UI: Add Disable TLS Certificate Validation (Insecure) toggle for Anchore Scanner, CoreOS Clair (Scanner), and Quay.io (Registry + Scanner) in Platform Configuration > Integrations > New Integration (ROX-5084)
- Added the ability to make secret creation for sensor, collector and admission controller optional when deploying using Helm charts.
- Added native Google Cloud Storage (GCS) external backup. This should now be the preferred way to backup to GCS as opposed to using the S3 integration because
  S3 upload to GCS is incompatible with large databases.
- The Central and Migrator binaries are now compiled without AVX2 instructions, which fixes an Illegal Instruction issue
  on older CPUs. SSE4.2 instructions are still used, which mean the lowest supported Intel processor is SandyBridge (2011) and the lowest
  supported AMD processor is BullDozer (2011).

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
- UI: In Platform Configuration > Roles and Permissions > Add New Role form: 1. disable the Save button when required Role Name is empty; 2. display `(required)` at the right of the Role Name label with gold warning color if the input box is empty, otherwise with gray color (ROX-1808)
- UI: Increase timeout for Axios-fetch for GraphQL endpoint, to allow Vuln Mgmt pages in large-scale customer environments to load (ROX-4989)

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
- Policy excluded scopes are now shown in the UI. Previously, we only showed excluded deployment names, and not the entire structure that was
  actually in the policy object. This means that users can now exclude by cluster, namespace and labels using the UI.
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
- Scopes now include support for Regex on the namespace and label fields including both Policy Scope and Exclusion Scope.
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
