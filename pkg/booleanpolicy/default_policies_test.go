package booleanpolicy

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/booleanpolicy/augmentedobjs"
	"github.com/stackrox/rox/pkg/booleanpolicy/fieldnames"
	"github.com/stackrox/rox/pkg/booleanpolicy/policyversion"
	"github.com/stackrox/rox/pkg/booleanpolicy/violationmessages/printer"
	"github.com/stackrox/rox/pkg/defaults/policies"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/images/types"
	imgUtils "github.com/stackrox/rox/pkg/images/utils"
	"github.com/stackrox/rox/pkg/kubernetes"
	policyUtils "github.com/stackrox/rox/pkg/policies"
	"github.com/stackrox/rox/pkg/protoassert"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/protoconv"
	"github.com/stackrox/rox/pkg/protoutils"
	"github.com/stackrox/rox/pkg/readable"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/signatures"
	"github.com/stackrox/rox/pkg/sliceutils"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"google.golang.org/protobuf/proto"
)

const (
	writableHostMountPolicyName = "Writeable Host Mount"
	anyHostPathPolicyName       = "Any Host Path"
)

func changeName(p *storage.Policy, newName string) *storage.Policy {
	p.SetName(newName)
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

func TestDefaultPolicies(t *testing.T) {
	suite.Run(t, new(DefaultPoliciesTestSuite))
}

type DefaultPoliciesTestSuite struct {
	suite.Suite

	defaultPolicies map[string]*storage.Policy
	customPolicies  map[string]*storage.Policy

	deployments             map[string]*storage.Deployment
	images                  map[string]*storage.Image
	deploymentsToImages     map[string][]*storage.Image
	deploymentsToIndicators map[string][]*storage.ProcessIndicator
}

func (suite *DefaultPoliciesTestSuite) SetupSuite() {
	defaultPolicies, err := policies.DefaultPolicies()
	suite.Require().NoError(err)

	suite.defaultPolicies = make(map[string]*storage.Policy, len(defaultPolicies))
	for _, p := range defaultPolicies {
		suite.defaultPolicies[p.GetName()] = p
	}

	suite.customPolicies = make(map[string]*storage.Policy)
	for _, customPolicy := range []*storage.Policy{
		changeName(policyWithSingleKeyValue(fieldnames.WritableHostMount, "true", false), writableHostMountPolicyName),
		changeName(policyWithSingleKeyValue(fieldnames.VolumeType, "hostpath", false), anyHostPathPolicyName),
	} {
		suite.customPolicies[customPolicy.GetName()] = customPolicy
	}
}

func (suite *DefaultPoliciesTestSuite) TearDownSuite() {}

func (suite *DefaultPoliciesTestSuite) SetupTest() {
	suite.deployments = make(map[string]*storage.Deployment)
	suite.images = make(map[string]*storage.Image)
	suite.deploymentsToImages = make(map[string][]*storage.Image)
	suite.deploymentsToIndicators = make(map[string][]*storage.ProcessIndicator)
}

func (suite *DefaultPoliciesTestSuite) imageIDFromDep(deployment *storage.Deployment) string {
	suite.Require().Len(deployment.GetContainers(), 1, "This function only supports deployments with exactly one container")
	id := deployment.GetContainers()[0].GetImage().GetId()
	suite.NotEmpty(id, "Deployment '%s' had no image id", protocompat.MarshalTextString(deployment))
	return id
}

func (suite *DefaultPoliciesTestSuite) TestNVDCVSSCriteria() {
	heartbleedDep := storage.Deployment_builder{
		Id: "HEARTBLEEDDEPID",
		Containers: []*storage.Container{
			storage.Container_builder{
				Name:            "nginx",
				SecurityContext: storage.SecurityContext_builder{Privileged: true}.Build(),
				Image:           storage.ContainerImage_builder{Id: "HEARTBLEEDDEPSHA"}.Build(),
			}.Build(),
		},
	}.Build()

	ts := time.Now().AddDate(0, 0, -5)
	protoTs, err := protocompat.ConvertTimeToTimestampOrError(ts)
	require.NoError(suite.T(), err)

	suite.addDepAndImages(heartbleedDep, storage.Image_builder{
		Id:   "HEARTBLEEDDEPSHA",
		Name: storage.ImageName_builder{FullName: "heartbleed"}.Build(),
		Scan: storage.ImageScan_builder{
			Components: []*storage.EmbeddedImageScanComponent{
				storage.EmbeddedImageScanComponent_builder{Name: "heartbleed", Version: "1.2", Vulns: []*storage.EmbeddedVulnerability{
					storage.EmbeddedVulnerability_builder{Cve: "CVE-2014-0160", Link: "https://heartbleed", Cvss: 6, NvdCvss: 8, FixedBy: proto.String("v1.2"),
						FirstImageOccurrence: protoTs}.Build(),
				}}.Build(),
			},
		}.Build(),
	}.Build())

	pv := &storage.PolicyValue{}
	pv.SetValue("> 6")
	nvdCvssPolicyGroup := &storage.PolicyGroup{}
	nvdCvssPolicyGroup.SetFieldName(fieldnames.NvdCvss)
	nvdCvssPolicyGroup.SetValues([]*storage.PolicyValue{
		pv,
	})

	policy := policyWithGroups(storage.EventSource_NOT_APPLICABLE, nvdCvssPolicyGroup)

	deployment := suite.deployments["HEARTBLEEDDEPID"]
	depMatcher, err := BuildDeploymentMatcher(policy)
	require.NoError(suite.T(), err)
	violations, err := depMatcher.MatchDeployment(nil, enhancedDeployment(deployment, suite.getImagesForDeployment(deployment)))
	require.Len(suite.T(), violations.AlertViolations, 1)
	require.NoError(suite.T(), err)
	require.Contains(suite.T(), violations.AlertViolations[0].GetMessage(), "NVD CVSS")

}

func (suite *DefaultPoliciesTestSuite) TestFixableAndImageFirstOccurenceCriteria() {
	heartbleedDep := storage.Deployment_builder{
		Id: "HEARTBLEEDDEPID",
		Containers: []*storage.Container{
			storage.Container_builder{
				Name:            "nginx",
				SecurityContext: storage.SecurityContext_builder{Privileged: true}.Build(),
				Image:           storage.ContainerImage_builder{Id: "HEARTBLEEDDEPSHA"}.Build(),
			}.Build(),
		},
	}.Build()

	ts := time.Now().AddDate(0, 0, -5)
	protoTs, err := protocompat.ConvertTimeToTimestampOrError(ts)
	require.NoError(suite.T(), err)

	suite.addDepAndImages(heartbleedDep, storage.Image_builder{
		Id:   "HEARTBLEEDDEPSHA",
		Name: storage.ImageName_builder{FullName: "heartbleed"}.Build(),
		Scan: storage.ImageScan_builder{
			Components: []*storage.EmbeddedImageScanComponent{
				storage.EmbeddedImageScanComponent_builder{Name: "heartbleed", Version: "1.2", Vulns: []*storage.EmbeddedVulnerability{
					storage.EmbeddedVulnerability_builder{Cve: "CVE-2014-0160", Link: "https://heartbleed", Cvss: 6, FixedBy: proto.String("v1.2"),
						FirstImageOccurrence: protoTs}.Build(),
				}}.Build(),
			},
		}.Build(),
	}.Build())

	pv := &storage.PolicyValue{}
	pv.SetValue("true")
	fixablePolicyGroup := &storage.PolicyGroup{}
	fixablePolicyGroup.SetFieldName(fieldnames.Fixable)
	fixablePolicyGroup.SetValues([]*storage.PolicyValue{pv})
	pv2 := &storage.PolicyValue{}
	pv2.SetValue("2")
	firstImageOccurrenceGroup := &storage.PolicyGroup{}
	firstImageOccurrenceGroup.SetFieldName(fieldnames.DaysSinceImageFirstDiscovered)
	firstImageOccurrenceGroup.SetValues([]*storage.PolicyValue{pv2})

	policy := policyWithGroups(storage.EventSource_NOT_APPLICABLE, fixablePolicyGroup, firstImageOccurrenceGroup)

	deployment := suite.deployments["HEARTBLEEDDEPID"]
	depMatcher, err := BuildDeploymentMatcher(policy)
	require.NoError(suite.T(), err)
	violations, err := depMatcher.MatchDeployment(nil, enhancedDeployment(deployment, suite.getImagesForDeployment(deployment)))
	require.Len(suite.T(), violations.AlertViolations, 1)
	require.NoError(suite.T(), err)

}

func (suite *DefaultPoliciesTestSuite) TestDaysSinceCVEPublishedCriteria() {
	heartbleedDep := storage.Deployment_builder{
		Id: "HEARTBLEEDDEPID",
		Containers: []*storage.Container{
			storage.Container_builder{
				Name:            "nginx",
				SecurityContext: storage.SecurityContext_builder{Privileged: true}.Build(),
				Image:           storage.ContainerImage_builder{Id: "HEARTBLEEDDEPSHA"}.Build(),
			}.Build(),
		},
	}.Build()

	ts := time.Now().AddDate(0, 0, -5)
	protoTs, err := protocompat.ConvertTimeToTimestampOrError(ts)
	require.NoError(suite.T(), err)

	suite.addDepAndImages(heartbleedDep, storage.Image_builder{
		Id:   "HEARTBLEEDDEPSHA",
		Name: storage.ImageName_builder{FullName: "heartbleed"}.Build(),
		Scan: storage.ImageScan_builder{
			Components: []*storage.EmbeddedImageScanComponent{
				storage.EmbeddedImageScanComponent_builder{Name: "heartbleed", Version: "1.2", Vulns: []*storage.EmbeddedVulnerability{
					storage.EmbeddedVulnerability_builder{Cve: "CVE-2014-0160", Link: "https://heartbleed", Cvss: 6, FixedBy: proto.String("v1.2"),
						PublishedOn: protoTs}.Build(),
				}}.Build(),
			},
		}.Build(),
	}.Build())

	pv := &storage.PolicyValue{}
	pv.SetValue("true")
	fixablePolicyGroup := &storage.PolicyGroup{}
	fixablePolicyGroup.SetFieldName(fieldnames.Fixable)
	fixablePolicyGroup.SetValues([]*storage.PolicyValue{pv})
	pv2 := &storage.PolicyValue{}
	pv2.SetValue("2")
	cvePublishedGroup := &storage.PolicyGroup{}
	cvePublishedGroup.SetFieldName(fieldnames.DaysSincePublished)
	cvePublishedGroup.SetValues([]*storage.PolicyValue{pv2})

	policy := policyWithGroups(storage.EventSource_NOT_APPLICABLE, fixablePolicyGroup, cvePublishedGroup)

	deployment := suite.deployments["HEARTBLEEDDEPID"]
	depMatcher, err := BuildDeploymentMatcher(policy)
	require.NoError(suite.T(), err)
	violations, err := depMatcher.MatchDeployment(nil, enhancedDeployment(deployment, suite.getImagesForDeployment(deployment)))
	require.Len(suite.T(), violations.AlertViolations, 1)
	require.NoError(suite.T(), err)

}

func (suite *DefaultPoliciesTestSuite) TestNoDuplicatePolicyIDs() {
	ids := set.NewStringSet()
	for _, p := range suite.defaultPolicies {
		suite.True(ids.Add(p.GetId()))
	}
}

func (suite *DefaultPoliciesTestSuite) MustGetPolicy(name string) *storage.Policy {
	p := suite.defaultPolicies[name]
	if p != nil {
		return p
	}
	p = suite.customPolicies[name]
	if p != nil {
		return p
	}
	suite.FailNow("Policy not found: ", name)
	return nil
}

func (suite *DefaultPoliciesTestSuite) addDepAndImages(deployment *storage.Deployment, images ...*storage.Image) {
	suite.deployments[deployment.GetId()] = deployment
	for _, i := range images {
		suite.images[i.GetId()] = i
		suite.deploymentsToImages[deployment.GetId()] = append(suite.deploymentsToImages[deployment.GetId()], i)
	}
}

func (suite *DefaultPoliciesTestSuite) addImage(img *storage.Image) *storage.Image {
	suite.images[img.GetId()] = img
	return img
}

func imageWithComponents(components []*storage.EmbeddedImageScanComponent) *storage.Image {
	imageName := &storage.ImageName{}
	imageName.SetFullName("docker.io/ASFASF")
	imageName.SetRemote("ASFASF")
	imageScan := &storage.ImageScan{}
	imageScan.SetComponents(components)
	image := &storage.Image{}
	image.SetId(uuid.NewV4().String())
	image.SetName(imageName)
	image.SetScan(imageScan)
	return image
}

func imageWithLayers(layers []*storage.ImageLayer) *storage.Image {
	imageName := &storage.ImageName{}
	imageName.SetFullName("docker.io/ASFASF")
	imageName.SetRemote("ASFASF")
	v1m := &storage.V1Metadata{}
	v1m.SetLayers(layers)
	im := &storage.ImageMetadata{}
	im.SetV1(v1m)
	image := &storage.Image{}
	image.SetId(uuid.NewV4().String())
	image.SetName(imageName)
	image.SetMetadata(im)
	return image
}

func imageWithOS(os string) *storage.Image {
	imageName := &storage.ImageName{}
	imageName.SetFullName("docker.io/ASFASF")
	imageName.SetRemote("ASFASF")
	imageScan := &storage.ImageScan{}
	imageScan.SetOperatingSystem(os)
	image := &storage.Image{}
	image.SetId(uuid.NewV4().String())
	image.SetName(imageName)
	image.SetScan(imageScan)
	return image
}

func (suite *DefaultPoliciesTestSuite) imageWithSignatureVerificationResults(name string, results []*storage.ImageSignatureVerificationResult) *storage.Image {
	// Use util to populate registry, remote and tag
	imageName, _, err := imgUtils.GenerateImageNameFromString(name)
	if err != nil {
		suite.T().Fatalf("failed to parse image name %q: %v", name, err)
	}

	// Restore fullName to the passed string, to maintain original behavior
	imageName.SetFullName(name)

	img := &storage.Image{}
	img.SetId(uuid.NewV4().String())
	img.SetName(imageName)

	if results != nil {
		isvd := &storage.ImageSignatureVerificationData{}
		isvd.SetResults(results)
		img.SetSignatureVerificationData(isvd)
	}
	return img
}

func deploymentWithImageAnyID(img *storage.Image) *storage.Deployment {
	return deploymentWithImage(uuid.NewV4().String(), img)
}

func deploymentWithImage(id string, img *storage.Image) *storage.Deployment {
	remoteSplit := strings.Split(img.GetName().GetFullName(), "/")
	alphaOnly := regexp.MustCompile("[^A-Za-z]+")
	containerName := alphaOnly.ReplaceAllString(remoteSplit[len(remoteSplit)-1], "")
	container := &storage.Container{}
	container.SetId(img.GetId())
	container.SetName(containerName)
	container.SetImage(types.ToContainerImage(img))
	deployment := &storage.Deployment{}
	deployment.SetId(id)
	deployment.SetContainers([]*storage.Container{container})
	return deployment
}

func (suite *DefaultPoliciesTestSuite) addIndicator(deploymentID, name, args, path string, lineage []string, uid uint32) *storage.ProcessIndicator {
	deployment := suite.deployments[deploymentID]
	if len(deployment.GetContainers()) == 0 {
		container := &storage.Container{}
		container.SetName(uuid.NewV4().String())
		deployment.SetContainers([]*storage.Container{container})
	}
	lineageInfo := make([]*storage.ProcessSignal_LineageInfo, len(lineage))
	for i, ancestor := range lineage {
		pl := &storage.ProcessSignal_LineageInfo{}
		pl.SetParentExecFilePath(ancestor)
		lineageInfo[i] = pl
	}
	ps := &storage.ProcessSignal{}
	ps.SetName(name)
	ps.SetArgs(args)
	ps.SetExecFilePath(path)
	ps.SetTime(protocompat.TimestampNow())
	ps.SetLineageInfo(lineageInfo)
	ps.SetUid(uid)
	indicator := &storage.ProcessIndicator{}
	indicator.SetId(uuid.NewV4().String())
	indicator.SetDeploymentId(deploymentID)
	indicator.SetContainerName(deployment.GetContainers()[0].GetName())
	indicator.SetSignal(ps)
	suite.deploymentsToIndicators[deploymentID] = append(suite.deploymentsToIndicators[deploymentID], indicator)
	return indicator
}

type testCase struct {
	policyName                string
	expectedViolations        map[string][]*storage.Alert_Violation
	expectedProcessViolations map[string][]*storage.ProcessIndicator

	// If shouldNotMatch is specified (which is the case for policies that check for the absence of something), we verify that
	// it matches everything except shouldNotMatch.
	// If sampleViolationForMatched is provided, we verify that all the matches are the string provided in sampleViolationForMatched.
	shouldNotMatch             map[string]struct{}
	sampleViolationForMatched  string
	allowUnvalidatedViolations bool
}

func (suite *DefaultPoliciesTestSuite) getImagesForDeployment(deployment *storage.Deployment) []*storage.Image {
	images := suite.deploymentsToImages[deployment.GetId()]
	if len(images) == 0 {
		return make([]*storage.Image, len(deployment.GetContainers()))
	}
	suite.Equal(len(deployment.GetContainers()), len(images))
	return images
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

func (suite *DefaultPoliciesTestSuite) TestDefaultPolicies() {
	fixtureDep := fixtures.GetDeployment()
	fixturesImages := fixtures.DeploymentImages()

	suite.addDepAndImages(fixtureDep, fixturesImages...)

	imageName := &storage.ImageName{}
	imageName.SetRegistry("docker.io")
	imageName.SetRemote("library/nginx")
	imageName.SetTag("1.10")
	imageName.SetFullName("docker.io/library/nginx:1.10")
	nginx110 := &storage.Image{}
	nginx110.SetId("SHANGINX110")
	nginx110.SetName(imageName)

	nginx110Dep := deploymentWithImage("nginx110", nginx110)
	suite.addDepAndImages(nginx110Dep, nginx110)

	oldScannedTime := time.Now().Add(-31 * 24 * time.Hour)

	imageName2 := &storage.ImageName{}
	imageName2.SetFullName("docker.io/stackrox/old-scanned-image:0.1")
	imageScan := &storage.ImageScan{}
	imageScan.SetScanTime(protoconv.ConvertTimeToTimestamp(oldScannedTime))
	oldScannedImage := &storage.Image{}
	oldScannedImage.SetId("SHAOLDSCANNED")
	oldScannedImage.SetName(imageName2)
	oldScannedImage.SetScan(imageScan)
	oldScannedDep := deploymentWithImage("oldscanned", oldScannedImage)
	suite.addDepAndImages(oldScannedDep, oldScannedImage)

	il := &storage.ImageLayer{}
	il.SetInstruction("ADD")
	il.SetValue("deploy.sh")
	il2 := &storage.ImageLayer{}
	il2.SetInstruction("RUN")
	il2.SetValue("deploy.sh")
	addDockerFileImg := imageWithLayers([]*storage.ImageLayer{
		il,
		il2,
	})
	addDockerFileDep := deploymentWithImageAnyID(addDockerFileImg)
	suite.addDepAndImages(addDockerFileDep, addDockerFileImg)

	il3 := &storage.ImageLayer{}
	il3.SetInstruction("EXPOSE")
	il3.SetValue("22/tcp")
	imagePort22Image := imageWithLayers([]*storage.ImageLayer{
		il3,
	})
	imagePort22Dep := deploymentWithImageAnyID(imagePort22Image)
	suite.addDepAndImages(imagePort22Dep, imagePort22Image)

	il4 := &storage.ImageLayer{}
	il4.SetInstruction("CMD")
	il4.SetValue("do an insecure thing")
	insecureCMDImage := imageWithLayers([]*storage.ImageLayer{
		il4,
	})

	insecureCMDDep := deploymentWithImageAnyID(insecureCMDImage)
	suite.addDepAndImages(insecureCMDDep, insecureCMDImage)

	il5 := &storage.ImageLayer{}
	il5.SetInstruction("VOLUME")
	il5.SetValue("/run/secrets")
	runSecretsImage := imageWithLayers([]*storage.ImageLayer{
		il5,
	})
	il6 := &storage.ImageLayer{}
	il6.SetInstruction("VOLUME")
	il6.SetValue("[/run/secrets]")
	runSecretsArrayImage := imageWithLayers([]*storage.ImageLayer{
		il6,
	})
	il7 := &storage.ImageLayer{}
	il7.SetInstruction("VOLUME")
	il7.SetValue("/var/something /run/secrets")
	runSecretsListImage := imageWithLayers([]*storage.ImageLayer{
		il7,
	})
	il8 := &storage.ImageLayer{}
	il8.SetInstruction("VOLUME")
	il8.SetValue("[/var/something /run/secrets]")
	runSecretsArrayListImage := imageWithLayers([]*storage.ImageLayer{
		il8,
	})
	runSecretsDep := deploymentWithImageAnyID(runSecretsImage)
	runSecretsArrayDep := deploymentWithImageAnyID(runSecretsArrayImage)
	runSecretsListDep := deploymentWithImageAnyID(runSecretsListImage)
	runSecretsArrayListDep := deploymentWithImageAnyID(runSecretsArrayListImage)
	suite.addDepAndImages(runSecretsDep, runSecretsImage)
	suite.addDepAndImages(runSecretsArrayDep, runSecretsArrayImage)
	suite.addDepAndImages(runSecretsListDep, runSecretsListImage)
	suite.addDepAndImages(runSecretsArrayListDep, runSecretsArrayListImage)

	oldImageCreationTime := time.Now().Add(-100 * 24 * time.Hour)
	imageName3 := &storage.ImageName{}
	imageName3.SetFullName("docker.io/stackrox/old-image:0.1")
	v1m := &storage.V1Metadata{}
	v1m.SetCreated(protoconv.ConvertTimeToTimestamp(oldImageCreationTime))
	im := &storage.ImageMetadata{}
	im.SetV1(v1m)
	oldCreatedImage := &storage.Image{}
	oldCreatedImage.SetId("SHA:OLDCREATEDIMAGE")
	oldCreatedImage.SetName(imageName3)
	oldCreatedImage.SetMetadata(im)
	oldImageDep := deploymentWithImage("oldimagedep", oldCreatedImage)
	suite.addDepAndImages(oldImageDep, oldCreatedImage)

	eisc := &storage.EmbeddedImageScanComponent{}
	eisc.SetName("apk-tools")
	eisc.SetVersion("1.2")
	eisc2 := &storage.EmbeddedImageScanComponent{}
	eisc2.SetName("asfa")
	eisc2.SetVersion("1.5")
	apkImage := imageWithComponents([]*storage.EmbeddedImageScanComponent{
		eisc,
		eisc2,
	})
	apkDep := deploymentWithImageAnyID(apkImage)
	suite.addDepAndImages(apkDep, apkImage)

	eisc3 := &storage.EmbeddedImageScanComponent{}
	eisc3.SetName("curl")
	eisc3.SetVersion("1.3")
	eisc4 := &storage.EmbeddedImageScanComponent{}
	eisc4.SetName("curlwithextra")
	eisc4.SetVersion("0.9")
	curlImage := imageWithComponents([]*storage.EmbeddedImageScanComponent{
		eisc3,
		eisc4,
	})
	curlDep := deploymentWithImageAnyID(curlImage)
	suite.addDepAndImages(curlDep, curlImage)

	componentDeps := make(map[string]*storage.Deployment)
	for _, component := range []string{"apt", "dnf", "wget"} {
		eisc5 := &storage.EmbeddedImageScanComponent{}
		eisc5.SetName(component)
		img := imageWithComponents([]*storage.EmbeddedImageScanComponent{
			eisc5,
		})
		dep := deploymentWithImageAnyID(img)
		suite.addDepAndImages(dep, img)
		componentDeps[component] = dep
	}

	heartbleedDep := storage.Deployment_builder{
		Id: "HEARTBLEEDDEPID",
		Containers: []*storage.Container{
			storage.Container_builder{
				Name:            "nginx",
				SecurityContext: storage.SecurityContext_builder{Privileged: true}.Build(),
				Image:           storage.ContainerImage_builder{Id: "HEARTBLEEDDEPSHA"}.Build(),
			}.Build(),
		},
	}.Build()
	suite.addDepAndImages(heartbleedDep, storage.Image_builder{
		Id:   "HEARTBLEEDDEPSHA",
		Name: storage.ImageName_builder{FullName: "heartbleed"}.Build(),
		Scan: storage.ImageScan_builder{
			Components: []*storage.EmbeddedImageScanComponent{
				storage.EmbeddedImageScanComponent_builder{Name: "heartbleed", Version: "1.2", Vulns: []*storage.EmbeddedVulnerability{
					storage.EmbeddedVulnerability_builder{Cve: "CVE-2014-0160", Link: "https://heartbleed", Cvss: 6, FixedBy: proto.String("v1.2")}.Build(),
				}}.Build(),
			},
		}.Build(),
	}.Build())

	requiredImageLabel := storage.Deployment_builder{
		Id: "requiredImageLabel",
		Containers: []*storage.Container{
			storage.Container_builder{
				Name:  "REQUIREDIMAGELABEL",
				Image: storage.ContainerImage_builder{Id: "requiredImageLabelImage"}.Build(),
			}.Build(),
		},
	}.Build()
	suite.addDepAndImages(requiredImageLabel, storage.Image_builder{
		Id: "requiredImageLabelImage",
		Name: storage.ImageName_builder{
			FullName: "docker.io/stackrox/required-image:0.1",
		}.Build(),
		Metadata: storage.ImageMetadata_builder{
			V1: storage.V1Metadata_builder{
				Labels: map[string]string{
					"required-label": "required-value",
				},
			}.Build(),
		}.Build(),
	}.Build())

	shellshockImage := imageWithComponents([]*storage.EmbeddedImageScanComponent{
		storage.EmbeddedImageScanComponent_builder{Name: "shellshock", Version: "1.2", Vulns: []*storage.EmbeddedVulnerability{
			storage.EmbeddedVulnerability_builder{Cve: "CVE-2014-6271", Link: "https://shellshock", Cvss: 6}.Build(),
			storage.EmbeddedVulnerability_builder{Cve: "CVE-ARBITRARY", Link: "https://notshellshock"}.Build(),
		}}.Build(),
	})
	shellshockDep := deploymentWithImageAnyID(shellshockImage)
	suite.addDepAndImages(shellshockDep, shellshockImage)

	suppressedShellshockImage := imageWithComponents([]*storage.EmbeddedImageScanComponent{
		storage.EmbeddedImageScanComponent_builder{Name: "shellshock", Version: "1.2", Vulns: []*storage.EmbeddedVulnerability{
			storage.EmbeddedVulnerability_builder{Cve: "CVE-2014-6271", Link: "https://shellshock", Cvss: 6, Suppressed: true}.Build(),
			storage.EmbeddedVulnerability_builder{Cve: "CVE-ARBITRARY", Link: "https://notshellshock"}.Build(),
		}}.Build(),
	})
	suppressedShellShockDep := deploymentWithImageAnyID(suppressedShellshockImage)
	suite.addDepAndImages(suppressedShellShockDep, suppressedShellshockImage)

	strutsImage := imageWithComponents([]*storage.EmbeddedImageScanComponent{
		storage.EmbeddedImageScanComponent_builder{Name: "struts", Version: "1.2", Vulns: []*storage.EmbeddedVulnerability{
			storage.EmbeddedVulnerability_builder{Cve: "CVE-2017-5638", Link: "https://struts", Cvss: 8, Severity: storage.VulnerabilitySeverity_IMPORTANT_VULNERABILITY_SEVERITY, FixedBy: proto.String("v1.3")}.Build(),
		}}.Build(),
		storage.EmbeddedImageScanComponent_builder{Name: "OTHER", Version: "1.3", Vulns: []*storage.EmbeddedVulnerability{
			storage.EmbeddedVulnerability_builder{Cve: "CVE-1223-451", Link: "https://cvefake"}.Build(),
		}}.Build(),
	})
	strutsDep := deploymentWithImageAnyID(strutsImage)
	suite.addDepAndImages(strutsDep, strutsImage)

	strutsImageSuppressed := imageWithComponents([]*storage.EmbeddedImageScanComponent{
		storage.EmbeddedImageScanComponent_builder{Name: "struts", Version: "1.2", Vulns: []*storage.EmbeddedVulnerability{
			storage.EmbeddedVulnerability_builder{Cve: "CVE-2017-5638", Link: "https://struts", Suppressed: true, Cvss: 8, Severity: storage.VulnerabilitySeverity_IMPORTANT_VULNERABILITY_SEVERITY, FixedBy: proto.String("v1.3")}.Build(),
		}}.Build(),
		storage.EmbeddedImageScanComponent_builder{Name: "OTHER", Version: "1.3", Vulns: []*storage.EmbeddedVulnerability{
			storage.EmbeddedVulnerability_builder{Cve: "CVE-1223-451", Link: "https://cvefake"}.Build(),
		}}.Build(),
	})
	strutsDepSuppressed := deploymentWithImageAnyID(strutsImageSuppressed)
	suite.addDepAndImages(strutsDepSuppressed, strutsImageSuppressed)

	// When image is pull out, the deferred field is set based upon the legacy suppressed field. Therefore, both are set.
	// However, here we are specifically testing whether detection is taking the new vulnerability state field into
	// account by not setting the suppressed field.
	structImageWithDeferredVulns := imageWithComponents([]*storage.EmbeddedImageScanComponent{
		storage.EmbeddedImageScanComponent_builder{Name: "deferred-struts", Version: "1.2", Vulns: []*storage.EmbeddedVulnerability{
			storage.EmbeddedVulnerability_builder{Cve: "CVE-2017-5638", Link: "https://struts", State: storage.VulnerabilityState_DEFERRED, Cvss: 8, Severity: storage.VulnerabilitySeverity_IMPORTANT_VULNERABILITY_SEVERITY, FixedBy: proto.String("v1.3")}.Build(),
			storage.EmbeddedVulnerability_builder{Cve: "CVE-2017-FP", Link: "https://struts", State: storage.VulnerabilityState_FALSE_POSITIVE, Cvss: 8, Severity: storage.VulnerabilitySeverity_IMPORTANT_VULNERABILITY_SEVERITY, FixedBy: proto.String("v1.3")}.Build(),
			storage.EmbeddedVulnerability_builder{Cve: "CVE-2017-FAKE", Link: "https://struts", Cvss: 8, Severity: storage.VulnerabilitySeverity_IMPORTANT_VULNERABILITY_SEVERITY, FixedBy: proto.String("v1.3")}.Build(),
		}}.Build(),
	})
	structDepWithDeferredVulns := deploymentWithImageAnyID(structImageWithDeferredVulns)
	suite.addDepAndImages(structDepWithDeferredVulns, structImageWithDeferredVulns)

	depWithNonSeriousVulnsImage := imageWithComponents([]*storage.EmbeddedImageScanComponent{
		storage.EmbeddedImageScanComponent_builder{Name: "NOSERIOUS", Version: "2.3", Vulns: []*storage.EmbeddedVulnerability{
			storage.EmbeddedVulnerability_builder{Cve: "CVE-1234-5678", Link: "https://abcdefgh"}.Build(),
			storage.EmbeddedVulnerability_builder{Cve: "CVE-5678-1234", Link: "https://lmnopqrst"}.Build(),
		}}.Build(),
	})
	depWithNonSeriousVulns := deploymentWithImageAnyID(depWithNonSeriousVulnsImage)
	suite.addDepAndImages(depWithNonSeriousVulns, depWithNonSeriousVulnsImage)

	dockerSockDep := storage.Deployment_builder{
		Id: "DOCKERSOCDEP",
		Containers: []*storage.Container{
			storage.Container_builder{
				Name: "dockersock",
				Volumes: []*storage.Volume{
					storage.Volume_builder{Source: "/var/run/docker.sock", Name: "DOCKERSOCK", Type: "HostPath", ReadOnly: true}.Build(),
					storage.Volume_builder{Source: "NOTDOCKERSOCK"}.Build(),
				}}.Build(),
		},
	}.Build()
	suite.addDepAndImages(dockerSockDep)

	crioSockDep := storage.Deployment_builder{
		Id: "CRIOSOCDEP",
		Containers: []*storage.Container{
			storage.Container_builder{
				Name: "criosock",
				Volumes: []*storage.Volume{
					storage.Volume_builder{Source: "/run/crio/crio.sock", Name: "CRIOSOCK", Type: "HostPath", ReadOnly: true}.Build(),
					storage.Volume_builder{Source: "NOTCRIORSOCK"}.Build(),
				}}.Build(),
		},
	}.Build()
	suite.addDepAndImages(crioSockDep)

	pc := &storage.PortConfig{}
	pc.SetProtocol("TCP")
	pc.SetContainerPort(22)
	pc2 := &storage.PortConfig{}
	pc2.SetProtocol("UDP")
	pc2.SetContainerPort(4125)
	containerPort22Dep := &storage.Deployment{}
	containerPort22Dep.SetId("CONTAINERPORT22DEP")
	containerPort22Dep.SetPorts([]*storage.PortConfig{
		pc,
		pc2,
	})
	suite.addDepAndImages(containerPort22Dep)

	secretEnvDep := storage.Deployment_builder{
		Id: "SECRETENVDEP",
		Containers: []*storage.Container{
			storage.Container_builder{
				Name: "secretenv",
				Config: storage.ContainerConfig_builder{
					Env: []*storage.ContainerConfig_EnvironmentConfig{
						storage.ContainerConfig_EnvironmentConfig_builder{Key: "THIS_IS_SECRET_VAR", Value: "stealthmode", EnvVarSource: storage.ContainerConfig_EnvironmentConfig_RAW}.Build(),
						storage.ContainerConfig_EnvironmentConfig_builder{Key: "HOME", Value: "/home/stackrox"}.Build(),
					},
				}.Build()}.Build(),
		},
	}.Build()
	suite.addDepAndImages(secretEnvDep)

	secretEnvSrcUnsetDep := storage.Deployment_builder{
		Id: "SECRETENVSRCUNSETDEP",
		Containers: []*storage.Container{
			storage.Container_builder{
				Name: "secretenvsrcunset",
				Config: storage.ContainerConfig_builder{
					Env: []*storage.ContainerConfig_EnvironmentConfig{
						storage.ContainerConfig_EnvironmentConfig_builder{Key: "THIS_IS_SECRET_VAR", Value: "stealthmode"}.Build(),
					},
				}.Build()}.Build(),
		},
	}.Build()
	suite.addDepAndImages(secretEnvSrcUnsetDep)

	secretKeyRefDep := storage.Deployment_builder{
		Id: "SECRETKEYREFDEP",
		Containers: []*storage.Container{
			storage.Container_builder{Config: storage.ContainerConfig_builder{
				Env: []*storage.ContainerConfig_EnvironmentConfig{
					storage.ContainerConfig_EnvironmentConfig_builder{Key: "THIS_IS_SECRET_VAR", EnvVarSource: storage.ContainerConfig_EnvironmentConfig_SECRET_KEY}.Build(),
					storage.ContainerConfig_EnvironmentConfig_builder{Key: "HOME", Value: "/home/stackrox"}.Build(),
				},
			}.Build()}.Build(),
		},
	}.Build()
	suite.addDepAndImages(secretKeyRefDep)

	// Fake deployment that shouldn't match anything, just to make sure
	// that none of our queries will accidentally match it.
	deployment := &storage.Deployment{}
	deployment.SetId("FAKEID")
	deployment.SetName("FAKENAME")
	suite.addDepAndImages(deployment)

	depWithGoodEmailAnnotation := &storage.Deployment{}
	depWithGoodEmailAnnotation.SetId("GOODEMAILDEPID")
	depWithGoodEmailAnnotation.SetAnnotations(map[string]string{
		"email": "vv@stackrox.com",
	})
	suite.addDepAndImages(depWithGoodEmailAnnotation)

	depWithOwnerAnnotation := &storage.Deployment{}
	depWithOwnerAnnotation.SetId("OWNERANNOTATIONDEP")
	depWithOwnerAnnotation.SetAnnotations(map[string]string{
		"owner": "IOWNTHIS",
		"blah":  "Blah",
	})
	suite.addDepAndImages(depWithOwnerAnnotation)

	depWithOwnerLabel := &storage.Deployment{}
	depWithOwnerLabel.SetId("OWNERLABELDEP")
	depWithOwnerLabel.SetLabels(map[string]string{
		"owner": "IOWNTHIS",
		"blah":  "Blah",
	})
	suite.addDepAndImages(depWithOwnerLabel)

	depWitharbitraryAnnotations := &storage.Deployment{}
	depWitharbitraryAnnotations.SetId("ARBITRARYANNOTATIONDEPID")
	depWitharbitraryAnnotations.SetAnnotations(map[string]string{
		"emailnot": "vv@stackrox.com",
		"notemail": "vv@stackrox.com",
		"ownernot": "vv",
		"nowner":   "vv",
	})
	suite.addDepAndImages(depWitharbitraryAnnotations)

	depWithBadEmailAnnotation := &storage.Deployment{}
	depWithBadEmailAnnotation.SetId("BADEMAILDEPID")
	depWithBadEmailAnnotation.SetAnnotations(map[string]string{
		"email": "NOTANEMAIL",
	})
	suite.addDepAndImages(depWithBadEmailAnnotation)

	sysAdminDep := storage.Deployment_builder{
		Id: "SYSADMINDEPID",
		Containers: []*storage.Container{
			storage.Container_builder{
				Name: "cap-sys",
				SecurityContext: storage.SecurityContext_builder{
					AddCapabilities: []string{"SYS_ADMIN"},
				}.Build(),
			}.Build(),
		},
	}.Build()
	suite.addDepAndImages(sysAdminDep)

	depWithAllResourceLimitsRequestsSpecified := storage.Deployment_builder{
		Id: "ALLRESOURCESANDLIMITSDEP",
		Containers: []*storage.Container{
			storage.Container_builder{Resources: storage.Resources_builder{
				CpuCoresRequest: 0.1,
				CpuCoresLimit:   0.3,
				MemoryMbLimit:   100,
				MemoryMbRequest: 1251,
			}.Build()}.Build(),
		},
	}.Build()
	suite.addDepAndImages(depWithAllResourceLimitsRequestsSpecified)

	depWithEnforcementBypassAnnotation := &storage.Deployment{}
	depWithEnforcementBypassAnnotation.SetId("ENFORCEMENTBYPASS")
	depWithEnforcementBypassAnnotation.SetAnnotations(map[string]string{
		"admission.stackrox.io/break-glass": "ticket-1234",
		"some-other":                        "annotation",
	})
	suite.addDepAndImages(depWithEnforcementBypassAnnotation)

	hostMountDep := storage.Deployment_builder{
		Id: "HOSTMOUNT",
		Containers: []*storage.Container{
			storage.Container_builder{
				Name: "hostmount",
				Volumes: []*storage.Volume{
					storage.Volume_builder{Source: "/etc/passwd", Name: "HOSTMOUNT", Type: "HostPath"}.Build(),
					storage.Volume_builder{Source: "/var/lib/kubelet", Name: "KUBELET", Type: "HostPath", ReadOnly: true}.Build(),
				}}.Build(),
		},
	}.Build()
	suite.addDepAndImages(hostMountDep)

	hostPIDDep := &storage.Deployment{}
	hostPIDDep.SetId("HOSTPID")
	hostPIDDep.SetHostPid(true)
	suite.addDepAndImages(hostPIDDep)

	hostIPCDep := &storage.Deployment{}
	hostIPCDep.SetId("HOSTIPC")
	hostIPCDep.SetHostIpc(true)
	suite.addDepAndImages(hostIPCDep)

	imgWithFixedByEmpty := suite.addImage(imageWithComponents([]*storage.EmbeddedImageScanComponent{
		storage.EmbeddedImageScanComponent_builder{Name: "EXplicitlyEmptyFixedBy", Version: "2.3", Vulns: []*storage.EmbeddedVulnerability{
			storage.EmbeddedVulnerability_builder{Cve: "CVE-1234-5678", Cvss: 8, Link: "https://abcdefgh", FixedBy: proto.String("")}.Build(),
		}}.Build(),
	}))

	imgWithFixedByEmptyOnlyForSome := suite.addImage(imageWithComponents([]*storage.EmbeddedImageScanComponent{
		storage.EmbeddedImageScanComponent_builder{Name: "EXplicitlyEmptyFixedBy", Version: "2.3", Vulns: []*storage.EmbeddedVulnerability{
			storage.EmbeddedVulnerability_builder{Cve: "CVE-1234-5678", Cvss: 8, Severity: storage.VulnerabilitySeverity_CRITICAL_VULNERABILITY_SEVERITY, Link: "https://abcdefgh", FixedBy: proto.String("")}.Build(),
		}}.Build(),
		storage.EmbeddedImageScanComponent_builder{Name: "Normal", Version: "2.3", Vulns: []*storage.EmbeddedVulnerability{
			storage.EmbeddedVulnerability_builder{Cve: "CVE-5612-1245", Cvss: 8, Severity: storage.VulnerabilitySeverity_CRITICAL_VULNERABILITY_SEVERITY, Link: "https://abcdefgh", FixedBy: proto.String("actually_fixable")}.Build(),
		}}.Build(),
	}))

	imageName4 := &storage.ImageName{}
	imageName4.SetFullName("docker.io/stackrox/rootuser:0.1")
	v1m2 := &storage.V1Metadata{}
	v1m2.SetUser("root")
	im2 := &storage.ImageMetadata{}
	im2.SetV1(v1m2)
	rootUserImage := &storage.Image{}
	rootUserImage.SetId("SHA:ROOTUSERIMAGE")
	rootUserImage.SetName(imageName4)
	rootUserImage.SetMetadata(im2)
	depWithRootUser := deploymentWithImageAnyID(rootUserImage)
	suite.addDepAndImages(depWithRootUser, rootUserImage)

	il9 := &storage.ImageLayer{}
	il9.SetInstruction("RUN")
	il9.SetValue("apt-get update")
	updateInstructionImage := imageWithLayers([]*storage.ImageLayer{
		il9,
	})
	depWithUpdate := deploymentWithImageAnyID(updateInstructionImage)
	suite.addDepAndImages(depWithUpdate, updateInstructionImage)

	restrictedHostPortDep := storage.Deployment_builder{
		Id: "RESTRICTEDHOSTPORT",
		Ports: []*storage.PortConfig{
			storage.PortConfig_builder{
				ExposureInfos: []*storage.PortConfig_ExposureInfo{
					storage.PortConfig_ExposureInfo_builder{
						NodePort: 22,
					}.Build(),
				},
			}.Build(),
		},
	}.Build()

	suite.addDepAndImages(restrictedHostPortDep)

	mountPropagationDep := storage.Deployment_builder{
		Id: "MOUNTPROPAGATIONDEP",
		Containers: []*storage.Container{
			storage.Container_builder{
				Id: "MOUNTPROPAGATIONCONTAINER",
				Volumes: []*storage.Volume{
					storage.Volume_builder{
						Name:             "ThisMountIsOnFire",
						MountPropagation: storage.Volume_BIDIRECTIONAL,
					}.Build(),
				},
			}.Build(),
		},
	}.Build()
	suite.addDepAndImages(mountPropagationDep)

	noSeccompProfileDep := storage.Deployment_builder{
		Id: "NOSECCOMPPROFILEDEP",
		Containers: []*storage.Container{
			storage.Container_builder{
				SecurityContext: storage.SecurityContext_builder{
					SeccompProfile: storage.SecurityContext_SeccompProfile_builder{
						Type: storage.SecurityContext_SeccompProfile_UNCONFINED,
					}.Build(),
				}.Build(),
			}.Build(),
		},
	}.Build()
	suite.addDepAndImages(noSeccompProfileDep)

	hostNetworkDep := &storage.Deployment{}
	hostNetworkDep.SetId("HOSTNETWORK")
	hostNetworkDep.SetHostNetwork(true)
	suite.addDepAndImages(hostNetworkDep)

	noAppArmorProfileDep := storage.Deployment_builder{
		Id: "NOAPPARMORPROFILEDEP",
		Containers: []*storage.Container{
			storage.Container_builder{
				Name: "No AppArmor Profile",
				Config: storage.ContainerConfig_builder{
					AppArmorProfile: "unconfined",
				}.Build(),
			}.Build(),
		},
	}.Build()
	suite.addDepAndImages(noAppArmorProfileDep)

	// Images "made by Red Hat" - coming from Red Hat registries or Red Hat remotes in quay.io
	isvr := &storage.ImageSignatureVerificationResult{}
	isvr.SetVerifierId(signatures.DefaultRedHatSignatureIntegration.GetId())
	isvr.SetStatus(storage.ImageSignatureVerificationResult_FAILED_VERIFICATION)
	registryAccessRedhatComUnverifiedImg := suite.imageWithSignatureVerificationResults("registry.access.redhat.com/redhat/ubi8:latest",
		[]*storage.ImageSignatureVerificationResult{
			isvr,
		},
	)
	isvr2 := &storage.ImageSignatureVerificationResult{}
	isvr2.SetVerifierId(signatures.DefaultRedHatSignatureIntegration.GetId())
	isvr2.SetStatus(storage.ImageSignatureVerificationResult_FAILED_VERIFICATION)
	registryRedHatIoUnverifiedImg := suite.imageWithSignatureVerificationResults("registry.redhat.io/redhat/ubi8:latest",
		[]*storage.ImageSignatureVerificationResult{
			isvr2,
		},
	)

	isvr3 := &storage.ImageSignatureVerificationResult{}
	isvr3.SetVerifierId(signatures.DefaultRedHatSignatureIntegration.GetId())
	isvr3.SetStatus(storage.ImageSignatureVerificationResult_FAILED_VERIFICATION)
	quayOCPReleaseUnverifiedImg := suite.imageWithSignatureVerificationResults("quay.io/openshift-release-dev/ocp-release:latest",
		[]*storage.ImageSignatureVerificationResult{
			isvr3,
		},
	)
	isvr4 := &storage.ImageSignatureVerificationResult{}
	isvr4.SetVerifierId(signatures.DefaultRedHatSignatureIntegration.GetId())
	isvr4.SetStatus(storage.ImageSignatureVerificationResult_FAILED_VERIFICATION)
	quayOCPArtDevUnverifiedImg := suite.imageWithSignatureVerificationResults("quay.io/openshift-release-dev/ocp-v4.0-art-dev:latest",
		[]*storage.ImageSignatureVerificationResult{
			isvr4,
		},
	)

	suite.addImage(registryAccessRedhatComUnverifiedImg)
	suite.addImage(registryRedHatIoUnverifiedImg)
	suite.addImage(quayOCPReleaseUnverifiedImg)
	suite.addImage(quayOCPArtDevUnverifiedImg)

	// Index processes
	bashLineage := []string{"/bin/bash"}
	fixtureDepAptIndicator := suite.addIndicator(fixtureDep.GetId(), "apt", "", "/usr/bin/apt", bashLineage, 1)
	sysAdminDepAptIndicator := suite.addIndicator(sysAdminDep.GetId(), "apt", "install blah", "/usr/bin/apt", bashLineage, 1)

	kubeletIndicator := suite.addIndicator(containerPort22Dep.GetId(), "curl", "-v -k -SL https://12.13.14.15:10250", "/bin/curl", bashLineage, 1)
	kubeletIndicator2 := suite.addIndicator(containerPort22Dep.GetId(), "wget", "https://heapster.kube-system/metrics", "/bin/wget", bashLineage, 1)
	kubeletIndicator3 := suite.addIndicator(containerPort22Dep.GetId(), "curl", "https://12.13.14.15:10250 -v -k", "/bin/curl", bashLineage, 1)

	crontabIndicator := suite.addIndicator(containerPort22Dep.GetId(), "crontab", "1 2 3 4 5 6", "/bin/crontab", bashLineage, 1)

	nmapIndicatorfixtureDep1 := suite.addIndicator(fixtureDep.GetId(), "nmap", "blah", "/usr/bin/nmap", bashLineage, 1)
	nmapIndicatorfixtureDep2 := suite.addIndicator(fixtureDep.GetId(), "nmap", "blah2", "/usr/bin/nmap", bashLineage, 1)
	nmapIndicatorNginx110Dep := suite.addIndicator(nginx110Dep.GetId(), "nmap", "", "/usr/bin/nmap", bashLineage, 1)

	ifconfigIndicatorfixtureDep1 := suite.addIndicator(fixtureDep.GetId(), "ifconfig", "blah", "/sbin/ifconfig", bashLineage, 1)
	ifconfigIndicatorfixtureDep2 := suite.addIndicator(fixtureDep.GetId(), "ifconfig", "blah2", "/usr/bin/ifconfig", bashLineage, 1)
	ipIndicatorfixtureDep := suite.addIndicator(fixtureDep.GetId(), "ip", "", "/sbin/ip", bashLineage, 1)
	arpIndicatorfixtureDep := suite.addIndicator(fixtureDep.GetId(), "arp", "", "/usr/sbin/arp", bashLineage, 1)
	ifconfigIndicatorNginx110Dep := suite.addIndicator(nginx110Dep.GetId(), "ifconfig", "", "/sbin/ifconfig", bashLineage, 1)
	ipIndicatorNginx110Dep := suite.addIndicator(nginx110Dep.GetId(), "ip", "", "/sbin/ip", bashLineage, 1)
	arpIndicatorNginx110Dep := suite.addIndicator(nginx110Dep.GetId(), "arp", "", "/usr/sbin/arp", bashLineage, 1)
	// These two should not match for the Network Management Execution policy. See ROX-6011
	suite.addIndicator(fixtureDep.GetId(), "pip", "", "/usr/bin/pip", bashLineage, 1)
	suite.addIndicator(nginx110Dep.GetId(), "pip", "", "/usr/bin/pip", bashLineage, 1)

	javaLineage := []string{"/bin/bash", "/mnt/scripts/run_server.sh", "/bin/java"}
	fixtureDepJavaIndicator := suite.addIndicator(fixtureDep.GetId(), "/bin/bash", "-attack", "/bin/bash", javaLineage, 0)

	deploymentTestCases := []testCase{
		{
			policyName: "Latest tag",
			expectedViolations: map[string][]*storage.Alert_Violation{
				fixtureDep.GetId(): {
					{
						Message: "Container 'supervulnerable' has image with tag 'latest'",
					},
				},
			},
		},
		{
			policyName: "Alpine Linux Package Manager (apk) in Image",
			expectedViolations: map[string][]*storage.Alert_Violation{
				apkDep.GetId(): {
					{
						Message: "Container 'ASFASF' includes component 'apk-tools' (version 1.2)",
					},
				},
			},
		},
		{
			policyName: "Ubuntu Package Manager in Image",
			expectedViolations: map[string][]*storage.Alert_Violation{
				componentDeps["apt"].GetId(): {
					{
						Message: "Container 'ASFASF' includes component 'apt'",
					},
				},
			},
		},
		{
			policyName: "Curl in Image",
			expectedViolations: map[string][]*storage.Alert_Violation{
				curlDep.GetId(): {
					{
						Message: "Container 'ASFASF' includes component 'curl' (version 1.3)",
					},
				},
			},
		},
		{
			policyName: "Red Hat Package Manager in Image",
			expectedViolations: map[string][]*storage.Alert_Violation{
				componentDeps["dnf"].GetId(): {
					{
						Message: "Container 'ASFASF' includes component 'dnf'",
					},
				},
			},
		},
		{
			policyName: "Wget in Image",
			expectedViolations: map[string][]*storage.Alert_Violation{
				componentDeps["wget"].GetId(): {
					{
						Message: "Container 'ASFASF' includes component 'wget'",
					},
				},
			},
		},
		{
			policyName: "Mount Container Runtime Socket",
			expectedViolations: map[string][]*storage.Alert_Violation{
				dockerSockDep.GetId(): {
					{
						Message: "Read-only volume 'DOCKERSOCK' has source '/var/run/docker.sock' and type 'HostPath'",
					},
				},
				crioSockDep.GetId(): {
					{
						Message: "Read-only volume 'CRIOSOCK' has source '/run/crio/crio.sock' and type 'HostPath'",
					},
				},
			},
		},
		{
			policyName: "90-Day Image Age",
			expectedViolations: map[string][]*storage.Alert_Violation{
				oldImageDep.GetId(): {
					{
						Message: fmt.Sprintf("Container 'oldimage' has image created at %s (UTC)", readable.Time(oldImageCreationTime)),
					},
				},
			},
		},
		{
			policyName: "30-Day Scan Age",
			expectedViolations: map[string][]*storage.Alert_Violation{
				oldScannedDep.GetId(): {
					{
						Message: fmt.Sprintf("Container 'oldscannedimage' has image last scanned at %s (UTC)", readable.Time(oldScannedTime)),
					},
				},
			},
		},
		{
			policyName: "Secure Shell (ssh) Port Exposed in Image",
			expectedViolations: map[string][]*storage.Alert_Violation{
				imagePort22Dep.GetId(): {
					{
						Message: "Dockerfile line 'EXPOSE 22/tcp' found in container 'ASFASF'",
					},
				},
			},
		},
		{
			policyName: "Secure Shell (ssh) Port Exposed",
			expectedViolations: map[string][]*storage.Alert_Violation{
				containerPort22Dep.GetId(): {
					{
						Message: "Exposed port 22/TCP is present",
					},
				},
			},
		},
		{
			policyName: "Privileged Container",
			expectedViolations: map[string][]*storage.Alert_Violation{
				fixtureDep.GetId(): {
					{
						Message: "Container 'nginx110container' is privileged",
					},
				},
				heartbleedDep.GetId(): {
					{
						Message: "Container 'nginx' is privileged",
					},
				},
			},
		},
		{
			policyName: "Container using read-write root filesystem",
			expectedViolations: map[string][]*storage.Alert_Violation{
				heartbleedDep.GetId(): {
					{
						Message: "Container 'nginx' uses a read-write root filesystem",
					},
				},
				fixtureDep.GetId(): {
					{
						Message: "Container 'nginx110container' uses a read-write root filesystem",
					},
				},
				sysAdminDep.GetId(): {
					{
						Message: "Container 'cap-sys' uses a read-write root filesystem",
					},
				},
				noSeccompProfileDep.GetId(): {
					{
						Message: "Container  uses a read-write root filesystem",
					},
				},
			},
		},
		{
			policyName: "Insecure specified in CMD",
			expectedViolations: map[string][]*storage.Alert_Violation{
				insecureCMDDep.GetId(): {
					{
						Message: "Dockerfile line 'CMD do an insecure thing' found in container 'ASFASF'",
					},
				},
			},
		},
		{
			policyName: "Improper Usage of Orchestrator Secrets Volume",
			expectedViolations: map[string][]*storage.Alert_Violation{
				runSecretsDep.GetId(): {
					{
						Message: "Dockerfile line 'VOLUME /run/secrets' found in container 'ASFASF'",
					},
				},
				runSecretsArrayDep.GetId(): {
					{
						Message: "Dockerfile line 'VOLUME [/run/secrets]' found in container 'ASFASF'",
					},
				},
				runSecretsListDep.GetId(): {
					{
						Message: "Dockerfile line 'VOLUME /var/something /run/secrets' found in container 'ASFASF'",
					},
				},
				runSecretsArrayListDep.GetId(): {
					{
						Message: "Dockerfile line 'VOLUME [/var/something /run/secrets]' found in container 'ASFASF'",
					},
				},
			},
		},
		{
			policyName: "Images with no scans",
			shouldNotMatch: map[string]struct{}{
				// These deployments have scans on their images.
				fixtureDep.GetId():                 {},
				oldScannedDep.GetId():              {},
				heartbleedDep.GetId():              {},
				apkDep.GetId():                     {},
				curlDep.GetId():                    {},
				componentDeps["apt"].GetId():       {},
				componentDeps["dnf"].GetId():       {},
				componentDeps["wget"].GetId():      {},
				shellshockDep.GetId():              {},
				suppressedShellShockDep.GetId():    {},
				strutsDep.GetId():                  {},
				strutsDepSuppressed.GetId():        {},
				structDepWithDeferredVulns.GetId(): {},
				depWithNonSeriousVulns.GetId():     {},
				// The rest of the deployments have no images!
				"FAKEID":                                          {},
				containerPort22Dep.GetId():                        {},
				dockerSockDep.GetId():                             {},
				crioSockDep.GetId():                               {},
				secretEnvDep.GetId():                              {},
				secretEnvSrcUnsetDep.GetId():                      {},
				secretKeyRefDep.GetId():                           {},
				depWithOwnerAnnotation.GetId():                    {},
				depWithOwnerLabel.GetId():                         {},
				depWithGoodEmailAnnotation.GetId():                {},
				depWithBadEmailAnnotation.GetId():                 {},
				depWitharbitraryAnnotations.GetId():               {},
				sysAdminDep.GetId():                               {},
				depWithAllResourceLimitsRequestsSpecified.GetId(): {},
				depWithEnforcementBypassAnnotation.GetId():        {},
				hostMountDep.GetId():                              {},
				restrictedHostPortDep.GetId():                     {},
				hostPIDDep.GetId():                                {},
				hostIPCDep.GetId():                                {},
				mountPropagationDep.GetId():                       {},
				noSeccompProfileDep.GetId():                       {},
				hostNetworkDep.GetId():                            {},
				noAppArmorProfileDep.GetId():                      {},
			},
			sampleViolationForMatched: "Image in container '%s' has not been scanned",
		},
		{
			policyName: "Required Annotation: Email",
			shouldNotMatch: map[string]struct{}{
				depWithGoodEmailAnnotation.GetId(): {},
			},
			sampleViolationForMatched: "Required annotation not found (key = 'email', value = '[a-zA-Z0-9_.+-]+@[a-zA-Z0-9-]+\\.[a-zA-Z0-9-.]+')",
		},
		{
			policyName: "Required Label: Owner/Team",
			shouldNotMatch: map[string]struct{}{
				depWithOwnerLabel.GetId(): {},
				fixtureDep.GetId():        {},
			},
			sampleViolationForMatched: "Required label not found (key = 'owner|team', value = '.+')",
		},
		{
			policyName: "Required Annotation: Owner/Team",
			shouldNotMatch: map[string]struct{}{
				depWithOwnerAnnotation.GetId(): {},
				fixtureDep.GetId():             {},
			},
			sampleViolationForMatched: "Required annotation not found (key = 'owner|team', value = '.+')",
		},
		{
			policyName: "CAP_SYS_ADMIN capability added",
			expectedViolations: map[string][]*storage.Alert_Violation{
				sysAdminDep.GetId(): {
					{
						Message: "Container 'cap-sys' adds capability SYS_ADMIN",
					},
				},
				fixtureDep.GetId(): {
					{
						Message: "Container 'nginx110container' adds capability SYS_ADMIN",
					},
				},
			},
		},
		{
			policyName: "Apache Struts: CVE-2017-5638",
			expectedViolations: map[string][]*storage.Alert_Violation{
				strutsDep.GetId(): {
					{
						Message: "CVE-2017-5638 (CVSS 8) (severity Important) found in component 'struts' (version 1.2) in container 'ASFASF'",
					},
				},
				// CVE-2017-5638 is deferred in `deferred-struct`, hence no violation.
			},
		},
		{
			policyName: "No CPU request or memory limit specified",
			expectedViolations: map[string][]*storage.Alert_Violation{
				fixtureDep.GetId(): {
					{Message: "Memory limit set to 0 MB for container 'nginx110container'"},
				},
			},
		},
		{
			policyName: "Environment Variable Contains Secret",
			expectedViolations: map[string][]*storage.Alert_Violation{
				secretEnvDep.GetId(): {
					{
						Message: "Environment variable 'THIS_IS_SECRET_VAR' is present in container 'secretenv'",
					},
				},
			},
		},
		{
			policyName: "Secret Mounted as Environment Variable",
			expectedViolations: map[string][]*storage.Alert_Violation{
				secretKeyRefDep.GetId(): {
					{
						Message: "Environment variable 'THIS_IS_SECRET_VAR' is present and references a Secret",
					},
				},
			},
		},
		{
			policyName: "Fixable CVSS >= 6 and Privileged",
			expectedViolations: map[string][]*storage.Alert_Violation{
				heartbleedDep.GetId(): {
					{
						Message: "Container 'nginx' is privileged",
					},
					{
						Message: "Fixable CVE-2014-0160 (CVSS 6) (severity Unknown) found in component 'heartbleed' (version 1.2) in container 'nginx', resolved by version v1.2",
					},
				},
			},
		},
		{
			policyName: "Fixable CVSS >= 7",
			expectedViolations: map[string][]*storage.Alert_Violation{
				strutsDep.GetId(): {
					{
						Message: "Fixable CVE-2017-5638 (CVSS 8) (severity Important) found in component 'struts' (version 1.2) in container 'ASFASF', resolved by version v1.3",
					},
				},
				structDepWithDeferredVulns.GetId(): {
					{
						Message: "Fixable CVE-2017-FAKE (CVSS 8) (severity Important) found in component 'deferred-struts' (version 1.2) in container 'ASFASF', resolved by version v1.3",
					},
				},
			},
		},
		{
			policyName: "Fixable Severity at least Important",
			expectedViolations: map[string][]*storage.Alert_Violation{
				strutsDep.GetId(): {
					{
						Message: "Fixable CVE-2017-5638 (CVSS 8) (severity Important) found in component 'struts' (version 1.2) in container 'ASFASF', resolved by version v1.3",
					},
				},
				structDepWithDeferredVulns.GetId(): {
					{
						Message: "Fixable CVE-2017-FAKE (CVSS 8) (severity Important) found in component 'deferred-struts' (version 1.2) in container 'ASFASF', resolved by version v1.3",
					},
				},
			},
		},
		{
			policyName: "ADD Command used instead of COPY",
			expectedViolations: map[string][]*storage.Alert_Violation{
				addDockerFileDep.GetId(): {
					{
						Message: "Dockerfile line 'ADD deploy.sh' found in container 'ASFASF'",
					},
				},
				fixtureDep.GetId(): {
					{
						Message: "Dockerfile line 'ADD FILE:blah' found in container 'nginx110container'",
					},
					{
						Message: "Dockerfile line 'ADD file:4eedf861fb567fffb2694b65ebd...' found in container 'supervulnerable'",
					},
				},
			},
		},
		{
			policyName: "nmap Execution",
			expectedProcessViolations: map[string][]*storage.ProcessIndicator{
				fixtureDep.GetId():  {nmapIndicatorfixtureDep1, nmapIndicatorfixtureDep2},
				nginx110Dep.GetId(): {nmapIndicatorNginx110Dep},
			},
		},
		{
			policyName: "Process Targeting Cluster Kubelet Endpoint",
			expectedProcessViolations: map[string][]*storage.ProcessIndicator{
				containerPort22Dep.GetId(): {kubeletIndicator, kubeletIndicator2, kubeletIndicator3},
			},
		},
		{
			policyName: "crontab Execution",
			expectedProcessViolations: map[string][]*storage.ProcessIndicator{
				containerPort22Dep.GetId(): {crontabIndicator},
			},
		},
		{
			policyName: "Ubuntu Package Manager Execution",
			expectedProcessViolations: map[string][]*storage.ProcessIndicator{
				fixtureDep.GetId():  {fixtureDepAptIndicator},
				sysAdminDep.GetId(): {sysAdminDepAptIndicator},
			},
		},
		{
			policyName: "Process with UID 0",
			expectedProcessViolations: map[string][]*storage.ProcessIndicator{
				fixtureDep.GetId(): {fixtureDepJavaIndicator},
			},
		},
		{
			policyName: "Shell Spawned by Java Application",
			expectedProcessViolations: map[string][]*storage.ProcessIndicator{
				fixtureDep.GetId(): {fixtureDepJavaIndicator},
			},
		},
		{
			policyName: "Network Management Execution",
			expectedProcessViolations: map[string][]*storage.ProcessIndicator{
				fixtureDep.GetId():  {ifconfigIndicatorfixtureDep1, ifconfigIndicatorfixtureDep2, ipIndicatorfixtureDep, arpIndicatorfixtureDep},
				nginx110Dep.GetId(): {ifconfigIndicatorNginx110Dep, ipIndicatorNginx110Dep, arpIndicatorNginx110Dep},
			},
		},
		{
			policyName: "Emergency Deployment Annotation",
			expectedViolations: map[string][]*storage.Alert_Violation{
				depWithEnforcementBypassAnnotation.GetId(): {
					{Message: "Disallowed annotations found: admission.stackrox.io/break-glass=ticket-1234"},
				},
			},
		},
		{
			policyName: "Mounting Sensitive Host Directories",
			expectedViolations: map[string][]*storage.Alert_Violation{
				hostMountDep.GetId(): {
					{Message: "Read-only volume 'KUBELET' has source '/var/lib/kubelet' and type 'HostPath'"},
					{Message: "Writable volume 'HOSTMOUNT' has source '/etc/passwd' and type 'HostPath'"},
				},
				dockerSockDep.GetId(): {
					{Message: "Read-only volume 'DOCKERSOCK' has source '/var/run/docker.sock' and type 'HostPath'"},
				},
			},
		},
		{
			policyName: writableHostMountPolicyName,
			expectedViolations: map[string][]*storage.Alert_Violation{
				hostMountDep.GetId(): {
					{Message: "Writable volume 'HOSTMOUNT' has source '/etc/passwd' and type 'HostPath'"},
				},
			},
		},
		{
			policyName: "Docker CIS 4.1: Ensure That a User for the Container Has Been Created",
			expectedViolations: map[string][]*storage.Alert_Violation{
				depWithRootUser.GetId(): {
					{
						Message: "Container 'rootuser' has image with user 'root'",
					},
				},
			},
		},
		{
			policyName: "Docker CIS 4.7: Alert on Update Instruction",
			expectedViolations: map[string][]*storage.Alert_Violation{
				depWithUpdate.GetId(): {
					{
						Message: "Dockerfile line 'RUN apt-get update' found in container 'ASFASF'",
					},
				},
			},
		},
		{
			policyName: "Docker CIS 5.7: Ensure privileged ports are not mapped within containers",
			expectedViolations: map[string][]*storage.Alert_Violation{
				restrictedHostPortDep.GetId(): {
					{
						Message: "Exposed node port 22 is present",
					},
				},
			},
		},
		{
			policyName: "Docker CIS 5.15: Ensure that the host's process namespace is not shared",
			expectedViolations: map[string][]*storage.Alert_Violation{
				hostPIDDep.GetId(): {
					{Message: "Deployment uses the host's process ID namespace"},
				},
			},
		},
		{
			policyName: "Docker CIS 5.16: Ensure that the host's IPC namespace is not shared",
			expectedViolations: map[string][]*storage.Alert_Violation{
				hostIPCDep.GetId(): {
					{Message: "Deployment uses the host's IPC namespace"},
				},
			},
		},
		{
			policyName: "Docker CIS 5.19: Ensure mount propagation mode is not enabled",
			expectedViolations: map[string][]*storage.Alert_Violation{
				mountPropagationDep.GetId(): {
					{Message: "Writable volume 'ThisMountIsOnFire' has mount propagation 'bidirectional'"},
				},
			},
		},
		{
			policyName: "Docker CIS 5.21: Ensure the default seccomp profile is not disabled",
			expectedViolations: map[string][]*storage.Alert_Violation{
				noSeccompProfileDep.GetId(): {
					{Message: "Container has Seccomp profile type 'unconfined'"},
				},
			},
		},
		{
			policyName: "Docker CIS 5.9 and 5.20: Ensure that the host's network namespace is not shared",
			expectedViolations: map[string][]*storage.Alert_Violation{
				hostNetworkDep.GetId(): {
					{Message: "Deployment uses the host's network namespace"},
				},
			},
		},
		{
			policyName: "Docker CIS 5.1 Ensure that, if applicable, an AppArmor Profile is enabled",
			expectedViolations: map[string][]*storage.Alert_Violation{
				noAppArmorProfileDep.GetId(): {
					{Message: "Container 'No AppArmor Profile' has AppArmor profile type 'unconfined'"},
				},
			},
		},
		{
			policyName:                 "Docker CIS 4.4: Ensure images are scanned and rebuilt to include security patches",
			allowUnvalidatedViolations: true,
			expectedViolations: map[string][]*storage.Alert_Violation{
				strutsDep.GetId(): {
					{
						Message: "Fixable CVE-2017-5638 (CVSS 8) (severity Important) found in component 'struts' (version 1.2) in container 'ASFASF', resolved by version v1.3",
					},
				},
				heartbleedDep.GetId(): {
					{
						Message: "Fixable CVE-2014-0160 (CVSS 6) (severity Unknown) found in component 'heartbleed' (version 1.2) in container 'nginx', resolved by version v1.2",
					},
				},
				fixtureDep.GetId(): {
					{
						Message: "Fixable CVE-2014-6200 (CVSS 5) (severity Moderate) found in component 'name' (version 1.2.3.4) in container 'supervulnerable', resolved by version abcdefg",
					},
				},
				fixtures.LightweightDeployment().GetId(): {
					{
						Message: "Fixable CVE-2014-6200 (CVSS 5) (severity Moderate) found in component 'name' (version 1.2.3.4) in container 'supervulnerable', resolved by version abcdefg",
					},
				},
				structDepWithDeferredVulns.GetId(): {
					{
						Message: "Fixable CVE-2017-FAKE (CVSS 8) (severity Important) found in component 'deferred-struts' (version 1.2) in container 'ASFASF', resolved by version v1.3",
					},
				},
			},
		},
		{
			policyName: anyHostPathPolicyName,
			expectedViolations: map[string][]*storage.Alert_Violation{
				dockerSockDep.GetId(): {
					{Message: "Read-only volume 'DOCKERSOCK' has source '/var/run/docker.sock' and type 'HostPath'"},
				},
				crioSockDep.GetId(): {
					{Message: "Read-only volume 'CRIOSOCK' has source '/run/crio/crio.sock' and type 'HostPath'"},
				},
				hostMountDep.GetId(): {
					{Message: "Read-only volume 'KUBELET' has source '/var/lib/kubelet' and type 'HostPath'"},
					{Message: "Writable volume 'HOSTMOUNT' has source '/etc/passwd' and type 'HostPath'"},
				},
			},
		},
	}

	for _, c := range deploymentTestCases {
		p := suite.MustGetPolicy(c.policyName)
		suite.T().Run(fmt.Sprintf("%s (on deployments)", c.policyName), func(t *testing.T) {
			if len(c.shouldNotMatch) == 0 {
				assert.True(t, (c.expectedViolations != nil) != (c.expectedProcessViolations != nil), "Every test case must "+
					"contain exactly one of expectedViolations and expectedProcessViolations")
			} else {
				assert.Nil(t, c.expectedViolations, "Cannot specify shouldNotMatch AND expectedViolations")
				assert.Nil(t, c.expectedProcessViolations, "Cannot specify shouldNotMatch AND expectedProcessViolations")
			}

			m, err := BuildDeploymentMatcher(p)
			require.NoError(t, err)

			if c.expectedProcessViolations != nil {
				processMatcher, err := BuildDeploymentWithProcessMatcher(p)
				require.NoError(t, err)
				for deploymentID, processes := range c.expectedProcessViolations {
					expectedProcesses := set.NewStringSet(sliceutils.Map(processes, func(p *storage.ProcessIndicator) string {
						return p.GetId()
					})...)
					deployment := suite.deployments[deploymentID]

					for _, process := range suite.deploymentsToIndicators[deploymentID] {
						match := getViolationsWithAndWithoutCaching(t, func(cache *CacheReceptacle) (Violations, error) {
							return processMatcher.MatchDeploymentWithProcess(nil, enhancedDeployment(deployment, suite.getImagesForDeployment(deployment)), process, false)
						})
						require.NoError(t, err)
						if expectedProcesses.Contains(process.GetId()) {
							assert.NotNil(t, match.ProcessViolation, "process %+v should match", process)
						} else {
							assert.Nil(t, match.ProcessViolation, "process %+v should not match", process)
						}
					}
				}
				return
			}

			actualViolations := make(map[string][]*storage.Alert_Violation)
			for id, deployment := range suite.deployments {
				violationsForDep := getViolationsWithAndWithoutCaching(t, func(cache *CacheReceptacle) (Violations, error) {
					return m.MatchDeployment(cache, enhancedDeployment(deployment, suite.getImagesForDeployment(deployment)))
				})
				assert.Nil(t, violationsForDep.ProcessViolation)
				if alertViolations := violationsForDep.AlertViolations; len(alertViolations) > 0 {
					actualViolations[id] = alertViolations
				}
			}
			if len(c.shouldNotMatch) > 0 {
				for shouldNotMatchID := range c.shouldNotMatch {
					assert.Contains(t, suite.deployments, shouldNotMatchID)
					assert.NotContains(t, actualViolations, shouldNotMatchID)
				}
				for id := range suite.deployments {
					if _, shouldNotMatch := c.shouldNotMatch[id]; !shouldNotMatch {
						assert.Contains(t, actualViolations, id)

						// TODO(rc) update for BPL and check all sampleViolationForMatched
						if c.policyName == "Images with no scans" {
							if len(suite.deployments[id].GetContainers()) == 1 {
								msg := fmt.Sprintf(c.sampleViolationForMatched, suite.deployments[id].GetContainers()[0].GetName())
								av := &storage.Alert_Violation{}
								av.SetMessage(msg)
								protoassert.SlicesEqual(t, actualViolations[id], []*storage.Alert_Violation{av})
							}
						}
					}
				}
				return
			}

			for id := range suite.deployments {
				violations, expected := c.expectedViolations[id]
				if expected {
					assert.Contains(t, actualViolations, id)

					if c.allowUnvalidatedViolations {
						assert.NotEmpty(t, violations)
						for _, violation := range violations {
							protoassert.SliceContains(t, actualViolations[id], violation)
						}
					} else {
						protoassert.SlicesEqual(t, violations, actualViolations[id])
					}
				} else {
					assert.NotContains(t, actualViolations, id)
				}
			}

		})
	}

	imageTestCases := []testCase{
		{
			policyName: "Latest tag",
			expectedViolations: map[string][]*storage.Alert_Violation{
				fixtureDep.GetContainers()[1].GetImage().GetId(): {
					{Message: "Image has tag 'latest'"},
				},
			},
		},
		{
			policyName: "Alpine Linux Package Manager (apk) in Image",
			expectedViolations: map[string][]*storage.Alert_Violation{
				suite.imageIDFromDep(apkDep): {
					{
						Message: "Image includes component 'apk-tools' (version 1.2)",
					},
				},
			},
		},
		{
			policyName: "Ubuntu Package Manager in Image",
			expectedViolations: map[string][]*storage.Alert_Violation{
				suite.imageIDFromDep(componentDeps["apt"]): {
					{
						Message: "Image includes component 'apt'",
					},
				},
			},
		},
		{
			policyName: "Curl in Image",
			expectedViolations: map[string][]*storage.Alert_Violation{
				suite.imageIDFromDep(curlDep): {
					{
						Message: "Image includes component 'curl' (version 1.3)",
					},
				},
			},
		},
		{
			policyName: "Red Hat Package Manager in Image",
			expectedViolations: map[string][]*storage.Alert_Violation{
				suite.imageIDFromDep(componentDeps["dnf"]): {
					{
						Message: "Image includes component 'dnf'",
					},
				},
			},
		},
		{
			policyName: "Wget in Image",
			expectedViolations: map[string][]*storage.Alert_Violation{
				suite.imageIDFromDep(componentDeps["wget"]): {
					{
						Message: "Image includes component 'wget'",
					},
				},
			},
		},
		{
			policyName: "90-Day Image Age",
			expectedViolations: map[string][]*storage.Alert_Violation{
				suite.imageIDFromDep(oldImageDep): {
					{
						Message: fmt.Sprintf("Image was created at %s (UTC)", readable.Time(oldImageCreationTime)),
					},
				},
			},
		},
		{
			policyName: "30-Day Scan Age",
			expectedViolations: map[string][]*storage.Alert_Violation{
				suite.imageIDFromDep(oldScannedDep): {
					{
						Message: fmt.Sprintf("Image was last scanned at %s (UTC)", readable.Time(oldScannedTime)),
					},
				},
			},
		},
		{
			policyName: "Secure Shell (ssh) Port Exposed in Image",
			expectedViolations: map[string][]*storage.Alert_Violation{
				suite.imageIDFromDep(imagePort22Dep): {
					{
						Message: "Dockerfile line 'EXPOSE 22/tcp' found",
					},
				},
			},
		},
		{
			policyName: "Insecure specified in CMD",
			expectedViolations: map[string][]*storage.Alert_Violation{
				suite.imageIDFromDep(insecureCMDDep): {
					{
						Message: "Dockerfile line 'CMD do an insecure thing' found",
					},
				},
			},
		},
		{
			policyName: "Improper Usage of Orchestrator Secrets Volume",
			expectedViolations: map[string][]*storage.Alert_Violation{
				suite.imageIDFromDep(runSecretsDep): {
					{
						Message: "Dockerfile line 'VOLUME /run/secrets' found",
					},
				},
				suite.imageIDFromDep(runSecretsArrayDep): {
					{
						Message: "Dockerfile line 'VOLUME [/run/secrets]' found",
					},
				},
				suite.imageIDFromDep(runSecretsListDep): {
					{
						Message: "Dockerfile line 'VOLUME /var/something /run/secrets' found",
					},
				},
				suite.imageIDFromDep(runSecretsArrayListDep): {
					{
						Message: "Dockerfile line 'VOLUME [/var/something /run/secrets]' found",
					},
				},
			},
		},
		{
			policyName: "Images with no scans",
			shouldNotMatch: map[string]struct{}{
				oldScannedImage.GetId():                          {},
				suite.imageIDFromDep(heartbleedDep):              {},
				apkImage.GetId():                                 {},
				curlImage.GetId():                                {},
				suite.imageIDFromDep(componentDeps["apt"]):       {},
				suite.imageIDFromDep(componentDeps["dnf"]):       {},
				suite.imageIDFromDep(componentDeps["wget"]):      {},
				shellshockImage.GetId():                          {},
				suppressedShellshockImage.GetId():                {},
				strutsImage.GetId():                              {},
				strutsImageSuppressed.GetId():                    {},
				structImageWithDeferredVulns.GetId():             {},
				depWithNonSeriousVulnsImage.GetId():              {},
				fixtureDep.GetContainers()[0].GetImage().GetId(): {},
				fixtureDep.GetContainers()[1].GetImage().GetId(): {},
				suite.imageIDFromDep(oldScannedDep):              {},
				imgWithFixedByEmpty.GetId():                      {},
				imgWithFixedByEmptyOnlyForSome.GetId():           {},
			},
			sampleViolationForMatched: "Image has not been scanned",
			expectedViolations:        map[string][]*storage.Alert_Violation{},
		},
		{
			policyName: "Apache Struts: CVE-2017-5638",
			expectedViolations: map[string][]*storage.Alert_Violation{
				suite.imageIDFromDep(strutsDep): {
					{
						Message: "CVE-2017-5638 (CVSS 8) (severity Important) found in component 'struts' (version 1.2)",
					},
				},
			},
		},
		{
			policyName: "Fixable CVSS >= 7",
			expectedViolations: map[string][]*storage.Alert_Violation{
				suite.imageIDFromDep(strutsDep): {
					{
						Message: "Fixable CVE-2017-5638 (CVSS 8) (severity Important) found in component 'struts' (version 1.2), resolved by version v1.3",
					},
				},
				imgWithFixedByEmptyOnlyForSome.GetId(): {
					{
						Message: "Fixable CVE-5612-1245 (CVSS 8) (severity Critical) found in component 'Normal' (version 2.3), resolved by version actually_fixable",
					},
				},
			},
		},
		{
			policyName: "Fixable Severity at least Important",
			expectedViolations: map[string][]*storage.Alert_Violation{
				suite.imageIDFromDep(strutsDep): {
					{
						Message: "Fixable CVE-2017-5638 (CVSS 8) (severity Important) found in component 'struts' (version 1.2), resolved by version v1.3",
					},
				},
				imgWithFixedByEmptyOnlyForSome.GetId(): {
					{
						Message: "Fixable CVE-5612-1245 (CVSS 8) (severity Critical) found in component 'Normal' (version 2.3), resolved by version actually_fixable",
					},
				},
			},
		},
		{
			policyName: "ADD Command used instead of COPY",
			expectedViolations: map[string][]*storage.Alert_Violation{
				suite.imageIDFromDep(addDockerFileDep): {
					{
						Message: "Dockerfile line 'ADD deploy.sh' found",
					},
				},
				fixtureDep.GetContainers()[0].GetImage().GetId(): {
					{
						Message: "Dockerfile line 'ADD FILE:blah' found",
					},
				},
				fixtureDep.GetContainers()[1].GetImage().GetId(): {
					{
						Message: "Dockerfile line 'ADD file:4eedf861fb567fffb2694b65ebd...' found",
					},
				},
			},
		},
		{
			policyName: "Required Image Label",
			shouldNotMatch: map[string]struct{}{
				"requiredImageLabelImage": {},
			},
			sampleViolationForMatched: "Required label not found (found labels: <empty>)",
		},

		{
			// We can only test that the policy triggers for unverified images. The "shouldNotMatch" field cannot be
			// used to verify that signed images don't trigger violations, because then the logic expects that all
			// other images (not listed in shouldNotMatch) trigger a violation; and in this case only unsigned images
			// in Red Hat registries trigger violations - any other unsigned images are fine and should not trigger.
			policyName: "Red Hat images must be signed by a Red Hat release key",
			expectedViolations: map[string][]*storage.Alert_Violation{
				registryRedHatIoUnverifiedImg.GetId(): {
					{
						Message: "Image has registry 'registry.redhat.io'",
					},
					{
						Message: "Image signature is not verified by the specified signature integration(s).",
					},
				},
				registryAccessRedhatComUnverifiedImg.GetId(): {
					{
						Message: "Image has registry 'registry.access.redhat.com'",
					},
					{
						Message: "Image signature is not verified by the specified signature integration(s).",
					},
				},
				quayOCPReleaseUnverifiedImg.GetId(): {
					{
						Message: "Image has registry 'quay.io' and remote 'openshift-release-dev/ocp-release'",
					},
					{
						Message: "Image signature is not verified by the specified signature integration(s).",
					},
				},
				quayOCPArtDevUnverifiedImg.GetId(): {
					{
						Message: "Image has registry 'quay.io' and remote 'openshift-release-dev/ocp-v4.0-art-dev'",
					},
					{
						Message: "Image signature is not verified by the specified signature integration(s).",
					},
				},
			},
		},
	}

	for _, c := range imageTestCases {
		p := suite.MustGetPolicy(c.policyName)
		suite.T().Run(fmt.Sprintf("%s (on images)", c.policyName), func(t *testing.T) {
			assert.Nil(t, c.expectedProcessViolations)

			m, err := BuildImageMatcher(p)
			require.NoError(t, err)

			actualViolations := make(map[string][]*storage.Alert_Violation)
			for id, image := range suite.images {
				violationsForImg := getViolationsWithAndWithoutCaching(t, func(cache *CacheReceptacle) (Violations, error) {
					return m.MatchImage(cache, image)
				})
				suite.Nil(violationsForImg.ProcessViolation)
				if alertViolations := violationsForImg.AlertViolations; len(alertViolations) > 0 {
					actualViolations[id] = alertViolations
				}
			}

			for id, violations := range c.expectedViolations {
				assert.Contains(t, actualViolations, id)
				protoassert.SlicesEqual(t, violations, actualViolations[id])
			}
			if len(c.shouldNotMatch) > 0 {
				if c.policyName == "Required Image Label" {
					for id, image := range suite.images {
						if image.GetMetadata() == nil {
							c.shouldNotMatch[id] = struct{}{}
						}
					}
				}

				for shouldNotMatchID := range c.shouldNotMatch {
					assert.Contains(t, suite.images, shouldNotMatchID, "%s is not a known image id in the suite", shouldNotMatchID)
					assert.NotContains(t, actualViolations, shouldNotMatchID)
				}

				for id := range suite.images {
					if _, shouldNotMatch := c.shouldNotMatch[id]; !shouldNotMatch {
						assert.Contains(t, actualViolations, id)
						av := &storage.Alert_Violation{}
						av.SetMessage(c.sampleViolationForMatched)
						protoassert.SlicesEqual(t, actualViolations[id], []*storage.Alert_Violation{av})
					}
				}
			}
		})
	}
}

func (suite *DefaultPoliciesTestSuite) TestMapPolicyMatchOne() {
	noAnnotation := &storage.Deployment{}
	noAnnotation.SetId("noAnnotation")
	suite.addDepAndImages(noAnnotation)

	noValidAnnotation := &storage.Deployment{}
	noValidAnnotation.SetId("noValidAnnotation")
	noValidAnnotation.SetAnnotations(map[string]string{
		"email":               "notavalidemail",
		"someotherannotation": "vv@stackrox.com",
	})
	suite.addDepAndImages(noValidAnnotation)

	validAnnotation := &storage.Deployment{}
	validAnnotation.SetId("validAnnotation")
	validAnnotation.SetAnnotations(map[string]string{
		"email": "joseph@rules.gov",
	})
	suite.addDepAndImages(validAnnotation)

	policy := suite.defaultPolicies["Required Annotation: Email"]

	m, err := BuildDeploymentMatcher(policy)
	suite.NoError(err)

	for _, testCase := range []struct {
		dep                *storage.Deployment
		expectedViolations []string
	}{
		{
			noAnnotation,
			[]string{"Required annotation not found (found annotations: <empty>)"},
		},
		{
			noValidAnnotation,
			[]string{"Required annotation not found (found annotations: email=notavalidemail, someotherannotation=vv@stackrox.com)"},
		},
		{
			validAnnotation,
			nil,
		},
	} {
		c := testCase
		suite.Run(c.dep.GetId(), func() {
			matched, err := m.MatchDeployment(nil, enhancedDeployment(c.dep, nil))
			suite.NoError(err)
			var expectedMessages []*storage.Alert_Violation
			for _, v := range c.expectedViolations {
				av := &storage.Alert_Violation{}
				av.SetMessage(v)
				expectedMessages = append(expectedMessages, av)
			}
			protoassert.SlicesEqual(suite.T(), matched.AlertViolations, expectedMessages)
		})
	}
}

func (suite *DefaultPoliciesTestSuite) TestRuntimePolicyFieldsCompile() {
	for _, p := range suite.defaultPolicies {
		if policyUtils.AppliesAtRunTime(p) {
			checkRegexCompiles(p.GetPolicySections(), fieldnames.ProcessName)
			checkRegexCompiles(p.GetPolicySections(), fieldnames.ProcessArguments)
			checkRegexCompiles(p.GetPolicySections(), fieldnames.ProcessAncestor)
		}
	}
}

func checkRegexCompiles(sections []*storage.PolicySection, fieldname string) {
	for _, s := range sections {
		for _, g := range s.GetPolicyGroups() {
			if g.GetFieldName() == fieldname {
				if policyVals := g.GetValues(); len(policyVals) > 0 {
					for _, policyVal := range policyVals {
						if v := policyVal.GetValue(); v != "" {
							regexp.MustCompile(v)
						}
					}
				}
			}
		}
	}
}

func policyWithGroups(eventSrc storage.EventSource, groups ...*storage.PolicyGroup) *storage.Policy {
	ps := &storage.PolicySection{}
	ps.SetPolicyGroups(groups)
	policy := &storage.Policy{}
	policy.SetPolicyVersion(policyversion.CurrentVersion().String())
	policy.SetName(uuid.NewV4().String())
	policy.SetEventSource(eventSrc)
	policy.SetPolicySections([]*storage.PolicySection{ps})
	return policy
}

func policyGroupWithSingleKeyValue(fieldName, value string, negate bool) *storage.PolicyGroup {
	pv := &storage.PolicyValue{}
	pv.SetValue(value)
	pg := &storage.PolicyGroup{}
	pg.SetFieldName(fieldName)
	pg.SetValues([]*storage.PolicyValue{pv})
	pg.SetNegate(negate)
	return pg
}

func policyWithSingleKeyValue(fieldName, value string, negate bool) *storage.Policy {
	return policyWithGroups(storage.EventSource_NOT_APPLICABLE, policyGroupWithSingleKeyValue(fieldName, value, negate))
}

func policyWithSingleFieldAndValues(fieldName string, values []string, negate bool, op storage.BooleanOperator) *storage.Policy {
	pg := &storage.PolicyGroup{}
	pg.SetFieldName(fieldName)
	pg.SetValues(sliceutils.Map(values, func(val string) *storage.PolicyValue {
		pv := &storage.PolicyValue{}
		pv.SetValue(val)
		return pv
	}))
	pg.SetNegate(negate)
	pg.SetBooleanOperator(op)
	return policyWithGroups(storage.EventSource_NOT_APPLICABLE, pg)
}

func processBaselineMessage(dep *storage.Deployment, baseline bool, privileged bool, processNames ...string) []*storage.Alert_Violation {
	violations := make([]*storage.Alert_Violation, 0, len(processNames))
	containerName := dep.GetContainers()[0].GetName()
	for _, p := range processNames {
		if baseline {
			msg := fmt.Sprintf("Unexpected process '%s' in container '%s'", p, containerName)
			av := &storage.Alert_Violation{}
			av.SetMessage(msg)
			violations = append(violations, av)
		}
		if privileged {
			violations = append(violations, privilegedMessage(dep)...)
		}
	}
	return violations
}

func networkBaselineMessage(
	suite *DefaultPoliciesTestSuite,
	flow *augmentedobjs.NetworkFlowDetails,
) *storage.Alert_Violation {
	violation, err := printer.GenerateNetworkFlowViolation(flow)
	suite.Nil(err)
	return violation
}

func assertNetworkBaselineMessagesEqual(
	suite *DefaultPoliciesTestSuite,
	this []*storage.Alert_Violation,
	that []*storage.Alert_Violation,
) {
	thisWithoutTime := make([]*storage.Alert_Violation, 0, len(this))
	thatWithoutTime := make([]*storage.Alert_Violation, 0, len(that))
	for _, violation := range this {
		cp := violation.CloneVT()
		cp.ClearTime()
		thisWithoutTime = append(thisWithoutTime, cp)
	}
	for _, violation := range that {
		cp := violation.CloneVT()
		cp.ClearTime()
		thatWithoutTime = append(thatWithoutTime, cp)
	}
	protoassert.ElementsMatch(suite.T(), thisWithoutTime, thatWithoutTime)
}

func privilegedMessage(dep *storage.Deployment) []*storage.Alert_Violation {
	containerName := dep.GetContainers()[0].GetName()
	av := &storage.Alert_Violation{}
	av.SetMessage(fmt.Sprintf("Container '%s' is privileged", containerName))
	return []*storage.Alert_Violation{av}
}

func rbacPermissionMessage(level string) []*storage.Alert_Violation {
	permissionToDescMap := map[string]string{
		"NONE":                  "no specified access",
		"DEFAULT":               "default access",
		"ELEVATED_IN_NAMESPACE": "elevated access in namespace",
		"ELEVATED_CLUSTER_WIDE": "elevated access cluster wide",
		"CLUSTER_ADMIN":         "cluster admin access"}
	av := &storage.Alert_Violation{}
	av.SetMessage(fmt.Sprintf("Service account permission level with %s", permissionToDescMap[level]))
	return []*storage.Alert_Violation{av}
}

func (suite *DefaultPoliciesTestSuite) TestK8sRBACField() {
	deployments := make(map[string]*storage.Deployment)
	for permissionLevelStr, permissionLevel := range storage.PermissionLevel_value {
		dep := fixtures.GetDeployment().CloneVT()
		dep.SetServiceAccountPermissionLevel(storage.PermissionLevel(permissionLevel))
		deployments[permissionLevelStr] = dep
	}

	for _, testCase := range []struct {
		value           string
		negate          bool
		expectedMatches []string
		// Deployment ids to violations
		expectedViolations map[string][]*storage.Alert_Violation
	}{
		{
			"DEFAULT",
			false,
			[]string{"DEFAULT", "ELEVATED_IN_NAMESPACE", "ELEVATED_CLUSTER_WIDE", "CLUSTER_ADMIN"},
			map[string][]*storage.Alert_Violation{
				"DEFAULT":               rbacPermissionMessage("DEFAULT"),
				"ELEVATED_CLUSTER_WIDE": rbacPermissionMessage("ELEVATED_CLUSTER_WIDE"),
				"ELEVATED_IN_NAMESPACE": rbacPermissionMessage("ELEVATED_IN_NAMESPACE"),
				"CLUSTER_ADMIN":         rbacPermissionMessage("CLUSTER_ADMIN"),
			},
		},
		{
			"ELEVATED_CLUSTER_WIDE",
			false,
			[]string{"ELEVATED_CLUSTER_WIDE", "CLUSTER_ADMIN"},
			map[string][]*storage.Alert_Violation{
				"ELEVATED_CLUSTER_WIDE": rbacPermissionMessage("ELEVATED_CLUSTER_WIDE"),
				"CLUSTER_ADMIN":         rbacPermissionMessage("CLUSTER_ADMIN"),
			},
		},
		{
			"cluster_admin",
			false,
			[]string{"CLUSTER_ADMIN"},
			map[string][]*storage.Alert_Violation{
				"CLUSTER_ADMIN": rbacPermissionMessage("CLUSTER_ADMIN"),
			},
		},
		{
			"ELEVATED_CLUSTER_WIDE",
			true,
			[]string{"NONE", "DEFAULT", "ELEVATED_IN_NAMESPACE"},
			map[string][]*storage.Alert_Violation{
				"ELEVATED_IN_NAMESPACE": rbacPermissionMessage("ELEVATED_IN_NAMESPACE"),
				"NONE":                  rbacPermissionMessage("NONE"),
				"DEFAULT":               rbacPermissionMessage("DEFAULT"),
			},
		},
	} {
		c := testCase
		suite.T().Run(fmt.Sprintf("%+v", c.expectedMatches), func(t *testing.T) {
			matcher, err := BuildDeploymentMatcher(policyWithSingleKeyValue(fieldnames.MinimumRBACPermissions, c.value, c.negate))
			require.NoError(t, err)
			matched := set.NewStringSet()
			for depRef, dep := range deployments {
				violations, err := matcher.MatchDeployment(nil, enhancedDeployment(dep, suite.getImagesForDeployment(dep)))
				require.NoError(t, err)
				if len(violations.AlertViolations) > 0 {
					matched.Add(depRef)
					protoassert.SlicesEqual(t, violations.AlertViolations, c.expectedViolations[depRef])
				} else {
					assert.Empty(t, c.expectedViolations[depRef])
				}
			}
			assert.ElementsMatch(t, matched.AsSlice(), c.expectedMatches, "Got %v, expected: %v", matched.AsSlice(), c.expectedMatches)
		})
	}
}

func (suite *DefaultPoliciesTestSuite) TestPortExposure() {
	deployments := make(map[string]*storage.Deployment)
	for exposureLevelStr, exposureLevel := range storage.PortConfig_ExposureLevel_value {
		dep := fixtures.GetDeployment().CloneVT()
		dep.SetPorts([]*storage.PortConfig{storage.PortConfig_builder{ExposureInfos: []*storage.PortConfig_ExposureInfo{storage.PortConfig_ExposureInfo_builder{Level: storage.PortConfig_ExposureLevel(exposureLevel)}.Build()}}.Build()})
		deployments[exposureLevelStr] = dep
	}

	assertMessageMatches := func(t *testing.T, depRef string, violations []*storage.Alert_Violation) {
		depRefToExpectedMsg := map[string]string{
			"EXTERNAL": "exposed with load balancer",
			"NODE":     "exposed on node port",
			"INTERNAL": "using internal cluster IP",
			"HOST":     "exposed on host port",
			"ROUTE":    "exposed with a route",
		}
		require.Len(t, violations, 1)
		assert.Equal(t, fmt.Sprintf("Deployment port(s) %s", depRefToExpectedMsg[depRef]), violations[0].GetMessage())
	}

	for _, testCase := range []struct {
		values          []string
		negate          bool
		expectedMatches []string
	}{
		{
			[]string{"external"},
			false,
			[]string{"EXTERNAL"},
		},
		{
			[]string{"external", "NODE"},
			false,
			[]string{"EXTERNAL", "NODE"},
		},
		{
			[]string{"external", "NODE"},
			true,
			[]string{"INTERNAL", "HOST", "ROUTE"},
		},
	} {
		c := testCase
		suite.T().Run(fmt.Sprintf("%+v", c), func(t *testing.T) {
			matcher, err := BuildDeploymentMatcher(policyWithSingleFieldAndValues(fieldnames.PortExposure, c.values, c.negate, storage.BooleanOperator_OR))
			require.NoError(t, err)
			matched := set.NewStringSet()
			for depRef, dep := range deployments {
				violations, err := matcher.MatchDeployment(nil, enhancedDeployment(dep, suite.getImagesForDeployment(dep)))
				require.NoError(t, err)
				if len(violations.AlertViolations) > 0 {
					assertMessageMatches(t, depRef, violations.AlertViolations)
					matched.Add(depRef)
				}
			}
			assert.ElementsMatch(t, matched.AsSlice(), c.expectedMatches, "Got %v, expected: %v", matched.AsSlice(), c.expectedMatches)
		})
	}
}

func (suite *DefaultPoliciesTestSuite) TestImageOS() {
	depToImg := make(map[*storage.Deployment]*storage.Image)
	for _, imgName := range []string{
		"unknown",
		"alpine:v3.4",
		"alpine:v3.11",
		"ubuntu:20.04",
		"debian:8",
		"debian:10",
	} {
		img := imageWithOS(imgName)
		dep := fixtures.GetDeployment().CloneVT()
		container := &storage.Container{}
		container.SetName(imgName)
		container.SetImage(types.ToContainerImage(img))
		dep.SetContainers([]*storage.Container{
			container,
		})
		depToImg[dep] = img
	}

	for _, testCase := range []struct {
		value           string
		expectedMatches []string
	}{
		{
			value:           "unknown",
			expectedMatches: []string{"unknown"},
		},
		{
			value:           "alpine",
			expectedMatches: []string{},
		},
		{
			value:           "alpine.*",
			expectedMatches: []string{"alpine:v3.4", "alpine:v3.11"},
		},
		{
			value:           "debian:8",
			expectedMatches: []string{"debian:8"},
		},
		{
			value:           "centos",
			expectedMatches: nil,
		},
	} {
		c := testCase

		suite.T().Run(fmt.Sprintf("DeploymentMatcher %+v", c), func(t *testing.T) {
			depMatcher, err := BuildDeploymentMatcher(policyWithSingleKeyValue(fieldnames.ImageOS, c.value, false))
			require.NoError(t, err)
			depMatched := set.NewStringSet()
			for dep, img := range depToImg {
				violations, err := depMatcher.MatchDeployment(nil, enhancedDeployment(dep, []*storage.Image{img}))
				require.NoError(t, err)
				if len(violations.AlertViolations) > 0 {
					depMatched.Add(img.GetScan().GetOperatingSystem())
					require.Len(t, violations.AlertViolations, 1)
					assert.Equal(t, fmt.Sprintf("Container '%s' has image with base OS '%s'", dep.GetContainers()[0].GetName(), img.GetScan().GetOperatingSystem()), violations.AlertViolations[0].GetMessage())
				}
			}
			assert.ElementsMatch(t, depMatched.AsSlice(), c.expectedMatches, "Got %v for policy %v; expected: %v", depMatched.AsSlice(), c.value, c.expectedMatches)
		})

		suite.T().Run(fmt.Sprintf("ImageMatcher %+v", c), func(t *testing.T) {
			imgMatcher, err := BuildImageMatcher(policyWithSingleKeyValue(fieldnames.ImageOS, c.value, false))
			require.NoError(t, err)
			imgMatched := set.NewStringSet()
			for _, img := range depToImg {
				violations, err := imgMatcher.MatchImage(nil, img)
				require.NoError(t, err)
				if len(violations.AlertViolations) > 0 {
					imgMatched.Add(img.GetScan().GetOperatingSystem())
					require.Len(t, violations.AlertViolations, 1)
					assert.Equal(t, fmt.Sprintf("Image has base OS '%s'", img.GetScan().GetOperatingSystem()), violations.AlertViolations[0].GetMessage())
				}
			}
			assert.ElementsMatch(t, imgMatched.AsSlice(), c.expectedMatches, "Got %v for policy %v; expected: %v", imgMatched.AsSlice(), c.value, c.expectedMatches)
		})
	}
}

func (suite *DefaultPoliciesTestSuite) TestImageVerified() {
	const (
		verifier0  = "io.stackrox.signatureintegration.00000000-0000-0000-0000-000000000001"
		verifier1  = "io.stackrox.signatureintegration.00000000-0000-0000-0000-000000000002"
		verifier2  = "io.stackrox.signatureintegration.00000000-0000-0000-0000-000000000003"
		verifier3  = "io.stackrox.signatureintegration.00000000-0000-0000-0000-000000000004"
		unverifier = "io.stackrox.signatureintegration.00000000-0000-0000-0000-00000000000F"
	)

	var images = []*storage.Image{
		suite.imageWithSignatureVerificationResults("image_no_results", []*storage.ImageSignatureVerificationResult{{}}),
		suite.imageWithSignatureVerificationResults("image_empty_results", []*storage.ImageSignatureVerificationResult{storage.ImageSignatureVerificationResult_builder{
			VerifierId: "",
			Status:     storage.ImageSignatureVerificationResult_UNSET,
		}.Build()}),
		suite.imageWithSignatureVerificationResults("image_nil_results", nil),
		suite.imageWithSignatureVerificationResults("verified_by_0", []*storage.ImageSignatureVerificationResult{storage.ImageSignatureVerificationResult_builder{
			VerifierId:              verifier0,
			Status:                  storage.ImageSignatureVerificationResult_VERIFIED,
			VerifiedImageReferences: []string{"verified_by_0"},
		}.Build()}),
		suite.imageWithSignatureVerificationResults("unverified_image", []*storage.ImageSignatureVerificationResult{storage.ImageSignatureVerificationResult_builder{
			VerifierId: unverifier,
			Status:     storage.ImageSignatureVerificationResult_UNSET,
		}.Build()}),
		suite.imageWithSignatureVerificationResults("verified_by_3", []*storage.ImageSignatureVerificationResult{storage.ImageSignatureVerificationResult_builder{
			VerifierId: verifier2,
			Status:     storage.ImageSignatureVerificationResult_FAILED_VERIFICATION,
		}.Build(), storage.ImageSignatureVerificationResult_builder{
			VerifierId:              verifier3,
			Status:                  storage.ImageSignatureVerificationResult_VERIFIED,
			VerifiedImageReferences: []string{"verified_by_3"},
		}.Build()}),
		suite.imageWithSignatureVerificationResults("verified_by_2_and_3", []*storage.ImageSignatureVerificationResult{storage.ImageSignatureVerificationResult_builder{
			VerifierId:              verifier2,
			Status:                  storage.ImageSignatureVerificationResult_VERIFIED,
			VerifiedImageReferences: []string{"verified_by_2_and_3"},
		}.Build(), storage.ImageSignatureVerificationResult_builder{
			VerifierId:              verifier3,
			Status:                  storage.ImageSignatureVerificationResult_VERIFIED,
			VerifiedImageReferences: []string{"verified_by_2_and_3"},
		}.Build()}),
	}

	var allImages set.FrozenStringSet
	{
		ai := set.NewStringSet()
		for _, img := range images {
			ai.Add(img.GetName().GetFullName())
		}
		allImages = ai.Freeze()
	}

	getViolationMessage := func(img *storage.Image) string {
		message := strings.Builder{}
		message.WriteString("Image signature is not verified by the specified signature integration(s)")
		successfulVerifierIDs := []string{}
		for _, r := range img.GetSignatureVerificationData().GetResults() {
			if r.GetVerifierId() != "" && r.GetStatus() == storage.ImageSignatureVerificationResult_VERIFIED {
				successfulVerifierIDs = append(successfulVerifierIDs, r.GetVerifierId())
			}
		}
		if len(successfulVerifierIDs) > 0 {
			message.WriteString(fmt.Sprintf(" (it is verified by other integration(s): %s)", printer.StringSliceToSortedSentence(successfulVerifierIDs)))
		}
		message.WriteString(".")
		return message.String()
	}

	suite.Run("Test disallowed AND operator", func() {
		_, err := BuildImageMatcher(policyWithSingleFieldAndValues(fieldnames.ImageSignatureVerifiedBy,
			[]string{verifier0}, false, storage.BooleanOperator_AND))
		suite.EqualError(err,
			"policy validation error: operator AND is not allowed for field \"Image Signature Verified By\"")
	})

	for i, testCase := range []struct {
		values          []string
		expectedMatches set.FrozenStringSet
	}{
		{
			values:          []string{unverifier},
			expectedMatches: allImages,
		},
		{
			values:          []string{verifier0},
			expectedMatches: allImages.Difference(set.NewFrozenStringSet("verified_by_0")),
		},
		{
			values:          []string{verifier1},
			expectedMatches: allImages,
		},
		{
			values:          []string{verifier2},
			expectedMatches: allImages.Difference(set.NewFrozenStringSet("verified_by_2_and_3")),
		},
		{
			values:          []string{verifier3},
			expectedMatches: allImages.Difference(set.NewFrozenStringSet("verified_by_3", "verified_by_2_and_3")),
		},
		{
			values:          []string{verifier0, verifier2},
			expectedMatches: allImages.Difference(set.NewFrozenStringSet("verified_by_0", "verified_by_2_and_3")),
		},
		{
			values:          []string{verifier2, verifier3},
			expectedMatches: allImages.Difference(set.NewFrozenStringSet("verified_by_3", "verified_by_2_and_3")),
		},
	} {
		c := testCase

		suite.Run(fmt.Sprintf("ImageMatcher %d: %+v", i, c), func() {
			imgMatcher, err := BuildImageMatcher(policyWithSingleFieldAndValues(fieldnames.ImageSignatureVerifiedBy,
				c.values, false, storage.BooleanOperator_OR))
			suite.NoError(err)
			matchedImages := set.NewStringSet()
			for _, img := range images {
				violations, err := imgMatcher.MatchImage(nil, img)
				suite.NoError(err)
				if len(violations.AlertViolations) == 0 {
					continue
				}
				matchedImages.Add(img.GetName().GetFullName())
				suite.Truef(c.expectedMatches.Contains(img.GetName().GetFullName()), "Image %q should not match",
					img.GetName().GetFullName())

				for _, violation := range violations.AlertViolations {
					suite.Equal(getViolationMessage(img), violation.GetMessage())
				}
			}
			suite.True(c.expectedMatches.Difference(matchedImages.Freeze()).IsEmpty(), matchedImages)
		})
	}
}

func (suite *DefaultPoliciesTestSuite) TestImageVerified_WithDeployment() {
	const (
		verifier1 = "io.stackrox.signatureintegration.00000000-0000-0000-0000-000000000002"
		verifier2 = "io.stackrox.signatureintegration.00000000-0000-0000-0000-000000000003"
		verifier3 = "io.stackrox.signatureintegration.00000000-0000-0000-0000-000000000004"
	)

	isvr := &storage.ImageSignatureVerificationResult{}
	isvr.SetVerifierId(verifier1)
	isvr.SetStatus(storage.ImageSignatureVerificationResult_VERIFIED)
	isvr.SetVerifiedImageReferences([]string{"image_verified_by_1"})
	imgVerifiedAndMatchingReference := suite.imageWithSignatureVerificationResults("image_verified_by_1",
		[]*storage.ImageSignatureVerificationResult{
			isvr,
		})

	isvr2 := &storage.ImageSignatureVerificationResult{}
	isvr2.SetVerifierId(verifier3)
	isvr2.SetStatus(storage.ImageSignatureVerificationResult_VERIFIED)
	isvr2.SetVerifiedImageReferences([]string{"image_with_alternative_verified_reference", "image_verified_by_2"})
	imgVerifiedAndMatchingMultipleReferences := suite.imageWithSignatureVerificationResults("image_verified_by_2",
		[]*storage.ImageSignatureVerificationResult{
			isvr2,
		})

	isvr3 := &storage.ImageSignatureVerificationResult{}
	isvr3.SetVerifierId(verifier2)
	isvr3.SetStatus(storage.ImageSignatureVerificationResult_VERIFIED)
	isvr3.SetVerifiedImageReferences([]string{"image_verified_by_2"})
	imgVerifiedButNotMatchingReference := suite.imageWithSignatureVerificationResults("image_with_alternative_verified_reference",
		[]*storage.ImageSignatureVerificationResult{
			isvr3,
		})

	cases := map[string]struct {
		deployment       *storage.Deployment
		image            *storage.Image
		matchingVerifier string
		expectViolation  bool
	}{
		"deployment with matching verified image reference shouldn't lead in alert message": {
			deployment:       deploymentWithImage("deployment_with_image_verified_by_1", imgVerifiedAndMatchingReference),
			image:            imgVerifiedAndMatchingReference,
			matchingVerifier: verifier1,
		},
		"deployment with verified result but no matching verified image reference should lead to alert message": {
			deployment:       deploymentWithImage("deployment_with_image_alternative_verified_reference", imgVerifiedButNotMatchingReference),
			image:            imgVerifiedButNotMatchingReference,
			matchingVerifier: verifier2,
			expectViolation:  true,
		},
		"deployment with verified result and multiple matching verified image references shouldn't lead to alert message": {
			deployment:       deploymentWithImage("deployment_with_image_verified_by_2", imgVerifiedAndMatchingMultipleReferences),
			image:            imgVerifiedAndMatchingMultipleReferences,
			matchingVerifier: verifier3,
		},
	}

	for name, c := range cases {
		suite.Run(name, func() {
			deploymentMatcher, err := BuildDeploymentMatcher(policyWithSingleFieldAndValues(fieldnames.ImageSignatureVerifiedBy,
				[]string{c.matchingVerifier}, false, storage.BooleanOperator_OR))
			suite.Require().NoError(err)

			violations, err := deploymentMatcher.MatchDeployment(nil, EnhancedDeployment{
				Deployment: c.deployment,
				Images:     []*storage.Image{c.image},
			})
			suite.Require().NoError(err)

			if c.expectViolation {
				suite.NotEmpty(violations.AlertViolations)
			} else {
				suite.Empty(violations.AlertViolations)
			}
		})
	}
}

func (suite *DefaultPoliciesTestSuite) TestContainerName() {
	var deps []*storage.Deployment
	for _, containerName := range []string{
		"container_staging",
		"container_prod0",
		"container_prod1",
		"container_internal",
		"external_container",
	} {
		dep := fixtures.GetDeployment().CloneVT()
		container := &storage.Container{}
		container.SetName(containerName)
		dep.SetContainers([]*storage.Container{
			container,
		})
		deps = append(deps, dep)
	}

	for _, testCase := range []struct {
		value           string
		expectedMatches []string
		negate          bool
	}{
		{
			value:           "container_[a-z0-9]*",
			expectedMatches: []string{"container_staging", "container_prod0", "container_prod1", "container_internal"},
			negate:          false,
		},
		{
			value:           "container_prod[a-z0-9]*",
			expectedMatches: []string{"container_prod0", "container_prod1"},
			negate:          false,
		},
		{
			value:           ".*external.*",
			expectedMatches: []string{"external_container"},
			negate:          false,
		},
		{
			value:           "doesnotexist",
			expectedMatches: nil,
			negate:          false,
		},
		{
			value:           ".*internal.*",
			expectedMatches: []string{"container_staging", "container_prod0", "container_prod1", "external_container"},
			negate:          true,
		},
	} {
		c := testCase

		suite.T().Run(fmt.Sprintf("DeploymentMatcher %+v", c), func(t *testing.T) {
			depMatcher, err := BuildDeploymentMatcher(policyWithSingleKeyValue(fieldnames.ContainerName, c.value, c.negate))
			require.NoError(t, err)
			containerNameMatched := set.NewStringSet()
			for _, dep := range deps {
				violations, err := depMatcher.MatchDeployment(nil, enhancedDeployment(dep, suite.getImagesForDeployment(dep)))
				require.NoError(t, err)
				// No match in case we are testing for doesnotexist
				if len(violations.AlertViolations) > 0 {
					containerNameMatched.Add(dep.GetContainers()[0].GetName())
					require.Len(t, violations.AlertViolations, 1)
					assert.Equal(t, fmt.Sprintf("Container has name '%s'", dep.GetContainers()[0].GetName()), violations.AlertViolations[0].GetMessage())
				}
			}
			assert.ElementsMatch(t, containerNameMatched.AsSlice(), c.expectedMatches, "Got %v for policy %v; expected: %v", containerNameMatched.AsSlice(), c.value, c.expectedMatches)
		})
	}
}

func (suite *DefaultPoliciesTestSuite) TestAllowPrivilegeEscalationPolicyCriteria() {
	const containerAllowPrivEsc = "Container with Privilege Escalation allowed"
	const containerNotAllowPrivEsc = "Container with Privilege Escalation not allowed"

	var deps []*storage.Deployment
	for _, d := range []struct {
		ContainerName            string
		AllowPrivilegeEscalation bool
	}{
		{
			ContainerName:            containerAllowPrivEsc,
			AllowPrivilegeEscalation: true,
		},
		{
			ContainerName:            containerNotAllowPrivEsc,
			AllowPrivilegeEscalation: false,
		},
	} {
		dep := fixtures.GetDeployment().CloneVT()
		dep.GetContainers()[0].SetName(d.ContainerName)
		if d.AllowPrivilegeEscalation {
			dep.GetContainers()[0].GetSecurityContext().SetAllowPrivilegeEscalation(d.AllowPrivilegeEscalation)
		}
		deps = append(deps, dep)
	}

	for _, testCase := range []struct {
		CaseName        string
		value           string
		expectedMatches []string
	}{
		{
			CaseName:        "Policy for containers with privilege escalation allowed",
			value:           "true",
			expectedMatches: []string{containerAllowPrivEsc},
		},
		{
			CaseName:        "Policy for containers with privilege escalation not allowed",
			value:           "false",
			expectedMatches: []string{containerNotAllowPrivEsc},
		},
	} {
		c := testCase

		suite.T().Run(c.CaseName, func(t *testing.T) {
			depMatcher, err := BuildDeploymentMatcher(policyWithSingleKeyValue(fieldnames.AllowPrivilegeEscalation, c.value, false))
			require.NoError(t, err)
			containerNameMatched := set.NewStringSet()
			for _, dep := range deps {
				violations, err := depMatcher.MatchDeployment(nil, enhancedDeployment(dep, suite.getImagesForDeployment(dep)))
				require.NoError(t, err)
				if len(violations.AlertViolations) > 0 {
					containerNameMatched.Add(dep.GetContainers()[0].GetName())
					require.Len(t, violations.AlertViolations, 1)
					if c.value == "true" {
						assert.Equal(t, fmt.Sprintf("Container '%s' allows privilege escalation", dep.GetContainers()[0].GetName()), violations.AlertViolations[0].GetMessage())
					} else {
						assert.Equal(t, fmt.Sprintf("Container '%s' does not allow privilege escalation", dep.GetContainers()[0].GetName()), violations.AlertViolations[0].GetMessage())
					}
				}
			}
			assert.ElementsMatch(t, containerNameMatched.AsSlice(), c.expectedMatches, "Matched containers %v for policy %v; expected: %v", containerNameMatched.AsSlice(), c.value, c.expectedMatches)
		})
	}
}

func (suite *DefaultPoliciesTestSuite) TestAutomountServiceAccountToken() {
	deployments := make(map[string]*storage.Deployment)
	for _, d := range []struct {
		DeploymentName                string
		ServiceAccountName            string
		AutomountServiceAccountTokens bool
	}{
		{
			DeploymentName:                "DefaultSAAutomountedTokens",
			ServiceAccountName:            "default",
			AutomountServiceAccountTokens: true,
		},
		{
			DeploymentName:     "DefaultSANotAutomountedTokens",
			ServiceAccountName: "default",
		},
		{
			DeploymentName:                "CustomSAAutomountedTokens",
			ServiceAccountName:            "custom",
			AutomountServiceAccountTokens: true,
		},
		{
			DeploymentName:     "CustomSANotAutomountedTokens",
			ServiceAccountName: "custom",
		},
	} {
		dep := fixtures.GetDeployment().CloneVT()
		dep.SetName(d.DeploymentName)
		dep.SetServiceAccount(d.ServiceAccountName)
		dep.SetAutomountServiceAccountToken(d.AutomountServiceAccountTokens)
		deployments[dep.GetName()] = dep
	}

	pv := &storage.PolicyValue{}
	pv.SetValue("true")
	automountServiceAccountTokenPolicyGroup := &storage.PolicyGroup{}
	automountServiceAccountTokenPolicyGroup.SetFieldName(fieldnames.AutomountServiceAccountToken)
	automountServiceAccountTokenPolicyGroup.SetValues([]*storage.PolicyValue{pv})
	pv2 := &storage.PolicyValue{}
	pv2.SetValue("default")
	defaultServiceAccountPolicyGroup := &storage.PolicyGroup{}
	defaultServiceAccountPolicyGroup.SetFieldName(fieldnames.ServiceAccount)
	defaultServiceAccountPolicyGroup.SetValues([]*storage.PolicyValue{pv2})

	allAutomountServiceAccountTokenPolicy := policyWithGroups(storage.EventSource_NOT_APPLICABLE, automountServiceAccountTokenPolicyGroup)
	defaultAutomountServiceAccountTokenPolicy := policyWithGroups(storage.EventSource_NOT_APPLICABLE, automountServiceAccountTokenPolicyGroup, defaultServiceAccountPolicyGroup)

	automountAlert := &storage.Alert_Violation{}
	automountAlert.SetMessage("Deployment mounts the service account tokens.")
	defaultServiceAccountAlert := &storage.Alert_Violation{}
	defaultServiceAccountAlert.SetMessage("Service Account is set to 'default'")

	for _, c := range []struct {
		CaseName       string
		Policy         *storage.Policy
		DeploymentName string
		ExpectedAlerts []*storage.Alert_Violation
	}{
		{
			CaseName:       "Automounted default service account tokens should alert on bare automount policy",
			Policy:         allAutomountServiceAccountTokenPolicy,
			DeploymentName: "DefaultSAAutomountedTokens",
			ExpectedAlerts: []*storage.Alert_Violation{automountAlert},
		},
		{
			CaseName:       "Automounted default service account tokens should alert on default only automount policy",
			Policy:         defaultAutomountServiceAccountTokenPolicy,
			DeploymentName: "DefaultSAAutomountedTokens",
			ExpectedAlerts: []*storage.Alert_Violation{automountAlert, defaultServiceAccountAlert},
		},
		{
			CaseName:       "Automounted custom service account tokens should alert on bare automount policy",
			Policy:         allAutomountServiceAccountTokenPolicy,
			DeploymentName: "CustomSAAutomountedTokens",
			ExpectedAlerts: []*storage.Alert_Violation{automountAlert},
		},
		{
			CaseName:       "Not automounted default service account should not alert on bare automount policy",
			Policy:         allAutomountServiceAccountTokenPolicy,
			DeploymentName: "DefaultSANotAutomountedTokens",
		},
		{
			CaseName:       "Not automounted custom service account should not alert on bare automount policy",
			Policy:         allAutomountServiceAccountTokenPolicy,
			DeploymentName: "CustomSANotAutomountedTokens",
		},
	} {
		suite.T().Run(c.CaseName, func(t *testing.T) {
			dep := deployments[c.DeploymentName]
			matcher, err := BuildDeploymentMatcher(c.Policy)
			suite.NoError(err, "deployment matcher creation must succeed")
			violations, err := matcher.MatchDeployment(nil, enhancedDeployment(dep, suite.getImagesForDeployment(dep)))
			suite.NoError(err, "deployment matcher run must succeed")
			suite.Empty(violations.ProcessViolation)
			protoassert.SlicesEqual(suite.T(), c.ExpectedAlerts, violations.AlertViolations)
		})
	}
}

func (suite *DefaultPoliciesTestSuite) TestRuntimeClass() {
	var deps []*storage.Deployment
	for _, runtimeClass := range []string{
		"",
		"blah",
	} {
		dep := fixtures.GetDeployment().CloneVT()
		dep.SetRuntimeClass(runtimeClass)
		deps = append(deps, dep)
	}

	for _, testCase := range []struct {
		value           string
		negate          bool
		expectedMatches []string
	}{
		{
			value:           ".*",
			negate:          false,
			expectedMatches: []string{"", "blah"},
		},
		{
			value:           ".+",
			negate:          false,
			expectedMatches: []string{"blah"},
		},
		{
			value:           ".+",
			negate:          true,
			expectedMatches: []string{""},
		},
		{
			value:           "blah",
			negate:          true,
			expectedMatches: []string{""},
		},
	} {
		c := testCase

		suite.T().Run(fmt.Sprintf("%+v", c), func(t *testing.T) {
			depMatcher, err := BuildDeploymentMatcher(policyWithSingleKeyValue(fieldnames.RuntimeClass, c.value, c.negate))
			require.NoError(t, err)
			matchedRuntimeClasses := set.NewStringSet()
			for _, dep := range deps {
				violations, err := depMatcher.MatchDeployment(nil, enhancedDeployment(dep, suite.getImagesForDeployment(dep)))
				require.NoError(t, err)
				if len(violations.AlertViolations) > 0 {
					matchedRuntimeClasses.Add(dep.GetRuntimeClass())
					require.Len(t, violations.AlertViolations, 1)
					assert.Equal(t, fmt.Sprintf("Runtime Class is set to '%s'", dep.GetRuntimeClass()), violations.AlertViolations[0].GetMessage())
				}
			}
			assert.ElementsMatch(t, matchedRuntimeClasses.AsSlice(), c.expectedMatches, "Got %v for policy %v; expected: %v", matchedRuntimeClasses.AsSlice(), c.value, c.expectedMatches)
		})
	}
}

func (suite *DefaultPoliciesTestSuite) TestNamespace() {
	var deps []*storage.Deployment
	for _, namespace := range []string{
		"dep_staging",
		"dep_prod0",
		"dep_prod1",
		"dep_internal",
		"external_dep",
	} {
		dep := fixtures.GetDeployment().CloneVT()
		dep.SetNamespace(namespace)
		deps = append(deps, dep)
	}

	for _, testCase := range []struct {
		value           string
		expectedMatches []string
		negate          bool
	}{
		{
			value:           "dep_[a-z0-9]*",
			expectedMatches: []string{"dep_staging", "dep_prod0", "dep_prod1", "dep_internal"},
			negate:          false,
		},
		{
			value:           "dep_prod[a-z0-9]*",
			expectedMatches: []string{"dep_prod0", "dep_prod1"},
			negate:          false,
		},
		{
			value:           ".*external.*",
			expectedMatches: []string{"external_dep"},
			negate:          false,
		},
		{
			value:           "doesnotexist",
			expectedMatches: nil,
			negate:          false,
		},
		{
			value:           ".*internal.*",
			expectedMatches: []string{"dep_staging", "dep_prod0", "dep_prod1", "external_dep"},
			negate:          true,
		},
	} {
		c := testCase

		suite.T().Run(fmt.Sprintf("DeploymentMatcher %+v", c), func(t *testing.T) {
			depMatcher, err := BuildDeploymentMatcher(policyWithSingleKeyValue(fieldnames.Namespace, c.value, c.negate))
			require.NoError(t, err)
			namespacesMatched := set.NewStringSet()
			for _, dep := range deps {
				violations, err := depMatcher.MatchDeployment(nil, enhancedDeployment(dep, suite.getImagesForDeployment(dep)))
				require.NoError(t, err)
				// No match in case we are testing for doesnotexist
				if len(violations.AlertViolations) > 0 {
					namespacesMatched.Add(dep.GetNamespace())
					require.Len(t, violations.AlertViolations, 1)
					assert.Equal(t, fmt.Sprintf("Namespace has name '%s'", dep.GetNamespace()), violations.AlertViolations[0].GetMessage())
				}
			}
			assert.ElementsMatch(t, namespacesMatched.AsSlice(), c.expectedMatches, "Got %v for policy %v; expected: %v", namespacesMatched.AsSlice(), c.value, c.expectedMatches)
		})
	}
}

func (suite *DefaultPoliciesTestSuite) TestDropCaps() {
	testCaps := []string{"SYS_MODULE", "SYS_NICE", "SYS_PTRACE", "ALL"}

	deployments := make(map[string]*storage.Deployment)
	for _, idxs := range [][]int{{}, {0}, {1}, {2}, {0, 1}, {1, 2}, {0, 1, 2}, {3}} {
		dep := fixtures.GetDeployment().CloneVT()
		dep.GetContainers()[0].GetSecurityContext().SetDropCapabilities(make([]string, 0, len(idxs)))
		for _, idx := range idxs {
			dep.GetContainers()[0].GetSecurityContext().SetDropCapabilities(append(dep.GetContainers()[0].GetSecurityContext().GetDropCapabilities(), testCaps[idx]))
		}
		deployments[strings.ReplaceAll(strings.Join(dep.GetContainers()[0].GetSecurityContext().GetDropCapabilities(), ","), "SYS_", "")] = dep
	}

	assertMessageMatches := func(t *testing.T, depRef string, violations []*storage.Alert_Violation) {
		depRefToExpectedMsg := map[string]string{
			"":                   "no capabilities",
			"ALL":                "all capabilities",
			"MODULE":             "SYS_MODULE",
			"NICE":               "SYS_NICE",
			"PTRACE":             "SYS_PTRACE",
			"MODULE,NICE":        "SYS_MODULE and SYS_NICE",
			"NICE,PTRACE":        "SYS_NICE and SYS_PTRACE",
			"MODULE,NICE,PTRACE": "SYS_MODULE, SYS_NICE, and SYS_PTRACE",
		}
		require.Len(t, violations, 1)
		assert.Equal(t, fmt.Sprintf("Container 'nginx110container' does not drop expected capabilities (drops %s)", depRefToExpectedMsg[depRef]), violations[0].GetMessage())
	}

	for _, testCase := range []struct {
		values          []string
		op              storage.BooleanOperator
		expectedMatches []string
	}{
		{
			// Nothing drops this capability
			[]string{"SYSLOG"},
			storage.BooleanOperator_OR,
			[]string{"", "MODULE", "NICE", "PTRACE", "MODULE,NICE", "NICE,PTRACE", "MODULE,NICE,PTRACE"},
		},
		{
			[]string{"SYS_NICE"},
			storage.BooleanOperator_OR,
			[]string{"", "MODULE", "PTRACE"},
		},
		{
			[]string{"SYS_NICE", "SYS_PTRACE"},
			storage.BooleanOperator_OR,
			[]string{"", "MODULE"},
		},
		{
			[]string{"SYS_NICE", "SYS_PTRACE"},
			storage.BooleanOperator_AND,
			[]string{"", "MODULE", "PTRACE", "NICE", "MODULE,NICE"},
		},
		{
			[]string{"ALL"},
			storage.BooleanOperator_AND,
			[]string{"", "MODULE", "NICE", "PTRACE", "MODULE,NICE", "NICE,PTRACE", "MODULE,NICE,PTRACE"},
		},
	} {
		c := testCase
		suite.T().Run(fmt.Sprintf("%+v", c), func(t *testing.T) {
			matcher, err := BuildDeploymentMatcher(policyWithSingleFieldAndValues(fieldnames.DropCaps, c.values, false, c.op))
			require.NoError(t, err)
			matched := set.NewStringSet()
			for depRef, dep := range deployments {
				violations, err := matcher.MatchDeployment(nil, enhancedDeployment(dep, suite.getImagesForDeployment(dep)))
				require.NoError(t, err)
				if len(violations.AlertViolations) > 0 {
					matched.Add(depRef)
					assertMessageMatches(t, depRef, violations.AlertViolations)
				}
			}
			assert.ElementsMatch(t, matched.AsSlice(), c.expectedMatches, "Got %v, expected: %v", matched.AsSlice(), c.expectedMatches)
		})
	}
}

func (suite *DefaultPoliciesTestSuite) TestAddCaps() {
	testCaps := []string{"SYS_MODULE", "SYS_NICE", "SYS_PTRACE"}

	deployments := make(map[string]*storage.Deployment)
	for _, idxs := range [][]int{{}, {0}, {1}, {2}, {0, 1}, {1, 2}, {0, 1, 2}} {
		dep := fixtures.GetDeployment().CloneVT()
		dep.GetContainers()[0].GetSecurityContext().SetAddCapabilities(make([]string, 0, len(idxs)))
		for _, idx := range idxs {
			dep.GetContainers()[0].GetSecurityContext().SetAddCapabilities(append(dep.GetContainers()[0].GetSecurityContext().GetAddCapabilities(), testCaps[idx]))
		}
		deployments[strings.ReplaceAll(strings.Join(dep.GetContainers()[0].GetSecurityContext().GetAddCapabilities(), ","), "SYS_", "")] = dep
	}

	for _, testCase := range []struct {
		values          []string
		op              storage.BooleanOperator
		expectedMatches []string
	}{
		{
			// Nothing adds this capability
			[]string{"SYSLOG"},
			storage.BooleanOperator_OR,
			[]string{},
		},
		{
			[]string{"SYS_NICE"},
			storage.BooleanOperator_OR,
			[]string{"NICE", "MODULE,NICE", "NICE,PTRACE", "MODULE,NICE,PTRACE"},
		},
		{
			[]string{"SYS_NICE", "SYS_PTRACE"},
			storage.BooleanOperator_OR,
			[]string{"NICE", "PTRACE", "MODULE,NICE", "NICE,PTRACE", "MODULE,NICE,PTRACE"},
		},
		{
			[]string{"SYS_NICE", "SYS_PTRACE"},
			storage.BooleanOperator_AND,
			[]string{"NICE,PTRACE", "MODULE,NICE,PTRACE"},
		},
	} {
		c := testCase
		suite.T().Run(fmt.Sprintf("%+v", c), func(t *testing.T) {
			matcher, err := BuildDeploymentMatcher(policyWithSingleFieldAndValues(fieldnames.AddCaps, c.values, false, c.op))
			require.NoError(t, err)
			matched := set.NewStringSet()
			for depRef, dep := range deployments {
				violations, err := matcher.MatchDeployment(nil, enhancedDeployment(dep, suite.getImagesForDeployment(dep)))
				require.NoError(t, err)
				if len(violations.AlertViolations) > 0 {
					matched.Add(depRef)
					require.Len(t, violations.AlertViolations, 1)
				}
			}
			assert.ElementsMatch(t, matched.AsSlice(), c.expectedMatches, "Got %v, expected: %v", matched.AsSlice(), c.expectedMatches)
		})
	}
}

func (suite *DefaultPoliciesTestSuite) TestProcessBaseline() {
	privilegedDep := fixtures.GetDeployment().CloneVT()
	privilegedDep.SetId("PRIVILEGED")
	suite.addDepAndImages(privilegedDep)

	nonPrivilegedDep := fixtures.GetDeployment().CloneVT()
	nonPrivilegedDep.SetId("NOTPRIVILEGED")
	nonPrivilegedDep.GetContainers()[0].GetSecurityContext().SetPrivileged(false)
	suite.addDepAndImages(nonPrivilegedDep)

	const aptGetKey = "apt-get"
	const aptGet2Key = "apt-get2"
	const curlKey = "curl"
	const bashKey = "bash"

	indicators := make(map[string]map[string]*storage.ProcessIndicator)
	for _, dep := range []*storage.Deployment{privilegedDep, nonPrivilegedDep} {
		indicators[dep.GetId()] = map[string]*storage.ProcessIndicator{
			aptGetKey:  suite.addIndicator(dep.GetId(), "apt-get", "install nginx", "/bin/apt-get", nil, 0),
			aptGet2Key: suite.addIndicator(dep.GetId(), "apt-get", "update", "/bin/apt-get", nil, 0),
			curlKey:    suite.addIndicator(dep.GetId(), "curl", "https://stackrox.io", "/bin/curl", nil, 0),
			bashKey:    suite.addIndicator(dep.GetId(), "bash", "attach.sh", "/bin/bash", nil, 0),
		}
	}
	processesNotInBaseline := map[string]set.StringSet{
		privilegedDep.GetId():    set.NewStringSet(aptGetKey, aptGet2Key, bashKey),
		nonPrivilegedDep.GetId(): set.NewStringSet(aptGetKey, curlKey, bashKey),
	}

	// Plain groups
	aptGetGroup := policyGroupWithSingleKeyValue(fieldnames.ProcessName, "apt-get", false)
	privilegedGroup := policyGroupWithSingleKeyValue(fieldnames.PrivilegedContainer, "true", false)
	baselineGroup := policyGroupWithSingleKeyValue(fieldnames.UnexpectedProcessExecuted, "true", false)

	for _, testCase := range []struct {
		groups []*storage.PolicyGroup

		// Deployment ids to indicator keys
		expectedMatches        map[string][]string
		expectedProcessMatches map[string][]string
		// Deployment ids to violations
		expectedViolations map[string][]*storage.Alert_Violation
	}{
		{
			groups: []*storage.PolicyGroup{aptGetGroup},
			// only process violation, no alert violation
			expectedMatches: map[string][]string{},
			expectedProcessMatches: map[string][]string{
				privilegedDep.GetId():    {aptGetKey, aptGet2Key},
				nonPrivilegedDep.GetId(): {aptGetKey, aptGet2Key},
			},
		},
		{
			groups:          []*storage.PolicyGroup{baselineGroup},
			expectedMatches: map[string][]string{},
			expectedProcessMatches: map[string][]string{
				privilegedDep.GetId():    {aptGetKey, aptGet2Key, bashKey},
				nonPrivilegedDep.GetId(): {aptGetKey, curlKey, bashKey},
			},
		},

		{
			groups: []*storage.PolicyGroup{privilegedGroup},
			expectedMatches: map[string][]string{
				privilegedDep.GetId(): {aptGetKey, aptGet2Key, curlKey, bashKey},
			},
			expectedProcessMatches: map[string][]string{},
			expectedViolations: map[string][]*storage.Alert_Violation{
				privilegedDep.GetId(): processBaselineMessage(privilegedDep, false, true, "apt-get", "apt-get", "curl", "bash"),
			},
		},
		{
			groups:          []*storage.PolicyGroup{aptGetGroup, baselineGroup},
			expectedMatches: map[string][]string{},
			expectedProcessMatches: map[string][]string{
				privilegedDep.GetId():    {aptGetKey, aptGet2Key},
				nonPrivilegedDep.GetId(): {aptGetKey},
			},
		},
		{
			groups: []*storage.PolicyGroup{aptGetGroup, privilegedGroup},
			expectedMatches: map[string][]string{
				privilegedDep.GetId(): {aptGetKey, aptGet2Key},
			},
			expectedViolations: map[string][]*storage.Alert_Violation{
				privilegedDep.GetId(): processBaselineMessage(privilegedDep, false, true, "apt-get", "apt-get"),
			},
			expectedProcessMatches: map[string][]string{
				privilegedDep.GetId(): {aptGetKey, aptGet2Key},
			},
		},
		{
			groups: []*storage.PolicyGroup{privilegedGroup, baselineGroup},
			expectedMatches: map[string][]string{
				privilegedDep.GetId(): {aptGetKey, aptGet2Key, bashKey},
			},
			expectedProcessMatches: map[string][]string{
				privilegedDep.GetId(): {aptGetKey, aptGet2Key, bashKey},
			},
		},
		{
			groups: []*storage.PolicyGroup{aptGetGroup, privilegedGroup, baselineGroup},
			expectedMatches: map[string][]string{
				privilegedDep.GetId(): {aptGetKey, aptGet2Key},
			},
			expectedProcessMatches: map[string][]string{
				privilegedDep.GetId(): {aptGetKey, aptGet2Key},
			},
		},
	} {
		c := testCase
		suite.T().Run(fmt.Sprintf("%+v", c.groups), func(t *testing.T) {
			policy := policyWithGroups(storage.EventSource_DEPLOYMENT_EVENT, c.groups...)

			m, err := BuildDeploymentWithProcessMatcher(policy)
			require.NoError(t, err)

			actualMatches := make(map[string][]string)
			actualProcessMatches := make(map[string][]string)
			actualViolations := make(map[string][]*storage.Alert_Violation)
			for _, dep := range []*storage.Deployment{privilegedDep, nonPrivilegedDep} {
				for _, key := range []string{aptGetKey, aptGet2Key, curlKey, bashKey} {
					violations, err := m.MatchDeploymentWithProcess(nil, enhancedDeployment(dep, suite.getImagesForDeployment(dep)), indicators[dep.GetId()][key], processesNotInBaseline[dep.GetId()].Contains(key))
					suite.Require().NoError(err)
					if len(violations.AlertViolations) > 0 {
						actualMatches[dep.GetId()] = append(actualMatches[dep.GetId()], key)
						actualViolations[dep.GetId()] = append(actualViolations[dep.GetId()], violations.AlertViolations...)
					}
					if violations.ProcessViolation != nil {
						actualProcessMatches[dep.GetId()] = append(actualProcessMatches[dep.GetId()], key)
					}

				}
			}
			assert.Equal(t, c.expectedMatches, actualMatches)
			assert.Equal(t, c.expectedProcessMatches, actualProcessMatches)

			for id, violations := range c.expectedViolations {
				assert.Contains(t, actualViolations, id)
				protoassert.ElementsMatch(t, violations, actualViolations[id])
			}
		})
	}
}

func (suite *DefaultPoliciesTestSuite) TestKubeEventConstraints() {
	createVerbGroup := policyGroupWithSingleKeyValue(fieldnames.KubeAPIVerb, "CREATE", false)
	podExecGroup := policyGroupWithSingleKeyValue(fieldnames.KubeResource, "PODS_EXEC", false)

	aptGetGroup := policyGroupWithSingleKeyValue(fieldnames.ProcessName, "apt-get", false)

	for _, c := range []struct {
		event              *storage.KubernetesEvent
		groups             []*storage.PolicyGroup
		expectedViolations []*storage.Alert_Violation
		builderErr         bool
		withProcessSection bool
	}{
		{
			event:              podExecEvent("p1", "c1", "cmd"),
			groups:             []*storage.PolicyGroup{createVerbGroup, podExecGroup},
			expectedViolations: []*storage.Alert_Violation{podExecViolationMsg("p1", "c1", "cmd")},
		},
		{
			event:              podExecEvent("p1", "c1", ""),
			groups:             []*storage.PolicyGroup{podExecGroup},
			expectedViolations: []*storage.Alert_Violation{podExecViolationMsg("p1", "c1", "")},
		},
		{
			event:              podExecEvent("p1", "c1", ""),
			groups:             []*storage.PolicyGroup{createVerbGroup},
			expectedViolations: []*storage.Alert_Violation{podExecViolationMsg("p1", "c1", "")},
		},
		{
			groups: []*storage.PolicyGroup{createVerbGroup, podExecGroup},
		},
		{
			event:  podPortForwardEvent("p1", 8000),
			groups: []*storage.PolicyGroup{podExecGroup},
		},
		{
			event:      podPortForwardEvent("p1", 8000),
			groups:     []*storage.PolicyGroup{podExecGroup, aptGetGroup},
			builderErr: true,
		},
		{
			event:              podExecEvent("p1", "c1", ""),
			groups:             []*storage.PolicyGroup{createVerbGroup},
			expectedViolations: []*storage.Alert_Violation{podExecViolationMsg("p1", "c1", "")},
			withProcessSection: true,
		},
	} {
		suite.T().Run(fmt.Sprintf("%+v", c.groups), func(t *testing.T) {
			policy := policyWithGroups(storage.EventSource_DEPLOYMENT_EVENT, c.groups...)
			if c.withProcessSection {
				ps := &storage.PolicySection{}
				ps.SetPolicyGroups([]*storage.PolicyGroup{aptGetGroup})
				policy.SetPolicySections(append(policy.GetPolicySections(), ps))
			}

			m, err := BuildKubeEventMatcher(policy)
			if c.builderErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)

			actualViolations, err := m.MatchKubeEvent(nil, c.event, &storage.Deployment{})
			suite.Require().NoError(err)

			assert.Nil(t, actualViolations.ProcessViolation)
			if len(c.expectedViolations) == 0 {
				assert.Nil(t, actualViolations.AlertViolations)
			} else {
				protoassert.ElementsMatch(t, c.expectedViolations, actualViolations.AlertViolations)
			}
		})
	}
}
func (suite *DefaultPoliciesTestSuite) TestKubeEventDefaultPolicies() {
	for _, c := range []struct {
		policyName         string
		event              *storage.KubernetesEvent
		expectedViolations []*storage.Alert_Violation
	}{
		{
			policyName:         "Kubernetes Actions: Exec into Pod",
			event:              podExecEvent("p1", "c1", "apt-get"),
			expectedViolations: []*storage.Alert_Violation{podExecViolationMsg("p1", "c1", "apt-get")},
		},
		{
			policyName: "Kubernetes Actions: Exec into Pod",
			event:      podPortForwardEvent("p1", 8000),
		},
		// Event without CREATE.
		{
			policyName: "Kubernetes Actions: Exec into Pod",
			event: storage.KubernetesEvent_builder{
				Object: storage.KubernetesEvent_Object_builder{
					Name:     "p1",
					Resource: storage.KubernetesEvent_Object_PODS_EXEC,
				}.Build(),
				PodExecArgs: storage.KubernetesEvent_PodExecArgs_builder{
					Container: "c1",
				}.Build(),
			}.Build(),
			expectedViolations: []*storage.Alert_Violation{podExecViolationMsg("p1", "c1", "")},
		},
		{
			policyName: "Kubernetes Actions: Port Forward to Pod",
		},
		{
			policyName:         "Kubernetes Actions: Port Forward to Pod",
			event:              podPortForwardEvent("p1", 8000),
			expectedViolations: []*storage.Alert_Violation{podPortForwardViolationMsg("p1", 8000)},
		},
		{
			policyName: "Kubernetes Actions: Port Forward to Pod",
			event: storage.KubernetesEvent_builder{
				Object: storage.KubernetesEvent_Object_builder{
					Name:     "p1",
					Resource: storage.KubernetesEvent_Object_PODS_PORTFORWARD,
				}.Build(),
				PodPortForwardArgs: storage.KubernetesEvent_PodPortForwardArgs_builder{
					Ports: []int32{8000},
				}.Build(),
			}.Build(),
			expectedViolations: []*storage.Alert_Violation{podPortForwardViolationMsg("p1", 8000)},
		},
	} {
		suite.T().Run(fmt.Sprintf("%s:%s", c.policyName, kubernetes.EventAsString(c.event)), func(t *testing.T) {
			policy := suite.MustGetPolicy(c.policyName)
			m, err := BuildKubeEventMatcher(policy)
			require.NoError(t, err)

			actualViolations, err := m.MatchKubeEvent(nil, c.event, &storage.Deployment{})
			suite.Require().NoError(err)

			assert.Nil(t, actualViolations.ProcessViolation)
			if len(c.expectedViolations) == 0 {
				for _, a := range actualViolations.AlertViolations {
					fmt.Printf("%v", protoutils.NewWrapper(a))
				}

				assert.Nil(t, actualViolations.AlertViolations)
			} else {
				protoassert.ElementsMatch(t, c.expectedViolations, actualViolations.AlertViolations)
			}
		})
	}
}

func (suite *DefaultPoliciesTestSuite) TestNetworkBaselinePolicy() {
	deployment := fixtures.GetDeployment().CloneVT()
	suite.addDepAndImages(deployment)

	// Create a policy for triggering flows that are not in baseline
	whitelistGroup := policyGroupWithSingleKeyValue(fieldnames.UnexpectedNetworkFlowDetected, "true", false)

	policy := policyWithGroups(storage.EventSource_DEPLOYMENT_EVENT, whitelistGroup)
	m, err := BuildDeploymentWithNetworkFlowMatcher(policy)
	suite.NoError(err)

	srcName, dstName, port, protocol := "deployment-name", "ext-source-name", 1, storage.L4Protocol_L4_PROTOCOL_TCP
	flow := &augmentedobjs.NetworkFlowDetails{
		SrcEntityName:        srcName,
		SrcEntityType:        storage.NetworkEntityInfo_DEPLOYMENT,
		DstEntityName:        dstName,
		DstEntityType:        storage.NetworkEntityInfo_DEPLOYMENT,
		DstPort:              uint32(port),
		L4Protocol:           protocol,
		NotInNetworkBaseline: true,
		LastSeenTimestamp:    time.Now(),
	}

	violations, err := m.MatchDeploymentWithNetworkFlowInfo(nil, enhancedDeployment(deployment, suite.getImagesForDeployment(deployment)), flow)
	suite.NoError(err)
	assertNetworkBaselineMessagesEqual(
		suite,
		violations.AlertViolations,
		[]*storage.Alert_Violation{networkBaselineMessage(suite, flow)})

	// And if the flow is in the baseline, no violations should exist
	flow.NotInNetworkBaseline = false
	violations, err = m.MatchDeploymentWithNetworkFlowInfo(nil, enhancedDeployment(deployment, suite.getImagesForDeployment(deployment)), flow)
	suite.NoError(err)
	suite.Empty(violations)
}

func (suite *DefaultPoliciesTestSuite) TestReplicasPolicyCriteria() {
	for _, testCase := range []struct {
		caseName    string
		replicas    int64
		policyValue string
		negate      bool
		alerts      []*storage.Alert_Violation
	}{
		{
			caseName:    "Should raise when replicas==5.",
			replicas:    5,
			policyValue: "5",
			negate:      false,
			alerts:      []*storage.Alert_Violation{storage.Alert_Violation_builder{Message: "Replicas is set to '5'"}.Build()},
		},
		{
			caseName:    "Should not raise unless replicas==3.",
			replicas:    5,
			policyValue: "3",
			negate:      false,
			alerts:      nil,
		},
		{
			caseName:    "Should raise unless replicas==3.",
			replicas:    5,
			policyValue: "3",
			negate:      true,
			alerts:      []*storage.Alert_Violation{storage.Alert_Violation_builder{Message: "Replicas is set to '5'"}.Build()},
		},
		{
			caseName:    "Should raise when replicas>=5.",
			replicas:    5,
			policyValue: ">=5",
			negate:      false,
			alerts:      []*storage.Alert_Violation{storage.Alert_Violation_builder{Message: "Replicas is set to '5'"}.Build()},
		},
		{
			caseName:    "Should raise when replicas<=5.",
			replicas:    5,
			policyValue: "<=5",
			negate:      false,
			alerts:      []*storage.Alert_Violation{storage.Alert_Violation_builder{Message: "Replicas is set to '5'"}.Build()},
		},
		{
			caseName:    "Should raise when replicas<5.",
			replicas:    1,
			policyValue: "<5",
			negate:      false,
			alerts:      []*storage.Alert_Violation{storage.Alert_Violation_builder{Message: "Replicas is set to '1'"}.Build()},
		},
		{
			caseName:    "Should raise when replicas>5.",
			replicas:    10,
			policyValue: ">5",
			negate:      false,
			alerts:      []*storage.Alert_Violation{storage.Alert_Violation_builder{Message: "Replicas is set to '10'"}.Build()},
		},
	} {
		suite.Run(testCase.caseName, func() {
			deployment := fixtures.GetDeployment().CloneVT()
			deployment.SetReplicas(testCase.replicas)
			policy := policyWithSingleKeyValue(fieldnames.Replicas, testCase.policyValue, testCase.negate)

			matcher, err := BuildDeploymentMatcher(policy)
			suite.NoError(err, "deployment matcher creation must succeed")
			violations, err := matcher.MatchDeployment(nil, enhancedDeployment(deployment, suite.getImagesForDeployment(deployment)))
			suite.NoError(err, "deployment matcher run must succeed")

			suite.Empty(violations.ProcessViolation)
			protoassert.SlicesEqual(suite.T(), violations.AlertViolations, testCase.alerts)
		})
	}
}

func (suite *DefaultPoliciesTestSuite) TestLivenessProbePolicyCriteria() {
	for _, testCase := range []struct {
		caseName    string
		containers  []*storage.Container
		policyValue string
		alerts      []*storage.Alert_Violation
	}{
		{
			caseName: "Should raise alert since liveness probe is defined.",
			containers: []*storage.Container{
				storage.Container_builder{Name: "container", LivenessProbe: storage.LivenessProbe_builder{Defined: true}.Build()}.Build(),
			},
			policyValue: "true",
			alerts: []*storage.Alert_Violation{
				storage.Alert_Violation_builder{Message: "Liveness probe is defined for container 'container'"}.Build(),
			},
		},
		{
			caseName: "Should not raise alert since liveness probe is defined.",
			containers: []*storage.Container{
				storage.Container_builder{Name: "container", LivenessProbe: storage.LivenessProbe_builder{Defined: true}.Build()}.Build(),
			},
			policyValue: "false",
			alerts:      nil,
		},
		{
			caseName: "Should not raise alert since liveness probe is not defined.",
			containers: []*storage.Container{
				storage.Container_builder{Name: "container", LivenessProbe: storage.LivenessProbe_builder{Defined: false}.Build()}.Build(),
			},
			policyValue: "true",
			alerts:      nil,
		},
		{
			caseName: "Should raise alert since liveness probe is not defined.",
			containers: []*storage.Container{
				storage.Container_builder{Name: "container", LivenessProbe: storage.LivenessProbe_builder{Defined: false}.Build()}.Build(),
			},
			policyValue: "false",
			alerts: []*storage.Alert_Violation{
				storage.Alert_Violation_builder{Message: "Liveness probe is not defined for container 'container'"}.Build(),
			},
		},
		{
			caseName: "Should raise alert for both containers.",
			containers: []*storage.Container{
				storage.Container_builder{Name: "container-1", LivenessProbe: storage.LivenessProbe_builder{Defined: false}.Build()}.Build(),
				storage.Container_builder{Name: "container-2", LivenessProbe: storage.LivenessProbe_builder{Defined: false}.Build()}.Build(),
			},
			policyValue: "false",
			alerts: []*storage.Alert_Violation{
				storage.Alert_Violation_builder{Message: "Liveness probe is not defined for container 'container-1'"}.Build(),
				storage.Alert_Violation_builder{Message: "Liveness probe is not defined for container 'container-2'"}.Build(),
			},
		},
		{
			caseName: "Should raise alert only for container-2.",
			containers: []*storage.Container{
				storage.Container_builder{Name: "container-1", LivenessProbe: storage.LivenessProbe_builder{Defined: true}.Build()}.Build(),
				storage.Container_builder{Name: "container-2", LivenessProbe: storage.LivenessProbe_builder{Defined: false}.Build()}.Build(),
			},
			policyValue: "false",
			alerts: []*storage.Alert_Violation{
				storage.Alert_Violation_builder{Message: "Liveness probe is not defined for container 'container-2'"}.Build(),
			},
		},
	} {
		suite.Run(testCase.caseName, func() {
			deployment := fixtures.GetDeployment().CloneVT()
			deployment.SetContainers(testCase.containers)
			policy := policyWithSingleKeyValue(fieldnames.LivenessProbeDefined, testCase.policyValue, false)

			matcher, err := BuildDeploymentMatcher(policy)
			suite.NoError(err, "deployment matcher creation must succeed")
			violations, err := matcher.MatchDeployment(nil, enhancedDeployment(deployment, suite.getImagesForDeployment(deployment)))
			suite.NoError(err, "deployment matcher run must succeed")

			suite.Empty(violations.ProcessViolation)
			protoassert.SlicesEqual(suite.T(), violations.AlertViolations, testCase.alerts)
		})
	}
}

func (suite *DefaultPoliciesTestSuite) getViolations(policy *storage.Policy, dep EnhancedDeployment) Violations {
	matcher, err := BuildDeploymentMatcher(policy)
	suite.NoError(err, "deployment matcher creation must succeed")
	violations, err := matcher.MatchDeployment(nil, dep)
	suite.NoError(err, "deployment matcher run must succeed")
	suite.Empty(violations.ProcessViolation)
	return violations
}

func (suite *DefaultPoliciesTestSuite) TestNetworkPolicyFields() {
	testCases := map[string]struct {
		netpolsApplied *augmentedobjs.NetworkPoliciesApplied
		alerts         []*storage.Alert_Violation
	}{
		"Missing Ingress Network Policy": {
			netpolsApplied: &augmentedobjs.NetworkPoliciesApplied{
				HasIngressNetworkPolicy: false,
				HasEgressNetworkPolicy:  true,
			},
			alerts: []*storage.Alert_Violation{
				storage.Alert_Violation_builder{Message: "The deployment is missing Ingress Network Policy.", Type: storage.Alert_Violation_NETWORK_POLICY}.Build(),
			},
		},
		"Missing Egress Network Policy": {
			netpolsApplied: &augmentedobjs.NetworkPoliciesApplied{
				HasIngressNetworkPolicy: true,
				HasEgressNetworkPolicy:  false,
			},
			alerts: []*storage.Alert_Violation{
				storage.Alert_Violation_builder{Message: "The deployment is missing Egress Network Policy.", Type: storage.Alert_Violation_NETWORK_POLICY}.Build(),
			},
		},
		"Both policies missing": {
			netpolsApplied: &augmentedobjs.NetworkPoliciesApplied{
				HasIngressNetworkPolicy: false,
				HasEgressNetworkPolicy:  false,
			},
			alerts: []*storage.Alert_Violation{
				storage.Alert_Violation_builder{Message: "The deployment is missing Ingress Network Policy.", Type: storage.Alert_Violation_NETWORK_POLICY}.Build(),
				storage.Alert_Violation_builder{Message: "The deployment is missing Egress Network Policy.", Type: storage.Alert_Violation_NETWORK_POLICY}.Build(),
			},
		},
		"No alerts": {
			netpolsApplied: &augmentedobjs.NetworkPoliciesApplied{
				HasIngressNetworkPolicy: true,
				HasEgressNetworkPolicy:  true,
			},
			alerts: []*storage.Alert_Violation(nil),
		},
		"No violations on nil augmentedobj": {
			netpolsApplied: nil,
			alerts:         []*storage.Alert_Violation(nil),
		},
		"Policies attached to augmentedobj": {
			netpolsApplied: &augmentedobjs.NetworkPoliciesApplied{
				HasIngressNetworkPolicy: false,
				HasEgressNetworkPolicy:  true,
				Policies: map[string]*storage.NetworkPolicy{
					"ID1": storage.NetworkPolicy_builder{Id: "ID1", Name: "policy1"}.Build(),
				},
			},
			alerts: []*storage.Alert_Violation{
				storage.Alert_Violation_builder{
					Message: "The deployment is missing Ingress Network Policy.",
					Type:    storage.Alert_Violation_NETWORK_POLICY,
					KeyValueAttrs: storage.Alert_Violation_KeyValueAttrs_builder{
						Attrs: []*storage.Alert_Violation_KeyValueAttrs_KeyValueAttr{
							storage.Alert_Violation_KeyValueAttrs_KeyValueAttr_builder{Key: printer.PolicyID, Value: "ID1"}.Build(),
							storage.Alert_Violation_KeyValueAttrs_KeyValueAttr_builder{Key: printer.PolicyName, Value: "policy1"}.Build(),
						},
					}.Build(),
				}.Build(),
			},
		},
	}

	for name, testCase := range testCases {
		suite.Run(name, func() {
			deployment := fixtures.GetDeployment().CloneVT()
			missingIngressPolicy := policyWithSingleKeyValue(fieldnames.HasIngressNetworkPolicy, "false", false)
			missingEgressPolicy := policyWithSingleKeyValue(fieldnames.HasEgressNetworkPolicy, "false", false)

			enhanced := enhancedDeploymentWithNetworkPolicies(
				deployment,
				suite.getImagesForDeployment(deployment),
				testCase.netpolsApplied,
			)

			v1 := suite.getViolations(missingIngressPolicy, enhanced)
			v2 := suite.getViolations(missingEgressPolicy, enhanced)

			allAlerts := append(v1.AlertViolations, v2.AlertViolations...)
			for i, expected := range testCase.alerts {
				suite.Equal(expected.GetType(), allAlerts[i].GetType())
				suite.Equal(expected.GetMessage(), allAlerts[i].GetMessage())
				protoassert.Equal(suite.T(), expected.GetKeyValueAttrs(), allAlerts[i].GetKeyValueAttrs())
				// We do not want to compare time, as the violation timestamp uses now()
				suite.NotNil(allAlerts[i].GetTime())
			}
		})
	}
}

func (suite *DefaultPoliciesTestSuite) TestReadinessProbePolicyCriteria() {
	for _, testCase := range []struct {
		caseName    string
		containers  []*storage.Container
		policyValue string
		alerts      []*storage.Alert_Violation
	}{
		{
			caseName: "Should raise alert since readiness probe is defined.",
			containers: []*storage.Container{
				storage.Container_builder{Name: "container", ReadinessProbe: storage.ReadinessProbe_builder{Defined: true}.Build()}.Build(),
			},
			policyValue: "true",
			alerts: []*storage.Alert_Violation{
				storage.Alert_Violation_builder{Message: "Readiness probe is defined for container 'container'"}.Build(),
			},
		},
		{
			caseName: "Should not raise alert since readiness probe is defined.",
			containers: []*storage.Container{
				storage.Container_builder{Name: "container", ReadinessProbe: storage.ReadinessProbe_builder{Defined: true}.Build()}.Build(),
			},
			policyValue: "false",
			alerts:      nil,
		},
		{
			caseName: "Should not raise alert since readiness probe is not defined.",
			containers: []*storage.Container{
				storage.Container_builder{Name: "container", ReadinessProbe: storage.ReadinessProbe_builder{Defined: false}.Build()}.Build(),
			},
			policyValue: "true",
			alerts:      nil,
		},
		{
			caseName: "Should raise alert since readiness probe is not defined.",
			containers: []*storage.Container{
				storage.Container_builder{Name: "container", ReadinessProbe: storage.ReadinessProbe_builder{Defined: false}.Build()}.Build(),
			},
			policyValue: "false",
			alerts: []*storage.Alert_Violation{
				storage.Alert_Violation_builder{Message: "Readiness probe is not defined for container 'container'"}.Build(),
			},
		},
		{
			caseName: "Should raise alert for both containers.",
			containers: []*storage.Container{
				storage.Container_builder{Name: "container-1", ReadinessProbe: storage.ReadinessProbe_builder{Defined: false}.Build()}.Build(),
				storage.Container_builder{Name: "container-2", ReadinessProbe: storage.ReadinessProbe_builder{Defined: false}.Build()}.Build(),
			},
			policyValue: "false",
			alerts: []*storage.Alert_Violation{
				storage.Alert_Violation_builder{Message: "Readiness probe is not defined for container 'container-1'"}.Build(),
				storage.Alert_Violation_builder{Message: "Readiness probe is not defined for container 'container-2'"}.Build(),
			},
		},
		{
			caseName: "Should raise alert only for container-2.",
			containers: []*storage.Container{
				storage.Container_builder{Name: "container-1", ReadinessProbe: storage.ReadinessProbe_builder{Defined: true}.Build()}.Build(),
				storage.Container_builder{Name: "container-2", ReadinessProbe: storage.ReadinessProbe_builder{Defined: false}.Build()}.Build(),
			},
			policyValue: "false",
			alerts: []*storage.Alert_Violation{
				storage.Alert_Violation_builder{Message: "Readiness probe is not defined for container 'container-2'"}.Build(),
			},
		},
	} {
		suite.Run(testCase.caseName, func() {
			deployment := fixtures.GetDeployment().CloneVT()
			deployment.SetContainers(testCase.containers)
			policy := policyWithSingleKeyValue(fieldnames.ReadinessProbeDefined, testCase.policyValue, false)

			matcher, err := BuildDeploymentMatcher(policy)
			suite.NoError(err, "deployment matcher creation must succeed")
			violations, err := matcher.MatchDeployment(nil, enhancedDeployment(deployment, suite.getImagesForDeployment(deployment)))
			suite.NoError(err, "deployment matcher run must succeed")

			suite.Empty(violations.ProcessViolation)
			protoassert.SlicesEqual(suite.T(), violations.AlertViolations, testCase.alerts)
		})
	}
}

func newIndicator(deployment *storage.Deployment, name, args, execFilePath string) *storage.ProcessIndicator {
	ps := &storage.ProcessSignal{}
	ps.SetName(name)
	ps.SetArgs(args)
	ps.SetExecFilePath(execFilePath)
	pi := &storage.ProcessIndicator{}
	pi.SetId(uuid.NewV4().String())
	pi.SetContainerName(deployment.GetContainers()[0].GetName())
	pi.SetSignal(ps)
	return pi
}

func BenchmarkProcessPolicies(b *testing.B) {
	privilegedDep := fixtures.GetDeployment().CloneVT()
	privilegedDep.SetId("PRIVILEGED")
	images := []*storage.Image{fixtures.GetImage(), fixtures.GetImage()}

	nonPrivilegedDep := fixtures.GetDeployment().CloneVT()
	nonPrivilegedDep.SetId("NOTPRIVILEGED")
	nonPrivilegedDep.GetContainers()[0].GetSecurityContext().SetPrivileged(false)

	const aptGetKey = "apt-get"
	const aptGet2Key = "apt-get2"
	const curlKey = "curl"
	const bashKey = "bash"

	indicators := make(map[string]map[string]*storage.ProcessIndicator)
	for _, dep := range []*storage.Deployment{privilegedDep, nonPrivilegedDep} {
		indicators[dep.GetId()] = map[string]*storage.ProcessIndicator{
			aptGetKey:  newIndicator(dep, "apt-get", "install nginx", "/bin/apt-get"),
			aptGet2Key: newIndicator(dep, "apt-get", "update", "/bin/apt-get"),
			curlKey:    newIndicator(dep, "curl", "https://stackrox.io", "/bin/curl"),
			bashKey:    newIndicator(dep, "bash", "attach.sh", "/bin/bash"),
		}
	}
	processesNotInBaseline := map[string]set.StringSet{
		privilegedDep.GetId():    set.NewStringSet(aptGetKey, aptGet2Key, bashKey),
		nonPrivilegedDep.GetId(): set.NewStringSet(aptGetKey, curlKey, bashKey),
	}

	// Plain groups
	aptGetGroup := policyGroupWithSingleKeyValue(fieldnames.ProcessName, "apt-get", false)
	privilegedGroup := policyGroupWithSingleKeyValue(fieldnames.PrivilegedContainer, "true", false)
	baselineGroup := policyGroupWithSingleKeyValue(fieldnames.UnexpectedProcessExecuted, "true", false)

	for _, testCase := range []struct {
		groups []*storage.PolicyGroup

		// Deployment ids to indicator keys
		expectedMatches        map[string][]string
		expectedProcessMatches map[string][]string
		// Deployment ids to violations
		expectedViolations map[string][]*storage.Alert_Violation
	}{
		{
			groups: []*storage.PolicyGroup{aptGetGroup},
			// only process violation, no alert violation
			expectedMatches: map[string][]string{},
			expectedProcessMatches: map[string][]string{
				privilegedDep.GetId():    {aptGetKey, aptGet2Key},
				nonPrivilegedDep.GetId(): {aptGetKey, aptGet2Key},
			},
		},
		{
			groups: []*storage.PolicyGroup{baselineGroup},
			expectedMatches: map[string][]string{
				privilegedDep.GetId():    {aptGetKey, aptGet2Key, bashKey},
				nonPrivilegedDep.GetId(): {aptGetKey, curlKey, bashKey},
			},
			expectedProcessMatches: map[string][]string{
				privilegedDep.GetId():    {aptGetKey, aptGet2Key, bashKey},
				nonPrivilegedDep.GetId(): {aptGetKey, curlKey, bashKey},
			},
			expectedViolations: map[string][]*storage.Alert_Violation{
				privilegedDep.GetId():    processBaselineMessage(privilegedDep, true, false, "apt-get", "apt-get", "bash"),
				nonPrivilegedDep.GetId(): processBaselineMessage(nonPrivilegedDep, true, false, "apt-get", "bash", "curl"),
			},
		},

		{
			groups: []*storage.PolicyGroup{privilegedGroup},
			expectedMatches: map[string][]string{
				privilegedDep.GetId(): {aptGetKey, aptGet2Key, curlKey, bashKey},
			},
			expectedProcessMatches: map[string][]string{},
			expectedViolations: map[string][]*storage.Alert_Violation{
				privilegedDep.GetId(): processBaselineMessage(privilegedDep, false, true, "apt-get", "apt-get", "curl", "bash"),
			},
		},
		{
			groups: []*storage.PolicyGroup{aptGetGroup, baselineGroup},
			expectedMatches: map[string][]string{
				privilegedDep.GetId():    {aptGetKey, aptGet2Key},
				nonPrivilegedDep.GetId(): {aptGetKey},
			},
			expectedViolations: map[string][]*storage.Alert_Violation{
				privilegedDep.GetId():    processBaselineMessage(privilegedDep, true, false, "apt-get", "apt-get"),
				nonPrivilegedDep.GetId(): processBaselineMessage(nonPrivilegedDep, true, false, "apt-get"),
			},
			expectedProcessMatches: map[string][]string{
				privilegedDep.GetId():    {aptGetKey, aptGet2Key},
				nonPrivilegedDep.GetId(): {aptGetKey},
			},
		},
		{
			groups: []*storage.PolicyGroup{aptGetGroup, privilegedGroup},
			expectedMatches: map[string][]string{
				privilegedDep.GetId(): {aptGetKey, aptGet2Key},
			},
			expectedViolations: map[string][]*storage.Alert_Violation{
				privilegedDep.GetId(): processBaselineMessage(privilegedDep, false, true, "apt-get", "apt-get"),
			},
			expectedProcessMatches: map[string][]string{
				privilegedDep.GetId(): {aptGetKey, aptGet2Key},
			},
		},
		{
			groups: []*storage.PolicyGroup{privilegedGroup, baselineGroup},
			expectedMatches: map[string][]string{
				privilegedDep.GetId(): {aptGetKey, aptGet2Key, bashKey},
			},
			expectedViolations: map[string][]*storage.Alert_Violation{
				privilegedDep.GetId(): processBaselineMessage(privilegedDep, true, true, "apt-get", "apt-get", "bash"),
			},
			expectedProcessMatches: map[string][]string{
				privilegedDep.GetId(): {aptGetKey, aptGet2Key, bashKey},
			},
		},
		{
			groups: []*storage.PolicyGroup{aptGetGroup, privilegedGroup, baselineGroup},
			expectedMatches: map[string][]string{
				privilegedDep.GetId(): {aptGetKey, aptGet2Key},
			},
			expectedViolations: map[string][]*storage.Alert_Violation{
				privilegedDep.GetId(): processBaselineMessage(privilegedDep, true, true, "apt-get", "apt-get"),
			},
			expectedProcessMatches: map[string][]string{
				privilegedDep.GetId(): {aptGetKey, aptGet2Key},
			},
		},
	} {
		c := testCase
		b.Run(fmt.Sprintf("%+v", c.groups), func(b *testing.B) {
			policy := policyWithGroups(storage.EventSource_DEPLOYMENT_EVENT, c.groups...)
			m, err := BuildDeploymentWithProcessMatcher(policy)
			require.NoError(b, err)

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				for _, dep := range []*storage.Deployment{privilegedDep, nonPrivilegedDep} {
					for _, key := range []string{aptGetKey, aptGet2Key, curlKey, bashKey} {
						_, err := m.MatchDeploymentWithProcess(nil, enhancedDeployment(dep, images), indicators[dep.GetId()][key], processesNotInBaseline[dep.GetId()].Contains(key))
						require.NoError(b, err)
					}
				}
			}
		})
	}

	policy := policyWithGroups(storage.EventSource_DEPLOYMENT_EVENT, aptGetGroup, privilegedGroup, baselineGroup)
	m, err := BuildDeploymentWithProcessMatcher(policy)
	require.NoError(b, err)
	for _, dep := range []*storage.Deployment{privilegedDep, nonPrivilegedDep} {
		for _, key := range []string{aptGetKey, aptGet2Key, curlKey, bashKey} {
			indicator := indicators[dep.GetId()][key]
			notInBaseline := processesNotInBaseline[dep.GetId()].Contains(key)
			b.Run(fmt.Sprintf("benchmark caching: %s/%s", dep.GetId(), key), func(b *testing.B) {
				var resNoCaching Violations
				b.Run("no caching", func(b *testing.B) {
					for i := 0; i < b.N; i++ {
						var err error
						resNoCaching, err = m.MatchDeploymentWithProcess(nil, enhancedDeployment(privilegedDep, images), indicator, notInBaseline)
						require.NoError(b, err)
					}
				})

				var resWithCaching Violations
				b.Run("with caching", func(b *testing.B) {
					var cache CacheReceptacle
					for i := 0; i < b.N; i++ {
						var err error
						resWithCaching, err = m.MatchDeploymentWithProcess(&cache, enhancedDeployment(privilegedDep, images), indicator, notInBaseline)
						require.NoError(b, err)
					}
				})
				assertViolations(b, resNoCaching, resWithCaching)
			})
		}
	}

}

func podExecViolationMsg(pod, container, command string) *storage.Alert_Violation {
	if command == "" {
		return storage.Alert_Violation_builder{
			Message: fmt.Sprintf("Kubernetes API received exec request into pod '%s' container '%s'", pod, container),
			Type:    storage.Alert_Violation_K8S_EVENT,
			KeyValueAttrs: storage.Alert_Violation_KeyValueAttrs_builder{
				Attrs: []*storage.Alert_Violation_KeyValueAttrs_KeyValueAttr{
					storage.Alert_Violation_KeyValueAttrs_KeyValueAttr_builder{Key: "pod", Value: pod}.Build(),
					storage.Alert_Violation_KeyValueAttrs_KeyValueAttr_builder{Key: "container", Value: container}.Build(),
				},
			}.Build(),
		}.Build()
	}

	return storage.Alert_Violation_builder{
		Message: fmt.Sprintf("Kubernetes API received exec '%s' request into pod '%s' container '%s'",
			command, pod, container),
		Type: storage.Alert_Violation_K8S_EVENT,
		KeyValueAttrs: storage.Alert_Violation_KeyValueAttrs_builder{
			Attrs: []*storage.Alert_Violation_KeyValueAttrs_KeyValueAttr{
				storage.Alert_Violation_KeyValueAttrs_KeyValueAttr_builder{Key: "pod", Value: pod}.Build(),
				storage.Alert_Violation_KeyValueAttrs_KeyValueAttr_builder{Key: "container", Value: container}.Build(),
				storage.Alert_Violation_KeyValueAttrs_KeyValueAttr_builder{Key: "commands", Value: command}.Build(),
			},
		}.Build(),
	}.Build()
}

func podPortForwardViolationMsg(pod string, port int) *storage.Alert_Violation {
	return storage.Alert_Violation_builder{
		Message: fmt.Sprintf("Kubernetes API received port forward request to pod '%s' ports '%s'", pod, strconv.Itoa(port)),
		Type:    storage.Alert_Violation_K8S_EVENT,
		KeyValueAttrs: storage.Alert_Violation_KeyValueAttrs_builder{
			Attrs: []*storage.Alert_Violation_KeyValueAttrs_KeyValueAttr{
				storage.Alert_Violation_KeyValueAttrs_KeyValueAttr_builder{Key: "pod", Value: pod}.Build(),
				storage.Alert_Violation_KeyValueAttrs_KeyValueAttr_builder{Key: "ports", Value: strconv.Itoa(port)}.Build(),
			},
		}.Build(),
	}.Build()
}

func podExecEvent(pod, container, command string) *storage.KubernetesEvent {
	return storage.KubernetesEvent_builder{
		Object: storage.KubernetesEvent_Object_builder{
			Name:     pod,
			Resource: storage.KubernetesEvent_Object_PODS_EXEC,
		}.Build(),
		ApiVerb: storage.KubernetesEvent_CREATE,
		PodExecArgs: storage.KubernetesEvent_PodExecArgs_builder{
			Container: container,
			Commands:  []string{command},
		}.Build(),
	}.Build()
}

func podPortForwardEvent(pod string, port int32) *storage.KubernetesEvent {
	return storage.KubernetesEvent_builder{
		Object: storage.KubernetesEvent_Object_builder{
			Name:     pod,
			Resource: storage.KubernetesEvent_Object_PODS_PORTFORWARD,
		}.Build(),
		ApiVerb: storage.KubernetesEvent_CREATE,
		PodPortForwardArgs: storage.KubernetesEvent_PodPortForwardArgs_builder{
			Ports: []int32{port},
		}.Build(),
	}.Build()
}

func assertViolations(t testing.TB, expected, actual Violations) {
	t.Helper()
	protoassert.Equal(t, expected.ProcessViolation, actual.ProcessViolation)
	protoassert.SlicesEqual(t, expected.AlertViolations, actual.AlertViolations)
}
