package common

import (
	"bytes"
	"fmt"

	"github.com/stackrox/rox/generated/storage"
)

var (
	idSeparator = []byte(":")
)

// GetDeploymentIDsFromKey take in an id []byte and return the deployments in the id
func GetDeploymentIDsFromKey(id []byte) ([]byte, []byte) {
	bytesSlices := bytes.Split(id, idSeparator)
	return bytesSlices[1], bytesSlices[3]
}

// GetID converts *storage.NetworkFlowProperties into a []byte
func GetID(props *storage.NetworkFlowProperties) []byte {
	return []byte(fmt.Sprintf("%x:%s:%x:%s:%x:%x", int32(props.GetSrcEntity().GetType()), props.GetSrcEntity().GetId(), int32(props.GetDstEntity().GetType()), props.GetDstEntity().GetId(), props.GetDstPort(), int32(props.GetL4Protocol())))
}
