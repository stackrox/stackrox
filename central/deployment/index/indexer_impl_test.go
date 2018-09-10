package index

import (
	"sort"
	"testing"

	"github.com/blevesearch/bleve"
	"github.com/stackrox/rox/central/globalindex"
	imageIndex "github.com/stackrox/rox/central/image/index"
	processIndicatorIndex "github.com/stackrox/rox/central/processindicator/index"
	secretIndex "github.com/stackrox/rox/central/secret/index"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stretchr/testify/suite"
)

const (
	fakeID   = "FAKEID"
	fakeName = "FAKENAME"
)

func TestDeploymentIndex(t *testing.T) {
	suite.Run(t, new(DeploymentIndexTestSuite))
}

type DeploymentIndexTestSuite struct {
	suite.Suite

	bleveIndex   bleve.Index
	indexer      Indexer
	imageIndexer imageIndex.Indexer
}

func (suite *DeploymentIndexTestSuite) SetupTest() {
	var err error
	suite.bleveIndex, err = globalindex.TempInitializeIndices("")
	suite.Require().NoError(err)

	suite.indexer = New(suite.bleveIndex)
	suite.imageIndexer = imageIndex.New(suite.bleveIndex)
}

func (suite *DeploymentIndexTestSuite) TearDownTest() {
	suite.bleveIndex.Close()
}

// This test makes sure that, when we search deployments by images,
// and request highlights from the search, the highlights we get
// actually match the value in the deployments.
func (suite *DeploymentIndexTestSuite) TestHighlighting() {
	img22 := &v1.Image{Name: &v1.ImageName{Sha: "SHA22", Tag: "2.2"}}
	img221 := &v1.Image{Name: &v1.ImageName{Sha: "SHA221", Tag: "2.2.1"}}

	deployment22 := &v1.Deployment{
		Id: "22",
		Containers: []*v1.Container{
			{Image: img22, Volumes: []*v1.Volume{{Name: "volume22a"}, {Name: "volume22b"}, {Name: "nomatch"}}},
		},
	}
	deployment221 := &v1.Deployment{
		Id: "221",
		Containers: []*v1.Container{
			{Image: img221, Volumes: []*v1.Volume{{Name: "volume221a"}}, Resources: &v1.Resources{CpuCoresRequest: 0.1}},
			{Resources: &v1.Resources{CpuCoresRequest: 0.75}},
		},
	}

	suite.NoError(suite.indexer.AddDeployments([]*v1.Deployment{deployment22, deployment221}))
	suite.NoError(suite.imageIndexer.AddImages([]*v1.Image{img22, img221}))

	cases := []struct {
		q                    *v1.Query
		expectedIdsToMatches map[string]map[string][]string
	}{
		{
			q: search.NewQueryBuilder().AddStringsHighlighted(search.ImageTag, "/2.2.*").ProtoQuery(),
			expectedIdsToMatches: map[string]map[string][]string{
				deployment22.GetId(): {
					"image.name.tag": {img22.GetName().GetTag()},
				},
				deployment221.GetId(): {
					"image.name.tag": {img221.GetName().GetTag()},
				},
			},
		},
		{
			q: search.NewQueryBuilder().AddStringsHighlighted(search.ImageTag, "/2.2.*").AddStrings(search.DeploymentID, "22").ProtoQuery(),
			expectedIdsToMatches: map[string]map[string][]string{
				deployment22.GetId(): {
					"image.name.tag": {img22.GetName().GetTag()},
				},
				deployment221.GetId(): {
					"image.name.tag": {img221.GetName().GetTag()},
				},
			},
		},
		{
			q: search.NewQueryBuilder().AddStringsHighlighted(search.DeploymentID, "22").ProtoQuery(),
			expectedIdsToMatches: map[string]map[string][]string{
				deployment22.GetId(): {
					"deployment.id": {deployment22.GetId()},
				},
				deployment221.GetId(): {
					"deployment.id": {deployment221.GetId()},
				},
			},
		},
		{
			q: search.NewQueryBuilder().
				AddStringsHighlighted(search.DeploymentID, "22").
				AddStringsHighlighted(search.ImageTag, "2.2").
				ProtoQuery(),

			expectedIdsToMatches: map[string]map[string][]string{
				deployment22.GetId(): {
					"image.name.tag": {img22.GetName().GetTag()},
					"deployment.id":  {deployment22.GetId()},
				},
				deployment221.GetId(): {
					"deployment.id":  {deployment221.GetId()},
					"image.name.tag": {img221.GetName().GetTag()},
				},
			},
		},
		{
			q: search.NewQueryBuilder().
				AddStringsHighlighted(search.DeploymentID, "22").
				AddStringsHighlighted(search.ImageTag, "2.2").
				ProtoQuery(),

			expectedIdsToMatches: map[string]map[string][]string{
				deployment22.GetId(): {
					"image.name.tag": {img22.GetName().GetTag()},
					"deployment.id":  {deployment22.GetId()},
				},
				deployment221.GetId(): {
					"deployment.id":  {deployment221.GetId()},
					"image.name.tag": {img221.GetName().GetTag()},
				},
			},
		},
		{
			q: search.NewQueryBuilder().AddStringsHighlighted(search.VolumeName, "volume22").ProtoQuery(),
			expectedIdsToMatches: map[string]map[string][]string{
				deployment22.GetId(): {
					"deployment.containers.volumes.name": {"volume22a", "volume22b"},
				},
				deployment221.GetId(): {
					"deployment.containers.volumes.name": {"volume221a"},
				},
			},
		},
		{
			q: search.NewQueryBuilder().AddStringsHighlighted(search.CPUCoresRequest, ">0.05").ProtoQuery(),
			expectedIdsToMatches: map[string]map[string][]string{
				deployment221.GetId(): {
					"deployment.containers.resources.cpu_cores_request": {"0.10", "0.75"},
				},
			},
		},
		{
			q: search.NewQueryBuilder().AddStringsHighlighted(search.CPUCoresRequest, ">0.5").ProtoQuery(),
			expectedIdsToMatches: map[string]map[string][]string{
				deployment221.GetId(): {
					"deployment.containers.resources.cpu_cores_request": {"0.75"},
				},
			},
		},
	}

	for _, c := range cases {
		results, err := suite.indexer.SearchDeployments(c.q)
		suite.Require().NoError(err)
		suite.Len(results, len(c.expectedIdsToMatches), "Results: %#v expected matches: %#v", results, c.expectedIdsToMatches)

		for _, r := range results {
			expectedMatches, ok := c.expectedIdsToMatches[r.ID]
			suite.Require().True(ok, "Results: %#v, expected matches: %#v", results, c.expectedIdsToMatches)
			// Sort for consistent test results.
			for _, m := range r.Matches {
				sort.Strings(m)
			}
			suite.Equal(expectedMatches, r.Matches)
		}
	}
}

func (suite *DeploymentIndexTestSuite) TestDeploymentsQuery() {
	deployment := fixtures.GetDeployment()
	suite.NoError(suite.indexer.AddDeployment(deployment))
	suite.NoError(suite.indexer.AddDeployment(&v1.Deployment{Id: fakeID, Name: fakeName}))

	suite.imageIndexer.AddImage(fixtures.GetImage())

	secretIndexer := secretIndex.New(suite.bleveIndex)
	secretIndexer.UpsertSecret(&v1.Secret{
		Id: "ABC",
	})

	processIndexer := processIndicatorIndex.New(suite.bleveIndex)
	processIndexer.AddProcessIndicator(fixtures.GetProcessIndicator())
	cases := []struct {
		fieldValues       map[search.FieldLabel]string
		highlightedFields []search.FieldLabel
		expectedIDs       []string
		expectedMatches   map[string][]string
	}{
		{
			fieldValues: map[search.FieldLabel]string{search.DeploymentName: "nginx"},
			expectedIDs: []string{fixtures.GetDeployment().GetId()},
		},
		{
			fieldValues: map[search.FieldLabel]string{search.DeploymentName: "!nginx"},
			expectedIDs: []string{fakeID},
		},
		{
			fieldValues: map[search.FieldLabel]string{search.Label: "com.docker.stack.namespace=prevent"},
			expectedIDs: []string{fixtures.GetDeployment().GetId()},
		},
		{
			fieldValues: map[search.FieldLabel]string{search.DeploymentName: "!nginx", search.Label: "com.docker.stack.namespace=prevent"},
			expectedIDs: []string{},
		},
		{
			fieldValues: map[search.FieldLabel]string{search.DeploymentName: "!nomatch", search.Label: "com.docker.stack.namespace=/.*"},
			expectedIDs: []string{fixtures.GetDeployment().GetId()},
		},
		{
			fieldValues: map[search.FieldLabel]string{search.DeploymentName: "!nomatch"},
			expectedIDs: []string{fixtures.GetDeployment().GetId(), fakeID},
		},
		{
			fieldValues: map[search.FieldLabel]string{search.DeploymentName: "!nomatch", search.ImageRegistry: "stackrox"},
			expectedIDs: []string{fixtures.GetDeployment().GetId()},
		},
		{
			fieldValues: map[search.FieldLabel]string{search.DeploymentName: "!nomatch", search.ImageRegistry: "nonexistent"},
			expectedIDs: []string{},
		},
		{
			fieldValues: map[search.FieldLabel]string{search.ProcessName: fixtures.GetProcessIndicator().GetSignal().GetProcessSignal().GetName()},
			expectedIDs: []string{fixtures.GetDeployment().GetId()},
		},
		{
			fieldValues: map[search.FieldLabel]string{search.DeploymentID: fixtures.GetDeployment().GetId()},
			expectedIDs: []string{fixtures.GetDeployment().GetId()},
		},
		{
			fieldValues: map[search.FieldLabel]string{search.ImageTag: "latest"},
			expectedIDs: []string{fixtures.GetDeployment().GetId()},
		},
		{
			fieldValues:       map[search.FieldLabel]string{search.ImageTag: "latest"},
			highlightedFields: []search.FieldLabel{search.ImageTag},
			expectedIDs:       []string{fixtures.GetDeployment().GetId()},
			expectedMatches:   map[string][]string{"image.name.tag": {"latest"}},
		},
		{
			fieldValues:       map[search.FieldLabel]string{search.ImageTag: "lat"},
			highlightedFields: []search.FieldLabel{search.ImageTag},
			expectedIDs:       []string{fixtures.GetDeployment().GetId()},
			expectedMatches:   map[string][]string{"image.name.tag": {"latest"}},
		},
		{
			fieldValues:       map[search.FieldLabel]string{search.ImageTag: "lata"},
			highlightedFields: []search.FieldLabel{search.ImageTag},
		},
		{
			fieldValues:       map[search.FieldLabel]string{search.ImageTag: "/latest"},
			highlightedFields: []search.FieldLabel{search.ImageTag},
			expectedIDs:       []string{fixtures.GetDeployment().GetId()},
			expectedMatches:   map[string][]string{"image.name.tag": {"latest"}},
		},
		{
			fieldValues:       map[search.FieldLabel]string{search.ImageTag: "/lat.*"},
			highlightedFields: []search.FieldLabel{search.ImageTag},
			expectedIDs:       []string{fixtures.GetDeployment().GetId()},
			expectedMatches:   map[string][]string{"image.name.tag": {"latest"}},
		},
		{
			fieldValues:       map[search.FieldLabel]string{search.ImageTag: "/lat"},
			highlightedFields: []search.FieldLabel{search.ImageTag},
		},
		{
			fieldValues:       map[search.FieldLabel]string{search.ImageTag: "/lata.*"},
			highlightedFields: []search.FieldLabel{search.ImageTag},
		},
		{
			fieldValues: map[search.FieldLabel]string{search.CPUCoresRequest: ">0.5"},
			expectedIDs: []string{fixtures.GetDeployment().GetId()},
		},
		{
			fieldValues:       map[search.FieldLabel]string{search.CPUCoresRequest: ">0.5"},
			expectedIDs:       []string{fixtures.GetDeployment().GetId()},
			highlightedFields: []search.FieldLabel{search.CPUCoresRequest},
			expectedMatches:   map[string][]string{"deployment.containers.resources.cpu_cores_request": {"0.90"}},
		},
	}

	for _, c := range cases {
		qb := search.NewQueryBuilder()
		for field, value := range c.fieldValues {
			qb.AddStrings(field, value)
		}
		for _, field := range c.highlightedFields {
			qb.MarkHighlighted(field)
		}
		results, err := suite.indexer.SearchDeployments(qb.ProtoQuery())
		suite.NoError(err)

		resultIDs := make([]string, 0, len(results))
		for _, r := range results {
			resultIDs = append(resultIDs, r.ID)
		}
		suite.ElementsMatch(resultIDs, c.expectedIDs, "Failed test case %#v", c)

		if c.expectedMatches == nil {
			for _, r := range results {
				suite.Empty(r.Matches)
			}
		} else {
			suite.Len(results, 1, "The expected matches option currently only works if you have 1 "+
				"result, please update the test if you want it to be more general.")
			suite.Equal(c.expectedMatches, results[0].Matches)
		}
	}
}
