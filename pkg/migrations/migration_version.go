package migrations

import (
	"os"
	"path/filepath"

	"github.com/docker/docker/pkg/ioutils"
	"github.com/pkg/errors"
	"github.com/stackrox/stackrox/pkg/logging"
	"github.com/stackrox/stackrox/pkg/utils"
	"github.com/stackrox/stackrox/pkg/version"
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
	dbPath      string
	MainVersion string `yaml:"image"`
	SeqNum      int    `yaml:"database"`
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

func (m *MigrationVersion) atomicWrite() error {
	bytes, err := yaml.Marshal(m)
	if err != nil {
		return err
	}
	return ioutils.AtomicWriteFile(filepath.Join(m.dbPath, MigrationVersionFile), bytes, migrationVersionFileMode)
}
