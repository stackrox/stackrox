package common

import (
	"testing"

	"github.com/docker/docker/daemon"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDockerConfigGet(t *testing.T) {
	config := make(Config)

	// Look for something in empty map
	var expectedParams DockerConfigParams
	actualParams, found := config.Get("hello")
	assert.False(t, found)
	assert.Equal(t, expectedParams, actualParams)

	config = make(Config)
	expectedParams = DockerConfigParams{"hi"}
	config["hello"] = expectedParams
	actualParams, found = config.Get("hello")
	assert.True(t, found)
	assert.Equal(t, expectedParams, actualParams)

	config = make(Config)
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
	d := &daemon.Config{
		CgroupParent:         "cgroup",
		EnableSelinuxSupport: true,
		OOMScoreAdjust:       55,
		CommonConfig: daemon.CommonConfig{
			ClusterOpts: map[string]string{
				"opt1-key": "opt1-value",
			},
		},
	}
	configMap := make(map[string]DockerConfigParams)
	walkStruct(configMap, d)
	var expectedMap = map[string]DockerConfigParams{
		"cgroup-parent":      {"cgroup"},
		"selinux-enabled":    {"true"},
		"oom-score-adjust":   {"55"},
		"cluster-store-opts": {"opt1-key=opt1-value"},
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
	config := make(Config)
	skip := parseArg(config, "--security-opt", "seccomp") // Use the next element due to space in commandline
	assert.Equal(t, Config{"security-opt": []string{"seccomp"}}, config)
	assert.True(t, skip)

	config = make(Config)
	skip = parseArg(config, "--security-opt=seccomp", "") // Use the next element due to space in commandline
	assert.Equal(t, Config{"security-opt": []string{"seccomp"}}, config)
	assert.False(t, skip)

	config = make(Config)
	skip = parseArg(config, "--no-new-privileges", "--selinux-enabled") // Use the next element due to space in commandline
	assert.Equal(t, Config{"no-new-privileges": []string{""}}, config)
	assert.False(t, skip)

	config = make(Config)
	skip = parseArg(config, "--no-new-privileges", "") // Use the next element due to space in commandline
	assert.Equal(t, Config{"no-new-privileges": []string{""}}, config)
}

func TestParseArgs(t *testing.T) {
	args := []string{}
	config := make(Config)
	parseArgs(config, args)
	assert.Equal(t, Config{}, config)

	config = make(Config)
	args = []string{"--security-opt", "seccomp", "--security-opt=apparmor", "--no-new-privileges", "--selinux-enabled"}
	expectedConfig := Config{
		"security-opt":      []string{"seccomp", "apparmor"},
		"no-new-privileges": []string{""},
		"selinux-enabled":   []string{""},
	}
	parseArgs(config, args)
	assert.Equal(t, expectedConfig, config)

	config = make(Config)
	args = []string{"--security-opt", "seccomp", "--security-opt=apparmor", "--no-new-privileges", "true"}
	expectedConfig = Config{
		"security-opt":      []string{"seccomp", "apparmor"},
		"no-new-privileges": []string{"true"},
	}
	parseArgs(config, args)
	assert.Equal(t, expectedConfig, config)
}
