package m55tom56

var (
	clusterBucketName = []byte("clusters")
	nodeBucketName    = []byte("nodes")
	graphBucket       = []byte("dackbox_graph")
)

func getGraphKey(id []byte) []byte {
	return prefixKey(graphBucket, id)
}

func getClusterKey(id []byte) []byte {
	return prefixKey(clusterBucketName, id)
}

func getNodeKey(id []byte) []byte {
	return prefixKey(nodeBucketName, id)
}

func prefixKey(prefix, key []byte) []byte {
	ret := make([]byte, 0, len(prefix)+len(key)+1)
	ret = append(ret, prefix...)
	ret = append(ret, byte('\x00')) // The separator is a null char
	ret = append(ret, key...)
	return ret
}
