package globaldb

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stackrox/rox/pkg/postgres/mocks"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

// versionRow implements pgx.Row for the two columns the version query selects.
type versionRow struct {
	installed string
	available string
	err       error
}

func (r versionRow) Scan(dest ...any) error {
	if r.err != nil {
		return r.err
	}
	if len(dest) != 2 {
		return errors.New("unexpected number of scan destinations")
	}
	installed, ok := dest[0].(*string)
	if !ok {
		return errors.New("unexpected first scan destination")
	}
	available, ok := dest[1].(*string)
	if !ok {
		return errors.New("unexpected second scan destination")
	}
	*installed = r.installed
	*available = r.available
	return nil
}

func TestUpdateExtensionIfOutdatedFlow(t *testing.T) {
	const updateSQL = `ALTER EXTENSION "pg_stat_statements" UPDATE`

	tests := map[string]struct {
		extensionName string
		row           versionRow
		execSQL       string
		execErr       error
		expectExec    bool
		expectedError string
	}{
		"extension not installed": {
			extensionName: "pg_stat_statements",
			row:           versionRow{err: pgx.ErrNoRows},
		},
		"versions match": {
			extensionName: "pg_stat_statements",
			row:           versionRow{installed: "1.10", available: "1.10"},
		},
		"installed version is outdated": {
			extensionName: "pg_stat_statements",
			row:           versionRow{installed: "1.10", available: "1.11"},
			execSQL:       updateSQL,
			expectExec:    true,
		},
		"version query fails": {
			extensionName: "pg_stat_statements",
			row:           versionRow{err: errors.New("query failed")},
			expectedError: `reading "pg_stat_statements" extension version`,
		},
		"update fails": {
			extensionName: "pg_stat_statements",
			row:           versionRow{installed: "1.10", available: "1.11"},
			execSQL:       updateSQL,
			execErr:       errors.New("permission denied"),
			expectExec:    true,
			expectedError: `updating "pg_stat_statements" extension from 1.10 to 1.11`,
		},
		"extension name is identifier-quoted": {
			extensionName: `odd"name`,
			row:           versionRow{installed: "1.10", available: "1.11"},
			execSQL:       `ALTER EXTENSION "odd""name" UPDATE`,
			expectExec:    true,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			db := mocks.NewMockDB(ctrl)

			db.EXPECT().QueryRow(gomock.Any(), gomock.Any(), test.extensionName).Return(test.row)
			if test.expectExec {
				db.EXPECT().Exec(gomock.Any(), test.execSQL).Return(pgconn.CommandTag{}, test.execErr)
			}

			err := updateExtensionIfOutdated(context.Background(), db, test.extensionName)
			if test.expectedError == "" {
				require.NoError(t, err)
				return
			}
			require.Error(t, err)
			require.Contains(t, err.Error(), test.expectedError)
		})
	}
}
