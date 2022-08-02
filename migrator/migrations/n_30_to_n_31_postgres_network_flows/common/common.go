package common

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
)

const (
	// GlobalPrefix is the generic prefix for network flows
	GlobalPrefix = "networkFlows2"
)

var (
	keySeperator = []byte("\x00")
)

// GetClusterIDFromKey gets cluster id from key
func GetClusterIDFromKey(key []byte) ([]byte, error) {
	parts := bytes.Split(key, keySeperator)
	if len(parts) < 2 || string(parts[0]) != GlobalPrefix {
		return nil, errors.Errorf("unexpected key networkflow %v", key)
	}
	return parts[1], nil
}

// FlowStoreKeyPrefix returns the prefix for a specific flow store
func FlowStoreKeyPrefix(clusterID string) []byte {
	return []byte(fmt.Sprintf("%s\x00%s\x00", GlobalPrefix, clusterID))
}

// GetID converts *storage.NetworkFlowProperties into a []byte
func GetID(props *storage.NetworkFlowProperties) []byte {
	return []byte(fmt.Sprintf("%x:%s:%x:%s:%x:%x", int32(props.GetSrcEntity().GetType()), props.GetSrcEntity().GetId(), int32(props.GetDstEntity().GetType()), props.GetDstEntity().GetId(), props.GetDstPort(), int32(props.GetL4Protocol())))
}

// ParseID parses the bytes into a storage.NetworkFlowProperties struct
func ParseID(id []byte) (*storage.NetworkFlowProperties, error) {
	parts := strings.Split(string(id), ":")
	if len(parts) != 6 {
		return nil, errors.Errorf("expected 6 parts when parsing network flow ID, got %d", len(parts))
	}

	srcType, err := strconv.ParseInt(parts[0], 16, 32)
	if err != nil {
		return nil, errors.Wrap(err, "parsing source type of network flow ID")
	}
	dstType, err := strconv.ParseInt(parts[2], 16, 32)
	if err != nil {
		return nil, errors.Wrap(err, "parsing dest type of network flow ID")
	}
	dstPort, err := strconv.ParseUint(parts[4], 16, 32)
	if err != nil {
		return nil, errors.Wrap(err, "parsing dest port of network flow ID")
	}
	l4proto, err := strconv.ParseInt(parts[5], 16, 32)
	if err != nil {
		return nil, errors.Wrap(err, "parsing l4 proto of network flow ID")
	}

	result := &storage.NetworkFlowProperties{
		SrcEntity: &storage.NetworkEntityInfo{
			Type: storage.NetworkEntityInfo_Type(srcType),
			Id:   parts[1],
		},
		DstEntity: &storage.NetworkEntityInfo{
			Type: storage.NetworkEntityInfo_Type(dstType),
			Id:   parts[3],
		},
		DstPort:    uint32(dstPort),
		L4Protocol: storage.L4Protocol(l4proto),
	}
	return result, nil
}
