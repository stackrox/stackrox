package backup

// Backup bundle structure in zip archive.
const (
	BoltFileName   = "bolt.db"
	BadgerFileName = "badger.db"
	RocksFileName  = "rocks.db"
	KeysBaseFolder = "keys"
	CaKeyPem       = "ca-key.pem"
	CaCertPem      = "ca.pem"
	JwtKeyInDer    = "jwt-key.der"
	JwtKeyInPem    = "jwt-key.pem"
)
