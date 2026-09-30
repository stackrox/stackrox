//go:build sql_integration

package datastore

import (
	"context"
	"testing"

	aimetadataDS "github.com/stackrox/rox/central/aimetadata/datastore"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/postgres/pgutils"
	"github.com/stackrox/rox/pkg/protoassert"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

func TestCustomResourceDataStore(t *testing.T) {
	suite.Run(t, new(customResourceDataStoreSuite))
}

type customResourceDataStoreSuite struct {
	suite.Suite
	ctx      context.Context
	db       *pgtest.TestPostgres
	customDS DataStore
	aiDS     aimetadataDS.DataStore
}

func (s *customResourceDataStoreSuite) SetupTest() {
	s.ctx = sac.WithAllAccess(context.Background())
	s.db = pgtest.ForT(s.T())
	require.NotNil(s.T(), s.db)
	_, err := s.db.Exec(s.ctx, "TRUNCATE custom_resources, ai_metadata, deployments CASCADE")
	s.Require().NoError(err)
	s.customDS = GetTestPostgresDataStore(s.T(), s.db.DB)
	s.aiDS = aimetadataDS.GetTestPostgresDataStore(s.T(), s.db.DB)
}

func (s *customResourceDataStoreSuite) TestAIMetadataLifecycle() {
	customResource := s.customResource("inference-service", "")
	metadata := &storage.AIMetadata{
		Id:             customResource.GetId(),
		ModelName:      "granite",
		ModelFormat:    "vLLM",
		ServingRuntime: "granite-runtime",
		State:          storage.AIMetadata_READY,
	}

	s.Require().NoError(s.customDS.UpsertCustomResource(s.ctx, customResource))
	s.Require().NoError(s.aiDS.UpsertAIMetadata(s.ctx, metadata))

	found, exists, err := s.aiDS.GetAIMetadata(s.ctx, customResource.GetId())
	s.Require().NoError(err)
	s.True(exists)
	protoassert.Equal(s.T(), metadata, found)

	metadataResults, err := s.aiDS.ListAIMetadata(s.ctx,
		search.NewQueryBuilder().AddExactMatches(search.AIModelName, "granite").ProtoQuery())
	s.Require().NoError(err)
	protoassert.ElementsMatch(s.T(), []*storage.AIMetadata{metadata}, metadataResults)

	s.Require().NoError(s.customDS.DeleteCustomResource(s.ctx, customResource.GetId()))
	_, exists, err = s.aiDS.GetAIMetadata(s.ctx, customResource.GetId())
	s.Require().NoError(err)
	s.False(exists)
}

func (s *customResourceDataStoreSuite) TestAIMetadataRequiresCustomResource() {
	err := s.aiDS.UpsertAIMetadata(s.ctx, &storage.AIMetadata{
		Id:          uuid.NewV4().String(),
		ModelFormat: "vLLM",
	})
	s.Error(err)
}

func (s *customResourceDataStoreSuite) TestOwnedDeployments() {
	root := s.customResource("inference-service", "")
	child := s.customResource("serving-resource", root.GetId())
	child.ClusterId = root.GetClusterId()
	deployment := &storage.Deployment{
		Id:                    uuid.NewV4().String(),
		Name:                  "predictor",
		Namespace:             root.GetNamespace(),
		ClusterId:             root.GetClusterId(),
		OwnerCustomResourceId: child.GetId(),
	}

	s.Require().NoError(s.customDS.UpsertCustomResource(s.ctx, root))
	s.Require().NoError(s.customDS.UpsertCustomResource(s.ctx, child))
	serializedDeployment, err := deployment.MarshalVT()
	s.Require().NoError(err)
	_, err = s.db.Exec(s.ctx,
		"INSERT INTO deployments (id, ownercustomresourceid, serialized) VALUES ($1, $2, $3)",
		pgutils.NilOrUUID(deployment.GetId()), pgutils.NilOrUUID(deployment.GetOwnerCustomResourceId()), serializedDeployment)
	s.Require().NoError(err)

	deploymentIDs, err := s.customDS.GetOwnedDeploymentIDs(s.ctx, root.GetId())
	s.Require().NoError(err)
	s.ElementsMatch([]string{deployment.GetId()}, deploymentIDs)
}

func (s *customResourceDataStoreSuite) customResource(name, ownerID string) *storage.CustomResource {
	return &storage.CustomResource{
		Id:         uuid.NewV4().String(),
		Name:       name,
		Namespace:  "ai-workloads",
		ClusterId:  uuid.NewV4().String(),
		ApiGroup:   "serving.kserve.io",
		ApiVersion: "v1beta1",
		Kind:       "InferenceService",
		Resource:   "inferenceservices",
		OwnerId:    ownerID,
	}
}
