//go:build sql_integration

package m192tom193

import (
	"context"
	"testing"

	protoTypes "github.com/gogo/protobuf/types"
	"github.com/stackrox/rox/generated/storage"
	oldApiTokenStore "github.com/stackrox/rox/migrator/migrations/m_192_to_m_193_make_api_token_names_unique/apitokenstore/old"
	oldPkgSchema "github.com/stackrox/rox/migrator/migrations/m_192_to_m_193_make_api_token_names_unique/schema/old"
	pghelper "github.com/stackrox/rox/migrator/migrations/postgreshelper"
	"github.com/stackrox/rox/migrator/types"
	"github.com/stackrox/rox/pkg/postgres/pgutils"
	"github.com/stackrox/rox/pkg/sac"
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
	pgutils.CreateTableFromModel(s.ctx, s.db.GetGormDB(), oldPkgSchema.CreateTableAPITokensStmt)
}

func (s *migrationTestSuite) TearDownSuite() {
	s.db.Teardown(s.T())
}

var (
	// No Collision
	// X -> X
	noCollisionTokenName = "No Collision"
	noCollisionToken     = &storage.TokenMetadata{
		Id:         "11111111-2222-3333-4444-111111111111",
		Name:       noCollisionTokenName,
		Roles:      []string{"Diamonds are forever"},
		IssuedAt:   &protoTypes.Timestamp{Seconds: int64(61516800)},
		Expiration: &protoTypes.Timestamp{Seconds: int64(253402214400)},
		Revoked:    false,
	}

	// Simple Collision
	// X -> X
	// X -> X (2)
	// X -> X (3)
	simpleCollisionTokenName  = "Simple Collision"
	simpleCollisionTokenName2 = "Simple Collision (2)"
	simpleCollisionTokenName3 = "Simple Collision (3)"
	simpleCollisionToken1     = &storage.TokenMetadata{
		Id:         "11111111-2222-3333-4444-222222222222",
		Name:       simpleCollisionTokenName,
		Roles:      []string{"Live and let die"},
		IssuedAt:   &protoTypes.Timestamp{Seconds: int64(109987200)},
		Expiration: &protoTypes.Timestamp{Seconds: int64(156643200)},
		Revoked:    false,
	}
	simpleCollisionToken2 = &storage.TokenMetadata{
		Id:         "11111111-2222-3333-4444-333333333333",
		Name:       simpleCollisionTokenName,
		Roles:      []string{"The man with the golden gun"},
		IssuedAt:   &protoTypes.Timestamp{Seconds: int64(156643200)},
		Expiration: &protoTypes.Timestamp{Seconds: int64(237124800)},
		Revoked:    false,
	}
	simpleCollisionToken3 = &storage.TokenMetadata{
		Id:         "11111111-2222-3333-4444-444444444444",
		Name:       simpleCollisionTokenName,
		Roles:      []string{"The spy who loved me"},
		IssuedAt:   &protoTypes.Timestamp{Seconds: int64(237124800)},
		Expiration: &protoTypes.Timestamp{Seconds: int64(299203200)},
		Revoked:    false,
	}

	// Collision on renamed (depending on the order in which item are fetched from database)
	// X     -> X     | X
	// X     -> X (3) | X (2)
	// X (2) -> X (2) | X (2) (2)
	collisionAfterRenameTokenName   = "Collision After Rename"
	collisionAfterRenameTokenName2  = "Collision After Rename (2)"
	collisionAfterRenameTokenName3  = "Collision After Rename (3)"
	collisionAfterRenameTokenName22 = "Collision After Rename (2) (2)"
	collisionAfterRenameToken1      = &storage.TokenMetadata{
		Id:         "11111111-2222-3333-4444-555555555555",
		Name:       collisionAfterRenameTokenName,
		Roles:      []string{"Moonraker"},
		IssuedAt:   &protoTypes.Timestamp{Seconds: int64(299203200)},
		Expiration: &protoTypes.Timestamp{Seconds: int64(362188800)},
		Revoked:    false,
	}
	collisionAfterRenameToken2 = &storage.TokenMetadata{
		Id:         "11111111-2222-3333-4444-666666666666",
		Name:       collisionAfterRenameTokenName,
		Roles:      []string{"For your eyes only"},
		IssuedAt:   &protoTypes.Timestamp{Seconds: int64(362188800)},
		Expiration: &protoTypes.Timestamp{Seconds: int64(423705600)},
		Revoked:    false,
	}
	collisionAfterRenameToken3 = &storage.TokenMetadata{
		Id:         "11111111-2222-3333-4444-777777777777",
		Name:       collisionAfterRenameTokenName2,
		Roles:      []string{"Octopussy"},
		IssuedAt:   &protoTypes.Timestamp{Seconds: int64(423705600)},
		Expiration: &protoTypes.Timestamp{Seconds: int64(485568000)},
		Revoked:    false,
	}

	preMigrationTokens = []*storage.TokenMetadata{
		noCollisionToken,
		simpleCollisionToken1,
		simpleCollisionToken2,
		simpleCollisionToken3,
		collisionAfterRenameToken1,
		collisionAfterRenameToken2,
		collisionAfterRenameToken3,
	}

	// There are multiple possible renaming scenarios. All are valid. They
	// depend on the order in which items were retrieved from the database
	// during the migration itself.
	// These two lists are the only two possible name sets that should
	// result from the migration.
	postMigrationTokenList1 = []string{
		noCollisionTokenName,
		simpleCollisionTokenName,
		simpleCollisionTokenName2,
		simpleCollisionTokenName3,
		collisionAfterRenameTokenName,
		collisionAfterRenameTokenName2,
		collisionAfterRenameTokenName3,
	}
	postMigrationTokenList2 = []string{
		noCollisionTokenName,
		simpleCollisionTokenName,
		simpleCollisionTokenName2,
		simpleCollisionTokenName3,
		collisionAfterRenameTokenName,
		collisionAfterRenameTokenName2,
		collisionAfterRenameTokenName22,
	}

	// There are multiple possible renaming scenarios. All are valid. They
	// depend on the order in which items were retrieved from the database
	// during the migration itself.
	// These two mapping are the only two reverse name mapping that should
	// result from the migration.
	revertNameMapping1 = map[string]string{
		noCollisionTokenName:           noCollisionTokenName,
		simpleCollisionTokenName:       simpleCollisionTokenName,
		simpleCollisionTokenName2:      simpleCollisionTokenName,
		simpleCollisionTokenName3:      simpleCollisionTokenName,
		collisionAfterRenameTokenName:  collisionAfterRenameTokenName,
		collisionAfterRenameTokenName2: collisionAfterRenameTokenName2,
		collisionAfterRenameTokenName3: collisionAfterRenameTokenName,
	}
	revertNameMapping2 = map[string]string{
		noCollisionTokenName:            noCollisionTokenName,
		simpleCollisionTokenName:        simpleCollisionTokenName,
		simpleCollisionTokenName2:       simpleCollisionTokenName,
		simpleCollisionTokenName3:       simpleCollisionTokenName,
		collisionAfterRenameTokenName:   collisionAfterRenameTokenName,
		collisionAfterRenameTokenName2:  collisionAfterRenameTokenName,
		collisionAfterRenameTokenName22: collisionAfterRenameTokenName2,
	}
)

func (s *migrationTestSuite) TestMigration() {
	store := oldApiTokenStore.New(s.db)

	s.Require().NoError(store.UpsertMany(s.ctx, preMigrationTokens))

	dbs := &types.Databases{
		GormDB:     s.db.GetGormDB(),
		PostgresDB: s.db.DB,
		DBCtx:      s.ctx,
	}

	s.Require().NoError(migration.Run(dbs))

	postMigrationTokenMap := make(map[string]*storage.TokenMetadata)
	s.Require().NoError(store.Walk(s.ctx, func(obj *storage.TokenMetadata) error {
		postMigrationTokenMap[obj.GetName()] = obj
		return nil
	}))

	// There are multiple possible renaming scenarios. All are valid. They depend
	// on the order in which items were retrieved from the database during the
	// migration itself.

	// In order to validate no object was lost, the objects pulled from the
	// database post-migration are renamed back to what they should have been
	// named before the migration.
	// In the renaming loop, the list of names found after the migration is
	// captured and compared to the list that the renaming scenario should
	// have had as result.
	renamedPostMigrationMappedBackTokens := make([]*storage.TokenMetadata, 0, len(postMigrationTokenMap))
	postMigrationTokenNames := make([]string, 0, len(postMigrationTokenMap))
	if _, doubleCollision := postMigrationTokenMap[collisionAfterRenameTokenName22]; doubleCollision {
		for name, token := range postMigrationTokenMap {
			mappedBackToken := token.Clone()
			mappedBackToken.Name = revertNameMapping2[name]
			postMigrationTokenNames = append(postMigrationTokenNames, name)
			renamedPostMigrationMappedBackTokens = append(renamedPostMigrationMappedBackTokens, mappedBackToken)
		}
		s.ElementsMatch(postMigrationTokenList2, postMigrationTokenNames)
	} else {
		for name, token := range postMigrationTokenMap {
			mappedBackToken := token.Clone()
			mappedBackToken.Name = revertNameMapping1[name]
			postMigrationTokenNames = append(postMigrationTokenNames, name)
			renamedPostMigrationMappedBackTokens = append(renamedPostMigrationMappedBackTokens, mappedBackToken)
		}
		s.ElementsMatch(postMigrationTokenList1, postMigrationTokenNames)
	}

	s.ElementsMatch(preMigrationTokens, renamedPostMigrationMappedBackTokens)
}
