package message

import (
	"github.com/stackrox/stackrox/generated/internalapi/central"
	"github.com/stackrox/stackrox/generated/storage"
)

// SensorHello returns a fake SensorHello message
func SensorHello(clsuterID string) *central.MsgToSensor {
	return &central.MsgToSensor{
		Msg: &central.MsgToSensor_Hello{
			Hello: &central.CentralHello{
				ClusterId:  clsuterID,
				CertBundle: map[string]string{},
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
