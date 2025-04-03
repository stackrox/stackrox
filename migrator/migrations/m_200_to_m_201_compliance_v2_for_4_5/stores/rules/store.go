package postgres

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/stackrox/rox/generated/storage"
	newSchema "github.com/stackrox/rox/migrator/migrations/m_200_to_m_201_compliance_v2_for_4_5/schema/new"
	"github.com/stackrox/rox/pkg/logging"
	ops "github.com/stackrox/rox/pkg/metrics"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/pgutils"
	"github.com/stackrox/rox/pkg/sac/resources"
	pgSearch "github.com/stackrox/rox/pkg/search/postgres"
)

var (
	log            = logging.LoggerForModule()
	schema         = newSchema.ComplianceOperatorRuleV2Schema
	targetResource = resources.Compliance
)

type storeType = storage.ComplianceOperatorRuleV2

// Store is the interface to interact with the storage for storage.ComplianceOperatorRuleV2
type Store interface {
	Upsert(ctx context.Context, obj *storeType) error
	UpsertMany(ctx context.Context, objs []*storeType) error
	Walk(ctx context.Context, fn func(obj *storeType) error) error
}

// New returns a new Store instance using the provided sql instance.
func New(db postgres.DB) Store {
	return pgSearch.NewGenericStore[storeType, *storeType](
		db,
		schema,
		pkGetter,
		insertIntoComplianceOperatorRuleV2,
		copyFromComplianceOperatorRuleV2,
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

func insertIntoComplianceOperatorRuleV2(batch *pgx.Batch, obj *storage.ComplianceOperatorRuleV2) error {

	serialized, marshalErr := obj.MarshalVT()
	if marshalErr != nil {
		return marshalErr
	}

	values := []interface{}{
		// parent primary keys start
		obj.GetId(),
		obj.GetName(),
		obj.GetRuleType(),
		obj.GetSeverity(),
		pgutils.NilOrUUID(obj.GetClusterId()),
		pgutils.NilOrUUID(obj.GetRuleRefId()),
		serialized,
	}

	finalStr := "INSERT INTO compliance_operator_rule_v2 (Id, Name, RuleType, Severity, ClusterId, RuleRefId, serialized) VALUES($1, $2, $3, $4, $5, $6, $7) ON CONFLICT(Id) DO UPDATE SET Id = EXCLUDED.Id, Name = EXCLUDED.Name, RuleType = EXCLUDED.RuleType, Severity = EXCLUDED.Severity, ClusterId = EXCLUDED.ClusterId, RuleRefId = EXCLUDED.RuleRefId, serialized = EXCLUDED.serialized"
	batch.Queue(finalStr, values...)

	var query string

	for childIndex, child := range obj.GetControls() {
		if err := insertIntoComplianceOperatorRuleV2Controls(batch, child, obj.GetId(), childIndex); err != nil {
			return err
		}
	}

	query = "delete from compliance_operator_rule_v2_controls where compliance_operator_rule_v2_Id = $1 AND idx >= $2"
	batch.Queue(query, obj.GetId(), len(obj.GetControls()))
	return nil
}

func insertIntoComplianceOperatorRuleV2Controls(batch *pgx.Batch, obj *storage.RuleControls, complianceOperatorRuleV2ID string, idx int) error {

	values := []interface{}{
		// parent primary keys start
		complianceOperatorRuleV2ID,
		idx,
		obj.GetStandard(),
		obj.GetControl(),
	}

	finalStr := "INSERT INTO compliance_operator_rule_v2_controls (compliance_operator_rule_v2_Id, idx, Standard, Control) VALUES($1, $2, $3, $4) ON CONFLICT(compliance_operator_rule_v2_Id, idx) DO UPDATE SET compliance_operator_rule_v2_Id = EXCLUDED.compliance_operator_rule_v2_Id, idx = EXCLUDED.idx, Standard = EXCLUDED.Standard, Control = EXCLUDED.Control"
	batch.Queue(finalStr, values...)

	return nil
}

func copyFromComplianceOperatorRuleV2(ctx context.Context, s pgSearch.Deleter, tx *postgres.Tx, objs ...*storage.ComplianceOperatorRuleV2) error {
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
		"name",
		"ruletype",
		"severity",
		"clusterid",
		"rulerefid",
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
			obj.GetName(),
			obj.GetRuleType(),
			obj.GetSeverity(),
			pgutils.NilOrUUID(obj.GetClusterId()),
			pgutils.NilOrUUID(obj.GetRuleRefId()),
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

			if _, err := tx.CopyFrom(ctx, pgx.Identifier{"compliance_operator_rule_v2"}, copyCols, pgx.CopyFromRows(inputRows)); err != nil {
				return err
			}
			// clear the input rows for the next batch
			inputRows = inputRows[:0]
		}
	}

	for idx, obj := range objs {
		_ = idx // idx may or may not be used depending on how nested we are, so avoid compile-time errors.

		if err := copyFromComplianceOperatorRuleV2Controls(ctx, s, tx, obj.GetId(), obj.GetControls()...); err != nil {
			return err
		}
	}

	return nil
}

func copyFromComplianceOperatorRuleV2Controls(ctx context.Context, s pgSearch.Deleter, tx *postgres.Tx, complianceOperatorRuleV2ID string, objs ...*storage.RuleControls) error {
	batchSize := pgSearch.MaxBatchSize
	if len(objs) < batchSize {
		batchSize = len(objs)
	}
	inputRows := make([][]interface{}, 0, batchSize)

	copyCols := []string{
		"compliance_operator_rule_v2_id",
		"idx",
		"standard",
		"control",
	}

	for idx, obj := range objs {
		// Todo: ROX-9499 Figure out how to more cleanly template around this issue.
		log.Debugf("This is here for now because there is an issue with pods_TerminatedInstances where the obj "+
			"in the loop is not used as it only consists of the parent ID and the index.  Putting this here as a stop gap "+
			"to simply use the object.  %s", obj)

		inputRows = append(inputRows, []interface{}{
			complianceOperatorRuleV2ID,
			idx,
			obj.GetStandard(),
			obj.GetControl(),
		})

		// if we hit our batch size we need to push the data
		if (idx+1)%batchSize == 0 || idx == len(objs)-1 {
			// copy does not upsert so have to delete first.  parent deletion cascades so only need to
			// delete for the top level parent

			if _, err := tx.CopyFrom(ctx, pgx.Identifier{"compliance_operator_rule_v2_controls"}, copyCols, pgx.CopyFromRows(inputRows)); err != nil {
				return err
			}
			// clear the input rows for the next batch
			inputRows = inputRows[:0]
		}
	}

	return nil
}

// endregion Helper functions
