package m71tom72

var (
	nsBucketName    = []byte("namespaces")
	nsSACBucketName = []byte("namespacesSACBucket")
	graphBucket     = []byte("dackbox_graph")
)

func getGraphKey(id []byte) []byte {
	return prefixKey(graphBucket, id)
}

func prefixKey(prefix, key []byte) []byte {
	ret := make([]byte, 0, len(prefix)+len(key)+1)
	ret = append(ret, prefix...)
	ret = append(ret, byte('\x00')) // The separator is a null char
	ret = append(ret, key...)
	return ret
}
