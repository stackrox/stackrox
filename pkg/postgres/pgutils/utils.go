package pgutils

import (
	"context"
	"fmt"
	"net"
	"reflect"
	"regexp"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/pkg/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
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

// Logger for Raw SQL portion of migration.
type printSQLLogger struct {
	logger.Interface
}

func (l *printSQLLogger) Trace(ctx context.Context, begin time.Time,
	fc func() (sql string, rowsAffected int64), err error) {

	sql, _ := fc()
	fmt.Println(sql + ";")
	l.Interface.Trace(ctx, begin, fc, err)
}

// ErrNilIfNoRows returns nil if the error is pgx.ErrNoRows
func ErrNilIfNoRows(err error) error {
	if err == pgx.ErrNoRows {
		return nil
	}
	return err
}

// ConvertEnumSliceToIntArray converts an enum slice into a Postgres intarray
func ConvertEnumSliceToIntArray[T ~int32](enumSlice []T) []int32 {
	resultSlice := make([]int32, 0, len(enumSlice))
	for _, v := range enumSlice {
		resultSlice = append(resultSlice, int32(v))
	}
	return resultSlice
}

// NilOrUUID allows for a proto string to be stored as a UUID type in Postgres
func NilOrUUID(value string) *uuid.UUID {
	if value == "" {
		return nil
	}
	id, err := uuid.FromString(value)
	if err != nil {
		return nil
	}
	return &id
}

// NilOrCIDR allows for a proto string to be stored as a CIDR type in Postgres
func NilOrCIDR(value string) *net.IPNet {
	if value == "" {
		return nil
	}
	_, cidr, err := net.ParseCIDR(value)
	if err != nil {
		return nil
	}
	return cidr
}

// EmptyOrMap allows for map to be stored explicit as an empty object ({}) rather than null.
func EmptyOrMap[K comparable, V any, M map[K]V](m M) interface{} {
	if m == nil {
		return make(M)
	}
	return m
}

// CreateTableFromModel executes input create statement using the input connection.
func CreateTableFromModel(ctx context.Context, db *gorm.DB, createStmt *postgres.CreateStmts) {
	// A DB object to use to perform non-GORM changes (i.e partitions creation
	// etc.) via raw SQL
	var rawSQLDB *gorm.DB

	// This function can access the database via GORM AutoMigrate and directly.
	// For the latter we have to respect certain aspects, e.g. DryRun mode,
	// which is used to print out SQL of the migration for troubleshooting.
	if db.DryRun {
		rawSQLDB = db.Session(&gorm.Session{
			Logger: &printSQLLogger{Interface: db.Logger},
		})
	} else {
		rawSQLDB = db
	}

	// Partitioned tables are not supported by Gorm migration or models
	// For partitioned tables the necessary DDL will be contained in PartitionCreate.
	if !createStmt.Partition {
		err := Retry(ctx, func() error {
			return db.WithContext(ctx).AutoMigrate(createStmt.GormModel)
		})
		err = errors.Wrapf(err, "Error creating table for %q: %v", reflect.TypeOf(createStmt.GormModel), err)
		utils.Must(err)
	} else {
		rdb := rawSQLDB.WithContext(ctx).Exec(createStmt.PartitionCreate)
		utils.Must(rdb.Error)
	}

	for _, child := range createStmt.Children {
		CreateTableFromModel(ctx, db, child)
	}

	for _, stmt := range createStmt.PostStmts {
		rdb := rawSQLDB.WithContext(ctx).Exec(stmt)
		utils.Must(rdb.Error)
	}
}

// PgxpoolDsnToPgxDsn removes pgxpoool specific Dsn entries
func PgxpoolDsnToPgxDsn(pgxpoolDsn string) string {
	return strings.TrimSpace(pgxPoolDSNRegex.ReplaceAllString(pgxpoolDsn, ""))
}
