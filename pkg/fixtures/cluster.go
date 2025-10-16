package fixtures

import (
	"github.com/stackrox/rox/generated/storage"
)

// GetCluster provides a filled cluster object for testing purposes.
func GetCluster(name string) *storage.Cluster {
	acc := &storage.AdmissionControllerConfig{}
	acc.SetEnabled(false)
	acc.SetTimeoutSeconds(10)
	acc.SetScanInline(false)
	acc.SetDisableBypass(false)
	acc.SetEnforceOnUpdates(false)
	dcc := &storage.DynamicClusterConfig{}
	dcc.SetAdmissionControllerConfig(acc)
	dcc.SetRegistryOverride("")
	dcc.SetDisableAuditLogs(true)
	sdi := &storage.SensorDeploymentIdentification{}
	sdi.SetSystemNamespaceId("dbcbf202-6086-4bf9-8bc1-d10af3e36883")
	sdi.SetDefaultNamespaceId("fcab1a6d-07a3-4da9-a9cf-e286537ed4e3")
	sdi.SetAppNamespace("stackrox")
	sdi.SetAppNamespaceId("cd14a849-21d3-4351-9a56-8a066c2e83e1")
	sdi.SetAppServiceaccountId("")
	sdi.SetK8SNodeName("colima")
	cluster := &storage.Cluster{}
	cluster.SetId("")
	cluster.SetName(name)
	cluster.SetType(storage.ClusterType_KUBERNETES_CLUSTER)
	cluster.SetLabels(nil)
	cluster.SetMainImage("quay.io/stackrox-io/main")
	cluster.SetCollectorImage("quay.io/stackrox-io/collector")
	cluster.SetCentralApiEndpoint("central.stackrox:443")
	cluster.SetRuntimeSupport(true)
	cluster.SetCollectionMethod(storage.CollectionMethod_CORE_BPF)
	cluster.SetAdmissionController(true)
	cluster.SetAdmissionControllerUpdates(false)
	cluster.SetAdmissionControllerEvents(true)
	cluster.ClearStatus()
	cluster.SetDynamicConfig(dcc)
	cluster.ClearTolerationsConfig()
	cluster.SetPriority(0)
	cluster.ClearHealthStatus()
	cluster.SetSlimCollector(true)
	cluster.ClearHelmConfig()
	cluster.SetMostRecentSensorId(sdi)
	cluster.SetAuditLogState(nil)
	cluster.SetInitBundleId("bb0e13e0-621a-4b2e-8fb9-af4e466763ff")
	cluster.SetManagedBy(storage.ManagerType_MANAGER_TYPE_MANUAL)
	return cluster
}
