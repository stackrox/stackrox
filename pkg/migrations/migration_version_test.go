package migrations

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/pkg/version"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMigrationVersion_Read(t *testing.T) {
	testCases := []struct {
		description string
		prepFunc    func(dbPath string)
		shouldFail  bool
		expectedVer string
		expectedSeq int
	}{
		{
			description: "Migration version missing",
			prepFunc:    nil,
			shouldFail:  false,
			expectedVer: "0",
			expectedSeq: 0,
		},
		{
			description: "Migration version corrupted",
			prepFunc: func(dbPath string) {
				f, err := os.Create(filepath.Join(dbPath, MigrationVersionFile))
				require.NoError(t, err)
				defer utils.IgnoreError(f.Close)
				_, err = f.Write([]byte("Something"))
				require.NoError(t, err)
			},
			shouldFail:  true,
			expectedVer: version.GetMainVersion(),
			expectedSeq: LastRocksDBVersionSeqNum(),
		},
		{
			description: "Migration version exists",
			prepFunc: func(dbPath string) {
				SetCurrent(dbPath)
			},
			shouldFail:  false,
			expectedVer: version.GetMainVersion(),
			expectedSeq: LastRocksDBVersionSeqNum(),
		},
	}

	for _, c := range testCases {
		t.Run(c.description, func(t *testing.T) {
			dir := t.TempDir()
			if c.prepFunc != nil {
				c.prepFunc(dir)
			}
			ver, err := Read(dir)
			require.Equal(t, c.shouldFail, err != nil)
			if !c.shouldFail {
				assert.Equal(t, c.expectedVer, ver.MainVersion)
				assert.Equal(t, c.expectedSeq, ver.SeqNum)
				assert.Equal(t, dir, ver.dbPath)
			}
		})
	}

}

func TestMigrationVersion_Write(t *testing.T) {
	testCases := []struct {
		description  string
		prepFunc     func(dbPath string)
		shouldUpdate bool
	}{
		{
			description:  "Migration version missing",
			prepFunc:     nil,
			shouldUpdate: true,
		},
		{
			description: "Migration version outdated",
			prepFunc: func(dbPath string) {
				ver := &MigrationVersion{
					dbPath:      dbPath,
					MainVersion: "",
					SeqNum:      9,
				}
				err := ver.atomicWrite()
				assert.NoError(t, err)
			},
			shouldUpdate: true,
		},
		{
			description: "Migration version current",
			prepFunc: func(dbPath string) {
				SetCurrent(dbPath)
			},
			shouldUpdate: false,
		},
	}

	for _, c := range testCases {
		t.Run(c.description, func(t *testing.T) {
			dir := t.TempDir()
			if c.prepFunc != nil {
				c.prepFunc(dir)
			}

			// Verify the migration file updated when needed.
			stat, err := os.Stat(filepath.Join(dir, MigrationVersionFile))
			time.Sleep(time.Millisecond * 10) // Make sure mod time changed.
			SetCurrent(dir)
			if err == nil {
				newStat, err := os.Stat(filepath.Join(dir, MigrationVersionFile))
				assert.NoError(t, err)
				assert.Equal(t, c.shouldUpdate, !stat.ModTime().Equal(newStat.ModTime()))
			}

			// Verify the content of migration version file.
			ver, err := Read(dir)
			require.NoError(t, err)

			assert.Equal(t, version.GetMainVersion(), ver.MainVersion)
			assert.Equal(t, LastRocksDBVersionSeqNum(), ver.SeqNum)
			assert.Equal(t, dir, ver.dbPath)
		})
	}
}
