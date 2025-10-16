package fixtures

import (
	"time"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/fixtures/fixtureconsts"
	"github.com/stackrox/rox/pkg/protoconv"
)

// GetOpenPlopObject1 Return an open plop object
func GetOpenPlopObject1() *storage.ProcessListeningOnPortFromSensor {
	piuk := &storage.ProcessIndicatorUniqueKey{}
	piuk.SetPodId(fixtureconsts.PodName1)
	piuk.SetContainerName("containername")
	piuk.SetProcessName("test_process1")
	piuk.SetProcessArgs("test_arguments1")
	piuk.SetProcessExecFilePath("test_path1")
	plopfs := &storage.ProcessListeningOnPortFromSensor{}
	plopfs.SetPort(1234)
	plopfs.SetProtocol(storage.L4Protocol_L4_PROTOCOL_TCP)
	plopfs.ClearCloseTimestamp()
	plopfs.SetProcess(piuk)
	plopfs.SetDeploymentId(fixtureconsts.Deployment1)
	plopfs.SetClusterId(fixtureconsts.Cluster1)
	plopfs.SetPodUid(fixtureconsts.PodUID1)
	return plopfs
}

// GetClosePlopObject1 Return an open plop object
func GetClosePlopObject1() *storage.ProcessListeningOnPortFromSensor {
	piuk := &storage.ProcessIndicatorUniqueKey{}
	piuk.SetPodId(fixtureconsts.PodName1)
	piuk.SetContainerName("containername")
	piuk.SetProcessName("test_process1")
	piuk.SetProcessArgs("test_arguments1")
	piuk.SetProcessExecFilePath("test_path1")
	plopfs := &storage.ProcessListeningOnPortFromSensor{}
	plopfs.SetPort(1234)
	plopfs.SetProtocol(storage.L4Protocol_L4_PROTOCOL_TCP)
	plopfs.SetCloseTimestamp(protoconv.ConvertTimeToTimestamp(time.Now()))
	plopfs.SetProcess(piuk)
	plopfs.SetDeploymentId(fixtureconsts.Deployment1)
	plopfs.SetClusterId(fixtureconsts.Cluster1)
	plopfs.SetPodUid(fixtureconsts.PodUID1)
	return plopfs
}

// GetOpenPlopObject2 Return an open plop object
func GetOpenPlopObject2() *storage.ProcessListeningOnPortFromSensor {
	piuk := &storage.ProcessIndicatorUniqueKey{}
	piuk.SetPodId(fixtureconsts.PodName1)
	piuk.SetContainerName("containername")
	piuk.SetProcessName("test_process2")
	piuk.SetProcessArgs("test_arguments2")
	piuk.SetProcessExecFilePath("test_path2")
	plopfs := &storage.ProcessListeningOnPortFromSensor{}
	plopfs.SetPort(80)
	plopfs.SetProtocol(storage.L4Protocol_L4_PROTOCOL_TCP)
	plopfs.ClearCloseTimestamp()
	plopfs.SetProcess(piuk)
	plopfs.SetDeploymentId(fixtureconsts.Deployment1)
	plopfs.SetClusterId(fixtureconsts.Cluster1)
	plopfs.SetPodUid(fixtureconsts.PodUID1)
	return plopfs
}

// GetOpenPlopObject3 Return an open plop object
func GetOpenPlopObject3() *storage.ProcessListeningOnPortFromSensor {
	piuk := &storage.ProcessIndicatorUniqueKey{}
	piuk.SetPodId(fixtureconsts.PodName2)
	piuk.SetContainerName("containername")
	piuk.SetProcessName("apt-get")
	piuk.SetProcessArgs("install nmap")
	piuk.SetProcessExecFilePath("bin")
	plopfs := &storage.ProcessListeningOnPortFromSensor{}
	plopfs.SetPort(80)
	plopfs.SetProtocol(storage.L4Protocol_L4_PROTOCOL_TCP)
	plopfs.ClearCloseTimestamp()
	plopfs.SetProcess(piuk)
	plopfs.SetDeploymentId(fixtureconsts.Deployment1)
	plopfs.SetClusterId(fixtureconsts.Cluster1)
	plopfs.SetPodUid(fixtureconsts.PodUID2)
	return plopfs
}

// GetOpenPlopObject4 Return an open plop object
func GetOpenPlopObject4() *storage.ProcessListeningOnPortFromSensor {
	piuk := &storage.ProcessIndicatorUniqueKey{}
	piuk.SetPodId(fixtureconsts.PodName2)
	piuk.SetContainerName("containername")
	piuk.SetProcessName("apt-get")
	piuk.SetProcessArgs("install nmap")
	piuk.SetProcessExecFilePath("bin")
	plopfs := &storage.ProcessListeningOnPortFromSensor{}
	plopfs.SetPort(80)
	plopfs.SetProtocol(storage.L4Protocol_L4_PROTOCOL_TCP)
	plopfs.ClearCloseTimestamp()
	plopfs.SetProcess(piuk)
	plopfs.SetDeploymentId(fixtureconsts.Deployment1)
	plopfs.SetClusterId(fixtureconsts.Cluster1)
	plopfs.SetPodUid(fixtureconsts.PodUID3)
	return plopfs
}

// GetPlopStorage1 Return a plop for the database
func GetPlopStorage1() *storage.ProcessListeningOnPortStorage {
	plops := &storage.ProcessListeningOnPortStorage{}
	plops.SetId(fixtureconsts.PlopUID1)
	plops.SetPort(1234)
	plops.SetProtocol(storage.L4Protocol_L4_PROTOCOL_TCP)
	plops.SetProcessIndicatorId(fixtureconsts.ProcessIndicatorID1)
	plops.SetCloseTimestamp(protoconv.NowMinus(20 * time.Minute))
	plops.SetClosed(true)
	plops.SetDeploymentId(fixtureconsts.Deployment6)
	return plops
}

// GetPlopStorage2 Return a plop for the database
func GetPlopStorage2() *storage.ProcessListeningOnPortStorage {
	plops := &storage.ProcessListeningOnPortStorage{}
	plops.SetId(fixtureconsts.PlopUID2)
	plops.SetPort(1234)
	plops.SetProtocol(storage.L4Protocol_L4_PROTOCOL_TCP)
	plops.SetProcessIndicatorId(fixtureconsts.ProcessIndicatorID2)
	plops.SetCloseTimestamp(protoconv.NowMinus(20 * time.Minute))
	plops.SetClosed(true)
	plops.SetDeploymentId(fixtureconsts.Deployment5)
	return plops
}

// GetPlopStorage3 Return a plop for the database
func GetPlopStorage3() *storage.ProcessListeningOnPortStorage {
	plops := &storage.ProcessListeningOnPortStorage{}
	plops.SetId(fixtureconsts.PlopUID3)
	plops.SetPort(1234)
	plops.SetProtocol(storage.L4Protocol_L4_PROTOCOL_TCP)
	plops.SetProcessIndicatorId(fixtureconsts.ProcessIndicatorID3)
	plops.SetCloseTimestamp(protoconv.NowMinus(20 * time.Minute))
	plops.SetClosed(true)
	plops.SetDeploymentId(fixtureconsts.Deployment3)
	return plops
}

// GetPlopStorage4 Return a plop for the database
// It is the same as GetPlopStorage1 except it has a PodUid
func GetPlopStorage4() *storage.ProcessListeningOnPortStorage {
	plops := &storage.ProcessListeningOnPortStorage{}
	plops.SetId(fixtureconsts.PlopUID4)
	plops.SetPort(1234)
	plops.SetProtocol(storage.L4Protocol_L4_PROTOCOL_TCP)
	plops.SetProcessIndicatorId(fixtureconsts.ProcessIndicatorID1)
	plops.SetCloseTimestamp(protoconv.NowMinus(20 * time.Minute))
	plops.SetClosed(true)
	plops.SetDeploymentId(fixtureconsts.Deployment6)
	plops.SetPodUid(fixtureconsts.PodUID1)
	return plops
}

// GetPlopStorage5 Return a plop for the database
// It is the same as GetPlopStorage2 except it has a PodUid
func GetPlopStorage5() *storage.ProcessListeningOnPortStorage {
	plops := &storage.ProcessListeningOnPortStorage{}
	plops.SetId(fixtureconsts.PlopUID5)
	plops.SetPort(1234)
	plops.SetProtocol(storage.L4Protocol_L4_PROTOCOL_TCP)
	plops.SetProcessIndicatorId(fixtureconsts.ProcessIndicatorID2)
	plops.SetCloseTimestamp(protoconv.NowMinus(20 * time.Minute))
	plops.SetClosed(true)
	plops.SetDeploymentId(fixtureconsts.Deployment5)
	plops.SetPodUid(fixtureconsts.PodUID2)
	return plops
}

// GetPlopStorage6 Return a plop for the database
// It is the same as GetPlopStorage3 except it has a PodUid
func GetPlopStorage6() *storage.ProcessListeningOnPortStorage {
	plops := &storage.ProcessListeningOnPortStorage{}
	plops.SetId(fixtureconsts.PlopUID6)
	plops.SetPort(1234)
	plops.SetProtocol(storage.L4Protocol_L4_PROTOCOL_TCP)
	plops.SetProcessIndicatorId(fixtureconsts.ProcessIndicatorID3)
	plops.SetCloseTimestamp(protoconv.NowMinus(20 * time.Minute))
	plops.SetClosed(true)
	plops.SetDeploymentId(fixtureconsts.Deployment3)
	plops.SetPodUid(fixtureconsts.PodUID3)
	return plops
}

// GetPlopStorage7 Return a plop for the database
func GetPlopStorage7() *storage.ProcessListeningOnPortStorage {
	plops := &storage.ProcessListeningOnPortStorage{}
	plops.SetId(fixtureconsts.PlopUID1)
	plops.SetPort(1234)
	plops.SetProtocol(storage.L4Protocol_L4_PROTOCOL_TCP)
	plops.SetProcessIndicatorId(fixtureconsts.ProcessIndicatorID1)
	plops.ClearCloseTimestamp()
	plops.SetClosed(false)
	plops.SetDeploymentId(fixtureconsts.Deployment1)
	plops.SetPodUid(fixtureconsts.PodUID1)
	return plops
}

// GetPlopStorage8 Return a plop for the database
func GetPlopStorage8() *storage.ProcessListeningOnPortStorage {
	plops := &storage.ProcessListeningOnPortStorage{}
	plops.SetId(fixtureconsts.PlopUID2)
	plops.SetPort(4321)
	plops.SetProtocol(storage.L4Protocol_L4_PROTOCOL_TCP)
	plops.SetProcessIndicatorId(fixtureconsts.ProcessIndicatorID2)
	plops.ClearCloseTimestamp()
	plops.SetClosed(false)
	plops.SetDeploymentId(fixtureconsts.Deployment1)
	plops.SetPodUid(fixtureconsts.PodUID3)
	return plops
}

// GetPlopStorage9 Return a plop for the database
func GetPlopStorage9() *storage.ProcessListeningOnPortStorage {
	plops := &storage.ProcessListeningOnPortStorage{}
	plops.SetId(fixtureconsts.PlopUID3)
	plops.SetPort(80)
	plops.SetProtocol(storage.L4Protocol_L4_PROTOCOL_TCP)
	plops.SetProcessIndicatorId(fixtureconsts.ProcessIndicatorID3)
	plops.ClearCloseTimestamp()
	plops.SetClosed(false)
	plops.SetDeploymentId(fixtureconsts.Deployment1)
	plops.SetPodUid(fixtureconsts.PodUID3)
	return plops
}

func GetPlop7() *storage.ProcessListeningOnPort {
	pe := &storage.ProcessListeningOnPort_Endpoint{}
	pe.SetPort(1234)
	pe.SetProtocol(storage.L4Protocol_L4_PROTOCOL_TCP)
	ps := &storage.ProcessSignal{}
	ps.SetName("test_process1")
	ps.SetArgs("test_arguments1")
	ps.SetExecFilePath("test_path1")
	plop := &storage.ProcessListeningOnPort{}
	plop.SetContainerName("test_container1")
	plop.SetPodId(fixtureconsts.PodName1)
	plop.SetPodUid(fixtureconsts.PodUID1)
	plop.SetDeploymentId(fixtureconsts.Deployment1)
	plop.SetClusterId(fixtureconsts.Cluster1)
	plop.SetNamespace(fixtureconsts.Namespace1)
	plop.SetEndpoint(pe)
	plop.SetSignal(ps)
	return plop
}

func GetPlop8() *storage.ProcessListeningOnPort {
	pe := &storage.ProcessListeningOnPort_Endpoint{}
	pe.SetPort(4321)
	pe.SetProtocol(storage.L4Protocol_L4_PROTOCOL_TCP)
	ps := &storage.ProcessSignal{}
	ps.SetName("test_process2")
	ps.SetArgs("test_arguments2")
	ps.SetExecFilePath("test_path2")
	plop := &storage.ProcessListeningOnPort{}
	plop.SetContainerName("test_container2")
	plop.SetPodId(fixtureconsts.PodName3)
	plop.SetPodUid(fixtureconsts.PodUID3)
	plop.SetDeploymentId(fixtureconsts.Deployment1)
	plop.SetClusterId(fixtureconsts.Cluster1)
	plop.SetNamespace(fixtureconsts.Namespace1)
	plop.SetEndpoint(pe)
	plop.SetSignal(ps)
	return plop
}

func GetPlop9() *storage.ProcessListeningOnPort {
	pe := &storage.ProcessListeningOnPort_Endpoint{}
	pe.SetPort(80)
	pe.SetProtocol(storage.L4Protocol_L4_PROTOCOL_TCP)
	ps := &storage.ProcessSignal{}
	ps.SetName("test_process3")
	ps.SetArgs("test_arguments3")
	ps.SetExecFilePath("test_path3")
	plop := &storage.ProcessListeningOnPort{}
	plop.SetContainerName("test_container2")
	plop.SetPodId(fixtureconsts.PodName3)
	plop.SetPodUid(fixtureconsts.PodUID3)
	plop.SetDeploymentId(fixtureconsts.Deployment1)
	plop.SetClusterId(fixtureconsts.Cluster1)
	plop.SetNamespace(fixtureconsts.Namespace1)
	plop.SetEndpoint(pe)
	plop.SetSignal(ps)
	return plop
}

// GetPlopStorageExpired1 Return an expired plop for the database
func GetPlopStorageExpired1() *storage.ProcessListeningOnPortStorage {
	plops := &storage.ProcessListeningOnPortStorage{}
	plops.SetId(fixtureconsts.PlopUID7)
	plops.SetPort(1234)
	plops.SetProtocol(storage.L4Protocol_L4_PROTOCOL_TCP)
	plops.SetProcessIndicatorId(fixtureconsts.ProcessIndicatorID1)
	plops.SetCloseTimestamp(protoconv.NowMinus(1 * time.Hour))
	plops.SetClosed(true)
	plops.SetDeploymentId(fixtureconsts.Deployment6)
	plops.SetPodUid(fixtureconsts.PodUID1)
	return plops
}

// GetPlopStorageExpired2 Return an expired plop for the database
func GetPlopStorageExpired2() *storage.ProcessListeningOnPortStorage {
	plops := &storage.ProcessListeningOnPortStorage{}
	plops.SetId(fixtureconsts.PlopUID8)
	plops.SetPort(1234)
	plops.SetProtocol(storage.L4Protocol_L4_PROTOCOL_TCP)
	plops.SetProcessIndicatorId(fixtureconsts.ProcessIndicatorID2)
	plops.SetCloseTimestamp(protoconv.NowMinus(1 * time.Hour))
	plops.SetClosed(true)
	plops.SetDeploymentId(fixtureconsts.Deployment5)
	plops.SetPodUid(fixtureconsts.PodUID2)
	return plops
}

// GetPlopStorageExpired3 Return an expired plop for the database
func GetPlopStorageExpired3() *storage.ProcessListeningOnPortStorage {
	plops := &storage.ProcessListeningOnPortStorage{}
	plops.SetId(fixtureconsts.PlopUID9)
	plops.SetPort(1234)
	plops.SetProtocol(storage.L4Protocol_L4_PROTOCOL_TCP)
	plops.SetProcessIndicatorId(fixtureconsts.ProcessIndicatorID3)
	plops.SetCloseTimestamp(protoconv.NowMinus(1 * time.Hour))
	plops.SetClosed(true)
	plops.SetDeploymentId(fixtureconsts.Deployment3)
	plops.SetPodUid(fixtureconsts.PodUID3)
	return plops
}
