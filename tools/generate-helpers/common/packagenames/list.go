package packagenames

// This block enumerates well-known package names.
const (
	GogoProto = "github.com/gogo/protobuf/proto"
	BBolt     = "go.etcd.io/bbolt"
	Bleve     = "github.com/blevesearch/bleve/v2"
)

// This block enumerates well-known Rox package names.
var (
	BoltHelperProto = PrefixRoxPkg("bolthelper/crud/proto")
	UUID            = PrefixRoxPkg("uuid")
	Metrics         = PrefixRox("central/metrics")
	Ops             = PrefixRoxPkg("metrics")
	V1              = PrefixRox("generated/api/v1")
	Storage         = PrefixRox("generated/storage")
	RoxBleve        = PrefixRoxPkg("search/blevesearch")
	RoxBatcher      = PrefixRoxPkg("batcher")
	RoxSearch       = PrefixRoxPkg("search")
	RoxCentral      = PrefixRox("central")
	RoxBleveHelper  = PrefixRoxPkg("blevehelper")
	SingletonStore  = PrefixRoxPkg("bolthelper/singletonstore")
	Sync            = PrefixRoxPkg("sync")
	StoreCache      = PrefixRoxPkg("storecache")
)
