package m37tom38

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/require"
	"gotest.tools/assert"
)

type testcase struct {
	desc                   string
	policyFields           *storage.PolicyFields
	expectedPolicySections []*storage.PolicySection
}

type convertTestCase struct {
	desc     string
	policy   *storage.Policy
	expected *storage.Policy
	hasError bool
}

const (
	bplVersion = "1"
)

func TestCloneAndEnsureConverted(t *testing.T) {
	fields := &storage.PolicyFields{
		Cvss: &storage.NumericalPolicy{
			Op:    storage.Comparator_GREATER_THAN_OR_EQUALS,
			Value: 7.0,
		},
	}
	sections := []*storage.PolicySection{
		{
			PolicyGroups: []*storage.PolicyGroup{
				{
					FieldName: cvss,
					Values: []*storage.PolicyValue{
						{
							Value: ">= 7.000000",
						},
					},
				},
			},
		},
	}
	cases := []convertTestCase{
		{
			desc:     "nil failure",
			policy:   nil,
			expected: nil,
			hasError: true,
		},
		{
			desc: "unknown version",
			policy: &storage.Policy{
				PolicyVersion: "-1",
			},
			expected: nil,
			hasError: true,
		},
		{
			desc: "empty sections",
			policy: &storage.Policy{
				PolicyVersion: bplVersion,
			},
			expected: nil,
			hasError: true,
		},
		{
			desc: "empty fields",
			policy: &storage.Policy{
				PolicyVersion: legacyVersion,
			},
			expected: nil,
			hasError: true,
		},
		{
			desc: "valid conversion",
			policy: &storage.Policy{
				Fields: fields,
			},
			expected: &storage.Policy{
				PolicyVersion:  bplVersion,
				PolicySections: sections,
			},
		},
		{
			desc: "valid conversion with legacy version",
			policy: &storage.Policy{
				PolicyVersion: legacyVersion,
				Fields:        fields,
			},
			expected: &storage.Policy{
				PolicyVersion:  bplVersion,
				PolicySections: sections,
			},
		},
		{
			desc: "valid noop",
			policy: &storage.Policy{
				PolicyVersion:  bplVersion,
				PolicySections: sections,
			},
			expected: &storage.Policy{
				PolicyVersion:  bplVersion,
				PolicySections: sections,
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			got, err := CloneAndEnsureConverted(tc.policy)
			assert.Assert(t, tc.hasError == (err != nil))
			assert.DeepEqual(t, tc.expected, got)
		})
	}
}

func TestConvertPolicyFieldsToSections(t *testing.T) {
	tcs := []*testcase{
		{
			desc: "cvss",
			policyFields: &storage.PolicyFields{
				Cvss: &storage.NumericalPolicy{
					Op:    storage.Comparator_GREATER_THAN_OR_EQUALS,
					Value: 7.0,
				},
			},
			expectedPolicySections: []*storage.PolicySection{
				{
					PolicyGroups: []*storage.PolicyGroup{
						{
							FieldName: cvss,
							Values: []*storage.PolicyValue{
								{
									Value: ">= 7.000000",
								},
							},
						},
					},
				},
			},
		},

		{
			desc: "fixed by",
			policyFields: &storage.PolicyFields{
				FixedBy: "pkg=4",
			},
			expectedPolicySections: []*storage.PolicySection{
				{
					PolicyGroups: []*storage.PolicyGroup{
						{
							FieldName: fixedBy,
							Values: []*storage.PolicyValue{
								{
									Value: "pkg=4",
								},
							},
						},
					},
				},
			},
		},

		{
			desc: "process policy",
			policyFields: &storage.PolicyFields{
				ProcessPolicy: &storage.ProcessPolicy{
					Name:     "process",
					Args:     "--arg 1",
					Ancestor: "parent",
					Uid:      "123",
				},
			},
			expectedPolicySections: []*storage.PolicySection{
				{
					PolicyGroups: []*storage.PolicyGroup{
						{
							FieldName: processName,
							Values:    []*storage.PolicyValue{{Value: "process"}},
						},

						{
							FieldName: processAncestor,
							Values:    []*storage.PolicyValue{{Value: "parent"}},
						},

						{
							FieldName: processArguments,
							Values:    []*storage.PolicyValue{{Value: "--arg 1"}},
						},

						{
							FieldName: processUID,
							Values:    []*storage.PolicyValue{{Value: "123"}},
						},
					},
				},
			},
		},

		{
			desc: "disallowed image label",
			policyFields: &storage.PolicyFields{
				DisallowedImageLabel: &storage.KeyValuePolicy{
					Key:   "k",
					Value: "v",
				},
			},
			expectedPolicySections: []*storage.PolicySection{
				{
					PolicyGroups: []*storage.PolicyGroup{
						{
							FieldName: disallowedImageLabel,
							Values: []*storage.PolicyValue{
								{
									Value: "k=v",
								},
							},
						},
					},
				},
			},
		},

		{
			desc: "required image label",
			policyFields: &storage.PolicyFields{
				RequiredImageLabel: &storage.KeyValuePolicy{
					Key:   "k",
					Value: "v",
				},
			},
			expectedPolicySections: []*storage.PolicySection{
				{
					PolicyGroups: []*storage.PolicyGroup{
						{
							FieldName: requiredImageLabel,
							Values: []*storage.PolicyValue{
								{
									Value: "k=v",
								},
							},
						},
					},
				},
			},
		},

		{
			desc: "disallowed annotation",
			policyFields: &storage.PolicyFields{
				DisallowedAnnotation: &storage.KeyValuePolicy{
					Key:   "k",
					Value: "v",
				},
			},
			expectedPolicySections: []*storage.PolicySection{
				{
					PolicyGroups: []*storage.PolicyGroup{
						{
							FieldName: disallowedAnnotation,
							Values: []*storage.PolicyValue{
								{
									Value: "k=v",
								},
							},
						},
					},
				},
			},
		},

		{
			desc: "required annotation",
			policyFields: &storage.PolicyFields{
				RequiredAnnotation: &storage.KeyValuePolicy{
					Key:   "k",
					Value: "v",
				},
			},
			expectedPolicySections: []*storage.PolicySection{
				{
					PolicyGroups: []*storage.PolicyGroup{
						{
							FieldName: requiredAnnotation,
							Values: []*storage.PolicyValue{
								{
									Value: "k=v",
								},
							},
						},
					},
				},
			},
		},

		{
			desc: "required label",
			policyFields: &storage.PolicyFields{
				RequiredLabel: &storage.KeyValuePolicy{
					Key:   "k",
					Value: "v",
				},
			},
			expectedPolicySections: []*storage.PolicySection{
				{
					PolicyGroups: []*storage.PolicyGroup{
						{
							FieldName: requiredLabel,
							Values: []*storage.PolicyValue{
								{
									Value: "k=v",
								},
							},
						},
					},
				},
			},
		},

		{
			desc: "env",
			policyFields: &storage.PolicyFields{
				Env: &storage.KeyValuePolicy{
					Key:          "k",
					Value:        "v",
					EnvVarSource: storage.ContainerConfig_EnvironmentConfig_RESOURCE_FIELD,
				},
			},
			expectedPolicySections: []*storage.PolicySection{
				{
					PolicyGroups: []*storage.PolicyGroup{
						{
							FieldName: environmentVariable,
							Values: []*storage.PolicyValue{
								{
									Value: "RESOURCE_FIELD=k=v",
								},
							},
						},
					},
				},
			},
		},
		{
			desc: "port policy",
			policyFields: &storage.PolicyFields{
				PortPolicy: &storage.PortPolicy{
					Port:     1234,
					Protocol: "protocol",
				},
			},
			expectedPolicySections: []*storage.PolicySection{
				{
					PolicyGroups: []*storage.PolicyGroup{
						{
							FieldName: exposedPort,
							Values: []*storage.PolicyValue{
								{
									Value: "1234",
								},
							},
						},

						{
							FieldName: exposedPortProtocol,
							Values: []*storage.PolicyValue{
								{
									Value: "protocol",
								},
							},
						},
					},
				},
			},
		},

		{
			desc: "volume policy",
			policyFields: &storage.PolicyFields{
				VolumePolicy: &storage.VolumePolicy{
					Name:        "v",
					Source:      "s",
					Destination: "d",
					Type:        "fs",
				},
			},
			expectedPolicySections: []*storage.PolicySection{
				{
					PolicyGroups: []*storage.PolicyGroup{
						{
							FieldName: volumeName,
							Values: []*storage.PolicyValue{
								{
									Value: "v",
								},
							},
						},

						{
							FieldName: volumeType,
							Values: []*storage.PolicyValue{
								{
									Value: "fs",
								},
							},
						},

						{
							FieldName: volumeDestination,
							Values: []*storage.PolicyValue{
								{
									Value: "d",
								},
							},
						},

						{
							FieldName: volumeSource,
							Values: []*storage.PolicyValue{
								{
									Value: "s",
								},
							},
						},
					},
				},
			},
		},

		{
			desc: "image name policy",
			policyFields: &storage.PolicyFields{
				ImageName: &storage.ImageNamePolicy{
					Registry: "reg",
					Remote:   "rem",
					Tag:      "tag",
				},
			},
			expectedPolicySections: []*storage.PolicySection{
				{
					PolicyGroups: []*storage.PolicyGroup{
						{
							FieldName: imageRegistry,
							Values: []*storage.PolicyValue{
								{
									Value: "reg",
								},
							},
						},

						{
							FieldName: imageRemote,
							Values: []*storage.PolicyValue{
								{
									Value: "r/.*rem.*",
								},
							},
						},

						{
							FieldName: imageTag,
							Values: []*storage.PolicyValue{
								{
									Value: "tag",
								},
							},
						},
					},
				},
			},
		},

		{
			desc: "cve",
			policyFields: &storage.PolicyFields{
				Cve: "cve",
			},
			expectedPolicySections: []*storage.PolicySection{
				{
					PolicyGroups: []*storage.PolicyGroup{
						{
							FieldName: cve,
							Values: []*storage.PolicyValue{
								{
									Value: "cve",
								},
							},
						},
					},
				},
			},
		},

		{
			desc: "component",
			policyFields: &storage.PolicyFields{
				Component: &storage.Component{
					Name:    "n",
					Version: "v",
				},
			},
			expectedPolicySections: []*storage.PolicySection{
				{
					PolicyGroups: []*storage.PolicyGroup{
						{
							FieldName: imageComponent,
							Values: []*storage.PolicyValue{
								{
									Value: "n=v",
								},
							},
						},
					},
				},
			},
		},

		{
			desc: "image age days",
			policyFields: &storage.PolicyFields{
				SetImageAgeDays: &storage.PolicyFields_ImageAgeDays{ImageAgeDays: 30},
			},
			expectedPolicySections: []*storage.PolicySection{
				{
					PolicyGroups: []*storage.PolicyGroup{
						{
							FieldName: imageAge,
							Values: []*storage.PolicyValue{
								{
									Value: "30",
								},
							},
						},
					},
				},
			},
		},

		{
			desc: "scan age days",
			policyFields: &storage.PolicyFields{
				SetScanAgeDays: &storage.PolicyFields_ScanAgeDays{ScanAgeDays: 30},
			},
			expectedPolicySections: []*storage.PolicySection{
				{
					PolicyGroups: []*storage.PolicyGroup{
						{
							FieldName: imageScanAge,
							Values: []*storage.PolicyValue{
								{
									Value: "30",
								},
							},
						},
					},
				},
			},
		},

		{
			desc: "unscanned image",
			policyFields: &storage.PolicyFields{
				SetNoScanExists: &storage.PolicyFields_NoScanExists{NoScanExists: true},
			},
			expectedPolicySections: []*storage.PolicySection{
				{
					PolicyGroups: []*storage.PolicyGroup{
						{
							FieldName: unscannedImage,
							Values: []*storage.PolicyValue{
								{
									Value: "true",
								},
							},
						},
					},
				},
			},
		},

		{
			desc: "privileged",
			policyFields: &storage.PolicyFields{
				SetPrivileged: &storage.PolicyFields_Privileged{Privileged: true},
			},
			expectedPolicySections: []*storage.PolicySection{
				{
					PolicyGroups: []*storage.PolicyGroup{
						{
							FieldName: privilegedContainer,
							Values: []*storage.PolicyValue{
								{
									Value: "true",
								},
							},
						},
					},
				},
			},
		},

		{
			desc: "read only root fs",
			policyFields: &storage.PolicyFields{
				SetReadOnlyRootFs: &storage.PolicyFields_ReadOnlyRootFs{ReadOnlyRootFs: true},
			},
			expectedPolicySections: []*storage.PolicySection{
				{
					PolicyGroups: []*storage.PolicyGroup{
						{
							FieldName: readOnlyRootFS,
							Values: []*storage.PolicyValue{
								{
									Value: "true",
								},
							},
						},
					},
				},
			},
		},

		{
			desc: "scope exclusion enabled",
			policyFields: &storage.PolicyFields{
				SetWhitelist: &storage.PolicyFields_WhitelistEnabled{WhitelistEnabled: true},
			},
			expectedPolicySections: []*storage.PolicySection{
				{
					PolicyGroups: []*storage.PolicyGroup{
						{
							FieldName: whitelistsEnabled,
							Values: []*storage.PolicyValue{
								{
									Value: "true",
								},
							},
						},
					},
				},
			},
		},

		{
			desc: "writable host mount",
			policyFields: &storage.PolicyFields{
				HostMountPolicy: &storage.HostMountPolicy{SetReadOnly: &storage.HostMountPolicy_ReadOnly{ReadOnly: true}},
			},
			expectedPolicySections: []*storage.PolicySection{
				{
					PolicyGroups: []*storage.PolicyGroup{
						{
							FieldName: writableHostMount,
							Values: []*storage.PolicyValue{
								{
									Value: "false",
								},
							},
						},
					},
				},
			},
		},

		{
			desc: "dockerfile line rule",
			policyFields: &storage.PolicyFields{
				LineRule: &storage.DockerfileLineRuleField{
					Instruction: "Joseph",
					Value:       "Rules",
				},
			},
			expectedPolicySections: []*storage.PolicySection{
				{
					PolicyGroups: []*storage.PolicyGroup{
						{
							FieldName: dockerfileLine,
							Values: []*storage.PolicyValue{
								{
									Value: "Joseph=Rules",
								},
							},
						},
					},
				},
			},
		},

		{
			desc: "drop capabilities",
			policyFields: &storage.PolicyFields{
				DropCapabilities: []string{"CAP_Joseph", "Rules", "caP_NOT"},
			},
			expectedPolicySections: []*storage.PolicySection{
				{
					PolicyGroups: []*storage.PolicyGroup{
						{
							FieldName:       dropCaps,
							BooleanOperator: storage.BooleanOperator_OR,
							Values: []*storage.PolicyValue{
								{
									Value: "Joseph",
								},
								{
									Value: "Rules",
								},
								{
									Value: "NOT",
								},
							},
						},
					},
				},
			},
		},

		{
			desc: "add capabilities",
			policyFields: &storage.PolicyFields{
				AddCapabilities: []string{"CAP_Joseph", "Rules", "caP_NOT"},
			},
			expectedPolicySections: []*storage.PolicySection{
				{
					PolicyGroups: []*storage.PolicyGroup{
						{
							FieldName:       addCaps,
							BooleanOperator: storage.BooleanOperator_OR,
							Values: []*storage.PolicyValue{
								{
									Value: "Joseph",
								},
								{
									Value: "Rules",
								},
								{
									Value: "NOT",
								},
							},
						},
					},
				},
			},
		},

		{
			desc: "container resource policy",
			policyFields: &storage.PolicyFields{
				ContainerResourcePolicy: &storage.ResourcePolicy{
					CpuResourceRequest: &storage.NumericalPolicy{
						Op:    storage.Comparator_LESS_THAN,
						Value: 1,
					},
					CpuResourceLimit: &storage.NumericalPolicy{
						Op:    storage.Comparator_EQUALS,
						Value: 2,
					},
					MemoryResourceRequest: &storage.NumericalPolicy{
						Op:    storage.Comparator_GREATER_THAN,
						Value: 3,
					},
					MemoryResourceLimit: &storage.NumericalPolicy{
						Op:    storage.Comparator_LESS_THAN_OR_EQUALS,
						Value: 4,
					},
				},
			},
			expectedPolicySections: []*storage.PolicySection{
				{
					PolicyGroups: []*storage.PolicyGroup{
						{
							FieldName: containerCPULimit,
							Values: []*storage.PolicyValue{
								{
									Value: "2.000000",
								},
							},
						},
					},
				},
				{
					PolicyGroups: []*storage.PolicyGroup{
						{
							FieldName: containerCPURequest,
							Values: []*storage.PolicyValue{
								{
									Value: "< 1.000000",
								},
							},
						},
					},
				},
				{
					PolicyGroups: []*storage.PolicyGroup{
						{
							FieldName: containerMemLimit,
							Values: []*storage.PolicyValue{
								{
									Value: "<= 4.000000",
								},
							},
						},
					},
				},
				{
					PolicyGroups: []*storage.PolicyGroup{
						{
							FieldName: containerMemRequest,
							Values: []*storage.PolicyValue{
								{
									Value: "> 3.000000",
								},
							},
						},
					},
				},
			},
		},

		{
			desc: "single container resource policy",
			policyFields: &storage.PolicyFields{
				ContainerResourcePolicy: &storage.ResourcePolicy{
					CpuResourceRequest: &storage.NumericalPolicy{
						Op:    storage.Comparator_LESS_THAN,
						Value: 1,
					},
				},
			},
			expectedPolicySections: []*storage.PolicySection{
				{
					PolicyGroups: []*storage.PolicyGroup{
						{
							FieldName: containerCPURequest,
							Values: []*storage.PolicyValue{
								{
									Value: "< 1.000000",
								},
							},
						},
					},
				},
			},
		},

		{
			desc: "container resource policy OR-ing",
			policyFields: &storage.PolicyFields{
				ContainerResourcePolicy: &storage.ResourcePolicy{
					CpuResourceRequest: &storage.NumericalPolicy{
						Op:    storage.Comparator_LESS_THAN,
						Value: 1,
					},
					CpuResourceLimit: &storage.NumericalPolicy{
						Op:    storage.Comparator_EQUALS,
						Value: 2,
					},
				},
				SetImageAgeDays: &storage.PolicyFields_ImageAgeDays{ImageAgeDays: 30},
			},
			expectedPolicySections: []*storage.PolicySection{
				{
					PolicyGroups: []*storage.PolicyGroup{
						{
							FieldName: imageAge,
							Values: []*storage.PolicyValue{
								{
									Value: "30",
								},
							},
						},
						{
							FieldName: containerCPULimit,
							Values: []*storage.PolicyValue{
								{
									Value: "2.000000",
								},
							},
						},
					},
				},
				{
					PolicyGroups: []*storage.PolicyGroup{
						{
							FieldName: imageAge,
							Values: []*storage.PolicyValue{
								{
									Value: "30",
								},
							},
						},
						{
							FieldName: containerCPURequest,
							Values: []*storage.PolicyValue{
								{
									Value: "< 1.000000",
								},
							},
						},
					},
				},
			},
		},

		{
			desc: "permission policy",
			policyFields: &storage.PolicyFields{
				PermissionPolicy: &storage.PermissionPolicy{
					PermissionLevel: storage.PermissionLevel_ELEVATED_CLUSTER_WIDE,
				},
			},
			expectedPolicySections: []*storage.PolicySection{
				{
					PolicyGroups: []*storage.PolicyGroup{
						{
							FieldName: minimumRBACPermissions,
							Values: []*storage.PolicyValue{
								{
									Value: "ELEVATED_CLUSTER_WIDE",
								},
							},
						},
					},
				},
			},
		},

		{
			desc: "exposure level policy",
			policyFields: &storage.PolicyFields{
				PortExposurePolicy: &storage.PortExposurePolicy{
					ExposureLevels: []storage.PortConfig_ExposureLevel{
						storage.PortConfig_UNSET,
						storage.PortConfig_EXTERNAL,
						storage.PortConfig_NODE,
						storage.PortConfig_INTERNAL,
						storage.PortConfig_HOST,
					},
				},
			},
			expectedPolicySections: []*storage.PolicySection{
				{
					PolicyGroups: []*storage.PolicyGroup{
						{
							FieldName: portExposure,
							Values: []*storage.PolicyValue{
								{
									Value: "UNSET",
								},
								{
									Value: "EXTERNAL",
								},
								{
									Value: "NODE",
								},
								{
									Value: "INTERNAL",
								},
								{
									Value: "HOST",
								},
							},
						},
					},
				},
			},
		},

		{
			desc: "writable mounted volume policy",
			policyFields: &storage.PolicyFields{
				VolumePolicy: &storage.VolumePolicy{
					SetReadOnly: &storage.VolumePolicy_ReadOnly{ReadOnly: false},
				},
			},
			expectedPolicySections: []*storage.PolicySection{
				{
					PolicyGroups: []*storage.PolicyGroup{
						{
							FieldName: writableMountedVolume,
							Values: []*storage.PolicyValue{
								{
									Value: "true",
								},
							},
						},
					},
				},
			},
		},
	}

	for _, tc := range tcs {
		t.Run(tc.desc, func(t *testing.T) {
			got := ConvertPolicyFieldsToSections(tc.policyFields)
			assert.DeepEqual(t, tc.expectedPolicySections, got)
		})
	}
}

func TestMigrateLegacyPolicy(t *testing.T) {
	mockWhitelist := &storage.Exclusion{
		Name: "abcd",
		Image: &storage.Exclusion_Image{
			Name: "some name",
		},
	}
	mockScope := &storage.Scope{
		Label: &storage.Scope_Label{
			Key:   "Joseph",
			Value: "Rules",
		},
	}

	legacyPolicy := &storage.Policy{
		Id:              "Some ID",
		Name:            "Some Name",
		Description:     "Some Description",
		LifecycleStages: nil,
		Whitelists: []*storage.Exclusion{
			mockWhitelist,
		},
		Scope: []*storage.Scope{
			mockScope,
		},
		Fields: &storage.PolicyFields{
			ImageName: &storage.ImageNamePolicy{
				Registry: "123",
				Remote:   "456",
				Tag:      "789",
			},
		},
	}
	expectedSections := []*storage.PolicySection{
		{
			PolicyGroups: []*storage.PolicyGroup{
				{
					FieldName: imageRegistry,
					Values: []*storage.PolicyValue{
						{
							Value: "123",
						},
					},
				},
				{
					FieldName: imageRemote,
					Values: []*storage.PolicyValue{
						{
							Value: "r/.*456.*",
						},
					},
				},
				{
					FieldName: imageTag,
					Values: []*storage.PolicyValue{
						{
							Value: "789",
						},
					},
				},
			},
		},
	}

	t.Run("test migrator", func(t *testing.T) {
		booleanPolicy, err := CloneAndEnsureConverted(legacyPolicy)
		require.NoError(t, err)
		require.Equal(t, bplVersion, booleanPolicy.GetPolicyVersion())
		require.Equal(t, expectedSections, booleanPolicy.GetPolicySections())
	})
}
