package gorm

import (
	"fmt"
	"os"
	"strings"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/migrator/log"
	"github.com/stackrox/rox/pkg/config"
	"github.com/stackrox/rox/pkg/postgres/pgconfig"
	"github.com/stackrox/rox/pkg/postgres/pgutils"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/utils"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var (
	once    sync.Once
	gConfig *gormConfig
)

// Config wraps Gorm configurations to connect to Postgres DB
type Config interface {
	// TODO(ROX-18005) remove this
	Connect(dbName string) (*gorm.DB, error)
	// TODO(ROX-18005) remove this
	ConnectWithRetries(dbName string) (*gorm.DB, error)

	ConnectDatabase() (*gorm.DB, error)
	ConnectDatabaseWithRetries() (*gorm.DB, error)
}

type gormConfig struct {
	source   string
	password string
}

// GetConfig returns Gorm config which can be used to make connection.
func GetConfig() Config {
	once.Do(func() {
		var err error
		gConfig, err = getConfig()
		utils.Must(err)
	})
	return gConfig
}

func getConfig() (*gormConfig, error) {
	log.WriteToStderrf("SHREWS getConfig")
	centralConfig := config.GetConfig()
	password, err := os.ReadFile(pgconfig.DBPasswordFile)
	if err != nil {
		return nil, errors.Wrapf(err, "pgsql: could not load password file %q", pgconfig.DBPasswordFile)
	}
	// Add the password to the source to pass to get the pool config
	source := fmt.Sprintf("%s password=%s", centralConfig.CentralDB.Source, password)
	source = pgutils.PgxpoolDsnToPgxDsn(source)
	gConfig = &gormConfig{source: source, password: string(password)}
	return gConfig, nil
}

// Connect connects to the Postgres database and returns a Gorm DB instance with error if applicable.
// TODO(ROX-18005) remove this
func (gc *gormConfig) Connect(dbName string) (*gorm.DB, error) {
	log.WriteToStderrf("SHREWS Connect to %s", dbName)
	log.WriteToStderrf("SHREWS -- gc.password %q", gc.password)
	source := gc.source
	log.WriteToStderrf("SHREWS -- raw source %q", source)
	log.WriteToStderrf("SHREWS -- %t", strings.Contains(source, gc.password))
	if !pgconfig.IsExternalDatabase() && dbName != "" {
		source = fmt.Sprintf("%s database=%s", gc.source, dbName)
	}
	log.WriteToStderrf("connect to gorm: %v", strings.Replace(source, gc.password, "<REDACTED>", -1))

	db, err := gorm.Open(postgres.Open(source), &gorm.Config{
		NamingStrategy:    pgutils.NamingStrategy,
		CreateBatchSize:   1000,
		AllowGlobalUpdate: true,
		Logger:            logger.Discard,
	})
	if err != nil {
		log.WriteToStderrf("fail to connect to central db %v", err)
		return nil, err
	}
	return db, nil
}

// Close closes a Gorm DB instance.
func Close(db *gorm.DB) {
	if db == nil {
		return
	}
	sqlDB, err := db.DB()
	if err != nil {
		return
	}
	utils.IgnoreError(sqlDB.Close)
}

// ConnectWithRetries ConnectWithRetires connects to the Postgres database and retries if it fails
// TODO(ROX-18005) remove this
func (gc *gormConfig) ConnectWithRetries(dbName string) (db *gorm.DB, err error) {
	// TODO(ROX-12235) be to implemented in seperated PR
	return gc.Connect(dbName)
}

// ConnectDatabase connects to the configured database within the Postgres instance and returns a Gorm DB instance with error if applicable.
func (gc *gormConfig) ConnectDatabase() (*gorm.DB, error) {
	log.WriteToStderrf("SHREWS ConnectDatabase")
	log.WriteToStderrf("SHREWS -- gc.password %q", gc.password)
	source := gc.source
	log.WriteToStderrf("SHREWS -- raw source %q", source)
	log.WriteToStderrf("SHREWS -- %t", strings.Contains(source, gc.password))
	log.WriteToStderrf("connect to gorm: %v", strings.Replace(source, gc.password, "<REDACTED>", -1))

	db, err := gorm.Open(postgres.Open(source), &gorm.Config{
		NamingStrategy:    pgutils.NamingStrategy,
		CreateBatchSize:   1000,
		AllowGlobalUpdate: true,
		Logger:            logger.Discard,
	})
	if err != nil {
		log.WriteToStderrf("fail to connect to central db %v", err)
		return nil, err
	}
	return db, nil
}

// ConnectDatabaseWithRetries connects to the configured database within the Postgres instance and retries if it fails
func (gc *gormConfig) ConnectDatabaseWithRetries() (db *gorm.DB, err error) {
	return gc.ConnectDatabase()
}
