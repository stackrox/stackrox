package message

import (
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
)

// SensorHello returns a fake SensorHello message
func SensorHello(clusterID string, centralCaps ...string) *central.MsgToSensor {
	hello := &central.CentralHello{}
	hello.SetClusterId(clusterID)
	hello.SetCertBundle(map[string]string{})
	hello.SetCapabilities(centralCaps)
	hello.SetSendDeduperState(true)

	msg := &central.MsgToSensor{}
	msg.SetHello(hello)
	return msg
}

// DeduperState returns as fake DeduperState message
func DeduperState(state map[string]uint64, current, total int32) *central.MsgToSensor {
	deduperState := &central.DeduperState{}
	deduperState.SetResourceHashes(state)
	deduperState.SetCurrent(current)
	deduperState.SetTotal(total)

	msg := &central.MsgToSensor{}
	msg.SetDeduperState(deduperState)
	return msg
}

// ClusterConfig returns a fake ClusterConfig message
func ClusterConfig() *central.MsgToSensor {
	dynamicConfig := &storage.DynamicClusterConfig{}
	dynamicConfig.SetAdmissionControllerConfig(&storage.AdmissionControllerConfig{})
	dynamicConfig.SetRegistryOverride("")
	dynamicConfig.SetDisableAuditLogs(false)

	clusterConfig := &central.ClusterConfig{}
	clusterConfig.SetConfig(dynamicConfig)

	msg := &central.MsgToSensor{}
	msg.SetClusterConfig(clusterConfig)
	return msg
}

// NetworkBaselineSync returns a fake NetworkBaselineSync message
func NetworkBaselineSync(baseline []*storage.NetworkBaseline) *central.MsgToSensor {
	networkBaselineSync := &central.NetworkBaselineSync{}
	networkBaselineSync.SetNetworkBaselines(baseline)

	msg := &central.MsgToSensor{}
	msg.SetNetworkBaselineSync(networkBaselineSync)
	return msg
}

// BaselineSync returns a fake BaselineSync message
func BaselineSync(baseline []*storage.ProcessBaseline) *central.MsgToSensor {
	baselineSync := &central.BaselineSync{}
	baselineSync.SetBaselines(baseline)

	msg := &central.MsgToSensor{}
	msg.SetBaselineSync(baselineSync)
	return msg
}

// PolicySync returns a fake PolicySync message
func PolicySync(policies []*storage.Policy) *central.MsgToSensor {
	policySync := &central.PolicySync{}
	policySync.SetPolicies(policies)

	msg := &central.MsgToSensor{}
	msg.SetPolicySync(policySync)
	return msg
}
