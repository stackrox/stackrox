package internaltov2storage

import (
	"strings"

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

// BuildNameRefID returns a UUID based off cluster and scan name to simplify SQL joins in SELECT statements.
// This is necessary due to how the compliance operator objects relate to each other and the fields
// used in those relations will have same values across clusters.
func BuildNameRefID(clusterID string, name string) string {
	return buildDeterministicID(clusterID, name)
}

// idToDNSFriendlyName mirrors compliance-operator's IDToDNSFriendlyName (https://github.com/ComplianceAsCode/compliance-operator/blob/ae5d84d57ee9abb1a6e6692fdf093d54283b739e/pkg/utils/nameutils.go#L43):
// it strips the standard ssgproject XCCDF prefix (if present) then replaces underscores with hyphens.
// Used to derive parentRule for CustomRule objects, which carry their identifier in Spec.ID
// rather than in the compliance.openshift.io/rule annotation.
func idToDNSFriendlyName(id string) string {
	const ssgPrefix = "xccdf_org.ssgproject.content_rule_"
	return strings.ToLower(strings.ReplaceAll(strings.TrimPrefix(id, ssgPrefix), "_", "-"))
}

func buildDeterministicID(part1 string, part2 string) string {
	baseUUID, err := uuid.FromString(part1)
	if err != nil {
		log.Error(err)
		return ""
	}
	return uuid.NewV5(baseUUID, part2).String()
}
