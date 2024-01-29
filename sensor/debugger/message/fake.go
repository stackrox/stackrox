package message

import (
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/centralsensor"
)

// SensorHello returns a fake SensorHello message
func SensorHello(clusterID string) *central.MsgToSensor {
	return &central.MsgToSensor{
		Msg: &central.MsgToSensor_Hello{
			Hello: &central.CentralHello{
				ClusterId:        clusterID,
				CertBundle:       map[string]string{},
				Capabilities:     []string{centralsensor.SendDeduperStateOnReconnect},
				SendDeduperState: true,
			},
		},
	}
}

// DeduperState returns as fake DeduperState message
func DeduperState(state map[string]uint64, current, total int32) *central.MsgToSensor {
	return &central.MsgToSensor{
		Msg: &central.MsgToSensor_DeduperState{
			DeduperState: &central.DeduperState{
				ResourceHashes: state,
				Current:        current,
				Total:          total,
			},
		},
	}
}

// ClusterConfig returns a fake ClusterConfig message
func ClusterConfig() *central.MsgToSensor {
	return &central.MsgToSensor{
		Msg: &central.MsgToSensor_ClusterConfig{
			ClusterConfig: &central.ClusterConfig{
				Config: &storage.DynamicClusterConfig{
					AdmissionControllerConfig: &storage.AdmissionControllerConfig{},
					RegistryOverride:          "",
					DisableAuditLogs:          false,
				},
			},
		},
	}
}

// NetworkBaselineSync returns a fake NetworkBaselineSync message
func NetworkBaselineSync(baseline []*storage.NetworkBaseline) *central.MsgToSensor {
	return &central.MsgToSensor{
		Msg: &central.MsgToSensor_NetworkBaselineSync{
			NetworkBaselineSync: &central.NetworkBaselineSync{
				NetworkBaselines: baseline,
			},
		},
	}
}

// BaselineSync returns a fake BaselineSync message
func BaselineSync(baseline []*storage.ProcessBaseline) *central.MsgToSensor {
	return &central.MsgToSensor{
		Msg: &central.MsgToSensor_BaselineSync{
			BaselineSync: &central.BaselineSync{
				Baselines: baseline,
			},
		},
	}
}

// PolicySync returns a fake PolicySync message
func PolicySync(policies []*storage.Policy) *central.MsgToSensor {
	return &central.MsgToSensor{
		Msg: &central.MsgToSensor_PolicySync{
			PolicySync: &central.PolicySync{
				Policies: policies,
			},
		},
	}
}
