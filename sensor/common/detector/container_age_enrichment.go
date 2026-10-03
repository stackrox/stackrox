package detector

// SPIKE: Container Age policy criterion — sensor-side enrichment stub
//
// This file documents the enrichment design for the ContainerAge policy criterion
// (fieldnames.ContainerAge) and should be replaced with a real implementation
// once the design questions below are resolved.
//
// # What this needs to do
//
// Before calling DetectDeployment, each container's oldest_container_started timestamp
// must be populated on storage.Deployment.Containers[i].OldestContainerStarted.
// This field is transient: it must be set at detection time from live pod data and
// must NOT be persisted to the deployments table.
//
// # Sensor-side approach (preferred)
//
// The detector builds the EnhancedDeployment immediately before calling DetectDeployment.
// Enrichment should happen in detectDeploymentFromScanResult, just before that call,
// using a per-deployment lookup into the sensor pod cache.
//
// ## What the pod cache needs
//
// The concrete PodStore (sensor/kubernetes/listener/resources.PodStore) already stores
// pods indexed by (namespace, deploymentID) and exposes a forEach method, but the
// store.PodStore INTERFACE only exports:
//
//   GetAll() []*storage.Pod
//   GetByName(podName, namespace string) *storage.Pod
//
// To support sensor-side enrichment without iterating all pods, the interface needs
// a new method, e.g.:
//
//   ForDeployment(ns, deploymentID string, fn func(*storage.Pod))
//
// The concrete implementation already has forEach — this would just expose it.
// The mock in sensor/common/store/mocks/types.go would need to be regenerated.
//
// ## Wire-up
//
// The detectorImpl struct would need a new field:
//
//   podStore store.PodStore
//
// ... and it would be passed in from the sensor main wiring (sensor.go),
// analogous to how admissioncontroller.SettingsManager already receives a PodStore.
//
// # Central-side alternative
//
// Enrichment could happen in central/sensor/service/pipeline/reprocessing/pipeline.go
// before ReprocessDeploymentRisk is called, using central's pod datastore. This avoids
// touching the sensor interface but requires an extra datastore dependency in central.
// Central's pod datastore is queryable by deployment ID via search options.
// Recommendation: prefer sensor-side to keep enrichment collocated with detection.
//
// # Enrichment algorithm
//
//   func enrichContainerAge(deployment *storage.Deployment, podStore store.PodStore) {
//       byContainerName := map[string]*timestamppb.Timestamp{}
//       podStore.ForDeployment(deployment.GetNamespace(), deployment.GetId(), func(pod *storage.Pod) {
//           for _, inst := range pod.GetLiveInstances() {
//               t := inst.GetStarted()
//               if t == nil {
//                   continue
//               }
//               name := inst.GetContainerName()
//               if prev := byContainerName[name]; prev == nil || t.AsTime().Before(prev.AsTime()) {
//                   byContainerName[name] = t
//               }
//           }
//       })
//       for _, c := range deployment.GetContainers() {
//           c.OldestContainerStarted = byContainerName[c.GetName()]
//       }
//   }
//
// For deployments with no live pods (Deployment with zero replicas, or not yet scheduled),
// oldest_container_started will remain nil, and the criterion will not fire — correct.
//
// # Clearing violations
//
// When a container restarts, the sensor receives a new ContainerInstance event. The
// next reprocessing cycle re-evaluates, byContainerName gets the fresh start time,
// and the violation clears automatically.
//
// # Open questions for acceptance criteria
//
// 1. DaemonSet containers: one pod per node, potentially hundreds of containers per
//    deployment. Is oldest_container_started across all DaemonSet pods the right value,
//    or should the user have a way to scope by node?
// 2. InitContainers: currently ContainerInstance records init containers. Should the
//    criterion apply to init containers? (probably not — filter by container type)
// 3. Violation granularity: per-deployment or per-pod? The current Deployment model
//    aggregates across all pods; per-pod would require restructuring violations.
