package clairify

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/protoassert"
	"github.com/stackrox/rox/pkg/scanners/clairify/mock"
	v1 "github.com/stackrox/scanner/generated/scanner/api/v1"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/proto"
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
			containerRuntime: storage.ContainerRuntimeInfo_builder{
				Type:    storage.ContainerRuntime_DOCKER_CONTAINER_RUNTIME,
				Version: "19.3.5",
			}.Build(),
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
			containerRuntime: storage.ContainerRuntimeInfo_builder{
				Type:    storage.ContainerRuntime_CRIO_CONTAINER_RUNTIME,
				Version: "1.11.13-1.rhaos3.11.gitfb88a9c.el7",
			}.Build(),
			expected: &v1.GetNodeVulnerabilitiesRequest{
				Runtime: &v1.GetNodeVulnerabilitiesRequest_ContainerRuntime{
					Name:    "cri-o",
					Version: "1.11.13-1.rhaos3.11.gitfb88a9c.el7",
				},
			},
		},
		{
			containerRuntime: storage.ContainerRuntimeInfo_builder{
				Type:    storage.ContainerRuntime_UNKNOWN_CONTAINER_RUNTIME,
				Version: "containerd://1.2.8",
			}.Build(),
			expected: &v1.GetNodeVulnerabilitiesRequest{
				Runtime: &v1.GetNodeVulnerabilitiesRequest_ContainerRuntime{
					Name:    "containerd",
					Version: "1.2.8",
				},
			},
		},
		{
			nodeInventory: storage.NodeInventory_builder{
				Components: storage.NodeInventory_Components_builder{
					Namespace:       "rhcos:4.11",
					RhelContentSets: []string{"rhel-8-for-x86_64-appstream-rpms", "rhel-8-for-x86_64-baseos-rpms"},
					RhelComponents: []*storage.NodeInventory_Components_RHELComponent{
						storage.NodeInventory_Components_RHELComponent_builder{
							Id:        int64(1),
							Name:      "vim-minimal",
							Namespace: "rhel:8",
							Version:   "2:7.4.629-6.el8",
							Arch:      "x86_64",
							Module:    "",
							AddedBy:   "",
						}.Build(),
					},
				}.Build(),
			}.Build(),
			kernelVersion:    "3.10.0-1127.13.1.el7.x86_64",
			osImage:          "linux",
			kubeletVersion:   "v1.14.8",
			kubeProxyVersion: "v1.16.13-gke.401",
			containerRuntime: storage.ContainerRuntimeInfo_builder{
				Type:    storage.ContainerRuntime_UNKNOWN_CONTAINER_RUNTIME,
				Version: "containerd://1.2.8",
			}.Build(),
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
		node := &storage.Node{}
		node.SetContainerRuntime(testCase.containerRuntime)
		node.SetKernelVersion(testCase.kernelVersion)
		node.SetOsImage(testCase.osImage)
		node.SetKubeletVersion(testCase.kubeletVersion)
		node.SetKubeProxyVersion(testCase.kubeProxyVersion)
		protoassert.Equal(t, testCase.expected, convertNodeToVulnRequest(node, testCase.nodeInventory))
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
				storage.EmbeddedNodeScanComponent_builder{
					Name:    "docker",
					Version: "19.3.5",
					Vulns: []*storage.EmbeddedVulnerability{
						storage.EmbeddedVulnerability_builder{
							Cve:               "CVE-2020-0000",
							Link:              "link0",
							FixedBy:           proto.String("0"),
							VulnerabilityType: storage.EmbeddedVulnerability_NODE_VULNERABILITY,
						}.Build(),
					},
				}.Build(),
				storage.EmbeddedNodeScanComponent_builder{
					Name:    "kernel",
					Version: "4.9.184-linuxkit",
					Vulns: []*storage.EmbeddedVulnerability{
						storage.EmbeddedVulnerability_builder{
							Cve:               "CVE-2020-1111",
							Link:              "link1",
							FixedBy:           proto.String("1"),
							VulnerabilityType: storage.EmbeddedVulnerability_NODE_VULNERABILITY,
						}.Build(),
					}}.Build(),
				storage.EmbeddedNodeScanComponent_builder{
					Name:    "kubelet",
					Version: "v1.16.13-gke.401",
					Vulns: []*storage.EmbeddedVulnerability{
						storage.EmbeddedVulnerability_builder{
							Cve:               "CVE-2020-2222",
							Link:              "link2",
							FixedBy:           proto.String("2"),
							VulnerabilityType: storage.EmbeddedVulnerability_NODE_VULNERABILITY,
						}.Build(),
					}}.Build(),
				storage.EmbeddedNodeScanComponent_builder{
					Name:    "kube-proxy",
					Version: "v1.17.14-gke.400",
					Vulns: []*storage.EmbeddedVulnerability{
						storage.EmbeddedVulnerability_builder{
							Cve:               "CVE-2020-3333",
							Link:              "link3",
							FixedBy:           proto.String("3"),
							VulnerabilityType: storage.EmbeddedVulnerability_NODE_VULNERABILITY,
						}.Build(),
						storage.EmbeddedVulnerability_builder{
							Cve:               "CVE-2020-4444",
							Link:              "link4",
							FixedBy:           proto.String("4"),
							VulnerabilityType: storage.EmbeddedVulnerability_NODE_VULNERABILITY,
						}.Build(),
					},
				}.Build(),
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
				storage.EmbeddedNodeScanComponent_builder{
					Name:    "vim-minimal",
					Version: "2:7.4.629-6.el8",
					Vulns: []*storage.EmbeddedVulnerability{
						storage.EmbeddedVulnerability_builder{
							Cve:               "CVE-2020-0000",
							Link:              "link0",
							FixedBy:           proto.String("0"),
							VulnerabilityType: storage.EmbeddedVulnerability_NODE_VULNERABILITY,
						}.Build(),
						storage.EmbeddedVulnerability_builder{
							Cve:               "CVE-2020-1111",
							Link:              "link1",
							FixedBy:           proto.String("1"),
							VulnerabilityType: storage.EmbeddedVulnerability_NODE_VULNERABILITY,
						}.Build(),
					},
				}.Build(),
			},
		},
	} {
		actual := convertVulnResponseToNodeScan(testCase.req, testCase.resp)
		assert.ElementsMatch(t, testCase.expectedNotes, actual.GetNotes())
		protoassert.ElementsMatch(t, testCase.expectedComponents, actual.GetComponents())
	}
}

func TestConvertNodeVulnerabilities(t *testing.T) {
	scannerVulns, protoVulns := mock.GetTestScannerVulns()
	for i := range scannerVulns {
		protoassert.Equal(t, protoVulns[i], convertVulnerability(&scannerVulns[i], storage.EmbeddedVulnerability_NODE_VULNERABILITY))
	}
}

func TestConvertFeatures(t *testing.T) {
	// metadata is based on the fixture used below.
	metadata := storage.ImageMetadata_builder{
		V1: storage.V1Metadata_builder{
			Digest: "sha256:idk",
			Author: "stackrox",
			Layers: []*storage.ImageLayer{
				storage.ImageLayer_builder{
					Instruction: "FROM",
					Value:       "ubi8",
					Author:      "Red Hat",
				}.Build(),
				storage.ImageLayer_builder{
					Instruction: "COPY",
					Value:       "stackrox.go /",
					Author:      "StackRox",
				}.Build(),
			},
			Command: []string{"go", "run", "stackrox.go"},
		}.Build(),
		V2: storage.V2Metadata_builder{
			Digest: "sha256:idk",
		}.Build(),
		LayerShas: []string{"sha256:idk0", "sha256:idk1"},
		Version:   0,
	}.Build()

	features := fixtures.ScannerFeaturesV1()

	expectedFeatures := []*storage.EmbeddedImageScanComponent{
		storage.EmbeddedImageScanComponent_builder{
			Name:    "rpm",
			Version: "4.16.0",
			FixedBy: "4.16.1",
			Vulns: []*storage.EmbeddedVulnerability{
				storage.EmbeddedVulnerability_builder{
					Cve:               "CVE-2022-1234",
					Summary:           "This is the worst vulnerability I have ever seen",
					Link:              "https://access.redhat.com/security/cve/CVE-2022-1234",
					VulnerabilityType: storage.EmbeddedVulnerability_IMAGE_VULNERABILITY,
					Severity:          storage.VulnerabilitySeverity_IMPORTANT_VULNERABILITY_SEVERITY,
					Cvss:              6.3,
					ScoreVersion:      storage.EmbeddedVulnerability_V3,
					CvssV2: storage.CVSSV2_builder{
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
					}.Build(),
					CvssV3: storage.CVSSV3_builder{
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
					}.Build(),
					FixedBy: proto.String("4.16.1"),
				}.Build(),
				storage.EmbeddedVulnerability_builder{
					Cve:               "CVE-2022-1235",
					Summary:           "This is the second worst vulnerability I have ever seen",
					Link:              "https://access.redhat.com/security/cve/CVE-2022-1235",
					VulnerabilityType: storage.EmbeddedVulnerability_IMAGE_VULNERABILITY,
					Severity:          storage.VulnerabilitySeverity_MODERATE_VULNERABILITY_SEVERITY,
					Cvss:              5.4,
					ScoreVersion:      storage.EmbeddedVulnerability_V2,
					CvssV2: storage.CVSSV2_builder{
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
					}.Build(),
					FixedBy: proto.String(""),
				}.Build(),
			},
			LayerIndex:  proto.Int32(0),
			Executables: []*storage.EmbeddedImageScanComponent_Executable{},
		}.Build(),
		storage.EmbeddedImageScanComponent_builder{
			Name:        "curl",
			Version:     "1",
			Vulns:       []*storage.EmbeddedVulnerability{},
			LayerIndex:  proto.Int32(0),
			Executables: []*storage.EmbeddedImageScanComponent_Executable{},
		}.Build(),
		storage.EmbeddedImageScanComponent_builder{
			Name:        "java.jar",
			Version:     "1",
			Location:    "/java/jar/path/java.jar",
			Source:      storage.SourceType_JAVA,
			Vulns:       []*storage.EmbeddedVulnerability{},
			LayerIndex:  proto.Int32(1),
			Executables: []*storage.EmbeddedImageScanComponent_Executable{},
		}.Build(),
	}

	converted := convertFeatures(metadata, features, "")
	protoassert.SlicesEqual(t, expectedFeatures, converted)
}
