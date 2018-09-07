package runner

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/stackrox/rox/benchmarks/checks/container_runtime"
	"github.com/stackrox/rox/benchmarks/checks/utils"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const containerName = "bench_test"
const networkName = "bench_test"

func runContainer(params ...string) error {
	var runCommands = []string{
		"run",
		"-d",
		"--name",
		containerName,
	}
	runCommands = append(runCommands, params...)
	runCommands = append(runCommands, "python:2.7", "python", "-m", "SimpleHTTPServer")
	log.Infof("docker %+v", runCommands)
	output, err := utils.CombinedOutput("docker", runCommands...)
	if err != nil {
		return fmt.Errorf("failed to run test container. Err: %+v. Output: %v", err, output)
	}
	time.Sleep(5 * time.Second)
	return nil
}

func createNetwork() {
	output, err := utils.CombinedOutput("docker", "network", "create", networkName)
	if err != nil {
		log.Info(output)
	} else {
		log.Infof("Successfully created network %v", networkName)
	}
}

func cleanupContainer() {
	output, err := utils.CombinedOutput("docker", "kill", containerName)
	if err != nil {
		log.Infof("Error killing %v: %+v: %v", containerName, err.Error(), output)
	} else {
		log.Infof("Successfully killed %v", containerName)
	}

	output, err = utils.CombinedOutput("docker", "rm", containerName)
	if err != nil {
		log.Infof("Error removing %v: %+v: %v", containerName, err.Error(), output)
	} else {
		log.Infof("Successfully removed %v", containerName)
	}
}

// This test cannot be run under Bazel so it is run with go test
func TestRuntimeBenchmarksWarn(t *testing.T) {
	defer cleanupContainer()
	err := runContainer("--privileged",
		"-v=/sys:/sys:shared",
		"-p=80:80",
		"--net=bridge",
		"--ipc=host",
		"--uts=host",
		"--security-opt=seccomp:unconfined",
		"--cgroup-parent=/foobar",
		"--userns=host",
		"--pid=host",
		"--device=/dev/temp_sda:/dev/temp_sda:rwm",
		"-v=/var/run/docker.sock:/var/run/docker.sock",
	)
	require.Nil(t, err)

	benchmarks := []utils.Check{
		containerruntime.NewAppArmorBenchmark(), // 5.1
		containerruntime.NewSELinuxBenchmark(),
		containerruntime.NewCapabilitiesBenchmark(),
		containerruntime.NewPrivilegedBenchmark(),
		containerruntime.NewSensitiveHostMountsBenchmark(), // 5.5
		containerruntime.NewSSHBenchmark(),
		containerruntime.NewPrivilegedPortsBenchmark(),
		containerruntime.NewNecessaryPortsBenchmark(),
		containerruntime.NewSharedNetworkBenchmark(),
		containerruntime.NewMemoryBenchmark(), // 5.10
		containerruntime.NewCPUPriorityBenchmark(),
		containerruntime.NewReadonlyRootfsBenchmark(),
		containerruntime.NewSpecificHostInterfaceBenchmark(),
		containerruntime.NewRestartPolicyBenchmark(),
		containerruntime.NewPidNamespaceBenchmark(), // 5.15
		containerruntime.NewIpcNamespaceBenchmark(),
		containerruntime.NewHostDevicesBenchmark(),
		containerruntime.NewUlimitBenchmark(),
		containerruntime.NewMountPropagationBenchmark(),
		containerruntime.NewUTSNamespaceBenchmark(), // 5.20
		containerruntime.NewSeccompBenchmark(),
		//NewPrivilegedDockerExecBenchmark(), // These check the audit logs for docker execs
		//NewUserDockerExecBenchmark(), // These check the audit logs for docker execs
		containerruntime.NewCgroupBenchmark(),
		containerruntime.NewAcquiringPrivilegesBenchmark(), // 5.25
		containerruntime.NewRuntimeHealthcheckBenchmark(),
		containerruntime.NewLatestImageBenchmark(),
		containerruntime.NewPidCgroupBenchmark(),
		containerruntime.NewBridgeNetworkBenchmark(),
		containerruntime.NewUsernsBenchmark(), // 5.30
		containerruntime.NewDockerSocketMountBenchmark(),
	}

	expectedResults := []v1.CheckStatus{
		v1.CheckStatus_WARN, // 1
		v1.CheckStatus_WARN,
		v1.CheckStatus_INFO,
		v1.CheckStatus_WARN,
		v1.CheckStatus_WARN, // 5
		v1.CheckStatus_NOTE,
		v1.CheckStatus_WARN,
		v1.CheckStatus_NOTE,
		v1.CheckStatus_PASS, // Cannot use both bridge and host network at the same time. Bridge removes port binding so allow host network test to pass
		v1.CheckStatus_WARN, // 10
		v1.CheckStatus_WARN,
		v1.CheckStatus_WARN,
		v1.CheckStatus_WARN,
		v1.CheckStatus_WARN,
		v1.CheckStatus_WARN, // 15
		v1.CheckStatus_WARN,
		v1.CheckStatus_WARN,
		v1.CheckStatus_NOTE,
		v1.CheckStatus_WARN,
		v1.CheckStatus_WARN, // 20
		v1.CheckStatus_WARN,
		//v1.CheckStatus_WARN, // Docker exec audits are commented out
		//v1.CheckStatus_WARN, // Docker exec audits are commented out
		v1.CheckStatus_WARN,
		v1.CheckStatus_WARN, // 25
		v1.CheckStatus_WARN,
		v1.CheckStatus_NOTE,
		v1.CheckStatus_WARN,
		v1.CheckStatus_WARN,
		v1.CheckStatus_WARN, // 30
		v1.CheckStatus_WARN,
	}
	require.Equal(t, len(benchmarks), len(expectedResults))

	// Set the containers manually to work around sync.Once
	containersRunning, containersAll, err := utils.GetContainers()
	require.Nil(t, err)
	utils.ContainersRunning = containersRunning
	utils.ContainersAll = containersAll

	// In order for the SELinux benchmark to see that SELinux has been enabled on dockerd
	// We set the configuration field explicitly
	utils.DockerConfig = make(map[string]utils.ConfigParams)
	if val := os.Getenv("CIRCLECI"); len(val) != 0 {
		t.Log("Daemon configuration cannot be accessed in CircleCI Docker-in-Docker")
	} else {
		err = utils.InitDockerConfig()
		require.Nil(t, err)
	}
	utils.DockerConfig["selinux-enabled"] = []string{""}
	defer func() {
		utils.DockerConfig = make(map[string]utils.ConfigParams)
	}()

	for i, container := range utils.ContainersRunning {
		if container.Name == containerName {
			utils.ContainersRunning = utils.ContainersRunning[i : i+1]
		}
	}
	for i, benchmark := range benchmarks {
		assert.Equal(t, benchmark.Run().Result, expectedResults[i], "Benchmark %v - %v has different results than expected",
			benchmark.Definition().Name,
			benchmark.Definition().Description,
		)
	}
}

func TestRuntimeBenchmarksPass(t *testing.T) {
	defer cleanupContainer()
	createNetwork()
	err := runContainer(
		"--cap-drop=NET_ADMIN",
		"--cap-drop=SYS_ADMIN",
		"--cap-drop=SYS_MODULE",
		"--health-cmd='stat /etc/passwd || exit 1'",
		"--pids-limit=10",
		"--security-opt=no-new-privileges",
		"--restart=on-failure:5",
		"--cpu-shares=1024",
		"--memory=104857600",
		"--read-only",
		"--net="+networkName)
	require.Nil(t, err)

	benchmarks := []utils.Check{
		containerruntime.NewAppArmorBenchmark(), // 5.1
		containerruntime.NewSELinuxBenchmark(),
		containerruntime.NewCapabilitiesBenchmark(),
		containerruntime.NewPrivilegedBenchmark(),
		containerruntime.NewSensitiveHostMountsBenchmark(), // 5.5
		containerruntime.NewSSHBenchmark(),
		containerruntime.NewPrivilegedPortsBenchmark(),
		containerruntime.NewNecessaryPortsBenchmark(),
		containerruntime.NewSharedNetworkBenchmark(),
		containerruntime.NewMemoryBenchmark(), // 5.10
		containerruntime.NewCPUPriorityBenchmark(),
		containerruntime.NewReadonlyRootfsBenchmark(),
		containerruntime.NewSpecificHostInterfaceBenchmark(),
		containerruntime.NewRestartPolicyBenchmark(),
		containerruntime.NewPidNamespaceBenchmark(), // 5.15
		containerruntime.NewIpcNamespaceBenchmark(),
		containerruntime.NewHostDevicesBenchmark(),
		containerruntime.NewUlimitBenchmark(),
		containerruntime.NewMountPropagationBenchmark(),
		containerruntime.NewUTSNamespaceBenchmark(), // 5.20
		containerruntime.NewSeccompBenchmark(),
		//NewPrivilegedDockerExecBenchmark(), // These are commented out because they require /var/log/audit/audit.log
		//NewUserDockerExecBenchmark(), // These are commented out because they require /var/log/audit/audit.log
		containerruntime.NewCgroupBenchmark(),
		containerruntime.NewAcquiringPrivilegesBenchmark(), // 5.25
		containerruntime.NewRuntimeHealthcheckBenchmark(),
		containerruntime.NewLatestImageBenchmark(),
		containerruntime.NewPidCgroupBenchmark(),
		containerruntime.NewBridgeNetworkBenchmark(),
		containerruntime.NewUsernsBenchmark(), // 5.30
		containerruntime.NewDockerSocketMountBenchmark(),
	}

	expectedResults := []v1.CheckStatus{
		v1.CheckStatus_PASS, // 1
		v1.CheckStatus_PASS,
		v1.CheckStatus_INFO,
		v1.CheckStatus_PASS,
		v1.CheckStatus_PASS, // 5
		v1.CheckStatus_NOTE,
		v1.CheckStatus_PASS,
		v1.CheckStatus_NOTE,
		v1.CheckStatus_PASS,
		v1.CheckStatus_PASS, // 10
		v1.CheckStatus_PASS,
		v1.CheckStatus_PASS,
		v1.CheckStatus_PASS,
		v1.CheckStatus_PASS,
		v1.CheckStatus_PASS, // 15
		v1.CheckStatus_PASS,
		v1.CheckStatus_PASS,
		v1.CheckStatus_NOTE,
		v1.CheckStatus_PASS,
		v1.CheckStatus_PASS, // 20
		v1.CheckStatus_PASS,
		// v1.CheckStatus_PASS, // Docker exec audits are commented out
		// v1.CheckStatus_PASS, // Docker exec audits are commented out
		v1.CheckStatus_PASS,
		v1.CheckStatus_PASS, // 25
		v1.CheckStatus_PASS,
		v1.CheckStatus_NOTE,
		v1.CheckStatus_PASS,
		v1.CheckStatus_PASS,
		v1.CheckStatus_PASS, // 30
		v1.CheckStatus_PASS,
	}
	require.Equal(t, len(benchmarks), len(expectedResults))
	// Set the containers manually to work around sync.Once
	containersRunning, containersAll, err := utils.GetContainers()
	require.Nil(t, err)
	utils.ContainersRunning = containersRunning
	utils.ContainersAll = containersAll

	for i, container := range utils.ContainersRunning {
		if container.Name == containerName {
			utils.ContainersRunning = utils.ContainersRunning[i : i+1]
		}
	}
	for i, benchmark := range benchmarks {
		result := benchmark.Run()
		assert.Equal(t, expectedResults[i], result.Result, "Benchmark %v - %v has different results than expected",
			benchmark.Definition().Name,
			benchmark.Definition().Description,
		)
		if result.Result == v1.CheckStatus_WARN {
			log.Infof("%+v", result.Notes)
		}
	}
}
