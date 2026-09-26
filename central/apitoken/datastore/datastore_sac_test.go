//go:build sql_integration

package datastore

import (
	"context"
	"testing"
	"time"

	pgScheduleStore "github.com/stackrox/rox/central/apitoken/datastore/internal/schedulestore/postgres"
	pgTokenStore "github.com/stackrox/rox/central/apitoken/datastore/internal/store/postgres"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/protoassert"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/suite"
)

var (
	timestamp1 = time.Date(2020, time.August, 1, 20, 30, 00, 0, time.UTC)
	timestamp2 = time.Date(2021, time.September, 2, 21, 40, 00, 0, time.UTC)
	timestamp3 = time.Date(2022, time.October, 3, 22, 50, 00, 0, time.UTC)
	timestamp4 = time.Date(2023, time.November, 4, 23, 55, 00, 0, time.UTC)
)

func TestAPITokenDataStoreSAC(t *testing.T) {
	suite.Run(t, new(apiTokenDatastoreSACTestSuite))
}

type apiTokenDatastoreSACTestSuite struct {
	suite.Suite

	noPermissionCtx    context.Context
	readOnlyCtx        context.Context
	readWriteCtx       context.Context
	otherPermissionCtx context.Context

	pool postgres.DB

	pgStore pgTokenStore.Store
	store   DataStore

	// Data for read tests
	testToken1 *storage.TokenMetadata
	testToken2 *storage.TokenMetadata
}

func (s *apiTokenDatastoreSACTestSuite) SetupSuite() {
	s.noPermissionCtx = sac.WithNoAccess(s.T().Context())
	s.readOnlyCtx = sac.WithGlobalAccessScopeChecker(
		s.T().Context(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
			sac.ResourceScopeKeys(resources.Integration),
		),
	)
	s.readWriteCtx = sac.WithGlobalAccessScopeChecker(
		s.T().Context(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.Integration),
		),
	)
	s.otherPermissionCtx = sac.WithGlobalAccessScopeChecker(
		s.T().Context(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.Notifications),
		),
	)
}

func (s *apiTokenDatastoreSACTestSuite) SetupTest() {
	s.pool = pgtest.ForT(s.T())
	s.pgStore = pgTokenStore.New(s.pool)
	s.store = NewPostgres(s.pool)

	// Populate DB for read tests
	s.testToken1 = &storage.TokenMetadata{
		Id:         uuid.NewTestUUID(1).String(),
		Name:       "test token 1",
		Roles:      []string{"Test Role 1"},
		IssuedAt:   protocompat.ConvertTimeToTimestampOrNil(&timestamp1),
		Expiration: protocompat.ConvertTimeToTimestampOrNil(&timestamp2),
		Revoked:    true,
	}
	s.testToken2 = &storage.TokenMetadata{
		Id:         uuid.NewTestUUID(2).String(),
		Name:       "test token 2",
		Roles:      []string{"Test Role 2", "Test Role 3"},
		IssuedAt:   protocompat.ConvertTimeToTimestampOrNil(&timestamp3),
		Expiration: protocompat.ConvertTimeToTimestampOrNil(&timestamp4),
		Revoked:    false,
	}
	allAccessCtx := sac.WithAllAccess(s.T().Context())
	s.Require().NoError(s.pgStore.UpsertMany(allAccessCtx, []*storage.TokenMetadata{s.testToken1, s.testToken2}))
}

func (s *apiTokenDatastoreSACTestSuite) TearDownTest() {
	s.pool.Close()
}

type searchTestCase struct {
	ctx            context.Context
	getQuery       *v1.GetAPITokensRequest
	searchQuery    *v1.Query
	expectedResult []*storage.TokenMetadata
	expectedSACErr error
}

func (s *apiTokenDatastoreSACTestSuite) getSearchTestCases() map[string]searchTestCase {
	return map[string]searchTestCase{
		"No access does not retrieve any token (nil request)": {
			ctx:            s.noPermissionCtx,
			getQuery:       nil,
			searchQuery:    nil,
			expectedResult: nil,
			expectedSACErr: sac.ErrResourceAccessDenied,
		},
		"No access does not retrieve any token (full request)": {
			ctx:            s.noPermissionCtx,
			getQuery:       &v1.GetAPITokensRequest{},
			searchQuery:    search.EmptyQuery(),
			expectedResult: nil,
			expectedSACErr: sac.ErrResourceAccessDenied,
		},
		"No access does not retrieve any token (revoked request)": {
			ctx: s.noPermissionCtx,
			getQuery: &v1.GetAPITokensRequest{
				RevokedOneof: &v1.GetAPITokensRequest_Revoked{
					Revoked: true,
				},
			},
			searchQuery:    search.NewQueryBuilder().AddBools(search.Revoked, true).ProtoQuery(),
			expectedResult: nil,
			expectedSACErr: sac.ErrResourceAccessDenied,
		},
		"No access does not retrieve any token (non-revoked request)": {
			ctx: s.noPermissionCtx,
			getQuery: &v1.GetAPITokensRequest{
				RevokedOneof: &v1.GetAPITokensRequest_Revoked{
					Revoked: false,
				},
			},
			searchQuery:    search.NewQueryBuilder().AddBools(search.Revoked, false).ProtoQuery(),
			expectedResult: nil,
			expectedSACErr: sac.ErrResourceAccessDenied,
		},
		"Read-only retrieves all tokens (nil request)": {
			ctx:            s.readOnlyCtx,
			getQuery:       nil,
			searchQuery:    nil,
			expectedResult: []*storage.TokenMetadata{s.testToken1, s.testToken2},
		},
		"Read-only retrieves all tokens (full request)": {
			ctx:            s.readOnlyCtx,
			getQuery:       &v1.GetAPITokensRequest{},
			searchQuery:    search.EmptyQuery(),
			expectedResult: []*storage.TokenMetadata{s.testToken1, s.testToken2},
		},
		"Read-only retrieves revoked tokens (revoked request)": {
			ctx: s.readOnlyCtx,
			getQuery: &v1.GetAPITokensRequest{
				RevokedOneof: &v1.GetAPITokensRequest_Revoked{
					Revoked: true,
				},
			},
			searchQuery:    search.NewQueryBuilder().AddBools(search.Revoked, true).ProtoQuery(),
			expectedResult: []*storage.TokenMetadata{s.testToken1},
		},
		"Read-only retrieves non-revoked tokens (non-revoked request)": {
			ctx: s.readOnlyCtx,
			getQuery: &v1.GetAPITokensRequest{
				RevokedOneof: &v1.GetAPITokensRequest_Revoked{
					Revoked: false,
				},
			},
			searchQuery:    search.NewQueryBuilder().AddBools(search.Revoked, false).ProtoQuery(),
			expectedResult: []*storage.TokenMetadata{s.testToken2},
		},
		"Read-write retrieves all tokens (nil request)": {
			ctx:            s.readWriteCtx,
			getQuery:       nil,
			searchQuery:    nil,
			expectedResult: []*storage.TokenMetadata{s.testToken1, s.testToken2},
		},
		"Read-write retrieves all tokens (full request)": {
			ctx:            s.readWriteCtx,
			getQuery:       &v1.GetAPITokensRequest{},
			searchQuery:    search.EmptyQuery(),
			expectedResult: []*storage.TokenMetadata{s.testToken1, s.testToken2},
		},
		"Read-write retrieves revoked tokens (revoked request)": {
			ctx: s.readWriteCtx,
			getQuery: &v1.GetAPITokensRequest{
				RevokedOneof: &v1.GetAPITokensRequest_Revoked{
					Revoked: true,
				},
			},
			searchQuery:    search.NewQueryBuilder().AddBools(search.Revoked, true).ProtoQuery(),
			expectedResult: []*storage.TokenMetadata{s.testToken1},
		},
		"Read-write retrieves non-revoked tokens (non-revoked request)": {
			ctx: s.readWriteCtx,
			getQuery: &v1.GetAPITokensRequest{
				RevokedOneof: &v1.GetAPITokensRequest_Revoked{
					Revoked: false,
				},
			},
			searchQuery:    search.NewQueryBuilder().AddBools(search.Revoked, false).ProtoQuery(),
			expectedResult: []*storage.TokenMetadata{s.testToken2},
		},
		"Wrong permission does not retrieve any token (nil request)": {
			ctx:            s.otherPermissionCtx,
			getQuery:       nil,
			searchQuery:    nil,
			expectedResult: nil,
			expectedSACErr: sac.ErrResourceAccessDenied,
		},
		"Wrong permission does not retrieve any token (full request)": {
			ctx:            s.otherPermissionCtx,
			getQuery:       &v1.GetAPITokensRequest{},
			searchQuery:    search.EmptyQuery(),
			expectedResult: nil,
			expectedSACErr: sac.ErrResourceAccessDenied,
		},
		"Wrong permission does not retrieve any token (revoked request)": {
			ctx: s.otherPermissionCtx,
			getQuery: &v1.GetAPITokensRequest{
				RevokedOneof: &v1.GetAPITokensRequest_Revoked{
					Revoked: true,
				},
			},
			searchQuery:    search.NewQueryBuilder().AddBools(search.Revoked, true).ProtoQuery(),
			expectedResult: nil,
			expectedSACErr: sac.ErrResourceAccessDenied,
		},
		"Wrong permission does not retrieve any token (non-revoked request)": {
			ctx: s.otherPermissionCtx,
			getQuery: &v1.GetAPITokensRequest{
				RevokedOneof: &v1.GetAPITokensRequest_Revoked{
					Revoked: false,
				},
			},
			searchQuery:    search.NewQueryBuilder().AddBools(search.Revoked, false).ProtoQuery(),
			expectedResult: nil,
			expectedSACErr: sac.ErrResourceAccessDenied,
		},
	}
}

func (s *apiTokenDatastoreSACTestSuite) TestGetTokenOrNil() {
	for name, tc := range map[string]struct {
		ctx           context.Context
		tokenID       string
		expectedToken *storage.TokenMetadata
	}{
		"No access cannot fetch revoked token": {
			ctx:           s.noPermissionCtx,
			tokenID:       s.testToken1.GetId(),
			expectedToken: nil,
		},
		"No access cannot fetch active token": {
			ctx:           s.noPermissionCtx,
			tokenID:       s.testToken2.GetId(),
			expectedToken: nil,
		},
		"Read-only can fetch revoked token": {
			ctx:           s.readOnlyCtx,
			tokenID:       s.testToken1.GetId(),
			expectedToken: s.testToken1,
		},
		"Read-only can fetch active token": {
			ctx:           s.readOnlyCtx,
			tokenID:       s.testToken2.GetId(),
			expectedToken: s.testToken2,
		},
		"Read-write can fetch revoked token": {
			ctx:           s.readWriteCtx,
			tokenID:       s.testToken1.GetId(),
			expectedToken: s.testToken1,
		},
		"Read-write can fetch active token": {
			ctx:           s.readWriteCtx,
			tokenID:       s.testToken2.GetId(),
			expectedToken: s.testToken2,
		},
		"wrong permission cannot fetch revoked token": {
			ctx:           s.otherPermissionCtx,
			tokenID:       s.testToken1.GetId(),
			expectedToken: nil,
		},
		"Wrong permission fetch active token": {
			ctx:           s.otherPermissionCtx,
			tokenID:       s.testToken2.GetId(),
			expectedToken: nil,
		},
	} {
		s.Run(name, func() {
			token, err := s.store.GetTokenOrNil(tc.ctx, tc.tokenID)
			s.NoError(err)
			if tc.expectedToken != nil {
				protoassert.Equal(s.T(), tc.expectedToken.CloneVT(), token)
			} else {
				s.Nil(token)
			}
		})
	}
}

func (s *apiTokenDatastoreSACTestSuite) TestGetTokens() {
	for name, tc := range s.getSearchTestCases() {
		s.Run(name, func() {
			tokens, err := s.store.GetTokens(tc.ctx, tc.getQuery)
			s.NoError(err)
			protoassert.ElementsMatch(s.T(), tc.expectedResult, tokens)
		})
	}
}

func (s *apiTokenDatastoreSACTestSuite) TestCount() {
	for name, tc := range s.getSearchTestCases() {
		s.Run(name, func() {
			expectedCount := len(tc.expectedResult)
			count, err := s.store.Count(tc.ctx, tc.searchQuery)
			if tc.expectedSACErr != nil {
				s.ErrorIs(err, tc.expectedSACErr)
			} else {
				s.NoError(err)
			}
			s.Equal(expectedCount, count)
		})
	}
}

func (s *apiTokenDatastoreSACTestSuite) TestSearchRawTokens() {
	for name, tc := range s.getSearchTestCases() {
		s.Run(name, func() {
			tokens, err := s.store.SearchRawTokens(tc.ctx, tc.searchQuery)
			if tc.expectedSACErr != nil {
				s.ErrorIs(err, tc.expectedSACErr)
				s.Nil(tokens)
			} else {
				s.NoError(err)
				protoassert.ElementsMatch(s.T(), tc.expectedResult, tokens)
			}
		})
	}
}

func (s *apiTokenDatastoreSACTestSuite) TestAddToken() {
	for name, tc := range map[string]struct {
		ctx         context.Context
		token       *storage.TokenMetadata
		expectedErr error
	}{
		"No access cannot add token": {
			ctx: s.noPermissionCtx,
			token: &storage.TokenMetadata{
				Id:   uuid.NewTestUUID(11).String(),
				Name: "No Access Add Token",
			},
			expectedErr: sac.ErrResourceAccessDenied,
		},
		"Read-only cannot add token": {
			ctx: s.readOnlyCtx,
			token: &storage.TokenMetadata{
				Id:   uuid.NewTestUUID(12).String(),
				Name: "Read-only Add Token",
			},
			expectedErr: sac.ErrResourceAccessDenied,
		},
		"Read-write can add token": {
			ctx: s.readWriteCtx,
			token: &storage.TokenMetadata{
				Id:   uuid.NewTestUUID(13).String(),
				Name: "Read-write Add Token",
			},
		},
		"Wrong permission cannot add token": {
			ctx: s.otherPermissionCtx,
			token: &storage.TokenMetadata{
				Id:   uuid.NewTestUUID(14).String(),
				Name: "Wrong permission Add Token",
			},
			expectedErr: sac.ErrResourceAccessDenied,
		},
	} {
		s.Run(name, func() {
			addErr := s.store.AddToken(tc.ctx, tc.token)
			token, found, checkErr := s.pgStore.Get(s.readOnlyCtx, tc.token.GetId())
			if tc.expectedErr == nil {
				s.NoError(addErr)
				// validate write succeeded by checking post-write fetch results
				s.NoError(checkErr)
				s.True(found)
				protoassert.Equal(s.T(), tc.token.CloneVT(), token)
			} else {
				s.ErrorIs(addErr, tc.expectedErr)
				// validate write was rejected and no data was stored
				s.NoError(checkErr)
				s.False(found)
				s.Nil(token)
			}
		})
	}
}

func (s *apiTokenDatastoreSACTestSuite) TestRevokeToken() {
	testToken := &storage.TokenMetadata{
		Id:   uuid.NewTestUUID(21).String(),
		Name: "Test revoke token",
	}
	noTokenToRevokeID := uuid.NewTestUUID(22).String()
	for name, tc := range map[string]struct {
		ctx         context.Context
		expectedErr error
	}{
		"No access cannot revoke token": {
			ctx:         s.noPermissionCtx,
			expectedErr: sac.ErrResourceAccessDenied,
		},
		"Read-only cannot revoke token": {
			ctx:         s.readOnlyCtx,
			expectedErr: sac.ErrResourceAccessDenied,
		},
		"Read-write can revoke token": {
			ctx: s.readWriteCtx,
		},
		"Wrong permission cannot revoke token": {
			ctx:         s.otherPermissionCtx,
			expectedErr: sac.ErrResourceAccessDenied,
		},
	} {
		s.Run(name, func() {
			s.Run("No Token to revoke", func() {
				exists, err := s.store.RevokeToken(tc.ctx, noTokenToRevokeID)
				token, found, checkErr := s.pgStore.Get(s.readOnlyCtx, noTokenToRevokeID)
				if tc.expectedErr == nil {
					s.NoError(err)
				} else {
					s.ErrorIs(err, tc.expectedErr)
				}
				s.False(exists)
				// Ensure the revoke call did not create a revoked token
				s.NoError(checkErr)
				s.False(found)
				s.Nil(token)
			})
			s.Run("Existing Token to revoke", func() {
				s.Require().NoError(s.pgStore.Upsert(s.readWriteCtx, testToken))
				exists, err := s.store.RevokeToken(tc.ctx, testToken.GetId())
				token, found, checkErr := s.pgStore.Get(s.readOnlyCtx, testToken.GetId())
				if tc.expectedErr == nil {
					s.True(exists)
					s.NoError(err)
					// ensure the revoke call did set the token as revoked
					s.NoError(checkErr)
					s.True(found)
					s.True(token.GetRevoked())
				} else {
					s.False(exists)
					s.ErrorIs(err, tc.expectedErr)
					// ensure revoke call was rejected and did not update the token
					s.NoError(checkErr)
					s.True(found)
					s.False(token.GetRevoked())
				}
				s.Require().NoError(s.pgStore.Delete(s.readWriteCtx, testToken.GetId()))
			})
		})
	}
}

func TestAPITokenDataStoreScheduleSAC(t *testing.T) {
	suite.Run(t, new(apiTokenScheduleDatastoreSACTestSuite))
}

type apiTokenScheduleDatastoreSACTestSuite struct {
	suite.Suite

	noPermissionCtx    context.Context
	readOnlyCtx        context.Context
	readWriteCtx       context.Context
	otherPermissionCtx context.Context

	pool postgres.DB

	pgStore pgScheduleStore.Store
	store   DataStore
}

func (s *apiTokenScheduleDatastoreSACTestSuite) SetupSuite() {
	s.noPermissionCtx = sac.WithNoAccess(s.T().Context())
	s.readOnlyCtx = sac.WithGlobalAccessScopeChecker(
		s.T().Context(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
			sac.ResourceScopeKeys(resources.Notifications),
		),
	)
	s.readWriteCtx = sac.WithGlobalAccessScopeChecker(
		s.T().Context(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.Notifications),
		),
	)
	s.otherPermissionCtx = sac.WithGlobalAccessScopeChecker(
		s.T().Context(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.Integration),
		),
	)
}

func (s *apiTokenScheduleDatastoreSACTestSuite) SetupTest() {
	s.pool = pgtest.ForT(s.T())
	s.pgStore = pgScheduleStore.New(s.pool)
	s.store = NewPostgres(s.pool)
}

func (s *apiTokenScheduleDatastoreSACTestSuite) TearDownTest() {
	s.pool.Close()
}

func (s *apiTokenScheduleDatastoreSACTestSuite) TestGetNotificationSchedule() {
	testSchedule := &storage.NotificationSchedule{
		LastRun: protocompat.ConvertTimeToTimestampOrNil(&timestamp1),
	}
	allAccessCtx := sac.WithAllAccess(s.T().Context())
	s.Require().NoError(s.pgStore.Upsert(allAccessCtx, testSchedule))

	tests := map[string]struct {
		ctx              context.Context
		expectedSchedule *storage.NotificationSchedule
	}{
		"No access": {
			ctx: s.noPermissionCtx,
		},
		"Read only": {
			ctx:              s.readOnlyCtx,
			expectedSchedule: testSchedule,
		},
		"Read write": {
			ctx:              s.readWriteCtx,
			expectedSchedule: testSchedule,
		},
		"Wrong permission": {
			ctx: s.otherPermissionCtx,
		},
	}

	for name, tc := range tests {
		s.Run(name, func() {
			schedule, found, err := s.store.GetNotificationSchedule(tc.ctx)
			s.NoError(err)
			if tc.expectedSchedule != nil {
				s.True(found)
				protoassert.Equal(s.T(), tc.expectedSchedule.CloneVT(), schedule)
			} else {
				s.False(found)
				s.Nil(schedule)
			}
		})
	}
}

func (s *apiTokenScheduleDatastoreSACTestSuite) TestGetNotificationScheduleNoSchedule() {
	tests := map[string]struct {
		ctx         context.Context
		expectedErr error
	}{
		"No access": {
			ctx:         s.noPermissionCtx,
			expectedErr: sac.ErrResourceAccessDenied,
		},
		"Read only": {
			ctx:         s.readOnlyCtx,
			expectedErr: nil,
		},
		"Read write": {
			ctx:         s.readWriteCtx,
			expectedErr: nil,
		},
		"Wrong permission": {
			ctx:         s.otherPermissionCtx,
			expectedErr: sac.ErrResourceAccessDenied,
		},
	}

	for name, tc := range tests {
		s.Run(name, func() {
			ctx := tc.ctx
			schedule, found, err := s.store.GetNotificationSchedule(ctx)
			s.NoError(err)
			s.False(found)
			s.Nil(schedule)
		})
	}
}

func (s *apiTokenScheduleDatastoreSACTestSuite) TestUpsertNotificationSchedule() {
	for name, tc := range map[string]struct {
		ctx         context.Context
		schedule    *storage.NotificationSchedule
		expectedErr error
	}{
		"No access": {
			ctx: s.noPermissionCtx,
			schedule: &storage.NotificationSchedule{
				LastRun: protocompat.ConvertTimeToTimestampOrNil(&timestamp1),
			},
			expectedErr: sac.ErrResourceAccessDenied,
		},
		"Read only": {
			ctx: s.readOnlyCtx,
			schedule: &storage.NotificationSchedule{
				LastRun: protocompat.ConvertTimeToTimestampOrNil(&timestamp2),
			},
			expectedErr: sac.ErrResourceAccessDenied,
		},
		"Read write": {
			ctx: s.readWriteCtx,
			schedule: &storage.NotificationSchedule{
				LastRun: protocompat.ConvertTimeToTimestampOrNil(&timestamp3),
			},
			expectedErr: nil,
		},
		"Wrong permission": {
			ctx: s.otherPermissionCtx,
			schedule: &storage.NotificationSchedule{
				LastRun: protocompat.ConvertTimeToTimestampOrNil(&timestamp4),
			},
			expectedErr: sac.ErrResourceAccessDenied,
		},
	} {
		s.Run(name, func() {
			err := s.store.UpsertNotificationSchedule(tc.ctx, tc.schedule)
			if tc.expectedErr != nil {
				s.ErrorIs(err, tc.expectedErr)

				// Verify that the schedule was not inserted
				schedule, found, fetchErr := s.pgStore.Get(s.readOnlyCtx)
				s.NoError(fetchErr)
				s.False(found)
				s.Nil(schedule)
			} else {
				s.NoError(err)

				// Verify that the schedule was inserted
				schedule, found, fetchErr := s.pgStore.Get(s.readOnlyCtx)
				s.NoError(fetchErr)
				s.True(found)
				protoassert.Equal(s.T(), tc.schedule.CloneVT(), schedule)
			}
			s.Require().NoError(s.pgStore.Delete(s.readWriteCtx))
		})
	}
}
