package version

import (
	"github.com/pkg/errors"
	"github.com/stackrox/rox/migrator/log"
	pkgSchema "github.com/stackrox/rox/pkg/postgres/schema"
	"gorm.io/gorm"
)

// WriteRollbackSeqNum atomically sets the rollbackseqnum marker using GORM.
// It only writes if the current marker is 0 (unset) or greater than the provided value,
// ensuring the lowest (oldest) version wins.
func WriteRollbackSeqNum(db *gorm.DB, seqNum int) error {
	val := int32(seqNum)
	result := db.Table(pkgSchema.VersionsSchema.Table).
		Where("rollbackseqnum = 0 OR rollbackseqnum > ?", val).
		Update("rollbackseqnum", val)
	if result.Error != nil {
		return errors.Wrap(result.Error, "writing rollbackseqnum marker")
	}
	log.WriteToStderrf("Wrote rollback marker: rollbackseqnum = %d", seqNum)
	return nil
}

// ClearRollbackSeqNum clears the rollbackseqnum marker using GORM.
func ClearRollbackSeqNum(db *gorm.DB) error {
	result := db.Table(pkgSchema.VersionsSchema.Table).
		Where("1=1").
		Update("rollbackseqnum", 0)
	return errors.Wrap(result.Error, "clearing rollbackseqnum marker")
}
