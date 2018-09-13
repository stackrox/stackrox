package matcher

import (
	"fmt"
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
	defaultPolicies, err := defaults.Policies()
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

func (suite *DefaultPoliciesTestSuite) TestDefaultPolicies() {
	fixtureDep := fixtures.GetDeployment()
	suite.deploymentIndexer.AddDeployment(fixtureDep)
	suite.imageIndexer.AddImage(fixtures.GetImage())

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
	suite.deploymentIndexer.AddDeployment(nginx110dep)
	suite.imageIndexer.AddImage(nginx110)

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
	suite.deploymentIndexer.AddDeployment(oldScannedDep)
	suite.imageIndexer.AddImage(oldScannedImage)

	addDockerFileImage := &v1.Image{
		Name: &v1.ImageName{
			Sha: "SHAADDINDOCKERFILE",
		},
		Metadata: &v1.ImageMetadata{
			Layers: []*v1.ImageLayer{
				{
					Instruction: "ADD",
					Value:       "deploy.sh",
				},
				{
					Instruction: "RUN",
					Value:       "deploy.sh",
				},
			},
		},
	}
	addDockerFileDep := &v1.Deployment{
		Id: "adddockerfiledep",
		Containers: []*v1.Container{
			{Image: addDockerFileImage},
		},
	}
	suite.deploymentIndexer.AddDeployment(addDockerFileDep)
	suite.imageIndexer.AddImage(addDockerFileImage)

	// Fake deployment that shouldn't match anything, just to make sure
	// that none of our queries will accidentally match it.
	suite.deploymentIndexer.AddDeployment(&v1.Deployment{Id: "FAKEID", Name: "FAKENAME"})

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
			policyName: "30-Day Scan Age",
			expectedViolations: map[string][]*v1.Alert_Violation{
				oldScannedDep.GetId(): {
					{
						Message: fmt.Sprintf("Time of last scan '%s' was more than 30 days ago", readable.Time(oldScannedTime)),
					},
				},
			},
		},
		/*
			{
				policyName: "ADD Command used instead of COPY",
				expectedViolations: map[string][]*v1.Alert_Violation{
					oldScannedDep.GetId(): {
						{
							Message: fmt.Sprintf("Time of last scan '%s' was more than 30 days ago", readable.Time(oldScannedTime)),
						},
					},
				},
			},
		*/
	}

	for _, c := range testCases {
		p := suite.MustGetPolicy(c.policyName)
		suite.T().Run(c.policyName, func(t *testing.T) {
			m, err := ForPolicy(p, mappings.OptionsMap)
			require.NoError(t, err)
			matches, err := m.Match(suite.deploymentIndexer)
			require.NoError(t, err)
			for id, violations := range c.expectedViolations {
				got, ok := matches[id]
				if !assert.True(t, ok, "Id '%s' didn't match, but should have", id) {
					continue
				}
				assert.ElementsMatch(t, violations, got)
			}
			assert.Len(t, matches, len(c.expectedViolations))
		})
	}

}
