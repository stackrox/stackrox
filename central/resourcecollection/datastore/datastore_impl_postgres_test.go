//go:build sql_integration

package datastore

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/heimdalr/dag"
	"github.com/stackrox/rox/central/resourcecollection/datastore/search"
	pgStore "github.com/stackrox/rox/central/resourcecollection/datastore/store/postgres"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/protoconv"
	"github.com/stackrox/rox/pkg/sac"
	pkgSearch "github.com/stackrox/rox/pkg/search"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

func TestCollectionDataStoreWithPostgres(t *testing.T) {
	suite.Run(t, new(CollectionPostgresDataStoreTestSuite))
}

type CollectionPostgresDataStoreTestSuite struct {
	suite.Suite

	ctx       context.Context
	testDB    *pgtest.TestPostgres
	store     pgStore.Store
	datastore DataStore
	qr        QueryResolver
}

func (s *CollectionPostgresDataStoreTestSuite) SetupSuite() {
	s.ctx = context.Background()
	s.testDB = pgtest.ForT(s.T())

	s.store = pgStore.New(s.testDB)
	index := pgStore.NewIndexer(s.testDB)
	ds, qs, err := New(s.store, search.New(s.store, index))
	s.NoError(err)
	s.datastore = ds
	s.qr = qs
}

// SetupTest removes the local graph before every test
func (s *CollectionPostgresDataStoreTestSuite) SetupTest() {
	s.NoError(resetLocalGraph(s.datastore.(*datastoreImpl)))
}

func (s *CollectionPostgresDataStoreTestSuite) TearDownSuite() {
	s.testDB.Teardown(s.T())
}

func (s *CollectionPostgresDataStoreTestSuite) TestGraphInit() {
	ctx := sac.WithAllAccess(context.Background())

	for _, tc := range []struct {
		desc string
		size int
	}{
		{
			desc: "Test Graph Init small",
			size: 2,
		},
		{
			desc: "Test Graph Init graphInitBatchSize-1",
			size: graphInitBatchSize - 1,
		},
		{
			desc: "Test Graph Init graphInitBatchSize",
			size: graphInitBatchSize,
		},
		{
			desc: "Test Graph Init graphInitBatchSize+1",
			size: graphInitBatchSize + 1,
		},
		{
			desc: "Test Graph Init graphInitBatchSize+2",
			size: graphInitBatchSize + 2,
		},
	} {
		s.T().Run(tc.desc, func(t *testing.T) {
			objs := make([]*storage.ResourceCollection, 0, tc.size)
			objIDs := make([]string, 0, tc.size+1)

			obj := getTestCollection("0", nil)
			obj.Id = "0"
			objs = append(objs, obj)
			objIDs = append(objIDs, "0")
			for i := 1; i < tc.size; i++ {
				edges := make([]string, 0, i)
				for j := 0; j < i; j++ {
					edges = append(edges, fmt.Sprintf("%d", j))
				}
				id := fmt.Sprintf("%d", i)
				obj = getTestCollection(id, edges)
				obj.Id = id
				objs = append(objs, obj)
				objIDs = append(objIDs, id)
			}

			// add objs directly through the store
			err := s.store.UpsertMany(ctx, objs)
			assert.NoError(s.T(), err)

			// trigger graph init
			err = resetLocalGraph(s.datastore.(*datastoreImpl))
			assert.NoError(s.T(), err)

			// get data and check it
			batch, err := s.datastore.GetMany(ctx, objIDs)
			assert.NoError(s.T(), err)
			assert.ElementsMatch(s.T(), objs, batch)

			// clean up data
			for i := len(objIDs) - 1; i >= 0; i-- {
				assert.NoError(s.T(), s.datastore.DeleteCollection(ctx, objIDs[i]))
			}
			assert.NoError(s.T(), resetLocalGraph(s.datastore.(*datastoreImpl)))
		})
	}
}

func (s *CollectionPostgresDataStoreTestSuite) TestCollectionWorkflows() {
	ctx := sac.WithAllAccess(context.Background())

	var err error

	// dryrun add object with an id set
	objID := getTestCollection("id", nil)
	objID.Id = "id"
	err = s.datastore.DryRunAddCollection(ctx, objID)
	assert.Error(s.T(), err)

	// add object with an id set
	_, err = s.datastore.AddCollection(ctx, objID)
	assert.Error(s.T(), err)

	// dryrun add 'a', verify not present
	objA := getTestCollection("a", nil)
	err = s.datastore.DryRunAddCollection(ctx, objA)
	assert.NoError(s.T(), err)
	assert.Equal(s.T(), "", objA.Id)
	count, err := s.datastore.Count(ctx, nil)
	assert.NoError(s.T(), err)
	assert.Equal(s.T(), 0, count)

	// add 'a', verify present
	_, err = s.datastore.AddCollection(ctx, objA)
	assert.NoError(s.T(), err)
	assert.NotEqual(s.T(), "", objA.Id)
	obj, ok, err := s.datastore.Get(ctx, objA.GetId())
	assert.NoError(s.T(), err)
	assert.True(s.T(), ok)
	assert.Equal(s.T(), objA, obj)

	// dryrun add duplicate 'a'
	objADup := getTestCollection("a", nil)
	err = s.datastore.DryRunAddCollection(ctx, objADup)
	assert.Error(s.T(), err)
	assert.Equal(s.T(), "", objADup.Id)

	// add duplicate 'a'
	_, err = s.datastore.AddCollection(ctx, objADup)
	assert.Error(s.T(), err)
	assert.Equal(s.T(), "", objADup.Id)

	// dryrun add 'b' which points to 'a', verify not present
	objB := getTestCollection("b", []string{objA.GetId()})
	err = s.datastore.DryRunAddCollection(ctx, objB)
	assert.NoError(s.T(), err)
	assert.Equal(s.T(), "", objB.Id)
	count, err = s.datastore.Count(ctx, nil)
	assert.NoError(s.T(), err)
	assert.Equal(s.T(), 1, count)

	// add 'b' which points to 'a', verify present
	_, err = s.datastore.AddCollection(ctx, objB)
	assert.NoError(s.T(), err)
	assert.NotEqual(s.T(), "", objB.Id)
	obj, ok, err = s.datastore.Get(ctx, objB.GetId())
	assert.NoError(s.T(), err)
	assert.True(s.T(), ok)
	assert.Equal(s.T(), objB, obj)

	// try to delete 'a' while 'b' points to it
	err = s.datastore.DeleteCollection(ctx, objA.GetId())
	assert.Error(s.T(), err)

	// dryrun update 'a' to point to 'b' which creates a cycle
	objACycle := getTestCollection("a", []string{objB.GetId()})
	objACycle.Id = objA.GetId()
	err = s.datastore.DryRunUpdateCollection(ctx, objACycle)
	assert.Error(s.T(), err)
	_, ok = err.(dag.EdgeLoopError)
	assert.True(s.T(), ok)

	// update 'a' to point to 'b' which creates a cycle
	err = s.datastore.UpdateCollection(ctx, objACycle)
	assert.Error(s.T(), err)
	_, ok = err.(dag.EdgeLoopError)
	assert.True(s.T(), ok)

	// dryrun update 'a' to point to itself which creates a self cycle
	updateTestCollection(objACycle, []string{objA.GetId()})
	err = s.datastore.DryRunUpdateCollection(ctx, objACycle)
	assert.Error(s.T(), err)
	_, ok = err.(dag.SrcDstEqualError)
	assert.True(s.T(), ok)

	// update 'a' to point to itself which creates a self cycle
	err = s.datastore.UpdateCollection(ctx, objACycle)
	assert.Error(s.T(), err)
	_, ok = err.(dag.SrcDstEqualError)
	assert.True(s.T(), ok)

	// dryrun update 'a' with a duplicate name
	objADup.Id = objA.GetId()
	objADup.Name = objB.GetName()
	err = s.datastore.DryRunUpdateCollection(ctx, objADup)
	assert.Error(s.T(), err)

	// update 'a' with a duplicate name
	err = s.datastore.UpdateCollection(ctx, objADup)
	assert.Error(s.T(), err)

	// dryrun update 'a' with a new name
	objADup = objA.Clone()
	objA.Name = "A"
	err = s.datastore.DryRunUpdateCollection(ctx, objA)
	assert.NoError(s.T(), err)
	obj, ok, err = s.datastore.Get(ctx, objA.GetId())
	assert.NoError(s.T(), err)
	assert.True(s.T(), ok)
	assert.Equal(s.T(), objADup, obj)

	// update 'a' with a new name
	err = s.datastore.UpdateCollection(ctx, objA)
	assert.NoError(s.T(), err)
	obj, ok, err = s.datastore.Get(ctx, objA.GetId())
	assert.NoError(s.T(), err)
	assert.True(s.T(), ok)
	assert.Equal(s.T(), objA, obj)

	// add 'e' that points to 'b' and verify
	objE := getTestCollection("e", []string{objB.GetId()})
	_, err = s.datastore.AddCollection(ctx, objE)
	assert.NoError(s.T(), err)
	assert.NotEqual(s.T(), "", objE.Id)
	obj, ok, err = s.datastore.Get(ctx, objE.GetId())
	assert.NoError(s.T(), err)
	assert.True(s.T(), ok)
	assert.Equal(s.T(), objE, obj)

	// dryrun update 'e' to point to only 'a'
	objEDup := objE.Clone()
	updateTestCollection(objE, []string{objA.GetId()})
	err = s.datastore.DryRunUpdateCollection(ctx, objE)
	assert.NoError(s.T(), err)
	obj, ok, err = s.datastore.Get(ctx, objE.GetId())
	assert.NoError(s.T(), err)
	assert.True(s.T(), ok)
	assert.Equal(s.T(), objEDup, obj)

	// update 'e' to point to only 'a', this tests addition and removal of edges
	updateTestCollection(objE, []string{objA.GetId()})
	err = s.datastore.UpdateCollection(ctx, objE)
	assert.NoError(s.T(), err)
	obj, ok, err = s.datastore.Get(ctx, objE.GetId())
	assert.NoError(s.T(), err)
	assert.True(s.T(), ok)
	assert.Equal(s.T(), objE, obj)

	// dryrun update 'b' to point to only 'e'
	objBDup := objB.Clone()
	updateTestCollection(objB, []string{objE.GetId()})
	err = s.datastore.DryRunUpdateCollection(ctx, objB)
	assert.NoError(s.T(), err)
	obj, ok, err = s.datastore.Get(ctx, objB.GetId())
	assert.NoError(s.T(), err)
	assert.True(s.T(), ok)
	assert.Equal(s.T(), objBDup, obj)

	// update 'b' to point to only 'e', making sure the original 'e' -> 'b' edge was removed
	err = s.datastore.UpdateCollection(ctx, objB)
	assert.NoError(s.T(), err)
	obj, ok, err = s.datastore.Get(ctx, objB.GetId())
	assert.NoError(s.T(), err)
	assert.True(s.T(), ok)
	assert.Equal(s.T(), objB, obj)

	// successful updates preserve createdAt and createdBy
	objC := getTestCollection("c", nil)
	objC.CreatedAt = protoconv.ConvertTimeToTimestamp(time.Now())
	objC.CreatedBy = &storage.SlimUser{
		Id:   "uid",
		Name: "uname",
	}
	_, err = s.datastore.AddCollection(ctx, objC)
	assert.NoError(s.T(), err)
	objC.Name = "C"
	err = s.datastore.UpdateCollection(ctx, objC)
	assert.NoError(s.T(), err)
	obj, ok, err = s.datastore.Get(ctx, objC.GetId())
	assert.NoError(s.T(), err)
	assert.True(s.T(), ok)
	assert.Equal(s.T(), objC, obj)

	// clean up testing data and verify the datastore is empty
	assert.NoError(s.T(), s.datastore.DeleteCollection(ctx, objB.GetId()))
	assert.NoError(s.T(), s.datastore.DeleteCollection(ctx, objE.GetId()))
	assert.NoError(s.T(), s.datastore.DeleteCollection(ctx, objA.GetId()))
	assert.NoError(s.T(), s.datastore.DeleteCollection(ctx, objC.GetId()))
	count, err = s.datastore.Count(ctx, nil)
	assert.NoError(s.T(), err)
	assert.Equal(s.T(), 0, count)
}

func (s *CollectionPostgresDataStoreTestSuite) TestVerifyCollectionConstraints() {

	verifyCollectionTests := []struct {
		name          string
		collectionObj *storage.ResourceCollection
		errExpected   bool
	}{
		{
			"nil collection",
			nil,
			true,
		},
		{
			"no selector rules",
			getTestCollectionWithSelectors("name", nil, []*storage.ResourceSelector{
				{
					Rules: []*storage.SelectorRule{},
				},
			}),
			false,
		},
		{
			"1 selector rule",
			getTestCollectionWithSelectors("name", nil, []*storage.ResourceSelector{
				{
					Rules: []*storage.SelectorRule{
						{
							FieldName: pkgSearch.Cluster.String(),
							Operator:  storage.BooleanOperator_OR,
							Values: []*storage.RuleValue{
								{
									Value: "value",
								},
							},
						},
					},
				},
			}),
			false,
		},
		{
			"more than 1 selector rules",
			getTestCollectionWithSelectors("name", nil, []*storage.ResourceSelector{
				{
					Rules: []*storage.SelectorRule{
						{
							FieldName: pkgSearch.Cluster.String(),
							Operator:  storage.BooleanOperator_OR,
							Values: []*storage.RuleValue{
								{
									Value: "value",
								},
							},
						},
					},
				},
				{
					Rules: []*storage.SelectorRule{
						{
							FieldName: pkgSearch.Cluster.String(),
							Operator:  storage.BooleanOperator_OR,
							Values: []*storage.RuleValue{
								{
									Value: "value",
								},
							},
						},
					},
				},
			}),
			true,
		},
		{
			"and operator",
			getTestCollectionWithSelectors("name", nil, []*storage.ResourceSelector{
				{
					Rules: []*storage.SelectorRule{
						{
							FieldName: pkgSearch.Cluster.String(),
							Operator:  storage.BooleanOperator_AND,
							Values: []*storage.RuleValue{
								{
									Value: "value",
								},
							},
						},
					},
				},
			}),
			true,
		},
		{
			"unsupported field name",
			getTestCollectionWithSelectors("name", nil, []*storage.ResourceSelector{
				{
					Rules: []*storage.SelectorRule{
						{
							FieldName: pkgSearch.ClusterRole.String(),
							Operator:  storage.BooleanOperator_OR,
							Values: []*storage.RuleValue{
								{
									Value: "value",
								},
							},
						},
					},
				},
			}),
			true,
		},
		{
			"unknown field name",
			getTestCollectionWithSelectors("name", nil, []*storage.ResourceSelector{
				{
					Rules: []*storage.SelectorRule{
						{
							FieldName: "bad name",
							Operator:  storage.BooleanOperator_OR,
							Values: []*storage.RuleValue{
								{
									Value: "value",
								},
							},
						},
					},
				},
			}),
			true,
		},
		{
			"no rule values with field name set",
			getTestCollectionWithSelectors("name", nil, []*storage.ResourceSelector{
				{
					Rules: []*storage.SelectorRule{
						{
							FieldName: pkgSearch.Cluster.String(),
							Operator:  storage.BooleanOperator_OR,
							Values:    []*storage.RuleValue{},
						},
					},
				},
			}),
			true,
		},
		{
			"bad formatted label value",
			getTestCollectionWithSelectors("name", nil, []*storage.ResourceSelector{
				{
					Rules: []*storage.SelectorRule{
						{
							FieldName: pkgSearch.ClusterLabel.String(),
							Operator:  storage.BooleanOperator_OR,
							Values: []*storage.RuleValue{
								{
									Value: "bad",
								},
							},
						},
					},
				},
			}),
			true,
		},
		{
			"bad match type on label rule",
			getTestCollectionWithSelectors("name", nil, []*storage.ResourceSelector{
				{
					Rules: []*storage.SelectorRule{
						{
							FieldName: pkgSearch.ClusterLabel.String(),
							Operator:  storage.BooleanOperator_OR,
							Values: []*storage.RuleValue{
								{
									Value:     "value",
									MatchType: storage.MatchType_REGEX,
								},
							},
						},
					},
				},
			}),
			true,
		},
	}

	// test all supported field names values
	for _, label := range GetSupportedFieldLabels() {
		verifyCollectionTests = append(verifyCollectionTests, struct {
			name          string
			collectionObj *storage.ResourceCollection
			errExpected   bool
		}{
			fmt.Sprintf("supported label test %s", label.String()),
			getTestCollectionWithSelectors("name", nil, []*storage.ResourceSelector{
				{
					Rules: []*storage.SelectorRule{
						{
							FieldName: label.String(),
							Operator:  storage.BooleanOperator_OR,
							Values: []*storage.RuleValue{
								{
									Value: "key=value",
								},
							},
						},
					},
				},
			}),
			false,
		})
	}

	for _, test := range verifyCollectionTests {
		s.T().Run(test.name, func(t *testing.T) {
			err := verifyCollectionConstraints(test.collectionObj)
			if test.errExpected {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func (s *CollectionPostgresDataStoreTestSuite) TestCollectionToQueries() {

	var supportedLabelRules []*storage.SelectorRule

	for _, label := range GetSupportedFieldLabels() {
		supportedLabelRules = append(supportedLabelRules, &storage.SelectorRule{
			FieldName: label.String(),
			Operator:  storage.BooleanOperator_OR,
		})
	}

	collectionToQueryTests := []struct {
		name              string
		resourceSelectors []*storage.ResourceSelector
		// if this value is nil we expect failure
		expectedQueries []*v1.Query
	}{
		{
			"all supported selector rule field names",
			[]*storage.ResourceSelector{{Rules: supportedLabelRules}},
			[]*v1.Query{},
		},
		{
			"unsupported selector rule field name",
			[]*storage.ResourceSelector{
				{
					Rules: []*storage.SelectorRule{
						{
							FieldName: pkgSearch.ClusterRole.String(),
							Operator:  storage.BooleanOperator_OR,
						},
					},
				},
			},
			nil,
		},
		{
			"unknown selector rule field name",
			[]*storage.ResourceSelector{
				{
					Rules: []*storage.SelectorRule{
						{
							FieldName: "bad label",
							Operator:  storage.BooleanOperator_OR,
						},
					},
				},
			},
			nil,
		},
		{
			"nil resource selector list",
			nil,
			[]*v1.Query{},
		},
		{
			"empty resource selector list",
			[]*storage.ResourceSelector{},
			[]*v1.Query{},
		},
		{
			"two entry resource selector list",
			[]*storage.ResourceSelector{
				{
					Rules: []*storage.SelectorRule{
						{
							FieldName: pkgSearch.Cluster.String(),
							Operator:  storage.BooleanOperator_OR,
							Values: []*storage.RuleValue{
								{
									Value: "1",
								},
								{
									Value: "2",
								},
							},
						},
					},
				},
				{
					Rules: []*storage.SelectorRule{
						{
							FieldName: pkgSearch.Namespace.String(),
							Operator:  storage.BooleanOperator_OR,
							Values: []*storage.RuleValue{
								{
									Value: "3",
								},
							},
						},
					},
				},
			},
			[]*v1.Query{
				{
					Query: &v1.Query_Disjunction{
						Disjunction: &v1.DisjunctionQuery{
							Queries: []*v1.Query{
								{
									Query: getBaseQuery(pkgSearch.Cluster, "\"1\""),
								},
								{
									Query: getBaseQuery(pkgSearch.Cluster, "\"2\""),
								},
							},
						},
					},
				},
				{
					Query: getBaseQuery(pkgSearch.Namespace, "\"3\""),
				},
			},
		},
		{
			"nil selector rule list",
			[]*storage.ResourceSelector{
				{
					Rules: nil,
				},
			},
			[]*v1.Query{},
		},
		{
			"empty selector rule list",
			[]*storage.ResourceSelector{
				{
					Rules: []*storage.SelectorRule{},
				},
			},
			[]*v1.Query{},
		},
		{
			"regex match rule",
			[]*storage.ResourceSelector{
				{
					Rules: []*storage.SelectorRule{
						{
							FieldName: pkgSearch.Cluster.String(),
							Operator:  storage.BooleanOperator_OR,
							Values: []*storage.RuleValue{
								{
									Value:     "1",
									MatchType: storage.MatchType_REGEX,
								},
							},
						},
					},
				},
			},
			[]*v1.Query{
				{
					Query: getBaseQuery(pkgSearch.Cluster, "r/1"),
				},
			},
		},
	}

	for _, test := range collectionToQueryTests {
		s.T().Run(test.name, func(t *testing.T) {
			testCollection := getTestCollection("1", nil)
			testCollection.ResourceSelectors = test.resourceSelectors
			parsedQueries, err := collectionToQueries(testCollection)
			if test.expectedQueries == nil {
				assert.Error(t, err)
				assert.Nil(s.T(), parsedQueries)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, len(test.expectedQueries), len(parsedQueries))
				for i := 0; i < len(test.expectedQueries) && i < len(parsedQueries); i++ {
					assert.Equal(t, test.expectedQueries[i], parsedQueries[i])
				}
			}
		})
	}
}

func (s *CollectionPostgresDataStoreTestSuite) TestResolveCollectionQuery() {
	ctx := sac.WithAllAccess(context.Background())

	var err error

	// upsert collections and reference those as embedded, make sure we resolve the embedded collections
	objA := getTestCollection("a", nil)
	objA.ResourceSelectors = []*storage.ResourceSelector{
		{
			Rules: []*storage.SelectorRule{
				{
					FieldName: pkgSearch.Cluster.String(),
					Operator:  storage.BooleanOperator_OR,
					Values: []*storage.RuleValue{
						{
							Value: "1",
						},
					},
				},
			},
		},
	}
	_, err = s.datastore.AddCollection(ctx, objA)
	s.NoError(err)
	objB := getTestCollection("b", []string{objA.GetId()})
	objB.ResourceSelectors = []*storage.ResourceSelector{
		{
			Rules: []*storage.SelectorRule{
				{
					FieldName: pkgSearch.Namespace.String(),
					Operator:  storage.BooleanOperator_OR,
					Values: []*storage.RuleValue{
						{
							Value: "2",
						},
					},
				},
			},
		},
	}
	_, err = s.datastore.AddCollection(ctx, objB)
	s.NoError(err)
	objC := getTestCollection("c", []string{objB.GetId()})
	objC.ResourceSelectors = []*storage.ResourceSelector{
		{
			Rules: []*storage.SelectorRule{
				{
					FieldName: pkgSearch.DeploymentName.String(),
					Operator:  storage.BooleanOperator_OR,
					Values: []*storage.RuleValue{
						{
							Value: "3",
						},
					},
				},
			},
		},
	}
	_, err = s.datastore.AddCollection(ctx, objC)
	s.NoError(err)

	expectedQuery := &v1.Query{
		Query: &v1.Query_Disjunction{
			Disjunction: &v1.DisjunctionQuery{
				Queries: []*v1.Query{
					{
						Query: getBaseQuery(pkgSearch.DeploymentName, "\"3\""),
					},
					{
						Query: getBaseQuery(pkgSearch.Namespace, "\"2\""),
					},
					{
						Query: getBaseQuery(pkgSearch.Cluster, "\"1\""),
					},
				},
			},
		},
	}
	testObj := getTestCollection("test", []string{objC.GetId()})
	query, err := s.qr.ResolveCollectionQuery(ctx, testObj)
	assert.NoError(s.T(), err)
	assert.Equal(s.T(), expectedQuery.String(), query.String())

	expectedQuery = &v1.Query{
		Query: &v1.Query_Disjunction{
			Disjunction: &v1.DisjunctionQuery{
				Queries: []*v1.Query{
					{
						Query: getBaseQuery(pkgSearch.Cluster, "\"1\""),
					},
					{
						Query: getBaseQuery(pkgSearch.Namespace, "\"2\""),
					},
				},
			},
		},
	}
	testObj = getTestCollection("test", []string{objA.GetId(), objB.GetId()})
	query, err = s.qr.ResolveCollectionQuery(ctx, testObj)
	assert.NoError(s.T(), err)
	assert.Equal(s.T(), expectedQuery.String(), query.String())
}

func (s *CollectionPostgresDataStoreTestSuite) TestFoo() {
	// TODO e2e testing ROX-12626
	// test regex doesn't compile
}

func getTestCollectionWithSelectors(name string, ids []string, selectors []*storage.ResourceSelector) *storage.ResourceCollection {
	ret := getTestCollection(name, ids)
	ret.ResourceSelectors = selectors
	return ret
}

func getTestCollection(name string, ids []string) *storage.ResourceCollection {
	return &storage.ResourceCollection{
		Name:                name,
		EmbeddedCollections: getEmbeddedTestCollection(ids),
	}
}

func updateTestCollection(obj *storage.ResourceCollection, ids []string) *storage.ResourceCollection {
	obj.EmbeddedCollections = getEmbeddedTestCollection(ids)
	return obj
}

func getEmbeddedTestCollection(ids []string) []*storage.ResourceCollection_EmbeddedResourceCollection {
	var embedded []*storage.ResourceCollection_EmbeddedResourceCollection
	if ids != nil {
		embedded = make([]*storage.ResourceCollection_EmbeddedResourceCollection, 0, len(ids))
		for _, i := range ids {
			embedded = append(embedded, &storage.ResourceCollection_EmbeddedResourceCollection{Id: i})
		}
	}
	return embedded
}

func resetLocalGraph(ds *datastoreImpl) error {
	if ds.graph != nil {
		ds.graph = nil
	}
	return ds.initGraph()
}

func getBaseQuery(field pkgSearch.FieldLabel, value string) *v1.Query_BaseQuery {
	return &v1.Query_BaseQuery{
		BaseQuery: &v1.BaseQuery{
			Query: &v1.BaseQuery_MatchFieldQuery{
				MatchFieldQuery: &v1.MatchFieldQuery{
					Field: field.String(),
					Value: value,
				},
			},
		},
	}
}
