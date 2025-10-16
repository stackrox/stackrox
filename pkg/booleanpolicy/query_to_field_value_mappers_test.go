package booleanpolicy

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/booleanpolicy/fieldnames"
	"github.com/stackrox/rox/pkg/protoassert"
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
	protoassert.Equal(s.T(), expectedGroup, policyGroup)
}

func (s *SearchMapperTestSuite) testDirectMapSearchString(fieldLabel search.FieldLabel, expectedPolicyField string) {
	searchTerms := []string{"abc"}
	pv := &storage.PolicyValue{}
	pv.SetValue("abc")
	expectedGroup := &storage.PolicyGroup{}
	expectedGroup.SetFieldName(expectedPolicyField)
	expectedGroup.SetValues([]*storage.PolicyValue{
		pv,
	})
	s.testMapSearchString(fieldLabel, searchTerms, expectedGroup, false, true)
}

func (s *SearchMapperTestSuite) TestNoMapper() {
	s.testMapSearchString(search.DropCapabilities, nil, nil, false, false)
}

func (s *SearchMapperTestSuite) TestConvertInstructionKeyword() {
	searchTerms := []string{"abc"}
	pv := &storage.PolicyValue{}
	pv.SetValue("abc=")
	expectedGroup := &storage.PolicyGroup{}
	expectedGroup.SetFieldName(fieldnames.DockerfileLine)
	expectedGroup.SetValues([]*storage.PolicyValue{
		pv,
	})
	s.testMapSearchString(search.DockerfileInstructionKeyword, searchTerms, expectedGroup, false, true)
}

func (s *SearchMapperTestSuite) TestConvertInstructionValue() {
	searchTerms := []string{"abc"}
	pv := &storage.PolicyValue{}
	pv.SetValue("=abc")
	expectedGroup := &storage.PolicyGroup{}
	expectedGroup.SetFieldName(fieldnames.DockerfileLine)
	expectedGroup.SetValues([]*storage.PolicyValue{
		pv,
	})
	s.testMapSearchString(search.DockerfileInstructionValue, searchTerms, expectedGroup, false, true)
}

func (s *SearchMapperTestSuite) TestConvertEnvironmentKey() {
	searchTerms := []string{"abc"}
	pv := &storage.PolicyValue{}
	pv.SetValue("=abc=")
	expectedGroup := &storage.PolicyGroup{}
	expectedGroup.SetFieldName(fieldnames.EnvironmentVariable)
	expectedGroup.SetValues([]*storage.PolicyValue{
		pv,
	})
	s.testMapSearchString(search.EnvironmentKey, searchTerms, expectedGroup, false, true)
}

func (s *SearchMapperTestSuite) TestConvertEnvironmentValue() {
	searchTerms := []string{"abc"}
	pv := &storage.PolicyValue{}
	pv.SetValue("==abc")
	expectedGroup := &storage.PolicyGroup{}
	expectedGroup.SetFieldName(fieldnames.EnvironmentVariable)
	expectedGroup.SetValues([]*storage.PolicyValue{
		pv,
	})
	s.testMapSearchString(search.EnvironmentValue, searchTerms, expectedGroup, false, true)
}

func (s *SearchMapperTestSuite) TestConvertEnvironmentVarSrc() {
	searchTerms := []string{"abc"}
	pv := &storage.PolicyValue{}
	pv.SetValue("abc==")
	expectedGroup := &storage.PolicyGroup{}
	expectedGroup.SetFieldName(fieldnames.EnvironmentVariable)
	expectedGroup.SetValues([]*storage.PolicyValue{
		pv,
	})
	s.testMapSearchString(search.EnvironmentVarSrc, searchTerms, expectedGroup, false, true)
}

func (s *SearchMapperTestSuite) TestConvertAnnotation() {
	searchTerms := []string{"abc"}
	pv := &storage.PolicyValue{}
	pv.SetValue("abc=")
	expectedGroup := &storage.PolicyGroup{}
	expectedGroup.SetFieldName(fieldnames.DisallowedAnnotation)
	expectedGroup.SetValues([]*storage.PolicyValue{
		pv,
	})
	s.testMapSearchString(search.DeploymentAnnotation, searchTerms, expectedGroup, false, true)
}

func (s *SearchMapperTestSuite) TestConvertImageLabel() {
	searchTerms := []string{"abc"}
	pv := &storage.PolicyValue{}
	pv.SetValue("abc=")
	expectedGroup := &storage.PolicyGroup{}
	expectedGroup.SetFieldName(fieldnames.DisallowedImageLabel)
	expectedGroup.SetValues([]*storage.PolicyValue{
		pv,
	})
	s.testMapSearchString(search.ImageLabel, searchTerms, expectedGroup, false, true)
}

func (s *SearchMapperTestSuite) TestConvertVolumeReadonly() {
	searchTerms := []string{"abc", "true"}
	pv := &storage.PolicyValue{}
	pv.SetValue("false")
	expectedGroup := &storage.PolicyGroup{}
	expectedGroup.SetFieldName(fieldnames.WritableMountedVolume)
	expectedGroup.SetValues([]*storage.PolicyValue{
		pv,
	})
	s.testMapSearchString(search.VolumeReadonly, searchTerms, expectedGroup, true, true)
}

func (s *SearchMapperTestSuite) TestConvertImageCreatedTime() {
	// We only convert searches of the form >Nd.  Other searches have no equivalent policy fields.
	searchTerms := []string{"abc", ">30d", "<50d"}
	pv := &storage.PolicyValue{}
	pv.SetValue("30")
	expectedGroup := &storage.PolicyGroup{}
	expectedGroup.SetFieldName(fieldnames.ImageAge)
	expectedGroup.SetValues([]*storage.PolicyValue{
		pv,
	})
	s.testMapSearchString(search.ImageCreatedTime, searchTerms, expectedGroup, true, true)
}

func (s *SearchMapperTestSuite) TestConvertImageScanTime() {
	searchTerms := []string{"abc", ">1337D"}
	pv := &storage.PolicyValue{}
	pv.SetValue("1337")
	expectedGroup := &storage.PolicyGroup{}
	expectedGroup.SetFieldName(fieldnames.ImageScanAge)
	expectedGroup.SetValues([]*storage.PolicyValue{
		pv,
	})
	s.testMapSearchString(search.ImageScanTime, searchTerms, expectedGroup, true, true)
}

func (s *SearchMapperTestSuite) TestConvertServiceAccountPermissionLevel() {
	searchTerms := []string{"abc"}
	s.testMapSearchString(search.ServiceAccountPermissionLevel, searchTerms, nil, false, true)
	searchTermsWithResults := []string{"ELEVATED_IN_NAMESPACE", "CLUSTER_ADMIN"}
	pv := &storage.PolicyValue{}
	pv.SetValue("ELEVATED_IN_NAMESPACE")
	expectedGroup := &storage.PolicyGroup{}
	expectedGroup.SetFieldName(fieldnames.MinimumRBACPermissions)
	expectedGroup.SetValues([]*storage.PolicyValue{
		pv,
	})
	s.testMapSearchString(search.ServiceAccountPermissionLevel, searchTermsWithResults, expectedGroup, true, true)
}

func (s *SearchMapperTestSuite) TestConvertExposureLevel() {
	s.testDirectMapSearchString(search.ExposureLevel, fieldnames.PortExposure)
}

func (s *SearchMapperTestSuite) TestConvertAddCapabilities() {
	s.testDirectMapSearchString(search.AddCapabilities, fieldnames.AddCaps)
}

func (s *SearchMapperTestSuite) TestConvertCVE() {
	s.testDirectMapSearchString(search.CVE, fieldnames.CVE)
}

func (s *SearchMapperTestSuite) TestConvertCVSS() {
	searchTerms := []string{">88", "7644"}
	pv := &storage.PolicyValue{}
	pv.SetValue("> 88")
	pv2 := &storage.PolicyValue{}
	pv2.SetValue("7644")
	expectedGroup := &storage.PolicyGroup{}
	expectedGroup.SetFieldName(fieldnames.CVSS)
	expectedGroup.SetValues([]*storage.PolicyValue{
		pv,
		pv2,
	})
	s.testMapSearchString(search.CVSS, searchTerms, expectedGroup, false, true)
}

func (s *SearchMapperTestSuite) TestConvertNVDCVSS() {
	searchTerms := []string{">88", "7644"}
	pv := &storage.PolicyValue{}
	pv.SetValue("> 88")
	pv2 := &storage.PolicyValue{}
	pv2.SetValue("7644")
	expectedGroup := &storage.PolicyGroup{}
	expectedGroup.SetFieldName(fieldnames.NvdCvss)
	expectedGroup.SetValues([]*storage.PolicyValue{
		pv,
		pv2,
	})
	s.testMapSearchString(search.NVDCVSS, searchTerms, expectedGroup, false, true)
}

func (s *SearchMapperTestSuite) TestConvertCPUCoresLimit() {
	searchTerms := []string{"5", "<7", ">=98"}
	pv := &storage.PolicyValue{}
	pv.SetValue("5")
	pv2 := &storage.PolicyValue{}
	pv2.SetValue("< 7")
	pv3 := &storage.PolicyValue{}
	pv3.SetValue(">= 98")
	expectedGroup := &storage.PolicyGroup{}
	expectedGroup.SetFieldName(fieldnames.ContainerCPULimit)
	expectedGroup.SetValues([]*storage.PolicyValue{
		pv,
		pv2,
		pv3,
	})
	s.testMapSearchString(search.CPUCoresLimit, searchTerms, expectedGroup, false, true)
}

func (s *SearchMapperTestSuite) TestConvertCPUCoresRequest() {
	searchTerms := []string{"5", "<7", ">=98"}
	pv := &storage.PolicyValue{}
	pv.SetValue("5")
	pv2 := &storage.PolicyValue{}
	pv2.SetValue("< 7")
	pv3 := &storage.PolicyValue{}
	pv3.SetValue(">= 98")
	expectedGroup := &storage.PolicyGroup{}
	expectedGroup.SetFieldName(fieldnames.ContainerCPURequest)
	expectedGroup.SetValues([]*storage.PolicyValue{
		pv,
		pv2,
		pv3,
	})
	s.testMapSearchString(search.CPUCoresRequest, searchTerms, expectedGroup, false, true)
}

func (s *SearchMapperTestSuite) TestConvertMemoryLimit() {
	searchTerms := []string{"5", "<7", ">=98"}
	pv := &storage.PolicyValue{}
	pv.SetValue("5")
	pv2 := &storage.PolicyValue{}
	pv2.SetValue("< 7")
	pv3 := &storage.PolicyValue{}
	pv3.SetValue(">= 98")
	expectedGroup := &storage.PolicyGroup{}
	expectedGroup.SetFieldName(fieldnames.ContainerMemLimit)
	expectedGroup.SetValues([]*storage.PolicyValue{
		pv,
		pv2,
		pv3,
	})
	s.testMapSearchString(search.MemoryLimit, searchTerms, expectedGroup, false, true)
}

func (s *SearchMapperTestSuite) TestConvertMemoryRequest() {
	searchTerms := []string{"5", "<7", ">=98"}
	pv := &storage.PolicyValue{}
	pv.SetValue("5")
	pv2 := &storage.PolicyValue{}
	pv2.SetValue("< 7")
	pv3 := &storage.PolicyValue{}
	pv3.SetValue(">= 98")
	expectedGroup := &storage.PolicyGroup{}
	expectedGroup.SetFieldName(fieldnames.ContainerMemRequest)
	expectedGroup.SetValues([]*storage.PolicyValue{
		pv,
		pv2,
		pv3,
	})
	s.testMapSearchString(search.MemoryRequest, searchTerms, expectedGroup, false, true)
}

func (s *SearchMapperTestSuite) TestConvertFixedBy() {
	s.testDirectMapSearchString(search.FixedBy, fieldnames.FixedBy)
}

func (s *SearchMapperTestSuite) TestConvertComponent() {
	searchTerms := []string{"abc"}
	pv := &storage.PolicyValue{}
	pv.SetValue("abc=")
	expectedGroup := &storage.PolicyGroup{}
	expectedGroup.SetFieldName(fieldnames.ImageComponent)
	expectedGroup.SetValues([]*storage.PolicyValue{
		pv,
	})
	s.testMapSearchString(search.Component, searchTerms, expectedGroup, false, true)
}

func (s *SearchMapperTestSuite) TestConvertComponentVersion() {
	searchTerms := []string{"abc"}
	pv := &storage.PolicyValue{}
	pv.SetValue("=abc")
	expectedGroup := &storage.PolicyGroup{}
	expectedGroup.SetFieldName(fieldnames.ImageComponent)
	expectedGroup.SetValues([]*storage.PolicyValue{
		pv,
	})
	s.testMapSearchString(search.ComponentVersion, searchTerms, expectedGroup, false, true)
}

func (s *SearchMapperTestSuite) TestConvertImageRegistry() {
	s.testDirectMapSearchString(search.ImageRegistry, fieldnames.ImageRegistry)
}

func (s *SearchMapperTestSuite) TestConvertImageRemote() {
	s.testDirectMapSearchString(search.ImageRemote, fieldnames.ImageRemote)
}

func (s *SearchMapperTestSuite) TestConvertImageTag() {
	s.testDirectMapSearchString(search.ImageTag, fieldnames.ImageTag)
}

func (s *SearchMapperTestSuite) TestConvertPort() {
	s.testDirectMapSearchString(search.Port, fieldnames.ExposedPort)
}

func (s *SearchMapperTestSuite) TestConvertPrivileged() {
	s.testDirectMapSearchString(search.Privileged, fieldnames.PrivilegedContainer)
}

func (s *SearchMapperTestSuite) TestConvertProcessAncestor() {
	s.testDirectMapSearchString(search.ProcessAncestor, fieldnames.ProcessAncestor)
}

func (s *SearchMapperTestSuite) TestConvertProcessArguments() {
	s.testDirectMapSearchString(search.ProcessArguments, fieldnames.ProcessArguments)
}

func (s *SearchMapperTestSuite) TestConvertProcessName() {
	s.testDirectMapSearchString(search.ProcessName, fieldnames.ProcessName)
}

func (s *SearchMapperTestSuite) TestConvertProcessUID() {
	s.testDirectMapSearchString(search.ProcessUID, fieldnames.ProcessUID)
}

func (s *SearchMapperTestSuite) TestConvertPortProtocol() {
	s.testDirectMapSearchString(search.PortProtocol, fieldnames.ExposedPortProtocol)
}

func (s *SearchMapperTestSuite) TestConvertReadOnlyRootFilesystem() {
	s.testDirectMapSearchString(search.ReadOnlyRootFilesystem, fieldnames.ReadOnlyRootFS)
}

func (s *SearchMapperTestSuite) TestConvertVolumeDestination() {
	s.testDirectMapSearchString(search.VolumeDestination, fieldnames.VolumeDestination)
}

func (s *SearchMapperTestSuite) TestConvertVolumeName() {
	s.testDirectMapSearchString(search.VolumeName, fieldnames.VolumeName)
}

func (s *SearchMapperTestSuite) TestConvertVolumeSource() {
	s.testDirectMapSearchString(search.VolumeSource, fieldnames.VolumeSource)
}

func (s *SearchMapperTestSuite) TestConvertVolumeType() {
	s.testDirectMapSearchString(search.VolumeType, fieldnames.VolumeType)
}
