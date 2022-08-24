package migrations

import (
	"context"
	"os"
	"path/filepath"
	"time"

	"github.com/docker/docker/pkg/ioutils"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/timestamp"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/pkg/version"
	vStore "github.com/stackrox/rox/pkg/version/postgres"
	"gopkg.in/yaml.v3"
)

const (
	// MigrationVersionFile records the latest central version in databases.
	MigrationVersionFile     = "migration_version.yaml"
	migrationVersionFileMode = 0644
)

var (
	log = logging.LoggerForModule()
)

// MigrationVersion is the last central version and migration sequence number
// that run with a database successfully. If central was up and ready to serve,
// it will update the migration version of current database if needed.
type MigrationVersion struct {
	dbPath        string
	MainVersion   string `yaml:"image"`
	SeqNum        int    `yaml:"database"`
	LastPersisted time.Time
}

// Read reads the migration version from dbPath.
func Read(dbPath string) (*MigrationVersion, error) {
	path := filepath.Join(dbPath, MigrationVersionFile)
	// If the migration file does not exist, the databases come from a version earlier than 3.0.57.0.
	_, err := os.Stat(path)
	if os.IsNotExist(err) {
		return &MigrationVersion{dbPath: dbPath, SeqNum: 0, MainVersion: "0"}, nil
	}

	bytes, err := os.ReadFile(filepath.Join(dbPath, MigrationVersionFile))
	if err != nil {
		return nil, err
	}

	version := &MigrationVersion{}
	version.dbPath = dbPath

	if err = yaml.Unmarshal(bytes, version); err != nil {
		log.Errorf("failed to get migration version from %s: %v", dbPath, err)
		return nil, err
	}

	log.Infof("Migration version of database at %v: %v", dbPath, version)
	return version, nil
}

// ReadVersionPostgres - reads the version from the postgres database.
func ReadVersionPostgres(pool *pgxpool.Pool) (*MigrationVersion, error) {
	ctx := sac.WithAllAccess(context.Background())

	store := vStore.New(ctx, pool)

	version, exists, err := store.Get(ctx)
	if err != nil {
		return nil, err
	}

	if !exists {
		return &MigrationVersion{MainVersion: "0", SeqNum: 0}, nil
	}

	log.Infof("Migration version from DB = %s.", version)

	return &MigrationVersion{
		MainVersion:   version.Version,
		SeqNum:        int(version.SeqNum),
		LastPersisted: timestamp.FromProtobuf(version.GetLastPersisted()).GoTime(),
	}, nil
}

// SetCurrent update the database migration version of a database directory.
func SetCurrent(dbPath string) {
	if curr, err := Read(dbPath); err != nil || curr.MainVersion != version.GetMainVersion() || curr.SeqNum != CurrentDBVersionSeqNum() {
		newVersion := &MigrationVersion{
			dbPath:      dbPath,
			MainVersion: version.GetMainVersion(),
			SeqNum:      CurrentDBVersionSeqNum(),
		}
		err := newVersion.atomicWrite()
		if err != nil {
			utils.Should(errors.Wrapf(err, "failed to write migration version to %s", dbPath))
		}
	}
}

// SetCurrentVersionPostgres - sets the current version in the postgres database
func SetCurrentVersionPostgres(pool *pgxpool.Pool) {
	if curr, err := ReadVersionPostgres(pool); err != nil || curr.MainVersion != version.GetMainVersion() || curr.SeqNum != CurrentDBVersionSeqNum() {
		newVersion := &storage.Version{
			SeqNum:        int32(CurrentDBVersionSeqNum()),
			Version:       version.GetMainVersion(),
			LastPersisted: timestamp.Now().GogoProtobuf(),
		}
		SetVersionPostgres(pool, newVersion)
	}
}

// SetVersionPostgres - sets the specified version in the postgres database
func SetVersionPostgres(pool *pgxpool.Pool, updatedVersion *storage.Version) {
	ctx := sac.WithAllAccess(context.Background())
	store := vStore.New(ctx, pool)

	err := store.Upsert(ctx, updatedVersion)
	if err != nil {
		utils.Should(errors.Wrapf(err, "failed to write migration version to %s", pool.Config().ConnConfig.Database))
	}
}

func (m *MigrationVersion) atomicWrite() error {
	bytes, err := yaml.Marshal(m)
	if err != nil {
		return err
	}
	return ioutils.AtomicWriteFile(filepath.Join(m.dbPath, MigrationVersionFile), bytes, migrationVersionFileMode)
}
