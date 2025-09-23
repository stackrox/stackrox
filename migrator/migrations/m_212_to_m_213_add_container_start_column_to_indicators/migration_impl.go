package m212tom213

import (
	"github.com/pkg/errors"
	updatedSchema "github.com/stackrox/rox/migrator/migrations/m_212_to_m_213_add_container_start_column_to_indicators/schema"
	"github.com/stackrox/rox/migrator/types"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/postgres/pgutils"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var (
	batchSize = 2000
	log       = logging.LoggerForModule()
)

func migrate(database *types.Databases) error {
	// We are simply promoting a field to a column so the serialized object is unchanged.  Thus, we
	// have no need to worry about the old schema and can simply perform all our work on the new one.
	db := database.GormDB
	db2 := database.GormDB
	pgutils.CreateTableFromModel(database.DBCtx, db, updatedSchema.CreateTableProcessIndicatorsStmt)
	db = db.WithContext(database.DBCtx).Table(updatedSchema.ProcessIndicatorsTableName)
	db2 = db2.WithContext(database.DBCtx).Table(updatedSchema.ProcessIndicatorsTableName)

	var clusters []string
	db.Model(&updatedSchema.ProcessIndicators{}).Distinct("clusterid").Pluck("clusterid", &clusters)
	log.Infof("clusters found: %v", clusters)

	//err := db.Transaction(func(tx *gorm.DB) error {
	//	db.Model(&updatedSchema.ProcessIndicators{}).Distinct("clusterid").Pluck("clusterid", &clusters)
	//
	//	rows, err := tx.Distinct(updatedSchema.ProcessIndicators.Rows()
	//	defer func() { _ = rows.Close() }()
	//	if err != nil {
	//		return errors.Wrapf(err, "failed to iterate table %s", updatedSchema.ProcessIndicatorsTableName)
	//	}
	//	for rows.Next() {
	//		var cluster string
	//		if err = tx.ScanRows(rows, &cluster); err != nil {
	//			return errors.Wrap(err, "failed to scan distinct cluster rows")
	//		}
	//		log.Infof("Found cluster %s", cluster)
	//		clusters = append(clusters, cluster)
	//	}
	//	if rows.Err() != nil {
	//		return errors.Wrapf(rows.Err(), "failed to get distinct clusters for %s", updatedSchema.ProcessIndicatorsTableName)
	//	}
	//	return nil
	//})
	//if err != nil {
	//	return err
	//}

	for _, cluster := range clusters {
		err := migrateByCluster(cluster, database)
		if err != nil {
			return err
		}
	}

	return nil
}

func migrateByCluster(cluster string, database *types.Databases) error {
	log.Infof("Processing %s", cluster)
	db := database.GormDB
	var convertedIndicators []*updatedSchema.ProcessIndicators
	var count int
	err := db.Transaction(func(tx *gorm.DB) error {
		rows, err := tx.WithContext(database.DBCtx).Table(updatedSchema.ProcessIndicatorsTableName).Select("serialized").Where(&updatedSchema.ProcessIndicators{ClusterID: cluster}).Rows()
		//rows, err := tx.Select("serialized").Rows() //.Where(&updatedSchema.ProcessIndicators{ClusterID: cluster}).Rows()
		defer func() { _ = rows.Close() }()
		if err != nil {
			return errors.Wrapf(err, "failed to iterate table %s", updatedSchema.ProcessIndicatorsTableName)
		}
		for rows.Next() {
			var processIndicator *updatedSchema.ProcessIndicators
			if err = tx.ScanRows(rows, &processIndicator); err != nil {
				return errors.Wrap(err, "failed to scan rows")
			}
			processIndicatorProto, err := updatedSchema.ConvertProcessIndicatorToProto(processIndicator)
			if err != nil {
				return errors.Wrapf(err, "failed to convert %+v to proto", processIndicator)
			}

			// We only need to rewrite the ones where the time is not null
			if processIndicatorProto.GetContainerStartTime() == nil {
				continue
			}

			converted, err := updatedSchema.ConvertProcessIndicatorFromProto(processIndicatorProto)
			if err != nil {
				return errors.Wrapf(err, "failed to convert from proto %+v", processIndicatorProto)
			}

			convertedIndicators = append(convertedIndicators, converted)
			count++
		}
		if rows.Err() != nil {
			return errors.Wrapf(rows.Err(), "failed to get rows for %s", updatedSchema.ProcessIndicatorsTableName)
		}
		return nil

	})
	if err != nil {
		return err
	}

	if len(convertedIndicators) > 0 {
		if err = db.
			Clauses(clause.OnConflict{UpdateAll: true}).
			Model(updatedSchema.CreateTableProcessIndicatorsStmt.GormModel).
			CreateInBatches(&convertedIndicators, batchSize).Error; err != nil {
			return errors.Wrap(err, "failed to upsert all converted objects")
		}
	}

	log.Infof("Populated container start time for %d process indicators", count)
	return nil
}
