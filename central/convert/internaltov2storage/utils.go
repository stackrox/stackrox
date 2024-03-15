package internaltov2storage

import (
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/uuid"
)

var (
	log = logging.LoggerForModule()
)

// BuildProfileRefID returns a UUID based off cluster, profileID, and product type to simplify SQL
// joins in SELECT statements.  This is necessary due to how the compliance operator objects relate to
// each other and the fields used in those relations will have same values across clusters.
func BuildProfileRefID(clusterID string, profileID string, productType string) string {
	interimUUID := buildDeterministicID(clusterID, profileID)

	return buildDeterministicID(interimUUID, productType)
}

// BuildScanRefID returns a UUID based off cluster and scan name to simplify SQL joins in SELECT statements.
// This is necessary due to how the compliance operator objects relate to each other and the fields
// used in those relations will have same values across clusters.
func BuildScanRefID(clusterID string, scanName string) string {
	return buildDeterministicID(clusterID, scanName)
}

func buildDeterministicID(part1 string, part2 string) string {
	baseUUID, err := uuid.FromString(part1)
	if err != nil {
		log.Error(err)
		return ""
	}
	return uuid.NewV5(baseUUID, part2).String()
}
