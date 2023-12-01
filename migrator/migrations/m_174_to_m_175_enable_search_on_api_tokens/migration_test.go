//go:build sql_integration

package m174tom175

import (
	"context"
	"fmt"
	"testing"
	"time"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	oldSchema "github.com/stackrox/rox/migrator/migrations/frozenschema/v73"
	newAPITokenStore "github.com/stackrox/rox/migrator/migrations/m_174_to_m_175_enable_search_on_api_tokens/newapitokenpostgresstore"
	oldAPITokenStore "github.com/stackrox/rox/migrator/migrations/m_174_to_m_175_enable_search_on_api_tokens/oldapitokenpostgresstore"
	pghelper "github.com/stackrox/rox/migrator/migrations/postgreshelper"
	"github.com/stackrox/rox/pkg/postgres/pgutils"
	"github.com/stackrox/rox/pkg/protoconv"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/suite"
)

const (
	timestampLayout = "01/02/2006 3:04:05 PM MST"
)

var (
	now                = time.Now().UTC()
	inAMinute          = now.Add(1 * time.Minute)
	inAnHour           = now.Add(1 * time.Hour)
	inAnHourAndAMinute = now.Add(1*time.Hour + 1*time.Minute)
	aWeekAgo           = now.Add(-7 * 24 * time.Hour)
	yesterday          = now.Add(-24 * time.Hour)

	token1 = &storage.TokenMetadata{
		Id:         uuid.NewV4().String(),
		Name:       "Test Token 1",
		Roles:      []string{"Admin"},
		IssuedAt:   protoconv.ConvertTimeToTimestamp(now),
		Expiration: protoconv.ConvertTimeToTimestamp(inAnHour),
		Revoked:    false,
		Role:       "",
	}
	token2 = &storage.TokenMetadata{
		Id:         uuid.NewV4().String(),
		Name:       "Test Token 2",
		Roles:      []string{"Admin"},
		IssuedAt:   protoconv.ConvertTimeToTimestamp(inAMinute),
		Expiration: protoconv.ConvertTimeToTimestamp(inAnHourAndAMinute),
		Revoked:    false,
		Role:       "",
	}
	token3 = &storage.TokenMetadata{
		Id:         uuid.NewV4().String(),
		Name:       "Test Token 3",
		Roles:      []string{"Admin"},
		IssuedAt:   protoconv.ConvertTimeToTimestamp(aWeekAgo),
		Expiration: protoconv.ConvertTimeToTimestamp(yesterday),
		Revoked:    false,
		Role:       "",
	}
	token4 = &storage.TokenMetadata{
		Id:         uuid.NewV4().String(),
		Name:       "Test Token 4",
		Roles:      []string{"Analyst"},
		IssuedAt:   protoconv.ConvertTimeToTimestamp(now),
		Expiration: protoconv.ConvertTimeToTimestamp(inAnHour),
		Revoked:    true,
		Role:       "",
	}
	token5 = &storage.TokenMetadata{
		Id:         uuid.NewV4().String(),
		Name:       "Test Token 5",
		Roles:      []string{"Analyst"},
		IssuedAt:   protoconv.ConvertTimeToTimestamp(inAMinute),
		Expiration: protoconv.ConvertTimeToTimestamp(inAnHourAndAMinute),
		Revoked:    true,
		Role:       "",
	}
	token6 = &storage.TokenMetadata{
		Id:         uuid.NewV4().String(),
		Name:       "Test Token 6",
		Roles:      []string{"Analyst"},
		IssuedAt:   protoconv.ConvertTimeToTimestamp(aWeekAgo),
		Expiration: protoconv.ConvertTimeToTimestamp(yesterday),
		Revoked:    true,
		Role:       "",
	}
	tokensToMigrate = []*storage.TokenMetadata{
		token1,
		token2,
		token3,
		token4,
		token5,
		token6,
	}
)

type apiTokenMigrationTestSuite struct {
	suite.Suite

	db            *pghelper.TestPostgres
	oldTokenStore oldAPITokenStore.Store
	newTokenStore newAPITokenStore.Store
}

func TestMigration(t *testing.T) {
	suite.Run(t, new(apiTokenMigrationTestSuite))
}

func (s *apiTokenMigrationTestSuite) SetupTest() {
	s.db = pghelper.ForT(s.T(), false)
	s.oldTokenStore = oldAPITokenStore.New(s.db.DB)
	s.newTokenStore = newAPITokenStore.New(s.db.DB)
	pgutils.CreateTableFromModel(context.Background(), s.db.GetGormDB(), oldSchema.CreateTableAPITokensStmt)
}

func (s *apiTokenMigrationTestSuite) TearDownTest() {
	s.db.Teardown(s.T())
}

func getTimestampLookupQuery(label search.FieldLabel, date time.Time) *v1.Query {
	oneSecondLater := date.Add(1 * time.Second)
	fmt.Println(date.Format(timestampLayout))
	lowerBoundQuery := search.NewQueryBuilder().
		AddStrings(label, fmt.Sprintf(">=%s", date.Format(timestampLayout))).
		ProtoQuery()
	upperBoundQuery := search.NewQueryBuilder().
		AddStrings(label, fmt.Sprintf("<%s", oneSecondLater.Format(timestampLayout))).
		ProtoQuery()
	return search.ConjunctionQuery(lowerBoundQuery, upperBoundQuery)
}

func (s *apiTokenMigrationTestSuite) TestMigration() {
	ctx := sac.WithAllAccess(context.Background())

	expiresInAnHourQuery := getTimestampLookupQuery(search.Expiration, inAnHour)
	expiresInAnHourAndAMinuteQuery := getTimestampLookupQuery(search.Expiration, inAnHourAndAMinute)
	expiredYesterdayQuery := getTimestampLookupQuery(search.Expiration, yesterday)
	isRevokedQuery := search.NewQueryBuilder().
		AddBools(search.Revoked, true).
		ProtoQuery()
	isNotRevokedQuery := search.NewQueryBuilder().
		AddBools(search.Revoked, false).
		ProtoQuery()

	token1Query := search.ConjunctionQuery(expiresInAnHourQuery, isNotRevokedQuery)
	token2Query := search.ConjunctionQuery(expiresInAnHourAndAMinuteQuery, isNotRevokedQuery)
	token3Query := search.ConjunctionQuery(expiredYesterdayQuery, isNotRevokedQuery)
	token4Query := search.ConjunctionQuery(expiresInAnHourQuery, isRevokedQuery)
	token5Query := search.ConjunctionQuery(expiresInAnHourAndAMinuteQuery, isRevokedQuery)
	token6Query := search.ConjunctionQuery(expiredYesterdayQuery, isRevokedQuery)

	err := s.oldTokenStore.UpsertMany(ctx, tokensToMigrate)
	s.NoError(err)
	res1x1, err1x1 := s.newTokenStore.GetByQuery(ctx, token1Query)
	s.ErrorContains(err1x1, "column api_tokens.expiration does not exist")
	s.Equal(0, len(res1x1))
	res1x2, err1x2 := s.newTokenStore.GetByQuery(ctx, token2Query)
	s.ErrorContains(err1x2, "column api_tokens.expiration does not exist")
	s.Equal(0, len(res1x2))
	res1x3, err1x3 := s.newTokenStore.GetByQuery(ctx, token3Query)
	s.ErrorContains(err1x3, "column api_tokens.expiration does not exist")
	s.Equal(0, len(res1x3))
	res1x4, err1x4 := s.newTokenStore.GetByQuery(ctx, token4Query)
	s.ErrorContains(err1x4, "column api_tokens.expiration does not exist")
	s.Equal(0, len(res1x4))
	res1x5, err1x5 := s.newTokenStore.GetByQuery(ctx, token5Query)
	s.ErrorContains(err1x5, "column api_tokens.expiration does not exist")
	s.Equal(0, len(res1x5))
	res1x6, err1x6 := s.newTokenStore.GetByQuery(ctx, token6Query)
	s.ErrorContains(err1x6, "column api_tokens.expiration does not exist")
	s.Equal(0, len(res1x6))

	err = migrateAPITokens(s.db.DB, s.db.GetGormDB())
	s.NoError(err)

	res2x1, err2x1 := s.newTokenStore.GetByQuery(ctx, token1Query)
	s.NoError(err2x1)
	s.ElementsMatch([]*storage.TokenMetadata{token1}, res2x1)
	res2x2, err2x2 := s.newTokenStore.GetByQuery(ctx, token2Query)
	s.NoError(err2x2)
	s.ElementsMatch([]*storage.TokenMetadata{token2}, res2x2)
	res2x3, err2x3 := s.newTokenStore.GetByQuery(ctx, token3Query)
	s.NoError(err2x3)
	s.ElementsMatch([]*storage.TokenMetadata{token3}, res2x3)
	res2x4, err2x4 := s.newTokenStore.GetByQuery(ctx, token4Query)
	s.NoError(err2x4)
	s.ElementsMatch([]*storage.TokenMetadata{token4}, res2x4)
	res2x5, err2x5 := s.newTokenStore.GetByQuery(ctx, token5Query)
	s.NoError(err2x5)
	s.ElementsMatch([]*storage.TokenMetadata{token5}, res2x5)
	res2x6, err2x6 := s.newTokenStore.GetByQuery(ctx, token6Query)
	s.NoError(err2x6)
	s.ElementsMatch([]*storage.TokenMetadata{token6}, res2x6)
}
