package backup

import (
	"github.com/stackrox/stackrox/pkg/certgen"
	"github.com/stackrox/stackrox/pkg/mtls"
)

// Backup bundle structure in zip archive.
const (
	BoltFileName     = "bolt.db"
	RocksFileName    = "rocks.db"
	PostgresFileName = "postgres.db.tar"
	KeysBaseFolder   = "keys"
	CaKeyPem         = mtls.CAKeyFileName
	CaCertPem        = mtls.CACertFileName
	JwtKeyInDer      = certgen.JWTKeyDERFileName
	JwtKeyInPem      = certgen.JWTKeyPEMFileName
	MigrationVersion = "migration_version.yaml"
)
