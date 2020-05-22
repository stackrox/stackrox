package booleanpolicy

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/gogo/protobuf/proto"
	gogoTypes "github.com/gogo/protobuf/types"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/image/policies"
	"github.com/stackrox/rox/pkg/booleanpolicy/fieldnames"
	"github.com/stackrox/rox/pkg/defaults"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/images/types"
	policyUtils "github.com/stackrox/rox/pkg/policies"
	"github.com/stackrox/rox/pkg/protoconv"
	"github.com/stackrox/rox/pkg/readable"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/sliceutils"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

const (
	writableHostMountPolicyName = "Writeable Host Mount"
)

func changeName(p *storage.Policy, newName string) *storage.Policy {
	p.Name = newName
	return p
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

	envIsolator *testutils.EnvIsolator
}

func (suite *DefaultPoliciesTestSuite) SetupSuite() {
	defaults.PoliciesPath = policies.Directory()

	defaultPolicies, err := defaults.Policies()
	suite.Require().NoError(err)

	suite.defaultPolicies = make(map[string]*storage.Policy, len(defaultPolicies))
	for _, p := range defaultPolicies {
		suite.defaultPolicies[p.GetName()] = p
	}

	suite.customPolicies = make(map[string]*storage.Policy)
	for _, customPolicy := range []*storage.Policy{
		changeName(policyWithSingleKeyValue(fieldnames.WritableHostMount, "true", false), writableHostMountPolicyName),
	} {
		suite.customPolicies[customPolicy.GetName()] = customPolicy
	}

	suite.envIsolator = testutils.NewEnvIsolator(suite.T())
}

func (suite *DefaultPoliciesTestSuite) SetupTest() {
	suite.deployments = make(map[string]*storage.Deployment)
	suite.images = make(map[string]*storage.Image)
	suite.deploymentsToImages = make(map[string][]*storage.Image)
	suite.deploymentsToIndicators = make(map[string][]*storage.ProcessIndicator)
}

func (suite *DefaultPoliciesTestSuite) imageIDFromDep(deployment *storage.Deployment) string {
	suite.Require().Len(deployment.GetContainers(), 1, "This function only supports deployments with exactly one container")
	id := deployment.GetContainers()[0].GetImage().GetId()
	suite.NotEmpty(id, "Deployment '%s' had no image id", proto.MarshalTextString(deployment))
	return id
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

func deploymentWithImageAnyID(img *storage.Image) *storage.Deployment {
	return deploymentWithImage(uuid.NewV4().String(), img)
}

func deploymentWithImage(id string, img *storage.Image) *storage.Deployment {
	remoteSplit := strings.Split(img.GetName().GetFullName(), "/")
	alphaOnly := regexp.MustCompile("[^A-Za-z]+")
	containerName := alphaOnly.ReplaceAllString(remoteSplit[len(remoteSplit)-1], "")
	return &storage.Deployment{
		Id:         id,
		Containers: []*storage.Container{{Name: containerName, Image: types.ToContainerImage(img)}},
	}
}

func (suite *DefaultPoliciesTestSuite) addIndicator(deploymentID, name, args, path string, lineage []string, uid uint32) *storage.ProcessIndicator {
	deployment := suite.deployments[deploymentID]
	if len(deployment.GetContainers()) == 0 {
		deployment.Containers = []*storage.Container{{Name: uuid.NewV4().String()}}
	}
	indicator := &storage.ProcessIndicator{
		Id:            uuid.NewV4().String(),
		DeploymentId:  deploymentID,
		ContainerName: deployment.GetContainers()[0].GetName(),
		Signal: &storage.ProcessSignal{
			Name:         name,
			Args:         args,
			ExecFilePath: path,
			Time:         gogoTypes.TimestampNow(),
			Lineage:      lineage,
			Uid:          uid,
		},
	}
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
	shouldNotMatch            map[string]struct{}
	sampleViolationForMatched string
}

func (suite *DefaultPoliciesTestSuite) getImagesForDeployment(deployment *storage.Deployment) []*storage.Image {
	images := suite.deploymentsToImages[deployment.GetId()]
	if len(images) == 0 {
		return make([]*storage.Image, len(deployment.GetContainers()))
	}
	suite.Equal(len(deployment.GetContainers()), len(images))
	return images
}

func (suite *DefaultPoliciesTestSuite) TestDefaultPolicies() {
	fixtureDep := fixtures.GetDeployment()
	fixturesImages := fixtures.DeploymentImages()

	suite.addDepAndImages(fixtureDep, fixturesImages...)

	nginx110 := &storage.Image{
		Id: "SHANGINX110",
		Name: &storage.ImageName{
			Registry: "docker.io",
			Remote:   "library/nginx",
			Tag:      "1.10",
			FullName: "docker.io/library/nginx:1.10",
		},
	}

	nginx110Dep := deploymentWithImage("nginx110", nginx110)
	suite.addDepAndImages(nginx110Dep, nginx110)

	oldScannedTime := time.Now().Add(-31 * 24 * time.Hour)

	oldScannedImage := &storage.Image{
		Id: "SHAOLDSCANNED",
		Name: &storage.ImageName{
			FullName: "docker.io/stackrox/old-scanned-image:0.1",
		},
		Scan: &storage.ImageScan{
			ScanTime: protoconv.ConvertTimeToTimestamp(oldScannedTime),
		},
	}
	oldScannedDep := deploymentWithImage("oldscanned", oldScannedImage)
	suite.addDepAndImages(oldScannedDep, oldScannedImage)

	addDockerFileImg := imageWithLayers([]*storage.ImageLayer{
		{
			Instruction: "ADD",
			Value:       "deploy.sh",
		},
		{
			Instruction: "RUN",
			Value:       "deploy.sh",
		},
	})
	addDockerFileDep := deploymentWithImageAnyID(addDockerFileImg)
	suite.addDepAndImages(addDockerFileDep, addDockerFileImg)

	imagePort22Image := imageWithLayers([]*storage.ImageLayer{
		{
			Instruction: "EXPOSE",
			Value:       "22/tcp",
		},
	})
	imagePort22Dep := deploymentWithImageAnyID(imagePort22Image)
	suite.addDepAndImages(imagePort22Dep, imagePort22Image)

	insecureCMDImage := imageWithLayers([]*storage.ImageLayer{
		{
			Instruction: "CMD",
			Value:       "do an insecure thing",
		},
	})

	insecureCMDDep := deploymentWithImageAnyID(insecureCMDImage)
	suite.addDepAndImages(insecureCMDDep, insecureCMDImage)

	runSecretsImage := imageWithLayers([]*storage.ImageLayer{
		{
			Instruction: "VOLUME",
			Value:       "/run/secrets",
		},
	})
	runSecretsDep := deploymentWithImageAnyID(runSecretsImage)
	suite.addDepAndImages(runSecretsDep, runSecretsImage)

	oldImageCreationTime := time.Now().Add(-100 * 24 * time.Hour)
	oldCreatedImage := &storage.Image{
		Id: "SHA:OLDCREATEDIMAGE",
		Name: &storage.ImageName{
			FullName: "docker.io/stackrox/old-image:0.1",
		},
		Metadata: &storage.ImageMetadata{
			V1: &storage.V1Metadata{
				Created: protoconv.ConvertTimeToTimestamp(oldImageCreationTime),
			},
		},
	}
	oldImageDep := deploymentWithImage("oldimagedep", oldCreatedImage)
	suite.addDepAndImages(oldImageDep, oldCreatedImage)

	apkImage := imageWithComponents([]*storage.EmbeddedImageScanComponent{
		{Name: "apk", Version: "1.2"},
		{Name: "asfa", Version: "1.5"},
	})
	apkDep := deploymentWithImageAnyID(apkImage)
	suite.addDepAndImages(apkDep, apkImage)

	curlImage := imageWithComponents([]*storage.EmbeddedImageScanComponent{
		{Name: "curl", Version: "1.3"},
		{Name: "curlwithextra", Version: "0.9"},
	})
	curlDep := deploymentWithImageAnyID(curlImage)
	suite.addDepAndImages(curlDep, curlImage)

	componentDeps := make(map[string]*storage.Deployment)
	for _, component := range []string{"apt", "dnf", "wget"} {
		img := imageWithComponents([]*storage.EmbeddedImageScanComponent{
			{Name: component},
		})
		dep := deploymentWithImageAnyID(img)
		suite.addDepAndImages(dep, img)
		componentDeps[component] = dep
	}

	heartbleedDep := &storage.Deployment{
		Id: "HEARTBLEEDDEPID",
		Containers: []*storage.Container{
			{
				Name:            "nginx",
				SecurityContext: &storage.SecurityContext{Privileged: true},
				Image:           &storage.ContainerImage{Id: "HEARTBLEEDDEPSHA"},
			},
		},
	}
	suite.addDepAndImages(heartbleedDep, &storage.Image{
		Id:   "HEARTBLEEDDEPSHA",
		Name: &storage.ImageName{FullName: "heartbleed"},
		Scan: &storage.ImageScan{
			Components: []*storage.EmbeddedImageScanComponent{
				{Name: "heartbleed", Version: "1.2", Vulns: []*storage.EmbeddedVulnerability{
					{Cve: "CVE-2014-0160", Link: "https://heartbleed", Cvss: 6, SetFixedBy: &storage.EmbeddedVulnerability_FixedBy{FixedBy: "v1.2"}},
				}},
			},
		},
	})

	requiredImageLabel := &storage.Deployment{
		Id: "requiredImageLabel",
		Containers: []*storage.Container{
			{
				Name:  "REQUIREDIMAGELABEL",
				Image: &storage.ContainerImage{Id: "requiredImageLabelImage"},
			},
		},
	}
	suite.addDepAndImages(requiredImageLabel, &storage.Image{
		Id: "requiredImageLabelImage",
		Name: &storage.ImageName{
			FullName: "docker.io/stackrox/required-image:0.1",
		},
		Metadata: &storage.ImageMetadata{
			V1: &storage.V1Metadata{
				Labels: map[string]string{
					"required-label": "required-value",
				},
			},
		},
	})

	shellshockImage := imageWithComponents([]*storage.EmbeddedImageScanComponent{
		{Name: "shellshock", Version: "1.2", Vulns: []*storage.EmbeddedVulnerability{
			{Cve: "CVE-2014-6271", Link: "https://shellshock", Cvss: 6},
			{Cve: "CVE-ARBITRARY", Link: "https://notshellshock"},
		}},
	})
	shellshockDep := deploymentWithImageAnyID(shellshockImage)
	suite.addDepAndImages(shellshockDep, shellshockImage)

	suppressedShellshockImage := imageWithComponents([]*storage.EmbeddedImageScanComponent{
		{Name: "shellshock", Version: "1.2", Vulns: []*storage.EmbeddedVulnerability{
			{Cve: "CVE-2014-6271", Link: "https://shellshock", Cvss: 6, Suppressed: true},
			{Cve: "CVE-ARBITRARY", Link: "https://notshellshock"},
		}},
	})
	suppressedShellShockDep := deploymentWithImageAnyID(suppressedShellshockImage)
	suite.addDepAndImages(suppressedShellShockDep, suppressedShellshockImage)

	strutsImage := imageWithComponents([]*storage.EmbeddedImageScanComponent{
		{Name: "struts", Version: "1.2", Vulns: []*storage.EmbeddedVulnerability{
			{Cve: "CVE-2017-5638", Link: "https://struts", Cvss: 8, SetFixedBy: &storage.EmbeddedVulnerability_FixedBy{FixedBy: "v1.3"}},
		}},
		{Name: "OTHER", Version: "1.3", Vulns: []*storage.EmbeddedVulnerability{
			{Cve: "CVE-1223-451", Link: "https://cvefake"},
		}},
	})
	strutsDep := deploymentWithImageAnyID(strutsImage)
	suite.addDepAndImages(strutsDep, strutsImage)

	strutsImageSuppressed := imageWithComponents([]*storage.EmbeddedImageScanComponent{
		{Name: "struts", Version: "1.2", Vulns: []*storage.EmbeddedVulnerability{
			{Cve: "CVE-2017-5638", Link: "https://struts", Suppressed: true, Cvss: 8, SetFixedBy: &storage.EmbeddedVulnerability_FixedBy{FixedBy: "v1.3"}},
		}},
		{Name: "OTHER", Version: "1.3", Vulns: []*storage.EmbeddedVulnerability{
			{Cve: "CVE-1223-451", Link: "https://cvefake"},
		}},
	})
	strutsDepSuppressed := deploymentWithImageAnyID(strutsImageSuppressed)
	suite.addDepAndImages(strutsDepSuppressed, strutsImageSuppressed)

	depWithNonSeriousVulnsImage := imageWithComponents([]*storage.EmbeddedImageScanComponent{
		{Name: "NOSERIOUS", Version: "2.3", Vulns: []*storage.EmbeddedVulnerability{
			{Cve: "CVE-1234-5678", Link: "https://abcdefgh"},
			{Cve: "CVE-5678-1234", Link: "https://lmnopqrst"},
		}},
	})
	depWithNonSeriousVulns := deploymentWithImageAnyID(depWithNonSeriousVulnsImage)
	suite.addDepAndImages(depWithNonSeriousVulns, depWithNonSeriousVulnsImage)

	dockerSockDep := &storage.Deployment{
		Id: "DOCKERSOCDEP",
		Containers: []*storage.Container{
			{
				Name: "dockersock",
				Volumes: []*storage.Volume{
					{Source: "/var/run/docker.sock", Name: "DOCKERSOCK", Type: "HostPath", ReadOnly: true},
					{Source: "NOTDOCKERSOCK"},
				}},
		},
	}
	suite.addDepAndImages(dockerSockDep)

	containerPort22Dep := &storage.Deployment{
		Id: "CONTAINERPORT22DEP",
		Ports: []*storage.PortConfig{
			{Protocol: "TCP", ContainerPort: 22},
			{Protocol: "UDP", ContainerPort: 4125},
		},
	}
	suite.addDepAndImages(containerPort22Dep)

	secretEnvDep := &storage.Deployment{
		Id: "SECRETENVDEP",
		Containers: []*storage.Container{
			{
				Name: "secretenv",
				Config: &storage.ContainerConfig{
					Env: []*storage.ContainerConfig_EnvironmentConfig{
						{Key: "THIS_IS_SECRET_VAR", Value: "stealthmode", EnvVarSource: storage.ContainerConfig_EnvironmentConfig_RAW},
						{Key: "HOME", Value: "/home/stackrox"},
					},
				}},
		},
	}
	suite.addDepAndImages(secretEnvDep)

	secretEnvSrcUnsetDep := &storage.Deployment{
		Id: "SECRETENVSRCUNSETDEP",
		Containers: []*storage.Container{
			{
				Name: "secretenvsrcunset",
				Config: &storage.ContainerConfig{
					Env: []*storage.ContainerConfig_EnvironmentConfig{
						{Key: "THIS_IS_SECRET_VAR", Value: "stealthmode"},
					},
				}},
		},
	}
	suite.addDepAndImages(secretEnvSrcUnsetDep)

	secretKeyRefDep := &storage.Deployment{
		Id: "SECRETKEYREFDEP",
		Containers: []*storage.Container{
			{Config: &storage.ContainerConfig{
				Env: []*storage.ContainerConfig_EnvironmentConfig{
					{Key: "THIS_IS_SECRET_VAR", EnvVarSource: storage.ContainerConfig_EnvironmentConfig_SECRET_KEY},
					{Key: "HOME", Value: "/home/stackrox"},
				},
			}},
		},
	}
	suite.addDepAndImages(secretKeyRefDep)

	// Fake deployment that shouldn't match anything, just to make sure
	// that none of our queries will accidentally match it.
	suite.addDepAndImages(&storage.Deployment{Id: "FAKEID", Name: "FAKENAME"})

	depWithGoodEmailAnnotation := &storage.Deployment{
		Id: "GOODEMAILDEPID",
		Annotations: map[string]string{
			"email": "vv@stackrox.com",
		},
	}
	suite.addDepAndImages(depWithGoodEmailAnnotation)

	depWithOwnerAnnotation := &storage.Deployment{
		Id: "OWNERANNOTATIONDEP",
		Annotations: map[string]string{
			"owner": "IOWNTHIS",
			"blah":  "Blah",
		},
	}
	suite.addDepAndImages(depWithOwnerAnnotation)

	depWitharbitraryAnnotations := &storage.Deployment{
		Id: "ARBITRARYANNOTATIONDEPID",
		Annotations: map[string]string{
			"emailnot": "vv@stackrox.com",
			"notemail": "vv@stackrox.com",
			"ownernot": "vv",
			"nowner":   "vv",
		},
	}
	suite.addDepAndImages(depWitharbitraryAnnotations)

	depWithBadEmailAnnotation := &storage.Deployment{
		Id: "BADEMAILDEPID",
		Annotations: map[string]string{
			"email": "NOTANEMAIL",
		},
	}
	suite.addDepAndImages(depWithBadEmailAnnotation)

	sysAdminDep := &storage.Deployment{
		Id: "SYSADMINDEPID",
		Containers: []*storage.Container{
			{
				Name: "cap-sys",
				SecurityContext: &storage.SecurityContext{
					AddCapabilities: []string{"CAP_SYS_ADMIN"},
				},
			},
		},
	}
	suite.addDepAndImages(sysAdminDep)

	depWithAllResourceLimitsRequestsSpecified := &storage.Deployment{
		Id: "ALLRESOURCESANDLIMITSDEP",
		Containers: []*storage.Container{
			{Resources: &storage.Resources{
				CpuCoresRequest: 0.1,
				CpuCoresLimit:   0.3,
				MemoryMbLimit:   100,
				MemoryMbRequest: 1251,
			}},
		},
	}
	suite.addDepAndImages(depWithAllResourceLimitsRequestsSpecified)

	depWithEnforcementBypassAnnotation := &storage.Deployment{
		Id: "ENFORCEMENTBYPASS",
		Annotations: map[string]string{
			"admission.stackrox.io/break-glass": "ticket-1234",
		},
	}
	suite.addDepAndImages(depWithEnforcementBypassAnnotation)

	hostMountDep := &storage.Deployment{
		Id: "HOSTMOUNT",
		Containers: []*storage.Container{
			{
				Name: "hostmount",
				Volumes: []*storage.Volume{
					{Source: "/etc/passwd", Name: "HOSTMOUNT", Type: "HostPath"},
					{Source: "/var/lib/kubelet", Name: "KUBELET", Type: "HostPath", ReadOnly: true},
				}},
		},
	}
	suite.addDepAndImages(hostMountDep)

	// Index processes
	bashLineage := []string{"/bin/bash"}
	fixtureDepAptIndicator := suite.addIndicator(fixtureDep.GetId(), "apt", "", "/usr/bin/apt", bashLineage, 1)
	sysAdminDepAptIndicator := suite.addIndicator(sysAdminDep.GetId(), "apt", "install blah", "/usr/bin/apt", bashLineage, 1)

	kubeletIndicator := suite.addIndicator(containerPort22Dep.GetId(), "curl", "https://12.13.14.15:10250", "/bin/curl", bashLineage, 1)
	kubeletIndicator2 := suite.addIndicator(containerPort22Dep.GetId(), "wget", "https://heapster.kube-system/metrics", "/bin/wget", bashLineage, 1)

	nmapIndicatorfixtureDep1 := suite.addIndicator(fixtureDep.GetId(), "nmap", "blah", "/usr/bin/nmap", bashLineage, 1)
	nmapIndicatorfixtureDep2 := suite.addIndicator(fixtureDep.GetId(), "nmap", "blah2", "/usr/bin/nmap", bashLineage, 1)
	nmapIndicatorNginx110Dep := suite.addIndicator(nginx110Dep.GetId(), "nmap", "", "/usr/bin/nmap", bashLineage, 1)

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
			policyName: "DockerHub NGINX 1.10",
			expectedViolations: map[string][]*storage.Alert_Violation{
				fixtureDep.GetId(): {
					{
						Message: "Container 'nginx110container' has image with registry 'docker.io', remote 'library/nginx', and tag '1.10'",
					},
				},
				nginx110Dep.GetId(): {
					{
						Message: "Container 'nginx' has image with registry 'docker.io', remote 'library/nginx', and tag '1.10'",
					},
				},
			},
		},
		{
			policyName: "Alpine Linux Package Manager (apk) in Image",
			expectedViolations: map[string][]*storage.Alert_Violation{
				apkDep.GetId(): {
					{
						Message: "Container 'ASFASF' includes component apk 1.2",
					},
				},
			},
		},
		{
			policyName: "Ubuntu Package Manager in Image",
			expectedViolations: map[string][]*storage.Alert_Violation{
				componentDeps["apt"].GetId(): {
					{
						Message: "Container 'ASFASF' includes component apt",
					},
				},
			},
		},
		{
			policyName: "Curl in Image",
			expectedViolations: map[string][]*storage.Alert_Violation{
				curlDep.GetId(): {
					{
						Message: "Container 'ASFASF' includes component curl 1.3",
					},
				},
			},
		},
		{
			policyName: "Red Hat Package Manager in Image",
			expectedViolations: map[string][]*storage.Alert_Violation{
				componentDeps["dnf"].GetId(): {
					{
						Message: "Container 'ASFASF' includes component dnf",
					},
				},
			},
		},
		{
			policyName: "Wget in Image",
			expectedViolations: map[string][]*storage.Alert_Violation{
				componentDeps["wget"].GetId(): {
					{
						Message: "Container 'ASFASF' includes component wget",
					},
				},
			},
		},
		{
			policyName: "Mount Docker Socket",
			expectedViolations: map[string][]*storage.Alert_Violation{
				dockerSockDep.GetId(): {
					{
						Message: "Read-only volume 'DOCKERSOCK' has source '/var/run/docker.sock' and type 'HostPath'",
					},
				},
			},
		},
		{
			policyName: "90-Day Image Age",
			expectedViolations: map[string][]*storage.Alert_Violation{
				oldImageDep.GetId(): {
					{
						Message: fmt.Sprintf("Container 'oldimage' has image with time of creation %s", readable.Time(oldImageCreationTime)),
					},
				},
			},
		},
		{
			policyName: "30-Day Scan Age",
			expectedViolations: map[string][]*storage.Alert_Violation{
				oldScannedDep.GetId(): {
					{
						Message: fmt.Sprintf("Container 'oldscannedimage' has image with time of last scan %s", readable.Time(oldScannedTime)),
					},
				},
			},
		},
		{
			policyName: "Secure Shell (ssh) Port Exposed in Image",
			expectedViolations: map[string][]*storage.Alert_Violation{
				imagePort22Dep.GetId(): {
					{
						Message: "Dockerfile line 'EXPOSE 22/tcp' present in container 'ASFASF'",
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
						Message: "Container 'nginx' using read-write root filesystem",
					},
				},
				fixtureDep.GetId(): {
					{
						Message: "Container 'nginx110container' using read-write root filesystem",
					},
				},
				sysAdminDep.GetId(): {
					{
						Message: "Container 'cap-sys' using read-write root filesystem",
					},
				},
			},
		},
		{
			policyName: "Insecure specified in CMD",
			expectedViolations: map[string][]*storage.Alert_Violation{
				insecureCMDDep.GetId(): {
					{
						Message: "Dockerfile line 'CMD do an insecure thing' present in container 'ASFASF'",
					},
				},
			},
		},
		{
			policyName: "Improper Usage of Orchestrator Secrets Volume",
			expectedViolations: map[string][]*storage.Alert_Violation{
				runSecretsDep.GetId(): {
					{
						Message: "Dockerfile line 'VOLUME /run/secrets' present in container 'ASFASF'",
					},
				},
			},
		},
		{
			policyName: "Images with no scans",
			shouldNotMatch: map[string]struct{}{
				// These deployments have scans on their images.
				fixtureDep.GetId():              {},
				oldScannedDep.GetId():           {},
				heartbleedDep.GetId():           {},
				apkDep.GetId():                  {},
				curlDep.GetId():                 {},
				componentDeps["apt"].GetId():    {},
				componentDeps["dnf"].GetId():    {},
				componentDeps["wget"].GetId():   {},
				shellshockDep.GetId():           {},
				suppressedShellShockDep.GetId(): {},
				strutsDep.GetId():               {},
				strutsDepSuppressed.GetId():     {},
				depWithNonSeriousVulns.GetId():  {},
				// The rest of the deployments have no images!
				"FAKEID":                                          {},
				containerPort22Dep.GetId():                        {},
				dockerSockDep.GetId():                             {},
				secretEnvDep.GetId():                              {},
				secretEnvSrcUnsetDep.GetId():                      {},
				secretKeyRefDep.GetId():                           {},
				depWithOwnerAnnotation.GetId():                    {},
				depWithGoodEmailAnnotation.GetId():                {},
				depWithBadEmailAnnotation.GetId():                 {},
				depWitharbitraryAnnotations.GetId():               {},
				sysAdminDep.GetId():                               {},
				depWithAllResourceLimitsRequestsSpecified.GetId(): {},
				depWithEnforcementBypassAnnotation.GetId():        {},
				hostMountDep.GetId():                              {},
			},
			sampleViolationForMatched: "Image in container '%s' has not been scanned",
		},
		{
			policyName:                "Required Label: Email",
			shouldNotMatch:            map[string]struct{}{fixtureDep.GetId(): {}},
			sampleViolationForMatched: "Required label not found (key = 'email', value = '[a-zA-Z0-9_.+-]+@[a-zA-Z0-9-]+\\.[a-zA-Z0-9-.]+')",
		},
		{
			policyName:                "Required Annotation: Email",
			shouldNotMatch:            map[string]struct{}{depWithGoodEmailAnnotation.GetId(): {}},
			sampleViolationForMatched: "Required annotation not found (key = 'email', value = '[a-zA-Z0-9_.+-]+@[a-zA-Z0-9-]+\\.[a-zA-Z0-9-.]+')",
		},
		{
			policyName:                "Required Label: Owner",
			shouldNotMatch:            map[string]struct{}{fixtureDep.GetId(): {}},
			sampleViolationForMatched: "Required label not found (key = 'owner', value = '.+')",
		},
		{
			policyName:                "Required Annotation: Owner",
			shouldNotMatch:            map[string]struct{}{depWithOwnerAnnotation.GetId(): {}},
			sampleViolationForMatched: "Required annotation not found (key = 'owner', value = '.+')",
		},
		{
			policyName: "CAP_SYS_ADMIN capability added",
			expectedViolations: map[string][]*storage.Alert_Violation{
				sysAdminDep.GetId(): {
					{
						Message: "Container cap-sys adds capability CAP_SYS_ADMIN",
					},
				},
			},
		},
		{
			policyName: "Shellshock: Multiple CVEs",
			expectedViolations: map[string][]*storage.Alert_Violation{
				shellshockDep.GetId(): {
					{
						Message: "CVE-2014-6271 (CVSS 6) found in component shellshock-1.2 in container 'ASFASF'",
					},
				},
				fixtureDep.GetId(): {
					{
						Message: "CVE-2014-6271 (CVSS 5) found in component name-1.2.3.4 in container 'supervulnerable'",
					},
				},
			},
		},
		{
			policyName: "Apache Struts: CVE-2017-5638",
			expectedViolations: map[string][]*storage.Alert_Violation{
				strutsDep.GetId(): {
					{
						Message: "CVE-2017-5638 (CVSS 8) found in component struts-1.2 in container 'ASFASF'",
					},
				},
			},
		},
		{
			policyName: "Heartbleed: CVE-2014-0160",
			expectedViolations: map[string][]*storage.Alert_Violation{
				heartbleedDep.GetId(): {
					{
						Message: "CVE-2014-0160 (CVSS 6) found in component heartbleed-1.2 in container 'nginx'",
					},
				},
			},
		},
		{
			policyName: "No resource requests or limits specified",
			expectedViolations: map[string][]*storage.Alert_Violation{
				fixtureDep.GetId(): {
					{Message: "CPU limit of 0 cores"},
					{Message: "Memory limit of 0 MB"},
					{Message: "Memory request of 0 MB"},
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
						Message: "Environment variable 'THIS_IS_SECRET_VAR' is present and references a secret key",
					},
				},
			},
		},
		{
			policyName: "Fixable CVSS >= 6 and Privileged",
			expectedViolations: map[string][]*storage.Alert_Violation{
				heartbleedDep.GetId(): {
					{
						Message: "Fixable CVE-2014-0160 (CVSS 6) found in component heartbleed-1.2 in container 'nginx', resolved by version v1.2",
					},
					{
						Message: "Container 'nginx' is privileged",
					},
				},
			},
		},
		{
			policyName: "Fixable CVSS >= 7",
			expectedViolations: map[string][]*storage.Alert_Violation{
				strutsDep.GetId(): {
					{
						Message: "Fixable CVE-2017-5638 (CVSS 8) found in component struts-1.2 in container 'ASFASF', resolved by version v1.3",
					},
				},
			},
		},
		{
			policyName: "ADD Command used instead of COPY",
			expectedViolations: map[string][]*storage.Alert_Violation{
				addDockerFileDep.GetId(): {
					{
						Message: "Dockerfile line 'ADD deploy.sh' present in container 'ASFASF'",
					},
				},
				fixtureDep.GetId(): {
					{
						Message: "Dockerfile line 'ADD FILE:blah' present in container 'nginx110container'",
					},
					{
						Message: "Dockerfile line 'ADD file:4eedf861fb567fffb2694b65ebd...' present in container 'supervulnerable'",
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
				containerPort22Dep.GetId(): {kubeletIndicator, kubeletIndicator2},
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
			policyName: "Emergency Deployment Annotation",
			expectedViolations: map[string][]*storage.Alert_Violation{
				depWithEnforcementBypassAnnotation.GetId(): {
					// TODO(rc) with map values
					{Message: "Deployment includes no annotations"},
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

			convertedP, err := CloneAndEnsureConverted(p)
			require.NoError(t, err)
			m, err := BuildDeploymentMatcher(convertedP)
			require.NoError(t, err)

			if c.expectedProcessViolations != nil {
				for deploymentID, processes := range c.expectedProcessViolations {
					expectedProcesses := set.NewStringSet(sliceutils.Map(processes, func(p *storage.ProcessIndicator) string {
						return p.GetId()
					}).([]string)...)
					deployment := suite.deployments[deploymentID]

					for _, process := range suite.deploymentsToIndicators[deploymentID] {
						match, err := m.MatchDeploymentWithProcess(context.Background(), deployment, suite.getImagesForDeployment(deployment), process, false)
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
				violationsForDep, err := m.MatchDeployment(context.Background(), deployment, suite.getImagesForDeployment(deployment))
				require.NoError(t, err)
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
								assert.ElementsMatch(t, actualViolations[id], []*storage.Alert_Violation{{Message: msg}})
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
					assert.ElementsMatch(t, violations, actualViolations[id])
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
			policyName: "DockerHub NGINX 1.10",
			expectedViolations: map[string][]*storage.Alert_Violation{
				fixtureDep.GetContainers()[0].GetImage().GetId(): {
					{
						Message: "Image has registry 'docker.io', remote 'library/nginx', and tag '1.10'",
					},
				},
				suite.imageIDFromDep(nginx110Dep): {
					{
						Message: "Image has registry 'docker.io', remote 'library/nginx', and tag '1.10'",
					},
				},
			},
		},
		{
			policyName: "Alpine Linux Package Manager (apk) in Image",
			expectedViolations: map[string][]*storage.Alert_Violation{
				suite.imageIDFromDep(apkDep): {
					{
						Message: "Image includes component apk 1.2",
					},
				},
			},
		},
		{
			policyName: "Ubuntu Package Manager in Image",
			expectedViolations: map[string][]*storage.Alert_Violation{
				suite.imageIDFromDep(componentDeps["apt"]): {
					{
						Message: "Image includes component apt",
					},
				},
			},
		},
		{
			policyName: "Curl in Image",
			expectedViolations: map[string][]*storage.Alert_Violation{
				suite.imageIDFromDep(curlDep): {
					{
						Message: "Image includes component curl 1.3",
					},
				},
			},
		},
		{
			policyName: "Red Hat Package Manager in Image",
			expectedViolations: map[string][]*storage.Alert_Violation{
				suite.imageIDFromDep(componentDeps["dnf"]): {
					{
						Message: "Image includes component dnf",
					},
				},
			},
		},
		{
			policyName: "Wget in Image",
			expectedViolations: map[string][]*storage.Alert_Violation{
				suite.imageIDFromDep(componentDeps["wget"]): {
					{
						Message: "Image includes component wget",
					},
				},
			},
		},
		{
			policyName: "90-Day Image Age",
			expectedViolations: map[string][]*storage.Alert_Violation{
				suite.imageIDFromDep(oldImageDep): {
					{
						Message: fmt.Sprintf("Image has time of creation %s", readable.Time(oldImageCreationTime)),
					},
				},
			},
		},
		{
			policyName: "30-Day Scan Age",
			expectedViolations: map[string][]*storage.Alert_Violation{
				suite.imageIDFromDep(oldScannedDep): {
					{
						Message: fmt.Sprintf("Image has time of last scan %s", readable.Time(oldScannedTime)),
					},
				},
			},
		},
		{
			policyName: "Secure Shell (ssh) Port Exposed in Image",
			expectedViolations: map[string][]*storage.Alert_Violation{
				suite.imageIDFromDep(imagePort22Dep): {
					{
						Message: "Dockerfile line 'EXPOSE 22/tcp' present",
					},
				},
			},
		},
		{
			policyName: "Insecure specified in CMD",
			expectedViolations: map[string][]*storage.Alert_Violation{
				suite.imageIDFromDep(insecureCMDDep): {
					{
						Message: "Dockerfile line 'CMD do an insecure thing' present",
					},
				},
			},
		},
		{
			policyName: "Improper Usage of Orchestrator Secrets Volume",
			expectedViolations: map[string][]*storage.Alert_Violation{
				suite.imageIDFromDep(runSecretsDep): {
					{
						Message: "Dockerfile line 'VOLUME /run/secrets' present",
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
				depWithNonSeriousVulnsImage.GetId():              {},
				fixtureDep.GetContainers()[0].GetImage().GetId(): {},
				fixtureDep.GetContainers()[1].GetImage().GetId(): {},
				suite.imageIDFromDep(oldScannedDep):              {},
			},
			sampleViolationForMatched: "Image has not been scanned",
			expectedViolations:        map[string][]*storage.Alert_Violation{},
		},
		{
			policyName: "Shellshock: Multiple CVEs",
			expectedViolations: map[string][]*storage.Alert_Violation{
				suite.imageIDFromDep(shellshockDep): {
					{
						Message: "CVE-2014-6271 (CVSS 6) found in component shellshock-1.2",
					},
				},
				fixtureDep.GetContainers()[1].GetImage().GetId(): {
					{
						Message: "CVE-2014-6271 (CVSS 5) found in component name-1.2.3.4",
					},
				},
			},
		},
		{
			policyName: "Apache Struts: CVE-2017-5638",
			expectedViolations: map[string][]*storage.Alert_Violation{
				suite.imageIDFromDep(strutsDep): {
					{
						Message: "CVE-2017-5638 (CVSS 8) found in component struts-1.2",
					},
				},
			},
		},
		{
			policyName: "Heartbleed: CVE-2014-0160",
			expectedViolations: map[string][]*storage.Alert_Violation{
				suite.imageIDFromDep(heartbleedDep): {
					{
						Message: "CVE-2014-0160 (CVSS 6) found in component heartbleed-1.2",
					},
				},
			},
		},
		{
			policyName: "Fixable CVSS >= 7",
			expectedViolations: map[string][]*storage.Alert_Violation{
				suite.imageIDFromDep(strutsDep): {
					{
						Message: "Fixable CVE-2017-5638 (CVSS 8) found in component struts-1.2, resolved by version v1.3",
					},
				},
			},
		},
		{
			policyName: "ADD Command used instead of COPY",
			expectedViolations: map[string][]*storage.Alert_Violation{
				suite.imageIDFromDep(addDockerFileDep): {
					{
						Message: "Dockerfile line 'ADD deploy.sh' present",
					},
				},
				fixtureDep.GetContainers()[0].GetImage().GetId(): {
					{
						Message: "Dockerfile line 'ADD FILE:blah' present",
					},
				},
				fixtureDep.GetContainers()[1].GetImage().GetId(): {
					{
						Message: "Dockerfile line 'ADD file:4eedf861fb567fffb2694b65ebd...' present",
					},
				},
			},
		},
		{
			policyName: "Required Image Label",
			shouldNotMatch: map[string]struct{}{
				"requiredImageLabelImage": {},
			},
			sampleViolationForMatched: "Image includes no labels",
		},
	}

	for _, c := range imageTestCases {
		p := suite.MustGetPolicy(c.policyName)
		suite.T().Run(fmt.Sprintf("%s (on images)", c.policyName), func(t *testing.T) {
			assert.Nil(t, c.expectedProcessViolations)

			convertedP, err := CloneAndEnsureConverted(p)
			require.NoError(t, err)
			m, err := BuildImageMatcher(convertedP)
			require.NoError(t, err)

			actualViolations := make(map[string][]*storage.Alert_Violation)
			for id, image := range suite.images {
				violationsForImg, err := m.MatchImage(context.Background(), image)
				suite.Require().NoError(err)
				suite.Nil(violationsForImg.ProcessViolation)
				if alertViolations := violationsForImg.AlertViolations; len(alertViolations) > 0 {
					actualViolations[id] = alertViolations
				}
			}

			for id, violations := range c.expectedViolations {
				assert.Contains(t, actualViolations, id)
				assert.ElementsMatch(t, violations, actualViolations[id])
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
						assert.ElementsMatch(t, actualViolations[id], []*storage.Alert_Violation{{Message: c.sampleViolationForMatched}})
					}
				}
			}
		})
	}
}

func (suite *DefaultPoliciesTestSuite) TestMapPolicyMatchOne() {
	suite.envIsolator.Setenv(features.BooleanPolicyLogic.EnvVar(), "true")
	defer suite.envIsolator.RestoreAll()

	noAnnotation := &storage.Deployment{
		Id: "noAnnotation",
	}
	suite.addDepAndImages(noAnnotation)

	validAnnotation := &storage.Deployment{
		Id: "validAnnotation",
		Annotations: map[string]string{
			"email": "joseph@rules.gov",
		},
	}
	// TODO(rc) update when we have map values
	expectedViolations := []*storage.Alert_Violation{{Message: "Deployment includes no annotations"}}

	suite.addDepAndImages(validAnnotation)

	legacyPolicy := suite.defaultPolicies["Required Annotation: Email"]
	policy, err := CloneAndEnsureConverted(legacyPolicy)
	suite.NoError(err)

	m, err := BuildDeploymentMatcher(policy)
	suite.NoError(err)

	matched, err := m.MatchDeployment(context.Background(), noAnnotation, nil)
	suite.NoError(err)

	suite.Len(matched.AlertViolations, 1)
	suite.ElementsMatch(matched.AlertViolations, expectedViolations)

	matched, err = m.MatchDeployment(context.Background(), validAnnotation, nil)
	suite.NoError(err)
	suite.Empty(matched.AlertViolations)
}

func (suite *DefaultPoliciesTestSuite) TestRuntimePolicyFieldsCompile() {
	for _, p := range suite.defaultPolicies {
		if policyUtils.AppliesAtRunTime(p) && p.GetFields().GetProcessPolicy() != nil {
			processPolicy := p.GetFields().GetProcessPolicy()
			if processPolicy.GetName() != "" {
				regexp.MustCompile(processPolicy.GetName())
			}
			if processPolicy.GetArgs() != "" {
				regexp.MustCompile(processPolicy.GetArgs())
			}
			if processPolicy.GetAncestor() != "" {
				regexp.MustCompile(processPolicy.GetAncestor())
			}
		}
	}
}

func policyWithGroups(groups ...*storage.PolicyGroup) *storage.Policy {
	return &storage.Policy{
		PolicyVersion:  Version,
		Name:           uuid.NewV4().String(),
		PolicySections: []*storage.PolicySection{{PolicyGroups: groups}},
	}
}

func policyWithSingleGroup(group *storage.PolicyGroup) *storage.Policy {
	return policyWithGroups(group)
}

func policyGroupWithSingleKeyValue(fieldName, value string, negate bool) *storage.PolicyGroup {
	return &storage.PolicyGroup{FieldName: fieldName, Values: []*storage.PolicyValue{{Value: value}}, Negate: negate}
}

func policyWithSingleKeyValue(fieldName, value string, negate bool) *storage.Policy {
	return policyWithSingleGroup(policyGroupWithSingleKeyValue(fieldName, value, negate))
}

func policyWithSingleFieldAndValues(fieldName string, values []string, negate bool, op storage.BooleanOperator) *storage.Policy {
	return policyWithSingleGroup(&storage.PolicyGroup{FieldName: fieldName, Values: sliceutils.Map(values, func(val *string) *storage.PolicyValue {
		return &storage.PolicyValue{Value: *val}
	}).([]*storage.PolicyValue), Negate: negate, BooleanOperator: op})
}

func processWhitelistMessage(dep *storage.Deployment, whitelist bool, privileged bool, processNames ...string) []*storage.Alert_Violation {
	violations := make([]*storage.Alert_Violation, 0, len(processNames))
	containerName := dep.GetContainers()[0].GetName()
	for _, p := range processNames {
		if whitelist {
			msg := fmt.Sprintf("Unexpected process '%s' in container '%s'", p, containerName)
			violations = append(violations, &storage.Alert_Violation{Message: msg})
		}
		if privileged {
			violations = append(violations, privilegedMessage(dep)...)
		}
	}
	return violations
}

func privilegedMessage(dep *storage.Deployment) []*storage.Alert_Violation {
	containerName := dep.GetContainers()[0].GetName()
	return []*storage.Alert_Violation{{Message: fmt.Sprintf("Container '%s' is privileged", containerName)}}
}

func rbacPermissionMessage(level string) []*storage.Alert_Violation {
	permissionToDescMap := map[string]string{
		"NONE":                  "no specified access",
		"DEFAULT":               "default access",
		"ELEVATED_IN_NAMESPACE": "elevated access in namespace",
		"ELEVATED_CLUSTER_WIDE": "elevated access cluster wide",
		"CLUSTER_ADMIN":         "cluster admin access"}
	return []*storage.Alert_Violation{{Message: fmt.Sprintf("Service account permission level with %s", permissionToDescMap[level])}}
}

func (suite *DefaultPoliciesTestSuite) TestK8sRBACField() {
	deployments := make(map[string]*storage.Deployment)
	for permissionLevelStr, permissionLevel := range storage.PermissionLevel_value {
		dep := fixtures.GetDeployment().Clone()
		dep.ServiceAccountPermissionLevel = storage.PermissionLevel(permissionLevel)
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
				violations, err := matcher.MatchDeployment(context.Background(), dep, suite.getImagesForDeployment(dep))
				require.NoError(t, err)
				if len(violations.AlertViolations) > 0 {
					matched.Add(depRef)
					assert.ElementsMatch(t, violations.AlertViolations, c.expectedViolations[depRef])
				} else {
					assert.Empty(t, c.expectedViolations[depRef])
				}
			}
			assert.ElementsMatch(t, matched.AsSlice(), c.expectedMatches, "Got %v, expected: %v", matched.AsSlice(), c.expectedMatches)
		})
	}
}

// TODO(rc) check violation messages for port exposure
func (suite *DefaultPoliciesTestSuite) TestPortExposure() {
	deployments := make(map[string]*storage.Deployment)
	for exposureLevelStr, exposureLevel := range storage.PortConfig_ExposureLevel_value {
		dep := fixtures.GetDeployment().Clone()
		dep.Ports = []*storage.PortConfig{{ExposureInfos: []*storage.PortConfig_ExposureInfo{{Level: storage.PortConfig_ExposureLevel(exposureLevel)}}}}
		deployments[exposureLevelStr] = dep
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
			[]string{"INTERNAL", "HOST"},
		},
	} {
		c := testCase
		suite.T().Run(fmt.Sprintf("%+v", c), func(t *testing.T) {
			matcher, err := BuildDeploymentMatcher(policyWithSingleFieldAndValues(fieldnames.PortExposure, c.values, c.negate, storage.BooleanOperator_OR))
			require.NoError(t, err)
			matched := set.NewStringSet()
			for depRef, dep := range deployments {
				violations, err := matcher.MatchDeployment(context.Background(), dep, suite.getImagesForDeployment(dep))
				require.NoError(t, err)
				if len(violations.AlertViolations) > 0 {
					matched.Add(depRef)
				}
			}
			assert.ElementsMatch(t, matched.AsSlice(), c.expectedMatches, "Got %v, expected: %v", matched.AsSlice(), c.expectedMatches)
		})
	}
}

// TODO(rc) check violation messages for drop caps
func (suite *DefaultPoliciesTestSuite) TestDropCaps() {
	testCaps := []string{"CAP_SYS_MODULE", "CAP_SYS_NICE", "CAP_SYS_PTRACE"}

	deployments := make(map[string]*storage.Deployment)
	for _, idxs := range [][]int{{0}, {1}, {2}, {0, 1}, {1, 2}, {0, 1, 2}} {
		dep := fixtures.GetDeployment().Clone()
		dep.Containers[0].SecurityContext.DropCapabilities = make([]string, 0, len(idxs))
		for _, idx := range idxs {
			dep.Containers[0].SecurityContext.DropCapabilities = append(dep.Containers[0].SecurityContext.DropCapabilities, testCaps[idx])
		}
		deployments[strings.ReplaceAll(strings.Join(dep.Containers[0].SecurityContext.DropCapabilities, ","), "CAP_SYS_", "")] = dep
	}

	for _, testCase := range []struct {
		values          []string
		op              storage.BooleanOperator
		expectedMatches []string
	}{
		{
			// Nothing drops this capability
			[]string{"CAP_SYSLOG"},
			storage.BooleanOperator_OR,
			[]string{"MODULE", "NICE", "PTRACE", "MODULE,NICE", "NICE,PTRACE", "MODULE,NICE,PTRACE"},
		},
		{
			[]string{"CAP_SYS_NICE"},
			storage.BooleanOperator_OR,
			[]string{"MODULE", "PTRACE"},
		},
		{
			[]string{"CAP_SYS_NICE", "CAP_SYS_PTRACE"},
			storage.BooleanOperator_OR,
			[]string{"MODULE"},
		},
		{
			[]string{"CAP_SYS_NICE", "CAP_SYS_PTRACE"},
			storage.BooleanOperator_AND,
			[]string{"MODULE", "PTRACE", "NICE", "MODULE,NICE"},
		},
	} {
		c := testCase
		suite.T().Run(fmt.Sprintf("%+v", c), func(t *testing.T) {
			matcher, err := BuildDeploymentMatcher(policyWithSingleFieldAndValues(fieldnames.DropCaps, c.values, false, c.op))
			require.NoError(t, err)
			matched := set.NewStringSet()
			for depRef, dep := range deployments {
				violations, err := matcher.MatchDeployment(context.Background(), dep, suite.getImagesForDeployment(dep))
				require.NoError(t, err)
				if len(violations.AlertViolations) > 0 {
					matched.Add(depRef)
				}
			}
			assert.ElementsMatch(t, matched.AsSlice(), c.expectedMatches, "Got %v, expected: %v", matched.AsSlice(), c.expectedMatches)
		})
	}
}

func (suite *DefaultPoliciesTestSuite) TestProcessWhitelist() {
	privilegedDep := fixtures.GetDeployment().Clone()
	privilegedDep.Id = "PRIVILEGED"
	suite.addDepAndImages(privilegedDep)

	nonPrivilegedDep := fixtures.GetDeployment().Clone()
	nonPrivilegedDep.Id = "NOTPRIVILEGED"
	nonPrivilegedDep.Containers[0].SecurityContext.Privileged = false
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
	processesOutsideWhitelist := map[string]set.StringSet{
		privilegedDep.GetId():    set.NewStringSet(aptGetKey, aptGet2Key, bashKey),
		nonPrivilegedDep.GetId(): set.NewStringSet(aptGetKey, curlKey, bashKey),
	}

	// Plain groups
	aptGetGroup := policyGroupWithSingleKeyValue(fieldnames.ProcessName, "apt-get", false)
	privilegedGroup := policyGroupWithSingleKeyValue(fieldnames.Privileged, "true", false)
	whitelistGroup := policyGroupWithSingleKeyValue(fieldnames.WhitelistsEnabled, "true", false)

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
			groups: []*storage.PolicyGroup{whitelistGroup},
			expectedMatches: map[string][]string{
				privilegedDep.GetId():    {aptGetKey, aptGet2Key, bashKey},
				nonPrivilegedDep.GetId(): {aptGetKey, curlKey, bashKey},
			},
			expectedProcessMatches: map[string][]string{
				privilegedDep.GetId():    {aptGetKey, aptGet2Key, bashKey},
				nonPrivilegedDep.GetId(): {aptGetKey, curlKey, bashKey},
			},
			expectedViolations: map[string][]*storage.Alert_Violation{
				privilegedDep.GetId():    processWhitelistMessage(privilegedDep, true, false, "apt-get", "apt-get", "bash"),
				nonPrivilegedDep.GetId(): processWhitelistMessage(nonPrivilegedDep, true, false, "apt-get", "bash", "curl"),
			},
		},

		{
			groups: []*storage.PolicyGroup{privilegedGroup},
			expectedMatches: map[string][]string{
				privilegedDep.GetId(): {aptGetKey, aptGet2Key, curlKey, bashKey},
			},
			expectedProcessMatches: map[string][]string{},
			expectedViolations: map[string][]*storage.Alert_Violation{
				privilegedDep.GetId(): processWhitelistMessage(privilegedDep, false, true, "apt-get", "apt-get", "curl", "bash"),
			},
		},
		{
			groups: []*storage.PolicyGroup{aptGetGroup, whitelistGroup},
			expectedMatches: map[string][]string{
				privilegedDep.GetId():    {aptGetKey, aptGet2Key},
				nonPrivilegedDep.GetId(): {aptGetKey},
			},
			expectedViolations: map[string][]*storage.Alert_Violation{
				privilegedDep.GetId():    processWhitelistMessage(privilegedDep, true, false, "apt-get", "apt-get"),
				nonPrivilegedDep.GetId(): processWhitelistMessage(nonPrivilegedDep, true, false, "apt-get"),
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
				privilegedDep.GetId(): processWhitelistMessage(privilegedDep, false, true, "apt-get", "apt-get"),
			},
			expectedProcessMatches: map[string][]string{
				privilegedDep.GetId(): {aptGetKey, aptGet2Key},
			},
		},
		{
			groups: []*storage.PolicyGroup{privilegedGroup, whitelistGroup},
			expectedMatches: map[string][]string{
				privilegedDep.GetId(): {aptGetKey, aptGet2Key, bashKey},
			},
			expectedViolations: map[string][]*storage.Alert_Violation{
				privilegedDep.GetId(): processWhitelistMessage(privilegedDep, true, true, "apt-get", "apt-get", "bash"),
			},
			expectedProcessMatches: map[string][]string{
				privilegedDep.GetId(): {aptGetKey, aptGet2Key, bashKey},
			},
		},
		{
			groups: []*storage.PolicyGroup{aptGetGroup, privilegedGroup, whitelistGroup},
			expectedMatches: map[string][]string{
				privilegedDep.GetId(): {aptGetKey, aptGet2Key},
			},
			expectedViolations: map[string][]*storage.Alert_Violation{
				privilegedDep.GetId(): processWhitelistMessage(privilegedDep, true, true, "apt-get", "apt-get"),
			},
			expectedProcessMatches: map[string][]string{
				privilegedDep.GetId(): {aptGetKey, aptGet2Key},
			},
		},
	} {
		c := testCase
		suite.T().Run(fmt.Sprintf("%+v", c.groups), func(t *testing.T) {
			m, err := BuildDeploymentMatcher(policyWithGroups(c.groups...))
			require.NoError(t, err)

			actualMatches := make(map[string][]string)
			actualProcessMatches := make(map[string][]string)
			actualViolations := make(map[string][]*storage.Alert_Violation)
			for _, dep := range []*storage.Deployment{privilegedDep, nonPrivilegedDep} {
				for _, key := range []string{aptGetKey, aptGet2Key, curlKey, bashKey} {
					violations, err := m.MatchDeploymentWithProcess(context.Background(), dep, suite.getImagesForDeployment(dep), indicators[dep.GetId()][key], processesOutsideWhitelist[dep.GetId()].Contains(key))
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
				assert.ElementsMatch(t, violations, actualViolations[id])
			}
		})
	}
}
