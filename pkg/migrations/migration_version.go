package migrations

import (
	"os"
	"path/filepath"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/mathutil"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/pkg/version"
	"gopkg.in/yaml.v3"
)

const (
	// MigrationVersionFile records the latest central version in databases.
	MigrationVersionFile     = "migration_version.yaml"
	migrationVersionFileMode = 0644
	lastRocksDBVersion       = "3.74.0"

	// LastPostgresPreviousVersion last software version that uses central_previous
	LastPostgresPreviousVersion = "4.1.0"
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
	MinimumSeqNum int    `yaml:"mindatabase"`
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

// SetCurrent update the database migration version of a database directory.
func SetCurrent(dbPath string) {
	if curr, err := Read(dbPath); err != nil || curr.MainVersion != version.GetMainVersion() || curr.SeqNum != LastRocksDBVersionSeqNum() {
		newVersion := &MigrationVersion{
			dbPath:      dbPath,
			MainVersion: version.GetMainVersion(),
			SeqNum:      mathutil.MinInt(LastRocksDBVersionSeqNum(), CurrentDBVersionSeqNum()),
		}
		err := newVersion.atomicWrite()
		if err != nil {
			utils.Should(errors.Wrapf(err, "failed to write migration version to %s", dbPath))
		}
	}
}

// SealLegacyDB update the database migration version of a database directory to:
// 1) last associated version if it is upgraded from 3.74; or
// 2) 3.74.0 to be consistent with LastRocksDBVersionSeqNum
// If the last associated version is 3.74 or later, then we will keep it untouched;
// otherwise we mark it a fake but possible version for recovery.
func SealLegacyDB(dbPath string) {
	if curr, err := Read(dbPath); err == nil && version.CompareVersions(curr.MainVersion, lastRocksDBVersion) < 0 {
		newVersion := &MigrationVersion{
			dbPath:      dbPath,
			MainVersion: lastRocksDBVersion,
			SeqNum:      LastRocksDBVersionSeqNum(),
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
	return atomicWriteFile(filepath.Join(m.dbPath, MigrationVersionFile), bytes, migrationVersionFileMode)
}

func atomicWriteFile(filename string, bytes []byte, mode os.FileMode) error {
	tempFile, err := os.CreateTemp(filepath.Split(filename))
	if err != nil {
		return err
	}
	tempName := tempFile.Name()

	if _, err := tempFile.Write(bytes); err != nil {
		_ = tempFile.Close()
		return errors.Wrapf(err, "could not write to %s", tempName)
	}

	if err := tempFile.Close(); err != nil {
		return errors.Wrapf(err, "could not close %s", tempName)
	}

	if err := os.Chmod(tempName, mode); err != nil {
		return errors.Wrapf(err, "could not chmod %s", tempName)
	}

	if _, err := os.Stat(tempName); err != nil {
		return errors.Wrapf(err, "could not stat %s", tempName)
	}

	if err = os.Rename(tempName, filename); err != nil {
		return errors.Wrapf(err, "could not rename %s", tempName)
	}

	return nil
}
