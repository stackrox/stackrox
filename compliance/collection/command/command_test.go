package command

import (
	"strings"
	"testing"

	"github.com/stackrox/rox/generated/internalapi/compliance"
	"github.com/stackrox/rox/pkg/protoassert"
	"github.com/stretchr/testify/assert"
)

const cmdLine = `/opt/k8s/bin/kubelet --kubeconfig=/etc/kubernetes/kubelet.conf --logtostderr=true --v=0 --container-runtime=docker --pod-manifest-path=/etc/kubernetes/manifests --hostname-override=hostname --tls-cert-file=/etc/kubernetes/pki/kubelet-crt.crt --tls-private-key-file=/etc/kubernetes/pki/kubelet-key.key --anonymous-auth=false --client-ca-file=/etc/kubernetes/pki/k8s-ca.crt --fail-swap-on=false --enable-test1 --feature-gates=A,B,C`

var (
	nullSeparatedCmdLine = strings.ReplaceAll(cmdLine, " ", "\x00")
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
		compliance.CommandLine_Args_builder{
			Key:    "kubeconfig",
			Values: sliceFromString("/etc/kubernetes/kubelet.conf"),
		}.Build(),
		compliance.CommandLine_Args_builder{
			Key:    "logtostderr",
			Values: sliceFromString("true"),
		}.Build(),
		compliance.CommandLine_Args_builder{
			Key:    "v",
			Values: sliceFromString("0"),
		}.Build(),
		compliance.CommandLine_Args_builder{
			Key:    "container-runtime",
			Values: sliceFromString("docker"),
		}.Build(),
		compliance.CommandLine_Args_builder{
			Key:    "pod-manifest-path",
			Values: sliceFromString("/etc/kubernetes/manifests"),
		}.Build(),
		compliance.CommandLine_Args_builder{
			Key:    "hostname-override",
			Values: sliceFromString("hostname"),
		}.Build(),
		compliance.CommandLine_Args_builder{
			Key:    "tls-cert-file",
			Values: sliceFromString("/etc/kubernetes/pki/kubelet-crt.crt"),
		}.Build(),
		compliance.CommandLine_Args_builder{
			Key:    "tls-private-key-file",
			Values: sliceFromString("/etc/kubernetes/pki/kubelet-key.key"),
		}.Build(),
		compliance.CommandLine_Args_builder{
			Key:    "anonymous-auth",
			Values: sliceFromString("false"),
		}.Build(),
		compliance.CommandLine_Args_builder{
			Key:    "client-ca-file",
			Values: sliceFromString("/etc/kubernetes/pki/k8s-ca.crt"),
		}.Build(),
		compliance.CommandLine_Args_builder{
			Key:    "fail-swap-on",
			Values: sliceFromString("false"),
		}.Build(),
		compliance.CommandLine_Args_builder{
			Key: "enable-test1",
		}.Build(),
		compliance.CommandLine_Args_builder{
			Key:    "feature-gates",
			Values: []string{"A", "B", "C"},
		}.Build(),
	}
	parsedArgs := parseArgs(args)
	protoassert.SlicesEqual(t, expectedArgs, parsedArgs)
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
