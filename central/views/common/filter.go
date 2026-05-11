package common

import (
	"fmt"

	"github.com/stackrox/rox/generated/storage"
)

// ApplyActiveDeploymentExclusion appends a NOT EXISTS sub-clause that excludes
// rows whose image column appears in any active deployment.
// imageColumnExpr is the fully-qualified column for the image key in the outer
// query (e.g. "images.id" or "image_cves_v2.imageidv2").
// containerImageColumn is the matching column name in deployments_containers
// (e.g. "image_id" or "image_idv2").
//
// The returned WHERE string uses $$ placeholders (the StackRox convention);
// they are renumbered to positional parameters by the query runner.
func ApplyActiveDeploymentExclusion(
	existingWhere string,
	existingValues []interface{},
	imageColumnExpr string,
	containerImageColumn string,
) (string, []interface{}) {
	notExists := fmt.Sprintf(
		"NOT EXISTS (SELECT 1 FROM deployments_containers dc "+
			"INNER JOIN deployments d ON dc.deployments_id = d.id "+
			"WHERE dc.%s = %s AND d.state = $$)",
		containerImageColumn, imageColumnExpr,
	)

	values := make([]interface{}, len(existingValues), len(existingValues)+1)
	copy(values, existingValues)
	values = append(values, int32(storage.DeploymentState_DEPLOYMENT_STATE_ACTIVE))

	if existingWhere == "" {
		return notExists, values
	}
	return fmt.Sprintf("(%s) and %s", existingWhere, notExists), values
}
