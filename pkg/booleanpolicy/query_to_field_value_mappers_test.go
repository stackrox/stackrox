package booleanpolicy

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stretchr/testify/suite"
)

func TestSearchMapper(t *testing.T) {
	suite.Run(t, new(SearchMapperTestSuite))
}

type SearchMapperTestSuite struct {
	suite.Suite
}

func (s *SearchMapperTestSuite) SetupTest() {

}

func (s *SearchMapperTestSuite) TearDownTest() {

}

func (s *SearchMapperTestSuite) testMapSearchString(fieldLabel search.FieldLabel, searchTerms []string, expectedGroup *storage.PolicyGroup, shouldBeAltered, shouldFindMaper bool) {
	policyGroup, fieldsAltered, foundMapper := GetPolicyGroupFromSearchTerms(fieldLabel, searchTerms)
	s.Equal(shouldFindMaper, foundMapper)
	s.Equal(shouldBeAltered, fieldsAltered)
	s.Equal(storage.BooleanOperator_OR, policyGroup.GetBooleanOperator())
	s.Equal(expectedGroup, policyGroup)
}

func (s *SearchMapperTestSuite) testDirectMapSearchString(fieldLabel search.FieldLabel, expectedPolicyField string) {
	searchTerms := []string{"abc"}
	expectedGroup := &storage.PolicyGroup{
		FieldName: expectedPolicyField,
		Values: []*storage.PolicyValue{
			{
				Value: "abc",
			},
		},
	}
	s.testMapSearchString(fieldLabel, searchTerms, expectedGroup, false, true)
}

func (s *SearchMapperTestSuite) TestNoMapper() {
	s.testMapSearchString(search.DropCapabilities, nil, nil, false, false)
}

func (s *SearchMapperTestSuite) TestConvertInstructionKeyword() {
	searchTerms := []string{"abc"}
	expectedGroup := &storage.PolicyGroup{
		FieldName: DockerfileLine,
		Values: []*storage.PolicyValue{
			{
				Value: "abc=",
			},
		},
	}
	s.testMapSearchString(search.DockerfileInstructionKeyword, searchTerms, expectedGroup, false, true)
}

func (s *SearchMapperTestSuite) TestConvertInstructionValue() {
	searchTerms := []string{"abc"}
	expectedGroup := &storage.PolicyGroup{
		FieldName: DockerfileLine,
		Values: []*storage.PolicyValue{
			{
				Value: "=abc",
			},
		},
	}
	s.testMapSearchString(search.DockerfileInstructionValue, searchTerms, expectedGroup, false, true)
}

func (s *SearchMapperTestSuite) TestConvertEnvironmentKey() {
	searchTerms := []string{"abc"}
	expectedGroup := &storage.PolicyGroup{
		FieldName: EnvironmentVariable,
		Values: []*storage.PolicyValue{
			{
				Value: "abc==",
			},
		},
	}
	s.testMapSearchString(search.EnvironmentKey, searchTerms, expectedGroup, false, true)
}

func (s *SearchMapperTestSuite) TestConvertEnvironmentValue() {
	searchTerms := []string{"abc"}
	expectedGroup := &storage.PolicyGroup{
		FieldName: EnvironmentVariable,
		Values: []*storage.PolicyValue{
			{
				Value: "=abc=",
			},
		},
	}
	s.testMapSearchString(search.EnvironmentValue, searchTerms, expectedGroup, false, true)
}

func (s *SearchMapperTestSuite) TestConvertEnvironmentVarSrc() {
	searchTerms := []string{"abc"}
	expectedGroup := &storage.PolicyGroup{
		FieldName: EnvironmentVariable,
		Values: []*storage.PolicyValue{
			{
				Value: "==abc",
			},
		},
	}
	s.testMapSearchString(search.EnvironmentVarSrc, searchTerms, expectedGroup, false, true)
}

func (s *SearchMapperTestSuite) TestConvertAnnotation() {
	s.testDirectMapSearchString(search.Annotation, DisallowedAnnotation)
}

func (s *SearchMapperTestSuite) TestConvertImageLabel() {
	s.testDirectMapSearchString(search.ImageLabel, DisallowedImageLabel)
}

func (s *SearchMapperTestSuite) TestConvertVolumeReadonly() {
	searchTerms := []string{"abc", "true"}
	expectedGroup := &storage.PolicyGroup{
		FieldName: WritableVolume,
		Values: []*storage.PolicyValue{
			{
				Value: "false",
			},
		},
	}
	s.testMapSearchString(search.VolumeReadonly, searchTerms, expectedGroup, true, true)
}

func (s *SearchMapperTestSuite) TestConvertImageCreatedTime() {
	// We only convert searches of the form >Nd.  Other searches have no equivalent policy fields.
	searchTerms := []string{"abc", ">30d", "<50d"}
	expectedGroup := &storage.PolicyGroup{
		FieldName: ImageAge,
		Values: []*storage.PolicyValue{
			{
				Value: "30",
			},
		},
	}
	s.testMapSearchString(search.ImageCreatedTime, searchTerms, expectedGroup, true, true)
}

func (s *SearchMapperTestSuite) TestConvertImageScanTime() {
	searchTerms := []string{"abc", ">1337D"}
	expectedGroup := &storage.PolicyGroup{
		FieldName: ImageScanAge,
		Values: []*storage.PolicyValue{
			{
				Value: "1337",
			},
		},
	}
	s.testMapSearchString(search.ImageScanTime, searchTerms, expectedGroup, true, true)
}

func (s *SearchMapperTestSuite) TestConvertServiceAccountPermissionLevel() {
	searchTerms := []string{"abc"}
	s.testMapSearchString(search.ServiceAccountPermissionLevel, searchTerms, nil, false, true)
	searchTermsWithResults := []string{"ELEVATED_IN_NAMESPACE", "CLUSTER_ADMIN"}
	expectedGroup := &storage.PolicyGroup{
		FieldName: MinimumRBACPermissions,
		Values: []*storage.PolicyValue{
			{
				Value: "ELEVATED_IN_NAMESPACE",
			},
		},
	}
	s.testMapSearchString(search.ServiceAccountPermissionLevel, searchTermsWithResults, expectedGroup, true, true)
}

func (s *SearchMapperTestSuite) TestConvertExposureLevel() {
	s.testDirectMapSearchString(search.ExposureLevel, PortExposure)
}

func (s *SearchMapperTestSuite) TestConvertAddCapabilities() {
	s.testDirectMapSearchString(search.AddCapabilities, AddCaps)
}

func (s *SearchMapperTestSuite) TestConvertCVE() {
	s.testDirectMapSearchString(search.CVE, CVE)
}

func (s *SearchMapperTestSuite) TestConvertCVSS() {
	searchTerms := []string{">88", "7644"}
	expectedGroup := &storage.PolicyGroup{
		FieldName: CVSS,
		Values: []*storage.PolicyValue{
			{
				Value: "> 88",
			},
			{
				Value: "7644",
			},
		},
	}
	s.testMapSearchString(search.CVSS, searchTerms, expectedGroup, false, true)
}

func (s *SearchMapperTestSuite) TestConvertCPUCoresLimit() {
	searchTerms := []string{"5", "<7", ">=98"}
	expectedGroup := &storage.PolicyGroup{
		FieldName: ContainerCPULimit,
		Values: []*storage.PolicyValue{
			{
				Value: "5",
			},
			{
				Value: "< 7",
			},
			{
				Value: ">= 98",
			},
		},
	}
	s.testMapSearchString(search.CPUCoresLimit, searchTerms, expectedGroup, false, true)
}

func (s *SearchMapperTestSuite) TestConvertCPUCoresRequest() {
	searchTerms := []string{"5", "<7", ">=98"}
	expectedGroup := &storage.PolicyGroup{
		FieldName: ContainerCPURequest,
		Values: []*storage.PolicyValue{
			{
				Value: "5",
			},
			{
				Value: "< 7",
			},
			{
				Value: ">= 98",
			},
		},
	}
	s.testMapSearchString(search.CPUCoresRequest, searchTerms, expectedGroup, false, true)
}

func (s *SearchMapperTestSuite) TestConvertMemoryLimit() {
	searchTerms := []string{"5", "<7", ">=98"}
	expectedGroup := &storage.PolicyGroup{
		FieldName: ContainerMemLimit,
		Values: []*storage.PolicyValue{
			{
				Value: "5",
			},
			{
				Value: "< 7",
			},
			{
				Value: ">= 98",
			},
		},
	}
	s.testMapSearchString(search.MemoryLimit, searchTerms, expectedGroup, false, true)
}

func (s *SearchMapperTestSuite) TestConvertMemoryRequest() {
	searchTerms := []string{"5", "<7", ">=98"}
	expectedGroup := &storage.PolicyGroup{
		FieldName: ContainerMemRequest,
		Values: []*storage.PolicyValue{
			{
				Value: "5",
			},
			{
				Value: "< 7",
			},
			{
				Value: ">= 98",
			},
		},
	}
	s.testMapSearchString(search.MemoryRequest, searchTerms, expectedGroup, false, true)
}

func (s *SearchMapperTestSuite) TestConvertFixedBy() {
	s.testDirectMapSearchString(search.FixedBy, FixedBy)
}

func (s *SearchMapperTestSuite) TestConvertComponent() {
	searchTerms := []string{"abc"}
	expectedGroup := &storage.PolicyGroup{
		FieldName: ImageComponent,
		Values: []*storage.PolicyValue{
			{
				Value: "abc=",
			},
		},
	}
	s.testMapSearchString(search.Component, searchTerms, expectedGroup, false, true)
}

func (s *SearchMapperTestSuite) TestConvertComponentVersion() {
	searchTerms := []string{"abc"}
	expectedGroup := &storage.PolicyGroup{
		FieldName: ImageComponent,
		Values: []*storage.PolicyValue{
			{
				Value: "=abc",
			},
		},
	}
	s.testMapSearchString(search.ComponentVersion, searchTerms, expectedGroup, false, true)
}

func (s *SearchMapperTestSuite) TestConvertImageRegistry() {
	s.testDirectMapSearchString(search.ImageRegistry, ImageRegistry)
}

func (s *SearchMapperTestSuite) TestConvertImageRemote() {
	s.testDirectMapSearchString(search.ImageRemote, ImageRemote)
}

func (s *SearchMapperTestSuite) TestConvertImageTag() {
	s.testDirectMapSearchString(search.ImageTag, ImageTag)
}

func (s *SearchMapperTestSuite) TestConvertPort() {
	s.testDirectMapSearchString(search.Port, Port)
}

func (s *SearchMapperTestSuite) TestConvertPrivileged() {
	s.testDirectMapSearchString(search.Privileged, Privileged)
}

func (s *SearchMapperTestSuite) TestConvertProcessAncestor() {
	s.testDirectMapSearchString(search.ProcessAncestor, ProcessAncestor)
}

func (s *SearchMapperTestSuite) TestConvertProcessArguments() {
	s.testDirectMapSearchString(search.ProcessArguments, ProcessArguments)
}

func (s *SearchMapperTestSuite) TestConvertProcessName() {
	s.testDirectMapSearchString(search.ProcessName, ProcessName)
}

func (s *SearchMapperTestSuite) TestConvertProcessUID() {
	s.testDirectMapSearchString(search.ProcessUID, ProcessUID)
}

func (s *SearchMapperTestSuite) TestConvertPortProtocol() {
	s.testDirectMapSearchString(search.PortProtocol, Protocol)
}

func (s *SearchMapperTestSuite) TestConvertReadOnlyRootFilesystem() {
	s.testDirectMapSearchString(search.ProcessArguments, ProcessArguments)
}

func (s *SearchMapperTestSuite) TestConvertVolumeDestination() {
	s.testDirectMapSearchString(search.VolumeDestination, VolumeDestination)
}

func (s *SearchMapperTestSuite) TestConvertVolumeName() {
	s.testDirectMapSearchString(search.VolumeName, VolumeName)
}

func (s *SearchMapperTestSuite) TestConvertVolumeSource() {
	s.testDirectMapSearchString(search.VolumeSource, VolumeSource)
}

func (s *SearchMapperTestSuite) TestConvertVolumeType() {
	s.testDirectMapSearchString(search.VolumeType, VolumeType)
}
