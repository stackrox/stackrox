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

// GetPlopStorage1 Return a plop for the database
func GetPlopStorage1() *storage.ProcessListeningOnPortStorage {
	return &storage.ProcessListeningOnPortStorage{
		Id:                 fixtureconsts.PlopUID1,
		Port:               1234,
		Protocol:           storage.L4Protocol_L4_PROTOCOL_TCP,
		ProcessIndicatorId: fixtureconsts.ProcessIndicatorID1,
		CloseTimestamp:     timestamp.NowMinus(20 * time.Minute),
		Closed:             true,
		Process:	    GetProcessIndicatorUniqueKey1(),
		DeploymentId:       fixtureconsts.Deployment6,
	}
}

// GetPlopStorage2 Return a plop for the database
func GetPlopStorage2() *storage.ProcessListeningOnPortStorage {
	return &storage.ProcessListeningOnPortStorage{
		Id:                 fixtureconsts.PlopUID2,
		Port:               1234,
		Protocol:           storage.L4Protocol_L4_PROTOCOL_TCP,
		ProcessIndicatorId: fixtureconsts.ProcessIndicatorID2,
		CloseTimestamp:     timestamp.NowMinus(20 * time.Minute),
		Closed:             true,
		Process:	    GetProcessIndicatorUniqueKey2(),
		DeploymentId:       fixtureconsts.Deployment5,
	}
}

// GetPlopStorage3 Return a plop for the database
func GetPlopStorage3() *storage.ProcessListeningOnPortStorage {
	return &storage.ProcessListeningOnPortStorage{
		Id:                 fixtureconsts.PlopUID3,
		Port:               1234,
		Protocol:           storage.L4Protocol_L4_PROTOCOL_TCP,
		ProcessIndicatorId: fixtureconsts.ProcessIndicatorID3,
		CloseTimestamp:     timestamp.NowMinus(20 * time.Minute),
		Closed:             true,
		DeploymentId:       fixtureconsts.Deployment3,
	}
}

// GetPlopStorage4 Return a plop for the database
// It is the same as GetPlopStorage1 except it has a PodUid
func GetPlopStorage4() *storage.ProcessListeningOnPortStorage {
	return &storage.ProcessListeningOnPortStorage{
		Id:                 fixtureconsts.PlopUID4,
		Port:               1234,
		Protocol:           storage.L4Protocol_L4_PROTOCOL_TCP,
		ProcessIndicatorId: fixtureconsts.ProcessIndicatorID1,
		CloseTimestamp:     timestamp.NowMinus(20 * time.Minute),
		Closed:             true,
		DeploymentId:       fixtureconsts.Deployment6,
		Process:	    GetProcessIndicatorUniqueKey1(),
		PodUid:             fixtureconsts.PodUID1,
	}
}

// GetPlopStorage5 Return a plop for the database
// It is the same as GetPlopStorage2 except it has a PodUid
func GetPlopStorage5() *storage.ProcessListeningOnPortStorage {
	return &storage.ProcessListeningOnPortStorage{
		Id:                 fixtureconsts.PlopUID5,
		Port:               1234,
		Protocol:           storage.L4Protocol_L4_PROTOCOL_TCP,
		ProcessIndicatorId: fixtureconsts.ProcessIndicatorID2,
		CloseTimestamp:     timestamp.NowMinus(20 * time.Minute),
		Closed:             true,
		DeploymentId:       fixtureconsts.Deployment5,
		Process:	    GetProcessIndicatorUniqueKey2(),
		PodUid:             fixtureconsts.PodUID2,
	}
}

// GetPlopStorage6 Return a plop for the database
// It is the same as GetPlopStorage3 except it has a PodUid
func GetPlopStorage6() *storage.ProcessListeningOnPortStorage {
	return &storage.ProcessListeningOnPortStorage{
		Id:                 fixtureconsts.PlopUID6,
		Port:               1234,
		Protocol:           storage.L4Protocol_L4_PROTOCOL_TCP,
		ProcessIndicatorId: fixtureconsts.ProcessIndicatorID3,
		CloseTimestamp:     timestamp.NowMinus(20 * time.Minute),
		Closed:             true,
		DeploymentId:       fixtureconsts.Deployment3,
		PodUid:             fixtureconsts.PodUID3,
	}
}

// GetPlopStorageExpired1 Return an expired plop for the database
func GetPlopStorageExpired1() *storage.ProcessListeningOnPortStorage {
	return &storage.ProcessListeningOnPortStorage{
		Id:                 fixtureconsts.PlopUID7,
		Port:               1234,
		Protocol:           storage.L4Protocol_L4_PROTOCOL_TCP,
		ProcessIndicatorId: fixtureconsts.ProcessIndicatorID1,
		CloseTimestamp:     timestamp.NowMinus(1 * time.Hour),
		Closed:             true,
		DeploymentId:       fixtureconsts.Deployment6,
		Process:	    GetProcessIndicatorUniqueKey1(),
		PodUid:             fixtureconsts.PodUID1,
	}
}

// GetPlopStorageExpired2 Return an expired plop for the database
func GetPlopStorageExpired2() *storage.ProcessListeningOnPortStorage {
	return &storage.ProcessListeningOnPortStorage{
		Id:                 fixtureconsts.PlopUID8,
		Port:               1234,
		Protocol:           storage.L4Protocol_L4_PROTOCOL_TCP,
		ProcessIndicatorId: fixtureconsts.ProcessIndicatorID2,
		CloseTimestamp:     timestamp.NowMinus(1 * time.Hour),
		Closed:             true,
		DeploymentId:       fixtureconsts.Deployment5,
		Process:	    GetProcessIndicatorUniqueKey2(),
		PodUid:             fixtureconsts.PodUID2,
	}
}

// GetPlopStorageExpired3 Return an expired plop for the database
func GetPlopStorageExpired3() *storage.ProcessListeningOnPortStorage {
	return &storage.ProcessListeningOnPortStorage{
		Id:                 fixtureconsts.PlopUID9,
		Port:               1234,
		Protocol:           storage.L4Protocol_L4_PROTOCOL_TCP,
		ProcessIndicatorId: fixtureconsts.ProcessIndicatorID3,
		CloseTimestamp:     timestamp.NowMinus(1 * time.Hour),
		Closed:             true,
		DeploymentId:       fixtureconsts.Deployment3,
		PodUid:             fixtureconsts.PodUID3,
	}
}
