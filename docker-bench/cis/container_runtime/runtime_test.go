package containerruntime

import (
	"fmt"
	"log"
	"os"
	"testing"
	"time"

	"bitbucket.org/stack-rox/apollo/docker-bench/utils"
	"bitbucket.org/stack-rox/apollo/pkg/api/generated/api/v1"
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
	log.Printf("docker %+v", runCommands)
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
		log.Print(output)
	} else {
		log.Printf("Successfully created network %v", networkName)
	}
}

func cleanupContainer() {
	output, err := utils.CombinedOutput("docker", "kill", containerName)
	if err != nil {
		log.Printf("Error killing %v: %+v: %v", containerName, err.Error(), output)
	} else {
		log.Printf("Successfully killed %v", containerName)
	}

	output, err = utils.CombinedOutput("docker", "rm", containerName)
	if err != nil {
		log.Printf("Error removing %v: %+v: %v", containerName, err.Error(), output)
	} else {
		log.Printf("Successfully removed %v", containerName)
	}
}

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
		NewAppArmorBenchmark(), // 5.1
		NewSELinuxBenchmark(),
		NewCapabilitiesBenchmark(),
		NewPrivilegedBenchmark(),
		NewSensitiveHostMountsBenchmark(), // 5.5
		NewSSHBenchmark(),
		NewPrivilegedPortsBenchmark(),
		NewNecessaryPortsBenchmark(),
		NewSharedNetworkBenchmark(),
		NewMemoryBenchmark(), // 5.10
		NewCPUPriorityBenchmark(),
		NewReadonlyRootfsBenchmark(),
		NewSpecificHostInterfaceBenchmark(),
		NewRestartPolicyBenchmark(),
		NewPidNamespaceBenchmark(), // 5.15
		NewIpcNamespaceBenchmark(),
		NewHostDevicesBenchmark(),
		NewUlimitBenchmark(),
		NewMountPropagationBenchmark(),
		NewUTSNamespaceBenchmark(), // 5.20
		NewSeccompBenchmark(),
		//NewPrivilegedDockerExecBenchmark(), // These check the audit logs for docker execs
		//NewUserDockerExecBenchmark(), // These check the audit logs for docker execs
		NewCgroupBenchmark(),
		NewAcquiringPrivilegesBenchmark(), // 5.25
		NewRuntimeHealthcheckBenchmark(),
		NewLatestImageBenchmark(),
		NewPidCgroupBenchmark(),
		NewBridgeNetworkBenchmark(),
		NewUsernsBenchmark(), // 5.30
		NewDockerSocketMountBenchmark(),
	}

	expectedResults := []v1.CheckStatus{
		v1.CheckStatus_WARN, // 1
		v1.CheckStatus_WARN,
		v1.CheckStatus_WARN,
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
	utils.DockerConfig = make(map[string]utils.DockerConfigParams)
	if val := os.Getenv("CIRCLECI"); len(val) != 0 {
		t.Log("Daemon configuration cannot be accessed in CircleCI Docker-in-Docker")
	} else {
		err = utils.InitDockerConfig()
		require.Nil(t, err)
	}
	utils.DockerConfig["selinux-enabled"] = []string{""}
	defer func() {
		utils.DockerConfig = make(map[string]utils.DockerConfigParams)
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
		NewAppArmorBenchmark(), // 5.1
		NewSELinuxBenchmark(),
		NewCapabilitiesBenchmark(),
		NewPrivilegedBenchmark(),
		NewSensitiveHostMountsBenchmark(), // 5.5
		NewSSHBenchmark(),
		NewPrivilegedPortsBenchmark(),
		NewNecessaryPortsBenchmark(),
		NewSharedNetworkBenchmark(),
		NewMemoryBenchmark(), // 5.10
		NewCPUPriorityBenchmark(),
		NewReadonlyRootfsBenchmark(),
		NewSpecificHostInterfaceBenchmark(),
		NewRestartPolicyBenchmark(),
		NewPidNamespaceBenchmark(), // 5.15
		NewIpcNamespaceBenchmark(),
		NewHostDevicesBenchmark(),
		NewUlimitBenchmark(),
		NewMountPropagationBenchmark(),
		NewUTSNamespaceBenchmark(), // 5.20
		NewSeccompBenchmark(),
		//NewPrivilegedDockerExecBenchmark(), // These are commented out because they require /var/log/audit/audit.log
		//NewUserDockerExecBenchmark(), // These are commented out because they require /var/log/audit/audit.log
		NewCgroupBenchmark(),
		NewAcquiringPrivilegesBenchmark(), // 5.25
		NewRuntimeHealthcheckBenchmark(),
		NewLatestImageBenchmark(),
		NewPidCgroupBenchmark(),
		NewBridgeNetworkBenchmark(),
		NewUsernsBenchmark(), // 5.30
		NewDockerSocketMountBenchmark(),
	}

	expectedResults := []v1.CheckStatus{
		v1.CheckStatus_PASS, // 1
		v1.CheckStatus_PASS,
		v1.CheckStatus_PASS,
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
			log.Printf("%+v", result.Notes)
		}
	}
}
