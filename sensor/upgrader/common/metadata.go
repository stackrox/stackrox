package common

import (
	"strings"

	"github.com/stackrox/rox/pkg/set"
)

var (
	metadataTransferKeyBlacklist = set.NewFrozenStringSet(
		"app",
		"service",
	)
)

// ShouldTransferMetadataKey returns whether a metadata key (label or annotation) should be transferred from an old to a new
// object.
func ShouldTransferMetadataKey(key string) bool {
	if key == "" {
		return false
	}

	if metadataTransferKeyBlacklist.Contains(key) {
		return false
	}

	keyParts := strings.SplitN(key, "/", 2)
	return !(keyParts[0] == "stackrox.io" || strings.HasSuffix(keyParts[0], ".stackrox.io"))
}
