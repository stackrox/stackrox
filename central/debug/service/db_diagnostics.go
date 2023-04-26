package service

import (
	"context"
	"os/exec"
	"strings"

	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/pkg/postgres"
)

const (
	databaseType = "PostgresDB"
)

func buildDBDiagnosticData(ctx context.Context, dbConfig *postgres.Config, dbPool postgres.DB) centralDBDiagnosticData {
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

func getPostgresExtensions(ctx context.Context, dbPool postgres.DB) []dbExtension {
	extensionQuery := "SELECT extname, extversion FROM pg_extension;"

	ctx, cancel := context.WithTimeout(ctx, globaldb.PostgresQueryTimeout)
	defer cancel()
	rows, err := dbPool.Query(ctx, extensionQuery)
	if err != nil {
		log.Errorf("error fetching Postgres extensions: %v", err)
		return nil
	}
	defer rows.Close()

	extSlice := make([]dbExtension, 0)

	for rows.Next() {
		var (
			extName    string
			extVersion string
		)
		if err := rows.Scan(&extName, &extVersion); err != nil {
			log.Errorf("error extension row: %v", err)
			return nil
		}

		dbExt := dbExtension{
			ExtensionName:    extName,
			ExtensionVersion: extVersion,
		}

		extSlice = append(extSlice, dbExt)
	}

	if err := rows.Err(); err != nil {
		log.Errorf("error getting complete extension information: %v", err)
	}

	return extSlice
}
