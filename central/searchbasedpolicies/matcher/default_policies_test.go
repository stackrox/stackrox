package matcher

import (
	"fmt"
	"os"
	"regexp"
	"testing"
	"time"

	"github.com/blevesearch/bleve"
	"github.com/boltdb/bolt"
	"github.com/gogo/protobuf/proto"
	gogoTypes "github.com/gogo/protobuf/types"
	deploymentIndex "github.com/stackrox/rox/central/deployment/index"
	deploymentMappings "github.com/stackrox/rox/central/deployment/index/mappings"
	"github.com/stackrox/rox/central/globalindex"
	imageIndex "github.com/stackrox/rox/central/image/index"
	imageMappings "github.com/stackrox/rox/central/image/index/mappings"
	processIndicatorDataStore "github.com/stackrox/rox/central/processindicator/datastore"
	processIndicatorIndex "github.com/stackrox/rox/central/processindicator/index"
	processIndicatorSearch "github.com/stackrox/rox/central/processindicator/search"
	processIndicatorStore "github.com/stackrox/rox/central/processindicator/store"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/image/policies"
	"github.com/stackrox/rox/pkg/bolthelper"
	"github.com/stackrox/rox/pkg/defaults"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/images/types"
	policyUtils "github.com/stackrox/rox/pkg/policies"
	"github.com/stackrox/rox/pkg/protoconv"
	"github.com/stackrox/rox/pkg/readable"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

func TestDefaultPolicies(t *testing.T) {
	suite.Run(t, new(DefaultPoliciesTestSuite))
}

type DefaultPoliciesTestSuite struct {
	suite.Suite

	bleveIndex bleve.Index
	db         *bolt.DB

	deploymentIndexer deploymentIndex.Indexer
	imageIndexer      imageIndex.Indexer
	processDataStore  processIndicatorDataStore.DataStore

	defaultPolicies map[string]*v1.Policy
}

func (suite *DefaultPoliciesTestSuite) SetupTest() {
	var err error
	suite.bleveIndex, err = globalindex.TempInitializeIndices("")
	suite.Require().NoError(err)

	suite.db, err = bolthelper.NewTemp("default_policies_test.db")
	suite.Require().NoError(err)

	suite.deploymentIndexer = deploymentIndex.New(suite.bleveIndex)
	suite.imageIndexer = imageIndex.New(suite.bleveIndex)
	processStore := processIndicatorStore.New(suite.db)
	processIndexer := processIndicatorIndex.New(suite.bleveIndex)
	processSearcher, err := processIndicatorSearch.New(processStore, processIndexer)
	suite.Require().NoError(err)
	suite.processDataStore = processIndicatorDataStore.New(processStore, processIndexer, processSearcher)

	defaults.PoliciesPath = policies.Directory()

	defaultPolicies, err := defaults.Policies()
	suite.Require().NoError(err)

	suite.defaultPolicies = make(map[string]*v1.Policy, len(defaultPolicies))
	for _, p := range defaultPolicies {
		suite.defaultPolicies[p.GetName()] = p
	}
}

func (suite *DefaultPoliciesTestSuite) TearDownTest() {
	suite.bleveIndex.Close()
	suite.db.Close()
	os.Remove(suite.db.Path())
}

func (suite *DefaultPoliciesTestSuite) MustGetPolicy(name string) *v1.Policy {
	p, ok := suite.defaultPolicies[name]
	suite.Require().True(ok, "Policy %s not found", name)
	return p
}

func (suite *DefaultPoliciesTestSuite) mustIndexDepAndImages(deployment *v1.Deployment) {
	suite.NoError(suite.deploymentIndexer.AddDeployment(deployment))
	for _, container := range deployment.GetContainers() {
		if container.GetImage() != nil {
			suite.NoError(suite.imageIndexer.AddImage(container.GetImage()))
		}
	}
}

func imageWithComponents(components []*v1.ImageScanComponent) *v1.Image {
	return &v1.Image{
		Id:   uuid.NewV4().String(),
		Name: &v1.ImageName{FullName: "ASFASF"},
		Scan: &v1.ImageScan{
			Components: components,
		},
	}
}

func imageWithLayers(layers []*v1.ImageLayer) *v1.Image {
	return &v1.Image{
		Id: uuid.NewV4().String(),
		Metadata: &v1.ImageMetadata{
			V1: &v1.V1Metadata{
				Layers: layers,
			},
		},
	}
}

func deploymentWithImage(img *v1.Image) *v1.Deployment {
	return &v1.Deployment{
		Id:         uuid.NewV4().String(),
		Containers: []*v1.Container{{Image: img}},
	}
}

func deploymentWithComponents(components []*v1.ImageScanComponent) *v1.Deployment {
	return deploymentWithImage(imageWithComponents(components))
}

func deploymentWithLayers(layers []*v1.ImageLayer) *v1.Deployment {
	return deploymentWithImage(imageWithLayers(layers))
}

func (suite *DefaultPoliciesTestSuite) imageIDFromDep(deployment *v1.Deployment) string {
	suite.Require().Len(deployment.GetContainers(), 1, "This function only supports deployments with exactly one container")
	id := deployment.GetContainers()[0].GetImage().GetId()
	suite.NotEmpty(id, "Deployment '%s' had no image id", proto.MarshalTextString(deployment))
	return types.NewDigest(id).Digest()
}

func (suite *DefaultPoliciesTestSuite) mustAddIndicator(deploymentID, name, args string) *v1.ProcessIndicator {
	indicator := &v1.ProcessIndicator{
		Id:           uuid.NewV4().String(),
		DeploymentId: deploymentID,
		Signal: &v1.ProcessSignal{
			Name: name,
			Args: args,
			Time: gogoTypes.TimestampNow(),
		},
	}
	err := suite.processDataStore.AddProcessIndicator(indicator)
	suite.NoError(err)
	return indicator
}

type testCase struct {
	policyName         string
	expectedViolations map[string][]*v1.Alert_Violation

	// If shouldNotMatch is specified (which is the case for policies that check for the absence of something), we verify that
	// it matches everything except shouldNotMatch.
	// If sampleViolationForMatched is provided, we verify that all the matches are the string provided in sampleViolationForMatched.
	shouldNotMatch            map[string]struct{}
	sampleViolationForMatched string
}

func (suite *DefaultPoliciesTestSuite) TestDefaultPolicies() {
	fixtureDep := fixtures.GetDeployment()
	suite.mustIndexDepAndImages(fixtureDep)

	nginx110 := &v1.Image{
		Id: "SHANGINX110",
		Name: &v1.ImageName{
			Registry: "docker.io",
			Remote:   "library/nginx",
			Tag:      "1.10",
		},
	}
	nginx110Dep := &v1.Deployment{
		Id: "nginx110",
		Containers: []*v1.Container{
			{Image: nginx110},
		},
	}
	suite.mustIndexDepAndImages(nginx110Dep)

	oldScannedTime := time.Now().Add(-31 * 24 * time.Hour)
	oldScannedImage := &v1.Image{
		Id: "SHAOLDSCANNED",
		Scan: &v1.ImageScan{
			ScanTime: protoconv.ConvertTimeToTimestamp(oldScannedTime),
		},
	}
	oldScannedDep := &v1.Deployment{
		Id: "oldscanned",
		Containers: []*v1.Container{
			{Image: oldScannedImage},
		},
	}
	suite.mustIndexDepAndImages(oldScannedDep)

	addDockerFileDep := deploymentWithLayers([]*v1.ImageLayer{
		{
			Instruction: "ADD",
			Value:       "deploy.sh",
		},
		{
			Instruction: "RUN",
			Value:       "deploy.sh",
		},
	})
	suite.mustIndexDepAndImages(addDockerFileDep)

	minerdDep := deploymentWithLayers([]*v1.ImageLayer{
		{
			Instruction: "RUN",
			Value:       "deploy.sh",
		},
		{
			Instruction: "ENTRYPOINT",
			Value:       "minerd",
		},
	})
	suite.mustIndexDepAndImages(minerdDep)

	imagePort22Dep := deploymentWithLayers([]*v1.ImageLayer{
		{
			Instruction: "EXPOSE",
			Value:       "22/tcp",
		},
	})
	suite.mustIndexDepAndImages(imagePort22Dep)

	insecureCMDDep := deploymentWithLayers([]*v1.ImageLayer{
		{
			Instruction: "CMD",
			Value:       "do an insecure thing",
		},
	})
	suite.mustIndexDepAndImages(insecureCMDDep)

	runSecretsDep := deploymentWithLayers([]*v1.ImageLayer{
		{
			Instruction: "VOLUME",
			Value:       "/run/secrets",
		},
	})
	suite.mustIndexDepAndImages(runSecretsDep)

	oldImageCreationTime := time.Now().Add(-100 * 24 * time.Hour)
	oldCreatedImage := &v1.Image{
		Id: "SHA:OLDCREATEDIMAGE",
		Metadata: &v1.ImageMetadata{
			V1: &v1.V1Metadata{
				Created: protoconv.ConvertTimeToTimestamp(oldImageCreationTime),
			},
		},
	}
	oldImageDep := &v1.Deployment{
		Id:         "oldimagedep",
		Containers: []*v1.Container{{Image: oldCreatedImage}},
	}
	suite.mustIndexDepAndImages(oldImageDep)

	apkDep := deploymentWithComponents([]*v1.ImageScanComponent{
		{Name: "apk", Version: "1.2"},
		{Name: "asfa", Version: "1.5"},
	})
	suite.mustIndexDepAndImages(apkDep)

	curlDep := deploymentWithComponents([]*v1.ImageScanComponent{
		{Name: "curl", Version: "1.3"},
		{Name: "curlwithextra", Version: "0.9"},
	})
	suite.mustIndexDepAndImages(curlDep)

	componentDeps := make(map[string]*v1.Deployment)
	for _, component := range []string{"apt", "dnf", "wget", "yum", "rpm"} {
		dep := deploymentWithComponents([]*v1.ImageScanComponent{
			{Name: component},
		})
		suite.mustIndexDepAndImages(dep)
		componentDeps[component] = dep
	}

	heartbleedDep := &v1.Deployment{
		Id: "HEARTBLEEDDEPID",
		Containers: []*v1.Container{
			{
				SecurityContext: &v1.SecurityContext{Privileged: true},
				Image: &v1.Image{
					Id: "HEARTBLEEDDEPSHA",
					Scan: &v1.ImageScan{
						Components: []*v1.ImageScanComponent{
							{Name: "heartbleed", Version: "1.2", Vulns: []*v1.Vulnerability{
								{Cve: "CVE-2014-0160", Link: "https://heartbleed", Cvss: 6},
							}},
						},
					},
				},
			},
		},
	}
	suite.mustIndexDepAndImages(heartbleedDep)

	shellshockDep := deploymentWithComponents([]*v1.ImageScanComponent{
		{Name: "shellshock", Version: "1.2", Vulns: []*v1.Vulnerability{
			{Cve: "CVE-2014-6271", Link: "https://shellshock", Cvss: 6},
			{Cve: "CVE-ARBITRARY", Link: "https://notshellshock"},
		}},
	})
	suite.mustIndexDepAndImages(shellshockDep)

	strutsDep := deploymentWithComponents([]*v1.ImageScanComponent{
		{Name: "struts", Version: "1.2", Vulns: []*v1.Vulnerability{
			{Cve: "CVE-2017-5638", Link: "https://struts", Cvss: 8},
		}},
		{Name: "OTHER", Version: "1.3", Vulns: []*v1.Vulnerability{
			{Cve: "CVE-1223-451", Link: "https://cvefake"},
		}},
	})
	suite.mustIndexDepAndImages(strutsDep)

	depWithNonSeriousVulns := deploymentWithComponents([]*v1.ImageScanComponent{
		{Name: "NOSERIOUS", Version: "2.3", Vulns: []*v1.Vulnerability{
			{Cve: "CVE-1234-5678", Link: "https://abcdefgh"},
			{Cve: "CVE-5678-1234", Link: "https://lmnopqrst"},
		}},
	})
	suite.mustIndexDepAndImages(depWithNonSeriousVulns)

	dockerSockDep := &v1.Deployment{
		Id: "DOCKERSOCDEP",
		Containers: []*v1.Container{
			{Volumes: []*v1.Volume{
				{Source: "/var/run/docker.sock", Name: "DOCKERSOCK"},
				{Source: "NOTDOCKERSOCK"},
			}},
		},
	}
	suite.mustIndexDepAndImages(dockerSockDep)

	containerPort22Dep := &v1.Deployment{
		Id: "CONTAINERPORT22DEP",
		Containers: []*v1.Container{
			{Ports: []*v1.PortConfig{
				{Protocol: "tcp", ContainerPort: 22},
				{Protocol: "udp", ContainerPort: 4125},
			}},
		},
	}
	suite.mustIndexDepAndImages(containerPort22Dep)

	secretEnvDep := &v1.Deployment{
		Id: "SECRETENVDEP",
		Containers: []*v1.Container{
			{Config: &v1.ContainerConfig{
				Env: []*v1.ContainerConfig_EnvironmentConfig{
					{Key: "THIS_IS_SECRET_VAR", Value: "stealthmode"},
					{Key: "HOME", Value: "/home/stackrox"},
				},
			}},
		},
	}
	suite.mustIndexDepAndImages(secretEnvDep)

	// Fake deployment that shouldn't match anything, just to make sure
	// that none of our queries will accidentally match it.
	suite.mustIndexDepAndImages(&v1.Deployment{Id: "FAKEID", Name: "FAKENAME"})

	depWithGoodEmailAnnotation := &v1.Deployment{
		Id: "GOODEMAILDEPID",
		Annotations: map[string]string{
			"email": "vv@stackrox.com",
		},
	}
	suite.mustIndexDepAndImages(depWithGoodEmailAnnotation)

	depWithOwnerAnnotation := &v1.Deployment{
		Id: "OWNERANNOTATIONDEP",
		Annotations: map[string]string{
			"owner": "IOWNTHIS",
			"blah":  "Blah",
		},
	}
	suite.mustIndexDepAndImages(depWithOwnerAnnotation)

	depWitharbitraryAnnotations := &v1.Deployment{
		Id: "ARBITRARYANNOTATIONDEPID",
		Annotations: map[string]string{
			"emailnot": "vv@stackrox.com",
			"notemail": "vv@stackrox.com",
			"ownernot": "vv",
			"nowner":   "vv",
		},
	}
	suite.mustIndexDepAndImages(depWitharbitraryAnnotations)

	depWithBadEmailAnnotation := &v1.Deployment{
		Id: "BADEMAILDEPID",
		Annotations: map[string]string{
			"email": "NOTANEMAIL",
		},
	}
	suite.mustIndexDepAndImages(depWithBadEmailAnnotation)

	sysAdminDep := &v1.Deployment{
		Id: "SYSADMINDEPID",
		Containers: []*v1.Container{
			{
				SecurityContext: &v1.SecurityContext{
					AddCapabilities: []string{"CAP_SYS_ADMIN"},
				},
			},
		},
	}
	suite.mustIndexDepAndImages(sysAdminDep)

	depWithAllResourceLimitsRequestsSpecified := &v1.Deployment{
		Id: "ALLRESOURCESANDLIMITSDEP",
		Containers: []*v1.Container{
			{Resources: &v1.Resources{
				CpuCoresRequest: 0.1,
				CpuCoresLimit:   0.3,
				MemoryMbLimit:   100,
				MemoryMbRequest: 1251,
			}},
		},
	}
	suite.mustIndexDepAndImages(depWithAllResourceLimitsRequestsSpecified)

	// Index processes
	aptgetIndicator := suite.mustAddIndicator(fixtureDep.GetId(), "apt-get", "install nmap")

	fixtureDepAptIndicator := suite.mustAddIndicator(fixtureDep.GetId(), "apt", "")
	sysAdminDepAptIndicator := suite.mustAddIndicator(sysAdminDep.GetId(), "apt", "install blah")

	kubeletIndicator := suite.mustAddIndicator(containerPort22Dep.GetId(), "curl", "https://12.13.14.15:10250")
	kubeletIndicator2 := suite.mustAddIndicator(containerPort22Dep.GetId(), "wget", "https://heapster.kube-system/metrics")

	nmapIndicatorfixtureDep1 := suite.mustAddIndicator(fixtureDep.GetId(), "nmap", "blah")
	nmapIndicatorfixtureDep2 := suite.mustAddIndicator(fixtureDep.GetId(), "nmap", "blah2")
	nmapIndicatorNginx110Dep := suite.mustAddIndicator(nginx110Dep.GetId(), "nmap", "")

	// Find all the deployments indexed.
	allDeployments, err := suite.deploymentIndexer.Search(search.EmptyQuery())
	suite.NoError(err)

	allImages, err := suite.imageIndexer.Search(search.EmptyQuery())
	suite.NoError(err)

	deploymentTestCases := []testCase{
		{
			policyName: "Latest tag",
			expectedViolations: map[string][]*v1.Alert_Violation{
				fixtureDep.GetId(): {
					{
						Message: "Image tag 'latest' matched latest",
					},
				},
			},
		},
		{
			policyName: "DockerHub NGINX 1.10",
			expectedViolations: map[string][]*v1.Alert_Violation{
				fixtureDep.GetId(): {
					{
						Message: "Image tag '1.10' matched 1.10",
					},
					{
						Message: "Image registry 'docker.io' matched docker.io",
					},
					{
						Message: "Image remote 'library/nginx' matched nginx",
					},
				},
				nginx110Dep.GetId(): {
					{
						Message: "Image tag '1.10' matched 1.10",
					},
					{
						Message: "Image registry 'docker.io' matched docker.io",
					},
					{
						Message: "Image remote 'library/nginx' matched nginx",
					},
				},
			},
		},
		{
			policyName: "Alpine Linux Package Manager (apk) in Image",
			expectedViolations: map[string][]*v1.Alert_Violation{
				apkDep.GetId(): {
					{
						Message: "Component name 'apk' matched ^apk$",
					},
				},
			},
		},
		{
			policyName: "Aptitude Package Manager (apt) in Image",
			expectedViolations: map[string][]*v1.Alert_Violation{
				componentDeps["apt"].GetId(): {
					{
						Message: "Component name 'apt' matched ^apt$",
					},
				},
			},
		},
		{
			policyName: "Curl in Image",
			expectedViolations: map[string][]*v1.Alert_Violation{
				curlDep.GetId(): {
					{
						Message: "Component name 'curl' matched ^curl$",
					},
				},
			},
		},
		{
			policyName: "DNF Package Manager (dnf) in Image",
			expectedViolations: map[string][]*v1.Alert_Violation{
				componentDeps["dnf"].GetId(): {
					{
						Message: "Component name 'dnf' matched ^dnf$",
					},
				},
			},
		},
		{
			policyName: "Wget in Image",
			expectedViolations: map[string][]*v1.Alert_Violation{
				componentDeps["wget"].GetId(): {
					{
						Message: "Component name 'wget' matched ^wget$",
					},
				},
			},
		},
		{
			policyName: "Yum Package Manager (yum) in Image",
			expectedViolations: map[string][]*v1.Alert_Violation{
				componentDeps["yum"].GetId(): {
					{
						Message: "Component name 'yum' matched ^yum$",
					},
				},
			},
		},
		{
			policyName: "RPM Package Manager (rpm) in Image",
			expectedViolations: map[string][]*v1.Alert_Violation{
				componentDeps["rpm"].GetId(): {
					{
						Message: "Component name 'rpm' matched ^rpm$",
					},
				},
			},
		},
		{
			policyName: "Mount Docker Socket",
			expectedViolations: map[string][]*v1.Alert_Violation{
				dockerSockDep.GetId(): {
					{
						Message: "Volume source '/var/run/docker.sock' matched ^/var/run/docker.sock$",
					},
				},
			},
		},
		{
			policyName: "90-Day Image Age",
			expectedViolations: map[string][]*v1.Alert_Violation{
				oldImageDep.GetId(): {
					{
						Message: fmt.Sprintf("Time of image creation '%s' was more than 90 days ago", readable.Time(oldImageCreationTime)),
					},
				},
			},
		},
		{
			policyName: "30-Day Scan Age",
			expectedViolations: map[string][]*v1.Alert_Violation{
				oldScannedDep.GetId(): {
					{
						Message: fmt.Sprintf("Time of last scan '%s' was more than 30 days ago", readable.Time(oldScannedTime)),
					},
				},
			},
		},
		{
			policyName: "Image Port 22",
			expectedViolations: map[string][]*v1.Alert_Violation{
				imagePort22Dep.GetId(): {
					{
						Message: "Dockerfile Line 'EXPOSE 22/tcp' matches the rule EXPOSE (^22/tcp|\\s+22/tcp)",
					},
				},
			},
		},
		{
			policyName: "Container Port 22",
			expectedViolations: map[string][]*v1.Alert_Violation{
				containerPort22Dep.GetId(): {
					{
						Message: "Port '22' matched 22",
					},
					{
						Message: "Protocol 'tcp' matched tcp",
					},
				},
			},
		},
		{
			policyName: "Privileged Container",
			expectedViolations: map[string][]*v1.Alert_Violation{
				fixtureDep.GetId(): {
					{
						Message: "Privileged container found",
					},
				},
				heartbleedDep.GetId(): {
					{
						Message: "Privileged container found",
					},
				},
			},
		},
		{
			policyName: "Insecure specified in CMD",
			expectedViolations: map[string][]*v1.Alert_Violation{
				insecureCMDDep.GetId(): {
					{
						Message: "Dockerfile Line 'CMD do an insecure thing' matches the rule CMD .*insecure.*",
					},
				},
			},
		},
		{
			policyName: "Overwrites /run/secrets Volume",
			expectedViolations: map[string][]*v1.Alert_Violation{
				runSecretsDep.GetId(): {
					{
						Message: "Dockerfile Line 'VOLUME /run/secrets' matches the rule VOLUME /run/secrets",
					},
				},
			},
		},
		{
			policyName: "Images with no scans",
			shouldNotMatch: map[string]struct{}{
				// These deployments have scans on their images.
				fixtureDep.GetId():    {},
				oldScannedDep.GetId(): {},
				// The rest of the deployments have no images!
				"FAKEID":                                          {},
				containerPort22Dep.GetId():                        {},
				dockerSockDep.GetId():                             {},
				secretEnvDep.GetId():                              {},
				depWithOwnerAnnotation.GetId():                    {},
				depWithGoodEmailAnnotation.GetId():                {},
				depWithBadEmailAnnotation.GetId():                 {},
				depWitharbitraryAnnotations.GetId():               {},
				sysAdminDep.GetId():                               {},
				depWithAllResourceLimitsRequestsSpecified.GetId(): {},
			},
			sampleViolationForMatched: "Image has not been scanned",
		},
		{
			policyName: "Cryptomining Entrypoint",
			expectedViolations: map[string][]*v1.Alert_Violation{
				minerdDep.GetId(): {
					{
						Message: "Dockerfile Line 'ENTRYPOINT minerd' matches the rule ENTRYPOINT .*minerd.*",
					},
				},
			},
		},
		{
			policyName:                "Required Label: Email",
			shouldNotMatch:            map[string]struct{}{fixtureDep.GetId(): {}},
			sampleViolationForMatched: "Required label not found (key = 'email', value = '^[a-zA-Z0-9_.+-]+@[a-zA-Z0-9-]+\\.[a-zA-Z0-9-.]+$')",
		},
		{
			policyName:                "Required Annotation: Email",
			shouldNotMatch:            map[string]struct{}{depWithGoodEmailAnnotation.GetId(): {}},
			sampleViolationForMatched: "Required annotation not found (key = 'email', value = '^[a-zA-Z0-9_.+-]+@[a-zA-Z0-9-]+\\.[a-zA-Z0-9-.]+$')",
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
			expectedViolations: map[string][]*v1.Alert_Violation{
				sysAdminDep.GetId(): {
					{
						Message: "CAP_SYS_ADMIN was in the ADD CAPABILITIES list",
					},
				},
			},
		},
		{
			policyName: "Shellshock: CVE-2014-6271",
			expectedViolations: map[string][]*v1.Alert_Violation{
				shellshockDep.GetId(): {
					{
						Message: "CVE CVE-2014-6271 matched regex 'CVE-2014-6271'",
						Link:    "https://shellshock",
					},
				},
				fixtureDep.GetId(): {
					{
						Message: "CVE CVE-2014-6271 matched regex 'CVE-2014-6271'",
						Link:    "https://nvd.nist.gov/vuln/detail/CVE-2014-6271",
					},
				},
			},
		},
		{
			policyName: "Apache Struts: CVE-2017-5638",
			expectedViolations: map[string][]*v1.Alert_Violation{
				strutsDep.GetId(): {
					{
						Message: "CVE CVE-2017-5638 matched regex 'CVE-2017-5638'",
						Link:    "https://struts",
					},
				},
			},
		},
		{
			policyName: "Heartbleed: CVE-2014-0160",
			expectedViolations: map[string][]*v1.Alert_Violation{
				heartbleedDep.GetId(): {
					{
						Message: "CVE CVE-2014-0160 matched regex 'CVE-2014-0160'",
						Link:    "https://heartbleed",
					},
				},
			},
		},
		{
			policyName: "No resource requests or limits specified",
			expectedViolations: map[string][]*v1.Alert_Violation{
				fixtureDep.GetId(): {
					{Message: "The CPU resource limit of 0 is equal to the threshold of 0.00"},
					{Message: "The memory resource limit of 0 is equal to the threshold of 0.00"},
					{Message: "The memory resource request of 0 is equal to the threshold of 0.00"},
				},
			},
		},
		{
			policyName: "Don't use environment variables with secrets",
			expectedViolations: map[string][]*v1.Alert_Violation{
				secretEnvDep.GetId(): {
					{
						Message: "Container Environment (key='THIS_IS_SECRET_VAR', value='stealthmode') matched environment policy (key = '.*SECRET.*', value = '.*')",
					},
				},
			},
		},
		{
			policyName: "CVSS >= 6 and Privileged",
			expectedViolations: map[string][]*v1.Alert_Violation{
				heartbleedDep.GetId(): {
					{
						Message: "Found a CVSS score of 6 (greater than or equal to 6.0) (cve: CVE-2014-0160)",
					},
					{
						Message: "Privileged container found",
					},
				},
			},
		},
		{
			policyName: "CVSS >= 7",
			expectedViolations: map[string][]*v1.Alert_Violation{
				strutsDep.GetId(): {
					{
						Message: "Found a CVSS score of 8 (greater than or equal to 7.0) (cve: CVE-2017-5638)",
					},
				},
			},
		},
		{
			policyName: "ADD Command used instead of COPY",
			expectedViolations: map[string][]*v1.Alert_Violation{
				addDockerFileDep.GetId(): {
					{
						Message: "Dockerfile Line 'ADD deploy.sh' matches the rule ADD .*",
					},
				},
				fixtureDep.GetId(): {
					{
						Message: "Dockerfile Line 'ADD FILE:blah' matches the rule ADD .*",
					},
					{
						Message: "Dockerfile Line 'ADD file:4eedf861fb567fffb2694b65ebdd58d5e371a2c28c3863f363f333cb34e5eb7b in /' matches the rule ADD .*",
					},
				},
			},
		},
		{
			policyName: "apt-get Execution",
			expectedViolations: map[string][]*v1.Alert_Violation{
				fixtureDep.GetId(): {
					{
						Message:   "Found process with name matching 'apt-get$'",
						Processes: []*v1.ProcessIndicator{aptgetIndicator},
					},
				},
			},
		},
		{
			policyName: "nmap Execution",
			expectedViolations: map[string][]*v1.Alert_Violation{
				fixtureDep.GetId(): {
					{
						Message:   "Found processes with name matching 'nmap$'",
						Processes: []*v1.ProcessIndicator{nmapIndicatorfixtureDep1, nmapIndicatorfixtureDep2},
					},
				},
				nginx110Dep.GetId(): {
					{
						Message:   "Found process with name matching 'nmap$'",
						Processes: []*v1.ProcessIndicator{nmapIndicatorNginx110Dep},
					},
				},
			},
		},
		{
			policyName: "Process Targeting Cluster Kubelet Endpoint",
			expectedViolations: map[string][]*v1.Alert_Violation{
				containerPort22Dep.GetId(): {
					{
						Message:   "Found processes with args matching '(https?://)?(\\d{1,3}\\.\\d{1,3}\\.\\d{1,3}\\.\\d{1,3}\\:(10250|10248|10255)|heapster\\.kube\\-system/metrics|KUBERNETES_PORT_443_TCP_ADDR|KUBERNETES_SERVICE_HOST).*'",
						Processes: []*v1.ProcessIndicator{kubeletIndicator, kubeletIndicator2},
					},
				},
			},
		},
		{
			policyName: "apt Execution",
			expectedViolations: map[string][]*v1.Alert_Violation{
				fixtureDep.GetId(): {
					{
						Message:   "Found process with name matching 'apt$'",
						Processes: []*v1.ProcessIndicator{fixtureDepAptIndicator},
					},
				},
				sysAdminDep.GetId(): {
					{
						Message:   "Found process with name matching 'apt$'",
						Processes: []*v1.ProcessIndicator{sysAdminDepAptIndicator},
					},
				},
			},
		},
	}

	for _, c := range deploymentTestCases {
		p := suite.MustGetPolicy(c.policyName)
		suite.T().Run(fmt.Sprintf("%s (on deployments)", c.policyName), func(t *testing.T) {
			m, err := ForPolicy(p, deploymentMappings.OptionsMap, suite.processDataStore)
			require.NoError(t, err)
			matches, err := m.Match(suite.deploymentIndexer)
			require.NoError(t, err)
			validateDeploymentMatches(matches, allDeployments, c, t)

			var allIDs []string
			for _, deployment := range allDeployments {
				allIDs = append(allIDs, deployment.ID)
			}
			matchesFromMatchMany, err := m.MatchMany(suite.deploymentIndexer, allIDs...)
			require.NoError(t, err)
			validateDeploymentMatches(matchesFromMatchMany, allDeployments, c, t)

			var matchingIDs []string
			for id := range c.expectedViolations {
				matchingIDs = append(matchingIDs, id)
			}
			matchesFromExactlyMatchMany, err := m.MatchMany(suite.deploymentIndexer, matchingIDs...)
			require.NoError(t, err)
			validateDeploymentMatches(matchesFromExactlyMatchMany, allDeployments, c, t)

			for id, violations := range c.expectedViolations {
				// Test match one
				gotFromMatchOne, err := m.MatchOne(suite.deploymentIndexer, id)
				require.NoError(t, err)
				assert.ElementsMatch(t, violations, gotFromMatchOne, "Expected violations from match one %+v don't match what we got %+v", violations, gotFromMatchOne)
			}
		})
	}

	imageTestCases := []testCase{
		{
			policyName: "Latest tag",
			expectedViolations: map[string][]*v1.Alert_Violation{
				fixtureDep.GetContainers()[1].GetImage().GetId(): {
					{Message: "Image tag 'latest' matched latest"},
				},
			},
		},
		{
			policyName: "DockerHub NGINX 1.10",
			expectedViolations: map[string][]*v1.Alert_Violation{
				fixtureDep.GetContainers()[0].GetImage().GetId(): {
					{
						Message: "Image tag '1.10' matched 1.10",
					},
					{
						Message: "Image registry 'docker.io' matched docker.io",
					},
					{
						Message: "Image remote 'library/nginx' matched nginx",
					},
				},
				suite.imageIDFromDep(nginx110Dep): {
					{
						Message: "Image tag '1.10' matched 1.10",
					},
					{
						Message: "Image registry 'docker.io' matched docker.io",
					},
					{
						Message: "Image remote 'library/nginx' matched nginx",
					},
				},
			},
		},
		{
			policyName: "Alpine Linux Package Manager (apk) in Image",
			expectedViolations: map[string][]*v1.Alert_Violation{
				suite.imageIDFromDep(apkDep): {
					{
						Message: "Component name 'apk' matched ^apk$",
					},
				},
			},
		},
		{
			policyName: "Aptitude Package Manager (apt) in Image",
			expectedViolations: map[string][]*v1.Alert_Violation{
				suite.imageIDFromDep(componentDeps["apt"]): {
					{
						Message: "Component name 'apt' matched ^apt$",
					},
				},
			},
		},
		{
			policyName: "Curl in Image",
			expectedViolations: map[string][]*v1.Alert_Violation{
				suite.imageIDFromDep(curlDep): {
					{
						Message: "Component name 'curl' matched ^curl$",
					},
				},
			},
		},
		{
			policyName: "DNF Package Manager (dnf) in Image",
			expectedViolations: map[string][]*v1.Alert_Violation{
				suite.imageIDFromDep(componentDeps["dnf"]): {
					{
						Message: "Component name 'dnf' matched ^dnf$",
					},
				},
			},
		},
		{
			policyName: "Wget in Image",
			expectedViolations: map[string][]*v1.Alert_Violation{
				suite.imageIDFromDep(componentDeps["wget"]): {
					{
						Message: "Component name 'wget' matched ^wget$",
					},
				},
			},
		},
		{
			policyName: "Yum Package Manager (yum) in Image",
			expectedViolations: map[string][]*v1.Alert_Violation{
				suite.imageIDFromDep(componentDeps["yum"]): {
					{
						Message: "Component name 'yum' matched ^yum$",
					},
				},
			},
		},
		{
			policyName: "RPM Package Manager (rpm) in Image",
			expectedViolations: map[string][]*v1.Alert_Violation{
				suite.imageIDFromDep(componentDeps["rpm"]): {
					{
						Message: "Component name 'rpm' matched ^rpm$",
					},
				},
			},
		},
		{
			policyName: "90-Day Image Age",
			expectedViolations: map[string][]*v1.Alert_Violation{
				suite.imageIDFromDep(oldImageDep): {
					{
						Message: fmt.Sprintf("Time of image creation '%s' was more than 90 days ago", readable.Time(oldImageCreationTime)),
					},
				},
			},
		},
		{
			policyName: "30-Day Scan Age",
			expectedViolations: map[string][]*v1.Alert_Violation{
				suite.imageIDFromDep(oldScannedDep): {
					{
						Message: fmt.Sprintf("Time of last scan '%s' was more than 30 days ago", readable.Time(oldScannedTime)),
					},
				},
			},
		},
		{
			policyName: "Image Port 22",
			expectedViolations: map[string][]*v1.Alert_Violation{
				suite.imageIDFromDep(imagePort22Dep): {
					{
						Message: "Dockerfile Line 'EXPOSE 22/tcp' matches the rule EXPOSE (^22/tcp|\\s+22/tcp)",
					},
				},
			},
		},
		{
			policyName: "Insecure specified in CMD",
			expectedViolations: map[string][]*v1.Alert_Violation{
				suite.imageIDFromDep(insecureCMDDep): {
					{
						Message: "Dockerfile Line 'CMD do an insecure thing' matches the rule CMD .*insecure.*",
					},
				},
			},
		},
		{
			policyName: "Overwrites /run/secrets Volume",
			expectedViolations: map[string][]*v1.Alert_Violation{
				suite.imageIDFromDep(runSecretsDep): {
					{
						Message: "Dockerfile Line 'VOLUME /run/secrets' matches the rule VOLUME /run/secrets",
					},
				},
			},
		},
		{
			policyName: "Images with no scans",
			shouldNotMatch: map[string]struct{}{
				fixtureDep.GetContainers()[0].GetImage().GetId(): {},
				fixtureDep.GetContainers()[1].GetImage().GetId(): {},
				suite.imageIDFromDep(oldScannedDep):              {},
			},
			sampleViolationForMatched: "Image has not been scanned",
		},
		{
			policyName: "Cryptomining Entrypoint",
			expectedViolations: map[string][]*v1.Alert_Violation{
				suite.imageIDFromDep(minerdDep): {
					{
						Message: "Dockerfile Line 'ENTRYPOINT minerd' matches the rule ENTRYPOINT .*minerd.*",
					},
				},
			},
		},
		{
			policyName: "Shellshock: CVE-2014-6271",
			expectedViolations: map[string][]*v1.Alert_Violation{
				suite.imageIDFromDep(shellshockDep): {
					{
						Message: "CVE CVE-2014-6271 matched regex 'CVE-2014-6271'",
						Link:    "https://shellshock",
					},
				},
				fixtureDep.GetContainers()[1].GetImage().GetId(): {
					{
						Message: "CVE CVE-2014-6271 matched regex 'CVE-2014-6271'",
						Link:    "https://nvd.nist.gov/vuln/detail/CVE-2014-6271",
					},
				},
			},
		},
		{
			policyName: "Apache Struts: CVE-2017-5638",
			expectedViolations: map[string][]*v1.Alert_Violation{
				suite.imageIDFromDep(strutsDep): {
					{
						Message: "CVE CVE-2017-5638 matched regex 'CVE-2017-5638'",
						Link:    "https://struts",
					},
				},
			},
		},
		{
			policyName: "Heartbleed: CVE-2014-0160",
			expectedViolations: map[string][]*v1.Alert_Violation{
				suite.imageIDFromDep(heartbleedDep): {
					{
						Message: "CVE CVE-2014-0160 matched regex 'CVE-2014-0160'",
						Link:    "https://heartbleed",
					},
				},
			},
		},
		{
			policyName: "CVSS >= 7",
			expectedViolations: map[string][]*v1.Alert_Violation{
				suite.imageIDFromDep(strutsDep): {
					{
						Message: "Found a CVSS score of 8 (greater than or equal to 7.0) (cve: CVE-2017-5638)",
					},
				},
			},
		},
		{
			policyName: "ADD Command used instead of COPY",
			expectedViolations: map[string][]*v1.Alert_Violation{
				suite.imageIDFromDep(addDockerFileDep): {
					{
						Message: "Dockerfile Line 'ADD deploy.sh' matches the rule ADD .*",
					},
				},
				fixtureDep.GetContainers()[0].GetImage().GetId(): {
					{
						Message: "Dockerfile Line 'ADD FILE:blah' matches the rule ADD .*",
					},
				},
				fixtureDep.GetContainers()[1].GetImage().GetId(): {
					{
						Message: "Dockerfile Line 'ADD file:4eedf861fb567fffb2694b65ebdd58d5e371a2c28c3863f363f333cb34e5eb7b in /' matches the rule ADD .*",
					},
				},
			},
		},
	}

	for _, c := range imageTestCases {
		p := suite.MustGetPolicy(c.policyName)
		suite.T().Run(fmt.Sprintf("%s (on images)", c.policyName), func(t *testing.T) {
			m, err := ForPolicy(p, imageMappings.OptionsMap, nil)
			require.NoError(t, err)
			matches, err := m.Match(suite.imageIndexer)
			require.NoError(t, err)
			validateImageMatches(matches, allImages, c, t)

			var allIDs []string
			for _, image := range allImages {
				allIDs = append(allIDs, types.NewDigest(image.ID).Digest())
			}
			matchesFromMatchMany, err := m.MatchMany(suite.imageIndexer, allIDs...)
			require.NoError(t, err)
			validateImageMatches(matchesFromMatchMany, allImages, c, t)

			var matchingIDs []string
			for id := range c.expectedViolations {
				matchingIDs = append(matchingIDs, types.NewDigest(id).Digest())
			}
			matchesFromExactlyMatchMany, err := m.MatchMany(suite.imageIndexer, matchingIDs...)
			require.NoError(t, err)
			validateImageMatches(matchesFromExactlyMatchMany, allImages, c, t)

			for id, violations := range c.expectedViolations {
				id = types.NewDigest(id).Digest()
				// Test match one
				gotFromMatchOne, err := m.MatchOne(suite.imageIndexer, id)
				require.NoError(t, err)
				assert.ElementsMatch(t, violations, gotFromMatchOne, "Expected violations from match one %+v don't match what we got %+v", violations, gotFromMatchOne)
			}
		})
	}
}

func validateImageMatches(matches map[string][]*v1.Alert_Violation, allImages []search.Result, c testCase, t *testing.T) {
	if len(c.shouldNotMatch) > 0 {
		assert.Nil(t, c.expectedViolations, "Don't specify expected violations and shouldNotMatch")
		for id := range c.shouldNotMatch {
			id = types.NewDigest(id).Digest()
			_, exists := matches[id]
			assert.False(t, exists, "Should not have matched %s", id)
		}

		for _, imageResult := range allImages {
			id := imageResult.ID
			_, shouldNotMatch := c.shouldNotMatch[id]
			if shouldNotMatch {
				continue
			}
			match, exists := matches[id]
			require.True(t, exists, "Should have matched %s. Got %+v", id, matches)
			if c.sampleViolationForMatched != "" {
				assert.Equal(t, c.sampleViolationForMatched, match[0].GetMessage())
			}
		}
		return
	}

	for id, violations := range c.expectedViolations {
		id = types.NewDigest(id).Digest()
		got, ok := matches[id]
		if !assert.True(t, ok, "Id '%s' didn't match, but should have. Got: %+v", id, matches) {
			continue
		}
		assert.ElementsMatch(t, violations, got, "Expected violations %+v don't match what we got %+v", violations, got)
	}
	assert.Len(t, matches, len(c.expectedViolations))
}

func validateDeploymentMatches(matches map[string][]*v1.Alert_Violation, allDeployments []search.Result, c testCase, t *testing.T) {
	if len(c.shouldNotMatch) > 0 {
		assert.Nil(t, c.expectedViolations, "Don't specify expected violations and shouldNotMatch")
		for id := range c.shouldNotMatch {
			_, exists := matches[id]
			assert.False(t, exists, "Should not have matched %s", id)
		}
		for _, depResult := range allDeployments {
			id := depResult.ID
			_, shouldNotMatch := c.shouldNotMatch[id]
			if shouldNotMatch {
				continue
			}
			match, exists := matches[id]
			require.True(t, exists, "Should have matched %s", id)
			if c.sampleViolationForMatched != "" {
				assert.Equal(t, c.sampleViolationForMatched, match[0].GetMessage())
			}
		}
		return
	}

	for id, violations := range c.expectedViolations {
		got, ok := matches[id]
		if !assert.True(t, ok, "Id '%s' didn't match, but should have. Got: %+v", id, matches) {
			continue
		}
		assert.ElementsMatch(t, violations, got, "Expected violations %+v don't match what we got %+v", violations, got)

	}
	assert.Len(t, matches, len(c.expectedViolations))

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
		}
	}
}
