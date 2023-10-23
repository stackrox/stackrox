package fixtures

import (
	"time"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/fixtures/fixtureconsts"
	"github.com/stackrox/rox/pkg/protoconv"
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
