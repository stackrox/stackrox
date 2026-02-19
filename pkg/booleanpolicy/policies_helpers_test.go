package booleanpolicy

import (
	"regexp"
	"strings"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/booleanpolicy/augmentedobjs"
	"github.com/stackrox/rox/pkg/booleanpolicy/fieldnames"
	"github.com/stackrox/rox/pkg/booleanpolicy/policyversion"
	"github.com/stackrox/rox/pkg/defaults/policies"
	"github.com/stackrox/rox/pkg/images/types"
	imgUtils "github.com/stackrox/rox/pkg/images/utils"
	"github.com/stackrox/rox/pkg/protoassert"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/sliceutils"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

const (
	writableHostMountPolicyName = "Writeable Host Mount"
	anyHostPathPolicyName       = "Any Host Path"
)

// basePoliciesTestSuite contains the shared state and helpers for all policy test suites.
type basePoliciesTestSuite struct {
	suite.Suite

	defaultPolicies map[string]*storage.Policy
	customPolicies  map[string]*storage.Policy

	deployments             map[string]*storage.Deployment
	images                  map[string]*storage.Image
	deploymentsToImages     map[string][]*storage.Image
	deploymentsToIndicators map[string][]*storage.ProcessIndicator
}

func (s *basePoliciesTestSuite) SetupSuite() {
	defaultPolicies, err := policies.DefaultPolicies()
	s.Require().NoError(err)

	s.defaultPolicies = make(map[string]*storage.Policy, len(defaultPolicies))
	for _, p := range defaultPolicies {
		s.defaultPolicies[p.GetName()] = p
	}

	s.customPolicies = make(map[string]*storage.Policy)
	for _, customPolicy := range []*storage.Policy{
		changeName(policyWithSingleKeyValue(fieldnames.WritableHostMount, "true", false), writableHostMountPolicyName),
		changeName(policyWithSingleKeyValue(fieldnames.VolumeType, "hostpath", false), anyHostPathPolicyName),
	} {
		s.customPolicies[customPolicy.GetName()] = customPolicy
	}
}

func (s *basePoliciesTestSuite) TearDownSuite() {}

func (s *basePoliciesTestSuite) SetupTest() {
	s.deployments = make(map[string]*storage.Deployment)
	s.images = make(map[string]*storage.Image)
	s.deploymentsToImages = make(map[string][]*storage.Image)
	s.deploymentsToIndicators = make(map[string][]*storage.ProcessIndicator)
}

func (s *basePoliciesTestSuite) imageIDFromDep(deployment *storage.Deployment) string {
	s.Require().Len(deployment.GetContainers(), 1, "This function only supports deployments with exactly one container")
	id := deployment.GetContainers()[0].GetImage().GetId()
	s.NotEmpty(id, "Deployment '%s' had no image id", protocompat.MarshalTextString(deployment))
	return id
}

func (s *basePoliciesTestSuite) MustGetPolicy(name string) *storage.Policy {
	p := s.defaultPolicies[name]
	if p != nil {
		return p
	}
	p = s.customPolicies[name]
	if p != nil {
		return p
	}
	s.FailNow("Policy not found: ", name)
	return nil
}

func (s *basePoliciesTestSuite) addDepAndImages(deployment *storage.Deployment, images ...*storage.Image) {
	s.deployments[deployment.GetId()] = deployment
	for _, i := range images {
		s.images[i.GetId()] = i
		s.deploymentsToImages[deployment.GetId()] = append(s.deploymentsToImages[deployment.GetId()], i)
	}
}

func (s *basePoliciesTestSuite) addImage(img *storage.Image) *storage.Image {
	s.images[img.GetId()] = img
	return img
}

func (s *basePoliciesTestSuite) imageWithSignatureVerificationResults(name string, results []*storage.ImageSignatureVerificationResult) *storage.Image {
	imageName, _, err := imgUtils.GenerateImageNameFromString(name)
	if err != nil {
		s.T().Fatalf("failed to parse image name %q: %v", name, err)
	}

	imageName.FullName = name

	img := &storage.Image{
		Id:   uuid.NewV4().String(),
		Name: imageName,
	}

	if results != nil {
		img.SignatureVerificationData = &storage.ImageSignatureVerificationData{
			Results: results,
		}
	}
	return img
}

func (s *basePoliciesTestSuite) addIndicator(deploymentID, name, args, path string, lineage []string, uid uint32) *storage.ProcessIndicator {
	deployment := s.deployments[deploymentID]
	if len(deployment.GetContainers()) == 0 {
		deployment.Containers = []*storage.Container{{Name: uuid.NewV4().String()}}
	}
	lineageInfo := make([]*storage.ProcessSignal_LineageInfo, len(lineage))
	for i, ancestor := range lineage {
		lineageInfo[i] = &storage.ProcessSignal_LineageInfo{
			ParentExecFilePath: ancestor,
		}
	}
	indicator := &storage.ProcessIndicator{
		Id:            uuid.NewV4().String(),
		DeploymentId:  deploymentID,
		ContainerName: deployment.GetContainers()[0].GetName(),
		Signal: &storage.ProcessSignal{
			Name:         name,
			Args:         args,
			ExecFilePath: path,
			Time:         protocompat.TimestampNow(),
			LineageInfo:  lineageInfo,
			Uid:          uid,
		},
	}
	s.deploymentsToIndicators[deploymentID] = append(s.deploymentsToIndicators[deploymentID], indicator)
	return indicator
}

type testCase struct {
	policyName                string
	expectedViolations        map[string][]*storage.Alert_Violation
	expectedProcessViolations map[string][]*storage.ProcessIndicator

	shouldNotMatch             map[string]struct{}
	sampleViolationForMatched  string
	allowUnvalidatedViolations bool
}

func (s *basePoliciesTestSuite) getImagesForDeployment(deployment *storage.Deployment) []*storage.Image {
	images := s.deploymentsToImages[deployment.GetId()]
	if len(images) == 0 {
		return make([]*storage.Image, len(deployment.GetContainers()))
	}
	s.Equal(len(deployment.GetContainers()), len(images))
	return images
}

func (s *basePoliciesTestSuite) getViolations(policy *storage.Policy, dep EnhancedDeployment) Violations {
	matcher, err := BuildDeploymentMatcher(policy)
	s.NoError(err, "deployment matcher creation must succeed")
	violations, err := matcher.MatchDeployment(nil, dep)
	s.NoError(err, "deployment matcher run must succeed")
	s.Empty(violations.ProcessViolation)
	return violations
}

// Free helper functions shared across test files.

func changeName(p *storage.Policy, newName string) *storage.Policy {
	p.Name = newName
	return p
}

func enhancedDeployment(dep *storage.Deployment, images []*storage.Image) EnhancedDeployment {
	return EnhancedDeployment{
		Deployment: dep,
		Images:     images,
		NetworkPoliciesApplied: &augmentedobjs.NetworkPoliciesApplied{
			HasIngressNetworkPolicy: true,
			HasEgressNetworkPolicy:  true,
		},
	}
}

func enhancedDeploymentWithNetworkPolicies(dep *storage.Deployment, images []*storage.Image, netpolApplied *augmentedobjs.NetworkPoliciesApplied) EnhancedDeployment {
	return EnhancedDeployment{
		Deployment:             dep,
		Images:                 images,
		NetworkPoliciesApplied: netpolApplied,
	}
}

func imageWithComponents(components []*storage.EmbeddedImageScanComponent) *storage.Image {
	return &storage.Image{
		Id:   uuid.NewV4().String(),
		Name: &storage.ImageName{FullName: "docker.io/ASFASF", Remote: "ASFASF"},
		Scan: &storage.ImageScan{
			Components: components,
		},
	}
}

func imageWithLayers(layers []*storage.ImageLayer) *storage.Image {
	return &storage.Image{
		Id:   uuid.NewV4().String(),
		Name: &storage.ImageName{FullName: "docker.io/ASFASF", Remote: "ASFASF"},
		Metadata: &storage.ImageMetadata{
			V1: &storage.V1Metadata{
				Layers: layers,
			},
		},
	}
}

func imageWithOS(os string) *storage.Image {
	return &storage.Image{
		Id:   uuid.NewV4().String(),
		Name: &storage.ImageName{FullName: "docker.io/ASFASF", Remote: "ASFASF"},
		Scan: &storage.ImageScan{
			OperatingSystem: os,
		},
	}
}

func deploymentWithImageAnyID(img *storage.Image) *storage.Deployment {
	return deploymentWithImage(uuid.NewV4().String(), img)
}

func deploymentWithImage(id string, img *storage.Image) *storage.Deployment {
	remoteSplit := strings.Split(img.GetName().GetFullName(), "/")
	alphaOnly := regexp.MustCompile("[^A-Za-z]+")
	containerName := alphaOnly.ReplaceAllString(remoteSplit[len(remoteSplit)-1], "")
	return &storage.Deployment{
		Id:         id,
		Containers: []*storage.Container{{Id: img.GetId(), Name: containerName, Image: types.ToContainerImage(img)}},
	}
}

func getViolationsWithAndWithoutCaching(t *testing.T, matcher func(cache *CacheReceptacle) (Violations, error)) Violations {
	violations, err := matcher(nil)
	require.NoError(t, err)

	var cache CacheReceptacle
	violationsWithEmptyCache, err := matcher(&cache)
	require.NoError(t, err)
	assertViolations(t, violations, violationsWithEmptyCache)

	violationsWithNonEmptyCache, err := matcher(&cache)
	require.NoError(t, err)
	assertViolations(t, violations, violationsWithNonEmptyCache)

	return violations
}

func assertViolations(t testing.TB, expected, actual Violations) {
	t.Helper()
	protoassert.Equal(t, expected.ProcessViolation, actual.ProcessViolation)
	protoassert.SlicesEqual(t, expected.AlertViolations, actual.AlertViolations)
}

func policyWithGroups(eventSrc storage.EventSource, groups ...*storage.PolicyGroup) *storage.Policy {
	return &storage.Policy{
		PolicyVersion:  policyversion.CurrentVersion().String(),
		Name:           uuid.NewV4().String(),
		EventSource:    eventSrc,
		PolicySections: []*storage.PolicySection{{PolicyGroups: groups}},
	}
}

func policyGroupWithSingleKeyValue(fieldName, value string, negate bool) *storage.PolicyGroup {
	return &storage.PolicyGroup{FieldName: fieldName, Values: []*storage.PolicyValue{{Value: value}}, Negate: negate}
}

func policyWithSingleKeyValue(fieldName, value string, negate bool) *storage.Policy {
	return policyWithGroups(storage.EventSource_NOT_APPLICABLE, policyGroupWithSingleKeyValue(fieldName, value, negate))
}

func policyWithSingleFieldAndValues(fieldName string, values []string, negate bool, op storage.BooleanOperator) *storage.Policy {
	return policyWithGroups(storage.EventSource_NOT_APPLICABLE, &storage.PolicyGroup{FieldName: fieldName, Values: sliceutils.Map(values, func(val string) *storage.PolicyValue {
		return &storage.PolicyValue{Value: val}
	}), Negate: negate, BooleanOperator: op})
}

// File access test helpers shared across runtime and node criteria tests.

func newActualFileAccessEvent(path string, operation storage.FileAccess_Operation) *storage.FileAccess {
	return &storage.FileAccess{
		File:      &storage.FileAccess_File{ActualPath: path},
		Operation: operation,
	}
}

func newEffectiveFileAccessEvent(path string, operation storage.FileAccess_Operation) *storage.FileAccess {
	return &storage.FileAccess{
		File:      &storage.FileAccess_File{EffectivePath: path},
		Operation: operation,
	}
}

func newDualPathFileAccessEvent(actualPath, effectivePath string, operation storage.FileAccess_Operation) *storage.FileAccess {
	return &storage.FileAccess{
		File: &storage.FileAccess_File{
			ActualPath:    actualPath,
			EffectivePath: effectivePath,
		},
		Operation: operation,
	}
}

func newFileAccessPolicy(eventSource storage.EventSource, operations []storage.FileAccess_Operation, negate bool, paths ...string) *storage.Policy {
	var pathValues []*storage.PolicyValue
	for _, path := range paths {
		pathValues = append(pathValues, &storage.PolicyValue{Value: path})
	}

	policyGroups := []*storage.PolicyGroup{
		{
			FieldName: fieldnames.FilePath,
			Values:    pathValues,
		},
	}

	var operationValues []*storage.PolicyValue
	for _, op := range operations {
		operationValues = append(operationValues, &storage.PolicyValue{Value: op.String()})
	}

	if len(operationValues) != 0 {
		policyGroups = append(policyGroups, &storage.PolicyGroup{
			FieldName: fieldnames.FileOperation,
			Values:    operationValues,
			Negate:    negate,
		})
	}

	return &storage.Policy{
		Id:            uuid.NewV4().String(),
		PolicyVersion: "1.1",
		Name:          "File Access Policy",
		Severity:      storage.Severity_HIGH_SEVERITY,
		Categories:    []string{"File System"},
		PolicySections: []*storage.PolicySection{
			{
				SectionName:  "section 1",
				PolicyGroups: policyGroups,
			},
		},
		LifecycleStages: []storage.LifecycleStage{storage.LifecycleStage_RUNTIME},
		EventSource:     eventSource,
	}
}

func newDualPathPolicy(actualPath, effectivePath string, operations []storage.FileAccess_Operation) *storage.Policy {
	policyGroups := []*storage.PolicyGroup{
		{
			FieldName: fieldnames.FilePath,
			Values:    []*storage.PolicyValue{{Value: actualPath}, {Value: effectivePath}},
		},
	}

	if len(operations) > 0 {
		var operationValues []*storage.PolicyValue
		for _, op := range operations {
			operationValues = append(operationValues, &storage.PolicyValue{Value: op.String()})
		}
		policyGroups = append(policyGroups, &storage.PolicyGroup{
			FieldName: fieldnames.FileOperation,
			Values:    operationValues,
		})
	}

	return &storage.Policy{
		Id:            uuid.NewV4().String(),
		PolicyVersion: "1.1",
		Name:          "Dual Path Policy",
		Severity:      storage.Severity_HIGH_SEVERITY,
		Categories:    []string{"File System"},
		PolicySections: []*storage.PolicySection{
			{
				SectionName:  "section 1",
				PolicyGroups: policyGroups,
			},
		},
		LifecycleStages: []storage.LifecycleStage{storage.LifecycleStage_RUNTIME},
		EventSource:     storage.EventSource_DEPLOYMENT_EVENT,
	}
}

func newMultiSectionPolicy(eventSource storage.EventSource, sections []*storage.PolicySection) *storage.Policy {
	return &storage.Policy{
		Id:              uuid.NewV4().String(),
		PolicyVersion:   "1.1",
		Name:            "Multi-Section Policy",
		Severity:        storage.Severity_HIGH_SEVERITY,
		Categories:      []string{"File System"},
		PolicySections:  sections,
		LifecycleStages: []storage.LifecycleStage{storage.LifecycleStage_RUNTIME},
		EventSource:     eventSource,
	}
}
