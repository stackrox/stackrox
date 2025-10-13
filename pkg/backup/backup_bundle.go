package backup

import (
	"github.com/stackrox/rox/pkg/certgen"
	"github.com/stackrox/rox/pkg/mtls"
)

// Backup bundle structure in zip archive.
const (
	PostgresFileName     = "postgres.dump"
	PostgresSizeFileName = "postgres.size"
	KeysBaseFolder       = "keys"
	CaKeyPem             = mtls.CAKeyFileName
	CaCertPem            = mtls.CACertFileName
	JwtKeyInDer          = certgen.JWTKeyDERFileName
	JwtKeyInPem          = certgen.JWTKeyPEMFileName
	MigrationVersion     = "migration_version.yaml"
	DatabaseBaseFolder   = "central-db"
	DatabasePassword     = "password"
)
