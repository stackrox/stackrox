# Changelog
All notable changes to this project that require documentation updates will be documented in this file.

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
