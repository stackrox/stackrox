package booleanpolicy

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/booleanpolicy/fieldnames"
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
					FieldName: fieldnames.CVSS,
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
				PolicyVersion: Version,
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
				PolicyVersion:  Version,
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
				PolicyVersion:  Version,
				PolicySections: sections,
			},
		},
		{
			desc: "valid noop",
			policy: &storage.Policy{
				PolicyVersion:  Version,
				PolicySections: sections,
			},
			expected: &storage.Policy{
				PolicyVersion:  Version,
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
							FieldName: fieldnames.CVSS,
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
							FieldName: fieldnames.FixedBy,
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
							FieldName: fieldnames.ProcessName,
							Values:    []*storage.PolicyValue{{Value: "process"}},
						},

						{
							FieldName: fieldnames.ProcessAncestor,
							Values:    []*storage.PolicyValue{{Value: "parent"}},
						},

						{
							FieldName: fieldnames.ProcessArguments,
							Values:    []*storage.PolicyValue{{Value: "--arg 1"}},
						},

						{
							FieldName: fieldnames.ProcessUID,
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
							FieldName: fieldnames.DisallowedImageLabel,
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
							FieldName: fieldnames.RequiredImageLabel,
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
							FieldName: fieldnames.DisallowedAnnotation,
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
							FieldName: fieldnames.RequiredAnnotation,
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
							FieldName: fieldnames.RequiredLabel,
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
							FieldName: fieldnames.EnvironmentVariable,
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
							FieldName: fieldnames.Port,
							Values: []*storage.PolicyValue{
								{
									Value: "1234",
								},
							},
						},

						{
							FieldName: fieldnames.Protocol,
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
							FieldName: fieldnames.VolumeName,
							Values: []*storage.PolicyValue{
								{
									Value: "v",
								},
							},
						},

						{
							FieldName: fieldnames.VolumeType,
							Values: []*storage.PolicyValue{
								{
									Value: "fs",
								},
							},
						},

						{
							FieldName: fieldnames.VolumeDestination,
							Values: []*storage.PolicyValue{
								{
									Value: "d",
								},
							},
						},

						{
							FieldName: fieldnames.VolumeSource,
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
							FieldName: fieldnames.ImageRegistry,
							Values: []*storage.PolicyValue{
								{
									Value: "reg",
								},
							},
						},

						{
							FieldName: fieldnames.ImageRemote,
							Values: []*storage.PolicyValue{
								{
									Value: "r/.*rem.*",
								},
							},
						},

						{
							FieldName: fieldnames.ImageTag,
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
							FieldName: fieldnames.CVE,
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
							FieldName: fieldnames.ImageComponent,
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
							FieldName: fieldnames.ImageAge,
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
							FieldName: fieldnames.ImageScanAge,
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
							FieldName: fieldnames.UnscannedImage,
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
							FieldName: fieldnames.Privileged,
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
							FieldName: fieldnames.ReadOnlyRootFS,
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
			desc: "whitelist enabled",
			policyFields: &storage.PolicyFields{
				SetWhitelist: &storage.PolicyFields_WhitelistEnabled{WhitelistEnabled: true},
			},
			expectedPolicySections: []*storage.PolicySection{
				{
					PolicyGroups: []*storage.PolicyGroup{
						{
							FieldName: fieldnames.WhitelistsEnabled,
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
							FieldName: fieldnames.WritableHostMount,
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
							FieldName: fieldnames.DockerfileLine,
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
				DropCapabilities: []string{"Joseph", "Rules"},
			},
			expectedPolicySections: []*storage.PolicySection{
				{
					PolicyGroups: []*storage.PolicyGroup{
						{
							FieldName:       fieldnames.DropCaps,
							BooleanOperator: storage.BooleanOperator_OR,
							Values: []*storage.PolicyValue{
								{
									Value: "Joseph",
								},
								{
									Value: "Rules",
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
				AddCapabilities: []string{"Joseph", "Rules"},
			},
			expectedPolicySections: []*storage.PolicySection{
				{
					PolicyGroups: []*storage.PolicyGroup{
						{
							FieldName:       fieldnames.AddCaps,
							BooleanOperator: storage.BooleanOperator_OR,
							Values: []*storage.PolicyValue{
								{
									Value: "Joseph",
								},
								{
									Value: "Rules",
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
							FieldName: fieldnames.ContainerCPULimit,
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
							FieldName: fieldnames.ContainerCPURequest,
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
							FieldName: fieldnames.ContainerMemLimit,
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
							FieldName: fieldnames.ContainerMemRequest,
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
							FieldName: fieldnames.ContainerCPURequest,
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
							FieldName: fieldnames.ImageAge,
							Values: []*storage.PolicyValue{
								{
									Value: "30",
								},
							},
						},
						{
							FieldName: fieldnames.ContainerCPULimit,
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
							FieldName: fieldnames.ImageAge,
							Values: []*storage.PolicyValue{
								{
									Value: "30",
								},
							},
						},
						{
							FieldName: fieldnames.ContainerCPURequest,
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
							FieldName: fieldnames.MinimumRBACPermissions,
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
							FieldName: fieldnames.PortExposure,
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
	}

	for _, tc := range tcs {
		t.Run(tc.desc, func(t *testing.T) {
			got := ConvertPolicyFieldsToSections(tc.policyFields)
			assert.DeepEqual(t, tc.expectedPolicySections, got)
		})
	}
}

func TestMigrateLegacyPolicy(t *testing.T) {
	mockWhitelist := &storage.Whitelist{
		Name: "abcd",
		Image: &storage.Whitelist_Image{
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
		Whitelists: []*storage.Whitelist{
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
					FieldName: fieldnames.ImageRegistry,
					Values: []*storage.PolicyValue{
						{
							Value: "123",
						},
					},
				},
				{
					FieldName: fieldnames.ImageRemote,
					Values: []*storage.PolicyValue{
						{
							Value: "r/.*456.*",
						},
					},
				},
				{
					FieldName: fieldnames.ImageTag,
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
		require.Equal(t, Version, booleanPolicy.GetPolicyVersion())
		require.Equal(t, expectedSections, booleanPolicy.GetPolicySections())
	})
}
