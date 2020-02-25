package resolvers

import (
	"testing"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stretchr/testify/assert"
)

func TestMapImagesToComponentResolvers(t *testing.T) {
	fakeRoot := &Resolver{}
	images := testImages()

	query := &v1.Query{}
	vulneranilityResolvers, err := mapImagesToComponentResolvers(fakeRoot, images, query)
	assert.NoError(t, err)
	assert.Len(t, vulneranilityResolvers, 4)

	query = search.NewQueryBuilder().AddExactMatches(search.FixedBy, "1.1").ProtoQuery()
	vulneranilityResolvers, err = mapImagesToComponentResolvers(fakeRoot, images, query)
	assert.NoError(t, err)
	assert.Len(t, vulneranilityResolvers, 1)

	query = search.NewQueryBuilder().AddExactMatches(search.CVE, "cve-2019-1", "cve-2019-2", "cve-2019-3").ProtoQuery()
	vulneranilityResolvers, err = mapImagesToComponentResolvers(fakeRoot, images, query)
	assert.NoError(t, err)
	assert.Len(t, vulneranilityResolvers, 1)

	query = search.NewQueryBuilder().AddExactMatches(search.ComponentVersion, "1.0").ProtoQuery()
	vulneranilityResolvers, err = mapImagesToComponentResolvers(fakeRoot, images, query)
	assert.NoError(t, err)
	assert.Len(t, vulneranilityResolvers, 2)
}
