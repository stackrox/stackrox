package index

import (
	"fmt"
	"sort"
	"testing"

	"github.com/blevesearch/bleve"
	"github.com/stackrox/rox/central/globalindex"
	imageIndex "github.com/stackrox/rox/central/image/index"
	processIndicatorIndex "github.com/stackrox/rox/central/processindicator/index"
	secretIndex "github.com/stackrox/rox/central/secret/index"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/suite"
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
	suite.bleveIndex, err = globalindex.MemOnlyIndex()
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
	img22 := &storage.Image{Id: "SHA22", Name: &storage.ImageName{Tag: "2.2"}}
	img221 := &storage.Image{Id: "SHA221", Name: &storage.ImageName{Tag: "2.2.1"}}

	deployment22 := &storage.Deployment{
		Id: "22",
		Containers: []*storage.Container{
			{Image: img22, Volumes: []*storage.Volume{{Name: "volume22a"}, {Name: "volume22b"}, {Name: "nomatch"}}},
		},
	}
	deployment221 := &storage.Deployment{
		Id: "221",
		Containers: []*storage.Container{
			{Image: img221, Volumes: []*storage.Volume{{Name: "volume221a"}}, Resources: &storage.Resources{CpuCoresRequest: 0.1}},
			{Resources: &storage.Resources{CpuCoresRequest: 0.75}},
		},
	}
	depWithBoth22And221 := &storage.Deployment{
		Id:         "Dep22And221",
		Containers: []*storage.Container{{Image: img22}, {Image: img221}},
	}

	suite.NoError(suite.indexer.AddDeployments([]*storage.Deployment{deployment22, deployment221, depWithBoth22And221}))
	suite.NoError(suite.imageIndexer.AddImages([]*storage.Image{img22, img221}))

	cases := []struct {
		q                    *v1.Query
		expectedIdsToMatches map[string]map[string][]string
	}{
		{
			q: search.NewQueryBuilder().AddStringsHighlighted(search.ImageTag, "r/2.2.*").ProtoQuery(),
			expectedIdsToMatches: map[string]map[string][]string{
				deployment22.GetId(): {
					"image.name.tag": {img22.GetName().GetTag()},
				},
				deployment221.GetId(): {
					"image.name.tag": {img221.GetName().GetTag()},
				},
				depWithBoth22And221.GetId(): {
					"image.name.tag": {img22.GetName().GetTag(), img221.GetName().GetTag()},
				},
			},
		},
		{
			q: search.NewQueryBuilder().AddStringsHighlighted(search.ImageTag, "r/2.2.*").AddStrings(search.DeploymentID, "22").ProtoQuery(),
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
		results, err := suite.indexer.Search(c.q)
		suite.Require().NoError(err)
		suite.Len(results, len(c.expectedIdsToMatches), "Results: %+v expected matches: %+v", results, c.expectedIdsToMatches)

		for _, r := range results {
			expectedMatches, ok := c.expectedIdsToMatches[r.ID]
			suite.Require().True(ok, "Results: %+v, expected matches: %+v", results, c.expectedIdsToMatches)
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
	for _, container := range deployment.GetContainers() {
		if container.GetImage() != nil {
			suite.NoError(suite.imageIndexer.AddImage(container.GetImage()))
		}
	}

	containerPort22Dep := &storage.Deployment{
		Id: "CONTAINERPORT22DEP",
		Ports: []*storage.PortConfig{
			{Protocol: "tcp", ContainerPort: 22},
			{Protocol: "udp", ContainerPort: 4125},
		},
	}
	suite.NoError(suite.indexer.AddDeployment(containerPort22Dep))

	img110 := &storage.Image{Id: "SHA110", Name: &storage.ImageName{Tag: "1.10"}}
	imgNginx := &storage.Image{Id: "SHANGINX", Name: &storage.ImageName{Remote: "nginx"}}
	notNginx110Dep := &storage.Deployment{
		Id:         "NOTNGINX110ID",
		Name:       "NOT110",
		Containers: []*storage.Container{{Image: img110}, {Image: imgNginx}},
	}

	suite.NoError(suite.indexer.AddDeployment(notNginx110Dep))
	suite.NoError(suite.imageIndexer.AddImage(img110))
	suite.NoError(suite.imageIndexer.AddImage(imgNginx))

	imgNginx110 := &storage.Image{Id: "SHANGINX110", Name: &storage.ImageName{Tag: "1.10", Remote: "nginx"}}
	nginx110Dep := &storage.Deployment{
		Id:         "NGINX110ID",
		Name:       "YES110",
		Containers: []*storage.Container{{Image: imgNginx110}},
	}
	suite.NoError(suite.indexer.AddDeployment(nginx110Dep))
	suite.NoError(suite.imageIndexer.AddImage(imgNginx110))

	badEmailDep := &storage.Deployment{
		Id:     "BADEMAILID",
		Labels: map[string]string{"email": "INVALIDEMAIL"},
	}
	suite.NoError(suite.indexer.AddDeployment(badEmailDep))

	secretIndexer := secretIndex.New(suite.bleveIndex)
	suite.NoError(secretIndexer.UpsertSecret(&storage.Secret{
		Id: "ABC",
	}))

	processIndexer := processIndicatorIndex.New(suite.bleveIndex)
	suite.NoError(processIndexer.AddProcessIndicator(fixtures.GetProcessIndicator()))

	cases := []struct {
		fieldValues           map[search.FieldLabel]string
		docIDS                []string
		linkedFields          []search.FieldLabel
		linkedFieldValues     []string
		highlightLinkedFields bool
		highlightedFields     []search.FieldLabel
		expectedIDs           []string
		expectedMatches       map[string][]string
	}{
		{
			docIDS:      []string{deployment.GetId(), badEmailDep.GetId()},
			expectedIDs: []string{deployment.GetId(), badEmailDep.GetId()},
		},
		{
			docIDS:      []string{nginx110Dep.GetId()},
			expectedIDs: []string{nginx110Dep.GetId()},
		},
		{
			fieldValues: map[search.FieldLabel]string{search.DeploymentName: "nginx"},
			expectedIDs: []string{deployment.GetId()},
		},
		{
			fieldValues: map[search.FieldLabel]string{search.DeploymentName: "r/ngi.*"},
			expectedIDs: []string{deployment.GetId()},
		},
		{
			fieldValues: map[search.FieldLabel]string{search.DeploymentName: "!nginx"},
			expectedIDs: []string{notNginx110Dep.GetId(), nginx110Dep.GetId(), containerPort22Dep.GetId(), badEmailDep.GetId()},
		},
		{
			fieldValues: map[search.FieldLabel]string{search.DeploymentName: "!nginx"},
			docIDS:      []string{containerPort22Dep.GetId()},
			expectedIDs: []string{containerPort22Dep.GetId()},
		},
		{
			fieldValues: map[search.FieldLabel]string{search.DeploymentName: "!nginx"},
			docIDS:      []string{deployment.GetId()},
		},
		{
			fieldValues: map[search.FieldLabel]string{search.DeploymentName: "!r/ngi.*"},
			expectedIDs: []string{notNginx110Dep.GetId(), nginx110Dep.GetId(), containerPort22Dep.GetId(), badEmailDep.GetId()},
		},
		{
			fieldValues: map[search.FieldLabel]string{search.DeploymentName: "!!nginx"},
			expectedIDs: []string{deployment.GetId()},
		},
		{
			fieldValues: map[search.FieldLabel]string{search.Label: "com.docker.stack.namespace=prevent"},
			expectedIDs: []string{deployment.GetId()},
		},
		{
			fieldValues: map[search.FieldLabel]string{search.Label: "email=r/^[a-zA-Z0-9_.+-]+@[a-zA-Z0-9-]+\\.[a-zA-Z0-9-.]+$"},
			expectedIDs: []string{deployment.GetId()},
		},
		{
			fieldValues: map[search.FieldLabel]string{search.Label: "email=!r/^[a-zA-Z0-9_.+-]+@[a-zA-Z0-9-]+\\.[a-zA-Z0-9-.]+$"},
			expectedIDs: []string{notNginx110Dep.GetId(), nginx110Dep.GetId(), containerPort22Dep.GetId(), badEmailDep.GetId()},
		},
		{
			fieldValues: map[search.FieldLabel]string{search.Label: "app=nginx"},
			expectedIDs: []string{},
		},
		{
			fieldValues: map[search.FieldLabel]string{search.PodLabel: "app=nginx"},
			expectedIDs: []string{deployment.GetId()},
		},
		{
			fieldValues: map[search.FieldLabel]string{search.PodLabel: "com.docker.stack.namespace=prevent"},
			expectedIDs: []string{},
		},
		{
			fieldValues: map[search.FieldLabel]string{search.DeploymentName: "!nginx", search.Label: "com.docker.stack.namespace=prevent"},
			expectedIDs: []string{},
		},
		{
			fieldValues: map[search.FieldLabel]string{search.DeploymentName: "!nomatch", search.Label: "com.docker.stack.namespace=r/.*"},
			expectedIDs: []string{deployment.GetId()},
		},
		{
			fieldValues: map[search.FieldLabel]string{search.DeploymentName: "!nomatch", search.Label: "com.docker.stack.namespace=*"},
			expectedIDs: []string{deployment.GetId()},
		},
		{
			fieldValues: map[search.FieldLabel]string{search.DeploymentName: "!nomatch"},
			expectedIDs: []string{deployment.GetId(), notNginx110Dep.GetId(), nginx110Dep.GetId(), containerPort22Dep.GetId(), badEmailDep.GetId()},
		},
		{
			fieldValues: map[search.FieldLabel]string{search.DeploymentName: "!nomatch", search.ImageRegistry: "stackrox"},
			expectedIDs: []string{deployment.GetId()},
		},
		{
			fieldValues: map[search.FieldLabel]string{search.DeploymentName: "!nomatch", search.ImageRegistry: "nonexistent"},
			expectedIDs: []string{},
		},
		{
			fieldValues: map[search.FieldLabel]string{search.ProcessName: fixtures.GetProcessIndicator().GetSignal().GetName()},
			expectedIDs: []string{deployment.GetId()},
		},
		{
			fieldValues: map[search.FieldLabel]string{search.Port: "22"},
			expectedIDs: []string{containerPort22Dep.GetId()},
		},
		{
			fieldValues: map[search.FieldLabel]string{search.Port: "22", search.PortProtocol: "tcp"},
			expectedIDs: []string{containerPort22Dep.GetId()},
		},
		{
			fieldValues: map[search.FieldLabel]string{search.DeploymentID: deployment.GetId()},
			expectedIDs: []string{deployment.GetId()},
		},
		{

			linkedFields:      []search.FieldLabel{search.ImageTag, search.ImageRemote},
			linkedFieldValues: []string{"1.10", "nginx"},
			expectedIDs:       []string{nginx110Dep.GetId()},
		},
		{
			fieldValues: map[search.FieldLabel]string{search.ImageTag: "latest"},
			expectedIDs: []string{deployment.GetId()},
		},
		{
			fieldValues:       map[search.FieldLabel]string{search.ImageTag: "latest"},
			highlightedFields: []search.FieldLabel{search.ImageTag},
			docIDS:            []string{nginx110Dep.GetId()},
		},
		{
			fieldValues:       map[search.FieldLabel]string{search.ImageTag: "latest"},
			highlightedFields: []search.FieldLabel{search.ImageTag},
			docIDS:            []string{deployment.GetId()},
			expectedIDs:       []string{deployment.GetId()},
			expectedMatches:   map[string][]string{"image.name.tag": {"latest"}},
		},
		{
			fieldValues:       map[search.FieldLabel]string{search.ImageTag: "latest"},
			highlightedFields: []search.FieldLabel{search.ImageTag},
			expectedIDs:       []string{deployment.GetId()},
			expectedMatches:   map[string][]string{"image.name.tag": {"latest"}},
		},
		{
			fieldValues:       map[search.FieldLabel]string{search.ImageTag: "lat"},
			highlightedFields: []search.FieldLabel{search.ImageTag},
			expectedIDs:       []string{deployment.GetId()},
			expectedMatches:   map[string][]string{"image.name.tag": {"latest"}},
		},
		{
			fieldValues:       map[search.FieldLabel]string{search.ImageTag: "=lat"},
			highlightedFields: []search.FieldLabel{search.ImageTag},
		},
		{
			fieldValues:       map[search.FieldLabel]string{search.ImageTag: "=latest"},
			highlightedFields: []search.FieldLabel{search.ImageTag},
			expectedIDs:       []string{deployment.GetId()},
			expectedMatches:   map[string][]string{"image.name.tag": {"latest"}},
		},
		{
			fieldValues:       map[search.FieldLabel]string{search.ImageTag: "lata"},
			highlightedFields: []search.FieldLabel{search.ImageTag},
		},
		{
			fieldValues:       map[search.FieldLabel]string{search.ImageTag: "r/latest"},
			highlightedFields: []search.FieldLabel{search.ImageTag},
			expectedIDs:       []string{deployment.GetId()},
			expectedMatches:   map[string][]string{"image.name.tag": {"latest"}},
		},
		{
			fieldValues:       map[search.FieldLabel]string{search.ImageTag: "r/lat.*"},
			highlightedFields: []search.FieldLabel{search.ImageTag},
			expectedIDs:       []string{deployment.GetId()},
			expectedMatches:   map[string][]string{"image.name.tag": {"latest"}},
		},
		{
			fieldValues:       map[search.FieldLabel]string{search.ImageTag: "r/lat"},
			highlightedFields: []search.FieldLabel{search.ImageTag},
		},
		{
			fieldValues:       map[search.FieldLabel]string{search.ImageTag: "r/lata.*"},
			highlightedFields: []search.FieldLabel{search.ImageTag},
		},
		{
			fieldValues: map[search.FieldLabel]string{search.CPUCoresRequest: ">0.5"},
			expectedIDs: []string{deployment.GetId()},
		},
		{
			fieldValues: map[search.FieldLabel]string{search.DockerfileInstructionKeyword: "r/.*"},
			expectedIDs: []string{deployment.GetId()},
		},
		{
			fieldValues: map[search.FieldLabel]string{search.DockerfileInstructionKeyword: search.RegexQueryString("CMD")},
			expectedIDs: []string{deployment.GetId()},
		},
		{
			fieldValues: map[search.FieldLabel]string{search.DockerfileInstructionKeyword: "r/cmd"},
			expectedIDs: []string{deployment.GetId()},
		},
		{
			fieldValues: map[search.FieldLabel]string{search.DockerfileInstructionValue: "r/.*"},
			expectedIDs: []string{deployment.GetId()},
		},
		{
			fieldValues: map[search.FieldLabel]string{search.ImageTag: search.WildcardString},
			expectedIDs: []string{nginx110Dep.GetId(), deployment.GetId(), notNginx110Dep.GetId()},
		},
		{
			linkedFields:      []search.FieldLabel{search.Port, search.PortProtocol},
			linkedFieldValues: []string{"22", "udp"},
			expectedIDs:       []string{},
		},
		{
			linkedFields:      []search.FieldLabel{search.Port, search.PortProtocol},
			linkedFieldValues: []string{"22", "tcp"},
			expectedIDs:       []string{containerPort22Dep.GetId()},
		},
		{
			linkedFields:          []search.FieldLabel{search.Port, search.PortProtocol},
			linkedFieldValues:     []string{"22", "tcp"},
			highlightLinkedFields: true,
			expectedIDs:           []string{containerPort22Dep.GetId()},
			expectedMatches: map[string][]string{
				"deployment.ports.container_port": {"22"},
				"deployment.ports.protocol":       {"tcp"},
			},
		},
		{
			linkedFields:      []search.FieldLabel{search.DockerfileInstructionKeyword, search.DockerfileInstructionValue},
			linkedFieldValues: []string{"ADD", "443/tcp"},
			expectedIDs:       []string{},
		},
		{
			linkedFields:      []search.FieldLabel{search.DockerfileInstructionKeyword, search.DockerfileInstructionValue},
			linkedFieldValues: []string{"ADD", "file:4ee"},
			expectedIDs:       []string{deployment.GetId()},
		},
		{
			linkedFields:          []search.FieldLabel{search.DockerfileInstructionKeyword, search.DockerfileInstructionValue},
			linkedFieldValues:     []string{"ADD", "file:4ee"},
			highlightLinkedFields: true,
			expectedIDs:           []string{deployment.GetId()},
			expectedMatches: map[string][]string{
				"image.metadata.v1.layers.instruction": {"ADD"},
				"image.metadata.v1.layers.value":       {"file:4eedf861fb567fffb2694b65ebdd58d5e371a2c28c3863f363f333cb34e5eb7b in /"},
			},
		},
		{
			linkedFields:      []search.FieldLabel{search.DockerfileInstructionKeyword, search.DockerfileInstructionValue},
			linkedFieldValues: []string{"CMD", "["},
			expectedIDs:       []string{deployment.GetId()},
		},
		{
			linkedFields:      []search.FieldLabel{search.DockerfileInstructionKeyword, search.DockerfileInstructionValue},
			linkedFieldValues: []string{"cmd", "["},
			expectedIDs:       []string{deployment.GetId()},
		},
		{
			linkedFields:      []search.FieldLabel{search.DockerfileInstructionKeyword, search.DockerfileInstructionValue},
			linkedFieldValues: []string{"r/cmd", "["},
			expectedIDs:       []string{deployment.GetId()},
		},
		{
			linkedFields:      []search.FieldLabel{search.DockerfileInstructionKeyword, search.DockerfileInstructionValue},
			linkedFieldValues: []string{"r/.*", "r/.*"},
			expectedIDs:       []string{deployment.GetId()},
		},
		{
			linkedFields:          []search.FieldLabel{search.DockerfileInstructionKeyword, search.DockerfileInstructionValue},
			linkedFieldValues:     []string{"r/.*", "r/.*"},
			highlightLinkedFields: true,
			expectedIDs:           []string{deployment.GetId()},
			expectedMatches: func() map[string][]string {
				m := make(map[string][]string)
				for _, container := range deployment.GetContainers() {
					for _, layer := range container.GetImage().GetMetadata().GetV1().GetLayers() {
						m["image.metadata.v1.layers.instruction"] = append(m["image.metadata.v1.layers.instruction"], layer.GetInstruction())
						m["image.metadata.v1.layers.value"] = append(m["image.metadata.v1.layers.value"], layer.GetValue())
					}
				}
				return m
			}(),
		},
		{
			linkedFields:          []search.FieldLabel{search.DockerfileInstructionKeyword, search.DockerfileInstructionValue},
			linkedFieldValues:     []string{"CMD", "["},
			highlightLinkedFields: true,
			expectedIDs:           []string{deployment.GetId()},
			expectedMatches: map[string][]string{
				"image.metadata.v1.layers.instruction": {"CMD", "CMD"},
				"image.metadata.v1.layers.value":       {`["nginx" "-g" "daemon off;"]`, `["/bin/bash"]`},
			},
		},
		{
			fieldValues:       map[search.FieldLabel]string{search.CPUCoresRequest: ">0.5"},
			expectedIDs:       []string{deployment.GetId()},
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
		if len(c.linkedFields) > 0 {
			suite.Require().Len(c.linkedFieldValues, len(c.linkedFields))
			if c.highlightLinkedFields {
				qb.AddLinkedFieldsHighlighted(c.linkedFields, c.linkedFieldValues)
			} else {
				qb.AddLinkedFields(c.linkedFields, c.linkedFieldValues)
			}
		}
		qb.AddDocIDs(c.docIDS...)
		results, err := suite.indexer.Search(qb.ProtoQuery())
		suite.NoError(err)

		resultIDs := make([]string, 0, len(results))
		for _, r := range results {
			resultIDs = append(resultIDs, r.ID)
		}
		suite.ElementsMatch(resultIDs, c.expectedIDs, "Failed test case %+v; got results %+v", c, results)

		if c.expectedMatches == nil {
			for _, r := range results {
				suite.Empty(r.Matches)
			}
		} else {
			suite.Require().Len(results, 1, "The expected matches option currently only works if you have 1 "+
				"result, please update the test if you want it to be more general.")
			suite.Equal(c.expectedMatches, results[0].Matches)
		}
	}
}

func (suite *DeploymentIndexTestSuite) TestBatches() {
	deployments := []*storage.Deployment{
		fixtures.GetDeployment(),
		fixtures.GetDeployment(),
		fixtures.GetDeployment(),
		fixtures.GetDeployment(),
	}
	for _, d := range deployments {
		d.Id = uuid.NewV4().String()
	}
	err := suite.indexer.AddDeployments(deployments)
	suite.NoError(err)
	for _, d := range deployments {
		results, err := suite.indexer.Search(search.NewQueryBuilder().AddExactMatches(search.DeploymentID, d.GetId()).ProtoQuery())
		suite.NoError(err)
		suite.Len(results, 1)
	}
}

func (suite *DeploymentIndexTestSuite) TestCaseInsensitivityOfFieldNames() {
	dep := fixtures.GetDeployment()
	suite.NoError(suite.indexer.AddDeployment(dep))
	ns := dep.GetNamespace()

	upperCaseQ, err := search.ParseRawQuery(fmt.Sprintf("Namespace:%s", ns))
	suite.NoError(err)
	lowerCaseQ, err := search.ParseRawQuery(fmt.Sprintf("namespace:%s", ns))
	suite.NoError(err)
	for _, q := range []*v1.Query{upperCaseQ, lowerCaseQ} {
		results, err := suite.indexer.Search(q)
		suite.NoError(err)
		suite.Len(results, 1)
	}
}

func (suite *DeploymentIndexTestSuite) TestDeploymentDelete() {
	dep := fixtures.GetDeployment()
	suite.NoError(suite.indexer.AddDeployment(dep))

	ns := dep.GetNamespace()
	upperCaseQ, err := search.ParseRawQuery(fmt.Sprintf("Namespace:%s", ns))
	suite.NoError(err)
	results, err := suite.indexer.Search(upperCaseQ)
	suite.NoError(err)
	suite.Len(results, 1)

	suite.NoError(suite.indexer.DeleteDeployment(dep.GetId()))
	results, err = suite.indexer.Search(upperCaseQ)
	suite.NoError(err)
	suite.Len(results, 0)
}
