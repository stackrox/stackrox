package imagecveedges

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/stackrox/rox/generated/storage"
	migrationSchema "github.com/stackrox/rox/migrator/migrations/m_202_to_m_203_vuln_requests_for_suppressed_cves/schema"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/sac/resources"
	pgSearch "github.com/stackrox/rox/pkg/search/postgres"
)

var (
	log            = logging.LoggerForModule()
	schema         = migrationSchema.ImageCveEdgesSchema
	targetResource = resources.Image
)

type storeType = storage.ImageCVEEdge

// Store is the interface to interact with the storage for storage.ImageCVEEdge
type Store interface {
	UpsertMany(ctx context.Context, objs []*storeType) error

	Walk(ctx context.Context, fn func(obj *storeType) error) error
}

// New returns a new Store instance using the provided sql instance.
func New(db postgres.DB) Store {
	return pgSearch.NewGloballyScopedGenericStore[storeType, *storeType](
		db,
		schema,
		pkGetter,
		insertIntoImageCveEdges,
		copyFromImageCveEdges,
		nil,
		nil,
		targetResource,
	)
}

// region Helper functions

func pkGetter(obj *storeType) string {
	return obj.GetId()
}

func insertIntoImageCveEdges(batch *pgx.Batch, obj *storage.ImageCVEEdge) error {

	serialized, marshalErr := obj.MarshalVT()
	if marshalErr != nil {
		return marshalErr
	}

	values := []interface{}{
		// parent primary keys start
		obj.GetId(),
		protocompat.NilOrTime(obj.GetFirstImageOccurrence()),
		obj.GetState(),
		obj.GetImageId(),
		obj.GetImageCveId(),
		serialized,
	}

	finalStr := "INSERT INTO image_cve_edges (Id, FirstImageOccurrence, State, ImageId, ImageCveId, serialized) VALUES($1, $2, $3, $4, $5, $6) ON CONFLICT(Id) DO UPDATE SET Id = EXCLUDED.Id, FirstImageOccurrence = EXCLUDED.FirstImageOccurrence, State = EXCLUDED.State, ImageId = EXCLUDED.ImageId, ImageCveId = EXCLUDED.ImageCveId, serialized = EXCLUDED.serialized"
	batch.Queue(finalStr, values...)

	return nil
}

func copyFromImageCveEdges(ctx context.Context, s pgSearch.Deleter, tx *postgres.Tx, objs ...*storage.ImageCVEEdge) error {
	batchSize := pgSearch.MaxBatchSize
	inputRows := make([][]interface{}, 0, batchSize)

	// This is a copy so first we must delete the rows and re-add them
	// Which is essentially the desired behaviour of an upsert.
	deletes := make([]string, 0, batchSize)

	copyCols := []string{
		"id",
		"firstimageoccurrence",
		"state",
		"imageid",
		"imagecveid",
		"serialized",
	}

	for idx, obj := range objs {
		// Todo: ROX-9499 Figure out how to more cleanly template around this issue.
		log.Debugf("This is here for now because there is an issue with pods_TerminatedInstances where the obj "+
			"in the loop is not used as it only consists of the parent ID and the index.  Putting this here as a stop gap "+
			"to simply use the object.  %s", obj)

		serialized, marshalErr := obj.MarshalVT()
		if marshalErr != nil {
			return marshalErr
		}

		inputRows = append(inputRows, []interface{}{
			obj.GetId(),
			protocompat.NilOrTime(obj.GetFirstImageOccurrence()),
			obj.GetState(),
			obj.GetImageId(),
			obj.GetImageCveId(),
			serialized,
		})

		// Add the ID to be deleted.
		deletes = append(deletes, obj.GetId())

		// if we hit our batch size we need to push the data
		if (idx+1)%batchSize == 0 || idx == len(objs)-1 {
			// copy does not upsert so have to delete first.  parent deletion cascades so only need to
			// delete for the top level parent

			if err := s.DeleteMany(ctx, deletes); err != nil {
				return err
			}
			// clear the inserts and vals for the next batch
			deletes = deletes[:0]

			if _, err := tx.CopyFrom(ctx, pgx.Identifier{"image_cve_edges"}, copyCols, pgx.CopyFromRows(inputRows)); err != nil {
				return err
			}
			// clear the input rows for the next batch
			inputRows = inputRows[:0]
		}
	}

	return nil
}

// endregion Helper functions

// region Used for testing
