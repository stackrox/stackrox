//go:build sql_integration

package m200tom201

import (
	"context"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/migrations/m_201_to_m_202_vuln_request_v1_to_v2/schema"
	"github.com/stackrox/rox/migrator/migrations/m_201_to_m_202_vuln_request_v1_to_v2/store"
	pghelper "github.com/stackrox/rox/migrator/migrations/postgreshelper"
	"github.com/stackrox/rox/migrator/types"
	"github.com/stackrox/rox/pkg/postgres/pgutils"
	"github.com/stackrox/rox/pkg/protoassert"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

var (
	ts               = protocompat.TimestampNow()
	globalImageScope = &storage.VulnerabilityRequest_Scope{
		Info: &storage.VulnerabilityRequest_Scope_ImageScope{
			ImageScope: &storage.VulnerabilityRequest_Scope_Image{
				Registry: ".*",
				Remote:   ".*",
				Tag:      ".*",
			},
		},
	}
)

type migrationTestSuite struct {
	suite.Suite

	db  *pghelper.TestPostgres
	ctx context.Context
}

type requsetScope struct {
	isGlobal bool
	registry string
	remote   string
	tag      string
}

type requestParams struct {
	id            string
	expiry        *storage.RequestExpiry
	targetSate    storage.VulnerabilityState
	requester     string
	approvers     []string
	scope         *requsetScope
	updatedExpiry *storage.RequestExpiry
}

func TestMigration(t *testing.T) {
	suite.Run(t, new(migrationTestSuite))
}

func (s *migrationTestSuite) SetupSuite() {
	s.ctx = sac.WithAllAccess(context.Background())
	s.db = pghelper.ForT(s.T(), false)

	pgutils.CreateTableFromModel(s.ctx, s.db.GetGormDB(), schema.CreateTableVulnerabilityRequestsStmt)
}

func (s *migrationTestSuite) TestMigration() {
	oldRequests := map[string]*storage.VulnerabilityRequest{
		"1": fakeOldVulnReq(&requestParams{
			id:            "1",
			expiry:        oldRequestExpiry(false),
			targetSate:    storage.VulnerabilityState_DEFERRED,
			requester:     "requester-1",
			scope:         &requsetScope{isGlobal: false, registry: "reg-1", remote: "remote-1", tag: "tag-1"},
			updatedExpiry: nil,
		}),
		"2": fakeOldVulnReq(&requestParams{
			id:            "2",
			expiry:        oldRequestExpiry(true),
			targetSate:    storage.VulnerabilityState_DEFERRED,
			requester:     "requester-2",
			approvers:     []string{"approver-1"},
			scope:         &requsetScope{isGlobal: false, registry: "reg-2", remote: "remote-1", tag: "tag-1"},
			updatedExpiry: nil,
		}),
		"3": fakeOldVulnReq(&requestParams{
			id:            "3",
			expiry:        oldRequestExpiry(false),
			targetSate:    storage.VulnerabilityState_DEFERRED,
			requester:     "requester-2",
			approvers:     []string{"approver-2"},
			scope:         &requsetScope{isGlobal: true},
			updatedExpiry: oldRequestExpiry(true),
		}),
		"4": fakeOldVulnReq(&requestParams{
			id:            "4",
			expiry:        oldRequestExpiry(true),
			targetSate:    storage.VulnerabilityState_DEFERRED,
			requester:     "requester-3",
			approvers:     []string{"approver-1", "approver-2"},
			scope:         &requsetScope{isGlobal: true},
			updatedExpiry: oldRequestExpiry(false),
		}),
		"5": fakeOldVulnReq(&requestParams{
			id:         "5",
			targetSate: storage.VulnerabilityState_FALSE_POSITIVE,
			requester:  "requester-3",
			approvers:  []string{"approver-1", "approver-2"},
			scope:      &requsetScope{isGlobal: false, registry: "reg-2", remote: "remote-1", tag: ""},
		}),
		"6": fakeOldVulnReq(&requestParams{
			id:         "6",
			targetSate: storage.VulnerabilityState_FALSE_POSITIVE,
			requester:  "requester-3",
			approvers:  []string{"approver-1", "approver-2"},
			scope:      &requsetScope{isGlobal: true},
		}),
		"7": func() *storage.VulnerabilityRequest {
			r := fakeOldVulnReq(&requestParams{
				id:         "7",
				expiry:     oldRequestExpiry(true),
				targetSate: storage.VulnerabilityState_DEFERRED,
				requester:  "",
				scope:      &requsetScope{isGlobal: false, registry: "reg-2", remote: "remote-1", tag: ".*"},
			})
			r.Requestor = nil
			r.Approvers = nil
			return r
		}(),
		"8": func() *storage.VulnerabilityRequest {
			r := fakeOldVulnReq(&requestParams{
				id:         "8",
				expiry:     oldRequestExpiry(false),
				targetSate: storage.VulnerabilityState_DEFERRED,
				requester:  "requester-4",
				scope:      &requsetScope{isGlobal: false, registry: "reg-2", remote: "", tag: ""},
			})
			r.Req = nil
			return r
		}(),
		"9": func() *storage.VulnerabilityRequest {
			r := fakeOldVulnReq(&requestParams{
				id:         "9",
				expiry:     oldRequestExpiry(true),
				targetSate: storage.VulnerabilityState_DEFERRED,
				requester:  "requester-4",
				scope:      &requsetScope{isGlobal: true},
			})
			r.GetDeferralReq().Expiry = nil
			return r
		}(),
	}

	newRequests := map[string]*storage.VulnerabilityRequest{
		"1": fakeNewVulnReq(&requestParams{
			id:            "1",
			expiry:        newRequestExpiry(false),
			targetSate:    storage.VulnerabilityState_DEFERRED,
			requester:     "requester-1",
			scope:         &requsetScope{isGlobal: false, registry: "reg-1", remote: "remote-1", tag: "tag-1"},
			updatedExpiry: nil,
		}),
		"2": fakeNewVulnReq(&requestParams{
			id:            "2",
			expiry:        newRequestExpiry(true),
			targetSate:    storage.VulnerabilityState_DEFERRED,
			requester:     "requester-2",
			approvers:     []string{"approver-1"},
			scope:         &requsetScope{isGlobal: false, registry: "reg-2", remote: "remote-1", tag: "tag-1"},
			updatedExpiry: nil,
		}),
		"3": fakeNewVulnReq(&requestParams{
			id:            "3",
			expiry:        newRequestExpiry(false),
			targetSate:    storage.VulnerabilityState_DEFERRED,
			requester:     "requester-2",
			approvers:     []string{"approver-2"},
			scope:         &requsetScope{isGlobal: true},
			updatedExpiry: newRequestExpiry(true),
		}),
		"4": fakeNewVulnReq(&requestParams{
			id:            "4",
			expiry:        newRequestExpiry(true),
			targetSate:    storage.VulnerabilityState_DEFERRED,
			requester:     "requester-3",
			approvers:     []string{"approver-1", "approver-2"},
			scope:         &requsetScope{isGlobal: true},
			updatedExpiry: newRequestExpiry(false),
		}),
		"5": fakeNewVulnReq(&requestParams{
			id:         "5",
			targetSate: storage.VulnerabilityState_FALSE_POSITIVE,
			requester:  "requester-3",
			approvers:  []string{"approver-1", "approver-2"},
			scope:      &requsetScope{isGlobal: false, registry: "reg-2", remote: "remote-1", tag: ""},
		}),
		"6": fakeNewVulnReq(&requestParams{
			id:         "6",
			targetSate: storage.VulnerabilityState_FALSE_POSITIVE,
			requester:  "requester-3",
			approvers:  []string{"approver-1", "approver-2"},
			scope:      &requsetScope{isGlobal: true},
		}),
		"7": func() *storage.VulnerabilityRequest {
			r := fakeNewVulnReq(&requestParams{
				id:         "7",
				expiry:     newRequestExpiry(true),
				targetSate: storage.VulnerabilityState_DEFERRED,
				requester:  "",
				scope:      &requsetScope{isGlobal: false, registry: "reg-2", remote: "remote-1", tag: ".*"},
			})
			r.Requestor = nil
			r.RequesterV2 = nil
			r.Approvers = nil
			r.ApproversV2 = nil
			return r
		}(),
		"8": func() *storage.VulnerabilityRequest {
			r := fakeNewVulnReq(&requestParams{
				id:         "8",
				expiry:     newRequestExpiry(false),
				targetSate: storage.VulnerabilityState_DEFERRED,
				requester:  "requester-4",
				scope:      &requsetScope{isGlobal: false, registry: "reg-2", remote: "", tag: ""},
			})
			r.Req = nil
			return r
		}(),
		"9": func() *storage.VulnerabilityRequest {
			r := fakeNewVulnReq(&requestParams{
				id:         "9",
				expiry:     newRequestExpiry(true),
				targetSate: storage.VulnerabilityState_DEFERRED,
				requester:  "requester-4",
				scope:      &requsetScope{isGlobal: true},
			})
			r.GetDeferralReq().Expiry = nil
			return r
		}(),
	}

	vulnReqStore := store.New(s.db)
	for _, r := range oldRequests {
		require.NoError(s.T(), vulnReqStore.Upsert(s.ctx, r))
	}

	dbs := &types.Databases{
		GormDB:     s.db.GetGormDB(),
		PostgresDB: s.db.DB,
		DBCtx:      s.ctx,
	}

	s.Require().NoError(migration.Run(dbs))

	objs, err := vulnReqStore.GetByQuery(s.ctx, search.EmptyQuery())
	assert.NoError(s.T(), err)
	s.verify(newRequests, objs)

	objs, err = vulnReqStore.GetByQuery(s.ctx,
		search.NewQueryBuilder().AddExactMatches(search.ImageRegistryScope, "reg-1").ProtoQuery())
	assert.NoError(s.T(), err)
	s.verify(map[string]*storage.VulnerabilityRequest{
		"1": newRequests["1"],
	}, objs)

	objs, err = vulnReqStore.GetByQuery(s.ctx,
		search.NewQueryBuilder().AddExactMatches(search.ApproverUserName, "approver-1").ProtoQuery())
	assert.NoError(s.T(), err)
	s.verify(map[string]*storage.VulnerabilityRequest{
		"2": newRequests["2"],
		"4": newRequests["4"],
		"5": newRequests["5"],
		"6": newRequests["6"],
	}, objs)

	objs, err = vulnReqStore.GetByQuery(s.ctx,
		search.NewQueryBuilder().AddExactMatches(search.DeferralUpdateCVEs, "cve-1").ProtoQuery())
	assert.NoError(s.T(), err)
	s.verify(map[string]*storage.VulnerabilityRequest{
		"3": newRequests["3"],
		"4": newRequests["4"],
	}, objs)

	objs, err = vulnReqStore.GetByQuery(s.ctx,
		search.NewQueryBuilder().
			AddExactMatches(search.RequesterUserName, "requester-2").
			AddExactMatches(search.ApproverUserName, "approver-1").ProtoQuery())
	assert.NoError(s.T(), err)
	s.verify(map[string]*storage.VulnerabilityRequest{
		"2": newRequests["2"],
	}, objs)

	objs, err = vulnReqStore.GetByQuery(s.ctx,
		search.NewQueryBuilder().AddExactMatches(search.ImageRegistryScope, ".*").ProtoQuery())
	assert.NoError(s.T(), err)
	s.verify(map[string]*storage.VulnerabilityRequest{
		"3": newRequests["3"],
		"4": newRequests["4"],
		"6": newRequests["6"],
		"9": newRequests["9"],
	}, objs)

	objs, err = vulnReqStore.GetByQuery(s.ctx,
		search.NewQueryBuilder().AddRegexes(search.ImageRemoteScope, ".*").ProtoQuery())
	assert.NoError(s.T(), err)
	s.verify(newRequests, objs)
}

func (s *migrationTestSuite) verify(expected map[string]*storage.VulnerabilityRequest, actual []*storage.VulnerabilityRequest) {
	for _, actualReq := range actual {
		expectedReq := expected[actualReq.GetId()]
		s.NotNil(expectedReq)
		protoassert.Equal(s.T(), expectedReq.GetRequesterV2(), actualReq.GetRequesterV2())
		protoassert.ElementsMatch(s.T(), expectedReq.GetApproversV2(), actualReq.GetApproversV2())
		protoassert.Equal(s.T(), expectedReq.GetRequestor(), actualReq.GetRequestor())
		protoassert.ElementsMatch(s.T(), expectedReq.GetApprovers(), actualReq.GetApprovers())
		protoassert.Equal(s.T(), expectedReq.GetDeferralReq(), actualReq.GetDeferralReq())
		protoassert.Equal(s.T(), expectedReq.GetUpdatedDeferralReq(), actualReq.GetUpdatedDeferralReq())
		protoassert.Equal(s.T(), expectedReq.GetDeferralUpdate(), actualReq.GetDeferralUpdate())
		protoassert.Equal(s.T(), expectedReq.GetFalsePositiveUpdate(), actualReq.GetFalsePositiveUpdate())
		protoassert.Equal(s.T(), expectedReq.Scope, actualReq.Scope)
	}
}

func fakeOldVulnReq(reqParams *requestParams) *storage.VulnerabilityRequest {
	ret := &storage.VulnerabilityRequest{
		Id:   reqParams.id,
		Name: reqParams.id,
		Requestor: &storage.SlimUser{
			Id:   reqParams.requester,
			Name: reqParams.requester,
		},
		Approvers: func() []*storage.SlimUser {
			var users []*storage.SlimUser
			for _, approver := range reqParams.approvers {
				users = append(users, &storage.SlimUser{
					Id:   approver,
					Name: approver,
				})
			}
			return users
		}(),
		TargetState: reqParams.targetSate,
		Entities: &storage.VulnerabilityRequest_Cves{
			Cves: &storage.VulnerabilityRequest_CVEs{
				Cves: []string{"cve-1"},
			},
		},
	}

	if reqParams.scope != nil {
		if reqParams.scope.isGlobal {
			ret.Scope = &storage.VulnerabilityRequest_Scope{
				Info: &storage.VulnerabilityRequest_Scope_GlobalScope{GlobalScope: &storage.VulnerabilityRequest_Scope_Global{}},
			}
		} else {
			ret.Scope = &storage.VulnerabilityRequest_Scope{
				Info: &storage.VulnerabilityRequest_Scope_ImageScope{
					ImageScope: &storage.VulnerabilityRequest_Scope_Image{
						Registry: reqParams.scope.registry,
						Remote:   reqParams.scope.remote,
						Tag:      reqParams.scope.tag,
					},
				},
			}
		}
	}

	if reqParams.targetSate == storage.VulnerabilityState_DEFERRED {
		ret.Req = &storage.VulnerabilityRequest_DeferralReq{
			DeferralReq: &storage.DeferralRequest{
				Expiry: reqParams.expiry,
			},
		}
	} else if reqParams.targetSate == storage.VulnerabilityState_FALSE_POSITIVE {
		ret.Req = &storage.VulnerabilityRequest_FpRequest{FpRequest: &storage.FalsePositiveRequest{}}
	}

	if reqParams.updatedExpiry != nil {
		ret.UpdatedReq = &storage.VulnerabilityRequest_UpdatedDeferralReq{
			UpdatedDeferralReq: &storage.DeferralRequest{
				Expiry: reqParams.updatedExpiry,
			},
		}
	}

	return ret
}

func fakeNewVulnReq(reqParams *requestParams) *storage.VulnerabilityRequest {
	ret := &storage.VulnerabilityRequest{
		Id:   reqParams.id,
		Name: reqParams.id,
		Requestor: &storage.SlimUser{
			Id:   reqParams.requester,
			Name: reqParams.requester,
		},
		RequesterV2: &storage.Requester{
			Id:   reqParams.requester,
			Name: reqParams.requester,
		},
		Approvers: func() []*storage.SlimUser {
			var users []*storage.SlimUser
			for _, approver := range reqParams.approvers {
				users = append(users, &storage.SlimUser{
					Id:   approver,
					Name: approver,
				})
			}
			return users
		}(),
		ApproversV2: func() []*storage.Approver {
			var users []*storage.Approver
			for _, approver := range reqParams.approvers {
				users = append(users, &storage.Approver{
					Id:   approver,
					Name: approver,
				})
			}
			return users
		}(),
		TargetState: reqParams.targetSate,
		Entities: &storage.VulnerabilityRequest_Cves{
			Cves: &storage.VulnerabilityRequest_CVEs{
				Cves: []string{"cve-1"},
			},
		},
	}

	if reqParams.scope != nil {
		if reqParams.scope.isGlobal {
			ret.Scope = globalImageScope
		} else {
			ret.Scope = &storage.VulnerabilityRequest_Scope{
				Info: &storage.VulnerabilityRequest_Scope_ImageScope{
					ImageScope: &storage.VulnerabilityRequest_Scope_Image{
						Registry: reqParams.scope.registry,
						Remote:   reqParams.scope.remote,
						Tag:      reqParams.scope.tag,
					},
				},
			}
		}
	}

	if reqParams.targetSate == storage.VulnerabilityState_DEFERRED {
		ret.Req = &storage.VulnerabilityRequest_DeferralReq{
			DeferralReq: &storage.DeferralRequest{
				Expiry: reqParams.expiry,
			},
		}
	}

	if reqParams.updatedExpiry != nil {
		ret.UpdatedReq = &storage.VulnerabilityRequest_DeferralUpdate{
			DeferralUpdate: &storage.DeferralUpdate{
				CVEs:   []string{"cve-1"},
				Expiry: reqParams.updatedExpiry,
			},
		}
	}

	return ret
}

func oldRequestExpiry(expiresWhenCVEFixable bool) *storage.RequestExpiry {
	if expiresWhenCVEFixable {
		return &storage.RequestExpiry{
			Expiry: &storage.RequestExpiry_ExpiresWhenFixed{
				ExpiresWhenFixed: true,
			},
		}
	}

	return &storage.RequestExpiry{
		Expiry: &storage.RequestExpiry_ExpiresOn{
			ExpiresOn: ts,
		},
	}
}

func newRequestExpiry(expiresWhenCVEFixable bool) *storage.RequestExpiry {
	if expiresWhenCVEFixable {
		return &storage.RequestExpiry{
			Expiry: &storage.RequestExpiry_ExpiresWhenFixed{
				ExpiresWhenFixed: true,
			},
			ExpiryType: storage.RequestExpiry_ANY_CVE_FIXABLE,
		}
	}

	return &storage.RequestExpiry{
		Expiry: &storage.RequestExpiry_ExpiresOn{
			ExpiresOn: ts,
		},
		ExpiryType: storage.RequestExpiry_TIME,
	}
}
