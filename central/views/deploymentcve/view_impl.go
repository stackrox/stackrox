package deploymentcve

import (
	"context"

	"github.com/stackrox/rox/central/views/common"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/walker"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/search"
	pgSearch "github.com/stackrox/rox/pkg/search/postgres"
)

var selects = []*v1.QuerySelect{
	search.NewQuerySelect(search.DeploymentID).Proto(),
	search.NewQuerySelect(search.DeploymentName).Proto(),
	search.NewQuerySelect(search.ClusterID).Proto(),
	search.NewQuerySelect(search.Cluster).Proto(),
	search.NewQuerySelect(search.Namespace).Proto(),
	search.NewQuerySelect(search.ImageSHA).Proto(),
	search.NewQuerySelect(search.ImageRegistry).Proto(),
	search.NewQuerySelect(search.ImageRemote).Proto(),
	search.NewQuerySelect(search.ImageTag).Proto(),
	search.NewQuerySelect(search.ImageOS).Proto(),
	search.NewQuerySelect(search.Component).Proto(),
	search.NewQuerySelect(search.ComponentVersion).Proto(),
	search.NewQuerySelect(search.CVE).Proto(),
	search.NewQuerySelect(search.CVSS).Proto(),
	search.NewQuerySelect(search.Severity).Proto(),
	search.NewQuerySelect(search.EPSSProbablity).Proto(),
	search.NewQuerySelect(search.FixedBy).Proto(),
}

type cveViewImpl struct {
	schema *walker.Schema
	db     postgres.DB
}

func (v *cveViewImpl) WalkVulnFindings(ctx context.Context, fn func(*VulnFinding) error) error {
	q := search.EmptyQuery()

	var err error
	if q, err = common.WithSACFilter(ctx, resources.Deployment, q); err != nil {
		return err
	}
	if q, err = common.WithSACFilter(ctx, resources.Image, q); err != nil {
		return err
	}
	q.Selects = selects

	return pgSearch.RunSelectRequestForSchemaFn(ctx, v.db, v.schema, q, fn)
}
