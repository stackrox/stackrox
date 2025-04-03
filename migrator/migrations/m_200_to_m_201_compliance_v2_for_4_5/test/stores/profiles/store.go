package postgres

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/stackrox/rox/generated/storage"
	oldSchema "github.com/stackrox/rox/migrator/migrations/m_200_to_m_201_compliance_v2_for_4_5/test/schema"
	"github.com/stackrox/rox/pkg/logging"
	ops "github.com/stackrox/rox/pkg/metrics"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/pgutils"
	"github.com/stackrox/rox/pkg/sac/resources"
	pgSearch "github.com/stackrox/rox/pkg/search/postgres"
)

var (
	log            = logging.LoggerForModule()
	schema         = oldSchema.ComplianceOperatorProfileV2Schema
	targetResource = resources.Compliance
)

type storeType = storage.ComplianceOperatorProfileV2

// Store is the interface to interact with the storage for storage.ComplianceOperatorProfileV2
type Store interface {
	Upsert(ctx context.Context, obj *storeType) error
	UpsertMany(ctx context.Context, objs []*storeType) error

	Get(ctx context.Context, id string) (*storeType, bool, error)
	GetMany(ctx context.Context, identifiers []string) ([]*storeType, []int, error)
}

// New returns a new Store instance using the provided sql instance.
func New(db postgres.DB) Store {
	return pgSearch.NewGenericStore[storeType, *storeType](
		db,
		schema,
		pkGetter,
		insertIntoComplianceOperatorProfileV2,
		copyFromComplianceOperatorProfileV2,
		metricsSetAcquireDBConnDuration,
		metricsSetPostgresOperationDurationTime,
		pgSearch.GloballyScopedUpsertChecker[storeType, *storeType](targetResource),
		targetResource,
	)
}

// region Helper functions

func pkGetter(obj *storeType) string {
	return obj.GetId()
}

func metricsSetPostgresOperationDurationTime(_ time.Time, _ ops.Op) {}

func metricsSetAcquireDBConnDuration(_ time.Time, _ ops.Op) {}

func insertIntoComplianceOperatorProfileV2(batch *pgx.Batch, obj *storage.ComplianceOperatorProfileV2) error {

	serialized, marshalErr := obj.MarshalVT()
	if marshalErr != nil {
		return marshalErr
	}

	values := []interface{}{
		// parent primary keys start
		obj.GetId(),
		obj.GetProfileId(),
		obj.GetName(),
		obj.GetProfileVersion(),
		obj.GetProductType(),
		obj.GetStandard(),
		pgutils.NilOrUUID(obj.GetClusterId()),
		serialized,
	}

	finalStr := "INSERT INTO compliance_operator_profile_v2 (Id, ProfileId, Name, ProfileVersion, ProductType, Standard, ClusterId, serialized) VALUES($1, $2, $3, $4, $5, $6, $7, $8) ON CONFLICT(Id) DO UPDATE SET Id = EXCLUDED.Id, ProfileId = EXCLUDED.ProfileId, Name = EXCLUDED.Name, ProfileVersion = EXCLUDED.ProfileVersion, ProductType = EXCLUDED.ProductType, Standard = EXCLUDED.Standard, ClusterId = EXCLUDED.ClusterId, serialized = EXCLUDED.serialized"
	batch.Queue(finalStr, values...)

	var query string

	for childIndex, child := range obj.GetRules() {
		if err := insertIntoComplianceOperatorProfileV2Rules(batch, child, obj.GetId(), childIndex); err != nil {
			return err
		}
	}

	query = "delete from compliance_operator_profile_v2_rules where compliance_operator_profile_v2_Id = $1 AND idx >= $2"
	batch.Queue(query, obj.GetId(), len(obj.GetRules()))
	return nil
}

func insertIntoComplianceOperatorProfileV2Rules(batch *pgx.Batch, obj *storage.ComplianceOperatorProfileV2_Rule, complianceOperatorProfileV2ID string, idx int) error {

	values := []interface{}{
		// parent primary keys start
		complianceOperatorProfileV2ID,
		idx,
		obj.GetRuleName(),
	}

	finalStr := "INSERT INTO compliance_operator_profile_v2_rules (compliance_operator_profile_v2_Id, idx, RuleName) VALUES($1, $2, $3) ON CONFLICT(compliance_operator_profile_v2_Id, idx) DO UPDATE SET compliance_operator_profile_v2_Id = EXCLUDED.compliance_operator_profile_v2_Id, idx = EXCLUDED.idx, RuleName = EXCLUDED.RuleName"
	batch.Queue(finalStr, values...)

	return nil
}

func copyFromComplianceOperatorProfileV2(ctx context.Context, s pgSearch.Deleter, tx *postgres.Tx, objs ...*storage.ComplianceOperatorProfileV2) error {
	batchSize := pgSearch.MaxBatchSize
	if len(objs) < batchSize {
		batchSize = len(objs)
	}
	inputRows := make([][]interface{}, 0, batchSize)

	// This is a copy so first we must delete the rows and re-add them
	// Which is essentially the desired behaviour of an upsert.
	deletes := make([]string, 0, batchSize)

	copyCols := []string{
		"id",
		"profileid",
		"name",
		"profileversion",
		"producttype",
		"standard",
		"clusterid",
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
			obj.GetProfileId(),
			obj.GetName(),
			obj.GetProfileVersion(),
			obj.GetProductType(),
			obj.GetStandard(),
			pgutils.NilOrUUID(obj.GetClusterId()),
			serialized,
		})

		// Add the ID to be deleted.
		deletes = append(deletes, obj.GetId())

		// if we hit our batch size we need to push the data
		if (idx+1)%batchSize == 0 || idx == len(objs)-1 {
			// copy does not upsert so have to delete first.  parent deletion cascades so only need to
			// delete for the top level parent

			if err := s.Delete(ctx, deletes...); err != nil {
				return err
			}
			// clear the inserts and vals for the next batch
			deletes = deletes[:0]

			if _, err := tx.CopyFrom(ctx, pgx.Identifier{"compliance_operator_profile_v2"}, copyCols, pgx.CopyFromRows(inputRows)); err != nil {
				return err
			}
			// clear the input rows for the next batch
			inputRows = inputRows[:0]
		}
	}

	for idx, obj := range objs {
		_ = idx // idx may or may not be used depending on how nested we are, so avoid compile-time errors.

		if err := copyFromComplianceOperatorProfileV2Rules(ctx, s, tx, obj.GetId(), obj.GetRules()...); err != nil {
			return err
		}
	}

	return nil
}

func copyFromComplianceOperatorProfileV2Rules(ctx context.Context, s pgSearch.Deleter, tx *postgres.Tx, complianceOperatorProfileV2ID string, objs ...*storage.ComplianceOperatorProfileV2_Rule) error {
	batchSize := pgSearch.MaxBatchSize
	if len(objs) < batchSize {
		batchSize = len(objs)
	}
	inputRows := make([][]interface{}, 0, batchSize)

	copyCols := []string{
		"compliance_operator_profile_v2_id",
		"idx",
		"rulename",
	}

	for idx, obj := range objs {
		// Todo: ROX-9499 Figure out how to more cleanly template around this issue.
		log.Debugf("This is here for now because there is an issue with pods_TerminatedInstances where the obj "+
			"in the loop is not used as it only consists of the parent ID and the index.  Putting this here as a stop gap "+
			"to simply use the object.  %s", obj)

		inputRows = append(inputRows, []interface{}{
			complianceOperatorProfileV2ID,
			idx,
			obj.GetRuleName(),
		})

		// if we hit our batch size we need to push the data
		if (idx+1)%batchSize == 0 || idx == len(objs)-1 {
			// copy does not upsert so have to delete first.  parent deletion cascades so only need to
			// delete for the top level parent

			if _, err := tx.CopyFrom(ctx, pgx.Identifier{"compliance_operator_profile_v2_rules"}, copyCols, pgx.CopyFromRows(inputRows)); err != nil {
				return err
			}
			// clear the input rows for the next batch
			inputRows = inputRows[:0]
		}
	}

	return nil
}

// endregion Helper functions
