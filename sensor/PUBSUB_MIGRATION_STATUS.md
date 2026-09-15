# Sensor PubSub Migration Status

Tracking table for the migration of sensor component communication to PubSub.

**Epic**: [ROX-32854](https://redhat.atlassian.net/browse/ROX-32854)

## Tag System

**Component**: `[detector]` `[eventpipeline]` `[resolver]` `[pi]` `[netflows]` `[fs]` `[compliance]` `[co]` `[ac]` `[enforcer]` `[telemetry]` `[enhancer]` `[certrefresh]` `[cluster-status]` `[netpol]` `[vm]`

**Direction**: `[central-in]` `[central-out]` `[inner]` `[inter]`

**Work type**: `[optional]` `[infrastructure]` `[cleanup]`

**Status legend**: :white_check_mark: Done | :construction: In Progress | :black_square_button: Not Started

---

## Infrastructure

| Component | Flow | Tags | Mechanism | Status | Jira | PR |
|-----------|------|------|-----------|:------:|------|----|
| Central Sender | Create ResponsesC PubSub bridge component | `[infrastructure][central-out]` | N/A (new) | :construction: | [ROX-35640](https://redhat.atlassian.net/browse/ROX-35640) | [#21650](https://github.com/stackrox/stackrox/pull/21650) |
| Central Sender | Remove ResponsesC bridge and refactor centralSenderImpl | `[infrastructure][cleanup][central-out]` | N/A | :black_square_button: | [ROX-35641](https://redhat.atlassian.net/browse/ROX-35641) | |
| Sensor Framework | Migrate `Notify(SensorComponentEvent)` to PubSub | `[infrastructure][inner]` | Direct method calls on each component | :black_square_button: | [ROX-35642](https://redhat.atlassian.net/browse/ROX-35642) | |
| Sensor Framework | Migrate `internalmessage.MessageSubscriber` to PubSub | `[infrastructure][inner]` | Lightweight pubsub (`SoftRestart`, `ResourceSyncFinished`) | :construction: | [ROX-35643](https://redhat.atlassian.net/browse/ROX-35643) | [#21316](https://github.com/stackrox/stackrox/pull/21316) |
| PubSub Lane/Consumer | Deduping queue logic in lane/consumer | `[infrastructure][inner]` | N/A (new) | :black_square_button: | [ROX-35644](https://redhat.atlassian.net/browse/ROX-35644) | |

## Detector

| Component | Flow | Tags | Mechanism | Status | Jira | PR |
|-----------|------|------|-----------|:------:|------|----|
| Detector | `indicatorsQueue` (PI pipeline) | `[detector][inner]` | PubSub `DetectorProcessIndicatorTopic` | :white_check_mark: | [ROX-34626](https://redhat.atlassian.net/browse/ROX-34626) | [#20436](https://github.com/stackrox/stackrox/pull/20436) |
| Detector | `networkFlowsQueue` (NF pipeline) | `[detector][inner]` | PubSub `DetectorNetworkFlowTopic` | :white_check_mark: | [ROX-34626](https://redhat.atlassian.net/browse/ROX-34626) | [#21009](https://github.com/stackrox/stackrox/pull/21009) |
| Detector | `fileAccessQueue` (file access pipeline) | `[detector][inner]` | PubSub `DetectorFileAccessTopic` | :white_check_mark: | [ROX-34626](https://redhat.atlassian.net/browse/ROX-34626) | [#21010](https://github.com/stackrox/stackrox/pull/21010) |
| Detector | `auditEventsChan` (audit log pipeline) | `[detector][inter]` | PubSub `DetectorAuditLogTopic` | :white_check_mark: | [ROX-34626](https://redhat.atlassian.net/browse/ROX-34626) | [#21011](https://github.com/stackrox/stackrox/pull/21011) |
| Detector | `deploymentsQueue` (deployment pipeline) | `[detector][inner]` | PubSub `DetectorDeploymentTopic` | :white_check_mark: | [ROX-34626](https://redhat.atlassian.net/browse/ROX-34626) | [#21115](https://github.com/stackrox/stackrox/pull/21115) |
| Detector | `scanResultChan` (enricher → detector) | `[detector][inner]` | PubSub `DetectorScanResultTopic` | :white_check_mark: | [ROX-34626](https://redhat.atlassian.net/browse/ROX-34626) | [#21174](https://github.com/stackrox/stackrox/pull/21174) |
| Detector | `deploymentAlertOutputChan` (serialization) | `[detector][inner]` | PubSub `DetectorDeployAlertOutputTopic` | :white_check_mark: | [ROX-34626](https://redhat.atlassian.net/browse/ROX-34626) | [#21175](https://github.com/stackrox/stackrox/pull/21175) |
| Detector | `output` → `ResponsesC()` | `[detector][central-out]` | Unbuffered channel | :black_square_button: | [ROX-35645](https://redhat.atlassian.net/browse/ROX-35645) | |

## Event Pipeline

| Component | Flow | Tags | Mechanism | Status | Jira | PR |
|-----------|------|------|-----------|:------:|------|----|
| Listener → Resolver | K8s event dispatchers | `[eventpipeline][inner]` | PubSub `KubernetesDispatcherEventTopic` | :white_check_mark: | [ROX-32265](https://redhat.atlassian.net/browse/ROX-32265) | [#18235](https://github.com/stackrox/stackrox/pull/18235) |
| Resolver → Output | Resolved events to output queue | `[eventpipeline][inner]` | PubSub `ResolvedResourceEventTopic` (dual-path) | :white_check_mark: | [ROX-34880](https://redhat.atlassian.net/browse/ROX-34880) | [#20898](https://github.com/stackrox/stackrox/pull/20898) |
| Pipeline → Resolver | Central reprocess/update messages | `[eventpipeline][central-in]` | PubSub `FromCentralResolverEventTopic` (dual-path) | :white_check_mark: | [ROX-34880](https://redhat.atlassian.net/browse/ROX-34880) | [#20898](https://github.com/stackrox/stackrox/pull/20898) |
| Output Queue | `innerQueue` processing | `[eventpipeline][inner]` | PubSub consumer callback (dual-path) | :white_check_mark: | [ROX-34880](https://redhat.atlassian.net/browse/ROX-34880) | [#20898](https://github.com/stackrox/stackrox/pull/20898) |
| Resolver | `deploymentRefQueue` (dedup/aggregation) | `[eventpipeline][inner]` | `DedupingQueue` with merge semantics | :black_square_button: | [ROX-35646](https://redhat.atlassian.net/browse/ROX-35646) | |
| Output Queue | `forwardQueue` → `eventsC` → `ResponsesC()` | `[eventpipeline][central-out]` | Buffered + unbuffered channels | :construction: | [ROX-35647](https://redhat.atlassian.net/browse/ROX-35647) | [#21651](https://github.com/stackrox/stackrox/pull/21651) |
| Output Queue | Remove stopper in PubSub path (cleanup) | `[eventpipeline][inner][cleanup]` | N/A | :black_square_button: | [ROX-35054](https://redhat.atlassian.net/browse/ROX-35054) | |

## Process Signals

| Component | Flow | Tags | Mechanism | Status | Jira | PR |
|-----------|------|------|-----------|:------:|------|----|
| Pipeline → Enricher | Unenriched indicators | `[pi][inner]` | PubSub `UnenrichedProcessIndicatorTopic` (dual-path) | :white_check_mark: | [ROX-31047](https://redhat.atlassian.net/browse/ROX-31047) | [#18546](https://github.com/stackrox/stackrox/pull/18546) |
| Enricher → Pipeline | Enriched indicators | `[pi][inner]` | PubSub `EnrichedProcessIndicatorTopic` (dual-path) | :white_check_mark: | [ROX-31047](https://redhat.atlassian.net/browse/ROX-31047) | [#18546](https://github.com/stackrox/stackrox/pull/18546) |
| Pipeline → Detector | `ProcessIndicator()` direct call decoupling | `[pi][inter]` | Direct function call | :construction: | [ROX-35620](https://redhat.atlassian.net/browse/ROX-35620) | [#21671](https://github.com/stackrox/stackrox/pull/21671) |
| Enricher | `metadataCallbackChan` (cluster entities) | `[pi][inner]` | Unbuffered channel | :black_square_button: | [ROX-35650](https://redhat.atlassian.net/browse/ROX-35650) | |
| Pipeline | Legacy channel mode + ChannelMultiplexer | `[pi][cleanup]` | Channels + multiplexer | :black_square_button: | [ROX-35649](https://redhat.atlassian.net/browse/ROX-35649) | |
| Signal Service | `indicators` → `ResponsesC()` | `[pi][central-out]` | Unbuffered channel (non-blocking drop) | :black_square_button: | [ROX-35648](https://redhat.atlassian.net/browse/ROX-35648) | |

## Network Flows

| Component | Flow | Tags | Mechanism | Status | Jira | PR |
|-----------|------|------|-----------|:------:|------|----|
| Network Flow Manager | `sensorUpdates` → `ResponsesC()` | `[netflows][central-out]` | Buffered channel (env-scaled, non-blocking drop) | :black_square_button: | [ROX-35651](https://redhat.atlassian.net/browse/ROX-35651) | |
| Network Flow Service | `recvdMsgC` (collector gRPC → processing) | `[netflows][inner]` | Unbuffered channel | :black_square_button: | [ROX-35652](https://redhat.atlassian.net/browse/ROX-35652) | |
| External Sources | `PushNetworkEntitiesRequest` → handler | `[netflows][central-in]` | Direct call + `concurrency.Signal` | :black_square_button: | [ROX-35655](https://redhat.atlassian.net/browse/ROX-35655) | |
| Public IPs Manager | Public IPs → Collector | `[netflows][inner]` | `ValueStream[*sensor.IPAddressList]` | :black_square_button: | [ROX-35653](https://redhat.atlassian.net/browse/ROX-35653) | |
| External Sources | External sources → Collector | `[netflows][inner]` | `ValueStream[*sensor.IPNetworkList]` | :black_square_button: | [ROX-35653](https://redhat.atlassian.net/browse/ROX-35653) | |

## File System

| Component | Flow | Tags | Mechanism | Status | Jira | PR |
|-----------|------|------|-----------|:------:|------|----|
| FS Service → Pipeline | `activityChan` (collector gRPC → pipeline) | `[fs][inner]` | Unbuffered channel | :black_square_button: | | |
| FS Pipeline | Enrichment via PubSub (UnenrichedPI/EnrichedPI) | `[fs][inner]` | PubSub (reuses PI topics) | :white_check_mark: | | |
| FS Settings Manager | `FactSettings` policies → Fact agent | `[fs][central-in]` | `ValueStream[*v1.ConfigMap]` | :black_square_button: | | |

## Compliance

| Component | Flow | Tags | Mechanism | Status | Jira | PR |
|-----------|------|------|-----------|:------:|------|----|
| Command Handler | `commands` (ScrapeCommand from Central) | `[compliance][central-in]` | Unbuffered channel | :black_square_button: | | |
| Command Handler | `updates` → `ResponsesC()` | `[compliance][central-out]` | Unbuffered channel | :black_square_button: | | |
| Node Inventory | `toCentral` → `ResponsesC()` | `[compliance][central-out]` | Unbuffered channel | :black_square_button: | | |
| Audit Log Manager | `fileStateUpdates` → `ResponsesC()` | `[compliance][central-out]` | Unbuffered channel | :black_square_button: | | |
| Service | `output` (ComplianceReturn → command handler) | `[compliance][inner]` | Unbuffered channel | :black_square_button: | | |
| Service | `auditEventMsgs` (gRPC → audit log manager) | `[compliance][inner]` | Unbuffered channel | :black_square_button: | | |
| Service | `nodeInventories` (gRPC → node inventory handler) | `[compliance][inner]` | Unbuffered channel | :black_square_button: | | |
| Service | `indexReportWraps` (gRPC → node inventory handler) | `[compliance][inner]` | Unbuffered channel | :black_square_button: | | |
| Node Inventory | `toCompliance` (ACKs → compliance pods) | `[compliance][inter]` | Unbuffered channel | :black_square_button: | | |
| Node Inventory | `acksFromCentral` (Central ACKs) | `[compliance][inter]` | Unbuffered channel | :black_square_button: | | |
| Multiplexer | ChannelMultiplexer for `MessageToComplianceWithAddress` | `[compliance][inter]` | ChannelMultiplexer | :black_square_button: | | |

## Compliance Operator

| Component | Flow | Tags | Mechanism | Status | Jira | PR |
|-----------|------|------|-----------|:------:|------|----|
| Handler | `request` (ComplianceRequest from Central) | `[co][central-in]` | Unbuffered channel | :black_square_button: | | |
| Handler | `response` → `ResponsesC()` | `[co][central-out]` | Unbuffered channel | :black_square_button: | | |
| Updater | `response` → `ResponsesC()` | `[co][central-out]` | Unbuffered channel | :black_square_button: | | |

## Admission Controller

| Component | Flow | Tags | Mechanism | Status | Jira | PR |
|-----------|------|------|-----------|:------:|------|----|
| Settings Manager | `settingsStream` (settings broadcast) | `[ac][inner]` | `ValueStream[*sensor.AdmissionControlSettings]` | :black_square_button: | | |
| Settings Manager | `configStream` (ConfigMap persistence) | `[ac][inner]` | `ValueStream[*v1.ConfigMap]` | :black_square_button: | | |
| Settings Manager | `sensorEventsStream` (resource updates) | `[ac][inner]` | `ValueStream[*sensor.AdmCtrlUpdateResourceRequest]` | :black_square_button: | | |
| Settings Manager | `imageCacheInvalidationStream` | `[ac][inner]` | `ValueStream[*sensor.AdmCtrlImageCacheInvalidation]` | :black_square_button: | | |
| AC Manager | `settingsC` (settings → processing loop) | `[ac][inner]` | Unbuffered channel | :black_square_button: | | |
| AC Manager | `resourceUpdatesC` (updates → processing loop) | `[ac][inner]` | Unbuffered channel | :black_square_button: | | |
| AC Manager | `imageCacheInvalidationC` (cache → processing loop) | `[ac][inner]` | Unbuffered channel | :black_square_button: | | |
| AC Manager | `alertsC` (webhook → alert sender) | `[ac][inner]` | Unbuffered channel | :black_square_button: | | |
| Alert Handler | `output` → forwarder → Central | `[ac][central-out]` | Unbuffered channel | :black_square_button: | | |

## Enforcer

| Component | Flow | Tags | Mechanism | Status | Jira | PR |
|-----------|------|------|-----------|:------:|------|----|
| Enforcer | `actionsC` (Central + detector → enforcement) | `[enforcer][central-in][inter]` | Buffered channel (size 10) | :black_square_button: | | |

## Telemetry

| Component | Flow | Tags | Mechanism | Status | Jira | PR |
|-----------|------|------|-----------|:------:|------|----|
| Command Handler | `responsesC` → `ResponsesC()` | `[telemetry][central-out]` | Unbuffered channel | :black_square_button: | | |

## Deployment Enhancer

| Component | Flow | Tags | Mechanism | Status | Jira | PR |
|-----------|------|------|-----------|:------:|------|----|
| Enhancer | `deploymentsQueue` (Central → enhancement loop) | `[enhancer][central-in][inner]` | Buffered channel (50, scaled) | :black_square_button: | | |
| Enhancer | `responsesC` → `ResponsesC()` | `[enhancer][central-out]` | Unbuffered channel | :black_square_button: | | |

## Cert Refresh

| Component | Flow | Tags | Mechanism | Status | Jira | PR |
|-----------|------|------|-----------|:------:|------|----|
| TLS Issuer | `responseQueue` (Central response → requestCertificates) | `[certrefresh][inner]` | Unbounded `queue.Queue` | :black_square_button: | | |
| TLS Issuer | `msgToCentralC` → `ResponsesC()` | `[certrefresh][central-out]` | Unbuffered channel | :black_square_button: | | |

## Cluster Metrics / Health / Status

| Component | Flow | Tags | Mechanism | Status | Jira | PR |
|-----------|------|------|-----------|:------:|------|----|
| ClusterMetrics | `output` → `ResponsesC()` | `[cluster-status][central-out]` | Unbuffered channel | :black_square_button: | | |
| ClusterHealth | `updates` → `ResponsesC()` | `[cluster-status][central-out]` | Unbuffered channel | :black_square_button: | | |
| ClusterStatus | `updates` → `ResponsesC()` | `[cluster-status][central-out]` | Unbuffered channel | :black_square_button: | | |

## Network Policies

| Component | Flow | Tags | Mechanism | Status | Jira | PR |
|-----------|------|------|-----------|:------:|------|----|
| Command Handler | `commandsC` (Central → command processing) | `[netpol][central-in]` | Unbuffered channel | :black_square_button: | | |
| Command Handler | `responsesC` → `ResponsesC()` | `[netpol][central-out]` | Unbuffered channel | :black_square_button: | | |

## VM4VM (KubeVirt)

| Component | Flow | Tags | Mechanism | Status | Jira | PR |
|-----------|------|------|-----------|:------:|------|----|
| Index Handler | `indexReports` (compliance gRPC → handler) | `[vm][inner]` | Buffered channel (100, env) | :black_square_button: | | |
| Index Handler | `toCompliance` (ACK routing → compliance pods) | `[vm][inter]` | Buffered channel (size 1) | :black_square_button: | | |
| Index Handler | `ch2Central` → `ResponsesC()` | `[vm][central-out]` | Unbuffered channel | :black_square_button: | | |
| Index Handler | `ComplianceC()` output (compliance multiplexer) | `[vm][inter]` | Channel via ChannelMultiplexer | :black_square_button: | | |

## Cleanup

| Component | Flow | Tags | Mechanism | Status | Jira | PR |
|-----------|------|------|-----------|:------:|------|----|
| ChannelMultiplexer | Remove `pkg/channelmultiplexer/` package | `[cleanup]` | N/A | :black_square_button: | | |
