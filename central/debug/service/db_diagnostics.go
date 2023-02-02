package service

import (
	"context"
	"os/exec"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stackrox/rox/central/globaldb"
)

const (
	databaseType = "PostgresDB"
)

func buildDBDiagnosticData(ctx context.Context, dbConfig *pgxpool.Config, dbPool *pgxpool.Pool) centralDBDiagnosticData {
	diagnosticData := centralDBDiagnosticData{}

	// Add the database version if Postgres
	diagnosticData.Database = databaseType
	diagnosticData.DatabaseServerVersion = globaldb.GetPostgresVersion(ctx, dbPool)
	// Get client software version
	diagnosticData.DatabaseClientVersion = getDBClientVersion()
	// Get extensions
	if dbConfig != nil {
		diagnosticData.DatabaseConnectString = strings.TrimSpace(strings.Replace(dbConfig.ConnString(), dbConfig.ConnConfig.Password, "REDACTED", -1))
	}
	diagnosticData.DatabaseExtensions = getPostgresExtensions(ctx, dbPool)

	return diagnosticData
}

func getDBClientVersion() string {
	options := []string{
		"-V",
	}

	clientVersion, err := exec.Command("pg_dump", options...).Output()
	if err != nil {
		log.Errorf("Unable to get Postgres client version:  %v", err)
		return ""
	}

	return strings.TrimSpace(string(clientVersion))
}

func getPostgresExtensions(ctx context.Context, dbPool *pgxpool.Pool) []dbExtension {
	extensionQuery := "SELECT extname, extversion FROM pg_extension;"

	ctx, cancel := context.WithTimeout(ctx, globaldb.PostgresQueryTimeout)
	defer cancel()
	row, err := dbPool.Query(ctx, extensionQuery)
	if err != nil {
		log.Errorf("error fetching Postgres extensions: %v", err)
		return nil
	}

	extSlice := make([]dbExtension, 0)

	defer row.Close()
	for row.Next() {
		var (
			extName    string
			extVersion string
		)
		if err := row.Scan(&extName, &extVersion); err != nil {
			log.Errorf("error extension row: %v", err)
			return nil
		}

		dbExt := dbExtension{
			ExtensionName:    extName,
			ExtensionVersion: extVersion,
		}

		extSlice = append(extSlice, dbExt)
	}

	return extSlice
}
