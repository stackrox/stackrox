
package fixtures

import (
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/fixtures/fixtureconsts"
)

func GetProcessIndicatorUniqueKey1() *storage.ProcessIndicatorUniqueKey {
	return &storage.ProcessIndicatorUniqueKey{
		PodId:               fixtureconsts.PodName1,
		ContainerName:       "containername",
		ProcessName:         "test_process1",
		ProcessArgs:         "test_arguments1",
		ProcessExecFilePath: "test_path1",
	}
}

func GetProcessIndicatorUniqueKey2() *storage.ProcessIndicatorUniqueKey {
	return &storage.ProcessIndicatorUniqueKey{
		PodId:               fixtureconsts.PodName2,
		ContainerName:       "containername",
		ProcessName:         "test_process2",
		ProcessArgs:         "test_arguments2",
		ProcessExecFilePath: "test_path2",
	}
}

func GetProcessIndicatorUniqueKey3() *storage.ProcessIndicatorUniqueKey {
	return &storage.ProcessIndicatorUniqueKey{
		PodId:               fixtureconsts.PodName2,
		ContainerName:       "containername",
		ProcessName:         "apt-get",
		ProcessArgs:         "install nmap",
		ProcessExecFilePath: "bin",
	}
}
