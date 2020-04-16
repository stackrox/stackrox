package m33tom34

import "bytes"

var (
	clusterBucketName      = []byte("clusters")
	namespaceSACBucketName = []byte("namespacesSACBucket")
	namespaceBucketName    = []byte("namespaces")
	deploymentBucketName   = []byte("deployments")

	graphBucket = []byte("dackbox_graph")
)

func getGraphKey(id []byte) []byte {
	return prefixKey(graphBucket, id)
}

func getClusterKey(id []byte) []byte {
	return prefixKey(clusterBucketName, id)
}

func getNamespaceKey(id []byte) []byte {
	return prefixKey(namespaceBucketName, id)
}

func getNamespaceSACKey(name []byte) []byte {
	return prefixKey(namespaceSACBucketName, name)
}

func getDeploymentKey(id []byte) []byte {
	return prefixKey(deploymentBucketName, id)
}

func getFullPrefix(prefix []byte) []byte {
	return append(append([]byte{}, prefix...), byte('\x00'))
}

func hasPrefix(prefix, key []byte) bool {
	return bytes.HasPrefix(key, prefix)
}

func prefixKey(prefix, key []byte) []byte {
	ret := make([]byte, 0, len(prefix)+len(key)+1)
	ret = append(ret, prefix...)
	ret = append(ret, byte('\x00')) // The separator is a null char
	ret = append(ret, key...)
	return ret
}
