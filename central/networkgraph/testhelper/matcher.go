package testhelper

import (
	"github.com/stackrox/rox/generated/storage"
	"google.golang.org/protobuf/proto"
)

// MatchElements compares two NetworkFlow slices ignoring the UpdatedAt field
func MatchElements(expected []*storage.NetworkFlow, actual []*storage.NetworkFlow) bool {
	foundCount := 0
	for _, expectedFlow := range expected {
		for _, actualFlow := range actual {
			if actualFlow.GetClusterId() == expectedFlow.GetClusterId() &&
				proto.Equal(actualFlow.GetProps(), expectedFlow.GetProps()) &&
				proto.Equal(actualFlow.GetLastSeenTimestamp(), expectedFlow.GetLastSeenTimestamp()) {
				foundCount++
				break
			}
		}
	}
	return foundCount == len(expected)
}
