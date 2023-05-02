package resolvers

import (
	"context"

	"github.com/stackrox/rox/central/views/imagecve"
	"github.com/stackrox/rox/pkg/utils"
)

func init() {
	schema := getBuilder()
	utils.Must(
		// NOTE: This list is and should remain alphabetically ordered
		schema.AddType("CVEDistroTuple",
			[]string{
				"description: String!",
				"reference: String!",
				"operatingSystem: String!",
				"cvss: Float!",
				"cvssVersion: String!",
			}),
	)
}

type cveDistroTupleResolver struct {
	ctx  context.Context
	root *Resolver
	data imagecve.CVEDistroTuple
}

func (resolver *Resolver) wrapCveDistroTuplesWithContext(ctx context.Context, values []imagecve.CVEDistroTuple, err error) ([]*cveDistroTupleResolver, error) {
	if err != nil || len(values) == 0 {
		return nil, err
	}

	output := make([]*cveDistroTupleResolver, len(values))
	for i, v := range values {
		output[i] = &cveDistroTupleResolver{ctx: ctx, root: resolver, data: v}
	}
	return output, nil
}

func (t *cveDistroTupleResolver) GetDescription() string {
	return t.data.GetDescription()
}

func (t *cveDistroTupleResolver) GetURL() string {
	return t.data.GetURL()
}

func (t *cveDistroTupleResolver) GetOperatingSystem() string {
	return t.data.GetOperatingSystem()
}

func (t *cveDistroTupleResolver) GetCvss() float64 {
	return float64(t.data.GetCvss())
}

func (t *cveDistroTupleResolver) GetCvssVersion() string {
	return t.data.GetCvssVersion()
}
