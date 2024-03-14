package internaltov2storage

import (
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/uuid"
)

var (
	log = logging.LoggerForModule()
)

func BuildProfileRefID(clusterID string, profileID string, productType string) string {
	interimUUID := buildDeterministicID(clusterID, profileID)

	return buildDeterministicID(interimUUID, productType)
}

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
