package m91tom92

import "github.com/stackrox/stackrox/migrator/migrations/rocksdbmigration"

var (
	cveBucket          = []byte("image_vuln")
	imageBucket        = []byte("imageBucket")
	imageCVEEdgePrefix = []byte("image_to_cve")
	graphBucket        = []byte("dackbox_graph")
)

func getCVEKey(id []byte) []byte {
	return rocksdbmigration.GetPrefixedKey(cveBucket, id)
}

func getImageKey(id []byte) []byte {
	return rocksdbmigration.GetPrefixedKey(imageBucket, id)
}

func getGraphKey(id []byte) []byte {
	return rocksdbmigration.GetPrefixedKey(graphBucket, id)
}
