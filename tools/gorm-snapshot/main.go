package main

// GormSnapshot is a tool to take a sql snapshot of the data schema GORM would
// create for an empty database. The schema will be printed out into stdout.
//
// The way it works is we connect to the provided management database, create a
// temporary database, then enhance GORM session with a trace logger and do
// AutoMigrate. At the end the temporary database will be dropped. Note that to
// catch non-GORM bits we rely on the ApplyAllSchemas code to use the same GORM
// session as for AutoMigrate, and thus the same trace logger.
//
// The usage scenario is simply:
//    gorm-snapshot > schema.sql
//
// Unfortunately DryRun is not enough, since due to the way how we bundle
// tables with FKs a single parent table will be created multiple times. Thus
// we have to go with slower "really create everything" approach.

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"strings"
	"time"

	"github.com/stackrox/rox/pkg/postgres/pgutils"
	pkgSchema "github.com/stackrox/rox/pkg/postgres/schema"
	pgGorm "gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type printSQLLogger struct {
	logger.Interface
}

func (l *printSQLLogger) Trace(ctx context.Context, begin time.Time,
	fc func() (sql string, rowsAffected int64), err error) {

	sql, _ := fc()

	// GORM will do lots of selects to learn about the current schema. Do not
	// record them.
	if strings.HasPrefix(sql, "SELECT") {
		return
	}

	fmt.Println(sql + ";")
	l.Interface.Trace(ctx, begin, fc, err)
}

func connect(database string, host string, user string) (*gorm.DB, error) {
	var connectionString = fmt.Sprintf(
		"host=%s database=%s application_name=%s user=%s",
		host, database, "snapshot", user)

	gormDB, err := gorm.Open(pgGorm.Open(connectionString), &gorm.Config{
		NamingStrategy:    pgutils.NamingStrategy,
		CreateBatchSize:   1000,
		AllowGlobalUpdate: true,
		Logger:            logger.Discard,
		QueryFields:       true,
	})

	if err != nil {
		return nil, err
	}

	return gormDB, nil
}

func main() {
	var err error
	var control, snapshot *gorm.DB
	var db *sql.DB

	// If any authentication is expected, provide PGPASSWORD environment
	// variable, it will be picked up by pgx.
	var database = flag.String("database", "postgres", "management database")
	var host = flag.String("host", "localhost", "db host")
	var user = flag.String("user", "postgres", "db user")

	flag.Parse()

	control, err = connect(*database, *host, *user)
	if err != nil {
		fmt.Println(err)
		return
	}

	control.Exec("DROP DATABASE IF EXISTS snapshot")
	control.Exec("CREATE DATABASE snapshot")

	snapshot, err = connect("snapshot", *host, *user)
	if err != nil {
		fmt.Println(err)
		return
	}

	session := snapshot.Session(&gorm.Session{
		Logger: &printSQLLogger{Interface: snapshot.Logger}})
	pkgSchema.ApplyAllSchemas(context.Background(), session)

	// We have to disconnect from the temporary db before dropping it
	db, err = session.DB()
	if err != nil {
		fmt.Println(err)
		return
	}
	err = db.Close()
	if err != nil {
		fmt.Println(err)
		return
	}

	control.Exec("DROP DATABASE IF EXISTS snapshot")
}
