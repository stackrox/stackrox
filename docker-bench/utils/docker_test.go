package utils

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDockerConfigGet(t *testing.T) {
	config := make(FlattenedDockerConfig)

	// Look for something in empty map
	var expectedParams DockerConfigParams
	actualParams, found := config.Get("hello")
	assert.False(t, found)
	assert.Equal(t, expectedParams, actualParams)

	config = make(FlattenedDockerConfig)
	expectedParams = DockerConfigParams{"hi"}
	config["hello"] = expectedParams
	actualParams, found = config.Get("hello")
	assert.True(t, found)
	assert.Equal(t, expectedParams, actualParams)

	config = make(FlattenedDockerConfig)
	expectedParams = DockerConfigParams{"hi"}
	config["hellos"] = expectedParams
	actualParams, found = config.Get("hello")
	assert.True(t, found)
	assert.Equal(t, expectedParams, actualParams)
}

func TestGetPID(t *testing.T) {
	pid, err := getPID("init")
	assert.Nil(t, err)
	assert.Equal(t, 1, pid)

	pid, err = getPID("howdy")
	assert.NotNil(t, err)
}

func TestGetProcessPID(t *testing.T) {
	processes := []string{"howdy", "init", "blah"}
	pid, name, err := getProcessPID(processes)
	assert.Nil(t, err)
	assert.Equal(t, 1, pid)
	assert.Equal(t, "init", name)

	processes = []string{"howdy"}
	_, _, err = getProcessPID(processes)
	assert.NotNil(t, err)
}

func TestGetCommandLine(t *testing.T) {
	cmdline, err := getCommandLine(1)
	require.Nil(t, err)
	assert.Contains(t, cmdline, "init")
}

func TestGetTagValue(t *testing.T) {
	tag, valid := getTagValue("hello")
	assert.True(t, valid)
	assert.Equal(t, "hello", tag)

	tag, valid = getTagValue("hello,omitempty")
	assert.True(t, valid)
	assert.Equal(t, "hello", tag)

	_, valid = getTagValue("-")
	assert.False(t, valid)

	_, valid = getTagValue("")
	assert.False(t, valid)
}

func TestWalkStruct(t *testing.T) {
	d := &Config{
		CgroupParent:         "cgroup",
		EnableSelinuxSupport: true,
		OOMScoreAdjust:       55,
		CommonConfig: CommonConfig{
			ClusterOpts: map[string]string{
				"opt1key": "opt1value",
			},
		},
	}
	configMap := make(map[string]DockerConfigParams)
	walkStruct(configMap, d)
	var expectedMap = map[string]DockerConfigParams{
		"cgroup-parent":      {"cgroup"},
		"selinux-enabled":    {"true"},
		"oom-score-adjust":   {"55"},
		"cluster-store-opts": {"opt1key=opt1value"},
	}
	for k, v := range expectedMap {
		assert.Equal(t, v, configMap[k])
	}
}

func TestDockerConfigParamsMatches(t *testing.T) {
	params := DockerConfigParams{"howdy", "hello"}
	assert.True(t, params.Matches("hello"))

	assert.False(t, params.Matches("hey"))
	assert.False(t, params.Matches("owdy"))
}

func TestDockerConfigParamsContains(t *testing.T) {
	params := DockerConfigParams{"howdy", "hello"}

	fullValue, exists := params.Contains("hello")
	assert.True(t, exists)
	assert.Equal(t, fullValue, "hello")

	fullValue, exists = params.Contains("owdy")
	assert.True(t, exists)
	assert.Equal(t, fullValue, "howdy")

	_, exists = params.Contains("hey")
	assert.False(t, exists)
}

func TestGetCommandLineArgs(t *testing.T) {
	processName := "dockerd"
	commandLine := processName + string(0x00) + "the" + string(0x00) + "quick" + string(0x00) + "brown" + string(0x00)

	expectedArgs := []string{"the", "quick", "brown"}
	args := getCommandLineArgs(commandLine, processName)
	assert.Equal(t, expectedArgs, args)
}

func TestGetKeyValueFromArg(t *testing.T) {
	k, v := getKeyValueFromArg("hello")
	assert.Equal(t, "hello", k)
	assert.Equal(t, "", v)

	k, v = getKeyValueFromArg("hello=")
	assert.Equal(t, "hello", k)
	assert.Equal(t, "", v)

	k, v = getKeyValueFromArg("hello=hey")
	assert.Equal(t, "hello", k)
	assert.Equal(t, "hey", v)

	k, v = getKeyValueFromArg("")
	assert.Equal(t, "", k)
	assert.Equal(t, "", v)
}

func TestGetExpandedKey(t *testing.T) {
	key := getExpandedKey("hello")
	assert.Equal(t, "hello", key)

	key = getExpandedKey("-D")
	assert.Equal(t, "debug", key)

	key = getExpandedKey("--hello")
	assert.Equal(t, "hello", key)

	key = getExpandedKey("--hello--hey")
	assert.Equal(t, "hello--hey", key)
}

func TestParseArg(t *testing.T) {
	config := make(FlattenedDockerConfig)
	skip := parseArg(config, "--security-opt", "seccomp") // Use the next element due to space in commandline
	assert.Equal(t, FlattenedDockerConfig{"security-opt": []string{"seccomp"}}, config)
	assert.True(t, skip)

	config = make(FlattenedDockerConfig)
	skip = parseArg(config, "--security-opt=seccomp", "") // Use the next element due to space in commandline
	assert.Equal(t, FlattenedDockerConfig{"security-opt": []string{"seccomp"}}, config)
	assert.False(t, skip)

	config = make(FlattenedDockerConfig)
	skip = parseArg(config, "--no-new-privileges", "--selinux-enabled") // Use the next element due to space in commandline
	assert.Equal(t, FlattenedDockerConfig{"no-new-privileges": []string{""}}, config)
	assert.False(t, skip)

	config = make(FlattenedDockerConfig)
	skip = parseArg(config, "--no-new-privileges", "") // Use the next element due to space in commandline
	assert.Equal(t, FlattenedDockerConfig{"no-new-privileges": []string{""}}, config)
}

func TestParseArgs(t *testing.T) {
	args := []string{}
	config := make(FlattenedDockerConfig)
	parseArgs(config, args)
	assert.Equal(t, FlattenedDockerConfig{}, config)

	config = make(FlattenedDockerConfig)
	args = []string{"--security-opt", "seccomp", "--security-opt=apparmor", "--no-new-privileges", "--selinux-enabled"}
	expectedConfig := FlattenedDockerConfig{
		"security-opt":      []string{"seccomp", "apparmor"},
		"no-new-privileges": []string{""},
		"selinux-enabled":   []string{""},
	}
	parseArgs(config, args)
	assert.Equal(t, expectedConfig, config)

	config = make(FlattenedDockerConfig)
	args = []string{"--security-opt", "seccomp", "--security-opt=apparmor", "--no-new-privileges", "true"}
	expectedConfig = FlattenedDockerConfig{
		"security-opt":      []string{"seccomp", "apparmor"},
		"no-new-privileges": []string{"true"},
	}
	parseArgs(config, args)
	assert.Equal(t, expectedConfig, config)
}

const dockerConfigFile = `
{
	"authorization-plugins": [],
	"data-root": "",
	"dns": [],
	"dns-opts": [],
	"dns-search": [],
	"exec-opts": [],
	"exec-root": "",
	"experimental": false,
	"storage-driver": "",
	"storage-opts": [],
	"labels": [],
	"live-restore": true,
	"log-driver": "",
	"log-opts": {},
	"mtu": 0,
	"pidfile": "",
	"cluster-store": "",
	"cluster-store-opts": {},
	"cluster-advertise": "",
	"max-concurrent-downloads": 3,
	"max-concurrent-uploads": 5,
	"default-shm-size": "64M",
	"shutdown-timeout": 15,
	"debug": true,
	"hosts": [],
	"log-level": "",
	"tls": true,
	"tlsverify": true,
	"tlscacert": "",
	"tlscert": "",
	"tlskey": "",
	"swarm-default-advertise-addr": "",
	"api-cors-header": "",
	"selinux-enabled": false,
	"userns-remap": "",
	"group": "",
	"cgroup-parent": "",
	"default-ulimits": {},
	"init": false,
	"init-path": "/usr/libexec/docker-init",
	"ipv6": false,
	"iptables": false,
	"ip-forward": false,
	"ip-masq": false,
	"userland-proxy": false,
	"userland-proxy-path": "/usr/libexec/docker-proxy",
	"ip": "0.0.0.0",
	"bridge": "",
	"bip": "",
	"fixed-cidr": "",
	"fixed-cidr-v6": "",
	"default-gateway": "",
	"default-gateway-v6": "",
	"icc": false,
	"raw-logs": false,
	"allow-nondistributable-artifacts": [],
	"registry-mirrors": [],
	"seccomp-profile": "",
	"insecure-registries": [],
	"disable-legacy-registry": false,
	"no-new-privileges": false,
	"default-runtime": "runc",
	"oom-score-adjust": -500,
	"runtimes": {
		"runc": {
			"path": "runc"
		},
		"custom": {
			"path": "/usr/local/bin/my-runc-replacement",
			"runtimeArgs": [
				"--debug"
			]
		}
	}
}
`

func TestDockerConfigFile(t *testing.T) {
	f, err := ioutil.TempFile("", "")
	require.Nil(t, err)
	defer os.Remove(f.Name())
	defer f.Close()

	_, err = f.Write([]byte(dockerConfigFile))
	require.Nil(t, err)

	m := make(map[string]DockerConfigParams)
	err = getDockerConfigFromFile(f.Name(), m)
	require.Nil(t, err)

	assert.Contains(t, m["oom-score-adjust"], "-500")
	assert.Contains(t, m["userland-proxy-path"], "/usr/libexec/docker-proxy")
}
