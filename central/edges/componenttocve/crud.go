package componenttocve

import (
	"github.com/gogo/protobuf/proto"
	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/dackbox/crud"
)

var (
	// Bucket stores the component to vulnerability edges.
	Bucket = []byte("comp_to_vuln")

	// Reader reads storage.CVEs directly from the store.
	Reader = crud.NewReader(
		crud.WithAllocFunction(alloc),
	)

	// Upserter writes storage.CVEs directly to the store.
	Upserter = crud.NewUpserter(
		crud.WithKeyFunction(crud.PrefixKey(Bucket, keyFunc)),
	)

	// Deleter deletes vulns from the store.
	Deleter = crud.NewDeleter(
		crud.GCAllChildren(),
	)
)

func init() {
	globaldb.RegisterBucket(Bucket, "Component Vuln Edge")
}

func keyFunc(msg proto.Message) []byte {
	return []byte(msg.(interface{ GetId() string }).GetId())
}

func alloc() proto.Message {
	return &storage.ComponentCVEEdge{}
}
