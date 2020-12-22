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
		operatingSystem  string
		kubeletVersion   string
		kubeProxyVersion string

		expected *v1.GetVulnerabilitiesRequest
	}{
		{
			containerRuntime: &storage.ContainerRuntimeInfo{
				Type:    storage.ContainerRuntime_DOCKER_CONTAINER_RUNTIME,
				Version: "19.3.5",
			},
			expected: &v1.GetVulnerabilitiesRequest{
				Components: []*v1.Component{
					{
						Component: &v1.Component_AppComponent{
							AppComponent: &v1.ApplicationComponent{
								Vendor:  "docker",
								Product: "docker",
								Version: "19.3.5",
							},
						},
					},
				},
			},
		},
		{
			containerRuntime: &storage.ContainerRuntimeInfo{
				Type:    storage.ContainerRuntime_CRIO_CONTAINER_RUNTIME,
				Version: "1.11.13-1.rhaos3.11.gitfb88a9c.el7",
			},
			expected: &v1.GetVulnerabilitiesRequest{
				Components: []*v1.Component{
					{
						Component: &v1.Component_AppComponent{
							AppComponent: &v1.ApplicationComponent{
								Vendor:  "kubernetes",
								Product: "cri-o",
								Version: "1.11.13-1.rhaos3.11.gitfb88a9c.el7",
							},
						},
					},
				},
			},
		},
		{
			containerRuntime: &storage.ContainerRuntimeInfo{
				Type:    storage.ContainerRuntime_UNKNOWN_CONTAINER_RUNTIME,
				Version: "containerd://1.2.8",
			},
			expected: &v1.GetVulnerabilitiesRequest{
				Components: []*v1.Component{
					{
						Component: &v1.Component_AppComponent{
							AppComponent: &v1.ApplicationComponent{
								Vendor:  "linuxfoundation",
								Product: "containerd",
								Version: "1.2.8",
							},
						},
					},
				},
			},
		},
		{
			kernelVersion:   "3.10.0-1127.13.1.el7.x86_64",
			operatingSystem: "linux",
			expected: &v1.GetVulnerabilitiesRequest{
				Components: []*v1.Component{
					{
						Component: &v1.Component_AppComponent{
							AppComponent: &v1.ApplicationComponent{
								Vendor:  "linux",
								Product: "linux_kernel",
								Version: "3.10.0-1127.13.1.el7.x86_64",
							},
						},
					},
				},
			},
		},
		{
			kernelVersion:   "4.19.112+",
			operatingSystem: "linux",
			expected: &v1.GetVulnerabilitiesRequest{
				Components: []*v1.Component{
					{
						Component: &v1.Component_AppComponent{
							AppComponent: &v1.ApplicationComponent{
								Vendor:  "linux",
								Product: "linux_kernel",
								Version: "4.19.112+",
							},
						},
					},
				},
			},
		},
		{
			kernelVersion:   "5.4.0-1027-gke",
			operatingSystem: "linux",
			expected: &v1.GetVulnerabilitiesRequest{
				Components: []*v1.Component{
					{
						Component: &v1.Component_AppComponent{
							AppComponent: &v1.ApplicationComponent{
								Vendor:  "linux",
								Product: "linux_kernel",
								Version: "5.4.0-1027-gke",
							},
						},
					},
				},
			},
		},
		{
			kernelVersion:   "4.14.203-156.332.amzn2.x86_64",
			operatingSystem: "linux",
			expected: &v1.GetVulnerabilitiesRequest{
				Components: []*v1.Component{
					{
						Component: &v1.Component_AppComponent{
							AppComponent: &v1.ApplicationComponent{
								Vendor:  "linux",
								Product: "linux_kernel",
								Version: "4.14.203-156.332.amzn2.x86_64",
							},
						},
					},
				},
			},
		},
		{
			kernelVersion:   "4.14.203",
			operatingSystem: "notlinux",
			expected: &v1.GetVulnerabilitiesRequest{
				Components: []*v1.Component{},
			},
		},
		{
			kubeletVersion: "v1.14.8",
			expected: &v1.GetVulnerabilitiesRequest{
				Components: []*v1.Component{
					{
						Component: &v1.Component_K8SComponent{
							K8SComponent: &v1.KubernetesComponent{
								Component: v1.KubernetesComponent_KUBELET,
								Version:   "v1.14.8",
							},
						},
					},
				},
			},
		},
		{
			kubeletVersion: "v1.11.0+d4cacc0",
			expected: &v1.GetVulnerabilitiesRequest{
				Components: []*v1.Component{
					{
						Component: &v1.Component_K8SComponent{
							K8SComponent: &v1.KubernetesComponent{
								Component: v1.KubernetesComponent_KUBELET,
								Version:   "v1.11.0+d4cacc0",
							},
						},
					},
				},
			},
		},
		{
			kubeProxyVersion: "v1.16.13-gke.401",
			expected: &v1.GetVulnerabilitiesRequest{
				Components: []*v1.Component{
					{
						Component: &v1.Component_K8SComponent{
							K8SComponent: &v1.KubernetesComponent{
								Component: v1.KubernetesComponent_KUBE_PROXY,
								Version:   "v1.16.13-gke.401",
							},
						},
					},
				},
			},
		},
		{
			kubeProxyVersion: "v1.17.12-eks-7684af",
			expected: &v1.GetVulnerabilitiesRequest{
				Components: []*v1.Component{
					{
						Component: &v1.Component_K8SComponent{
							K8SComponent: &v1.KubernetesComponent{
								Component: v1.KubernetesComponent_KUBE_PROXY,
								Version:   "v1.17.12-eks-7684af",
							},
						},
					},
				},
			},
		},
	} {
		node := &storage.Node{
			ContainerRuntime: testCase.containerRuntime,
			KernelVersion:    testCase.kernelVersion,
			OperatingSystem:  testCase.operatingSystem,
			KubeletVersion:   testCase.kubeletVersion,
			KubeProxyVersion: testCase.kubeProxyVersion,
		}
		assert.Equal(t, testCase.expected, convertNodeToVulnRequest(node))
	}
}

func TestConvertVulnResponseToNodeScan(t *testing.T) {
	for _, testCase := range []struct {
		node *storage.Node
		resp *v1.GetVulnerabilitiesResponse

		expected []*storage.EmbeddedNodeScanComponent
	}{
		{
			node: &storage.Node{
				ContainerRuntime: &storage.ContainerRuntimeInfo{
					Type:    storage.ContainerRuntime_DOCKER_CONTAINER_RUNTIME,
					Version: "19.3.5",
				},
				KernelVersion:    "4.9.184-linuxkit",
				OperatingSystem:  "linux",
				KubeletVersion:   "v1.16.13-gke.401",
				KubeProxyVersion: "v1.17.14-gke.400",
			},
			resp: &v1.GetVulnerabilitiesResponse{
				VulnerabilitiesByComponent: []*v1.ComponentWithVulns{
					{
						Component: &v1.Component{
							Component: &v1.Component_AppComponent{
								AppComponent: &v1.ApplicationComponent{
									Vendor:  "docker",
									Product: "docker",
									Version: "19.3.5",
								},
							},
						},
						Vulnerabilities: []*v1.Vulnerability{
							{
								Name:    "CVE-2020-1234",
								FixedBy: "19.4.0",
							},
						},
					},
					{
						Component: &v1.Component{
							Component: &v1.Component_AppComponent{
								AppComponent: &v1.ApplicationComponent{
									Vendor:  "linux",
									Product: "linux_kernel",
									Version: "4.9.184",
								},
							},
						},
						Vulnerabilities: []*v1.Vulnerability{},
					},
					{
						Component: &v1.Component{
							Component: &v1.Component_K8SComponent{
								K8SComponent: &v1.KubernetesComponent{
									Component: v1.KubernetesComponent_KUBELET,
									Version:   "1.16.13",
								},
							},
						},
						Vulnerabilities: []*v1.Vulnerability{},
					},
					{
						Component: &v1.Component{
							Component: &v1.Component_K8SComponent{
								K8SComponent: &v1.KubernetesComponent{
									Component: v1.KubernetesComponent_KUBE_PROXY,
									Version:   "1.17.14",
								},
							},
						},
						Vulnerabilities: []*v1.Vulnerability{},
					},
				},
			},
			expected: []*storage.EmbeddedNodeScanComponent{
				{
					Name:    "docker",
					Version: "19.3.5",
					Vulns: []*storage.EmbeddedVulnerability{
						{
							Cve:  "CVE-2020-1234",
							Link: "https://nvd.nist.gov/vuln/detail/CVE-2020-1234",
							SetFixedBy: &storage.EmbeddedVulnerability_FixedBy{
								FixedBy: "19.4.0",
							},
							VulnerabilityType: storage.EmbeddedVulnerability_NODE_VULNERABILITY,
						},
					},
				},
				{
					Name:    "linux kernel",
					Version: "4.9.184-linuxkit",
					Vulns:   []*storage.EmbeddedVulnerability{},
				},
				{
					Name:    "kubelet",
					Version: "v1.16.13-gke.401",
					Vulns:   []*storage.EmbeddedVulnerability{},
				},
				{
					Name:    "kube-proxy",
					Version: "v1.17.14-gke.400",
					Vulns:   []*storage.EmbeddedVulnerability{},
				},
			},
		},
		{
			node: &storage.Node{
				ContainerRuntime: &storage.ContainerRuntimeInfo{
					Type:    storage.ContainerRuntime_CRIO_CONTAINER_RUNTIME,
					Version: "1.11.13-1.rhaos3.11.gitfb88a9c.el7",
				},
				KernelVersion:    "5.4.0-1027-gke",
				OperatingSystem:  "linux",
				KubeletVersion:   "v1.11.0+d4cacc0",
				KubeProxyVersion: "v1.17.12-eks-7684af",
			},
			resp: &v1.GetVulnerabilitiesResponse{
				VulnerabilitiesByComponent: []*v1.ComponentWithVulns{
					{
						Component: &v1.Component{
							Component: &v1.Component_AppComponent{
								AppComponent: &v1.ApplicationComponent{
									Vendor:  "kubernetes",
									Product: "cri-o",
									Version: "1.11.13",
								},
							},
						},
						Vulnerabilities: []*v1.Vulnerability{
							{
								Name:    "CVE-2020-1234",
								FixedBy: "19.4.0",
							},
						},
					},
					{
						Component: &v1.Component{
							Component: &v1.Component_AppComponent{
								AppComponent: &v1.ApplicationComponent{
									Vendor:  "linux",
									Product: "linux_kernel",
									Version: "5.4.0",
								},
							},
						},
						Vulnerabilities: []*v1.Vulnerability{},
					},
					{
						Component: &v1.Component{
							Component: &v1.Component_K8SComponent{
								K8SComponent: &v1.KubernetesComponent{
									Component: v1.KubernetesComponent_KUBELET,
									Version:   "1.11.0",
								},
							},
						},
						Vulnerabilities: []*v1.Vulnerability{},
					},
					{
						Component: &v1.Component{
							Component: &v1.Component_K8SComponent{
								K8SComponent: &v1.KubernetesComponent{
									Component: v1.KubernetesComponent_KUBE_PROXY,
									Version:   "1.17.12",
								},
							},
						},
						Vulnerabilities: []*v1.Vulnerability{},
					},
				},
			},
			expected: []*storage.EmbeddedNodeScanComponent{
				{
					Name:    "cri-o",
					Version: "1.11.13-1.rhaos3.11.gitfb88a9c.el7",
					Vulns: []*storage.EmbeddedVulnerability{
						{
							Cve:  "CVE-2020-1234",
							Link: "https://nvd.nist.gov/vuln/detail/CVE-2020-1234",
							SetFixedBy: &storage.EmbeddedVulnerability_FixedBy{
								FixedBy: "19.4.0",
							},
							VulnerabilityType: storage.EmbeddedVulnerability_NODE_VULNERABILITY,
						},
					},
				},
				{
					Name:    "linux kernel",
					Version: "5.4.0-1027-gke",
					Vulns:   []*storage.EmbeddedVulnerability{},
				},
				{
					Name:    "kubelet",
					Version: "v1.11.0+d4cacc0",
					Vulns:   []*storage.EmbeddedVulnerability{},
				},
				{
					Name:    "kube-proxy",
					Version: "v1.17.12-eks-7684af",
					Vulns:   []*storage.EmbeddedVulnerability{},
				},
			},
		},
		{
			node: &storage.Node{
				ContainerRuntime: &storage.ContainerRuntimeInfo{
					Type:    storage.ContainerRuntime_UNKNOWN_CONTAINER_RUNTIME,
					Version: "containerd://1.2.8",
				},
				KernelVersion:    "4.14.203-156.332.amzn2.x86_64",
				OperatingSystem:  "linux",
				KubeletVersion:   "v1.14.8",
				KubeProxyVersion: "v1.14.8",
			},
			resp: &v1.GetVulnerabilitiesResponse{
				VulnerabilitiesByComponent: []*v1.ComponentWithVulns{
					{
						Component: &v1.Component{
							Component: &v1.Component_AppComponent{
								AppComponent: &v1.ApplicationComponent{
									Vendor:  "linuxfoundation",
									Product: "containerd",
									Version: "1.2.8",
								},
							},
						},
						Vulnerabilities: []*v1.Vulnerability{
							{
								Name:    "CVE-2020-1234",
								FixedBy: "19.4.0",
							},
						},
					},
					{
						Component: &v1.Component{
							Component: &v1.Component_AppComponent{
								AppComponent: &v1.ApplicationComponent{
									Vendor:  "linux",
									Product: "linux_kernel",
									Version: "4.14.203",
								},
							},
						},
						Vulnerabilities: []*v1.Vulnerability{},
					},
					{
						Component: &v1.Component{
							Component: &v1.Component_K8SComponent{
								K8SComponent: &v1.KubernetesComponent{
									Component: v1.KubernetesComponent_KUBELET,
									Version:   "1.14.8",
								},
							},
						},
						Vulnerabilities: []*v1.Vulnerability{},
					},
					{
						Component: &v1.Component{
							Component: &v1.Component_K8SComponent{
								K8SComponent: &v1.KubernetesComponent{
									Component: v1.KubernetesComponent_KUBE_PROXY,
									Version:   "1.14.8",
								},
							},
						},
						Vulnerabilities: []*v1.Vulnerability{},
					},
				},
			},
			expected: []*storage.EmbeddedNodeScanComponent{
				{
					Name:    "containerd",
					Version: "1.2.8",
					Vulns: []*storage.EmbeddedVulnerability{
						{
							Cve:  "CVE-2020-1234",
							Link: "https://nvd.nist.gov/vuln/detail/CVE-2020-1234",
							SetFixedBy: &storage.EmbeddedVulnerability_FixedBy{
								FixedBy: "19.4.0",
							},
							VulnerabilityType: storage.EmbeddedVulnerability_NODE_VULNERABILITY,
						},
					},
				},
				{
					Name:    "linux kernel",
					Version: "4.14.203-156.332.amzn2.x86_64",
					Vulns:   []*storage.EmbeddedVulnerability{},
				},
				{
					Name:    "kubelet",
					Version: "v1.14.8",
					Vulns:   []*storage.EmbeddedVulnerability{},
				},
				{
					Name:    "kube-proxy",
					Version: "v1.14.8",
					Vulns:   []*storage.EmbeddedVulnerability{},
				},
			},
		},
	} {
		actual := convertVulnResponseToNodeScan(testCase.node, testCase.resp)
		assert.Equal(t, testCase.expected, actual.Components)
	}
}

func TestConvertNodeVulnerabilities(t *testing.T) {
	scannerVulns, protoVulns := mock.GetTestScannerVulns()
	for i, vuln := range scannerVulns {
		assert.Equal(t, protoVulns[i], convertNodeVulnerability(&vuln))
	}
}
