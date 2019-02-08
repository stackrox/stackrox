package matcher

import (
	"fmt"
	"os"
	"regexp"
	"testing"
	"time"

	"github.com/blevesearch/bleve"
	bolt "github.com/etcd-io/bbolt"
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
	"github.com/stackrox/rox/central/searchbasedpolicies"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/image/policies"
	"github.com/stackrox/rox/pkg/bolthelper"
	"github.com/stackrox/rox/pkg/defaults"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/images/types"
	policyUtils "github.com/stackrox/rox/pkg/policies"
	"github.com/stackrox/rox/pkg/protoconv"
	"github.com/stackrox/rox/pkg/readable"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/set"
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

	defaultPolicies map[string]*storage.Policy
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
	suite.processDataStore = processIndicatorDataStore.New(processStore, processIndexer, processSearcher, nil)

	defaults.PoliciesPath = policies.Directory()

	defaultPolicies, err := defaults.Policies()
	suite.Require().NoError(err)

	suite.defaultPolicies = make(map[string]*storage.Policy, len(defaultPolicies))
	for _, p := range defaultPolicies {
		suite.defaultPolicies[p.GetName()] = p
	}
}

func (suite *DefaultPoliciesTestSuite) TearDownTest() {
	suite.bleveIndex.Close()
	suite.db.Close()
	os.Remove(suite.db.Path())
}

func (suite *DefaultPoliciesTestSuite) TestNoDuplicatePolicyIDs() {
	ids := set.NewStringSet()
	for _, p := range suite.defaultPolicies {
		suite.True(ids.Add(p.GetId()))
	}
}

func (suite *DefaultPoliciesTestSuite) MustGetPolicy(name string) *storage.Policy {
	p, ok := suite.defaultPolicies[name]
	suite.Require().True(ok, "Policy %s not found", name)
	return p
}

func (suite *DefaultPoliciesTestSuite) mustIndexDepAndImages(deployment *storage.Deployment) {
	suite.NoError(suite.deploymentIndexer.AddDeployment(deployment))
	for _, container := range deployment.GetContainers() {
		if container.GetImage() != nil {
			suite.NoError(suite.imageIndexer.AddImage(container.GetImage()))
		}
	}
}

func wrapAlertViolations(slice []*storage.Alert_Violation) searchbasedpolicies.Violations {
	return searchbasedpolicies.Violations{AlertViolations: slice}
}

func imageWithComponents(components []*storage.ImageScanComponent) *storage.Image {
	return &storage.Image{
		Id:   uuid.NewV4().String(),
		Name: &storage.ImageName{FullName: "ASFASF"},
		Scan: &storage.ImageScan{
			Components: components,
		},
	}
}

func imageWithLayers(layers []*storage.ImageLayer) *storage.Image {
	return &storage.Image{
		Id: uuid.NewV4().String(),
		Metadata: &storage.ImageMetadata{
			V1: &storage.V1Metadata{
				Layers: layers,
			},
		},
	}
}

func deploymentWithImage(img *storage.Image) *storage.Deployment {
	return &storage.Deployment{
		Id:         uuid.NewV4().String(),
		Containers: []*storage.Container{{Image: img}},
	}
}

func deploymentWithComponents(components []*storage.ImageScanComponent) *storage.Deployment {
	return deploymentWithImage(imageWithComponents(components))
}

func deploymentWithLayers(layers []*storage.ImageLayer) *storage.Deployment {
	return deploymentWithImage(imageWithLayers(layers))
}

func (suite *DefaultPoliciesTestSuite) imageIDFromDep(deployment *storage.Deployment) string {
	suite.Require().Len(deployment.GetContainers(), 1, "This function only supports deployments with exactly one container")
	id := deployment.GetContainers()[0].GetImage().GetId()
	suite.NotEmpty(id, "Deployment '%s' had no image id", proto.MarshalTextString(deployment))
	return types.NewDigest(id).Digest()
}

func (suite *DefaultPoliciesTestSuite) mustAddIndicator(deploymentID, name, args, path string, lineage []string) *storage.ProcessIndicator {
	indicator := &storage.ProcessIndicator{
		Id:           uuid.NewV4().String(),
		DeploymentId: deploymentID,
		Signal: &storage.ProcessSignal{
			Name:         name,
			Args:         args,
			ExecFilePath: path,
			Time:         gogoTypes.TimestampNow(),
			Lineage:      lineage,
		},
	}
	err := suite.processDataStore.AddProcessIndicator(indicator)
	suite.NoError(err)
	return indicator
}

type testCase struct {
	policyName         string
	policy             *storage.Policy
	expectedViolations map[string]searchbasedpolicies.Violations

	// If shouldNotMatch is specified (which is the case for policies that check for the absence of something), we verify that
	// it matches everything except shouldNotMatch.
	// If sampleViolationForMatched is provided, we verify that all the matches are the string provided in sampleViolationForMatched.
	shouldNotMatch            map[string]struct{}
	sampleViolationForMatched string
}

func (suite *DefaultPoliciesTestSuite) TestDefaultPolicies() {
	fixtureDep := fixtures.GetDeployment()
	suite.mustIndexDepAndImages(fixtureDep)

	nginx110 := &storage.Image{
		Id: "SHANGINX110",
		Name: &storage.ImageName{
			Registry: "docker.io",
			Remote:   "library/nginx",
			Tag:      "1.10",
		},
	}
	nginx110Dep := &storage.Deployment{
		Id: "nginx110",
		Containers: []*storage.Container{
			{Image: nginx110},
		},
	}
	suite.mustIndexDepAndImages(nginx110Dep)

	oldScannedTime := time.Now().Add(-31 * 24 * time.Hour)
	oldScannedImage := &storage.Image{
		Id: "SHAOLDSCANNED",
		Scan: &storage.ImageScan{
			ScanTime: protoconv.ConvertTimeToTimestamp(oldScannedTime),
		},
	}
	oldScannedDep := &storage.Deployment{
		Id: "oldscanned",
		Containers: []*storage.Container{
			{Image: oldScannedImage},
		},
	}
	suite.mustIndexDepAndImages(oldScannedDep)

	addDockerFileDep := deploymentWithLayers([]*storage.ImageLayer{
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

	imagePort22Dep := deploymentWithLayers([]*storage.ImageLayer{
		{
			Instruction: "EXPOSE",
			Value:       "22/tcp",
		},
	})
	suite.mustIndexDepAndImages(imagePort22Dep)

	insecureCMDDep := deploymentWithLayers([]*storage.ImageLayer{
		{
			Instruction: "CMD",
			Value:       "do an insecure thing",
		},
	})
	suite.mustIndexDepAndImages(insecureCMDDep)

	runSecretsDep := deploymentWithLayers([]*storage.ImageLayer{
		{
			Instruction: "VOLUME",
			Value:       "/run/secrets",
		},
	})
	suite.mustIndexDepAndImages(runSecretsDep)

	oldImageCreationTime := time.Now().Add(-100 * 24 * time.Hour)
	oldCreatedImage := &storage.Image{
		Id: "SHA:OLDCREATEDIMAGE",
		Metadata: &storage.ImageMetadata{
			V1: &storage.V1Metadata{
				Created: protoconv.ConvertTimeToTimestamp(oldImageCreationTime),
			},
		},
	}
	oldImageDep := &storage.Deployment{
		Id:         "oldimagedep",
		Containers: []*storage.Container{{Image: oldCreatedImage}},
	}
	suite.mustIndexDepAndImages(oldImageDep)

	apkDep := deploymentWithComponents([]*storage.ImageScanComponent{
		{Name: "apk", Version: "1.2"},
		{Name: "asfa", Version: "1.5"},
	})
	suite.mustIndexDepAndImages(apkDep)

	curlDep := deploymentWithComponents([]*storage.ImageScanComponent{
		{Name: "curl", Version: "1.3"},
		{Name: "curlwithextra", Version: "0.9"},
	})
	suite.mustIndexDepAndImages(curlDep)

	componentDeps := make(map[string]*storage.Deployment)
	for _, component := range []string{"apt", "dnf", "wget"} {
		dep := deploymentWithComponents([]*storage.ImageScanComponent{
			{Name: component},
		})
		suite.mustIndexDepAndImages(dep)
		componentDeps[component] = dep
	}

	heartbleedDep := &storage.Deployment{
		Id: "HEARTBLEEDDEPID",
		Containers: []*storage.Container{
			{
				SecurityContext: &storage.SecurityContext{Privileged: true},
				Image: &storage.Image{
					Id: "HEARTBLEEDDEPSHA",
					Scan: &storage.ImageScan{
						Components: []*storage.ImageScanComponent{
							{Name: "heartbleed", Version: "1.2", Vulns: []*storage.Vulnerability{
								{Cve: "CVE-2014-0160", Link: "https://heartbleed", Cvss: 6},
							}},
						},
					},
				},
			},
		},
	}
	suite.mustIndexDepAndImages(heartbleedDep)

	shellshockDep := deploymentWithComponents([]*storage.ImageScanComponent{
		{Name: "shellshock", Version: "1.2", Vulns: []*storage.Vulnerability{
			{Cve: "CVE-2014-6271", Link: "https://shellshock", Cvss: 6},
			{Cve: "CVE-ARBITRARY", Link: "https://notshellshock"},
		}},
	})
	suite.mustIndexDepAndImages(shellshockDep)

	strutsDep := deploymentWithComponents([]*storage.ImageScanComponent{
		{Name: "struts", Version: "1.2", Vulns: []*storage.Vulnerability{
			{Cve: "CVE-2017-5638", Link: "https://struts", Cvss: 8},
		}},
		{Name: "OTHER", Version: "1.3", Vulns: []*storage.Vulnerability{
			{Cve: "CVE-1223-451", Link: "https://cvefake"},
		}},
	})
	suite.mustIndexDepAndImages(strutsDep)

	depWithNonSeriousVulns := deploymentWithComponents([]*storage.ImageScanComponent{
		{Name: "NOSERIOUS", Version: "2.3", Vulns: []*storage.Vulnerability{
			{Cve: "CVE-1234-5678", Link: "https://abcdefgh"},
			{Cve: "CVE-5678-1234", Link: "https://lmnopqrst"},
		}},
	})
	suite.mustIndexDepAndImages(depWithNonSeriousVulns)

	dockerSockDep := &storage.Deployment{
		Id: "DOCKERSOCDEP",
		Containers: []*storage.Container{
			{Volumes: []*storage.Volume{
				{Source: "/var/run/docker.sock", Name: "DOCKERSOCK"},
				{Source: "NOTDOCKERSOCK"},
			}},
		},
	}
	suite.mustIndexDepAndImages(dockerSockDep)

	containerPort22Dep := &storage.Deployment{
		Id: "CONTAINERPORT22DEP",
		Ports: []*storage.PortConfig{
			{Protocol: "tcp", ContainerPort: 22},
			{Protocol: "udp", ContainerPort: 4125},
		},
	}
	suite.mustIndexDepAndImages(containerPort22Dep)

	secretEnvDep := &storage.Deployment{
		Id: "SECRETENVDEP",
		Containers: []*storage.Container{
			{Config: &storage.ContainerConfig{
				Env: []*storage.ContainerConfig_EnvironmentConfig{
					{Key: "THIS_IS_SECRET_VAR", Value: "stealthmode"},
					{Key: "HOME", Value: "/home/stackrox"},
				},
			}},
		},
	}
	suite.mustIndexDepAndImages(secretEnvDep)

	// Fake deployment that shouldn't match anything, just to make sure
	// that none of our queries will accidentally match it.
	suite.mustIndexDepAndImages(&storage.Deployment{Id: "FAKEID", Name: "FAKENAME"})

	depWithGoodEmailAnnotation := &storage.Deployment{
		Id: "GOODEMAILDEPID",
		Annotations: map[string]string{
			"email": "vv@stackrox.com",
		},
	}
	suite.mustIndexDepAndImages(depWithGoodEmailAnnotation)

	depWithOwnerAnnotation := &storage.Deployment{
		Id: "OWNERANNOTATIONDEP",
		Annotations: map[string]string{
			"owner": "IOWNTHIS",
			"blah":  "Blah",
		},
	}
	suite.mustIndexDepAndImages(depWithOwnerAnnotation)

	depWitharbitraryAnnotations := &storage.Deployment{
		Id: "ARBITRARYANNOTATIONDEPID",
		Annotations: map[string]string{
			"emailnot": "vv@stackrox.com",
			"notemail": "vv@stackrox.com",
			"ownernot": "vv",
			"nowner":   "vv",
		},
	}
	suite.mustIndexDepAndImages(depWitharbitraryAnnotations)

	depWithBadEmailAnnotation := &storage.Deployment{
		Id: "BADEMAILDEPID",
		Annotations: map[string]string{
			"email": "NOTANEMAIL",
		},
	}
	suite.mustIndexDepAndImages(depWithBadEmailAnnotation)

	sysAdminDep := &storage.Deployment{
		Id: "SYSADMINDEPID",
		Containers: []*storage.Container{
			{
				SecurityContext: &storage.SecurityContext{
					AddCapabilities: []string{"CAP_SYS_ADMIN"},
				},
			},
		},
	}
	suite.mustIndexDepAndImages(sysAdminDep)

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
	suite.mustIndexDepAndImages(depWithAllResourceLimitsRequestsSpecified)

	depWithEnforcementBypassAnnotation := &storage.Deployment{
		Id: "ENFORCEMENTBYPASS",
		Annotations: map[string]string{
			"admission.stackrox.io/break-glass": "ticket-1234",
		},
	}
	suite.mustIndexDepAndImages(depWithEnforcementBypassAnnotation)

	hostMountDep := &storage.Deployment{
		Id: "HOSTMOUNT",
		Containers: []*storage.Container{
			{Volumes: []*storage.Volume{
				{Source: "/etc/passwd", Name: "HOSTMOUNT"},
				{Source: "/var/lib/kubelet", Name: "KUBELET"},
			}},
		},
	}
	suite.mustIndexDepAndImages(hostMountDep)

	// Index processes
	bashLineage := []string{"/bin/bash"}
	fixtureDepAptIndicator := suite.mustAddIndicator(fixtureDep.GetId(), "apt", "", "/usr/bin/apt", bashLineage)
	sysAdminDepAptIndicator := suite.mustAddIndicator(sysAdminDep.GetId(), "apt", "install blah", "/usr/bin/apt", bashLineage)

	kubeletIndicator := suite.mustAddIndicator(containerPort22Dep.GetId(), "curl", "https://12.13.14.15:10250", "/bin/curl", bashLineage)
	kubeletIndicator2 := suite.mustAddIndicator(containerPort22Dep.GetId(), "wget", "https://heapster.kube-system/metrics", "/bin/wget", bashLineage)

	nmapIndicatorfixtureDep1 := suite.mustAddIndicator(fixtureDep.GetId(), "nmap", "blah", "/usr/bin/nmap", bashLineage)
	nmapIndicatorfixtureDep2 := suite.mustAddIndicator(fixtureDep.GetId(), "nmap", "blah2", "/usr/bin/nmap", bashLineage)
	nmapIndicatorNginx110Dep := suite.mustAddIndicator(nginx110Dep.GetId(), "nmap", "", "/usr/bin/nmap", bashLineage)

	javaLineage := []string{"/bin/bash", "/mnt/scripts/run_server.sh", "/bin/java"}
	fixtureDepJavaIndicator := suite.mustAddIndicator(fixtureDep.GetId(), "/bin/bash", "-attack", "/bin/bash", javaLineage)

	// Find all the deployments indexed.
	allDeployments, err := suite.deploymentIndexer.Search(search.EmptyQuery())
	suite.NoError(err)

	allImages, err := suite.imageIndexer.Search(search.EmptyQuery())
	suite.NoError(err)

	deploymentTestCases := []testCase{
		{
			policyName: "Latest tag",
			expectedViolations: map[string]searchbasedpolicies.Violations{
				fixtureDep.GetId(): {AlertViolations: []*storage.Alert_Violation{
					{
						Message: "Image tag 'latest' matched latest",
					},
				},
				},
			},
		},
		{
			policyName: "DockerHub NGINX 1.10",
			expectedViolations: map[string]searchbasedpolicies.Violations{
				fixtureDep.GetId(): {AlertViolations: []*storage.Alert_Violation{
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
				nginx110Dep.GetId(): {AlertViolations: []*storage.Alert_Violation{
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
		},
		{
			policyName: "Alpine Linux Package Manager (apk) in Image",
			expectedViolations: map[string]searchbasedpolicies.Violations{
				apkDep.GetId(): {AlertViolations: []*storage.Alert_Violation{
					{
						Message: "Component name 'apk' matched apk",
					},
				},
				},
			},
		},
		{
			policyName: "Ubuntu Package Manager in Image",
			expectedViolations: map[string]searchbasedpolicies.Violations{
				componentDeps["apt"].GetId(): {AlertViolations: []*storage.Alert_Violation{
					{
						Message: "Component name 'apt' matched apt|dpkg",
					},
				},
				},
			},
		},
		{
			policyName: "Curl in Image",
			expectedViolations: map[string]searchbasedpolicies.Violations{
				curlDep.GetId(): {AlertViolations: []*storage.Alert_Violation{
					{
						Message: "Component name 'curl' matched curl",
					},
				},
				},
			},
		},
		{
			policyName: "Red Hat Package Manager in Image",
			expectedViolations: map[string]searchbasedpolicies.Violations{
				componentDeps["dnf"].GetId(): {AlertViolations: []*storage.Alert_Violation{
					{
						Message: "Component name 'dnf' matched rpm|dnf|yum",
					},
				},
				},
			},
		},
		{
			policyName: "Wget in Image",
			expectedViolations: map[string]searchbasedpolicies.Violations{
				componentDeps["wget"].GetId(): {AlertViolations: []*storage.Alert_Violation{
					{
						Message: "Component name 'wget' matched wget",
					},
				},
				},
			},
		},
		{
			policyName: "Mount Docker Socket",
			expectedViolations: map[string]searchbasedpolicies.Violations{
				dockerSockDep.GetId(): {AlertViolations: []*storage.Alert_Violation{
					{
						Message: "Volume source '/var/run/docker.sock' matched /var/run/docker.sock",
					},
				},
				},
			},
		},
		{
			policyName: "90-Day Image Age",
			expectedViolations: map[string]searchbasedpolicies.Violations{
				oldImageDep.GetId(): {AlertViolations: []*storage.Alert_Violation{
					{
						Message: fmt.Sprintf("Time of image creation '%s' was more than 90 days ago", readable.Time(oldImageCreationTime)),
					},
				},
				},
			},
		},
		{
			policyName: "30-Day Scan Age",
			expectedViolations: map[string]searchbasedpolicies.Violations{
				oldScannedDep.GetId(): {AlertViolations: []*storage.Alert_Violation{
					{
						Message: fmt.Sprintf("Time of last scan '%s' was more than 30 days ago", readable.Time(oldScannedTime)),
					},
				},
				},
			},
		},
		{
			policyName: "Secure Shell (ssh) Port Exposed in Image",
			expectedViolations: map[string]searchbasedpolicies.Violations{
				imagePort22Dep.GetId(): {AlertViolations: []*storage.Alert_Violation{
					{
						Message: "Dockerfile Line 'EXPOSE 22/tcp' matches the rule EXPOSE (22/tcp|\\s+22/tcp)",
					},
				},
				},
			},
		},
		{
			policyName: "Secure Shell (ssh) Port Exposed",
			expectedViolations: map[string]searchbasedpolicies.Violations{
				containerPort22Dep.GetId(): {AlertViolations: []*storage.Alert_Violation{
					{
						Message: "Port '22' matched 22",
					},
					{
						Message: "Protocol 'tcp' matched tcp",
					},
				},
				},
			},
		},
		{
			policyName: "Privileged Container",
			expectedViolations: map[string]searchbasedpolicies.Violations{
				fixtureDep.GetId(): {AlertViolations: []*storage.Alert_Violation{
					{
						Message: "Privileged container found",
					},
				},
				},
				heartbleedDep.GetId(): {AlertViolations: []*storage.Alert_Violation{
					{
						Message: "Privileged container found",
					},
				},
				},
			},
		},
		{
			policyName: "Insecure specified in CMD",
			expectedViolations: map[string]searchbasedpolicies.Violations{
				insecureCMDDep.GetId(): {AlertViolations: []*storage.Alert_Violation{
					{
						Message: "Dockerfile Line 'CMD do an insecure thing' matches the rule CMD .*insecure.*",
					},
				},
				},
			},
		},
		{
			policyName: "Improper Usage of Orchestrator Secrets Volume",
			expectedViolations: map[string]searchbasedpolicies.Violations{
				runSecretsDep.GetId(): {AlertViolations: []*storage.Alert_Violation{
					{
						Message: "Dockerfile Line 'VOLUME /run/secrets' matches the rule VOLUME /run/secrets",
					},
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
				depWithEnforcementBypassAnnotation.GetId():        {},
				hostMountDep.GetId():                              {},
			},
			sampleViolationForMatched: "Image has not been scanned",
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
			expectedViolations: map[string]searchbasedpolicies.Violations{
				sysAdminDep.GetId(): {AlertViolations: []*storage.Alert_Violation{
					{
						Message: "CAP_SYS_ADMIN was in the ADD CAPABILITIES list",
					},
				},
				},
			},
		},
		{
			policyName: "Shellshock: CVE-2014-6271",
			expectedViolations: map[string]searchbasedpolicies.Violations{
				shellshockDep.GetId(): {AlertViolations: []*storage.Alert_Violation{
					{
						Message: "CVE CVE-2014-6271 matched regex 'CVE-2014-6271'",
						Link:    "https://shellshock",
					},
				},
				},
				fixtureDep.GetId(): {AlertViolations: []*storage.Alert_Violation{
					{
						Message: "CVE CVE-2014-6271 matched regex 'CVE-2014-6271'",
						Link:    "https://nvd.nist.gov/vuln/detail/CVE-2014-6271",
					},
				},
				},
			},
		},
		{
			policyName: "Apache Struts: CVE-2017-5638",
			expectedViolations: map[string]searchbasedpolicies.Violations{
				strutsDep.GetId(): {AlertViolations: []*storage.Alert_Violation{
					{
						Message: "CVE CVE-2017-5638 matched regex 'CVE-2017-5638'",
						Link:    "https://struts",
					},
				},
				},
			},
		},
		{
			policyName: "Heartbleed: CVE-2014-0160",
			expectedViolations: map[string]searchbasedpolicies.Violations{
				heartbleedDep.GetId(): {AlertViolations: []*storage.Alert_Violation{
					{
						Message: "CVE CVE-2014-0160 matched regex 'CVE-2014-0160'",
						Link:    "https://heartbleed",
					},
				},
				},
			},
		},
		{
			policyName: "No resource requests or limits specified",
			expectedViolations: map[string]searchbasedpolicies.Violations{
				fixtureDep.GetId(): {AlertViolations: []*storage.Alert_Violation{
					{Message: "The CPU resource limit of 0 is equal to the threshold of 0.00"},
					{Message: "The memory resource limit of 0 is equal to the threshold of 0.00"},
					{Message: "The memory resource request of 0 is equal to the threshold of 0.00"},
				},
				},
			},
		},
		{
			policyName: "Environment Variable Contains Secret",
			expectedViolations: map[string]searchbasedpolicies.Violations{
				secretEnvDep.GetId(): {AlertViolations: []*storage.Alert_Violation{
					{
						Message: "Container Environment (key='THIS_IS_SECRET_VAR', value='stealthmode') matched environment policy (key = '.*SECRET.*', value = '.*')",
					},
				},
				},
			},
		},
		{
			policyName: "CVSS >= 6 and Privileged",
			expectedViolations: map[string]searchbasedpolicies.Violations{
				heartbleedDep.GetId(): {AlertViolations: []*storage.Alert_Violation{
					{
						Message: "Found a CVSS score of 6 (greater than or equal to 6.0) (cve: CVE-2014-0160)",
					},
					{
						Message: "Privileged container found",
					},
				},
				},
			},
		},
		{
			policyName: "CVSS >= 7",
			expectedViolations: map[string]searchbasedpolicies.Violations{
				strutsDep.GetId(): {AlertViolations: []*storage.Alert_Violation{
					{
						Message: "Found a CVSS score of 8 (greater than or equal to 7.0) (cve: CVE-2017-5638)",
					},
				},
				},
			},
		},
		{
			policyName: "ADD Command used instead of COPY",
			expectedViolations: map[string]searchbasedpolicies.Violations{
				addDockerFileDep.GetId(): {AlertViolations: []*storage.Alert_Violation{
					{
						Message: "Dockerfile Line 'ADD deploy.sh' matches the rule ADD .*",
					},
				},
				},
				fixtureDep.GetId(): {AlertViolations: []*storage.Alert_Violation{
					{
						Message: "Dockerfile Line 'ADD FILE:blah' matches the rule ADD .*",
					},
					{
						Message: "Dockerfile Line 'ADD file:4eedf861fb567fffb2694b65ebdd58d5e371a2c28c3863f363f333cb34e5eb7b in /' matches the rule ADD .*",
					},
				},
				},
			},
		},
		{
			policyName: "nmap Execution",
			expectedViolations: map[string]searchbasedpolicies.Violations{
				fixtureDep.GetId(): {ProcessViolation: &storage.Alert_ProcessViolation{
					Message:   "Detected executions of binary '/usr/bin/nmap' with 2 different arguments",
					Processes: []*storage.ProcessIndicator{nmapIndicatorfixtureDep1, nmapIndicatorfixtureDep2},
				},
				},
				nginx110Dep.GetId(): {ProcessViolation: &storage.Alert_ProcessViolation{
					Message:   "Detected execution of binary '/usr/bin/nmap'",
					Processes: []*storage.ProcessIndicator{nmapIndicatorNginx110Dep},
				},
				},
			},
		},
		{
			policyName: "Process Targeting Cluster Kubelet Endpoint",
			expectedViolations: map[string]searchbasedpolicies.Violations{
				containerPort22Dep.GetId(): {ProcessViolation: &storage.Alert_ProcessViolation{
					Message:   "Detected executions of 2 binaries with 2 different arguments",
					Processes: []*storage.ProcessIndicator{kubeletIndicator, kubeletIndicator2},
				},
				},
			},
		},
		{
			policyName: "Ubuntu Package Manager Execution",
			expectedViolations: map[string]searchbasedpolicies.Violations{
				fixtureDep.GetId(): {ProcessViolation: &storage.Alert_ProcessViolation{
					Message:   "Detected execution of binary '/usr/bin/apt'",
					Processes: []*storage.ProcessIndicator{fixtureDepAptIndicator},
				},
				},
				sysAdminDep.GetId(): {ProcessViolation: &storage.Alert_ProcessViolation{
					Message:   "Detected execution of binary '/usr/bin/apt' with arguments 'install blah'",
					Processes: []*storage.ProcessIndicator{sysAdminDepAptIndicator},
				},
				},
			},
		},
		{
			policyName: "Shell Spawned by Java Application",
			expectedViolations: map[string]searchbasedpolicies.Violations{
				fixtureDep.GetId(): {ProcessViolation: &storage.Alert_ProcessViolation{
					Message:   "Detected execution of binary '/bin/bash' with arguments '-attack'",
					Processes: []*storage.ProcessIndicator{fixtureDepJavaIndicator},
				},
				},
			},
		},
		{
			policyName: "Emergency Deployment Annotation",
			expectedViolations: map[string]searchbasedpolicies.Violations{
				depWithEnforcementBypassAnnotation.GetId(): {AlertViolations: []*storage.Alert_Violation{{
					Message: "Disallowed annotation found (key = 'admission.stackrox.io/break-glass')",
				},
				},
				},
			},
		},
		{
			policyName: "Mounting Sensitive Host Directories",
			expectedViolations: map[string]searchbasedpolicies.Violations{
				hostMountDep.GetId(): {AlertViolations: []*storage.Alert_Violation{
					{Message: "Volume source '/etc/passwd' matched (/etc/.*|/sys/.*|/dev/.*|/proc/.*|/var/.*)"},
					{Message: "Volume source '/var/lib/kubelet' matched (/etc/.*|/sys/.*|/dev/.*|/proc/.*|/var/.*)"},
				},
				},
				dockerSockDep.GetId(): {AlertViolations: []*storage.Alert_Violation{
					{Message: "Volume source '/var/run/docker.sock' matched (/etc/.*|/sys/.*|/dev/.*|/proc/.*|/var/.*)"},
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
				assert.ElementsMatch(t, violations.AlertViolations, gotFromMatchOne.AlertViolations, "Expected violations from match one %+v don't match what we got %+v", violations, gotFromMatchOne)
				assert.Equal(t, violations.ProcessViolation, gotFromMatchOne.ProcessViolation)
			}
		})
	}

	imageTestCases := []testCase{
		{
			policyName: "Latest tag",
			expectedViolations: map[string]searchbasedpolicies.Violations{
				fixtureDep.GetContainers()[1].GetImage().GetId(): {AlertViolations: []*storage.Alert_Violation{
					{Message: "Image tag 'latest' matched latest"},
				},
				},
			},
		},
		{
			policyName: "DockerHub NGINX 1.10",
			expectedViolations: map[string]searchbasedpolicies.Violations{
				fixtureDep.GetContainers()[0].GetImage().GetId(): {AlertViolations: []*storage.Alert_Violation{
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
				suite.imageIDFromDep(nginx110Dep): {AlertViolations: []*storage.Alert_Violation{
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
		},
		{
			policyName: "Alpine Linux Package Manager (apk) in Image",
			expectedViolations: map[string]searchbasedpolicies.Violations{
				suite.imageIDFromDep(apkDep): {AlertViolations: []*storage.Alert_Violation{
					{
						Message: "Component name 'apk' matched apk",
					},
				},
				},
			},
		},
		{
			policyName: "Ubuntu Package Manager in Image",
			expectedViolations: map[string]searchbasedpolicies.Violations{
				suite.imageIDFromDep(componentDeps["apt"]): {AlertViolations: []*storage.Alert_Violation{
					{
						Message: "Component name 'apt' matched apt|dpkg",
					},
				},
				},
			},
		},
		{
			policyName: "Curl in Image",
			expectedViolations: map[string]searchbasedpolicies.Violations{
				suite.imageIDFromDep(curlDep): {AlertViolations: []*storage.Alert_Violation{
					{
						Message: "Component name 'curl' matched curl",
					},
				},
				},
			},
		},
		{
			policyName: "Red Hat Package Manager in Image",
			expectedViolations: map[string]searchbasedpolicies.Violations{
				suite.imageIDFromDep(componentDeps["dnf"]): {AlertViolations: []*storage.Alert_Violation{
					{
						Message: "Component name 'dnf' matched rpm|dnf|yum",
					},
				},
				},
			},
		},
		{
			policyName: "Wget in Image",
			expectedViolations: map[string]searchbasedpolicies.Violations{
				suite.imageIDFromDep(componentDeps["wget"]): {AlertViolations: []*storage.Alert_Violation{
					{
						Message: "Component name 'wget' matched wget",
					},
				},
				},
			},
		},
		{
			policyName: "90-Day Image Age",
			expectedViolations: map[string]searchbasedpolicies.Violations{
				suite.imageIDFromDep(oldImageDep): {AlertViolations: []*storage.Alert_Violation{
					{
						Message: fmt.Sprintf("Time of image creation '%s' was more than 90 days ago", readable.Time(oldImageCreationTime)),
					},
				},
				},
			},
		},
		{
			policyName: "30-Day Scan Age",
			expectedViolations: map[string]searchbasedpolicies.Violations{
				suite.imageIDFromDep(oldScannedDep): {AlertViolations: []*storage.Alert_Violation{
					{
						Message: fmt.Sprintf("Time of last scan '%s' was more than 30 days ago", readable.Time(oldScannedTime)),
					},
				},
				},
			},
		},
		{
			policyName: "Secure Shell (ssh) Port Exposed in Image",
			expectedViolations: map[string]searchbasedpolicies.Violations{
				suite.imageIDFromDep(imagePort22Dep): {AlertViolations: []*storage.Alert_Violation{
					{
						Message: "Dockerfile Line 'EXPOSE 22/tcp' matches the rule EXPOSE (22/tcp|\\s+22/tcp)",
					},
				},
				},
			},
		},
		{
			policyName: "Insecure specified in CMD",
			expectedViolations: map[string]searchbasedpolicies.Violations{
				suite.imageIDFromDep(insecureCMDDep): {AlertViolations: []*storage.Alert_Violation{
					{
						Message: "Dockerfile Line 'CMD do an insecure thing' matches the rule CMD .*insecure.*",
					},
				},
				},
			},
		},
		{
			policyName: "Improper Usage of Orchestrator Secrets Volume",
			expectedViolations: map[string]searchbasedpolicies.Violations{
				suite.imageIDFromDep(runSecretsDep): {AlertViolations: []*storage.Alert_Violation{
					{
						Message: "Dockerfile Line 'VOLUME /run/secrets' matches the rule VOLUME /run/secrets",
					},
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
			policyName: "Shellshock: CVE-2014-6271",
			expectedViolations: map[string]searchbasedpolicies.Violations{
				suite.imageIDFromDep(shellshockDep): {AlertViolations: []*storage.Alert_Violation{
					{
						Message: "CVE CVE-2014-6271 matched regex 'CVE-2014-6271'",
						Link:    "https://shellshock",
					},
				},
				},
				fixtureDep.GetContainers()[1].GetImage().GetId(): {AlertViolations: []*storage.Alert_Violation{
					{
						Message: "CVE CVE-2014-6271 matched regex 'CVE-2014-6271'",
						Link:    "https://nvd.nist.gov/vuln/detail/CVE-2014-6271",
					},
				},
				},
			},
		},
		{
			policyName: "Apache Struts: CVE-2017-5638",
			expectedViolations: map[string]searchbasedpolicies.Violations{
				suite.imageIDFromDep(strutsDep): {AlertViolations: []*storage.Alert_Violation{
					{
						Message: "CVE CVE-2017-5638 matched regex 'CVE-2017-5638'",
						Link:    "https://struts",
					},
				},
				},
			},
		},
		{
			policyName: "Heartbleed: CVE-2014-0160",
			expectedViolations: map[string]searchbasedpolicies.Violations{
				suite.imageIDFromDep(heartbleedDep): {AlertViolations: []*storage.Alert_Violation{
					{
						Message: "CVE CVE-2014-0160 matched regex 'CVE-2014-0160'",
						Link:    "https://heartbleed",
					},
				},
				},
			},
		},
		{
			policyName: "CVSS >= 7",
			expectedViolations: map[string]searchbasedpolicies.Violations{
				suite.imageIDFromDep(strutsDep): {AlertViolations: []*storage.Alert_Violation{
					{
						Message: "Found a CVSS score of 8 (greater than or equal to 7.0) (cve: CVE-2017-5638)",
					},
				},
				},
			},
		},
		{
			policyName: "ADD Command used instead of COPY",
			expectedViolations: map[string]searchbasedpolicies.Violations{
				suite.imageIDFromDep(addDockerFileDep): {AlertViolations: []*storage.Alert_Violation{
					{
						Message: "Dockerfile Line 'ADD deploy.sh' matches the rule ADD .*",
					},
				},
				},
				fixtureDep.GetContainers()[0].GetImage().GetId(): {AlertViolations: []*storage.Alert_Violation{
					{
						Message: "Dockerfile Line 'ADD FILE:blah' matches the rule ADD .*",
					},
				},
				},
				fixtureDep.GetContainers()[1].GetImage().GetId(): {AlertViolations: []*storage.Alert_Violation{
					{
						Message: "Dockerfile Line 'ADD file:4eedf861fb567fffb2694b65ebdd58d5e371a2c28c3863f363f333cb34e5eb7b in /' matches the rule ADD .*",
					},
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
				assert.ElementsMatch(t, violations.AlertViolations, gotFromMatchOne.AlertViolations, "Expected violations from match one %+v don't match what we got %+v", violations, gotFromMatchOne)
			}
		})
	}
}

func validateImageMatches(matches map[string]searchbasedpolicies.Violations, allImages []search.Result, c testCase, t *testing.T) {
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
				assert.Equal(t, c.sampleViolationForMatched, match.AlertViolations[0].GetMessage())
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
		assert.ElementsMatch(t, violations.AlertViolations, got.AlertViolations, "Expected violations %+v don't match what we got %+v", violations, got)
		assert.Equal(t, violations.ProcessViolation, got.ProcessViolation, "Expected violations %+v don't match what we got %+v", violations, got)
	}
	assert.Len(t, matches, len(c.expectedViolations))
}

func validateDeploymentMatches(matches map[string]searchbasedpolicies.Violations, allDeployments []search.Result, c testCase, t *testing.T) {
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
				assert.Equal(t, c.sampleViolationForMatched, match.AlertViolations[0].GetMessage())
			}
		}
		return
	}

	for id, violations := range c.expectedViolations {
		got, ok := matches[id]
		if !assert.True(t, ok, "Id '%s' didn't match, but should have. Got: %+v", id, matches) {
			continue
		}
		assert.ElementsMatch(t, violations.AlertViolations, got.AlertViolations, "Expected violations %+v don't match what we got %+v", violations, got)
		assert.Equal(t, violations.ProcessViolation, got.ProcessViolation)

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
			if processPolicy.GetAncestor() != "" {
				regexp.MustCompile(processPolicy.GetAncestor())
			}
		}
	}
}
