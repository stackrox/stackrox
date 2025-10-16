package message

import (
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"google.golang.org/protobuf/proto"
)

// SensorHello returns a fake SensorHello message
func SensorHello(clusterID string, centralCaps ...string) *central.MsgToSensor {
	return central.MsgToSensor_builder{
		Hello: central.CentralHello_builder{
			ClusterId:        clusterID,
			CertBundle:       map[string]string{},
			Capabilities:     centralCaps,
			SendDeduperState: true,
		}.Build(),
	}.Build()
}

// DeduperState returns as fake DeduperState message
func DeduperState(state map[string]uint64, current, total int32) *central.MsgToSensor {
	ds := &central.DeduperState{}
	ds.SetResourceHashes(state)
	ds.SetCurrent(current)
	ds.SetTotal(total)
	mts := &central.MsgToSensor{}
	mts.SetDeduperState(proto.ValueOrDefault(ds))
	return mts
}

// ClusterConfig returns a fake ClusterConfig message
func ClusterConfig() *central.MsgToSensor {
	return central.MsgToSensor_builder{
		ClusterConfig: central.ClusterConfig_builder{
			Config: storage.DynamicClusterConfig_builder{
				AdmissionControllerConfig: &storage.AdmissionControllerConfig{},
				RegistryOverride:          "",
				DisableAuditLogs:          false,
			}.Build(),
		}.Build(),
	}.Build()
}

// NetworkBaselineSync returns a fake NetworkBaselineSync message
func NetworkBaselineSync(baseline []*storage.NetworkBaseline) *central.MsgToSensor {
	nbs := &central.NetworkBaselineSync{}
	nbs.SetNetworkBaselines(baseline)
	mts := &central.MsgToSensor{}
	mts.SetNetworkBaselineSync(proto.ValueOrDefault(nbs))
	return mts
}

// BaselineSync returns a fake BaselineSync message
func BaselineSync(baseline []*storage.ProcessBaseline) *central.MsgToSensor {
	bs := &central.BaselineSync{}
	bs.SetBaselines(baseline)
	mts := &central.MsgToSensor{}
	mts.SetBaselineSync(proto.ValueOrDefault(bs))
	return mts
}

// PolicySync returns a fake PolicySync message
func PolicySync(policies []*storage.Policy) *central.MsgToSensor {
	ps := &central.PolicySync{}
	ps.SetPolicies(policies)
	mts := &central.MsgToSensor{}
	mts.SetPolicySync(proto.ValueOrDefault(ps))
	return mts
}
