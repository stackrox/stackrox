package command

import (
	"strings"
	"testing"

	"github.com/stackrox/stackrox/generated/internalapi/compliance"
	"github.com/stretchr/testify/assert"
)

const cmdLine = `/opt/k8s/bin/kubelet --kubeconfig=/etc/kubernetes/kubelet.conf --logtostderr=true --v=0 --container-runtime=docker --pod-manifest-path=/etc/kubernetes/manifests --hostname-override=hostname --tls-cert-file=/etc/kubernetes/pki/kubelet-crt.crt --tls-private-key-file=/etc/kubernetes/pki/kubelet-key.key --anonymous-auth=false --client-ca-file=/etc/kubernetes/pki/k8s-ca.crt --fail-swap-on=false --enable-test1 --feature-gates=A,B,C`

var (
	nullSeparatedCmdLine = strings.Replace(cmdLine, " ", "\x00", -1)
)

func TestGetCommandLineArgs(t *testing.T) {
	exec, args := getCommandLineArgs(nullSeparatedCmdLine)

	spaceSplit := strings.Split(cmdLine, " ")
	assert.Equal(t, spaceSplit[0], exec)
	assert.Equal(t, spaceSplit[1:], args)
}

func sliceFromString(s string) []string {
	return []string{s}
}

func TestParseArgs(t *testing.T) {
	_, args := getCommandLineArgs(nullSeparatedCmdLine)

	expectedArgs := []*compliance.CommandLine_Args{
		{
			Key:    "kubeconfig",
			Values: sliceFromString("/etc/kubernetes/kubelet.conf"),
		},
		{
			Key:    "logtostderr",
			Values: sliceFromString("true"),
		},
		{
			Key:    "v",
			Values: sliceFromString("0"),
		},
		{
			Key:    "container-runtime",
			Values: sliceFromString("docker"),
		},
		{
			Key:    "pod-manifest-path",
			Values: sliceFromString("/etc/kubernetes/manifests"),
		},
		{
			Key:    "hostname-override",
			Values: sliceFromString("hostname"),
		},
		{
			Key:    "tls-cert-file",
			Values: sliceFromString("/etc/kubernetes/pki/kubelet-crt.crt"),
		},
		{
			Key:    "tls-private-key-file",
			Values: sliceFromString("/etc/kubernetes/pki/kubelet-key.key"),
		},
		{
			Key:    "anonymous-auth",
			Values: sliceFromString("false"),
		},
		{
			Key:    "client-ca-file",
			Values: sliceFromString("/etc/kubernetes/pki/k8s-ca.crt"),
		},
		{
			Key:    "fail-swap-on",
			Values: sliceFromString("false"),
		},
		{
			Key: "enable-test1",
		},
		{
			Key:    "feature-gates",
			Values: []string{"A", "B", "C"},
		},
	}
	parsedArgs := parseArgs(args)
	assert.Equal(t, expectedArgs, parsedArgs)
}

func TestGetProcessFromCmdLineBytes(t *testing.T) {
	cases := []struct {
		cmdline         string
		expectedProcess string
	}{
		{
			cmdline:         "",
			expectedProcess: "",
		},
		{
			cmdline:         "dockerd",
			expectedProcess: "dockerd",
		},
		{
			cmdline:         "/usr/local/bin/dockerd",
			expectedProcess: "dockerd",
		},
		{
			cmdline:         "/usr/local/bin/dockerd\x00",
			expectedProcess: "dockerd",
		},
		{
			cmdline:         "/usr/local/bin/dockerd\x00abc\x00def",
			expectedProcess: "dockerd",
		},
		{
			cmdline:         "/usr/local/bin/kube-controller-manager\x00abc\x00def",
			expectedProcess: "kube-controller-manager",
		},
	}
	for _, c := range cases {
		t.Run(c.cmdline, func(t *testing.T) {
			assert.Equal(t, c.expectedProcess, getProcessFromCmdLineBytes([]byte(c.cmdline)))
		})
	}
}
