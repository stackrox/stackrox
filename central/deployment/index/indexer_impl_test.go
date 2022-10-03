package index

import (
	"context"
	"fmt"
	"sort"
	"testing"

	"github.com/blevesearch/bleve"
	"github.com/stackrox/rox/central/globalindex"
	processIndicatorIndex "github.com/stackrox/rox/central/processindicator/index"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/images/types"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/blevesearch"
	"github.com/stackrox/rox/pkg/search/paginated"
	"github.com/stackrox/rox/pkg/sliceutils"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

func TestDeploymentIndex(t *testing.T) {
	suite.Run(t, new(DeploymentIndexTestSuite))
}

type DeploymentIndexTestSuite struct {
	suite.Suite

	bleveIndex   bleve.Index
	processIndex bleve.Index
	indexer      Indexer
}

func (suite *DeploymentIndexTestSuite) SetupTest() {
	var err error
	suite.bleveIndex, err = globalindex.MemOnlyIndex()
	suite.Require().NoError(err)

	suite.processIndex, err = globalindex.MemOnlyIndex()
	suite.Require().NoError(err)

	suite.indexer = New(suite.bleveIndex, suite.processIndex)
}

func (suite *DeploymentIndexTestSuite) TearDownTest() {
	suite.NoError(suite.bleveIndex.Close())
}

// TODO(ROX-2986) Re-add unit test once performance hit on negation query is resolved

// This test makes sure that, when we search deployments by images,
// and request highlights from the search, the highlights we get
// actually match the value in the deployments.
func (suite *DeploymentIndexTestSuite) TestHighlighting() {
	deployment22 := &storage.Deployment{
		Id: "22",
		Containers: []*storage.Container{
			{Volumes: []*storage.Volume{{Name: "volume22a"}, {Name: "volume22b"}, {Name: "nomatch"}}},
		},
	}
	deployment221 := &storage.Deployment{
		Id: "221",
		Containers: []*storage.Container{
			{Volumes: []*storage.Volume{{Name: "volume221a"}}, Resources: &storage.Resources{CpuCoresRequest: 0.1}},
			{Resources: &storage.Resources{CpuCoresRequest: 0.75}},
		},
	}
	depWithBoth22And221 := &storage.Deployment{
		Id:         "Dep22And221",
		Containers: []*storage.Container{},
	}

	suite.NoError(suite.indexer.AddDeployments([]*storage.Deployment{deployment22, deployment221, depWithBoth22And221}))

	cases := []struct {
		q                    *v1.Query
		expectedIdsToMatches map[string]map[string][]string
	}{
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
				ProtoQuery(),

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
		results, err := suite.indexer.Search(ctx, c.q)
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

	containerPort22Dep := &storage.Deployment{
		Id:   "CONTAINERPORT22DEP",
		Name: "containerport",
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
		Containers: []*storage.Container{{Image: types.ToContainerImage(img110)}, {Image: types.ToContainerImage(imgNginx)}},
	}

	suite.NoError(suite.indexer.AddDeployment(notNginx110Dep))

	imgNginx110 := &storage.Image{Id: "SHANGINX110", Name: &storage.ImageName{Tag: "1.10", Remote: "nginx"}}
	nginx110Dep := &storage.Deployment{
		Id:         "NGINX110ID",
		Name:       "YES110",
		Containers: []*storage.Container{{Image: types.ToContainerImage(imgNginx110)}},
	}
	suite.NoError(suite.indexer.AddDeployment(nginx110Dep))

	badEmailDep := &storage.Deployment{
		Id:     "BADEMAILID",
		Name:   "bademail",
		Labels: map[string]string{"email": "INVALIDEMAIL"},
	}
	suite.NoError(suite.indexer.AddDeployment(badEmailDep))

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
			fieldValues: map[search.FieldLabel]string{search.DeploymentLabel: "com.docker.stack.namespace=prevent"},
			expectedIDs: []string{deployment.GetId()},
		},
		{
			fieldValues: map[search.FieldLabel]string{search.DeploymentLabel: "email=r/^[a-zA-Z0-9_.+-]+@[a-zA-Z0-9-]+\\.[a-zA-Z0-9-.]+$"},
			expectedIDs: []string{deployment.GetId()},
		},
		{
			fieldValues: map[search.FieldLabel]string{search.DeploymentLabel: "email=!r/^[a-zA-Z0-9_.+-]+@[a-zA-Z0-9-]+\\.[a-zA-Z0-9-.]+$"},
			expectedIDs: []string{badEmailDep.GetId()},
		},
		{
			fieldValues: map[search.FieldLabel]string{search.DeploymentLabel: "!email"},
			expectedIDs: []string{notNginx110Dep.GetId(), nginx110Dep.GetId(), containerPort22Dep.GetId()},
		},
		{
			fieldValues: map[search.FieldLabel]string{search.DeploymentLabel: "app=nginx"},
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
			fieldValues: map[search.FieldLabel]string{search.DeploymentName: "!nginx", search.DeploymentLabel: "com.docker.stack.namespace=prevent"},
			expectedIDs: []string{},
		},
		{
			fieldValues: map[search.FieldLabel]string{search.DeploymentName: "!nomatch", search.DeploymentLabel: "com.docker.stack.namespace=r/.*"},
			expectedIDs: []string{deployment.GetId()},
		},
		{
			fieldValues: map[search.FieldLabel]string{search.DeploymentName: "!nomatch", search.DeploymentLabel: "com.docker.stack.namespace=*"},
			expectedIDs: []string{deployment.GetId()},
		},
		{
			fieldValues: map[search.FieldLabel]string{search.DeploymentName: "!nomatch"},
			expectedIDs: []string{deployment.GetId(), notNginx110Dep.GetId(), nginx110Dep.GetId(), containerPort22Dep.GetId(), badEmailDep.GetId()},
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
			fieldValues: map[search.FieldLabel]string{search.CPUCoresRequest: ">0.5"},
			expectedIDs: []string{deployment.GetId()},
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
		results, err := suite.indexer.Search(ctx, qb.ProtoQuery())
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
		results, err := suite.indexer.Search(ctx, search.NewQueryBuilder().AddExactMatches(search.DeploymentID, d.GetId()).ProtoQuery())
		suite.NoError(err)
		suite.Len(results, 1)
	}
}

func (suite *DeploymentIndexTestSuite) TestCaseInsensitivityOfFieldNames() {
	dep := fixtures.GetDeployment()
	suite.NoError(suite.indexer.AddDeployment(dep))
	ns := dep.GetNamespace()

	upperCaseQ, err := search.ParseQuery(fmt.Sprintf("Namespace:%s", ns))
	suite.NoError(err)
	lowerCaseQ, err := search.ParseQuery(fmt.Sprintf("namespace:%s", ns))
	suite.NoError(err)
	for _, q := range []*v1.Query{upperCaseQ, lowerCaseQ} {
		results, err := suite.indexer.Search(ctx, q)
		suite.NoError(err)
		suite.Len(results, 1)
	}
}

func (suite *DeploymentIndexTestSuite) TestDeploymentDelete() {
	dep := fixtures.GetDeployment()
	suite.NoError(suite.indexer.AddDeployment(dep))

	ns := dep.GetNamespace()
	upperCaseQ, err := search.ParseQuery(fmt.Sprintf("Namespace:%s", ns))
	suite.NoError(err)
	results, err := suite.indexer.Search(ctx, upperCaseQ)
	suite.NoError(err)
	suite.Len(results, 1)

	suite.NoError(suite.indexer.DeleteDeployment(dep.GetId()))
	results, err = suite.indexer.Search(ctx, upperCaseQ)
	suite.NoError(err)
	suite.Len(results, 0)
}

func newPagination(field search.FieldLabel, from, size int32, reversed bool) *v1.QueryPagination {
	return &v1.QueryPagination{
		Limit:  size,
		Offset: from,
		SortOptions: []*v1.QuerySortOption{
			{
				Field:    field.String(),
				Reversed: reversed,
			},
		},
	}
}

func (suite *DeploymentIndexTestSuite) TestSearchSorting() {
	var ids []string
	d := fixtures.GetDeployment()
	for i := 0; i < 10; i++ {
		id := fmt.Sprintf("id%d", i)
		d.Id = id
		ids = append(ids, id)
		d.Containers = []*storage.Container{
			{
				Resources: &storage.Resources{
					MemoryMbLimit: float32(i),
				},
			},
		}
		suite.NoError(suite.indexer.AddDeployment(d))
	}

	reversedIds := sliceutils.StringClone(ids)
	sort.Sort(sort.Reverse(sort.StringSlice(reversedIds)))

	var cases = []struct {
		field      search.FieldLabel
		from, size int
		reversed   bool
	}{
		// String search
		{
			field:    search.DeploymentID,
			from:     0,
			size:     5,
			reversed: false,
		},
		{
			field:    search.DeploymentID,
			from:     2,
			size:     5,
			reversed: false,
		},
		{
			field:    search.DeploymentID,
			from:     5,
			size:     5,
			reversed: false,
		},
		{
			field:    search.DeploymentID,
			from:     0,
			size:     5,
			reversed: true,
		},
		{
			field:    search.DeploymentID,
			from:     2,
			size:     5,
			reversed: true,
		},
		{
			field:    search.DeploymentID,
			from:     5,
			size:     5,
			reversed: true,
		},
		// Numeric Search
		{
			field:    search.MemoryLimit,
			from:     0,
			size:     5,
			reversed: false,
		},
		{
			field:    search.MemoryLimit,
			from:     2,
			size:     5,
			reversed: false,
		},
		{
			field:    search.MemoryLimit,
			from:     5,
			size:     5,
			reversed: false,
		},
		{
			field:    search.MemoryLimit,
			from:     0,
			size:     5,
			reversed: true,
		},
		{
			field:    search.MemoryLimit,
			from:     2,
			size:     5,
			reversed: true,
		},
		{
			field:    search.MemoryLimit,
			from:     5,
			size:     5,
			reversed: true,
		},
	}
	qb := search.NewQueryBuilder().AddStrings(search.DeploymentID, "id").ProtoQuery()
	for _, c := range cases {
		suite.T().Run(fmt.Sprintf("%s-%d-%d-%t", c.field, c.from, c.size, c.reversed), func(t *testing.T) {
			qb.Pagination = newPagination(search.DeploymentID, int32(c.from), int32(c.size), c.reversed)
			results, err := paginated.Paginated(blevesearch.WrapUnsafeSearcherAsSearcher(suite.indexer)).Search(context.Background(), qb)
			require.NoError(t, err)

			resultIDs := search.ResultsToIDs(results)
			if !c.reversed {
				assert.Equal(t, ids[c.from:c.from+c.size], resultIDs)
			} else {
				assert.Equal(t, reversedIds[c.from:c.from+c.size], resultIDs)
			}
		})
	}
}

func TestEnumComparisonSearch(t *testing.T) {
	bleveIndex, err := globalindex.TempInitializeIndices("")
	require.NoError(t, err)
	indexer := New(bleveIndex, bleveIndex)

	cases := []struct {
		prefix          string
		queryLevel      storage.PermissionLevel
		deploymentLevel storage.PermissionLevel
		expectedMatch   bool
	}{
		{
			prefix:          ">=",
			queryLevel:      storage.PermissionLevel_DEFAULT,
			deploymentLevel: storage.PermissionLevel_CLUSTER_ADMIN,
			expectedMatch:   true,
		},
		{
			prefix:          ">=",
			queryLevel:      storage.PermissionLevel_DEFAULT,
			deploymentLevel: storage.PermissionLevel_DEFAULT,
			expectedMatch:   true,
		},
		{
			prefix:          ">=",
			queryLevel:      storage.PermissionLevel_DEFAULT,
			deploymentLevel: storage.PermissionLevel_NONE,
			expectedMatch:   false,
		},
	}
	for _, c := range cases {
		t.Run(fmt.Sprintf("%s%s-%s", c.prefix, c.queryLevel, c.deploymentLevel), func(t *testing.T) {
			d := fixtures.GetDeployment()
			d.ServiceAccountPermissionLevel = c.deploymentLevel
			require.NoError(t, indexer.AddDeployment(d))

			q := search.NewQueryBuilder().AddStringsHighlighted(search.ServiceAccountPermissionLevel, c.prefix+c.queryLevel.String()).ProtoQuery()
			results, err := indexer.Search(ctx, q)
			require.NoError(t, err)
			assert.Equal(t, c.expectedMatch, len(results) == 1)
		})
	}
}

func getDeployment(id string, labels map[string]string) *storage.Deployment {
	d := fixtures.GetDeployment()
	d.Id = id
	d.Labels = labels
	return d
}

func TestMapQueries(t *testing.T) {
	indexer, err := globalindex.TempInitializeIndices("")
	require.NoError(t, err)

	deploymentIndexer := New(indexer, indexer)

	d1 := getDeployment("d1", map[string]string{"h1": "h2", "h3": "h4"})
	d2 := getDeployment("d2", map[string]string{"not-h1": "h2", "h5": "h6", "h7": "h8"})
	d3 := getDeployment("d3", nil)
	d4 := getDeployment("d4", map[string]string{"h1": "not-h2", "h5": "h6", "h7": "h8"})

	require.NoError(t, deploymentIndexer.AddDeployment(d1))
	require.NoError(t, deploymentIndexer.AddDeployment(d2))
	require.NoError(t, deploymentIndexer.AddDeployment(d3))
	require.NoError(t, deploymentIndexer.AddDeployment(d4))

	var cases = []struct {
		key, value  string
		expectedIDs []string
	}{
		// Key and value must exist
		{
			key:         "h1",
			value:       "h2",
			expectedIDs: []string{"d1"},
		},
		// Key must exist and not equal value
		{
			key:         "h1",
			value:       "!h2",
			expectedIDs: []string{"d4"},
		},
		// Key cannot have a value that's not h1 and a value of h2
		{
			key:         "!h1",
			value:       "h2",
			expectedIDs: []string{"d2"},
		},
		// Key cannot have any values that aren't h1 -> h2, also check if value is nil
		{
			key:         "!h1",
			value:       "!h2",
			expectedIDs: []string{"d2", "d3", "d4"},
		},
		// !h1 means that key doesn't exist by itself also check against d3 which is nil
		{
			key:         "!h1",
			expectedIDs: []string{"d2", "d3"},
		},
	}

	for _, c := range cases {
		q := search.NewQueryBuilder().AddMapQuery(search.DeploymentLabel, c.key, c.value).ProtoQuery()
		results, err := deploymentIndexer.Search(ctx, q)
		assert.NoError(t, err)
		assert.ElementsMatch(t, c.expectedIDs, search.ResultsToIDs(results))
	}
}
