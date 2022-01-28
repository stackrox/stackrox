package clairify

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/scanners/clairify/mock"
	v1 "github.com/stackrox/scanner/generated/shared/api/v1"
	"github.com/stretchr/testify/assert"
)

func TestConvertNodeToVulnRequest(t *testing.T) {
	for _, testCase := range []struct {
		containerRuntime *storage.ContainerRuntimeInfo
		kernelVersion    string
		osImage          string
		kubeletVersion   string
		kubeProxyVersion string

		expected *v1.GetNodeVulnerabilitiesRequest
	}{
		{
			kernelVersion:    "3.10.0-1127.13.1.el7.x86_64",
			osImage:          "linux",
			kubeletVersion:   "v1.14.8",
			kubeProxyVersion: "v1.16.13-gke.401",
			containerRuntime: &storage.ContainerRuntimeInfo{
				Type:    storage.ContainerRuntime_DOCKER_CONTAINER_RUNTIME,
				Version: "19.3.5",
			},
			expected: &v1.GetNodeVulnerabilitiesRequest{
				KernelVersion:    "3.10.0-1127.13.1.el7.x86_64",
				OsImage:          "linux",
				KubeletVersion:   "v1.14.8",
				KubeproxyVersion: "v1.16.13-gke.401",
				Runtime: &v1.GetNodeVulnerabilitiesRequest_ContainerRuntime{
					Name:    "docker",
					Version: "19.3.5",
				},
			},
		},
		{
			containerRuntime: &storage.ContainerRuntimeInfo{
				Type:    storage.ContainerRuntime_CRIO_CONTAINER_RUNTIME,
				Version: "1.11.13-1.rhaos3.11.gitfb88a9c.el7",
			},
			expected: &v1.GetNodeVulnerabilitiesRequest{
				Runtime: &v1.GetNodeVulnerabilitiesRequest_ContainerRuntime{
					Name:    "cri-o",
					Version: "1.11.13-1.rhaos3.11.gitfb88a9c.el7",
				},
			},
		},
		{
			containerRuntime: &storage.ContainerRuntimeInfo{
				Type:    storage.ContainerRuntime_UNKNOWN_CONTAINER_RUNTIME,
				Version: "containerd://1.2.8",
			},
			expected: &v1.GetNodeVulnerabilitiesRequest{
				Runtime: &v1.GetNodeVulnerabilitiesRequest_ContainerRuntime{
					Name:    "containerd",
					Version: "1.2.8",
				},
			},
		},
	} {
		node := &storage.Node{
			ContainerRuntime: testCase.containerRuntime,
			KernelVersion:    testCase.kernelVersion,
			OsImage:          testCase.osImage,
			KubeletVersion:   testCase.kubeletVersion,
			KubeProxyVersion: testCase.kubeProxyVersion,
		}
		assert.Equal(t, testCase.expected, convertNodeToVulnRequest(node))
	}
}

func TestConvertVulnResponseToNodeScan(t *testing.T) {
	for _, testCase := range []struct {
		req  *v1.GetNodeVulnerabilitiesRequest
		resp *v1.GetNodeVulnerabilitiesResponse

		expected []*storage.EmbeddedNodeScanComponent
	}{
		{
			req: &v1.GetNodeVulnerabilitiesRequest{
				Runtime: &v1.GetNodeVulnerabilitiesRequest_ContainerRuntime{
					Name:    "docker",
					Version: "19.3.5",
				},
				KernelVersion:    "4.9.184-linuxkit",
				OsImage:          "linux",
				KubeletVersion:   "v1.16.13-gke.401",
				KubeproxyVersion: "v1.17.14-gke.400",
			},
			resp: &v1.GetNodeVulnerabilitiesResponse{
				KernelComponent: &v1.GetNodeVulnerabilitiesResponse_KernelComponent{
					Name:    "kernel",
					Version: "4.9.184-linuxkit",
				},
				RuntimeVulnerabilities: []*v1.Vulnerability{
					{
						Name:    "CVE-2020-0000",
						Link:    "link0",
						FixedBy: "0",
					},
				},
				KernelVulnerabilities: []*v1.Vulnerability{
					{
						Name:    "CVE-2020-1111",
						Link:    "link1",
						FixedBy: "1",
					},
				},
				KubeletVulnerabilities: []*v1.Vulnerability{
					{
						Name:    "CVE-2020-2222",
						Link:    "link2",
						FixedBy: "2",
					},
				},
				KubeproxyVulnerabilities: []*v1.Vulnerability{
					{
						Name:    "CVE-2020-3333",
						Link:    "link3",
						FixedBy: "3",
					},
					{
						Name:    "CVE-2020-4444",
						Link:    "link4",
						FixedBy: "4",
					},
				},
			},
			expected: []*storage.EmbeddedNodeScanComponent{
				{
					Name:    "docker",
					Version: "19.3.5",
					Vulns: []*storage.EmbeddedVulnerability{
						{
							Cve:  "CVE-2020-0000",
							Link: "link0",
							SetFixedBy: &storage.EmbeddedVulnerability_FixedBy{
								FixedBy: "0",
							},
							VulnerabilityType: storage.EmbeddedVulnerability_NODE_VULNERABILITY,
						},
					},
				},
				{
					Name:    "kernel",
					Version: "4.9.184-linuxkit",
					Vulns: []*storage.EmbeddedVulnerability{
						{
							Cve:  "CVE-2020-1111",
							Link: "link1",
							SetFixedBy: &storage.EmbeddedVulnerability_FixedBy{
								FixedBy: "1",
							},
							VulnerabilityType: storage.EmbeddedVulnerability_NODE_VULNERABILITY,
						},
					}},
				{
					Name:    "kubelet",
					Version: "v1.16.13-gke.401",
					Vulns: []*storage.EmbeddedVulnerability{
						{
							Cve:  "CVE-2020-2222",
							Link: "link2",
							SetFixedBy: &storage.EmbeddedVulnerability_FixedBy{
								FixedBy: "2",
							},
							VulnerabilityType: storage.EmbeddedVulnerability_NODE_VULNERABILITY,
						},
					}},
				{
					Name:    "kube-proxy",
					Version: "v1.17.14-gke.400",
					Vulns: []*storage.EmbeddedVulnerability{
						{
							Cve:  "CVE-2020-3333",
							Link: "link3",
							SetFixedBy: &storage.EmbeddedVulnerability_FixedBy{
								FixedBy: "3",
							},
							VulnerabilityType: storage.EmbeddedVulnerability_NODE_VULNERABILITY,
						},
						{
							Cve:  "CVE-2020-4444",
							Link: "link4",
							SetFixedBy: &storage.EmbeddedVulnerability_FixedBy{
								FixedBy: "4",
							},
							VulnerabilityType: storage.EmbeddedVulnerability_NODE_VULNERABILITY,
						},
					},
				},
			},
		},
	} {
		actual := convertVulnResponseToNodeScan(testCase.req, testCase.resp)
		assert.ElementsMatch(t, testCase.expected, actual.Components)
	}
}

func TestConvertNodeVulnerabilities(t *testing.T) {
	scannerVulns, protoVulns := mock.GetTestScannerVulns()
	for i := range scannerVulns {
		assert.Equal(t, protoVulns[i], convertVulnerability(&scannerVulns[i], storage.EmbeddedVulnerability_NODE_VULNERABILITY))
	}
}
