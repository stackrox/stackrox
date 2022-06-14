package pgutils

import (
	"context"
	"reflect"
	"regexp"
	"strings"
	"time"

	"github.com/gogo/protobuf/types"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/pkg/errors"
	"github.com/stackrox/stackrox/pkg/logging"
	"github.com/stackrox/stackrox/pkg/postgres"
	"github.com/stackrox/stackrox/pkg/utils"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

var (
	log = logging.LoggerForModule()

	// NamingStrategy explicitly defines the naming strategy for Postgres
	// Do not change this strategy after PostgresDB released. It has global impact on the
	// names of PostgresDB tables, columns etc.
	// If you have to, consider making a data migration plan.
	NamingStrategy = schema.NamingStrategy{
		TablePrefix:   "",
		SingularTable: false,
		NameReplacer:  nil,
		NoLowerCase:   false,
	}
	pgxPoolDSNRegex = regexp.MustCompile(`(^| )(pool_max_conns|pool_min_conns|pool_max_conn_lifetime|pool_max_conn_idle_time|pool_health_check_period)=\S+`)
)

// ErrNilIfNoRows returns nil if the error is pgx.ErrNoRows
func ErrNilIfNoRows(err error) error {
	if err == pgx.ErrNoRows {
		return nil
	}
	return err
}

// ConvertEnumSliceToIntArray converts an enum slice into a Postgres intarray
func ConvertEnumSliceToIntArray(i interface{}) []int32 {
	enumSlice := reflect.ValueOf(i)
	enumSliceLen := enumSlice.Len()
	resultSlice := make([]int32, 0, enumSliceLen)
	for i := 0; i < enumSlice.Len(); i++ {
		resultSlice = append(resultSlice, int32(enumSlice.Index(i).Int()))
	}
	return resultSlice
}

// NilOrTime allows for a proto timestamp to be stored a timestamp type in Postgres
func NilOrTime(t *types.Timestamp) *time.Time {
	if t == nil {
		return nil
	}
	ts, err := types.TimestampFromProto(t)
	if err != nil {
		return nil
	}
	return &ts
}

// CreateTable executes input create statement using the input connection.
func CreateTable(ctx context.Context, db *pgxpool.Pool, createStmt *postgres.CreateStmts) {
	_, err := db.Exec(ctx, createStmt.Table)
	if err != nil {
		log.Panicf("Error creating table %s: %v", createStmt.Table, err)
	}

	for _, index := range createStmt.Indexes {
		if _, err := db.Exec(ctx, index); err != nil {
			log.Panicf("Error creating index %s: %v", index, err)
		}
	}

	for _, child := range createStmt.Children {
		CreateTable(ctx, db, child)
	}
}

// CreateTableFromModel executes input create statement using the input connection.
func CreateTableFromModel(ctx context.Context, db *gorm.DB, createStmt *postgres.CreateStmts) {
	err := db.WithContext(ctx).AutoMigrate(createStmt.GormModel)
	err = errors.Wrapf(err, "Error creating table %s: %v", createStmt.Table, err)
	utils.Must(err)

	for _, child := range createStmt.Children {
		CreateTableFromModel(ctx, db, child)
	}
	for _, stmt := range createStmt.PostStmts {
		rdb := db.WithContext(ctx).Exec(stmt)
		utils.Must(rdb.Error)
	}
}

// PgxpoolDsnToPgxDsn removes pgxpoool specific Dsn entries
func PgxpoolDsnToPgxDsn(pgxpoolDsn string) string {
	return strings.TrimSpace(pgxPoolDSNRegex.ReplaceAllString(pgxpoolDsn, ""))
}
