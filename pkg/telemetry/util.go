package telemetry

import (
	"time"

	"github.com/gogo/protobuf/types"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/set"
)

var (
	// WellKnownNamespaces is a set of known namespaces used to sanitize Namespace telemetry
	WellKnownNamespaces = set.NewFrozenSet("stackrox", "default", "kube-system", "kube-public", "kube-node-lease")
)

// GetProviderString returns a string based on the cluster provider or an empty string for an unrecognized provider
func GetProviderString(metadata *storage.ProviderMetadata) string {
	if metadata == nil {
		return ""
	}
	if metadata.GetAws() != nil {
		return "AWS"
	}
	if metadata.GetAzure() != nil {
		return "Azure"
	}
	if metadata.GetGoogle() != nil {
		return "Google"
	}
	return ""
}

// GetTimeOrNil takes a pointer to a protobuf timestamp and returns a pointer to a Go Time or nil
func GetTimeOrNil(inTime *types.Timestamp) *time.Time {
	outTime, err := types.TimestampFromProto(inTime)
	if err != nil {
		return nil
	}
	return &outTime
}
