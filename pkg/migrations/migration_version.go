package migrations

import (
	"io/ioutil"
	"path/filepath"

	"github.com/docker/docker/pkg/ioutils"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/pkg/version"
	"gopkg.in/yaml.v2"
)

const (
	migrationVersionFile     = "migration_version.yaml"
	migrationVersionfileMode = 0644
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
	bytes, err := ioutil.ReadFile(filepath.Join(dbPath, migrationVersionFile))
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
	if curr, err := Read(dbPath); err != nil || curr.MainVersion != version.GetMainVersion() || curr.SeqNum != CurrentDBVersionSeqNum {
		newVersion := &MigrationVersion{
			dbPath:      dbPath,
			MainVersion: version.GetMainVersion(),
			SeqNum:      CurrentDBVersionSeqNum,
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
	return ioutils.AtomicWriteFile(filepath.Join(m.dbPath, migrationVersionFile), bytes, migrationVersionfileMode)
}
