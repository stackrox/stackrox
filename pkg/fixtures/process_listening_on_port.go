package fixtures

import (
	"time"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/fixtures/fixtureconsts"
	"github.com/stackrox/rox/pkg/protoconv"
	"github.com/stackrox/rox/pkg/timestamp"
)

// GetOpenPlopObject1 Return an open plop object
func GetOpenPlopObject1() *storage.ProcessListeningOnPortFromSensor {
	return &storage.ProcessListeningOnPortFromSensor{
		Port:           1234,
		Protocol:       storage.L4Protocol_L4_PROTOCOL_TCP,
		CloseTimestamp: nil,
		Process: &storage.ProcessIndicatorUniqueKey{
			PodId:               fixtureconsts.PodName1,
			ContainerName:       "containername",
			ProcessName:         "test_process1",
			ProcessArgs:         "test_arguments1",
			ProcessExecFilePath: "test_path1",
		},
		DeploymentId: fixtureconsts.Deployment1,
		ClusterId:    fixtureconsts.Cluster1,
		PodUid:       fixtureconsts.PodUID1,
	}
}

// GetClosePlopObject1 Return an open plop object
func GetClosePlopObject1() *storage.ProcessListeningOnPortFromSensor {
	return &storage.ProcessListeningOnPortFromSensor{
		Port:           1234,
		Protocol:       storage.L4Protocol_L4_PROTOCOL_TCP,
		CloseTimestamp: protoconv.ConvertTimeToTimestamp(time.Now()),
		Process: &storage.ProcessIndicatorUniqueKey{
			PodId:               fixtureconsts.PodName1,
			ContainerName:       "containername",
			ProcessName:         "test_process1",
			ProcessArgs:         "test_arguments1",
			ProcessExecFilePath: "test_path1",
		},
		DeploymentId: fixtureconsts.Deployment1,
		ClusterId:    fixtureconsts.Cluster1,
		PodUid:       fixtureconsts.PodUID1,
	}
}

// GetOpenPlopObject2 Return an open plop object
func GetOpenPlopObject2() *storage.ProcessListeningOnPortFromSensor {
	return &storage.ProcessListeningOnPortFromSensor{
		Port:           80,
		Protocol:       storage.L4Protocol_L4_PROTOCOL_TCP,
		CloseTimestamp: nil,
		Process: &storage.ProcessIndicatorUniqueKey{
			PodId:               fixtureconsts.PodName1,
			ContainerName:       "containername",
			ProcessName:         "test_process2",
			ProcessArgs:         "test_arguments2",
			ProcessExecFilePath: "test_path2",
		},
		DeploymentId: fixtureconsts.Deployment1,
		ClusterId:    fixtureconsts.Cluster1,
		PodUid:       fixtureconsts.PodUID1,
	}
}

// GetOpenPlopObject3 Return an open plop object
func GetOpenPlopObject3() *storage.ProcessListeningOnPortFromSensor {
	return &storage.ProcessListeningOnPortFromSensor{
		Port:           80,
		Protocol:       storage.L4Protocol_L4_PROTOCOL_TCP,
		CloseTimestamp: nil,
		Process: &storage.ProcessIndicatorUniqueKey{
			PodId:               fixtureconsts.PodName2,
			ContainerName:       "containername",
			ProcessName:         "apt-get",
			ProcessArgs:         "install nmap",
			ProcessExecFilePath: "bin",
		},
		DeploymentId: fixtureconsts.Deployment1,
		ClusterId:    fixtureconsts.Cluster1,
		PodUid:       fixtureconsts.PodUID2,
	}
}

// GetOpenPlopObject4 Return an open plop object
func GetOpenPlopObject4() *storage.ProcessListeningOnPortFromSensor {
	return &storage.ProcessListeningOnPortFromSensor{
		Port:           80,
		Protocol:       storage.L4Protocol_L4_PROTOCOL_TCP,
		CloseTimestamp: nil,
		Process: &storage.ProcessIndicatorUniqueKey{
			PodId:               fixtureconsts.PodName2,
			ContainerName:       "containername",
			ProcessName:         "apt-get",
			ProcessArgs:         "install nmap",
			ProcessExecFilePath: "bin",
		},
		DeploymentId: fixtureconsts.Deployment1,
		ClusterId:    fixtureconsts.Cluster1,
		PodUid:       fixtureconsts.PodUID3,
	}
}

func GetPlopStorage1() *storage.ProcessListeningOnPortStorage {
	return &storage.ProcessListeningOnPortStorage{
                Id:                     fixtureconsts.PlopUID1,
                Port:                   1234,
                Protocol:               storage.L4Protocol_L4_PROTOCOL_TCP,
                ProcessIndicatorId:     fixtureconsts.ProcessIndicatorID1,
		CloseTimestamp:		timestamp.TimestampNowMinus(1*time.Hour),
                Closed:                 true,
                DeploymentId:           fixtureconsts.Deployment6,
        }
}

func GetPlopStorage2() *storage.ProcessListeningOnPortStorage {
	return &storage.ProcessListeningOnPortStorage{
                Id:                     fixtureconsts.PlopUID2,
                Port:                   1234,
                Protocol:               storage.L4Protocol_L4_PROTOCOL_TCP,
                ProcessIndicatorId:     fixtureconsts.ProcessIndicatorID2,
		CloseTimestamp:		timestamp.TimestampNowMinus(1*time.Hour),
                Closed:                 true,
                DeploymentId:           fixtureconsts.Deployment6,
        }
}

func GetPlopStorage3() *storage.ProcessListeningOnPortStorage {
	return &storage.ProcessListeningOnPortStorage{
                Id:                     fixtureconsts.PlopUID3,
                Port:                   1234,
                Protocol:               storage.L4Protocol_L4_PROTOCOL_TCP,
                ProcessIndicatorId:     fixtureconsts.ProcessIndicatorID3,
		CloseTimestamp:		timestamp.TimestampNowMinus(1*time.Hour),
                Closed:                 true,
                DeploymentId:           fixtureconsts.Deployment3,
        }
}

func GetPlopStorage4() *storage.ProcessListeningOnPortStorage {
	return &storage.ProcessListeningOnPortStorage{
                Id:                     fixtureconsts.PlopUID4,
                Port:                   1234,
                Protocol:               storage.L4Protocol_L4_PROTOCOL_TCP,
                ProcessIndicatorId:     fixtureconsts.ProcessIndicatorID1,
		CloseTimestamp:		timestamp.TimestampNowMinus(1*time.Hour),
                Closed:                 true,
                DeploymentId:           fixtureconsts.Deployment6,
		PodUid:			fixtureconsts.PodUID1,
        }
}

func GetPlopStorage5() *storage.ProcessListeningOnPortStorage {
	return &storage.ProcessListeningOnPortStorage{
                Id:                     fixtureconsts.PlopUID5,
                Port:                   1234,
                Protocol:               storage.L4Protocol_L4_PROTOCOL_TCP,
                ProcessIndicatorId:     fixtureconsts.ProcessIndicatorID2,
		CloseTimestamp:		timestamp.TimestampNowMinus(1*time.Hour),
                Closed:                 true,
                DeploymentId:           fixtureconsts.Deployment6,
		PodUid:			fixtureconsts.PodUID2,
        }
}

func GetPlopStorage6() *storage.ProcessListeningOnPortStorage {
	return &storage.ProcessListeningOnPortStorage{
                Id:                     fixtureconsts.PlopUID6,
                Port:                   1234,
                Protocol:               storage.L4Protocol_L4_PROTOCOL_TCP,
                ProcessIndicatorId:     fixtureconsts.ProcessIndicatorID3,
		CloseTimestamp:		timestamp.TimestampNowMinus(1*time.Hour),
                Closed:                 true,
                DeploymentId:           fixtureconsts.Deployment3,
		PodUid:			fixtureconsts.PodUID3,
        }
}
