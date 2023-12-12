package postgrestests

import (
	"context"
	"testing"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/sac"
	reportconfigstore "github.com/stackrox/rox/pkg/search/postgres/tests/reportconfigstore"
	"github.com/stretchr/testify/assert"
)

func getTestReportConfig(id string, name string) *storage.ReportConfiguration {
	return &storage.ReportConfiguration{
		Id:   id,
		Name: name,
	}
}

const (
	identifier1 = "ID 1"
	identifier2 = "ID 2"
	identifier3 = "ID 3"
	identifier4 = "ID 4"
	identifier5 = "ID 5"

	name1 = "Report 1"
	name2 = "Report 2"
	name3 = "Report 3"
	name4 = "Report 4"
	name5 = "Report 5"
)

func getTestReportConfigs() []*storage.ReportConfiguration {
	return []*storage.ReportConfiguration{
		getTestReportConfig(identifier1, name1),
		getTestReportConfig(identifier2, name2),
		getTestReportConfig(identifier3, name3),
		getTestReportConfig(identifier4, name4),
		getTestReportConfig(identifier5, name5),
	}
}

func getReportNameQuery(reportName string) *v1.Query {
	return &v1.Query{
		Query: &v1.Query_BaseQuery{
			BaseQuery: &v1.BaseQuery{
				Query: &v1.BaseQuery_MatchFieldQuery{
					MatchFieldQuery: &v1.MatchFieldQuery{
						Field:     "Report Name",
						Value:     reportName,
						Highlight: false,
					},
				},
			},
		},
	}
}

func TestGloballyScopedDeleteByQuery(t *testing.T) {
	testDB := pgtest.ForT(t)
	reportConfigStore := reportconfigstore.New(testDB.DB)
	ctx := sac.WithAllAccess(context.Background())
	assert.NoError(t, reportConfigStore.UpsertMany(ctx, getTestReportConfigs()))
	query := &v1.Query{
		Query: &v1.Query_Disjunction{
			Disjunction: &v1.DisjunctionQuery{
				Queries: []*v1.Query{
					getReportNameQuery(name2),
					getReportNameQuery(name4),
				},
			},
		},
	}
	obj1, found1, err1 := reportConfigStore.Get(ctx, identifier1)
	assert.Equal(t, getTestReportConfig(identifier1, name1), obj1)
	assert.True(t, found1)
	assert.NoError(t, err1)
	obj2, found2, err2 := reportConfigStore.Get(ctx, identifier2)
	assert.Nil(t, obj2)
	assert.False(t, found2)
	assert.NoError(t, err2)
	obj3, found3, err3 := reportConfigStore.Get(ctx, identifier3)
	assert.Equal(t, getTestReportConfig(identifier3, name3), obj3)
	assert.True(t, found3)
	assert.NoError(t, err3)
	obj4, found4, err4 := reportConfigStore.Get(ctx, identifier4)
	assert.Nil(t, obj4)
	assert.False(t, found4)
	assert.NoError(t, err4)
	obj5, found5, err5 := reportConfigStore.Get(ctx, identifier5)
	assert.Equal(t, getTestReportConfig(identifier5, name5), obj5)
	assert.True(t, found5)
	assert.NoError(t, err5)
	assert.NoError(t, reportConfigStore.DeleteByQuery(ctx, query))
}
