package postgresmigrationhelper

import (
	"gorm.io/gorm"
)

func CountObjectWithModel(gormDB *gorm.DB, model interface{}) int64 {
	var count int64
	gormDB.Model(model).Count(&count)
	return count
}
