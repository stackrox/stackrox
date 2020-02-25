package m27tom28

import (
	"encoding/base64"
	"fmt"
)

var (
	clusterBucketName    = []byte("clusters")
	namespaceBucketName  = []byte("namespaces")
	deploymentBucketName = []byte("deployments")

	imageBucketName             = []byte("imageBucket")
	imageToComponentsBucketName = []byte("image_to_comp")
	componentsBucketName        = []byte("image_component")
	componentsToCVEsBucketName  = []byte("comp_to_vuln")
	cveBucketName               = []byte("image_vuln")

	graphBucket = []byte("dackbox_graph")
)

func encodeIDPair(parentID, childID string) string {
	nameEncoded := base64.RawURLEncoding.EncodeToString([]byte(parentID))
	versionEncoded := base64.RawURLEncoding.EncodeToString([]byte(childID))
	return fmt.Sprintf("%s:%s", nameEncoded, versionEncoded)
}

func getGraphKey(id string) []byte {
	return prefixKey(graphBucket, []byte(id))
}

func getClusterKey(id string) []byte {
	return prefixKey(clusterBucketName, []byte(id))
}

func getNamespaceKey(id string) []byte {
	return prefixKey(namespaceBucketName, []byte(id))
}

func getDeploymentKey(id string) []byte {
	return prefixKey(deploymentBucketName, []byte(id))
}

func getImageKey(id string) []byte {
	return prefixKey(imageBucketName, []byte(id))
}

func getImageComponentEdgeKey(id string) []byte {
	return prefixKey(imageToComponentsBucketName, []byte(id))
}

func getComponentKey(id string) []byte {
	return prefixKey(componentsBucketName, []byte(id))
}

func getComponentCVEEdgeKey(id string) []byte {
	return prefixKey(componentsToCVEsBucketName, []byte(id))
}

func getCVEKey(id string) []byte {
	return prefixKey(cveBucketName, []byte(id))
}

func prefixKey(prefix, key []byte) []byte {
	ret := make([]byte, 0, len(prefix)+len(key)+1)
	ret = append(ret, prefix...)
	ret = append(ret, byte('\x00')) // The separator is a null char
	ret = append(ret, key...)
	return ret
}
