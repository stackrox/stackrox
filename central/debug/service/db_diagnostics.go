package service

import (
	"context"
	"os/exec"
	"strings"

	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/pkg/postgres/pgconfig"
)

func buildDBDiagnosticData(ctx context.Context) centralDBDiagnosticData {
	diagnosticData := centralDBDiagnosticData{}
	_, dbConfig, err := pgconfig.GetPostgresConfig()
	if err != nil {
		log.Warnf("Could not parse postgres config: %v", err)
		return centralDBDiagnosticData{}
	}

	// Add the database version if Postgres
	diagnosticData.Database = "PostgresDB"
	diagnosticData.DatabaseServerVersion = globaldb.GetPostgresVersion(ctx, globaldb.GetPostgres())
	// Get client software version
	diagnosticData.DatabaseClientVersion = getDBClientVersion()
	// Get extensions
	diagnosticData.DatabaseConnectString = strings.Replace(dbConfig.ConnString(), dbConfig.ConnConfig.Password, "REDACTED", -1)
	diagnosticData.DatabaseExtensions = getPostgresExtensions(ctx)

	return diagnosticData
}

func getDBClientVersion() string {
	options := []string{
		"-V",
	}

	clientVersion, err := exec.Command("pg_dump", options...).Output()
	if err != nil {
		log.Errorf("Unable to get client version:  %v", err)
		return ""
	}

	return strings.TrimSpace(string(clientVersion))
}

func getPostgresExtensions(ctx context.Context) []dbExtension {
	extensionQuery := "SELECT extname, extversion FROM pg_extension;"

	ctx, cancel := context.WithTimeout(ctx, globaldb.PostgresQueryTimeout)
	defer cancel()
	row, err := globaldb.GetPostgres().Query(ctx, extensionQuery)
	if err != nil {
		log.Errorf("error fetching object counts: %v", err)
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
