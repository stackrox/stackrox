package nodes

import (
	"bytes"
	"context"
	"testing"
	"time"

	// Embed is used to import the serialized test object file.
	_ "embed"

	"github.com/spf13/cobra"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/fixtures/fixtureconsts"
	pkgGRPC "github.com/stackrox/rox/pkg/grpc"
	"github.com/stackrox/rox/pkg/protocompat"
	envMocks "github.com/stackrox/rox/roxctl/common/environment/mocks"
	"github.com/stackrox/rox/roxctl/common/flags"
	ioMocks "github.com/stackrox/rox/roxctl/common/io/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"
)

//go:embed serialized_test_node.json
var expectedJSONSerializedNode string

func TestExportNodes(t *testing.T) {
	fakeService := &fakeNodeService{tb: t}
	conn, closeFunc, err := pkgGRPC.CreateTestGRPCStreamingService(
		context.Background(),
		t,
		func(registrar grpc.ServiceRegistrar) {
			v1.RegisterNodeServiceServer(registrar, fakeService)
		},
	)
	require.NoError(t, err)
	defer closeFunc()

	mockCtrl := gomock.NewController(t)
	var buf bytes.Buffer

	mockIO := ioMocks.NewMockIO(mockCtrl)
	mockIO.EXPECT().Out().Times(1).Return(&buf)

	mockEnv := envMocks.NewMockEnvironment(mockCtrl)
	mockEnv.EXPECT().GRPCConnection().Times(1).Return(conn, nil)
	mockEnv.EXPECT().InputOutput().Times(1).Return(mockIO)

	fakeCmd := &cobra.Command{}
	flags.AddTimeoutWithDefault(fakeCmd, 10*time.Second)

	cmd := Command(mockEnv)
	err = cmd.RunE(fakeCmd, []string{})
	assert.NoError(t, err)
	assert.JSONEq(t, `{"node":`+expectedJSONSerializedNode+`}`, buf.String())
}

type fakeNodeService struct {
	tb testing.TB
}

func (s *fakeNodeService) ExportNodes(_ *v1.ExportNodeRequest, srv v1.NodeService_ExportNodesServer) error {
	enr := &v1.ExportNodeResponse{}
	enr.SetNode(testNode)
	return srv.Send(enr)
}

func (s *fakeNodeService) ListNodes(_ context.Context, _ *v1.ListNodesRequest) (*v1.ListNodesResponse, error) {
	return nil, errox.NotImplemented
}

func (s *fakeNodeService) GetNode(_ context.Context, _ *v1.GetNodeRequest) (*storage.Node, error) {
	return nil, errox.NotImplemented
}

var (
	joinedDate  = time.Date(2020, time.December, 24, 23, 59, 59, 999999999, time.UTC)
	updatedDate = time.Date(2020, time.December, 31, 23, 59, 59, 999999999, time.UTC)
	scanDate    = time.Date(2021, time.January, 1, 1, 1, 1, 111111111, time.UTC)

	testNode = storage.Node_builder{
		Id:          fixtureconsts.Node1,
		Name:        "fp-3-14-rethorical-hhg-h2g2-worker-b-abcde",
		ClusterId:   fixtureconsts.Cluster1,
		ClusterName: "my-cluster",
		Labels: map[string]string{
			"kubernetes.io/arch":               "amd64",
			"kubernetes.io/hostname":           "fp-3-14-rethorical-hhg-h2g2-worker-b-abcde",
			"kubernetes.io/os":                 "linux",
			"node-role.kubernetes.io/worker":   "",
			"node.kubernetes.io/instance-type": "e4-standard-32",
			"topology.gke.io/zone":             "us-east17-b",
			"topology.kubernetes.io/region":    "us-east17",
			"topology.kubernetes.io/zone":      "us-east17-b",
		},
		Annotations: map[string]string{
			"machine.openshift.io/machine":                                                "openshift-machine-api/fp-3-14-rethorical-hhg-h2g2-worker-b-abcde",
			"machineconfiguration.openshift.io/controlPlaneTopology":                      "HighlyAvailable",
			"machineconfiguration.openshift.io/currentConfig":                             "rendered-worker-1234567890abcdef1234567890abcdef",
			"machineconfiguration.openshift.io/desiredConfig":                             "rendered-worker-1234567890abcdef1234567890abcdef",
			"machineconfiguration.openshift.io/desiredDrain":                              "uncordon-rendered-worker-1234567890abcdef1234567890abcdef",
			"machineconfiguration.openshift.io/lastAppliedDrain":                          "uncordon-rendered-worker-1234567890abcdef1234567890abcdef",
			"machineconfiguration.openshift.io/lastSyncedControllerConfigResourceVersion": "12345",
			"machineconfiguration.openshift.io/reason":                                    "",
			"machineconfiguration.openshift.io/state":                                     "Done",
			"volumes.kubernetes.io/controller-managed-attach-detach":                      "true",
		},
		JoinedAt:            protocompat.ConvertTimeToTimestampOrNil(&joinedDate),
		InternalIpAddresses: []string{"192.168.0.1"},
		ExternalIpAddresses: []string{},
		ContainerRuntime: storage.ContainerRuntimeInfo_builder{
			Type:    storage.ContainerRuntime_DOCKER_CONTAINER_RUNTIME,
			Version: "3.14-15.9",
		}.Build(),
		KernelVersion:    "5.6.7-89.0.x86_64",
		OperatingSystem:  "linux",
		OsImage:          "Some Real good Linux",
		KubeletVersion:   "v3.14.2",
		KubeProxyVersion: "v3.14.2",
		LastUpdated:      protocompat.ConvertTimeToTimestampOrNil(&updatedDate),
		K8SUpdated:       protocompat.ConvertTimeToTimestampOrNil(&updatedDate),
		Scan: storage.NodeScan_builder{
			ScanTime:        protocompat.ConvertTimeToTimestampOrNil(&scanDate),
			OperatingSystem: "linux-5.6.7-89.0",
			Components: []*storage.EmbeddedNodeScanComponent{
				storage.EmbeddedNodeScanComponent_builder{
					Name:    "NetworkManager",
					Version: "1:1.42.2-12.el9_2.x86_64",
					Vulnerabilities: []*storage.NodeVulnerability{
						storage.NodeVulnerability_builder{
							CveBaseInfo: storage.CVEInfo_builder{
								Cve:          "CVE-2021-20297",
								Summary:      "DOCUMENTATION: A flaw was found in NetworkManager. Setting match.path and activating a profile crashes NetworkManager. The highest threat from this vulnerability is to system availability.",
								Link:         "https://access.redhat.com/security/cve/CVE-2021-20297",
								CreatedAt:    protocompat.ConvertTimeToTimestampOrNil(&scanDate),
								LastModified: protocompat.ConvertTimeToTimestampOrNil(&updatedDate),
								ScoreVersion: storage.CVEInfo_V3,
								CvssV3: storage.CVSSV3_builder{
									Vector:              "CVSS:3.1/AV:L/AC:L/PR:L/UI:N/S:U/C:N/I:N/A:H",
									ExploitabilityScore: 1.8,
									ImpactScore:         3.6,
									AttackVector:        storage.CVSSV3_ATTACK_LOCAL,
									AttackComplexity:    storage.CVSSV3_COMPLEXITY_LOW,
									PrivilegesRequired:  storage.CVSSV3_PRIVILEGE_LOW,
									UserInteraction:     storage.CVSSV3_UI_NONE,
									Scope:               storage.CVSSV3_UNCHANGED,
									Confidentiality:     storage.CVSSV3_IMPACT_NONE,
									Integrity:           storage.CVSSV3_IMPACT_NONE,
									Availability:        storage.CVSSV3_IMPACT_HIGH,
									Score:               5.5,
									Severity:            storage.CVSSV3_MEDIUM,
								}.Build(),
								References: []*storage.CVEInfo_Reference{},
							}.Build(),
							Cvss:     5.5,
							Severity: storage.VulnerabilitySeverity_MODERATE_VULNERABILITY_SEVERITY,
							Snoozed:  false,
						}.Build(),
					},
					Priority:  25,
					TopCvss:   proto.Float32(5.5),
					RiskScore: 1.1,
				}.Build(),
				storage.EmbeddedNodeScanComponent_builder{
					Name:            "basesystem",
					Version:         "11-13.el9.noarch",
					Vulnerabilities: []*storage.NodeVulnerability{},
					Priority:        26,
					RiskScore:       0,
				}.Build(),
				storage.EmbeddedNodeScanComponent_builder{
					Name:            "bash",
					Version:         "5.1.8-6.el9_1.x86_64",
					Vulnerabilities: []*storage.NodeVulnerability{},
					Priority:        26,
					RiskScore:       0,
				}.Build(),
				storage.EmbeddedNodeScanComponent_builder{
					Name:    "common",
					Version: "2:2.1.7-1.el9_2.x86_64",
					Vulnerabilities: []*storage.NodeVulnerability{
						storage.NodeVulnerability_builder{
							CveBaseInfo: storage.CVEInfo_builder{
								Cve:          "CVE-2024-28176",
								Summary:      "DOCUMENTATION: Jose was found to have an uncontrolled resource consumption vulnerability. Under certain conditions, the user's environment can consume an unreasonable amount of CPU time or memory during JWE decryption operations, leading to a denial of service.",
								Link:         "https://access.redhat.com/security/cve/CVE-2024-28176",
								CreatedAt:    protocompat.ConvertTimeToTimestampOrNil(&scanDate),
								LastModified: protocompat.ConvertTimeToTimestampOrNil(&updatedDate),
								ScoreVersion: storage.CVEInfo_V3,
								CvssV3: storage.CVSSV3_builder{
									Vector:              "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:N/I:N/A:L",
									ExploitabilityScore: 3.9,
									ImpactScore:         1.4,
									AttackVector:        storage.CVSSV3_ATTACK_NETWORK,
									AttackComplexity:    storage.CVSSV3_COMPLEXITY_LOW,
									PrivilegesRequired:  storage.CVSSV3_PRIVILEGE_NONE,
									UserInteraction:     storage.CVSSV3_UI_NONE,
									Scope:               storage.CVSSV3_UNCHANGED,
									Confidentiality:     storage.CVSSV3_IMPACT_NONE,
									Integrity:           storage.CVSSV3_IMPACT_NONE,
									Availability:        storage.CVSSV3_IMPACT_LOW,
									Score:               5.3,
									Severity:            storage.CVSSV3_MEDIUM,
								}.Build(),
								References: []*storage.CVEInfo_Reference{},
							}.Build(),
							Cvss:       5.3,
							Severity:   storage.VulnerabilitySeverity_MODERATE_VULNERABILITY_SEVERITY,
							SetFixedBy: nil,
						}.Build(),
						storage.NodeVulnerability_builder{
							CveBaseInfo: storage.CVEInfo_builder{
								Cve:          "CVE-2024-28180",
								Summary:      "DOCUMENTATION: A vulnerability was found in Jose due to improper handling of highly compressed data. This issue could allow an attacker to send a JWE containing compressed data that uses large amounts of memory and CPU when decompressed by Decrypt or DecryptMulti.",
								Link:         "https://access.redhat.com/security/cve/CVE-2024-28180",
								CreatedAt:    protocompat.ConvertTimeToTimestampOrNil(&scanDate),
								LastModified: protocompat.ConvertTimeToTimestampOrNil(&updatedDate),
								ScoreVersion: storage.CVEInfo_V3,
								CvssV3: storage.CVSSV3_builder{
									Vector:              "CVSS:3.1/AV:N/AC:L/PR:L/UI:N/S:U/C:N/I:N/A:L",
									ExploitabilityScore: 2.8,
									ImpactScore:         1.4,
									AttackVector:        storage.CVSSV3_ATTACK_NETWORK,
									AttackComplexity:    storage.CVSSV3_COMPLEXITY_LOW,
									PrivilegesRequired:  storage.CVSSV3_PRIVILEGE_LOW,
									UserInteraction:     storage.CVSSV3_UI_NONE,
									Scope:               storage.CVSSV3_UNCHANGED,
									Confidentiality:     storage.CVSSV3_IMPACT_NONE,
									Integrity:           storage.CVSSV3_IMPACT_NONE,
									Availability:        storage.CVSSV3_IMPACT_LOW,
									Score:               4.3,
									Severity:            storage.CVSSV3_MEDIUM,
								}.Build(),
								References: []*storage.CVEInfo_Reference{},
							}.Build(),
							Cvss:       4.3,
							Severity:   storage.VulnerabilitySeverity_MODERATE_VULNERABILITY_SEVERITY,
							SetFixedBy: nil,
						}.Build(),
					},
					Priority:  23,
					TopCvss:   proto.Float32(5.3),
					RiskScore: 1.1815,
				}.Build(),
				storage.EmbeddedNodeScanComponent_builder{
					Name:            "glibc",
					Version:         "2.34-60.el9_2.7.x86_64",
					Vulnerabilities: []*storage.NodeVulnerability{},
					Priority:        26,
					RiskScore:       0,
				}.Build(),
				storage.EmbeddedNodeScanComponent_builder{
					Name:            "glibc-common",
					Version:         "2.34-60.el9_2.7.x86_64",
					Vulnerabilities: []*storage.NodeVulnerability{},
					Priority:        26,
					RiskScore:       0,
				}.Build(),
				storage.EmbeddedNodeScanComponent_builder{
					Name:    "oniguruma",
					Version: "6.9.6-1.el9.5.x86_64",
					Vulnerabilities: []*storage.NodeVulnerability{
						storage.NodeVulnerability_builder{
							CveBaseInfo: storage.CVEInfo_builder{
								Cve:          "CVE-2017-9226",
								Summary:      "DOCUMENTATION: The MITRE CVE dictionary describes this issue as: An issue was discovered in Oniguruma 6.2.0, as used in Oniguruma-mod in Ruby through 2.4.1 and mbstring in PHP through 7.1.5. A heap out-of-bounds write or read occurs in next_state_val() during regular expression compilation. Octal numbers larger than 0xff are not handled correctly in fetch_token() and fetch_token_in_cc(). A malformed regular expression containing an octal number in the form of '\\\\700' would produce an invalid code point value larger than 0xff in next_state_val(), resulting in an out-of-bounds write memory corruption.",
								Link:         "https://access.redhat.com/security/cve/CVE-2017-9226",
								CreatedAt:    protocompat.ConvertTimeToTimestampOrNil(&updatedDate),
								LastModified: protocompat.ConvertTimeToTimestampOrNil(&joinedDate),
								ScoreVersion: storage.CVEInfo_V3,
								CvssV3: storage.CVSSV3_builder{
									Vector:              "CVSS:3.0/AV:N/AC:H/PR:N/UI:N/S:U/C:N/I:L/A:L",
									ExploitabilityScore: 2.2,
									ImpactScore:         2.5,
									AttackVector:        storage.CVSSV3_ATTACK_NETWORK,
									AttackComplexity:    storage.CVSSV3_COMPLEXITY_HIGH,
									PrivilegesRequired:  storage.CVSSV3_PRIVILEGE_NONE,
									UserInteraction:     storage.CVSSV3_UI_NONE,
									Scope:               storage.CVSSV3_UNCHANGED,
									Confidentiality:     storage.CVSSV3_IMPACT_NONE,
									Integrity:           storage.CVSSV3_IMPACT_LOW,
									Availability:        storage.CVSSV3_IMPACT_LOW,
									Score:               4.8,
									Severity:            storage.CVSSV3_MEDIUM,
								}.Build(),
								References: []*storage.CVEInfo_Reference{},
							}.Build(),
							Cvss:     4.8,
							Severity: storage.VulnerabilitySeverity_MODERATE_VULNERABILITY_SEVERITY,
						}.Build(),
						storage.NodeVulnerability_builder{
							CveBaseInfo: storage.CVEInfo_builder{
								Cve:          "CVE-2019-16163",
								Summary:      "DOCUMENTATION: The MITRE CVE dictionary describes this issue as: Oniguruma before 6.9.3 allows Stack Exhaustion in regcomp.c because of recursion in regparse.c.",
								Link:         "https://access.redhat.com/security/cve/CVE-2019-16163",
								CreatedAt:    protocompat.ConvertTimeToTimestampOrNil(&scanDate),
								LastModified: protocompat.ConvertTimeToTimestampOrNil(&updatedDate),
								ScoreVersion: storage.CVEInfo_V3,
								CvssV3: storage.CVSSV3_builder{
									Vector:              "CVSS:3.0/AV:N/AC:L/PR:L/UI:N/S:U/C:N/I:N/A:H",
									ExploitabilityScore: 2.8,
									ImpactScore:         3.6,
									AttackVector:        storage.CVSSV3_ATTACK_NETWORK,
									PrivilegesRequired:  storage.CVSSV3_PRIVILEGE_LOW,
									UserInteraction:     storage.CVSSV3_UI_NONE,
									Scope:               storage.CVSSV3_UNCHANGED,
									Confidentiality:     storage.CVSSV3_IMPACT_NONE,
									Integrity:           storage.CVSSV3_IMPACT_NONE,
									Availability:        storage.CVSSV3_IMPACT_HIGH,
									Score:               6.5,
									Severity:            storage.CVSSV3_MEDIUM,
								}.Build(),
								References: []*storage.CVEInfo_Reference{},
							}.Build(),
							Cvss:     6.5,
							Severity: storage.VulnerabilitySeverity_MODERATE_VULNERABILITY_SEVERITY,
						}.Build(),
						storage.NodeVulnerability_builder{
							CveBaseInfo: storage.CVEInfo_builder{
								Cve:          "CVE-2020-26159",
								Summary:      "DOCUMENTATION: A flaw was found in oniguruma. An attacker, able to supply a regular expression for compilation, may be able to overflow a buffer by one byte in concat_opt_exact_str in src/regcomp.c . \\n				STATEMENT: Red Hat Ceph Storage 4 is not affected because the affected method, concat_opt_exact_str is not shipped. However, there is an identical flaw in concat_opt_exact_info_str and concat_opt_exact_info, which do not exist in the most recent version of oniguruma as methods. The impact is rated as low because we ship an older version without this exact exploit, so an attacker could not simply copy and paste this exploit, but would need to dig into the code itself and modify this attack for the older version of the code.\\n            MITIGATION: Mitigation for this issue is either not available or the currently available options do not meet the Red Hat Product Security criteria comprising ease of use and deployment, applicability to widespread installation base or stability.",
								Link:         "https://access.redhat.com/security/cve/CVE-2020-26159",
								CreatedAt:    protocompat.ConvertTimeToTimestampOrNil(&scanDate),
								LastModified: protocompat.ConvertTimeToTimestampOrNil(&joinedDate),
								ScoreVersion: storage.CVEInfo_V3,
								CvssV3: storage.CVSSV3_builder{
									Vector:              "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:L/I:L/A:H",
									ExploitabilityScore: 3.9,
									ImpactScore:         4.7,
									AttackVector:        storage.CVSSV3_ATTACK_NETWORK,
									PrivilegesRequired:  storage.CVSSV3_PRIVILEGE_NONE,
									UserInteraction:     storage.CVSSV3_UI_NONE,
									Scope:               storage.CVSSV3_UNCHANGED,
									Confidentiality:     storage.CVSSV3_IMPACT_LOW,
									Integrity:           storage.CVSSV3_IMPACT_LOW,
									Availability:        storage.CVSSV3_IMPACT_HIGH,
									Score:               8.6,
									Severity:            storage.CVSSV3_HIGH,
								}.Build(),
								References: []*storage.CVEInfo_Reference{},
							}.Build(),
							Cvss:     8.6,
							Severity: storage.VulnerabilitySeverity_MODERATE_VULNERABILITY_SEVERITY,
						}.Build(),
					},
					Priority:  13,
					TopCvss:   proto.Float32(8.6),
					RiskScore: 1.5565,
				}.Build(),
				storage.EmbeddedNodeScanComponent_builder{
					Name:            "rpcbind",
					Version:         "1.2.6-5.el9.x86_64",
					Vulnerabilities: []*storage.NodeVulnerability{},
					Priority:        26,
					RiskScore:       0,
				}.Build(),
				storage.EmbeddedNodeScanComponent_builder{
					Name:            "tar",
					Version:         "2:1.34-6.el9_1.x86_64",
					Vulnerabilities: []*storage.NodeVulnerability{},
					Priority:        26,
					RiskScore:       0,
				}.Build(),
				storage.EmbeddedNodeScanComponent_builder{
					Name:            "xz",
					Version:         "5.2.5-8.el9_0.x86_64",
					Vulnerabilities: []*storage.NodeVulnerability{},
					Priority:        26,
					RiskScore:       0,
				}.Build(),
			},
			Notes: []storage.NodeScan_Note{},
		}.Build(),
		Components:  proto.Int32(10),
		Cves:        proto.Int32(6),
		FixableCves: proto.Int32(0),
		Priority:    1,
		RiskScore:   3.14,
		TopCvss:     proto.Float32(8.6),
		Notes:       []storage.Node_Note{},
	}.Build()
)
