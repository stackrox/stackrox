//go:build sql_integration

package m223tom224

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/migrations/m_223_to_m_224_populate_deployment_containers_imageidv2/schema"
	pghelper "github.com/stackrox/rox/migrator/migrations/postgreshelper"
	"github.com/stackrox/rox/migrator/types"
	"github.com/stackrox/rox/pkg/postgres/pgutils"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/uuid"
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
	pgutils.CreateTableFromModel(s.ctx, s.db.GetGormDB(), schema.CreateTableDeploymentsStmt)
}

func (s *migrationTestSuite) TearDownSuite() {
	s.db.Teardown(s.T())
}

func (s *migrationTestSuite) TestMigration() {
	for _, bs := range []int{1, 2, 5} {
		s.Run(fmt.Sprintf("batch_%d", bs), func() {
			batchSize = bs

			// --- Test data ---

			// Deployment 1: two containers that both need image_idv2.
			dep1 := makeDeployment("needs-migration",
				makeContainer("sha256:"+strings.Repeat("a", 64), "registry.example.com/img1:v1@sha256:"+strings.Repeat("a", 64)),
				makeContainer("sha256:"+strings.Repeat("b", 64), "registry.example.com/img2:v1@sha256:"+strings.Repeat("b", 64)),
			)

			// Deployment 2: container without image_id — should NOT be migrated.
			dep2 := makeDeployment("no-image-id",
				makeContainer("", "registry.example.com/img3:latest"),
			)

			// Deployment 3: container that already has image_idv2 — should NOT be migrated.
			dep3FullName := "registry.example.com/img4:v1@sha256:" + strings.Repeat("c", 64)
			dep3Digest := "sha256:" + strings.Repeat("c", 64)
			existingIDV2 := newImageV2ID(dep3FullName, dep3Digest)
			dep3 := makeDeployment("already-migrated",
				&storage.Container{
					Image: &storage.ContainerImage{
						Id:   dep3Digest,
						Name: &storage.ImageName{FullName: dep3FullName},
						IdV2: existingIDV2,
					},
				},
			)

			// Deployment 4: mixed — one container needs update, one doesn't (no image_id).
			dep4 := makeDeployment("mixed",
				makeContainer("sha256:"+strings.Repeat("d", 64), "registry.example.com/img5:v1@sha256:"+strings.Repeat("d", 64)),
				makeContainer("", "registry.example.com/img6:latest"),
			)

			allDeps := []*storage.Deployment{dep1, dep2, dep3, dep4}

			// Insert all deployments and their containers.
			for _, dep := range allDeps {
				s.insertDeployment(dep)
			}

			// --- Run migration ---
			dbs := &types.Databases{
				GormDB:     s.db.GetGormDB(),
				PostgresDB: s.db.DB,
				DBCtx:      s.ctx,
			}
			s.Require().NoError(migration.Run(dbs))

			// --- Verify results ---

			// dep1: both containers should have image_idv2 in column AND blob.
			s.verifyContainerIDV2(dep1, 0, expectedIDV2(dep1.GetContainers()[0]))
			s.verifyContainerIDV2(dep1, 1, expectedIDV2(dep1.GetContainers()[1]))
			s.verifyBlobIDV2(dep1)

			// dep2: container should NOT have image_idv2 (no image_id).
			s.verifyContainerIDV2(dep2, 0, "")

			// dep3: container should still have original image_idv2.
			s.verifyContainerIDV2(dep3, 0, existingIDV2)

			// dep4: first container should have image_idv2, second should not.
			s.verifyContainerIDV2(dep4, 0, expectedIDV2(dep4.GetContainers()[0]))
			s.verifyContainerIDV2(dep4, 1, "")
			s.verifyBlobIDV2(dep4)

			// --- Cleanup ---
			for _, dep := range allDeps {
				_, err := s.db.Exec(s.ctx, "DELETE FROM deployments WHERE id = $1", pgutils.NilOrUUID(dep.GetId()))
				s.Require().NoError(err)
			}
		})
	}
}

// insertDeployment inserts a deployment and its containers into the database.
func (s *migrationTestSuite) insertDeployment(dep *storage.Deployment) {
	serialized, err := dep.MarshalVT()
	s.Require().NoError(err)

	_, err = s.db.Exec(s.ctx,
		"INSERT INTO deployments (id, name, type, namespace, serialized) VALUES ($1, $2, $3, $4, $5)",
		pgutils.NilOrUUID(dep.GetId()), dep.GetName(), dep.GetType(), dep.GetNamespace(), serialized)
	s.Require().NoError(err)

	for idx, c := range dep.GetContainers() {
		_, err = s.db.Exec(s.ctx,
			"INSERT INTO deployments_containers (deployments_id, idx, image_id, image_name_fullname, image_idv2) VALUES ($1, $2, $3, $4, $5)",
			pgutils.NilOrUUID(dep.GetId()), idx,
			c.GetImage().GetId(), c.GetImage().GetName().GetFullName(),
			nullableString(c.GetImage().GetIdV2()))
		s.Require().NoError(err)
	}
}

// verifyContainerIDV2 checks the image_idv2 column value for a specific container.
func (s *migrationTestSuite) verifyContainerIDV2(dep *storage.Deployment, idx int, expected string) {
	var idv2 *string
	err := s.db.QueryRow(s.ctx,
		"SELECT image_idv2 FROM deployments_containers WHERE deployments_id = $1 AND idx = $2",
		pgutils.NilOrUUID(dep.GetId()), idx).Scan(&idv2)
	s.Require().NoError(err)
	if expected == "" {
		s.True(idv2 == nil || *idv2 == "", "expected empty idv2 for deployment %s container %d, got %v", dep.GetName(), idx, idv2)
	} else {
		s.Require().NotNil(idv2, "expected idv2 for deployment %s container %d", dep.GetName(), idx)
		s.Equal(expected, *idv2, "idv2 mismatch for deployment %s container %d", dep.GetName(), idx)
	}
}

// verifyBlobIDV2 checks that the serialized blob has correct id_v2 on all containers.
func (s *migrationTestSuite) verifyBlobIDV2(dep *storage.Deployment) {
	var serialized []byte
	err := s.db.QueryRow(s.ctx,
		"SELECT serialized FROM deployments WHERE id = $1",
		pgutils.NilOrUUID(dep.GetId())).Scan(&serialized)
	s.Require().NoError(err)

	updated := &storage.Deployment{}
	s.Require().NoError(updated.UnmarshalVT(serialized))

	for i, c := range updated.GetContainers() {
		expected := expectedIDV2(c)
		s.Equal(expected, c.GetImage().GetIdV2(),
			"blob idv2 mismatch for deployment %s container %d", dep.GetName(), i)
	}
}

func makeDeployment(name string, containers ...*storage.Container) *storage.Deployment {
	return &storage.Deployment{
		Id:         uuid.NewV4().String(),
		Name:       name,
		Type:       "Deployment",
		Namespace:  "default",
		Containers: containers,
	}
}

func makeContainer(imageID, fullName string) *storage.Container {
	return &storage.Container{
		Image: &storage.ContainerImage{
			Id:   imageID,
			Name: &storage.ImageName{FullName: fullName},
		},
	}
}

func expectedIDV2(c *storage.Container) string {
	return newImageV2ID(c.GetImage().GetName().GetFullName(), c.GetImage().GetId())
}

func nullableString(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
