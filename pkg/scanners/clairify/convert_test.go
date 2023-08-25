package clairify

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/scanners/clairify/mock"
	v1 "github.com/stackrox/scanner/generated/scanner/api/v1"
	"github.com/stretchr/testify/assert"
)

func TestConvertNodeToVulnRequest(t *testing.T) {
	for _, testCase := range []struct {
		containerRuntime *storage.ContainerRuntimeInfo
		kernelVersion    string
		osImage          string
		kubeletVersion   string
		kubeProxyVersion string
		nodeInventory    *storage.NodeInventory

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
		{
			nodeInventory: &storage.NodeInventory{
				Components: &storage.NodeInventory_Components{
					Namespace:       "rhcos:4.11",
					RhelContentSets: []string{"rhel-8-for-x86_64-appstream-rpms", "rhel-8-for-x86_64-baseos-rpms"},
					RhelComponents: []*storage.NodeInventory_Components_RHELComponent{
						{
							Id:        int64(1),
							Name:      "vim-minimal",
							Namespace: "rhel:8",
							Version:   "2:7.4.629-6.el8",
							Arch:      "x86_64",
							Module:    "",
							AddedBy:   "",
						},
					},
				},
			},
			kernelVersion:    "3.10.0-1127.13.1.el7.x86_64",
			osImage:          "linux",
			kubeletVersion:   "v1.14.8",
			kubeProxyVersion: "v1.16.13-gke.401",
			containerRuntime: &storage.ContainerRuntimeInfo{
				Type:    storage.ContainerRuntime_UNKNOWN_CONTAINER_RUNTIME,
				Version: "containerd://1.2.8",
			},
			expected: &v1.GetNodeVulnerabilitiesRequest{
				KernelVersion:    "3.10.0-1127.13.1.el7.x86_64",
				OsImage:          "linux",
				KubeletVersion:   "v1.14.8",
				KubeproxyVersion: "v1.16.13-gke.401",
				Runtime: &v1.GetNodeVulnerabilitiesRequest_ContainerRuntime{
					Name:    "containerd",
					Version: "1.2.8",
				},
				Components: &v1.Components{
					Namespace:       "rhcos:4.11",
					RhelContentSets: []string{"rhel-8-for-x86_64-appstream-rpms", "rhel-8-for-x86_64-baseos-rpms"},
					RhelComponents: []*v1.RHELComponent{
						{
							Id:          int64(1),
							Name:        "vim-minimal",
							Namespace:   "rhel:8",
							Version:     "2:7.4.629-6.el8",
							Arch:        "x86_64",
							Module:      "",
							AddedBy:     "",
							Cpes:        nil,
							Executables: []*v1.Executable{},
						},
					},
					OsComponents:       nil,
					LanguageComponents: nil,
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
		assert.Equal(t, testCase.expected, convertNodeToVulnRequest(node, testCase.nodeInventory))
	}
}

func TestConvertVulnResponseToNodeScan(t *testing.T) {
	for _, testCase := range []struct {
		req  *v1.GetNodeVulnerabilitiesRequest
		resp *v1.GetNodeVulnerabilitiesResponse

		expectedNotes      []storage.NodeScan_Note
		expectedComponents []*storage.EmbeddedNodeScanComponent
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
				NodeNotes: []v1.NodeNote{v1.NodeNote_NODE_UNSUPPORTED, v1.NodeNote_NODE_KERNEL_UNSUPPORTED, v1.NodeNote_NODE_CERTIFIED_RHEL_CVES_UNAVAILABLE},
			},
			expectedNotes: []storage.NodeScan_Note{storage.NodeScan_UNSUPPORTED, storage.NodeScan_KERNEL_UNSUPPORTED, storage.NodeScan_CERTIFIED_RHEL_CVES_UNAVAILABLE},
			expectedComponents: []*storage.EmbeddedNodeScanComponent{
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
		{
			req: &v1.GetNodeVulnerabilitiesRequest{
				KubeletVersion:   "v1.24.6+deccab3",
				KubeproxyVersion: "v1.24.6+deccab3",
				Runtime: &v1.GetNodeVulnerabilitiesRequest_ContainerRuntime{
					Name:    "cri-o",
					Version: "1.24.4-5.rhaos4.11.git57d7127.el8",
				},
				Components: &v1.Components{
					Namespace:       "rhcos:4.11",
					RhelContentSets: []string{"rhel-8-for-x86_64-appstream-rpms", "rhel-8-for-x86_64-baseos-rpms"},
					RhelComponents: []*v1.RHELComponent{
						{
							Id:          int64(1),
							Name:        "vim-minimal",
							Namespace:   "rhel:8",
							Version:     "2:7.4.629-6.el8",
							Arch:        "x86_64",
							Module:      "",
							AddedBy:     "",
							Cpes:        nil,
							Executables: []*v1.Executable{},
						},
					},
					OsComponents:       nil,
					LanguageComponents: nil,
				},
			},
			resp: &v1.GetNodeVulnerabilitiesResponse{
				Features: []*v1.Feature{
					{
						Name:    "vim-minimal",
						Version: "2:7.4.629-6.el8",
						Vulnerabilities: []*v1.Vulnerability{
							{
								Name:    "CVE-2020-0000",
								Link:    "link0",
								FixedBy: "0",
							},
							{
								Name:    "CVE-2020-1111",
								Link:    "link1",
								FixedBy: "1",
							},
						},
					},
				},
				NodeNotes: []v1.NodeNote{v1.NodeNote_NODE_UNSUPPORTED, v1.NodeNote_NODE_KERNEL_UNSUPPORTED},
			},
			expectedNotes: []storage.NodeScan_Note{storage.NodeScan_UNSUPPORTED, storage.NodeScan_KERNEL_UNSUPPORTED},
			expectedComponents: []*storage.EmbeddedNodeScanComponent{
				{
					Name:    "vim-minimal",
					Version: "2:7.4.629-6.el8",
					Vulns: []*storage.EmbeddedVulnerability{
						{
							Cve:  "CVE-2020-0000",
							Link: "link0",
							SetFixedBy: &storage.EmbeddedVulnerability_FixedBy{
								FixedBy: "0",
							},
							VulnerabilityType: storage.EmbeddedVulnerability_NODE_VULNERABILITY,
						},
						{
							Cve:  "CVE-2020-1111",
							Link: "link1",
							SetFixedBy: &storage.EmbeddedVulnerability_FixedBy{
								FixedBy: "1",
							},
							VulnerabilityType: storage.EmbeddedVulnerability_NODE_VULNERABILITY,
						},
					},
				},
			},
		},
	} {
		actual := convertVulnResponseToNodeScan(testCase.req, testCase.resp)
		assert.ElementsMatch(t, testCase.expectedNotes, actual.Notes)
		assert.ElementsMatch(t, testCase.expectedComponents, actual.Components)
	}
}

func TestConvertNodeVulnerabilities(t *testing.T) {
	scannerVulns, protoVulns := mock.GetTestScannerVulns()
	for i := range scannerVulns {
		assert.Equal(t, protoVulns[i], convertVulnerability(&scannerVulns[i], storage.EmbeddedVulnerability_NODE_VULNERABILITY))
	}
}

func TestConvertFeatures(t *testing.T) {
	pgtest.SkipIfPostgresEnabled(t)
	// metadata is based on the fixture used below.
	metadata := &storage.ImageMetadata{
		V1: &storage.V1Metadata{
			Digest: "sha256:idk",
			Author: "stackrox",
			Layers: []*storage.ImageLayer{
				{
					Instruction: "FROM",
					Value:       "ubi8",
					Author:      "Red Hat",
				},
				{
					Instruction: "COPY",
					Value:       "stackrox.go /",
					Author:      "StackRox",
				},
			},
			Command: []string{"go", "run", "stackrox.go"},
		},
		V2: &storage.V2Metadata{
			Digest: "sha256:idk",
		},
		LayerShas: []string{"sha256:idk0", "sha256:idk1"},
		Version:   0,
	}

	features := fixtures.ScannerFeaturesV1()

	expectedFeatures := []*storage.EmbeddedImageScanComponent{
		{
			Name:    "rpm",
			Version: "4.16.0",
			FixedBy: "4.16.1",
			Vulns: []*storage.EmbeddedVulnerability{
				{
					Cve:               "CVE-2022-1234",
					Summary:           "This is the worst vulnerability I have ever seen",
					Link:              "https://access.redhat.com/security/cve/CVE-2022-1234",
					VulnerabilityType: storage.EmbeddedVulnerability_IMAGE_VULNERABILITY,
					Severity:          storage.VulnerabilitySeverity_IMPORTANT_VULNERABILITY_SEVERITY,
					Cvss:              6.3,
					ScoreVersion:      storage.EmbeddedVulnerability_V3,
					CvssV2: &storage.CVSSV2{
						Vector:              "AV:A/AC:M/Au:M/C:N/I:P/A:C",
						AttackVector:        storage.CVSSV2_ATTACK_ADJACENT,
						AccessComplexity:    storage.CVSSV2_ACCESS_MEDIUM,
						Authentication:      storage.CVSSV2_AUTH_MULTIPLE,
						Confidentiality:     storage.CVSSV2_IMPACT_NONE,
						Integrity:           storage.CVSSV2_IMPACT_PARTIAL,
						Availability:        storage.CVSSV2_IMPACT_COMPLETE,
						ExploitabilityScore: 3.5,
						ImpactScore:         7.8,
						Score:               5.4,
						Severity:            storage.CVSSV2_MEDIUM,
					},
					CvssV3: &storage.CVSSV3{
						Vector:              "CVSS:3.1/AV:A/AC:L/PR:L/UI:N/S:U/C:L/I:N/A:H",
						ExploitabilityScore: 2.1,
						ImpactScore:         4.2,
						AttackVector:        storage.CVSSV3_ATTACK_ADJACENT,
						AttackComplexity:    storage.CVSSV3_COMPLEXITY_LOW,
						PrivilegesRequired:  storage.CVSSV3_PRIVILEGE_LOW,
						UserInteraction:     storage.CVSSV3_UI_NONE,
						Scope:               storage.CVSSV3_UNCHANGED,
						Confidentiality:     storage.CVSSV3_IMPACT_LOW,
						Integrity:           storage.CVSSV3_IMPACT_NONE,
						Availability:        storage.CVSSV3_IMPACT_HIGH,
						Score:               6.3,
						Severity:            storage.CVSSV3_MEDIUM,
					},
					SetFixedBy: &storage.EmbeddedVulnerability_FixedBy{
						FixedBy: "4.16.1",
					},
				},
				{
					Cve:               "CVE-2022-1235",
					Summary:           "This is the second worst vulnerability I have ever seen",
					Link:              "https://access.redhat.com/security/cve/CVE-2022-1235",
					VulnerabilityType: storage.EmbeddedVulnerability_IMAGE_VULNERABILITY,
					Severity:          storage.VulnerabilitySeverity_MODERATE_VULNERABILITY_SEVERITY,
					Cvss:              5.4,
					ScoreVersion:      storage.EmbeddedVulnerability_V2,
					CvssV2: &storage.CVSSV2{
						Vector:              "AV:A/AC:M/Au:M/C:N/I:P/A:C",
						AttackVector:        storage.CVSSV2_ATTACK_ADJACENT,
						AccessComplexity:    storage.CVSSV2_ACCESS_MEDIUM,
						Authentication:      storage.CVSSV2_AUTH_MULTIPLE,
						Confidentiality:     storage.CVSSV2_IMPACT_NONE,
						Integrity:           storage.CVSSV2_IMPACT_PARTIAL,
						Availability:        storage.CVSSV2_IMPACT_COMPLETE,
						ExploitabilityScore: 3.5,
						ImpactScore:         7.8,
						Score:               5.4,
						Severity:            storage.CVSSV2_MEDIUM,
					},
					SetFixedBy: &storage.EmbeddedVulnerability_FixedBy{},
				},
			},
			HasLayerIndex: &storage.EmbeddedImageScanComponent_LayerIndex{
				LayerIndex: 0,
			},
			Executables: []*storage.EmbeddedImageScanComponent_Executable{
				{
					Path:         "/bin/rpm",
					Dependencies: []string{"Z2xpYmM:MQ", "bGliLnNv:Mg"},
				},
			},
		},
		{
			Name:    "curl",
			Version: "1",
			Vulns:   []*storage.EmbeddedVulnerability{},
			HasLayerIndex: &storage.EmbeddedImageScanComponent_LayerIndex{
				LayerIndex: 0,
			},
			Executables: []*storage.EmbeddedImageScanComponent_Executable{},
		},
		{
			Name:     "java.jar",
			Version:  "1",
			Location: "/java/jar/path/java.jar",
			Source:   storage.SourceType_JAVA,
			Vulns:    []*storage.EmbeddedVulnerability{},
			HasLayerIndex: &storage.EmbeddedImageScanComponent_LayerIndex{
				LayerIndex: 1,
			},
			Executables: []*storage.EmbeddedImageScanComponent_Executable{},
		},
	}

	converted := convertFeatures(metadata, features, "")
	assert.Equal(t, expectedFeatures, converted)
}
