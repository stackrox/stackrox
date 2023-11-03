//go:build sql_integration

package m195tom196

import (
	"context"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/migrations/m_195_to_m_196_vuln_request_users/schema/old"
	"github.com/stackrox/rox/migrator/migrations/m_195_to_m_196_vuln_request_users/store/previous"
	"github.com/stackrox/rox/migrator/migrations/m_195_to_m_196_vuln_request_users/store/updated"
	pghelper "github.com/stackrox/rox/migrator/migrations/postgreshelper"
	"github.com/stackrox/rox/migrator/types"
	"github.com/stackrox/rox/pkg/postgres/pgutils"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type migrationTestSuite struct {
	suite.Suite

	db  *pghelper.TestPostgres
	ctx context.Context
}

func TestMigration(t *testing.T) {
	suite.Run(t, new(migrationTestSuite))
}

func (s *migrationTestSuite) SetupSuite() {
	s.ctx = sac.WithAllAccess(context.Background())
	s.db = pghelper.ForT(s.T(), false)

	pgutils.CreateTableFromModel(s.ctx, s.db.GetGormDB(), old.CreateTableVulnerabilityRequestsStmt)
}

func (s *migrationTestSuite) TearDownSuite() {
	s.db.Teardown(s.T())
}

func (s *migrationTestSuite) TestMigration() {
	oldRequests := map[string]*storage.VulnerabilityRequest{
		"1": fakeOldVulnReq("1", "requester-1"),
		"2": fakeOldVulnReq("2", "requester-2", "approver-1"),
		"3": fakeOldVulnReq("3", "requester-2", "approver-2"),
		"4": fakeOldVulnReq("4", "requester-3", "approver-1", "approver-2"),
		"5": fakeOldVulnReq("5", "", ""),
		"6": func() *storage.VulnerabilityRequest {
			r := fakeOldVulnReq("6", "", "")
			r.Requestor = nil
			r.Approvers = nil
			return r
		}(),
	}

	newRequests := map[string]*storage.VulnerabilityRequest{
		"1": fakeNewVulnReq("1", "requester-1"),
		"2": fakeNewVulnReq("2", "requester-2", "approver-1"),
		"3": fakeNewVulnReq("3", "requester-2", "approver-2"),
		"4": fakeNewVulnReq("4", "requester-3", "approver-1", "approver-2"),
		"5": fakeNewVulnReq("5", "", ""),
		"6": func() *storage.VulnerabilityRequest {
			r := fakeNewVulnReq("6", "", "")
			r.Requestor = nil
			r.RequesterV2 = nil
			r.Approvers = nil
			r.ApproversV2 = nil
			return r
		}(),
	}

	oldStore := previous.New(s.db)
	for _, r := range oldRequests {
		require.NoError(s.T(), oldStore.Upsert(s.ctx, r))
	}

	dbs := &types.Databases{
		GormDB:     s.db.GetGormDB(),
		PostgresDB: s.db.DB,
		DBCtx:      s.ctx,
	}

	s.Require().NoError(migration.Run(dbs))

	newStore := updated.New(s.db)
	objs, err := newStore.GetByQuery(s.ctx, search.EmptyQuery())
	assert.NoError(s.T(), err)
	s.verify(newRequests, objs)

	objs, err = newStore.GetByQuery(s.ctx,
		search.NewQueryBuilder().AddExactMatches(search.RequesterUserName, "requester-1").ProtoQuery())
	assert.NoError(s.T(), err)
	s.verify(map[string]*storage.VulnerabilityRequest{
		"1": newRequests["1"],
	}, objs)

	objs, err = newStore.GetByQuery(s.ctx,
		search.NewQueryBuilder().AddExactMatches(search.ApproverUserName, "approver-1").ProtoQuery())
	assert.NoError(s.T(), err)
	s.verify(map[string]*storage.VulnerabilityRequest{
		"2": newRequests["2"],
		"4": newRequests["4"],
	}, objs)

	objs, err = newStore.GetByQuery(s.ctx,
		search.NewQueryBuilder().
			AddExactMatches(search.RequesterUserName, "requester-2").
			AddExactMatches(search.ApproverUserName, "approver-1").ProtoQuery())
	assert.NoError(s.T(), err)
	s.verify(map[string]*storage.VulnerabilityRequest{
		"2": newRequests["2"],
	}, objs)
}

func (s *migrationTestSuite) verify(expected map[string]*storage.VulnerabilityRequest, actual []*storage.VulnerabilityRequest) {
	for _, actualReq := range actual {
		expectedReq := expected[actualReq.GetId()]
		s.NotNil(expectedReq)
		s.EqualValues(expectedReq.GetRequesterV2(), actualReq.GetRequesterV2())
		s.ElementsMatch(expectedReq.GetApproversV2(), actualReq.GetApproversV2())
		s.EqualValues(expectedReq.GetRequestor(), actualReq.GetRequestor())
		s.ElementsMatch(expectedReq.GetApprovers(), actualReq.GetApprovers())
	}
}

func fakeOldVulnReq(id string, requester string, approvers ...string) *storage.VulnerabilityRequest {
	return &storage.VulnerabilityRequest{
		Id:   id,
		Name: id,
		Requestor: &storage.SlimUser{
			Id:   requester,
			Name: requester,
		},
		Approvers: func() []*storage.SlimUser {
			var users []*storage.SlimUser
			for _, approver := range approvers {
				users = append(users, &storage.SlimUser{
					Id:   approver,
					Name: approver,
				})
			}
			return users
		}(),
	}
}

func fakeNewVulnReq(id string, requester string, approvers ...string) *storage.VulnerabilityRequest {
	return &storage.VulnerabilityRequest{
		Id:   id,
		Name: id,
		Requestor: &storage.SlimUser{
			Id:   requester,
			Name: requester,
		},
		RequesterV2: &storage.Requester{
			Id:   requester,
			Name: requester,
		},
		Approvers: func() []*storage.SlimUser {
			var users []*storage.SlimUser
			for _, approver := range approvers {
				users = append(users, &storage.SlimUser{
					Id:   approver,
					Name: approver,
				})
			}
			return users
		}(),
		ApproversV2: func() []*storage.Approver {
			var users []*storage.Approver
			for _, approver := range approvers {
				users = append(users, &storage.Approver{
					Id:   approver,
					Name: approver,
				})
			}
			return users
		}(),
	}
}
