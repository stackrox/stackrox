package matcher

import (
	"fmt"
	"regexp"
	"testing"
	"time"

	"github.com/blevesearch/bleve"
	deploymentIndex "github.com/stackrox/rox/central/deployment/index"
	"github.com/stackrox/rox/central/deployment/index/mappings"
	"github.com/stackrox/rox/central/globalindex"
	imageIndex "github.com/stackrox/rox/central/image/index"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/image/policies"
	"github.com/stackrox/rox/pkg/defaults"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/protoconv"
	"github.com/stackrox/rox/pkg/readable"
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

	bleveIndex        bleve.Index
	deploymentIndexer deploymentIndex.Indexer
	imageIndexer      imageIndex.Indexer

	defaultPolicies map[string]*v1.Policy
}

func (suite *DefaultPoliciesTestSuite) SetupTest() {
	var err error
	suite.bleveIndex, err = globalindex.TempInitializeIndices("")
	suite.Require().NoError(err)

	suite.deploymentIndexer = deploymentIndex.New(suite.bleveIndex)
	suite.imageIndexer = imageIndex.New(suite.bleveIndex)

	defaults.PoliciesPath = policies.Directory()
	defaultPolicies, err := defaults.Policies(true)
	suite.Require().NoError(err)

	suite.defaultPolicies = make(map[string]*v1.Policy, len(defaultPolicies))
	for _, p := range defaultPolicies {
		suite.defaultPolicies[p.GetName()] = p
	}
}

func (suite *DefaultPoliciesTestSuite) TearDownTest() {
	suite.bleveIndex.Close()
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
		Name: &v1.ImageName{Sha: uuid.NewV4().String(), FullName: "ASFASF"},
		Scan: &v1.ImageScan{
			Components: components,
		},
	}
}

func imageWithLayers(layers []*v1.ImageLayer) *v1.Image {
	return &v1.Image{
		Name: &v1.ImageName{Sha: uuid.NewV4().String()},
		Metadata: &v1.ImageMetadata{
			Layers: layers,
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

func (suite *DefaultPoliciesTestSuite) TestDefaultPolicies() {
	fixtureDep := fixtures.GetDeployment()
	suite.mustIndexDepAndImages(fixtureDep)

	nginx110 := &v1.Image{
		Name: &v1.ImageName{
			Sha:      "SHANGINX110",
			Registry: "docker.io",
			Remote:   "library/nginx",
			Tag:      "1.10",
		},
	}
	nginx110dep := &v1.Deployment{
		Id: "nginx110",
		Containers: []*v1.Container{
			{Image: nginx110},
		},
	}
	suite.mustIndexDepAndImages(nginx110dep)

	oldScannedTime := time.Now().Add(-31 * 24 * time.Hour)
	oldScannedImage := &v1.Image{
		Name: &v1.ImageName{
			Sha: "SHAOLDSCANNED",
		},
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
		Name:     &v1.ImageName{Sha: "SHA:OLDCREATEDIMAGE"},
		Metadata: &v1.ImageMetadata{Created: protoconv.ConvertTimeToTimestamp(oldImageCreationTime)},
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

	testCases := []struct {
		policyName         string
		expectedViolations map[string][]*v1.Alert_Violation
	}{
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
				nginx110dep.GetId(): {
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
			// This policy is special-cased in the test handling code.
			policyName: "Images with no scans",
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
	}

	for _, c := range testCases {
		p := suite.MustGetPolicy(c.policyName)
		suite.T().Run(c.policyName, func(t *testing.T) {
			m, err := ForPolicy(p, mappings.OptionsMap)
			require.NoError(t, err)
			matches, err := m.Match(suite.deploymentIndexer)
			require.NoError(t, err)

			// This particular policy matches almost all our test deployments, so treat it specially.
			if c.policyName == "Images with no scans" {
				assert.True(t, len(matches) > 12, "Expected at least 12 matches but found %+v", matches)
				_, fixtureDepMatched := matches[fixtureDep.GetId()]
				assert.False(t, fixtureDepMatched, "Fixture dep has scans and shouldn't match this policy")
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
		})
	}

}

func (suite *DefaultPoliciesTestSuite) TestRuntimePolicyFieldsCompile() {
	for _, p := range suite.defaultPolicies {
		if p.LifecycleStage == v1.LifecycleStage_RUN_TIME && p.GetFields().GetProcessPolicy() != nil {
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
