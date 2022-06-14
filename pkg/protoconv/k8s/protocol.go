package k8s

import (
	"strings"

	"github.com/stackrox/stackrox/generated/storage"
	v1 "k8s.io/api/core/v1"
)

var (
	k8sProtoToStorageProtocol = map[v1.Protocol]storage.Protocol{
		v1.ProtocolTCP:  storage.Protocol_TCP_PROTOCOL,
		v1.ProtocolUDP:  storage.Protocol_UDP_PROTOCOL,
		v1.ProtocolSCTP: storage.Protocol_SCTP_PROTOCOL,
	}

	normalizedK8sProtoNameToStorageProtocol = func() map[string]storage.Protocol {
		m := make(map[string]storage.Protocol, len(k8sProtoToStorageProtocol))
		for k, v := range k8sProtoToStorageProtocol {
			m[strings.ToLower(string(k))] = v
		}
		return m
	}()
)

// ProtoNameToStorageProtocol converts a Kubernetes protocol name to a `storage.Protocol`
// value.
func ProtoNameToStorageProtocol(protoName string) storage.Protocol {
	if protoName == "" {
		// empty protocol in K8s always means TCP
		protoName = string(v1.ProtocolTCP)
	}
	return normalizedK8sProtoNameToStorageProtocol[strings.ToLower(protoName)]
}
