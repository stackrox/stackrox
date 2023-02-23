//go:build sql_integration

package datastore

import (
	"context"
	"testing"

	boltStore "github.com/stackrox/rox/central/config/store/bolt"
	pgStore "github.com/stackrox/rox/central/config/store/postgres"
	"github.com/stackrox/rox/central/role/resources"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/bolthelper"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stretchr/testify/suite"
	bolt "go.etcd.io/bbolt"
)

func TestConfigDatastore(t *testing.T) {
	suite.Run(t, new(configDataStorePostgresTestSuite))
}

type configDataStorePostgresTestSuite struct {
	suite.Suite

	engine *bolt.DB

	postgrestest *pgtest.TestPostgres

	dataStore DataStore

	hasNoneCtx  context.Context
	hasReadCtx  context.Context
	hasWriteCtx context.Context

	// TODO: ROX-12750 Remove this variable
	hasReadAdministrationCtx  context.Context
	hasWriteAdministrationCtx context.Context
}

func (s *configDataStorePostgresTestSuite) SetupTest() {
	var err error
	configSuiteObj := "configTest"

	if env.PostgresDatastoreEnabled.BooleanSetting() {
		s.postgrestest = pgtest.ForT(s.T())
		s.Require().NotNil(s.postgrestest)
		store := pgStore.New(s.postgrestest.DB)
		s.dataStore = New(store)
	} else {
		s.engine, err = bolthelper.NewTemp(configSuiteObj)
		s.Require().NoError(err)
		store := boltStore.New(s.engine)
		s.dataStore = New(store)
	}

	s.hasNoneCtx = sac.WithGlobalAccessScopeChecker(context.Background(), sac.DenyAllAccessScopeChecker())
	s.hasReadCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
			// TODO: ROX-12750 Replace Config with Administration
			sac.ResourceScopeKeys(resources.Config)))
	s.hasWriteCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			// TODO: ROX-12750 Replace Config with Administration
			sac.ResourceScopeKeys(resources.Config)))
	s.hasReadAdministrationCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.Administration)))
	s.hasWriteAdministrationCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.Administration)))
}

func (s *configDataStorePostgresTestSuite) TearDownTest() {
	if env.PostgresDatastoreEnabled.BooleanSetting() {
		s.postgrestest.Teardown(s.T())
	} else {
		s.Require().NoError(s.engine.Close())
	}
}

func getTestConfig(text1 string, text2 string, text3 string) *storage.Config {
	return &storage.Config{
		PublicConfig: &storage.PublicConfig{
			LoginNotice: &storage.LoginNotice{
				Enabled: false,
				Text:    text1,
			},
			Header: &storage.BannerConfig{
				Enabled:         false,
				Text:            text2,
				Size_:           0,
				Color:           "black",
				BackgroundColor: "green",
			},
			Footer: &storage.BannerConfig{
				Enabled:         false,
				Text:            text3,
				Size_:           0,
				Color:           "black",
				BackgroundColor: "yellow",
			},
		},
		PrivateConfig: &storage.PrivateConfig{
			AlertRetention:                      nil,
			ImageRetentionDurationDays:          10,
			ExpiredVulnReqRetentionDurationDays: 20,
			DecommissionedClusterRetention: &storage.DecommissionedClusterRetentionConfig{
				RetentionDurationDays: 1,
				IgnoreClusterLabels:   nil,
				LastUpdated:           nil,
				CreatedAt:             nil,
			},
		},
	}
}

func (s *configDataStorePostgresTestSuite) TestGet() {
	// Test retrieve from empty store

	configEmptyNoAccess, err := s.dataStore.GetConfig(s.hasNoneCtx)
	s.NoError(err)
	s.Nil(configEmptyNoAccess)

	configEmptyReadAccess, err := s.dataStore.GetConfig(s.hasReadCtx)
	s.NoError(err)
	s.Nil(configEmptyReadAccess)

	configEmptyWriteAccess, err := s.dataStore.GetConfig(s.hasWriteCtx)
	s.NoError(err)
	s.Nil(configEmptyWriteAccess)

	configEmptyReadAdmAccess, err := s.dataStore.GetConfig(s.hasReadAdministrationCtx)
	s.NoError(err)
	s.Nil(configEmptyReadAdmAccess)

	configEmptyWriteAdmAccess, err := s.dataStore.GetConfig(s.hasWriteAdministrationCtx)
	s.NoError(err)
	s.Nil(configEmptyWriteAdmAccess)

	// Test with dummy config
	testConfig := getTestConfig(
		"Lorem ipsum dolor sit amet",
		"consectetur adipiscing elit",
		"sed do eiusmod tempor incididunt ut labore et dolore magna aliqua",
	)
	s.Require().NoError(s.dataStore.UpsertConfig(sac.WithAllAccess(context.Background()), testConfig))

	configStoredNoAccess, err := s.dataStore.GetConfig(s.hasNoneCtx)
	s.NoError(err)
	s.Nil(configStoredNoAccess)

	configStoredReadAccess, err := s.dataStore.GetConfig(s.hasReadCtx)
	s.NoError(err)
	s.Equal(testConfig, configStoredReadAccess)

	configStoredWriteAccess, err := s.dataStore.GetConfig(s.hasWriteCtx)
	s.NoError(err)
	s.Equal(testConfig, configStoredWriteAccess)

	configStoredReadAdmAccess, err := s.dataStore.GetConfig(s.hasReadAdministrationCtx)
	s.NoError(err)
	s.Equal(testConfig, configStoredReadAdmAccess)

	configStoredWriteAdmAccess, err := s.dataStore.GetConfig(s.hasWriteAdministrationCtx)
	s.NoError(err)
	s.Equal(testConfig, configStoredWriteAdmAccess)
}

func (s *configDataStorePostgresTestSuite) TestUpsert() {
	allAccessCtx := sac.WithAllAccess(context.Background())
	// Test with dummy config
	testConfig := getTestConfig(
		"Lorem ipsum dolor sit amet",
		"consectetur adipiscing elit",
		"sed do eiusmod tempor incididunt ut labore et dolore magna aliqua",
	)
	s.Require().NoError(s.dataStore.UpsertConfig(allAccessCtx, testConfig))

	var err error
	testConfigNoAccess := getTestConfig(
		"Sed ut perspiciatis",
		"unde omnis iste natus error sit voluptatem accusantium doloremque laudantium",
		"totam rem aperiam eaque ipsa",
	)
	err = s.dataStore.UpsertConfig(s.hasNoneCtx, testConfigNoAccess)
	s.Error(err)
	configPostNoAccessUpsert, err := s.dataStore.GetConfig(allAccessCtx)
	s.NoError(err)
	s.Equal(testConfig, configPostNoAccessUpsert)

	testConfigReadAccess := getTestConfig(
		"quae ab illo inventore veritatis et quasi architecto beatae vitae dicta sunt",
		"explicabo",
		"Nemo enim ipsam voluptatem",
	)
	err = s.dataStore.UpsertConfig(s.hasReadCtx, testConfigReadAccess)
	s.Error(err)
	configPostReadAccessUpsert, err := s.dataStore.GetConfig(allAccessCtx)
	s.NoError(err)
	s.Equal(testConfig, configPostReadAccessUpsert)

	testConfigWriteAccess := getTestConfig(
		"quia voluptas sit",
		"aspernatur aut odit aut fugit",
		"sed quia consequuntur magni dolores eos",
	)
	err = s.dataStore.UpsertConfig(s.hasWriteCtx, testConfigWriteAccess)
	s.NoError(err)
	configPostWriteAccessUpsert, err := s.dataStore.GetConfig(allAccessCtx)
	s.NoError(err)
	s.Equal(testConfigWriteAccess, configPostWriteAccessUpsert)

	testConfigReadAdmAccess := getTestConfig(
		"qui ratione voluptatem sequi nesciunt",
		"neque porro quisquam est, qui dolorem ipsum",
		"quia dolor sit amet consectetur adipisci[ng] velit",
	)
	err = s.dataStore.UpsertConfig(s.hasReadAdministrationCtx, testConfigReadAdmAccess)
	s.Error(err)
	configPostReadAdmAccessUpsert, err := s.dataStore.GetConfig(allAccessCtx)
	s.NoError(err)
	s.Equal(testConfigWriteAccess, configPostReadAdmAccessUpsert)

	testConfigWriteAdmAccess := getTestConfig(
		"sed quia non numquam [do] eius modi tempora inci[di]dunt",
		"ut labore et dolore magnam aliquam quaerat voluptatem",
		"Ut enim ad minima veniam, quis nostrum[d] exercitationem ullam corporis suscipit laboriosam",
	)
	err = s.dataStore.UpsertConfig(s.hasWriteAdministrationCtx, testConfigWriteAdmAccess)
	s.NoError(err)
	configPostWriteAdmAccessUpsert, err := s.dataStore.GetConfig(allAccessCtx)
	s.NoError(err)
	s.Equal(testConfigWriteAdmAccess, configPostWriteAdmAccessUpsert)
}
